package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/nickng/bibtex"
	"github.com/thofma/bibi/lib/bibliography"
	"github.com/thofma/bibi/util"
)

func getPreprint(publicationDOI string) bibliography.Record {
	entry := bibtex.NewBibEntry("misc", "arXivFixture")
	entry.AddField("title", bibtex.NewBibConst("{Preprint title}"))
	entry.AddField("author", bibtex.NewBibConst("Doe, Jane"))
	entry.AddField("year", bibtex.NewBibConst("2020"))
	return bibliography.Record{Entry: entry, Work: bibliography.Work{
		Title: "Preprint title", Authors: []string{"Doe, Jane"}, Year: "2020",
		DOI: publicationDOI, IDs: map[string]string{"arxiv": "2301.12345"},
	}}
}

func TestGetRoutesExactIdentifiersAndExplicitProviders(t *testing.T) {
	for _, test := range []struct {
		name, wantKey, providerTitle, providerYear string
		args                                       []string
		arxiv, doi, metadata, provider             int
	}{
		{"arXiv PDF version", "arXivFixture", "", "", []string{"https://arxiv.org/pdf/2301.12345v2.pdf"}, 1, 0, 0, 0},
		{"DOI URL", "NativeDOI", "", "", []string{"https://doi.org/10.1000/PUBLISHED"}, 0, 1, 0, 0},
		{"explicit auto", "NativeDOI", "", "", []string{"doi:10.1000/published", "--bib", "auto"}, 0, 1, 0, 0},
		{"DOI to MR", "MRChosen", "Published title", "2024", []string{"10.1000/published", "--bib", "mr"}, 0, 0, 1, 1},
		{"arXiv to MR", "MRChosen", "Preprint title", "2020", []string{"2301.12345", "--bib", "mr"}, 1, 0, 0, 1},
		{"published native DOI", "NativeDOI", "", "", []string{"2301.12345", "--published"}, 1, 1, 0, 0},
		{"published to MR", "MRChosen", "Published title", "2024", []string{"2301.12345", "--published", "--bib", "mr"}, 1, 0, 1, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			arxivCalls, doiCalls, metadataCalls, providerCalls := 0, 0, 0, 0
			publication := bibliography.Work{DOI: "10.1000/published", Title: "Published title", Authors: []string{"Doe, Jane"}, Year: "2024"}
			services := getServices{
				arxiv: func(ctx context.Context, id string) (bibliography.Record, error) {
					arxivCalls++
					if ctx == nil || !strings.HasPrefix(id, "2301.12345") || strings.Contains(id, "https:") {
						t.Fatalf("arXiv input was not normalized: %q", id)
					}
					if test.name == "arXiv PDF version" && id != "2301.12345v2" {
						t.Fatalf("version was lost: %q", id)
					}
					return getPreprint("10.1000/published"), nil
				},
				doi: func(ctx context.Context, id string) (bibliography.Record, error) {
					doiCalls++
					if id != "10.1000/published" {
						t.Fatalf("wrong DOI lookup: %q", id)
					}
					return searchRecord("NativeDOI", id), nil
				},
				doiMetadata: func(ctx context.Context, id string) (bibliography.Work, error) {
					metadataCalls++
					if id != "10.1000/published" {
						t.Fatalf("wrong DOI metadata lookup: %q", id)
					}
					return publication, nil
				},
				bib: map[string]bibliography.Provider{"mr": fakeProvider(func(work bibliography.Work) ([]bibliography.Record, error) {
					providerCalls++
					if work.Title != test.providerTitle || work.Year != test.providerYear || work.DOI != publication.DOI {
						t.Fatalf("provider received wrong work: %+v", work)
					}
					return []bibliography.Record{searchRecord("MRChosen", work.DOI)}, nil
				})},
			}
			root := debugTestRoot(newGetCommand(func() getServices { return services }, func(util.ChooserRequest) (int, error) {
				t.Fatal("exact lookup opened a discovery or confirmation picker")
				return 0, nil
			}))
			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetArgs(append([]string{"get"}, test.args...))
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			assertMRBibTeX(t, stdout.String(), test.wantKey)
			if arxivCalls != test.arxiv || doiCalls != test.doi || metadataCalls != test.metadata || providerCalls != test.provider || stderr.Len() != 0 {
				t.Fatalf("calls arxiv=%d doi=%d metadata=%d provider=%d; stderr=%q", arxivCalls, doiCalls, metadataCalls, providerCalls, stderr.String())
			}
		})
	}
}

