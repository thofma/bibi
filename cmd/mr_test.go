package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/nickng/bibtex"
	"github.com/spf13/cobra"
	"github.com/thofma/bibi/lib/mr"
)

func useMRQuery(t *testing.T, query func(string, string, string) ([]*mr.Entry, error)) {
	t.Helper()
	originalQuery := mrQuery
	mrQuery = query
	t.Cleanup(func() {
		mrQuery = originalQuery
	})
}

func useMRChoose(t *testing.T, choose func([]string) (int, error)) {
	t.Helper()
	originalChoose := mrChoose
	mrChoose = choose
	t.Cleanup(func() {
		mrChoose = originalChoose
	})
}

func TestRunMRWritesOneBibTeXEntry(t *testing.T) {
	var gotAuthor, gotYear, gotTitle string
	useMRQuery(t, func(author, year, title string) ([]*mr.Entry, error) {
		gotAuthor, gotYear, gotTitle = author, year, title
		return []*mr.Entry{mrTestEntry("MR1", "A course in arithmetic", "1973", "Serre, Jean-Pierre")}, nil
	})
	useMRChoose(t, func([]string) (int, error) {
		t.Fatal("runMR() opened a chooser for one result")
		return 0, nil
	})

	command := &cobra.Command{}
	var output bytes.Buffer
	command.SetOut(&output)
	if err := runMR(command, []string{"serre", "-", "1973"}); err != nil {
		t.Fatalf("runMR() error = %v", err)
	}
	if got, want := gotAuthor, "serre"; got != want {
		t.Errorf("author = %q, want %q", got, want)
	}
	if got, want := gotTitle, ""; got != want {
		t.Errorf("title = %q, want empty", got)
	}
	if got, want := gotYear, "1973"; got != want {
		t.Errorf("year = %q, want %q", got, want)
	}
	assertMRBibTeX(t, output.String(), "MR1")
}

func TestRunMRChoosesAndHandlesCancellation(t *testing.T) {
	entries := []*mr.Entry{
		mrTestEntry("MR1", "First result", "2025"),
		mrTestEntry("MR2", "Selected result", "2026", "Doe, Jane", "Roe, John"),
	}
	useMRQuery(t, func(string, string, string) ([]*mr.Entry, error) {
		return entries, nil
	})
	useMRChoose(t, func(choices []string) (int, error) {
		if got, want := len(choices), 2; got != want {
			t.Errorf("choice count = %d, want %d", got, want)
		}
		if !strings.Contains(choices[0], "Unknown author") {
			t.Errorf("first choice = %q, want unknown-author label", choices[0])
		}
		if !strings.Contains(choices[1], "Doe, Jane et al.") {
			t.Errorf("second choice = %q, want author label", choices[1])
		}
		return 1, nil
	})

	command := &cobra.Command{}
	var output bytes.Buffer
	command.SetOut(&output)
	if err := runMR(command, []string{"multiple"}); err != nil {
		t.Fatalf("runMR() error = %v", err)
	}
	assertMRBibTeX(t, output.String(), "MR2")

	useMRChoose(t, func([]string) (int, error) { return -1, nil })
	if err := runMR(&cobra.Command{}, []string{"cancel"}); err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("runMR() error = %v, want cancellation error", err)
	}
}

func TestRunMRReportsErrors(t *testing.T) {
	t.Run("no entries", func(t *testing.T) {
		useMRQuery(t, func(string, string, string) ([]*mr.Entry, error) {
			return nil, nil
		})
		if err := runMR(&cobra.Command{}, []string{"missing"}); err == nil || !strings.Contains(err.Error(), "no MR entries") {
			t.Fatalf("runMR() error = %v, want empty-result error", err)
		}
	})

	t.Run("query error", func(t *testing.T) {
		want := errors.New("network unavailable")
		useMRQuery(t, func(string, string, string) ([]*mr.Entry, error) {
			return nil, want
		})
		err := runMR(&cobra.Command{}, []string{"failure"})
		if !errors.Is(err, want) {
			t.Fatalf("runMR() error = %v, want %v", err, want)
		}
	})

	t.Run("chooser error", func(t *testing.T) {
		want := errors.New("terminal unavailable")
		useMRQuery(t, func(string, string, string) ([]*mr.Entry, error) {
			return []*mr.Entry{mrTestEntry("MR1", "First", "2025"), mrTestEntry("MR2", "Second", "2026")}, nil
		})
		useMRChoose(t, func([]string) (int, error) { return -1, want })
		err := runMR(&cobra.Command{}, []string{"chooser"})
		if !errors.Is(err, want) {
			t.Fatalf("runMR() error = %v, want %v", err, want)
		}
	})
}

func TestMRCommandAcceptsOneToThreeArguments(t *testing.T) {
	if err := mrCmd.Args(mrCmd, nil); err == nil {
		t.Fatal("mr command accepted no arguments")
	}
	if err := mrCmd.Args(mrCmd, []string{"a", "b", "c", "d"}); err == nil {
		t.Fatal("mr command accepted four arguments")
	}
}

func mrTestEntry(citeName, title, year string, authors ...string) *mr.Entry {
	bib := bibtex.NewBibEntry("article", citeName)
	bib.AddField("title", bibtex.NewBibConst(title))
	bib.AddField("year", bibtex.NewBibConst(year))
	if len(authors) > 0 {
		bib.AddField("author", bibtex.NewBibConst(strings.Join(authors, " and ")))
	}
	return &mr.Entry{Authors: authors, Title: title, Year: year, BibTeX: bib}
}

func assertMRBibTeX(t *testing.T, output, citeName string) {
	t.Helper()
	parsed, err := bibtex.Parse(strings.NewReader(output))
	if err != nil {
		t.Fatalf("invalid BibTeX output: %v\n%s", err, output)
	}
	if len(parsed.Entries) != 1 || parsed.Entries[0].CiteName != citeName {
		t.Fatalf("output entry = %#v, want %s", parsed.Entries, citeName)
	}
}
