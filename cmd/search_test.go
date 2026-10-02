package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/nickng/bibtex"
	"github.com/thofma/bibi/lib/bibliography"
	"github.com/thofma/bibi/util"
)

type fakeDiscovery func(string) ([]bibliography.Work, error)

func (search fakeDiscovery) Search(query string) ([]bibliography.Work, error) { return search(query) }

type fakeProvider func(bibliography.Work) ([]bibliography.Record, error)

func (provider fakeProvider) BibTeX(work bibliography.Work) ([]bibliography.Record, error) {
	return provider(work)
}

type fakePagedDiscovery func(context.Context, string, string) (bibliography.SearchPage, error)

func (fakePagedDiscovery) Search(string) ([]bibliography.Work, error) {
	panic("paged discovery used the unpaged interface")
}

func (search fakePagedDiscovery) SearchPage(ctx context.Context, query, token string) (bibliography.SearchPage, error) {
	return search(ctx, query, token)
}

func TestSearchLaterPageSelectionAndRetryRetrieveOnlySelectedWork(t *testing.T) {
	works := []bibliography.Work{{Title: "First"}, {Title: "Second"}, {Title: "Third"},
		{Title: "Fourth", Authors: []string{"One, Alice", "Two, Bob"}, Venue: "Journal", Notes: "Revised version"}}
	loads, exports, picks := 0, 0, 0
	services := searchServices{
		discovery: map[string]bibliography.Discoverer{"zb": fakePagedDiscovery(func(ctx context.Context, query, token string) (bibliography.SearchPage, error) {
			if query != "free query" || ctx == nil {
				t.Errorf("query=%q context=%v", query, ctx)
			}
			loads++
			if exports != 0 {
				t.Fatal("BibTeX retrieved while browsing")
			}
			if token == "" {
				return bibliography.SearchPage{Works: works[:2], NextToken: "next", Total: 4}, nil
			}
			if token != "next" {
				t.Fatalf("wrong token %q", token)
			}
			if loads == 2 {
				return bibliography.SearchPage{}, errors.New("temporary failure")
			}
			return bibliography.SearchPage{Works: works[2:], Total: 4}, nil
		})},
		bib: map[string]bibliography.Provider{"mr": fakeProvider(func(work bibliography.Work) ([]bibliography.Record, error) {
			exports++
			if !reflect.DeepEqual(work, works[3]) {
				t.Fatalf("provider received wrong work: %+v", work)
			}
			record := searchRecord("MRLater", "")
			record.Work = work
			return []bibliography.Record{record}, nil
		})},
	}
	command := newSearchCommand(func() searchServices { return services }, func(request util.ChooserRequest) (int, error) {
		picks++
		if picks != 1 || request.LoadPage == nil || request.NextToken != "next" {
			t.Fatal("missing continuation or unexpected second selector")
		}
		if _, err := request.LoadPage(context.Background(), request.NextToken); err == nil {
			t.Fatal("page failure was swallowed")
		}
		page, err := request.LoadPage(context.Background(), request.NextToken)
		if err != nil || len(page.Choices) != 2 || !strings.Contains(page.Choices[1].Details, "Two, Bob") || !strings.Contains(page.Choices[1].Details, "Revised version") {
			t.Fatalf("missing later-page details: %+v, error=%v", page, err)
		}
		return 3, nil
	})
	var output, stderr bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&stderr)
	command.SetArgs([]string{"free query", "--bib", "mr"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	assertMRBibTeX(t, output.String(), "MRLater")
	if loads != 3 || exports != 1 || picks != 1 || stderr.Len() != 0 {
		t.Fatalf("loads=%d exports=%d picks=%d stderr=%q", loads, exports, picks, stderr.String())
	}
}

func TestSearchCancellationAfterLoadingDoesNotExport(t *testing.T) {
	services := searchServices{
		discovery: map[string]bibliography.Discoverer{"zb": fakePagedDiscovery(func(ctx context.Context, _, token string) (bibliography.SearchPage, error) {
			if token != "" {
				return bibliography.SearchPage{}, ctx.Err()
			}
			return bibliography.SearchPage{Works: []bibliography.Work{{Title: "First"}}, NextToken: "next"}, nil
		})},
		bib: map[string]bibliography.Provider{"zb": fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
			t.Fatal("exported after cancellation")
			return nil, nil
		})},
	}
	command := newSearchCommand(func() searchServices { return services }, func(request util.ChooserRequest) (int, error) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := request.LoadPage(ctx, request.NextToken); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled page error=%v", err)
		}
		return -1, nil
	})
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"query"})
	if err := command.Execute(); err == nil || !strings.Contains(err.Error(), "cancelled") || output.Len() != 0 {
		t.Fatalf("error=%v output=%q", err, output.String())
	}
}

