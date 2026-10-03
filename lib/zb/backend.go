package zb

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/lib/bibliography"
)

// Backend adapts zbMATH discovery and BibTeX generation to the shared interfaces.
// It retains native search records so same-service output loses no metadata.
type Backend struct {
	items []Item
}

func (backend *Backend) Search(query string) ([]bibliography.Work, error) {
	page, err := backend.SearchPage(context.Background(), query, "")
	return page.Works, err
}

func (backend *Backend) SearchPage(ctx context.Context, query, token string) (bibliography.SearchPage, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return bibliography.SearchPage{}, fmt.Errorf("zbMath search query cannot be empty")
	}
	pageNumber := 0
	if token != "" {
		var err error
		pageNumber, err = strconv.Atoi(token)
		if err != nil || pageNumber < 0 {
			return bibliography.SearchPage{}, fmt.Errorf("invalid zbMATH page token %q", token)
		}
	}
	values := url.Values{"results_per_page": {strconv.Itoa(MaxSearchResults)}, "page": {strconv.Itoa(pageNumber)}}
	path := "_search"
	doi, isDOI := bibliography.DOIQuery(query)
	if isDOI {
		path = "_structured_search"
		values.Set("DOI", doi)
		values.Set("results_per_page", "1")
	} else {
		values.Set("search_string", query)
	}
	body, err := getZBAPIContext(ctx, path, values)
	var response Response
	if err != nil {
		var noResults bool
		response, noResults = zbNoResultsResponse(err)
		if !noResults {
			return bibliography.SearchPage{}, err
		}
	} else {
		response, err = ParseToStruct(body)
		if err != nil {
			diagnostics.Preview("zbMATH search", body)
			return bibliography.SearchPage{}, fmt.Errorf("parse zbMath search response: %w", err)
		}
	}
	if pageNumber == 0 {
		backend.items = nil
	}
	backend.items = append(backend.items, response.Result...)
	works := make([]bibliography.Work, 0, len(response.Result))
	for _, item := range response.Result {
		work := ItemWork(item)
		if isDOI && bibliography.NormalizeDOI(work.DOI) != doi {
			continue
		}
		works = append(works, work)
	}
	page := bibliography.SearchPage{Works: works, Total: response.Status.NrTotalResults}
	if !isDOI && len(works) > 0 && (pageNumber*MaxSearchResults+len(works) < page.Total ||
		(page.Total == 0 && len(works) == MaxSearchResults)) {
		page.NextToken = strconv.Itoa(pageNumber + 1)
	}
	return page, nil
}

func (backend *Backend) BibTeX(work bibliography.Work) ([]bibliography.Record, error) {
	if id := work.IDs["zb"]; id != "" {
		for _, item := range backend.items {
			if strconv.Itoa(item.ID) == id {
				diagnostics.Printf("using cached zbMATH record %s for BibTeX", id)
				entry, err := ItemToBibEntry(item, backend.items...)
				if err != nil {
					diagnostics.Printf("bib zb stage=conversion failed: ID=%d type=%q title=%q error=%v", item.ID, item.DocumentType.Code, ItemGetTitle(item), err)
					return nil, err
				}
				return []bibliography.Record{{Work: ItemWork(item), Entry: entry, Journals: ItemJournalNames(item)}}, nil
			}
		}
	}

	var response Response
	var err error
	if work.DOI != "" {
		diagnostics.Printf("bib zb lookup strategy=DOI DOI=%q normalized_DOI=%q", work.DOI, bibliography.NormalizeDOI(work.DOI))
		response, err = SearchDOI(bibliography.NormalizeDOI(work.DOI))
	} else {
		diagnostics.Printf("bib zb lookup strategy=metadata query=%q (selected work has no DOI)", work.Query())
		response, err = Search(work.Query())
	}
	if err != nil {
		return nil, err
	}
	diagnostics.Printf("bib zb lookup returned %d native records; API status=%q", len(response.Result), response.Status.InternalCode)
	if len(response.Result) == 0 {
		diagnostics.Printf("bib zb stage=lookup failed: service returned no native records")
	}
	records := make([]bibliography.Record, 0, len(response.Result))
	for _, item := range response.Result {
		candidate := ItemWork(item)
		if _, compatible := bibliography.Match(work, candidate, "zb"); !compatible {
			diagnostics.Printf("bib zb rejected native record ID=%d title=%q authors=%q year=%q DOI=%q: %s", item.ID, candidate.Title, candidate.Authors, candidate.Year, candidate.DOI, bibliography.MatchReason(work, candidate, "zb"))
			continue
		}
		entry, err := ItemToBibEntry(item, response.Result...)
		if err != nil {
			diagnostics.Printf("bib zb stage=conversion failed: ID=%d type=%q title=%q error=%v", item.ID, item.DocumentType.Code, ItemGetTitle(item), err)
			return nil, err
		}
		records = append(records, bibliography.Record{Work: candidate, Entry: entry, Journals: ItemJournalNames(item)})
	}
	if len(records) == 0 && len(response.Result) > 0 {
		diagnostics.Printf("bib zb stage=matching failed: all %d native records had conflicting DOIs", len(response.Result))
	}
	return records, nil
}

// ItemWork exposes the source's metadata without converting it to BibTeX.
func ItemWork(item Item) bibliography.Work {
	doi, _ := ItemGetDOI(item)
	work := bibliography.Work{
		Title: ItemGetTitle(item), Year: item.Year, DOI: doi,
		Type: item.DocumentType.Description, Notes: item.Title.Addition,
		IDs: make(map[string]string),
	}
	if work.Type == "" {
		work.Type = map[string]string{"j": "journal article", "b": "book", "a": "proceedings article", "p": "preprint"}[item.DocumentType.Code]
	}
	for _, note := range []struct{ label, value string }{
		{"Subtitle", item.Title.Subtitle}, {"Original title", item.Title.Original},
	} {
		if note.value != "" {
			if work.Notes != "" {
				work.Notes += "; "
			}
			work.Notes += note.label + ": " + note.value
		}
	}
	if item.ID != 0 {
		work.IDs["zb"] = strconv.Itoa(item.ID)
	}
	if item.Identifier != "" && strings.EqualFold(item.Database, "Zbl") {
		work.IDs["zbl"] = item.Identifier
	}
	if arxiv, _ := ItemGetArXiv(item); arxiv != "" {
		work.IDs["arxiv"] = arxiv
	}
	work.Venue = item.Source.Source
	if work.Venue == "" {
		for _, series := range item.Source.Series {
			if series.Title != "" {
				work.Venue = series.Title
				break
			}
		}
	}
	if work.Venue == "" && len(item.Source.Book) > 0 {
		work.Venue = item.Source.Book[0].Title
	}
	for _, author := range item.Contributors.Authors {
		if author.Name != "" {
			work.Authors = append(work.Authors, author.Name)
		}
	}
	for _, editor := range item.Contributors.Editors {
		if editor.Name != "" {
			work.Editors = append(work.Editors, editor.Name)
		}
	}
	return work
}
