package journals

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
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
	// Freeze all four retained fields against the catalog before conversion.
	var records [][4]string
	for _, entry := range entries {
		j := entry.journal
		records = append(records, [4]string{j.Abbreviation, j.Title, j.TranslatedTitle, j.ISSN})
	}
	data, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := fmt.Sprintf("%x", sha256.Sum256(data)), "bcd6826e9f8d0053beb795a19097426fabc670505210fd7ed175f3ffaa76cd7a"; got != want {
		t.Fatalf("snapshot fields checksum = %s, want %s", got, want)
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

func TestParseCatalogFieldsAndTranslatedTitles(t *testing.T) {
	input := `[["Rev. Études","Revue, des Études","Review of Studies","1234-5678"],["Dopov.","Dopovidi","","1025-6415"]]`
	entries, err := parse(bytes.NewReader(compressCatalog(t, input)))
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
	for _, input := range []string{"", `null`, `[]`, `"wrong"`, `[["J.","Journal"]]`, `[["","Journal","","1234-5678"]]`, `[["J.","Journal","","1234-5678"]] trailing`} {
		if _, err := parse(bytes.NewReader(compressCatalog(t, input))); err == nil {
			t.Errorf("parse(%q) succeeded, want invalid catalog", input)
		}
	}
	data := compressCatalog(t, `[["J.","Journal","","1234-5678"]]`)
	corrupt := bytes.Clone(data)
	corrupt[len(corrupt)-8] ^= 1 // Damage the gzip checksum.
	for _, input := range [][]byte{nil, []byte("not gzip"), data[:len(data)-1], corrupt} {
		if _, err := parse(bytes.NewReader(input)); err == nil {
			t.Error("damaged compressed catalog was accepted")
		}
	}
}

func compressCatalog(t *testing.T, input string) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := gzip.NewWriter(&output)
	if _, err := writer.Write([]byte(input)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
