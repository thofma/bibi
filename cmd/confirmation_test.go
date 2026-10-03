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

func confirmationServices(work bibliography.Work, records ...bibliography.Record) searchServices {
	return searchServices{
		discovery: map[string]bibliography.Discoverer{"zb": fakeDiscovery(func(string) ([]bibliography.Work, error) { return []bibliography.Work{work}, nil })},
		bib:       map[string]bibliography.Provider{"mr": fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) { return records, nil })},
	}
}

func TestVerifiedCandidateRequiresReviewForEditionOrPublicationChange(t *testing.T) {
	for _, test := range []struct {
		name            string
		work, candidate bibliography.Work
		confirm         bool
	}{
		{"different edition", bibliography.Work{Type: "book", Edition: "1"}, bibliography.Work{Type: "book", Edition: "2"}, true},
		{"published preprint", bibliography.Work{Type: "preprint"}, bibliography.Work{Type: "article"}, true},
		{"ordinary year difference", bibliography.Work{Year: "2020"}, bibliography.Work{Year: "2024"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.work.DOI, test.candidate.DOI = "10.1000/work", "10.1000/work"
			record := searchRecord("Reviewed", "10.1000/work")
			record.Work = test.candidate
			picks := 0
			services := confirmationServices(test.work, record)
			command := newSearchCommand(func() searchServices { return services }, func(request util.ChooserRequest) (int, error) {
				picks++
				if !test.confirm || len(request.Choices) != 1 || !strings.Contains(request.Choices[0].Details, "Verified by DOI") || !strings.Contains(request.Choices[0].Details, "Review required:") {
					t.Fatalf("unexpected confirmation: %+v", request)
				}
				return 0, nil
			})
			var output bytes.Buffer
			command.SetOut(&output)
			command.SetErr(io.Discard)
			command.SetArgs([]string{"work", "--bib", "mr"})
			if err := command.Execute(); err != nil {
				t.Fatal(err)
			}
			if (picks == 1) != test.confirm {
				t.Fatalf("picker calls=%d", picks)
			}
			assertMRBibTeX(t, output.String(), "Reviewed")
		})
	}
}

func TestMultipleVerifiedCandidatesShowEvidenceAndExcludeUnverifiedOnes(t *testing.T) {
	work := bibliography.Work{DOI: "10.1000/work"}
	services := confirmationServices(work, searchRecord("Unverified", ""), searchRecord("First", work.DOI), searchRecord("Second", work.DOI))
	picks := 0
	command := newSearchCommand(func() searchServices { return services }, func(request util.ChooserRequest) (int, error) {
		picks++
		if len(request.Choices) != 2 || !strings.Contains(request.Choices[0].Details, "Verified by DOI") || !strings.Contains(request.Choices[1].Label, "[Second]") {
			t.Fatalf("wrong candidates: %+v", request)
		}
		return 1, nil
	})
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"work", "--bib", "mr"})
	if err := command.Execute(); err != nil || picks != 1 {
		t.Fatalf("error=%v picker calls=%d", err, picks)
	}
	assertMRBibTeX(t, output.String(), "Second")
}

func TestUnverifiedConfirmationFailureNeverExportsOrChangesBibliography(t *testing.T) {
	for _, mode := range []string{"cancel", "invalid selection", "no terminal", "context cancelled"} {
		for _, existing := range []bool{false, true} {
			t.Run(mode+"/existing="+map[bool]string{false: "false", true: "true"}[existing], func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "references.bib")
				const original = "% preserve formatting\n@misc{Existing,title={Existing}}\n"
				if existing {
					if err := os.WriteFile(path, []byte(original), 0600); err != nil {
						t.Fatal(err)
					}
				}
				services := confirmationServices(bibliography.Work{Title: "Selected"}, searchRecord("Uncertain", ""))
				ctx := context.Background()
				var cancel context.CancelFunc
				choose := func(request util.ChooserRequest) (int, error) {
					if len(request.Choices) != 1 || !strings.Contains(request.Choices[0].Details, "Selected work:") {
						t.Fatalf("missing single-candidate comparison: %+v", request)
					}
					switch mode {
					case "cancel":
						return -1, nil
					case "invalid selection":
						return 1, nil
					case "no terminal":
						return -1, util.ErrInteractiveTerminalUnavailable
					default:
						cancel()
						return 0, nil
					}
				}
				for _, command := range []string{"search", "add"} {
					ctx, cancel = context.WithCancel(context.Background())
					var output bytes.Buffer
					var err error
					if command == "search" {
						cmd := newSearchCommand(func() searchServices { return services }, choose)
						cmd.SetContext(ctx)
						cmd.SetOut(&output)
						cmd.SetErr(io.Discard)
						cmd.SetArgs([]string{"work", "--bib", "mr"})
						err = cmd.Execute()
					} else {
						cmd := newAddCommand(func() searchServices { return services }, choose)
						cmd.SetContext(ctx)
						cmd.SetOut(&output)
						cmd.SetErr(io.Discard)
						cmd.SetArgs([]string{"work", path, "--bib", "mr"})
						err = cmd.Execute()
					}
					if err == nil || output.Len() != 0 {
						t.Fatalf("%s exported without confirmation: output=%q error=%v", command, output.String(), err)
					}
					cancel()
				}
				data, err := os.ReadFile(path)
				if existing {
					if err != nil || string(data) != original {
						t.Fatalf("bibliography changed: %q %v", data, err)
					}
				} else if !os.IsNotExist(err) {
					t.Fatal("unconfirmed entry created a bibliography")
				}
			})
		}
	}
}

func TestGetBatchContinuesWhenConfirmationUnavailableWithoutConsumingStdin(t *testing.T) {
	var calls []string
	services := getServices{
		doiMetadata: func(_ context.Context, id string) (bibliography.Work, error) {
			calls = append(calls, id)
			return bibliography.Work{DOI: id, Title: "Selected"}, nil
		},
		bib: map[string]bibliography.Provider{"mr": fakeProvider(func(work bibliography.Work) ([]bibliography.Record, error) {
			if work.DOI == "10.1000/uncertain" {
				return []bibliography.Record{searchRecord("Unconfirmed", "")}, nil
			}
			return []bibliography.Record{searchRecord(strings.TrimPrefix(work.DOI, "10.1000/"), work.DOI)}, nil
		})},
	}
	picks := 0
	command := newGetCommand(func() getServices { return services }, func(util.ChooserRequest) (int, error) {
		picks++
		return -1, util.ErrInteractiveTerminalUnavailable
	})
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(io.Discard)
	command.SetIn(strings.NewReader("10.1000/first\n10.1000/uncertain\n10.1000/last\n"))
	command.SetArgs([]string{"--bib", "mr"})
	err := command.Execute()
	if !errors.Is(err, util.ErrInteractiveTerminalUnavailable) || picks != 1 || !reflect.DeepEqual(calls, []string{"10.1000/first", "10.1000/uncertain", "10.1000/last"}) {
		t.Fatalf("error=%v picks=%d calls=%v", err, picks, calls)
	}
	if got := getOutputKeys(t, output.String()); !reflect.DeepEqual(got, []string{"first", "last"}) {
		t.Fatalf("unconfirmed entry reached output: %v", got)
	}
}
