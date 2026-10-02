// Package crossref implements optional discovery and BibTeX retrieval via Crossref.
package crossref

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nickng/bibtex"
	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/lib/bibliography"
)

const defaultBaseURL = "https://api.crossref.org"

var defaultClient = &http.Client{Timeout: 15 * time.Second}

// Backend's zero value uses the public Crossref API. BaseURL and HTTPClient can
// be supplied for a proxy or a local test server.
type Backend struct {
	BaseURL    string
	HTTPClient *http.Client
	journals   map[string]bibliography.JournalNames
}

type contributor struct {
	Family string `json:"family"`
	Given  string `json:"given"`
	Name   string `json:"name"`
}

type date struct {
	Parts [][]int `json:"date-parts"`
}

type item struct {
	DOI             string        `json:"DOI"`
	Title           []string      `json:"title"`
	Subtitle        []string      `json:"subtitle"`
	ContainerTitle  []string      `json:"container-title"`
	ShortContainer  []string      `json:"short-container-title"`
	Authors         []contributor `json:"author"`
	Editors         []contributor `json:"editor"`
	Published       date          `json:"published"`
	PublishedPrint  date          `json:"published-print"`
	PublishedOnline date          `json:"published-online"`
	Issued          date          `json:"issued"`
	Type            string        `json:"type"`
	Subtype         string        `json:"subtype"`
	Edition         string        `json:"edition-number"`
}

func (backend *Backend) Search(query string) ([]bibliography.Work, error) {
	page, err := backend.SearchPage(context.Background(), query, "")
	return page.Works, err
}

func (backend *Backend) SearchPage(ctx context.Context, query, token string) (bibliography.SearchPage, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return bibliography.SearchPage{}, fmt.Errorf("Crossref search query cannot be empty")
	}
	if doi, ok := bibliography.DOIQuery(query); ok {
		body, found, err := backend.getContext(ctx, "/works/"+doi, nil, "application/json")
		if err != nil || !found {
			return bibliography.SearchPage{}, err
		}
		var payload struct {
			Message item `json:"message"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return bibliography.SearchPage{}, fmt.Errorf("parse Crossref DOI response: %w", err)
		}
		work := payload.Message.work()
		if bibliography.NormalizeDOI(work.DOI) != doi {
			return bibliography.SearchPage{}, fmt.Errorf("Crossref returned a different DOI for %q", doi)
		}
		backend.rememberJournals(payload.Message)
		return bibliography.SearchPage{Works: []bibliography.Work{work}, Total: 1}, nil
	}

	values := url.Values{}
	values.Set("query.bibliographic", query)
	values.Set("rows", strconv.Itoa(bibliography.MaxResults))
	if token == "" {
		token = "*"
	}
	values.Set("cursor", token)
	body, _, err := backend.getContext(ctx, "/works", values, "application/json")
	if err != nil {
		return bibliography.SearchPage{}, err
	}
	var payload struct {
		Message struct {
			Items     []item `json:"items"`
			NextToken string `json:"next-cursor"`
			Total     int    `json:"total-results"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return bibliography.SearchPage{}, fmt.Errorf("parse Crossref search response: %w", err)
	}
	works := make([]bibliography.Work, 0, len(payload.Message.Items))
	for _, item := range payload.Message.Items {
		backend.rememberJournals(item)
		works = append(works, item.work())
	}
	page := bibliography.SearchPage{Works: works, Total: payload.Message.Total}
	if len(works) == bibliography.MaxResults && (page.Total == 0 || page.Total > len(works)) {
		page.NextToken = payload.Message.NextToken
	}
	return page, nil
}

// BibTeX requires a DOI and retrieves only Crossref's export, without fallback.
func (backend *Backend) BibTeX(work bibliography.Work) ([]bibliography.Record, error) {
	doi, ok := bibliography.DOIQuery(work.DOI)
	diagnostics.Printf("bib crossref lookup strategy=DOI DOI=%q normalized_DOI=%q", work.DOI, doi)
	if !ok {
		diagnostics.Printf("bib crossref stage=lookup failed: selected work has no valid DOI; export was not requested")
		return nil, fmt.Errorf("Crossref BibTeX requires a DOI for the selected work")
	}
	body, found, err := backend.get("/works/"+doi+"/transform", nil, "application/x-bibtex")
	if err != nil {
		return nil, err
	}
	if !found {
		diagnostics.Printf("bib crossref stage=lookup failed: HTTP 404 for DOI %q; Crossref supplied no export", doi)
		return nil, nil
	}
	diagnostics.Printf("bib crossref stage=parse started: export_bytes=%d", len(body))
	diagnostics.Preview("Crossref BibTeX export", string(body))
	parsed, err := parseBibTeX(body)
	if err != nil {
		diagnostics.Printf("bib crossref stage=parse failed: %v", err)
		return nil, fmt.Errorf("parse Crossref BibTeX: %w", err)
	}
	if len(parsed.Entries) != 1 {
		diagnostics.Printf("bib crossref stage=parse failed: expected one entry, got %d", len(parsed.Entries))
		return nil, fmt.Errorf("Crossref export contains %d BibTeX entries, expected one", len(parsed.Entries))
	}
	entry := parsed.Entries[0]
	exportedDOI := field(entry, "doi")
	if exportedDOI != "" && bibliography.NormalizeDOI(exportedDOI) != doi {
		diagnostics.Printf("bib crossref stage=matching failed: selected DOI=%q normalized=%q export DOI=%q normalized=%q", work.DOI, doi, exportedDOI, bibliography.NormalizeDOI(exportedDOI))
		return nil, fmt.Errorf("Crossref export has a different DOI from the selected work")
	}
	// The exact DOI route establishes identity even if the export omits its DOI field.
	candidate := bibliography.WorkFromEntry(entry)
	candidate.DOI = doi
	names := backend.journals[doi]
	if full := field(entry, "journal"); full != "" {
		names.Full = full
	}
	return []bibliography.Record{{Work: candidate, Entry: entry, Journals: names}}, nil
}

