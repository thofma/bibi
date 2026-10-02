package zb

import (
	"testing"

	"github.com/thofma/bibi/lib/bibliography"
)

func TestGeneratedArticleRetainsIssueAndProtectsText(t *testing.T) {
	item := Item{
		ID: 1, DocumentType: DocumentType{Code: "j"}, Year: "1999",
		Title:        Title{Title: `Galois groups of \(GL_2(K)\) & 100% results`},
		Contributors: Contributors{Authors: []Author{{Name: "Brinch Hansen, Per"}, {Name: `G{\"o}del, Kurt`}, {Name: "Ducas, Léo"}}},
		Source:       Source{Pages: "1-9, 11–15", Series: []Series{{Title: "Algebra & Number Theory", ShortTitle: "Alg. Number Theory", Issue: "3-4", Volume: "42"}}},
	}
	entry, err := ItemToBibEntry(item)
	if err != nil {
		t.Fatal(err)
	}
	for field, want := range map[string]string{
		"number": "3-4", "volume": "42", "pages": "1--9, 11--15",
		"title":    `{Galois groups of \(GL_2(K)\) \& 100\% results}`,
		"author":   `Brinch Hansen, Per and G{\"o}del, Kurt and Ducas, Léo`,
		"journal":  "Alg. Number Theory",
		"fjournal": `Algebra \& Number Theory`,
	} {
		if value, ok := entry.Fields[field]; !ok || value.String() != want {
			t.Errorf("%s = %v, want %q", field, value, want)
		}
	}
	AssertValidBibTeX(t, entry)
	if missing := bibliography.MissingFields(entry); len(missing) != 0 {
		t.Errorf("complete article has missing fields: %v", missing)
	}
}

func TestJournalNamesDoNotInventMissingAbbreviations(t *testing.T) {
	item := Item{DocumentType: DocumentType{Code: "j"}, Source: Source{Series: []Series{{Title: "Full Journal Name"}}}}
	if names := ItemJournalNames(item); names.Full != "Full Journal Name" || names.Short != "" {
		t.Errorf("names = %+v", names)
	}
}
