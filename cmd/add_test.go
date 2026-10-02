package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nickng/bibtex"
	"github.com/thofma/bibi/lib/bibliography"
	"github.com/thofma/bibi/util"
)

func addTestServices(record bibliography.Record, discovery, provider string) searchServices {
	return searchServices{
		discovery: map[string]bibliography.Discoverer{discovery: fakeDiscovery(func(string) ([]bibliography.Work, error) { return []bibliography.Work{record.Work}, nil })},
		bib:       map[string]bibliography.Provider{provider: fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) { return []bibliography.Record{record}, nil })},
	}
}

func runAddTest(t *testing.T, args []string, services searchServices, choose func(util.ChooserRequest) (int, error)) (string, string, error) {
	t.Helper()
	if choose == nil {
		choose = func(util.ChooserRequest) (int, error) { t.Fatal("unexpected extra selector"); return 0, nil }
	}
	command := newAddCommand(func() searchServices { return services }, choose)
	addBibTeXFlags(command)
	addDebugFlag(command)
	var output, stderr bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&stderr)
	command.SetArgs(args)
	err := command.Execute()
	return output.String(), stderr.String(), err
}

func TestAddProviderChoicesAndDuplicateNoOp(t *testing.T) {
	for _, discovery := range []string{"zb", "crossref"} {
		for _, provider := range []string{"zb", "mr", "crossref"} {
			t.Run(discovery+" to "+provider, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "references.bib")
				record := searchRecord(provider+"Native", "10.1000/example")
				record.Entry.AddField("doi", bibtex.NewBibConst("10.1000/example"))
				services := addTestServices(record, discovery, provider)
				args := []string{"anything can go in here 1999", path, "--discovery", discovery, "--bib", provider}
				output, stderr, err := runAddTest(t, args, services, nil)
				if err != nil || output != "" || !strings.Contains(stderr, "Added "+provider+"Native") {
					t.Fatalf("output=%q stderr=%q error=%v", output, stderr, err)
				}
				before, _ := os.ReadFile(path)
				assertMRBibTeX(t, string(before), provider+"Native")
				output, stderr, err = runAddTest(t, args, services, nil)
				if err != nil || output != "" || !strings.Contains(stderr, "Already present") {
					t.Fatalf("duplicate output=%q stderr=%q error=%v", output, stderr, err)
				}
				after, _ := os.ReadFile(path)
				if !bytes.Equal(before, after) {
					t.Fatal("duplicate modified bibliography")
				}
			})
		}
	}
}

func TestAddKeyDryRunAndJournalPreference(t *testing.T) {
	path := filepath.Join(t.TempDir(), "references.bib")
	record := searchRecord("NativeKey", "")
	record.Journals = bibliography.JournalNames{Full: "Full Journal Name", Short: "Full J."}
	services := addTestServices(record, "zb", "mr")
	args := []string{"local fields", path, "--bib", "mr", "--key", "Chosen:1999", "--journal", "short", "--dry-run"}
	output, stderr, err := runAddTest(t, args, services, nil)
	if err != nil || !strings.Contains(stderr, "Would add Chosen:1999") {
		t.Fatalf("stderr=%q error=%v", stderr, err)
	}
	assertMRBibTeX(t, output, "Chosen:1999")
	if !strings.Contains(output, "Full J.") {
		t.Fatalf("journal preference missing: %q", output)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("dry run created destination")
	}
	if record.Entry.CiteName != "NativeKey" || record.Entry.Fields["journal"].String() != "Example Journal" {
		t.Fatal("mutated source entry")
	}
	output, stderr, err = runAddTest(t, args[:len(args)-1], services, nil)
	if err != nil || output != "" || !strings.Contains(stderr, "Added Chosen:1999") {
		t.Fatalf("output=%q stderr=%q error=%v", output, stderr, err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "Full J.") {
		t.Fatal("saved wrong journal")
	}
}

func TestAddValidationBeforeNetwork(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "broken.bib")
	if err := os.WriteFile(broken, []byte("@misc{Broken,title={"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{}, {"query"}, {"local", "fields", "references.bib"}, {"query", "file", "extra"},
		{" ", "references.bib"}, {"query", " "}, {"query", broken}, {"query", dir},
		{"query", filepath.Join(dir, "absent", "refs.bib")},
		{"query", filepath.Join(dir, "refs.bib"), "--key", "bad key"},
		{"query", filepath.Join(dir, "refs.bib"), "--key", ""},
		{"query", filepath.Join(dir, "refs.bib"), "--journal", "invented"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			command := newAddCommand(func() searchServices {
				t.Fatal("performed lookup before validating add arguments/file")
				return searchServices{}
			}, nil)
			addBibTeXFlags(command)
			var output, stderr bytes.Buffer
			command.SetOut(&output)
			command.SetErr(&stderr)
			command.SetArgs(args)
			err := command.Execute()
			if err == nil || output.Len() != 0 {
				t.Fatalf("error=%v output=%q", err, output.String())
			}
			if len(args) != 2 && !strings.Contains(strings.Join(args, " "), "--") && !strings.Contains(err.Error(), "quote multiword queries") {
				t.Fatalf("no quoting hint: %v", err)
			}
		})
	}
}

