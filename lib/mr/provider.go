package mr

import (
	"context"
	"strings"

	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/lib/bibliography"
)

// Provider retrieves MR Lookup's own BibTeX for a work discovered elsewhere.
type Provider struct{}

func (Provider) BibTeX(work bibliography.Work) ([]bibliography.Record, error) {
	return (Provider{}).BibTeXContext(context.Background(), work)
}

// BibTeXContext searches MR Lookup for candidates with cancellation support.
func (Provider) BibTeXContext(ctx context.Context, work bibliography.Work) ([]bibliography.Record, error) {
	author := ""
	if len(work.Authors) > 0 {
		author = lookupFamilyName(work.Authors[0])
		diagnostics.Printf("bib mr author from discovery=%q lookup_author=%q", work.Authors[0], author)
	}
	diagnostics.Printf("bib mr lookup strategy=author-title-year author=%q title=%q year=%q", author, work.Title, work.Year)
	diagnostics.Printf("bib mr selected DOI=%q is used to validate candidates; it is not sent to MR Lookup", work.DOI)
	entries, err := MRQueryAYTContext(ctx, author, work.Year, work.Title)
	if err != nil {
		return nil, err
	}
	diagnostics.Printf("bib mr lookup returned %d parsed records", len(entries))
	if len(entries) == 0 {
		diagnostics.Printf("bib mr lookup found no records for the author/title/year query; this does not prove that MathSciNet has no record for the work")
	}
	records := make([]bibliography.Record, 0, len(entries))
	for i, entry := range entries {
		if entry == nil || entry.BibTeX == nil {
			diagnostics.Printf("bib mr skipped record %d: missing BibTeX entry", i+1)
			continue
		}
		candidate := bibliography.WorkFromEntry(entry.BibTeX)
		candidate.Title, candidate.Authors, candidate.Year = entry.Title, entry.Authors, entry.Year
		candidate.IDs = map[string]string{"mr": entry.BibTeX.CiteName}
		if entry.Doi != nil {
			candidate.DOI = *entry.Doi
		}
		records = append(records, bibliography.Record{Work: candidate, Entry: entry.BibTeX})
	}
	return records, nil
}

// Discovery names written as "Family, Given" identify the complete family name,
// including particles and compound surnames. Keep unstructured names intact
// rather than guessing that their last word is the family name.
func lookupFamilyName(name string) string {
	family, _, _ := strings.Cut(name, ",")
	return strings.TrimSpace(family)
}
