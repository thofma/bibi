package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nickng/bibtex"
	"github.com/spf13/cobra"
	"github.com/thofma/bibi/lib/bibliography"
)

func TestFormatBibTeXPreservesFieldContents(t *testing.T) {
	for _, value := range []string{
		"A simple title",
		"First line\n\tSecond line",
		`Ducas, L\'eo`,
		`10.1007/example\_23`,
		`[2022] \copyright 2022`,
		`Literal \n and \t commands`,
		`A LaTeX \\ line break`,
		`The "quoted" title`,
		`A {nested {title}} with \alpha`,
		"Léo and Göttingen",
		"  leading and trailing spaces  ",
		"",
		"00123",
	} {
		t.Run(value, func(t *testing.T) {
			entry := bibtex.NewBibEntry("article", "test")
			entry.AddField("title", bibtex.NewBibConst(value))
			output := formatBibTeX(entry)
			parsed, err := bibtex.Parse(strings.NewReader(output))
			if err != nil {
				t.Fatalf("parse output: %v\n%s", err, output)
			}
			if len(parsed.Entries) != 1 {
				t.Fatalf("output entry count = %d, want 1", len(parsed.Entries))
			}
			title, ok := parsed.Entries[0].Fields["title"]
			if !ok {
				t.Fatal("output is missing title")
			}
			if got := title.String(); got != value {
				t.Errorf("title = %q, want %q", got, value)
			}
		})
	}
}

func TestWriteBibTeXWarnsWithoutChangingOutputFlow(t *testing.T) {
	entry := bibtex.NewBibEntry("inproceedings", "Incomplete")
	entry.AddField("author", bibtex.NewBibConst("Doe, Jane"))
	entry.AddField("title", bibtex.NewBibConst("A chapter"))
	entry.AddField("year", bibtex.NewBibConst("1999"))
	command := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	if err := writeBibTeX(command, entry, bibliography.JournalNames{}); err != nil {
		t.Fatal(err)
	}
	assertMRBibTeX(t, stdout.String(), "Incomplete")
	if stdout.String() != formatBibTeX(entry) || strings.Contains(stdout.String(), "Warning") {
		t.Errorf("quality warning changed stdout: %q", stdout.String())
	}
	if got, want := stderr.String(), "Warning: Incomplete: missing required BibTeX fields: booktitle\n"; got != want {
		t.Errorf("stderr = %q, want %q", got, want)
	}
}

func TestWriteBibTeXPreservesNativeExports(t *testing.T) {
	const source = `@article{MR1,
  AUTHOR = {van der Waerden, B. L. and G{\"o}del, Kurt and Ducas, L\'eo},
  TITLE = {An {ABC} theorem on {$GL_2(\mathbb{Q})$} and {Galois} theory},
  JOURNAL = {Algebra \& Number Theory},
  FJOURNAL = {Algebra \& Number Theory},
  YEAR = {1999},
  NOTE = {Göttingen and Léo},
  DOI = {10.1000/a\_b},
}`
	parsed, err := bibtex.Parse(strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	entry := parsed.Entries[0]
	command := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	if err := writeBibTeX(command, entry, bibliography.JournalNamesFromEntry(entry)); err != nil {
		t.Fatal(err)
	}
	output, err := bibtex.Parse(strings.NewReader(stdout.String()))
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range entry.Fields {
		if got := output.Entries[0].Fields[key].String(); got != value.String() {
			t.Errorf("native %s = %q, want %q", key, got, value.String())
		}
	}
	if stderr.Len() != 0 {
		t.Errorf("complete native entry produced warnings: %s", stderr.String())
	}
}

func TestFormatBibTeXFieldOrderAndAlignment(t *testing.T) {
	entry := bibtex.NewBibEntry("article", "test")
	for key, value := range map[string]string{
		"year":      "2022",
		"author":    `Ducas, L\'eo`,
		"booktitle": "{EUROCRYPT} 2022",
		"title":     "First line\n              second line",
		"url":       "https://example.com",
	} {
		entry.AddField(key, bibtex.NewBibConst(value))
	}
	const want = `@article{test,
    title     = "First line
              second line",
    author    = "Ducas, L\'eo",
    url       = "https://example.com",
    booktitle = {{EUROCRYPT} 2022},
    year      = 2022,
}
`
	if got := formatBibTeX(entry); got != want {
		t.Errorf("BibTeX output:\n%s\nwant:\n%s", got, want)
	}
}
