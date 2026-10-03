package journals

import (
	"strings"
	"testing"
)

func TestBundledSnapshot(t *testing.T) {
	entries, err := loadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(entries), 2385; got != want {
		t.Fatalf("snapshot has %d records, want %d", got, want)
	}
	for _, test := range []struct {
		query, abbreviation, issn string
	}{
		{"inventiones mathematicae", "Invent. Math.", "0020-9910"},
		// The publisher in this source row contains malformed CSV quoting.
		{"dopovidi natsionalnoi", "Dopov. Nats. Akad. Nauk Ukr. Mat. Prirodozn. Tekh. Nauki", "1025-6415"},
		// Abbreviation-only source records must remain searchable.
		{"acta sci natur univ pekinensis", "Acta Sci. Natur. Univ. Pekinensis", ""},
	} {
		t.Run(test.query, func(t *testing.T) {
			matches, err := Search(test.query)
			if err != nil {
				t.Fatal(err)
			}
			for _, journal := range matches {
				if journal.Abbreviation == test.abbreviation && journal.ISSN == test.issn {
					return
				}
			}
			t.Fatalf("matches = %+v, want abbreviation %q and ISSN %q", matches, test.abbreviation, test.issn)
		})
	}
}

func TestSearchWordPrefixes(t *testing.T) {
	for _, test := range []struct {
		query, want string
	}{
		{"MATHÉMATÍCAE, INVENTIONES", "Invent. Math."},
		{"math invent", "Invent. Math."},
		{"InVeNt. MaTh.", "Invent. Math."},
		{"für REINE, angewandte", "J. Reine Angew. Math."},
		{"fu\u0308r reine angewandte", "J. Reine Angew. Math."},
	} {
		t.Run(test.query, func(t *testing.T) {
			matches, err := Search(test.query)
			if err != nil {
				t.Fatal(err)
			}
			if len(matches) != 1 || matches[0].Abbreviation != test.want {
				t.Fatalf("matches = %+v, want only %q", matches, test.want)
			}
		})
	}
	for _, query := range []string{"ventiones mathematicae", "inventiones unmatchedword", "journalwhichdoesnotexist"} {
		matches, err := Search(query)
		if err != nil || len(matches) != 0 {
			t.Errorf("Search(%q) = %+v, %v; want no matches", query, matches, err)
		}
	}
	for _, query := range []string{"", " \t ", "... --"} {
		if _, err := Search(query); err == nil {
			t.Errorf("Search(%q) succeeded, want invalid query", query)
		}
	}
}

func TestSearchIncludesEveryPrefixInAnyOrder(t *testing.T) {
	forward, err := Search("jour numbe theor")
	if err != nil {
		t.Fatal(err)
	}
	reverse, err := Search("theor numbe jour")
	if err != nil {
		t.Fatal(err)
	}
	if len(forward) <= 1 || len(forward) != len(reverse) {
		t.Fatalf("expected multiple stable matches; forward=%+v reverse=%+v", forward, reverse)
	}
	found := false
	for i, journal := range forward {
		if journal != reverse[i] {
			t.Errorf("word order changed result %d", i)
		}
		if journal.Abbreviation == "J. Number Theory" {
			found = true
		}
	}
	if !found {
		t.Fatal("Journal of Number Theory is missing")
	}
}

func TestParseQuotedFieldsAndTranslatedTitles(t *testing.T) {
	input := "\ufeffAbbrev,Full Title,Trnsl. Title,Publ.,ISSN,New,Cover-to-cover,Book Ser\n" +
		"\"Rev. Études\",\"Revue, des Études\",\"Review of Studies\",\"Publisher, City\",1234-5678,N,Y,N\n" +
		"\"Dopov.\",\"Dopovidi\",\"\",\"Vidavn. Dim \"Akademperiodika\", Kiev.\",\"1025-6415\",\"N\",\"N\",\"N\"\n"
	entries, err := parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].journal.Title != "Revue, des Études" || entries[1].journal.ISSN != "1025-6415" {
		t.Fatalf("parsed entries = %+v", entries)
	}
	for _, query := range []string{"etudes rev", "studies review"} {
		matches := entries.search(searchWords(query))
		if len(matches) != 1 || matches[0].Abbreviation != "Rev. Études" {
			t.Errorf("search %q = %+v", query, matches)
		}
	}
}

func TestParseRejectsBrokenCatalog(t *testing.T) {
	header := "Abbrev,Full Title,Trnsl. Title,Publ.,ISSN,New,Cover-to-cover,Book Ser\n"
	for _, input := range []string{"", "wrong,columns\n", header, header + "J.,Journal\n", header + ",Journal,,,1234-5678,N,N,N\n"} {
		if _, err := parse(strings.NewReader(input)); err == nil {
			t.Errorf("parse(%q) succeeded, want invalid catalog", input)
		}
	}
}