func searchRecord(key, doi string) bibliography.Record {
	entry := bibtex.NewBibEntry("article", key)
	entry.AddField("title", bibtex.NewBibConst("A selected work"))
	entry.AddField("author", bibtex.NewBibConst("Doe, Jane"))
	entry.AddField("journal", bibtex.NewBibConst("Example Journal"))
	entry.AddField("year", bibtex.NewBibConst("1999"))
	return bibliography.Record{Work: bibliography.Work{Title: "A selected work", DOI: doi}, Entry: entry}
}

func TestSearchDiscoveryAndBibTeXAreIndependent(t *testing.T) {
	for _, discovery := range []string{"zb", "crossref"} {
		for _, provider := range []string{"zb", "mr", "crossref"} {
			t.Run(discovery+" to "+provider, func(t *testing.T) {
				work := bibliography.Work{Title: "A selected work", Authors: []string{"Doe, Jane"}, Year: "1999", DOI: "10.1000/example"}
				searchCalls, providerCalls := 0, 0
				services := searchServices{
					discovery: map[string]bibliography.Discoverer{discovery: fakeDiscovery(func(query string) ([]bibliography.Work, error) {
						searchCalls++
						if query != "anything can go in here 1999" {
							t.Errorf("query = %q, want unparsed free text", query)
						}
						return []bibliography.Work{work}, nil
					})},
					bib: map[string]bibliography.Provider{provider: fakeProvider(func(got bibliography.Work) ([]bibliography.Record, error) {
						providerCalls++
						if !reflect.DeepEqual(got, work) {
							t.Errorf("provider received %+v, want %+v", got, work)
						}
						return []bibliography.Record{searchRecord(provider+"Entry", "10.1000/example")}, nil
					})},
				}
				command := newSearchCommand(func() searchServices { return services }, func(util.ChooserRequest) (int, error) {
					t.Fatal("exact single match opened a picker")
					return 0, nil
				})
				var output, stderr bytes.Buffer
				command.SetOut(&output)
				command.SetErr(&stderr)
				command.SetArgs([]string{"anything", "can go in here 1999", "--discovery", discovery, "--bib", provider})
				if err := command.Execute(); err != nil {
					t.Fatal(err)
				}
				assertMRBibTeX(t, output.String(), provider+"Entry")
				if searchCalls != 1 || providerCalls != 1 || stderr.Len() != 0 {
					t.Errorf("calls = %d/%d, stderr = %q", searchCalls, providerCalls, stderr.String())
				}
			})
		}
	}
}

func TestSearchDefaultsToZB(t *testing.T) {
	command := newSearchCommand(defaultSearchServices, nil)
	for _, name := range []string{"discovery", "bib"} {
		if value, _ := command.Flags().GetString(name); value != "zb" {
			t.Errorf("%s default = %q, want zb", name, value)
		}
	}
}

