// Package arxiv retrieves exact preprint metadata directly from the arXiv API.
package arxiv

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nickng/bibtex"
	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/lib/bibliography"
)

const defaultBaseURL = "https://export.arxiv.org/api/query"

var defaultClient = &http.Client{Timeout: 15 * time.Second}

// Backend's zero value uses the public API. Overrides support proxies and tests.
type Backend struct {
	BaseURL    string
	HTTPClient *http.Client
}

type feed struct {
	XMLName xml.Name `xml:"http://www.w3.org/2005/Atom feed"`
	Entries []entry  `xml:"http://www.w3.org/2005/Atom entry"`
}

type entry struct {
	ID        string `xml:"http://www.w3.org/2005/Atom id"`
	Title     string `xml:"http://www.w3.org/2005/Atom title"`
	Summary   string `xml:"http://www.w3.org/2005/Atom summary"`
	Published string `xml:"http://www.w3.org/2005/Atom published"`
	Authors   []struct {
		Name string `xml:"http://www.w3.org/2005/Atom name"`
	} `xml:"http://www.w3.org/2005/Atom author"`
	DOI             string `xml:"http://arxiv.org/schemas/atom doi"`
	JournalRef      string `xml:"http://arxiv.org/schemas/atom journal_ref"`
	PrimaryCategory struct {
		Term string `xml:"term,attr"`
	} `xml:"http://arxiv.org/schemas/atom primary_category"`
}

func (backend *Backend) Lookup(ctx context.Context, query string) (bibliography.Record, error) {
	id, ok := bibliography.ArXivQuery(query)
	if !ok {
		return bibliography.Record{}, fmt.Errorf("invalid arXiv identifier %q", query)
	}
	base := backend.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	endpoint, err := url.Parse(base)
	if err != nil {
		return bibliography.Record{}, fmt.Errorf("parse arXiv API URL: %w", err)
	}
	values := endpoint.Query()
	values.Set("id_list", id)
	endpoint.RawQuery = values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return bibliography.Record{}, fmt.Errorf("create arXiv request: %w", err)
	}
	req.Header.Set("Accept", "application/atom+xml")
	req.Header.Set("User-Agent", "bibi (https://github.com/thofma/bibi)")
	client := backend.HTTPClient
	if client == nil {
		client = defaultClient
	}
	diagnostics.Printf("arxiv lookup strategy=identifier id=%q", id)
	response, err := diagnostics.Do(client, req)
	if err != nil {
		return bibliography.Record{}, fmt.Errorf("request arXiv: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return bibliography.Record{}, fmt.Errorf("read arXiv response: %w", err)
	}
	if response.StatusCode == http.StatusNotFound {
		return bibliography.Record{}, fmt.Errorf("arXiv identifier %q not found", id)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		diagnostics.Preview("arXiv", string(body))
		return bibliography.Record{}, fmt.Errorf("arXiv returned %s", response.Status)
	}
	var payload feed
	if err := xml.Unmarshal(body, &payload); err != nil {
		diagnostics.Preview("arXiv", string(body))
		return bibliography.Record{}, fmt.Errorf("parse arXiv response: %w", err)
	}
	if len(payload.Entries) == 0 {
		return bibliography.Record{}, fmt.Errorf("arXiv identifier %q not found", id)
	}
	if len(payload.Entries) != 1 {
		return bibliography.Record{}, fmt.Errorf("arXiv returned %d entries, expected one", len(payload.Entries))
	}
	item := payload.Entries[0]
	if strings.HasPrefix(item.ID, "http://arxiv.org/api/errors") || strings.HasPrefix(item.ID, "https://arxiv.org/api/errors") {
		return bibliography.Record{}, fmt.Errorf("arXiv rejected %q: %s", id, cleanText(item.Summary))
	}
	returned, valid := bibliography.ArXivQuery(item.ID)
	if !valid || bibliography.ArXivBaseID(returned) != bibliography.ArXivBaseID(id) ||
		(bibliography.ArXivBaseID(id) != id && returned != id) {
		return bibliography.Record{}, fmt.Errorf("arXiv returned a different identifier or version: requested %q, received %q", id, item.ID)
	}
	published, err := time.Parse(time.RFC3339, strings.TrimSpace(item.Published))
	if err != nil {
		return bibliography.Record{}, fmt.Errorf("invalid arXiv publication date for %q: %w", id, err)
	}
	work := bibliography.Work{
		Title: cleanText(item.Title), Year: fmt.Sprint(published.Year()), Type: "preprint",
		Venue: "arXiv", Notes: cleanText(item.JournalRef),
		IDs: map[string]string{"arxiv": id},
	}
	if strings.TrimSpace(item.DOI) != "" {
		work.DOI, ok = bibliography.DOIQuery(item.DOI)
		if !ok {
			work.DOI = ""
			diagnostics.Printf("arxiv supplied an unusable publication DOI=%q; retaining preprint metadata only", item.DOI)
		}
	}
	for _, author := range item.Authors {
		if name := cleanText(author.Name); name != "" {
			work.Authors = append(work.Authors, name)
		}
	}
	if work.Title == "" || len(work.Authors) == 0 {
		return bibliography.Record{}, fmt.Errorf("arXiv metadata for %q is missing a title or authors", id)
	}
	preprint := bibtex.NewBibEntry("misc", "arXiv"+strings.ReplaceAll(id, "/", "_"))
	add := func(name, value string) {
		if value != "" {
			preprint.AddField(name, bibtex.NewBibConst(value))
		}
	}
	add("title", bibliography.ProtectTitle(bibliography.EscapeTeXText(work.Title)))
	var authors []string
	for _, name := range work.Authors {
		authors = append(authors, bibliography.EscapeTeXText(name))
	}
	add("author", strings.Join(authors, " and "))
	add("year", work.Year)
	add("eprint", id)
	add("archiveprefix", "arXiv")
	add("primaryclass", strings.TrimSpace(item.PrimaryCategory.Term))
	add("howpublished", bibliography.EscapeTeXText("arXiv preprint arXiv:"+id))
	add("url", "https://arxiv.org/abs/"+id)
	// A publication DOI belongs to the article, not the generated preprint. Keep
	// it in metadata for --published and explicit provider lookups only.
	diagnostics.Printf("arxiv lookup matched id=%q returned_id=%q publication_DOI=%q", id, returned, work.DOI)
	return bibliography.Record{Work: work, Entry: preprint}, nil
}

func cleanText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
