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
	"github.com/thofma/bibi/util"
)

func TestSearchPreservesProviderJournal(t *testing.T) {
	for _, provider := range []string{"zb", "mr", "crossref"} {
		t.Run(provider, func(t *testing.T) {
			original := "J. Number Theory"
			if provider == "crossref" {
				original = "Journal of Number Theory"
			}
			record := searchRecord("Selected", "10.1000/example")
			record.Entry.AddField("journal", bibtex.NewBibConst(original))
			services := searchServices{
				discovery: map[string]bibliography.Discoverer{"zb": fakeDiscovery(func(string) ([]bibliography.Work, error) {
					return []bibliography.Work{record.Work}, nil
				})},
				bib: map[string]bibliography.Provider{provider: fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
					return []bibliography.Record{record}, nil
				})},
			}
			root := debugTestRoot(newSearchCommand(func() searchServices { return services }, func(util.ChooserRequest) (int, error) {
				t.Fatal("export opened an extra picker")
				return 0, nil
			}))
			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetArgs([]string{"search", "work", "--bib", provider})
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			parsed, err := bibtex.Parse(strings.NewReader(stdout.String()))
			if err != nil {
				t.Fatal(err)
			}
			if got := parsed.Entries[0].Fields["journal"].String(); got != original {
				t.Errorf("journal = %q, want %q", got, original)
			}
			if record.Entry.Fields["journal"].String() != original || stderr.Len() != 0 {
				t.Errorf("source entry was changed or warning was emitted: %q", stderr.String())
			}
		})
	}
}

func TestRemovedJournalFlagFailsBeforeLookup(t *testing.T) {
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
			root.SetArgs([]string{command.Name(), "query", "--journal", "short"})
			if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "unknown flag: --journal") {
				t.Fatalf("error = %v, want unknown journal flag", err)
			}
			if stdout.Len() != 0 {
				t.Errorf("invalid option produced stdout: %s", stdout.String())
			}
		})
	}
}
