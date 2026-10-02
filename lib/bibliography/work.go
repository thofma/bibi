// Package bibliography defines the records shared by discovery and BibTeX providers.
package bibliography

import (
	"fmt"
	"net/url"
	"regexp"
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
	DOI     string
	IDs     map[string]string
}

type Discoverer interface {
	Search(query string) ([]Work, error)
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
