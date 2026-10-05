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