func TestSearchSelectsBeforeRetrievingBibTeX(t *testing.T) {
	works := []bibliography.Work{{Title: "First"}, {Title: "Second", Authors: []string{"Artin, Emil"}, Year: "2006"}}
	pickerCalls := 0
	services := searchServices{
		discovery: map[string]bibliography.Discoverer{"zb": fakeDiscovery(func(string) ([]bibliography.Work, error) { return works, nil })},
		bib: map[string]bibliography.Provider{"mr": fakeProvider(func(work bibliography.Work) ([]bibliography.Record, error) {
			if pickerCalls != 1 || !reflect.DeepEqual(work, works[1]) {
				t.Errorf("provider called before selection or for wrong work: %+v", work)
			}
			record := searchRecord("MR1", "10.1000/second")
			record.Work = work
			record.DOI = "10.1000/second"
			return []bibliography.Record{record}, nil
		})},
	}
	command := newSearchCommand(func() searchServices { return services }, func(request util.ChooserRequest) (int, error) {
		labels := choiceLabels(request)
		pickerCalls++
		if pickerCalls > 1 {
			t.Fatal("single provider candidate opened a second picker")
		}
		if len(labels) != 2 || !strings.Contains(labels[1], "Second") {
			t.Errorf("choices = %v", labels)
		}
		return 1, nil
	})
	var output, stderr bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&stderr)
	command.SetArgs([]string{"query", "--bib", "mr"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	assertMRBibTeX(t, output.String(), "MR1")
	if pickerCalls != 1 || stderr.Len() != 0 {
		t.Errorf("picker calls = %d, stderr = %q", pickerCalls, stderr.String())
	}
}

func TestSearchSingleCandidateAndDOIValidation(t *testing.T) {
	for _, test := range []struct {
		name      string
		discovery string
		candidate string
		wantError bool
	}{
		{"discovery missing DOI", "", "10.1000/example", false},
		{"candidate missing DOI", "10.1000/example", "", false},
		{"both missing DOI", "", "", false},
		{"DOI conflict", "10.1000/example", "10.1000/wrong", true},
		{"normalized DOI", "10.1000/example", "https://doi.org/10.1000/EXAMPLE", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			services := searchServices{
				discovery: map[string]bibliography.Discoverer{"zb": fakeDiscovery(func(string) ([]bibliography.Work, error) {
					return []bibliography.Work{{Title: "A selected work", DOI: test.discovery}}, nil
				})},
				bib: map[string]bibliography.Provider{"mr": fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
					return []bibliography.Record{searchRecord("MR1", test.candidate)}, nil
				})},
			}
			command := newSearchCommand(func() searchServices { return services }, func(util.ChooserRequest) (int, error) {
				t.Fatal("single candidate opened a picker")
				return 0, nil
			})
			var output, stderr bytes.Buffer
			command.SetOut(&output)
			command.SetErr(&stderr)
			command.SetArgs([]string{"query", "--bib", "mr"})
			err := command.Execute()
			if (err != nil) != test.wantError {
				t.Errorf("error = %v, want error = %v", err, test.wantError)
			}
			if test.wantError && output.Len() != 0 {
				t.Errorf("error printed BibTeX: %s", output.String())
			}
			if !test.wantError {
				assertMRBibTeX(t, output.String(), "MR1")
				if stderr.Len() != 0 {
					t.Errorf("single candidate prompted for confirmation: %q", stderr.String())
				}
			}
		})
	}
}

