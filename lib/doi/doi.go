// Package doi retrieves exact records via DOI resolver content negotiation.
package doi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/internal/httpclient"
	"github.com/thofma/bibi/lib/bibliography"
)

const defaultBaseURL = "https://doi.org"

var defaultClient = &http.Client{Timeout: 15 * time.Second}

// Backend follows DOI resolver redirects to the responsible registration agency.
// Its zero value uses doi.org; overrides support proxies and local tests.
type Backend struct {
	BaseURL    string
	HTTPClient *http.Client
}

func (backend *Backend) Lookup(ctx context.Context, query string) (bibliography.Record, error) {
	doi, ok := bibliography.DOIQuery(query)
	if !ok {
		return bibliography.Record{}, fmt.Errorf("invalid DOI %q", query)
	}
	body, err := backend.get(ctx, doi, "application/x-bibtex")
	if err != nil {
		return bibliography.Record{}, err
	}
	diagnostics.Preview("DOI BibTeX export", string(body))
	parsed, err := bibliography.ParseBibTeX(body)
	if err != nil {
		return bibliography.Record{}, fmt.Errorf("parse DOI BibTeX: %w", err)
	}
	if len(parsed.Entries) != 1 {
		return bibliography.Record{}, fmt.Errorf("DOI export contains %d BibTeX entries, expected one", len(parsed.Entries))
	}
	entry := parsed.Entries[0]
	work := bibliography.WorkFromEntry(entry)
	if work.DOI != "" && bibliography.NormalizeDOI(work.DOI) != doi {
		return bibliography.Record{}, fmt.Errorf("DOI export has a different DOI: requested %q, received %q", doi, work.DOI)
	}
	work.DOI = doi // The exact resolver route establishes identity if the field is absent.
	return bibliography.Record{Work: work, Entry: entry}, nil
}

// Metadata requests CSL metadata, rather than another provider's BibTeX, when
// an explicit --bib choice needs author/title/year for its own lookup.
func (backend *Backend) Metadata(ctx context.Context, query string) (bibliography.Work, error) {
	doi, ok := bibliography.DOIQuery(query)
	if !ok {
		return bibliography.Work{}, fmt.Errorf("invalid DOI %q", query)
	}
	body, err := backend.get(ctx, doi, "application/vnd.citationstyles.csl+json")
	if err != nil {
		return bibliography.Work{}, err
	}
	var item struct {
		DOI       string          `json:"DOI"`
		Title     string          `json:"title"`
		Container string          `json:"container-title"`
		Type      string          `json:"type"`
		Edition   json.RawMessage `json:"edition"`
		Note      string          `json:"note"`
		Author    []name          `json:"author"`
		Editor    []name          `json:"editor"`
		Issued    struct {
			Parts [][]int `json:"date-parts"`
		} `json:"issued"`
	}
	if err := json.Unmarshal(body, &item); err != nil {
		diagnostics.Preview("DOI metadata", string(body))
		return bibliography.Work{}, fmt.Errorf("parse DOI metadata: %w", err)
	}
	if item.DOI != "" && bibliography.NormalizeDOI(item.DOI) != doi {
		return bibliography.Work{}, fmt.Errorf("DOI metadata has a different DOI: requested %q, received %q", doi, item.DOI)
	}
	if strings.TrimSpace(item.Title) == "" {
		return bibliography.Work{}, fmt.Errorf("DOI metadata for %q has no title", doi)
	}
	work := bibliography.Work{DOI: doi, Title: item.Title, Venue: item.Container, Type: item.Type,
		Authors: names(item.Author), Editors: names(item.Editor), Notes: item.Note}
	// CSL providers may encode edition as either text or a number.
	if err := json.Unmarshal(item.Edition, &work.Edition); err != nil {
		var number json.Number
		if json.Unmarshal(item.Edition, &number) == nil {
			work.Edition = number.String()
		}
	}
	if len(item.Issued.Parts) > 0 && len(item.Issued.Parts[0]) > 0 && item.Issued.Parts[0][0] > 0 {
		work.Year = strconv.Itoa(item.Issued.Parts[0][0])
	}
	return work, nil
}

type name struct {
	Family  string `json:"family"`
	Given   string `json:"given"`
	Literal string `json:"literal"`
}

func names(items []name) []string {
	var result []string
	for _, item := range items {
		value := item.Literal
		if value == "" {
			value = item.Family
			if item.Given != "" {
				if value != "" {
					value += ", "
				}
				value += item.Given
			}
		}
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func (backend *Backend) get(ctx context.Context, doi, accept string) ([]byte, error) {
	base := backend.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	endpoint, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("parse DOI resolver URL: %w", err)
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/" + doi
	endpoint.RawPath = ""
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create DOI request: %w", err)
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", "bibi (https://github.com/thofma/bibi)")
	client := backend.HTTPClient
	if client == nil {
		client = defaultClient
	}
	diagnostics.Printf("doi lookup strategy=content-negotiation DOI=%q format=%q", doi, accept)
	response, body, err := httpclient.Do(client, req, "DOI service")
	if err != nil {
		return nil, fmt.Errorf("request DOI resolver: %w", err)
	}
	switch response.StatusCode {
	case http.StatusNotFound:
		return nil, fmt.Errorf("DOI %q not found", doi)
	case http.StatusNotAcceptable:
		return nil, fmt.Errorf("DOI service cannot supply %s for %q", accept, doi)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		diagnostics.Preview("DOI resolver", string(body))
		return nil, fmt.Errorf("DOI service returned %s", response.Status)
	}
	return body, nil
}
