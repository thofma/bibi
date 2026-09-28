package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/nickng/bibtex"
	"github.com/spf13/cobra"
	"github.com/thofma/bibi/lib/phd"
)

func usePhDQuery(t *testing.T, query func(string) ([]phd.MGPEntry, error)) {
	t.Helper()
	originalQuery := phdQuery
	phdQuery = query
	t.Cleanup(func() {
		phdQuery = originalQuery
	})
}

func usePhDGetBibTeX(t *testing.T, getBibTeX func(phd.MGPEntry) (*bibtex.BibEntry, error)) {
	t.Helper()
	originalGetBibTeX := phdGetBibTeX
	phdGetBibTeX = getBibTeX
	t.Cleanup(func() {
		phdGetBibTeX = originalGetBibTeX
	})
}

func usePhDChoose(t *testing.T, choose func([]string) (int, error)) {
	t.Helper()
	originalChoose := phdChoose
	phdChoose = choose
	t.Cleanup(func() {
		phdChoose = originalChoose
	})
}

func TestRunPhDWritesOneBibTeXEntry(t *testing.T) {
	entry := phdTestEntry("Noether1907", "Noether, Emmy", "1907", "Invariantentheorie", "Göttingen")
	var gotName string
	usePhDQuery(t, func(name string) ([]phd.MGPEntry, error) {
		gotName = name
		return []phd.MGPEntry{entry}, nil
	})
	usePhDGetBibTeX(t, func(got phd.MGPEntry) (*bibtex.BibEntry, error) {
		if got != entry {
			t.Errorf("selected entry = %#v, want %#v", got, entry)
		}
		return got.BibTeX, nil
	})
	usePhDChoose(t, func([]string) (int, error) {
		t.Fatal("runPhD() opened a chooser for one result")
		return 0, nil
	})

	command := &cobra.Command{}
	var output bytes.Buffer
	command.SetOut(&output)
	if err := runPhD(command, []string{"emmy", "noether"}); err != nil {
		t.Fatalf("runPhD() error = %v", err)
	}
	if got, want := gotName, "emmy noether"; got != want {
		t.Errorf("query = %q, want %q", got, want)
	}
	if strings.Contains(output.String(), "phd called") {
		t.Errorf("output includes debug text: %q", output.String())
	}
	assertPhDBibTeX(t, output.String(), "Noether1907")
}

func TestRunPhDChoosesAndHandlesCancellation(t *testing.T) {
	entries := []phd.MGPEntry{
		phdTestEntry("First", "Noether, Emmy", "1907", "First thesis", "Göttingen"),
		phdTestEntry("Second", "Noether, Fritz", "1911", "Selected thesis", "Erlangen"),
	}
	usePhDQuery(t, func(string) ([]phd.MGPEntry, error) {
		return entries, nil
	})
	usePhDGetBibTeX(t, func(entry phd.MGPEntry) (*bibtex.BibEntry, error) {
		return entry.BibTeX, nil
	})
	usePhDChoose(t, func(choices []string) (int, error) {
		if got, want := len(choices), 2; got != want {
			t.Errorf("choice count = %d, want %d", got, want)
		}
		if !strings.Contains(choices[1], "Noether, Fritz, 1911, Erlangen") {
			t.Errorf("second choice = %q, want full entry label", choices[1])
		}
		return 1, nil
	})

	command := &cobra.Command{}
	var output bytes.Buffer
	command.SetOut(&output)
	if err := runPhD(command, []string{"noether"}); err != nil {
		t.Fatalf("runPhD() error = %v", err)
	}
	assertPhDBibTeX(t, output.String(), "Second")

	usePhDChoose(t, func([]string) (int, error) { return -1, nil })
	if err := runPhD(&cobra.Command{}, []string{"cancel"}); err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("runPhD() error = %v, want cancellation error", err)
	}
}

func TestRunPhDReportsErrors(t *testing.T) {
	t.Run("no entries", func(t *testing.T) {
		usePhDQuery(t, func(string) ([]phd.MGPEntry, error) {
			return nil, nil
		})
		if err := runPhD(&cobra.Command{}, []string{"missing"}); err == nil || !strings.Contains(err.Error(), "no PhD theses") {
			t.Fatalf("runPhD() error = %v, want empty-result error", err)
		}
	})

	t.Run("query error", func(t *testing.T) {
		want := errors.New("network unavailable")
		usePhDQuery(t, func(string) ([]phd.MGPEntry, error) {
			return nil, want
		})
		err := runPhD(&cobra.Command{}, []string{"failure"})
		if !errors.Is(err, want) {
			t.Fatalf("runPhD() error = %v, want %v", err, want)
		}
	})

	t.Run("BibTeX error", func(t *testing.T) {
		entry := phdTestEntry("Broken", "Doe, Jane", "2025", "Broken", "Example University")
		want := errors.New("bad response")
		usePhDQuery(t, func(string) ([]phd.MGPEntry, error) {
			return []phd.MGPEntry{entry}, nil
		})
		usePhDGetBibTeX(t, func(phd.MGPEntry) (*bibtex.BibEntry, error) {
			return nil, want
		})
		err := runPhD(&cobra.Command{}, []string{"broken"})
		if !errors.Is(err, want) {
			t.Fatalf("runPhD() error = %v, want %v", err, want)
		}
	})

	t.Run("chooser error", func(t *testing.T) {
		entries := []phd.MGPEntry{
			phdTestEntry("First", "Doe, Jane", "2025", "First", "Example University"),
			phdTestEntry("Second", "Doe, John", "2026", "Second", "Example University"),
		}
		want := errors.New("terminal unavailable")
		usePhDQuery(t, func(string) ([]phd.MGPEntry, error) {
			return entries, nil
		})
		usePhDChoose(t, func([]string) (int, error) { return -1, want })
		err := runPhD(&cobra.Command{}, []string{"chooser"})
		if !errors.Is(err, want) {
			t.Fatalf("runPhD() error = %v, want %v", err, want)
		}
	})
}

func TestPhDCommandRequiresAnArgument(t *testing.T) {
	if err := phdCmd.Args(phdCmd, nil); err == nil {
		t.Fatal("phd command accepted no arguments")
	}
}

func phdTestEntry(citeName, author, year, title, university string) phd.MGPEntry {
	bib := bibtex.NewBibEntry("phdthesis", citeName)
	bib.AddField("author", bibtex.NewBibConst(author))
	bib.AddField("title", bibtex.NewBibConst(title))
	bib.AddField("year", bibtex.NewBibConst(year))
	bib.AddField("school", bibtex.NewBibConst(university))
	return phd.MGPEntry{
		Author:     author,
		Year:       year,
		University: university,
		Title:      title,
		BibTeX:     bib,
	}
}

func assertPhDBibTeX(t *testing.T, output, citeName string) {
	t.Helper()
	parsed, err := bibtex.Parse(strings.NewReader(output))
	if err != nil {
		t.Fatalf("invalid BibTeX output: %v\n%s", err, output)
	}
	if len(parsed.Entries) != 1 || parsed.Entries[0].CiteName != citeName {
		t.Fatalf("output entry = %#v, want %s", parsed.Entries, citeName)
	}
}
