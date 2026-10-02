package zb

import (
	"strconv"

	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/lib/bibliography"
)

// Backend adapts zbMATH discovery and BibTeX generation to the shared interfaces.
// It retains native search records so same-service output loses no metadata.
type Backend struct {
	items []Item
}

func (backend *Backend) Search(query string) ([]bibliography.Work, error) {
	var response Response
	var err error
	doi, isDOI := bibliography.DOIQuery(query)
	if isDOI {
		response, err = SearchDOI(doi)
	} else {
		response, err = Search(query)
	}
	if err != nil {
		return nil, err
	}
	backend.items = response.Result
	works := make([]bibliography.Work, 0, len(response.Result))
	for _, item := range response.Result {
		work := itemWork(item)
		if isDOI && bibliography.NormalizeDOI(work.DOI) != doi {
			continue
		}
		works = append(works, work)
	}
	return works, nil
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
				return []bibliography.Record{{Work: itemWork(item), Entry: entry, Journals: ItemJournalNames(item)}}, nil
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
		candidate := itemWork(item)
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

func itemWork(item Item) bibliography.Work {
	doi, _ := ItemGetDOI(item)
	work := bibliography.Work{
		Title: ItemGetTitle(item), Year: item.Year, DOI: doi,
		IDs: make(map[string]string),
	}
	if item.ID != 0 {
		work.IDs["zb"] = strconv.Itoa(item.ID)
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
