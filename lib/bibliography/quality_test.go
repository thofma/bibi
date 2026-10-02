package bibliography

import (
	"reflect"
	"testing"

	"github.com/nickng/bibtex"
)

func TestMissingFields(t *testing.T) {
	for _, test := range []struct {
		name, entryType string
		fields          map[string]string
		want            []string
	}{
		{"complete article without DOI or issue", "article", map[string]string{"AUTHOR": "Gödel, Kurt", "TITLE": "On {Galois} theory", "JOURNAL": "Journal", "YEAR": "1999"}, nil},
		{"article missing journal", "article", map[string]string{"author": "Doe, Jane", "title": "Work", "year": "1999"}, []string{"journal"}},
		{"edited book", "book", map[string]string{"editor": "Doe, Jane", "title": "Book", "publisher": "Publisher", "year": "1999"}, nil},
		{"book without author or editor", "book", map[string]string{"title": "Book", "publisher": "Publisher", "year": "1999"}, []string{"author or editor"}},
		{"proceedings article without booktitle", "inproceedings", map[string]string{"author": "Doe, Jane", "title": "Chapter", "year": "1999"}, []string{"booktitle"}},
		{"collection article without publisher", "incollection", map[string]string{"author": "Doe, Jane", "title": "Chapter", "booktitle": "Book", "year": "1999"}, []string{"publisher"}},
		{"thesis without school", "phdthesis", map[string]string{"author": "Doe, Jane", "title": "Thesis", "year": "1999"}, []string{"school"}},
		{"blank protected title", "manual", map[string]string{"title": "{  }"}, []string{"title"}},
		{"preprint without journal", "misc", map[string]string{"title": "Preprint", "author": "Doe, Jane", "year": "1999"}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			entry := bibtex.NewBibEntry(test.entryType, "test")
			for key, value := range test.fields {
				entry.AddField(key, bibtex.NewBibConst(value))
			}
			if got := MissingFields(entry); !reflect.DeepEqual(got, test.want) {
				t.Errorf("missing fields = %v, want %v", got, test.want)
			}
		})
	}
}

func TestJournalPreferencePreservesOtherFieldsAndSourceEntry(t *testing.T) {
	entry := bibtex.NewBibEntry("article", "MR1")
	entry.AddField("JOURNAL", bibtex.NewBibConst(`Alg. \& Number Theory`))
	entry.AddField("TITLE", bibtex.NewBibConst(`On {$GL_2$} and {Galois} theory`))
	entry.AddField("AUTHOR", bibtex.NewBibConst(`Brinch Hansen, Per and G{\"o}del, Kurt`))
	names := JournalNames{Full: `Algebra \& Number Theory`, Short: `Alg. \& Number Theory`}
	for _, style := range []string{"source", "short", "full"} {
		copy, warning := WithJournalName(entry, names, style)
		if warning != "" {
			t.Fatal(warning)
		}
		want := names.Short
		if style == "full" {
			want = names.Full
		}
		if got := copy.Fields["JOURNAL"].String(); got != want {
			t.Errorf("%s journal = %q, want %q", style, got, want)
		}
		for _, key := range []string{"TITLE", "AUTHOR"} {
			if copy.Fields[key].String() != entry.Fields[key].String() {
				t.Errorf("journal preference changed %s", key)
			}
		}
	}
	if got := entry.Fields["JOURNAL"].String(); got != names.Short {
		t.Errorf("source entry was mutated: %q", got)
	}
}

func TestUnavailableJournalNameIsNotInvented(t *testing.T) {
	entry := bibtex.NewBibEntry("article", "test")
	entry.AddField("journal", bibtex.NewBibConst("Full Journal Name"))
	copy, warning := WithJournalName(entry, JournalNames{Full: "Full Journal Name"}, "short")
	if warning == "" || copy.Fields["journal"].String() != "Full Journal Name" {
		t.Fatalf("fields = %+v, warning = %q", copy.Fields, warning)
	}
	chapter := bibtex.NewBibEntry("incollection", "chapter")
	copy, warning = WithJournalName(chapter, JournalNames{Full: "Collected Papers"}, "full")
	if warning != "" || len(copy.Fields) != 0 {
		t.Fatalf("journal preference added a journal to a chapter: %+v, warning = %q", copy.Fields, warning)
	}
}
