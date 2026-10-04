package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/thofma/bibi/lib/bibliography"
	"github.com/thofma/bibi/util"
)

func chooseMRShortcut(t *testing.T, request util.ChooserRequest) {
	t.Helper()
	if request.Confirmation || len(request.Actions) != 1 || request.Actions[0].Key != "m" || request.Actions[0].Label != "MR BibTeX" || request.Actions[0].OnSelect == nil {
		t.Fatalf("missing MR selection shortcut: %+v", request)
	}
	request.Actions[0].OnSelect()
}

func TestSearchMRShortcutRespectsExplicitBibProvider(t *testing.T) {
	for _, discovery := range []string{"zb", "crossref"} {
		for _, test := range []struct {
			name, provider string
			bibArgs        []string
			useMR          bool
		}{
			{name: "Enter uses default", provider: "zb"},
			{name: "m uses MR", provider: "mr", useMR: true},
			{name: "explicit default", provider: "zb", bibArgs: []string{"--bib", "zb"}},
			{name: "explicit default with equals", provider: "zb", bibArgs: []string{"--bib=zb"}},
			{name: "explicit MR", provider: "mr", bibArgs: []string{"--bib", "mr"}},
			{name: "explicit Crossref", provider: "crossref", bibArgs: []string{"--bib", "crossref"}},
		} {
			t.Run(discovery+"/"+test.name, func(t *testing.T) {
				works := []bibliography.Work{{Title: "First", DOI: "10.1000/first"}, {Title: "Second", DOI: "10.1000/second"}}
				calls := make(map[string]int)
				services := searchServices{
					discovery: map[string]bibliography.Discoverer{discovery: fakeDiscovery(func(string) ([]bibliography.Work, error) { return works, nil })},
					bib:       make(map[string]bibliography.Provider),
				}
				for _, provider := range []string{"zb", "mr", "crossref"} {
					services.bib[provider] = fakeProvider(func(work bibliography.Work) ([]bibliography.Record, error) {
						calls[provider]++
						if provider != test.provider || !reflect.DeepEqual(work, works[1]) {
							t.Fatalf("provider=%s work=%+v", provider, work)
						}
						record := searchRecord(provider+"Entry", work.DOI)
						record.Work = work
						return []bibliography.Record{record}, nil
					})
				}
				picks := 0
				command := newSearchCommand(func() searchServices { return services }, func(request util.ChooserRequest) (int, error) {
					picks++
					if picks != 1 || request.Confirmation {
						t.Fatal("unexpected confirmation for a verified result")
					}
					if explicit := len(test.bibArgs) > 0; explicit != (len(request.Actions) == 0) {
						t.Fatalf("explicit provider=%t actions=%+v", explicit, request.Actions)
					}
					if test.useMR {
						chooseMRShortcut(t, request)
					}
					return 1, nil
				})
				root := debugTestRoot(command)
				var stdout, stderr bytes.Buffer
				root.SetOut(&stdout)
				root.SetErr(&stderr)
				root.SetArgs(append([]string{"search", "query", "--discovery", discovery, "--debug"}, test.bibArgs...))
				if err := root.Execute(); err != nil {
					t.Fatal(err)
				}
				assertMRBibTeX(t, stdout.String(), test.provider+"Entry")
				if len(calls) != 1 || calls[test.provider] != 1 || !strings.Contains(stderr.String(), "writing "+test.provider+" BibTeX entry") {
					t.Fatalf("provider calls=%v trace=%s", calls, stderr.String())
				}
				if command.Flags().Changed("bib") != (len(test.bibArgs) > 0) {
					t.Fatal("selector changed whether --bib was explicitly supplied")
				}
			})
		}
	}
}