func TestAddUsesExportedIdentifiersAndLatestFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "references.bib")
	record := searchRecord("NativeKey", "10.1000/discovered")
	record.Entry.AddField("doi", bibtex.NewBibConst("10.1000/exported"))
	services := addTestServices(record, "zb", "mr")
	services.bib["mr"] = fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
		// Simulate editing the bibliography while the lookup/selection is running.
		if err := os.WriteFile(path, []byte("@misc{Existing,doi={10.1000/discovered}}\n"), 0600); err != nil {
			t.Fatal(err)
		}
		return []bibliography.Record{record}, nil
	})
	_, stderr, err := runAddTest(t, []string{"query", path, "--bib", "mr", "--debug"}, services, nil)
	if err != nil || !strings.Contains(stderr, "stage=save succeeded") {
		t.Fatalf("stderr=%q error=%v", stderr, err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "Existing") || !strings.Contains(string(data), "NativeKey") {
		t.Fatalf("used discovery DOI or lost a recent edit: %q", data)
	}
	// The next lookup sees a recently added matching export and uses its existing key.
	services.bib["mr"] = fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
		if err := os.WriteFile(path, []byte("@misc{MyExistingKey,doi={10.1000/exported}}\n"), 0600); err != nil {
			t.Fatal(err)
		}
		return []bibliography.Record{record}, nil
	})
	output, stderr, err := runAddTest(t, []string{"query", path, "--bib", "mr", "--dry-run"}, services, nil)
	if err != nil || output != "" || !strings.Contains(stderr, "MyExistingKey") || !strings.Contains(stderr, "Already present") {
		t.Fatalf("output=%q stderr=%q error=%v", output, stderr, err)
	}
}

func TestAddLaterPageHasOneDiscoverySelection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "references.bib")
	record := searchRecord("Later", "")
	services := addTestServices(record, "zb", "mr")
	services.discovery["zb"] = fakePagedDiscovery(func(ctx context.Context, query, token string) (bibliography.SearchPage, error) {
		if token == "" {
			return bibliography.SearchPage{Works: []bibliography.Work{{Title: "First"}}, NextToken: "next"}, nil
		}
		return bibliography.SearchPage{Works: []bibliography.Work{record.Work}}, nil
	})
	picks := 0
	_, _, err := runAddTest(t, []string{"query", path, "--bib", "mr"}, services, func(request util.ChooserRequest) (int, error) {
		picks++
		if picks != 1 || request.LoadPage == nil {
			t.Fatal("unexpected extra selector")
		}
		if _, err := request.LoadPage(context.Background(), "next"); err != nil {
			t.Fatal(err)
		}
		return 1, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	assertMRBibTeX(t, string(data), "Later")
}

func TestAddLookupFailureOrCancellationLeavesFileUntouched(t *testing.T) {
	for _, mode := range []string{"provider fails", "cancel picker", "missing destination"} {
		t.Run(mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "references.bib")
			if mode != "missing destination" {
				if err := os.WriteFile(path, []byte("% preserve me\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			record := searchRecord("Native", "")
			services := addTestServices(record, "zb", "mr")
			services.bib["mr"] = fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) { return nil, errors.New("provider unavailable") })
			var choose func(util.ChooserRequest) (int, error)
			if mode == "cancel picker" {
				services.discovery["zb"] = fakeDiscovery(func(string) ([]bibliography.Work, error) {
					return []bibliography.Work{{Title: "One"}, {Title: "Two"}}, nil
				})
				choose = func(util.ChooserRequest) (int, error) { return -1, nil }
				services.bib["mr"] = fakeProvider(func(bibliography.Work) ([]bibliography.Record, error) {
					t.Fatal("provider called after cancellation")
					return nil, nil
				})
			}
			output, _, err := runAddTest(t, []string{"query", path, "--bib", "mr"}, services, choose)
			if err == nil || output != "" {
				t.Fatalf("output=%q error=%v", output, err)
			}
			data, err := os.ReadFile(path)
			if mode == "missing destination" {
				if !os.IsNotExist(err) {
					t.Fatal("failed lookup created file")
				}
			} else if err != nil || string(data) != "% preserve me\n" {
				t.Fatalf("modified file: %q %v", data, err)
			}
		})
	}
}