func TestGetRejectsInvalidOptionsBeforeLookup(t *testing.T) {
	for _, args := range [][]string{
		{}, {"one", "two"}, {"serre local fields"}, {"https://example.org/2301.12345"},
		{"2301.12345", "--bib", "unknown"}, {"10.1000/example", "--published"},
		{"2301.12345", "--journal", "unknown"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			root := debugTestRoot(newGetCommand(func() getServices {
				t.Fatal("invalid input or option reached service initialization")
				return getServices{}
			}, nil))
			var stdout bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(io.Discard)
			root.SetIn(strings.NewReader(""))
			root.SetArgs(append([]string{"get"}, args...))
			if err := root.Execute(); err == nil || stdout.Len() != 0 {
				t.Fatalf("error=%v stdout=%q", err, stdout.String())
			}
		})
	}
}

func TestGetFailuresNeverFallbackOrPrintBibTeX(t *testing.T) {
	want := errors.New("service unavailable")
	for _, stage := range []string{"arxiv", "doi", "metadata", "provider", "conflicting DOI", "no publication DOI", "no entry"} {
		t.Run(stage, func(t *testing.T) {
			services := getServices{
				arxiv: func(context.Context, string) (bibliography.Record, error) {
					if stage == "arxiv" {
						return bibliography.Record{}, want
					}
					if stage == "no publication DOI" {
						return getPreprint(""), nil
					}
					t.Fatal("unexpected arXiv lookup")
					return bibliography.Record{}, nil
				},
				doi: func(context.Context, string) (bibliography.Record, error) {
					if stage == "doi" {
						return bibliography.Record{}, want
					}
					if stage == "no entry" {
						return bibliography.Record{}, nil
					}
					t.Fatal("unexpected DOI export or fallback")
					return bibliography.Record{}, nil
				},
				doiMetadata: func(context.Context, string) (bibliography.Work, error) {
					if stage == "metadata" {
						return bibliography.Work{}, want
					}
					return bibliography.Work{Title: "Selected work", DOI: "10.1000/example"}, nil
				},
				bib: map[string]bibliography.Provider{"mr": fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
					if stage == "provider" {
						return nil, want
					}
					if stage == "conflicting DOI" {
						return []bibliography.Record{searchRecord("Wrong", "10.1000/wrong")}, nil
					}
					t.Fatal("provider queried after a lookup failure")
					return nil, nil
				})},
			}
			args := []string{"get", "10.1000/example"}
			switch stage {
			case "arxiv":
				args = []string{"get", "2301.12345"}
			case "no publication DOI":
				args = []string{"get", "2301.12345", "--published"}
			case "metadata", "provider", "conflicting DOI":
				args = append(args, "--bib", "mr")
			}
			root := debugTestRoot(newGetCommand(func() getServices { return services }, nil))
			var stdout bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(io.Discard)
			root.SetArgs(args)
			err := root.Execute()
			if err == nil || stdout.Len() != 0 {
				t.Fatalf("error=%v stdout=%q", err, stdout.String())
			}
			if stage == "arxiv" || stage == "doi" || stage == "metadata" || stage == "provider" {
				if !errors.Is(err, want) {
					t.Fatalf("underlying error lost: %v", err)
				}
			}
		})
	}
}

func TestGetDebugAndJournalPreferencePreserveCleanStdout(t *testing.T) {
	services := getServices{
		doiMetadata: func(context.Context, string) (bibliography.Work, error) {
			return bibliography.Work{Title: "A selected work", DOI: "10.1000/example"}, nil
		},
		bib: map[string]bibliography.Provider{"mr": fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
			record := searchRecord("MR1", "10.1000/example")
			record.Journals = bibliography.JournalNames{Full: "Full Journal", Short: "Abbr. J."}
			return []bibliography.Record{record}, nil
		})},
	}
	root := debugTestRoot(newGetCommand(func() getServices { return services }, nil))
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"--debug", "get", "10.1000/example", "--bib", "mr", "--journal", "short"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	parsed, err := bibtex.Parse(strings.NewReader(stdout.String()))
	if err != nil || len(parsed.Entries) != 1 || parsed.Entries[0].Fields["journal"].String() != "Abbr. J." {
		t.Fatalf("output=%q error=%v", stdout.String(), err)
	}
	for _, want := range []string{"command=bibi get", "identifier_type=doi", "bib=mr", "exact identifier match", "get writing BibTeX"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("missing %q in trace: %s", want, stderr.String())
		}
	}
}

func TestGetCancellationAfterLookupProducesNoOutput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	services := getServices{doi: func(context.Context, string) (bibliography.Record, error) {
		cancel()
		return searchRecord("Cancelled", "10.1000/example"), nil
	}}
	root := debugTestRoot(newGetCommand(func() getServices { return services }, nil))
	root.SetContext(ctx)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"get", "10.1000/example"})
	if err := root.Execute(); !errors.Is(err, context.Canceled) || stdout.Len() != 0 {
		t.Fatalf("error=%v stdout=%q", err, stdout.String())
	}
}
