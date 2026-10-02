// Package bibliography defines the records shared by discovery and BibTeX providers.
package bibliography

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/nickng/bibtex"
)

const MaxResults = 10

// Work contains citation metadata, independent of the service that discovered it.
// IDs holds service identifiers, such as IDs["zb"] or IDs["mr"].
type Work struct {
	Title   string
	Authors []string
	Editors []string
	Year    string
	Venue   string
	Type    string
	Edition string
	Notes   string
	DOI     string
	IDs     map[string]string
}

type Discoverer interface {
	Search(query string) ([]Work, error)
}

// SearchPage uses an opaque continuation token so discovery services can use
// their own pagination mechanism. An empty NextToken means there are no more results.
type SearchPage struct {
	Works     []Work
	NextToken string
	Total     int
}

type PagedDiscoverer interface {
	SearchPage(ctx context.Context, query, token string) (SearchPage, error)
}

// Provider retrieves candidates from the requested BibTeX service only.
type Provider interface {
	BibTeX(work Work) ([]Record, error)
}

type Record struct {
	Work
	Entry    *bibtex.BibEntry
	Journals JournalNames
}

func (work Work) Label() string {
	contributors := work.Authors
	if len(contributors) == 0 {
		contributors = work.Editors
	}
	author := "Unknown author"
	if len(contributors) > 0 {
		author = contributors[0]
		if len(contributors) > 1 {
			author += " et al."
		}
	}
	parts := []string{author}
	for _, part := range []string{work.Year, work.Title} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, ", ")
}

// Details shows the metadata supplied by the source, without inferring editions
// or publication status from titles or identifiers.
func (work Work) Details() string {
	var lines []string
	add := func(label, value string) {
		if strings.TrimSpace(value) != "" {
			lines = append(lines, label+": "+value)
		}
	}
	add("Title", work.Title)
	add("Authors", strings.Join(work.Authors, "; "))
	add("Editors", strings.Join(work.Editors, "; "))
	add("Year", work.Year)
	add("Venue", work.Venue)
	add("Type", work.Type)
	add("Edition", work.Edition)
	add("Notes", work.Notes)
	add("DOI", work.DOI)
	keys := make([]string, 0, len(work.IDs))
	for key := range work.IDs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		add(key, work.IDs[key])
	}
	return strings.Join(lines, "\n\n")
}

// WorkFromEntry exposes existing BibTeX metadata for candidate and legacy pickers.
func WorkFromEntry(entry *bibtex.BibEntry) Work {
	if entry == nil {
		return Work{}
	}
	field := func(name string) string {
		for key, value := range entry.Fields {
			if strings.EqualFold(key, name) {
				return strings.TrimSpace(value.String())
			}
		}
		return ""
	}
	names := func(value string) []string {
		if value == "" {
			return nil
		}
		return strings.Split(value, " and ")
	}
	work := Work{Title: field("title"), Authors: names(field("author")), Editors: names(field("editor")),
		Year: field("year"), DOI: field("doi"), Type: entry.Type,
		Edition: field("edition"), Notes: field("note")}
	for _, name := range []string{"fjournal", "journal", "booktitle", "school", "publisher"} {
		if venue := field(name); venue != "" {
			work.Venue = venue
			break
		}
	}
	return work
}

// Query builds a metadata query for mapping a selected work to another service.
func (work Work) Query() string {
	parts := []string{work.Title}
	if len(work.Authors) > 0 {
		parts = append(parts, work.Authors[0])
	}
	parts = append(parts, work.Year)
	return strings.TrimSpace(strings.Join(parts, " "))
}

// NormalizeDOI accepts bare DOIs, doi: prefixes, DOI resolver URLs, and BibTeX escapes.
func NormalizeDOI(value string) string {
	value = strings.TrimSpace(value)
	if parsed, err := url.Parse(value); err == nil &&
		(strings.EqualFold(parsed.Host, "doi.org") || strings.EqualFold(parsed.Host, "dx.doi.org")) {
		value = strings.TrimPrefix(parsed.Path, "/")
	}
	value = strings.TrimSpace(value)
	if len(value) >= 4 && strings.EqualFold(value[:4], "doi:") {
		value = strings.TrimSpace(value[4:])
	}
	value = strings.NewReplacer(`\_`, "_", `\%`, "%", `\&`, "&").Replace(value)
	return strings.ToLower(value)
}

var doiPattern = regexp.MustCompile(`^10\.\d{4,9}/\S+$`)

// DOIQuery recognizes an entire query as a DOI without interpreting free text.
func DOIQuery(query string) (string, bool) {
	doi := NormalizeDOI(query)
	return doi, doiPattern.MatchString(doi)
}

// Match distinguishes verified identifier matches from compatible candidates.
// Different DOIs veto a match; missing identifiers do not verify identity.
func Match(work, candidate Work, provider string) (exact, compatible bool) {
	exact, compatible, _ = matchDetails(work, candidate, provider)
	return exact, compatible
}

// MatchReason explains the identifier checks without treating metadata similarity
// as proof of identity. The decision and explanation share the same checks.
func MatchReason(work, candidate Work, provider string) string {
	_, _, reason := matchDetails(work, candidate, provider)
	return reason
}

func matchDetails(work, candidate Work, provider string) (exact, compatible bool, reason string) {
	doi, candidateDOI := NormalizeDOI(work.DOI), NormalizeDOI(candidate.DOI)
	if doi != "" && candidateDOI != "" {
		if doi == candidateDOI {
			return true, true, fmt.Sprintf("same normalized DOI %q", doi)
		}
		return false, false, fmt.Sprintf("conflicting DOI: selected=%q candidate=%q", doi, candidateDOI)
	}
	if id := work.IDs[provider]; id != "" && id == candidate.IDs[provider] {
		return true, true, fmt.Sprintf("same %s identifier %q", provider, id)
	}
	if doi == "" && candidateDOI == "" {
		return false, true, fmt.Sprintf("neither record has a DOI and no shared %s identifier; identity is unverified", provider)
	}
	if doi == "" {
		return false, true, "the discovered work has no DOI; the candidate's DOI cannot establish identity"
	}
	return false, true, "the provider candidate has no DOI; identity is unverified"
}
