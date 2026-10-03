package cmd

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/lib/bibliography"
	"github.com/thofma/bibi/lib/mr"
	"github.com/thofma/bibi/lib/phd"
	"github.com/thofma/bibi/lib/zb"
	"github.com/thofma/bibi/util"
)

func debugTestRoot(command *cobra.Command) *cobra.Command {
	root := &cobra.Command{Use: "bibi", SilenceUsage: true}
	addDebugFlag(root)
	addBibTeXFlags(root)
	root.AddCommand(command)
	return root
}

func TestSearchDebugFlagKeepsBibTeXOnStdout(t *testing.T) {
	var normalOutput string
	for _, test := range []struct {
		name    string
		args    []string
		enabled bool
	}{
		{"default quiet", []string{"search", "10.1000/example", "--bib", "mr"}, false},
		{"before command", []string{"--debug", "search", "10.1000/example", "--bib", "mr"}, true},
		{"after command", []string{"search", "10.1000/example", "--bib", "mr", "--debug"}, true},
		{"explicitly disabled", []string{"search", "10.1000/example", "--bib", "mr", "--debug=false"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			services := searchServices{
				discovery: map[string]bibliography.Discoverer{"zb": fakeDiscovery(func(string) ([]bibliography.Work, error) {
					return []bibliography.Work{{Title: "A selected work", DOI: "10.1000/example"}}, nil
				})},
				bib: map[string]bibliography.Provider{"mr": fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
					return []bibliography.Record{searchRecord("Wrong", "10.1000/other"), searchRecord("Uncertain", ""), searchRecord("MR1", "10.1000/example")}, nil
				})},
			}
			root := debugTestRoot(newSearchCommand(func() searchServices { return services }, nil))
			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetArgs(test.args)
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			assertMRBibTeX(t, stdout.String(), "MR1")
			if normalOutput == "" {
				normalOutput = stdout.String()
			} else if stdout.String() != normalOutput {
				t.Fatalf("debug mode changed stdout: %q", stdout.String())
			}
			if test.enabled {
				for _, want := range []string{
					"[debug] command=bibi search", "discovery=zb bib=mr", `query_type="DOI"`,
					"returned 1 works", "selected discovery result 1", "returned 3 candidates",
					"rejected, conflicting DOI", "compatible without identifier verification", "exact identifier match",
					"writing mr BibTeX entry MR1", "command completed in",
				} {
					if !strings.Contains(stderr.String(), want) {
						t.Errorf("debug output = %q, want %q", stderr.String(), want)
					}
				}
			} else if stderr.Len() != 0 {
				t.Errorf("normal mode stderr = %q, want empty", stderr.String())
			}
			if diagnostics.Enabled() {
				t.Fatal("debug logging remained enabled after command completion")
			}
		})
	}
}

func TestDebugFailureIsReportedAndLoggerRestored(t *testing.T) {
	want := errors.New("discovery unavailable")
	services := searchServices{
		discovery: map[string]bibliography.Discoverer{"zb": fakeDiscovery(func(string) ([]bibliography.Work, error) { return nil, want })},
		bib: map[string]bibliography.Provider{"zb": fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
			t.Fatal("retrieved BibTeX after discovery failure")
			return nil, nil
		})},
	}
	root := debugTestRoot(newSearchCommand(func() searchServices { return services }, nil))
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"search", "anything 1999", "--debug"})
	if err := root.Execute(); !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "command failed after") || !strings.Contains(stderr.String(), want.Error()) {
		t.Fatalf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
	if diagnostics.Enabled() {
		t.Fatal("debug logging remained enabled after failure")
	}
}

func TestDebugExplainsConfirmedUnverifiedCandidate(t *testing.T) {
	services := searchServices{
		discovery: map[string]bibliography.Discoverer{"zb": fakeDiscovery(func(string) ([]bibliography.Work, error) {
			return []bibliography.Work{{Title: "A selected work"}}, nil
		})},
		bib: map[string]bibliography.Provider{"mr": fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
			return []bibliography.Record{searchRecord("MR1", "10.1000/example")}, nil
		})},
	}
	picks := 0
	root := debugTestRoot(newSearchCommand(func() searchServices { return services }, func(util.ChooserRequest) (int, error) {
		picks++
		return 0, nil
	}))
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"search", "work", "--bib", "mr", "--debug"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	assertMRBibTeX(t, stdout.String(), "MR1")
	for _, want := range []string{
		"compatible without identifier verification", "the discovered work has no DOI",
		"stage=matching exact=0 compatible=1 rejected=0",
		"stage=confirmation started: 1 remaining candidates",
		"stage=confirmation succeeded: key=\"MR1\"",
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("trace = %q, want %q", stderr.String(), want)
		}
	}
	if picks != 1 || strings.Contains(stderr.String(), "stage=selection automatic") {
		t.Errorf("unverified match bypassed confirmation: picks=%d trace=%s", picks, stderr.String())
	}
}

