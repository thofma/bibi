package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/nickng/bibtex"
	"github.com/spf13/cobra"
	"github.com/thofma/bibi/lib/zb"
	"github.com/thofma/bibi/util"
)

func useZBSearch(t *testing.T, search func(string) (zb.Response, error)) {
	t.Helper()
	originalSearch := zbSearch
	zbSearch = search
	t.Cleanup(func() {
		zbSearch = originalSearch
	})
}

func useZBChoose(t *testing.T, choose func([]string) (int, error)) {
	t.Helper()
	originalChoose := zbChoose
	zbChoose = func(request util.ChooserRequest) (int, error) { return choose(choiceLabels(request)) }
	t.Cleanup(func() {
		zbChoose = originalChoose
	})
}

func TestRunZBWritesOneBibTeXEntry(t *testing.T) {
	var receivedQuery string
	useZBSearch(t, func(query string) (zb.Response, error) {
		receivedQuery = query
		return zb.Response{Result: []zb.Item{{
			DocumentType: zb.DocumentType{Code: "j"},
			ID:           1,
			Title:        zb.Title{Title: "A local result"},
			Year:         "2026",
		}}}, nil
	})
	useZBChoose(t, func([]string) (int, error) {
		t.Fatal("runZB() opened a chooser for one result")
		return 0, nil
	})

	command := &cobra.Command{}
	var output bytes.Buffer
	command.SetOut(&output)
	if err := runZB(command, []string{"local", "result"}); err != nil {
		t.Fatalf("runZB() error = %v", err)
	}
	if got, want := receivedQuery, "local result"; got != want {
		t.Errorf("query = %q, want %q", got, want)
	}
	parsed, err := bibtex.Parse(strings.NewReader(output.String()))
	if err != nil {
		t.Fatalf("runZB() wrote invalid BibTeX: %v\n%s", err, output.String())
	}
	if len(parsed.Entries) != 1 {
		t.Fatalf("runZB() wrote %d entries, want 1", len(parsed.Entries))
	}
}

func TestRunZBChoosesAmongMultipleResults(t *testing.T) {
	useZBSearch(t, func(string) (zb.Response, error) {
		return zb.Response{Result: []zb.Item{
			journalResult(1, "First result"),
			journalResult(2, "Selected result"),
		}}, nil
	})
	useZBChoose(t, func(choices []string) (int, error) {
		if got, want := len(choices), 2; got != want {
			t.Errorf("choice count = %d, want %d", got, want)
		}
		if !strings.Contains(choices[1], "Selected result") {
			t.Errorf("second choice = %q, want selected title", choices[1])
		}
		return 1, nil
	})

	command := &cobra.Command{}
	var output bytes.Buffer
	command.SetOut(&output)
	if err := runZB(command, []string{"multiple"}); err != nil {
		t.Fatalf("runZB() error = %v", err)
	}
	parsed, err := bibtex.Parse(strings.NewReader(output.String()))
	if err != nil {
		t.Fatalf("runZB() wrote invalid BibTeX: %v", err)
	}
	if got, want := parsed.Entries[0].CiteName, "zbMATH2"; got != want {
		t.Errorf("selected cite name = %q, want %q", got, want)
	}
}

func TestRunZBWarnsAboutProceedingsWithoutBookTitle(t *testing.T) {
	useZBSearch(t, func(string) (zb.Response, error) {
		return zb.Response{Result: []zb.Item{{
			DocumentType: zb.DocumentType{Code: "a"},
			ID:           1,
			Title:        zb.Title{Title: "Incomplete collection article"},
			Year:         "2026",
		}}}, nil
	})

	command := &cobra.Command{}
	var output, stderr bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&stderr)
	if err := runZB(command, []string{"incomplete"}); err != nil {
		t.Fatalf("runZB() error = %v", err)
	}
	parsed, err := bibtex.Parse(strings.NewReader(output.String()))
	if err != nil {
		t.Fatalf("runZB() wrote invalid BibTeX: %v", err)
	}
	if got, want := parsed.Entries[0].Type, "inproceedings"; got != want {
		t.Errorf("entry type = %q, want %q", got, want)
	}
	if _, ok := parsed.Entries[0].Fields["booktitle"]; ok {
		t.Errorf("booktitle = %q, want omitted", parsed.Entries[0].Fields["booktitle"])
	}
	if !strings.Contains(stderr.String(), "missing required BibTeX fields: author, booktitle") {
		t.Errorf("stderr = %q, want missing booktitle warning", stderr.String())
	}
}

func TestZBChoiceLabelUsesEditorsWhenAuthorsAreMissing(t *testing.T) {
	label := zbChoiceLabel(zb.Item{
		Contributors: zb.Contributors{Editors: []zb.Author{
			{Name: "Decker, Wolfram"},
			{Name: "Eder, Christian"},
		}},
		Title: zb.Title{Title: "The computer algebra system OSCAR"},
		Year:  "2025",
	})
	if got, want := label, "Decker, Wolfram et al., 2025, The computer algebra system OSCAR"; got != want {
		t.Errorf("zbChoiceLabel() = %q, want %q", got, want)
	}
}

