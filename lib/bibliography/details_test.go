package bibliography

import (
	"strings"
	"testing"

	"github.com/nickng/bibtex"
)

func TestDetailsKeepAllContributorsAndSourceMetadata(t *testing.T) {
	work := Work{Title: "A long title", Authors: []string{"First, Alice", "Second, Bob", "Third, Carol"},
		Editors: []string{"Editor, Eve"}, Year: "2002", Venue: "Full Journal Name", Type: "book",
		Edition: "2", Notes: "Translation of the original edition", DOI: "10.1000/example",
		IDs: map[string]string{"zb": "42", "arxiv": "1234.5678"}}
	details := work.Details()
	for _, value := range []string{"Third, Carol", "Editor, Eve", "Full Journal Name", "Edition: 2", work.Notes, work.DOI} {
		if !strings.Contains(details, value) {
			t.Errorf("missing %q in %s", value, details)
		}
	}
	if strings.Index(details, "arxiv:") > strings.Index(details, "zb:") {
		t.Fatal("identifiers are not ordered deterministically")
	}
	if details := (Work{Title: "A second edition preprint"}).Details(); strings.Contains(details, "Edition:") || strings.Contains(details, "Type:") {
		t.Fatal("inferred type or edition from title")
	}
}

func TestWorkFromEntryReadsNativeMetadataWithoutMutation(t *testing.T) {
	entry := bibtex.NewBibEntry("book", "MR123")
	for key, value := range map[string]string{"TITLE": "A {TeX} title", "EDITOR": "One, Alice and Two, Bob",
		"PUBLISHER": "Publisher", "EDITION": "Second", "NOTE": "Reprint of the 1967 original", "YEAR": "2006"} {
		entry.AddField(key, bibtex.NewBibConst(value))
	}
	work := WorkFromEntry(entry)
	if work.Title != "A {TeX} title" || len(work.Editors) != 2 || len(work.Authors) != 0 || work.Edition != "Second" || work.Venue != "Publisher" || work.Type != "book" || work.Notes != "Reprint of the 1967 original" {
		t.Fatalf("metadata = %+v", work)
	}
	if entry.Fields["TITLE"].String() != "A {TeX} title" || len(entry.Fields) != 6 {
		t.Fatal("detail extraction changed native BibTeX")
	}
}
