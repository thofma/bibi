// Package journals searches the bundled journal abbreviation catalog.
package journals

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

//go:embed catalog.json.gz
var serialsData []byte

// Journal retains the catalog spelling for display and abbreviation output.
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
	return parse(bytes.NewReader(serialsData))
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
		return nil, fmt.Errorf("read bundled journal catalog: %w", err)
	}
	return entries.search(words), nil
}

func parse(input io.Reader) (catalog, error) {
	compressed, err := gzip.NewReader(input)
	if err != nil {
		return nil, fmt.Errorf("open compressed catalog: %w", err)
	}
	defer compressed.Close()
	data, err := io.ReadAll(compressed)
	if err != nil {
		return nil, fmt.Errorf("read compressed catalog: %w", err)
	}
	var records [][]string
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("read catalog JSON: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("catalog has no journal entries")
	}
	entries := make(catalog, 0, len(records))
	for i, record := range records {
		if len(record) != 4 {
			return nil, fmt.Errorf("catalog record %d has %d fields, want 4", i+1, len(record))
		}
		journal := Journal{
			Abbreviation:    strings.TrimSpace(record[0]),
			Title:           strings.TrimSpace(record[1]),
			TranslatedTitle: strings.TrimSpace(record[2]),
			ISSN:            strings.TrimSpace(record[3]),
		}
		if journal.Abbreviation == "" {
			return nil, fmt.Errorf("catalog record %d has no abbreviation", i+1)
		}
		entries = append(entries, entry{journal: journal,
			words: searchWords(journal.Title + " " + journal.TranslatedTitle + " " + journal.Abbreviation)})
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
