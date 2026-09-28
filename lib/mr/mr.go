package mr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nickng/bibtex"
)

const (
	defaultMRAPIBaseURL = "https://mathscinet.ams.org/mathscinet/api/freetools/mrlookup"
	mrHTTPTimeout       = 15 * time.Second
)

var (
	mrAPIBaseURL = defaultMRAPIBaseURL
	mrHTTPClient = &http.Client{Timeout: mrHTTPTimeout}
)

type lookupResponse struct {
	All struct {
		Results []lookupResult `json:"results"`
	} `json:"all"`
}

type lookupResult struct {
	BibTeXFormat string `json:"bibTexFormat"`
}

// Entry is a bibliographic record returned by MR Lookup.
type Entry struct {
	Doi     *string
	Authors []string
	Title   string
	Year    string
	BibTeX  *bibtex.BibEntry
}

// MRQueryAYT queries MR Lookup by author, year, and title.
func MRQueryAYT(author, year, title string) ([]*Entry, error) {
	responses, err := mrMultiResponseFromAYT(author, year, title)
	if err != nil {
		return nil, err
	}

	entries := make([]*Entry, 0, len(responses))
	for i, response := range responses {
		parsed, err := bibtex.Parse(bytes.NewReader([]byte(response)))
		if err != nil {
			return nil, fmt.Errorf("parse MR BibTeX result %d: %w", i+1, err)
		}
		if len(parsed.Entries) == 0 {
			return nil, fmt.Errorf("MR BibTeX result %d contains no entry", i+1)
		}

		bib := parsed.Entries[0]
		entry := &Entry{
			Authors: ExtractAuthorsFromBibtex(bib),
			Title:   ExtractTitleFromBibtex(bib),
			Year:    ExtractYearFromBibtex(bib),
			BibTeX:  bib,
		}
		if doi := ExtractDOIFromBibtex(bib); doi != "" {
			entry.Doi = &doi
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func mrMultiResponseFromAYT(author, year, title string) ([]string, error) {
	author, err := fixName(author)
	if err != nil {
		return nil, err
	}

	endpoint, err := url.Parse(mrAPIBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse MR Lookup URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("author", author)
	query.Set("year", year)
	query.Set("journal", "")
	query.Set("firstPage", "")
	query.Set("lastPage", "")
	query.Set("title", title)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create MR Lookup request: %w", err)
	}

	resp, err := mrHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request MR Lookup: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read MR Lookup response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("MR Lookup returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload lookupResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse MR Lookup response: %w", err)
	}

	responses := make([]string, 0, len(payload.All.Results))
	for i, result := range payload.All.Results {
		bibTeX := strings.TrimSpace(result.BibTeXFormat)
		if bibTeX == "" {
			return nil, fmt.Errorf("MR Lookup result %d has no BibTeX", i+1)
		}
		responses = append(responses, bibTeX)
	}
	return responses, nil
}

func fixName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if !strings.Contains(name, ",") {
		return name, nil
	}

	parts := strings.Split(name, ",")
	if len(parts) != 2 {
		return "", fmt.Errorf("malformed name %q: use LAST or LAST,FIRST", name)
	}
	return strings.TrimSpace(parts[0]) + ", " + strings.TrimSpace(parts[1]), nil
}

func ExtractFieldFromBibtex(bib *bibtex.BibEntry, field string) string {
	if bib == nil {
		return ""
	}
	for key, value := range bib.Fields {
		if strings.EqualFold(key, field) {
			return strings.TrimSpace(fmt.Sprint(value))
		}
	}
	return ""
}

func ExtractAuthorsFromBibtex(bib *bibtex.BibEntry) []string {
	authorField := ExtractFieldFromBibtex(bib, "author")
	if authorField == "" {
		return nil
	}

	authors := strings.Split(authorField, " and ")
	result := make([]string, 0, len(authors))
	for _, author := range authors {
		if author = strings.TrimSpace(author); author != "" {
			result = append(result, author)
		}
	}
	return result
}

func ExtractTitleFromBibtex(bib *bibtex.BibEntry) string {
	return ExtractFieldFromBibtex(bib, "title")
}

func ExtractYearFromBibtex(bib *bibtex.BibEntry) string {
	return ExtractFieldFromBibtex(bib, "year")
}

func ExtractDOIFromBibtex(bib *bibtex.BibEntry) string {
	return ExtractFieldFromBibtex(bib, "doi")
}