func TestSearchMRShortcutSelectsLaterPage(t *testing.T) {
	record := searchRecord("MRLater", "10.1000/later")
	services := searchServices{
		discovery: map[string]bibliography.Discoverer{"zb": fakePagedDiscovery(func(_ context.Context, _, token string) (bibliography.SearchPage, error) {
			if token == "" {
				return bibliography.SearchPage{Works: []bibliography.Work{{Title: "First"}}, NextToken: "next"}, nil
			}
			return bibliography.SearchPage{Works: []bibliography.Work{record.Work}}, nil
		})},
		bib: map[string]bibliography.Provider{
			"zb": fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
				t.Fatal("used default provider after MR selection")
				return nil, nil
			}),
			"mr": fakeProvider(func(work bibliography.Work) ([]bibliography.Record, error) {
				if !reflect.DeepEqual(work, record.Work) {
					t.Fatalf("retrieved wrong work: %+v", work)
				}
				return []bibliography.Record{record}, nil
			}),
		},
	}
	command := newSearchCommand(func() searchServices { return services }, func(request util.ChooserRequest) (int, error) {
		if _, err := request.LoadPage(request.Context, request.NextToken); err != nil {
			t.Fatal(err)
		}
		chooseMRShortcut(t, request)
		return 1, nil
	})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"query"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	assertMRBibTeX(t, stdout.String(), "MRLater")
}

func TestSearchMRShortcutKeepsConfirmationAndFailureBehavior(t *testing.T) {
	for _, mode := range []string{"accept", "cancel", "lookup fails"} {
		t.Run(mode, func(t *testing.T) {
			providerError := errors.New("MR unavailable")
			services := searchServices{
				discovery: map[string]bibliography.Discoverer{"zb": fakeDiscovery(func(string) ([]bibliography.Work, error) {
					return []bibliography.Work{{Title: "First"}, {Title: "Selected"}}, nil
				})},
				bib: map[string]bibliography.Provider{
					"zb": fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
						t.Fatal("fell back to default provider after MR selection")
						return nil, nil
					}),
					"mr": fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
						if mode == "lookup fails" {
							return nil, providerError
						}
						return []bibliography.Record{searchRecord("MRUnverified", "")}, nil
					}),
				},
			}
			picks := 0
			command := newSearchCommand(func() searchServices { return services }, func(request util.ChooserRequest) (int, error) {
				picks++
				if picks == 1 {
					chooseMRShortcut(t, request)
					return 1, nil
				}
				if !request.Confirmation || len(request.Actions) != 0 || !strings.Contains(request.Choices[0].Details, "Identity unverified") {
					t.Fatalf("MR confirmation was changed: %+v", request)
				}
				if mode == "cancel" {
					return -1, nil
				}
				return 0, nil
			})
			var stdout bytes.Buffer
			command.SetOut(&stdout)
			command.SetErr(io.Discard)
			command.SetArgs([]string{"query"})
			err := command.Execute()
			switch mode {
			case "accept":
				if err != nil || picks != 2 {
					t.Fatalf("error=%v picker calls=%d", err, picks)
				}
				assertMRBibTeX(t, stdout.String(), "MRUnverified")
			case "cancel":
				if !errors.Is(err, errSelectionCancelled) || stdout.Len() != 0 || picks != 2 {
					t.Fatalf("error=%v output=%q picker calls=%d", err, stdout.String(), picks)
				}
			case "lookup fails":
				if !errors.Is(err, providerError) || stdout.Len() != 0 || picks != 1 {
					t.Fatalf("error=%v output=%q picker calls=%d", err, stdout.String(), picks)
				}
			}
		})
	}
}

func TestAddMRShortcutSavesMRBibTeX(t *testing.T) {
	path := filepath.Join(t.TempDir(), "references.bib")
	record := searchRecord("MRFromSelector", "10.1000/selected")
	services := addTestServices(record, "zb", "mr")
	services.discovery["zb"] = fakeDiscovery(func(string) ([]bibliography.Work, error) {
		return []bibliography.Work{{Title: "First"}, record.Work}, nil
	})
	services.bib["zb"] = fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
		t.Fatal("saved default provider entry after MR selection")
		return nil, nil
	})
	stdout, stderr, err := runAddTest(t, []string{"query", path}, services, func(request util.ChooserRequest) (int, error) {
		chooseMRShortcut(t, request)
		return 1, nil
	})
	if err != nil || stdout != "" || !strings.Contains(stderr, "Added MRFromSelector") {
		t.Fatalf("output=%q stderr=%q error=%v", stdout, stderr, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	assertMRBibTeX(t, string(data), "MRFromSelector")
}