func TestRunZBRendersArXivPreprint(t *testing.T) {
	useZBSearch(t, func(string) (zb.Response, error) {
		return zb.Response{Result: []zb.Item{arXivPreprintResult()}}, nil
	})

	command := &cobra.Command{}
	var output bytes.Buffer
	command.SetOut(&output)
	if err := runZB(command, []string{"tommy", "hofmann", "2025"}); err != nil {
		t.Fatalf("runZB() error = %v", err)
	}
	parsed, err := bibtex.Parse(strings.NewReader(output.String()))
	if err != nil {
		t.Fatalf("runZB() wrote invalid BibTeX: %v", err)
	}
	if got, want := parsed.Entries[0].Type, "misc"; got != want {
		t.Errorf("entry type = %q, want %q", got, want)
	}
	if got, want := parsed.Entries[0].Fields["eprint"].String(), "2507.15999"; got != want {
		t.Errorf("eprint = %q, want %q", got, want)
	}
}

func TestRunZBCapsChoicesAtTen(t *testing.T) {
	results := make([]zb.Item, zb.MaxSearchResults+2)
	for i := range results {
		results[i] = journalResult(i+1, "Result")
	}
	useZBSearch(t, func(string) (zb.Response, error) {
		return zb.Response{Result: results}, nil
	})
	useZBChoose(t, func(choices []string) (int, error) {
		if got, want := len(choices), zb.MaxSearchResults; got != want {
			t.Errorf("choice count = %d, want %d", got, want)
		}
		return zb.MaxSearchResults - 1, nil
	})

	command := &cobra.Command{}
	var output bytes.Buffer
	command.SetOut(&output)
	if err := runZB(command, []string{"many"}); err != nil {
		t.Fatalf("runZB() error = %v", err)
	}
	parsed, err := bibtex.Parse(strings.NewReader(output.String()))
	if err != nil {
		t.Fatalf("runZB() wrote invalid BibTeX: %v", err)
	}
	if got, want := parsed.Entries[0].CiteName, "zbMATH10"; got != want {
		t.Errorf("selected cite name = %q, want %q", got, want)
	}
}

func TestRunZBHandlesCancelledSelection(t *testing.T) {
	useZBSearch(t, func(string) (zb.Response, error) {
		return zb.Response{Result: []zb.Item{
			journalResult(1, "First result"),
			journalResult(2, "Second result"),
		}}, nil
	})
	useZBChoose(t, func([]string) (int, error) { return -1, nil })

	err := runZB(&cobra.Command{}, []string{"cancel"})
	if err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("runZB() error = %v, want cancellation error", err)
	}
}

func TestRunZBPropagatesChooserErrors(t *testing.T) {
	want := errors.New("terminal unavailable")
	useZBSearch(t, func(string) (zb.Response, error) {
		return zb.Response{Result: []zb.Item{
			journalResult(1, "First result"),
			journalResult(2, "Second result"),
		}}, nil
	})
	useZBChoose(t, func([]string) (int, error) { return -1, want })

	err := runZB(&cobra.Command{}, []string{"chooser"})
	if !errors.Is(err, want) {
		t.Fatalf("runZB() error = %v, want %v", err, want)
	}
}

func TestRunZBRejectsBlankQuery(t *testing.T) {
	called := false
	useZBSearch(t, func(string) (zb.Response, error) {
		called = true
		return zb.Response{}, nil
	})

	err := runZB(&cobra.Command{}, []string{"  "})
	if err == nil || !strings.Contains(err.Error(), "cannot be empty") {
		t.Fatalf("runZB() error = %v, want empty-query error", err)
	}
	if called {
		t.Error("runZB() called the search API for a blank query")
	}
}

func TestRunZBReportsEmptyResults(t *testing.T) {
	useZBSearch(t, func(string) (zb.Response, error) {
		return zb.Response{}, nil
	})

	err := runZB(&cobra.Command{}, []string{"missing"})
	if err == nil || !strings.Contains(err.Error(), "no zbMath entries") {
		t.Fatalf("runZB() error = %v, want empty-result error", err)
	}
}

func TestRunZBPropagatesSearchErrors(t *testing.T) {
	want := errors.New("network unavailable")
	useZBSearch(t, func(string) (zb.Response, error) {
		return zb.Response{}, want
	})

	err := runZB(&cobra.Command{}, []string{"failure"})
	if !errors.Is(err, want) {
		t.Fatalf("runZB() error = %v, want %v", err, want)
	}
}

func TestZBCommandRequiresSearchTerms(t *testing.T) {
	if err := zbCmd.Args(zbCmd, nil); err == nil {
		t.Fatal("zb command accepted an empty argument list")
	}
	if !hexhexCmd.Hidden {
		t.Error("hexhex compatibility command must be hidden")
	}
}

func journalResult(id int, title string) zb.Item {
	return zb.Item{
		Contributors: zb.Contributors{Authors: []zb.Author{{Name: "Example, Author"}}},
		DocumentType: zb.DocumentType{Code: "j"},
		ID:           id,
		Title:        zb.Title{Title: title},
		Year:         "2026",
		Source:       zb.Source{Series: []zb.Series{{Title: "Example Journal", ShortTitle: "Ex. J."}}},
	}
}

func arXivPreprintResult() zb.Item {
	return zb.Item{
		Contributors: zb.Contributors{Authors: []zb.Author{
			{Name: "Tommy Hofmann"},
			{Name: "John Nicholson"},
		}},
		Database:   "arXiv",
		ID:         902789343,
		Identifier: "arXiv:2507.15999",
		Links: []zb.Link{{
			Identifier: "2507.15999",
			Type:       "arxiv",
			URL:        "https://arxiv.org/abs/2507.15999",
		}},
		Title: zb.Title{Title: "Exotic presentations of quaternion groups and Wall's D2 problem"},
		Year:  "2025",
	}
}