func TestSearchChoosesAmongMultipleBibTeXCandidates(t *testing.T) {
	for _, test := range []struct {
		name      string
		selection int
		wantError bool
	}{
		{"choose second", 1, false},
		{"cancel", -1, true},
		{"invalid selection", 4, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			services := searchServices{
				discovery: map[string]bibliography.Discoverer{"zb": fakeDiscovery(func(string) ([]bibliography.Work, error) {
					return []bibliography.Work{{Title: "A selected work"}}, nil
				})},
				bib: map[string]bibliography.Provider{"mr": fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
					return []bibliography.Record{searchRecord("MR1", ""), searchRecord("MR2", "")}, nil
				})},
			}
			pickerCalls := 0
			command := newSearchCommand(func() searchServices { return services }, func(request util.ChooserRequest) (int, error) {
				labels := choiceLabels(request)
				pickerCalls++
				if len(labels) != 2 || !strings.Contains(labels[1], "[MR2]") {
					t.Errorf("choices = %v", labels)
				}
				return test.selection, nil
			})
			var output bytes.Buffer
			command.SetOut(&output)
			command.SetErr(io.Discard)
			command.SetArgs([]string{"query", "--bib", "mr"})
			err := command.Execute()
			if (err != nil) != test.wantError || pickerCalls != 1 {
				t.Errorf("error = %v, picker calls = %d", err, pickerCalls)
			}
			if test.wantError {
				if output.Len() != 0 {
					t.Errorf("error printed BibTeX: %s", output.String())
				}
			} else {
				assertMRBibTeX(t, output.String(), "MR2")
			}
		})
	}
}

func TestSearchProviderFailureHasNoFallback(t *testing.T) {
	want := errors.New("provider unavailable")
	services := searchServices{
		discovery: map[string]bibliography.Discoverer{"zb": fakeDiscovery(func(string) ([]bibliography.Work, error) {
			return []bibliography.Work{{Title: "Work"}}, nil
		})},
		bib: map[string]bibliography.Provider{
			"mr": fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) { return nil, want }),
			"crossref": fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
				t.Fatal("unexpected Crossref fallback")
				return nil, nil
			}),
		},
	}
	command := newSearchCommand(func() searchServices { return services }, nil)
	var output, stderr bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&stderr)
	command.SetArgs([]string{"query", "--bib", "mr"})
	if err := command.Execute(); !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
	if output.Len() != 0 {
		t.Errorf("error printed BibTeX: %s", output.String())
	}
}

func TestSearchDoesNotRetrieveAfterEmptyOrCancelledDiscovery(t *testing.T) {
	for _, test := range []struct {
		name  string
		works []bibliography.Work
	}{
		{"empty", nil},
		{"cancelled", []bibliography.Work{{Title: "First"}, {Title: "Second"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			services := searchServices{
				discovery: map[string]bibliography.Discoverer{"zb": fakeDiscovery(func(string) ([]bibliography.Work, error) { return test.works, nil })},
				bib: map[string]bibliography.Provider{"zb": fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
					t.Fatal("retrieved BibTeX without a selected work")
					return nil, nil
				})},
			}
			command := newSearchCommand(func() searchServices { return services }, func(util.ChooserRequest) (int, error) { return -1, nil })
			var output bytes.Buffer
			command.SetOut(&output)
			command.SetErr(io.Discard)
			command.SetArgs([]string{"query"})
			if err := command.Execute(); err == nil || output.Len() != 0 {
				t.Errorf("error = %v, output = %q", err, output.String())
			}
		})
	}
}

func TestSearchRejectsInvalidOptionsBeforeNetwork(t *testing.T) {
	for _, args := range [][]string{
		{"query", "--discovery", "mr"}, {"query", "--bib", "unknown"}, {" "}, {},
	} {
		services := searchServices{discovery: map[string]bibliography.Discoverer{"zb": fakeDiscovery(func(string) ([]bibliography.Work, error) {
			t.Fatal("invalid command made a network call")
			return nil, nil
		})}}
		command := newSearchCommand(func() searchServices { return services }, nil)
		command.SetOut(io.Discard)
		command.SetErr(io.Discard)
		command.SetArgs(args)
		if err := command.Execute(); err == nil {
			t.Errorf("arguments %v succeeded", args)
		}
	}
}

func choiceLabels(request util.ChooserRequest) []string {
	labels := make([]string, len(request.Choices))
	for i, choice := range request.Choices {
		labels[i] = choice.Label
	}
	return labels
}