type failedDebugOutput struct{}

func (failedDebugOutput) Write([]byte) (int, error) { return 0, errors.New("stdout is closed") }

func TestDebugExplainsBibFailureStage(t *testing.T) {
	for _, test := range []struct {
		name        string
		records     []bibliography.Record
		providerErr error
		noDiscovery bool
		outputError bool
		want        string
	}{
		{name: "lookup error", providerErr: errors.New("service unavailable"), want: "stage=retrieval failed: service unavailable"},
		{name: "no candidates", want: "stage=retrieval failed: provider supplied no usable BibTeX candidates"},
		{name: "conflicting DOI", records: []bibliography.Record{searchRecord("Wrong", "10.1000/other")}, want: "stage=matching failed: all 1 candidates were rejected for conflicting DOIs"},
		{name: "missing export", records: []bibliography.Record{{Work: bibliography.Work{Title: "Work"}}}, want: "stage=export failed: candidate 1 contains metadata but no BibTeX entry"},
		{name: "confirmation cancelled", records: []bibliography.Record{searchRecord("Uncertain1", ""), searchRecord("Uncertain2", "")}, want: "stage=confirmation failed: selection cancelled; provider lookup succeeded"},
		{name: "no discovered work", noDiscovery: true, want: "bib provider=mr not queried: discovery returned no works"},
		{name: "output failed", records: []bibliography.Record{searchRecord("MR1", "10.1000/example")}, outputError: true, want: "stage=output failed: write BibTeX entry: stdout is closed; provider lookup and matching succeeded"},
	} {
		t.Run(test.name, func(t *testing.T) {
			services := searchServices{
				discovery: map[string]bibliography.Discoverer{"zb": fakeDiscovery(func(string) ([]bibliography.Work, error) {
					if test.noDiscovery {
						return nil, nil
					}
					return []bibliography.Work{{Title: "Work", Authors: []string{"Doe, Jane", "Roe, John"}, Year: "1999", DOI: "https://doi.org/10.1000/EXAMPLE"}}, nil
				})},
				bib: map[string]bibliography.Provider{"mr": fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
					if test.noDiscovery {
						t.Fatal("provider was queried without a work")
					}
					return test.records, test.providerErr
				})},
			}
			root := debugTestRoot(newSearchCommand(func() searchServices { return services }, func(util.ChooserRequest) (int, error) { return -1, nil }))
			var stderr bytes.Buffer
			root.SetErr(&stderr)
			var stdout io.Writer = io.Discard
			if test.outputError {
				stdout = failedDebugOutput{}
			}
			root.SetOut(stdout)
			root.SetArgs([]string{"search", "work", "--bib", "mr", "--debug"})
			if err := root.Execute(); err == nil {
				t.Fatal("expected failure")
			}
			if !strings.Contains(stderr.String(), test.want) {
				t.Errorf("trace = %q, want %q", stderr.String(), test.want)
			}
			if !test.noDiscovery && !strings.Contains(stderr.String(), `authors=["Doe, Jane" "Roe, John"] year="1999"`) {
				t.Errorf("trace omits complete selected metadata: %s", stderr.String())
			}
		})
	}
}

func TestLegacyCommandsSupportDebugFlag(t *testing.T) {
	useZBSearch(t, func(string) (zb.Response, error) {
		return zb.Response{Result: []zb.Item{journalResult(1, "Work")}}, nil
	})
	useMRQuery(t, func(string, string, string) ([]*mr.Entry, error) {
		return []*mr.Entry{mrTestEntry("MR1", "Work", "1999")}, nil
	})
	usePhDQuery(t, func(string) ([]phd.MGPEntry, error) {
		return []phd.MGPEntry{phdTestEntry("PhD1", "Doe, Jane", "1999", "Thesis", "University")}, nil
	})
	usePhDGetBibTeX(t, phd.MGPEntryGetBibtex)
	for _, command := range []*cobra.Command{zbCmd, hexhexCmd, mrCmd, phdCmd} {
		t.Run(command.Name(), func(t *testing.T) {
			// Reuse the production handler without moving the global command to a new parent.
			root := debugTestRoot(&cobra.Command{Use: command.Use, RunE: command.RunE, Args: command.Args})
			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetArgs([]string{command.Name(), "fixture", "--debug"})
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(stdout.String(), "[debug]") || !strings.HasPrefix(stdout.String(), "@") || !strings.Contains(stderr.String(), "[debug] command=bibi "+command.Name()) {
				t.Fatalf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
			}
		})
	}
}
