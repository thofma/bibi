// Package journals searches the bundled AMS MR Serials Abbreviations List.
package journals

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

//go:embed annser.csv
var serialsCSV string

// Journal retains the AMS spelling for display and abbreviation output.
type Journal struct {
	Abbreviation    string
	Title           string
	TranslatedTitle string
	ISSN            string
}

type entry struct {
	journal Journal
	words   []string
}

type catalog []entry

var loadCatalog = sync.OnceValues(func() (catalog, error) {
	return parse(strings.NewReader(serialsCSV))
})

// Search matches every query word against a word prefix in a journal's title,
// translated title or abbreviation. It never accesses the network or disk.
func Search(query string) ([]Journal, error) {
	words := searchWords(query)
	if len(words) == 0 {
		return nil, fmt.Errorf("journal query must contain at least one letter or number")
	}
	entries, err := loadCatalog()
	if err != nil {
		return nil, fmt.Errorf("read bundled AMS journal list: %w", err)
	}
	return entries.search(words), nil
}

func parse(input io.Reader) (catalog, error) {
	reader := csv.NewReader(input)
	// The AMS source contains an unescaped quote and comma in a publisher field.
	// Keep the original snapshot and tolerate that field while reading the fixed
	// title columns and the ISSN/final flags from the end of each record.
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}
	wantHeader := []string{"Abbrev", "Full Title", "Trnsl. Title", "Publ.", "ISSN", "New", "Cover-to-cover", "Book Ser"}
	if len(header) != len(wantHeader) {
		return nil, fmt.Errorf("unexpected AMS CSV header")
	}
	for i, name := range wantHeader {
		if strings.TrimSpace(strings.TrimPrefix(header[i], "\ufeff")) != name {
			return nil, fmt.Errorf("unexpected AMS CSV column %d: %q", i+1, header[i])
		}
	}
	var entries catalog
	for row := 2; ; row++ {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read CSV row %d: %w", row, err)
		}
		if len(record) < len(wantHeader) {
			return nil, fmt.Errorf("AMS CSV row %d has %d columns, want at least %d", row, len(record), len(wantHeader))
		}
		journal := Journal{
			Abbreviation:    strings.TrimSpace(record[0]),
			Title:           strings.TrimSpace(record[1]),
			TranslatedTitle: strings.TrimSpace(record[2]),
			ISSN:            strings.TrimSpace(record[len(record)-4]),
		}
		if journal.Abbreviation == "" {
			return nil, fmt.Errorf("AMS CSV row %d has no abbreviation", row)
		}
		entries = append(entries, entry{journal: journal,
			words: searchWords(journal.Title + " " + journal.TranslatedTitle + " " + journal.Abbreviation)})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("AMS CSV has no journal entries")
	}
	return entries, nil
}

func (entries catalog) search(words []string) []Journal {
	var matches []Journal
	for _, entry := range entries {
		matchesAll := true
		for _, queryWord := range words {
			found := false
			for _, word := range entry.words {
				if strings.HasPrefix(word, queryWord) {
					found = true
					break
				}
			}
			if !found {
				matchesAll = false
				break
			}
		}
		if matchesAll {
			matches = append(matches, entry.journal)
		}
	}
	return matches
}

func searchWords(text string) []string {
	text = strings.Map(func(r rune) rune {
		switch {
		case unicode.Is(unicode.Mn, r):
			return -1
		case unicode.IsLetter(r), unicode.IsNumber(r):
			return unicode.ToLower(r)
		default:
			return ' '
		}
	}, norm.NFD.String(text))
	return strings.Fields(text)
}
