package cmd

import (
	"strings"
	"testing"

	"github.com/nickng/bibtex"
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
