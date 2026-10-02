package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nickng/bibtex"
	"github.com/spf13/cobra"
	"github.com/thofma/bibi/lib/bibliography"
	"github.com/thofma/bibi/lib/mr"
	"github.com/thofma/bibi/lib/phd"
	"github.com/thofma/bibi/lib/zb"
)

func TestSearchJournalPreferencesAreProviderIndependent(t *testing.T) {
	for _, provider := range []string{"zb", "mr", "crossref"} {
		for _, style := range []string{"source", "short", "full"} {
			t.Run(provider+"/"+style, func(t *testing.T) {
				names := bibliography.JournalNames{Full: "Journal of Number Theory", Short: "J. Number Theory"}
				original := names.Short
				if provider == "crossref" {
					original = names.Full
				}
				record := searchRecord("Selected", "10.1000/example")
				record.Entry.AddField("journal", bibtex.NewBibConst(original))
				record.Journals = names
				services := searchServices{
					discovery: map[string]bibliography.Discoverer{"zb": fakeDiscovery(func(string) ([]bibliography.Work, error) {
						return []bibliography.Work{record.Work}, nil
					})},
					bib: map[string]bibliography.Provider{provider: fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
						return []bibliography.Record{record}, nil
					})},
				}
				root := debugTestRoot(newSearchCommand(func() searchServices { return services }, func([]string) (int, error) {
					t.Fatal("journal preference opened a picker")
					return 0, nil
				}))
				var stdout, stderr bytes.Buffer
				root.SetOut(&stdout)
				root.SetErr(&stderr)
				root.SetArgs([]string{"--journal", style, "search", "work", "--bib", provider})
				if err := root.Execute(); err != nil {
					t.Fatal(err)
				}
				parsed, err := bibtex.Parse(strings.NewReader(stdout.String()))
				if err != nil {
					t.Fatal(err)
				}
				want := original
				if style == "short" {
					want = names.Short
				} else if style == "full" {
					want = names.Full
				}
				if got := parsed.Entries[0].Fields["journal"].String(); got != want {
					t.Errorf("journal = %q, want %q", got, want)
				}
				if record.Entry.Fields["journal"].String() != original || stderr.Len() != 0 {
					t.Errorf("source entry was changed or warning was emitted: %q", stderr.String())
				}
			})
		}
	}
}

func TestInvalidJournalPreferenceFailsBeforeLookup(t *testing.T) {
	useZBSearch(t, func(string) (zb.Response, error) { t.Fatal("invalid option queried zbMATH"); return zb.Response{}, nil })
	useMRQuery(t, func(string, string, string) ([]*mr.Entry, error) {
		t.Fatal("invalid option queried MR")
		return nil, nil
	})
	usePhDQuery(t, func(string) ([]phd.MGPEntry, error) { t.Fatal("invalid option queried MGP"); return nil, nil })
	search := newSearchCommand(func() searchServices { t.Fatal("invalid option initialized discovery"); return searchServices{} }, nil)
	for _, command := range []*cobra.Command{search, zbCmd, mrCmd, phdCmd, hexhexCmd} {
		t.Run(command.Name(), func(t *testing.T) {
			root := debugTestRoot(&cobra.Command{Use: command.Use, RunE: command.RunE, Args: command.Args})
			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetArgs([]string{command.Name(), "query", "--journal", "invent"})
			if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "unknown journal style") {
				t.Fatalf("error = %v, want invalid journal option", err)
			}
			if stdout.Len() != 0 {
				t.Errorf("invalid option produced stdout: %s", stdout.String())
			}
		})
	}
}