func (backend *Backend) rememberJournals(item item) {
	if backend.journals == nil {
		backend.journals = make(map[string]bibliography.JournalNames)
	}
	var names bibliography.JournalNames
	if len(item.ContainerTitle) > 0 {
		names.Full = bibliography.EscapeTeXText(item.ContainerTitle[0])
	}
	if len(item.ShortContainer) > 0 {
		names.Short = bibliography.EscapeTeXText(item.ShortContainer[0])
	}
	backend.journals[bibliography.NormalizeDOI(item.DOI)] = names
}

func parseBibTeX(body []byte) (*bibtex.BibTex, error) {
	// Crossref emits capitalized month macros (e.g. Dec). The parser's built-in
	// month names are case sensitive, so define aliases without changing fields.
	var source strings.Builder
	for _, month := range []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"} {
		capitalized := strings.ToUpper(month[:1]) + month[1:]
		fmt.Fprintf(&source, "@string{%s = %s}\n@string{%s = %s}\n", capitalized, month, strings.ToUpper(month), month)
	}
	source.Write(body)
	return bibtex.Parse(strings.NewReader(source.String()))
}

func (backend *Backend) get(path string, values url.Values, accept string) ([]byte, bool, error) {
	return backend.getContext(context.Background(), path, values, accept)
}

func (backend *Backend) getContext(ctx context.Context, path string, values url.Values, accept string) ([]byte, bool, error) {
	base := backend.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	endpoint, err := url.Parse(base)
	if err != nil {
		return nil, false, fmt.Errorf("parse Crossref API URL: %w", err)
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + path
	endpoint.RawPath = ""
	endpoint.RawQuery = values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, false, fmt.Errorf("create Crossref request: %w", err)
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", "bibi (https://github.com/thofma/bibi)")
	client := backend.HTTPClient
	if client == nil {
		client = defaultClient
	}
	response, err := diagnostics.Do(client, req)
	if err != nil {
		return nil, false, fmt.Errorf("request Crossref: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, false, fmt.Errorf("read Crossref response: %w", err)
	}
	if response.StatusCode == http.StatusNotFound && path != "/works" {
		return nil, false, nil
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, false, fmt.Errorf("Crossref returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	return body, true, nil
}

func (item item) work() bibliography.Work {
	work := bibliography.Work{DOI: item.DOI, Type: item.Type, Edition: item.Edition}
	if item.Subtype != "" {
		work.Notes = "Subtype: " + item.Subtype
	}
	if len(item.ContainerTitle) > 0 {
		work.Venue = strings.Join(item.ContainerTitle, "; ")
	}
	if len(item.Title) > 0 {
		work.Title = item.Title[0]
	}
	if len(item.Subtitle) > 0 && item.Subtitle[0] != "" {
		work.Title += ": " + item.Subtitle[0]
	}
	work.Authors = names(item.Authors)
	work.Editors = names(item.Editors)
	for _, date := range []date{item.Published, item.PublishedPrint, item.PublishedOnline, item.Issued} {
		if len(date.Parts) > 0 && len(date.Parts[0]) > 0 && date.Parts[0][0] > 0 {
			work.Year = strconv.Itoa(date.Parts[0][0])
			break
		}
	}
	return work
}

func names(contributors []contributor) []string {
	var result []string
	for _, contributor := range contributors {
		name := contributor.Name
		if name == "" {
			name = contributor.Family
			if contributor.Given != "" {
				if name != "" {
					name += ", "
				}
				name += contributor.Given
			}
		}
		if name = strings.TrimSpace(name); name != "" {
			result = append(result, name)
		}
	}
	return result
}

func field(entry *bibtex.BibEntry, name string) string {
	for key, value := range entry.Fields {
		if strings.EqualFold(key, name) {
			return strings.TrimSpace(value.String())
		}
	}
	return ""
}
