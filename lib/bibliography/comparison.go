package bibliography

import (
	"fmt"
	"sort"
	"strings"
)

// ReviewReasons identifies explicit edition or publication changes that need
// review even when identifiers match. Ordinary metadata differences remain
// informational; a title or an arXiv link alone does not establish work type.
func ReviewReasons(work, candidate Work) []string {
	var reasons []string
	if comparisonText(work.Edition) != "" && comparisonText(candidate.Edition) != "" && comparisonText(work.Edition) != comparisonText(candidate.Edition) {
		reasons = append(reasons, "The supplied editions differ.")
	}
	if publicationType(work.Type) == "preprint" && publicationType(candidate.Type) == "published" {
		reasons = append(reasons, "The selected work is a preprint; the provider candidate is a published work.")
	}
	if publicationType(work.Type) == "published" && publicationType(candidate.Type) == "preprint" {
		reasons = append(reasons, "The selected work is published; the provider candidate is a preprint.")
	}
	return reasons
}

func publicationType(value string) string {
	value = strings.NewReplacer("-", " ", "_", " ").Replace(comparisonText(value))
	switch value {
	case "preprint":
		return "preprint"
	case "article", "journal article", "article journal", "book", "monograph", "inbook", "incollection", "book chapter", "chapter", "inproceedings", "conference", "proceedings article", "paper conference":
		return "published"
	default:
		return ""
	}
}

func comparisonText(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

type comparisonField struct {
	label, selected, candidate string
}

// ComparisonDetails shows a stacked comparison suitable for narrow terminals.
// Source values are displayed without rewriting the exported BibTeX.
func ComparisonDetails(work, candidate Work, assessment MatchAssessment) string {
	fields := []comparisonField{
		{"Title", work.Title, candidate.Title},
		{"Authors", strings.Join(work.Authors, "; "), strings.Join(candidate.Authors, "; ")},
		{"Editors", strings.Join(work.Editors, "; "), strings.Join(candidate.Editors, "; ")},
		{"Year", work.Year, candidate.Year},
		{"Venue", work.Venue, candidate.Venue},
		{"Type", work.Type, candidate.Type},
		{"Edition", work.Edition, candidate.Edition},
		{"Notes", work.Notes, candidate.Notes},
		{"DOI", work.DOI, candidate.DOI},
	}
	keys := make(map[string]bool)
	for key := range work.IDs {
		keys[key] = true
	}
	for key := range candidate.IDs {
		keys[key] = true
	}
	var identifiers []string
	for key := range keys {
		identifiers = append(identifiers, key)
	}
	sort.Strings(identifiers)
	for _, key := range identifiers {
		fields = append(fields, comparisonField{key + " identifier", work.IDs[key], candidate.IDs[key]})
	}
	var differences, selected, supplied []string
	for _, field := range fields {
		selected = append(selected, field.label+": "+available(field.selected))
		supplied = append(supplied, field.label+": "+available(field.candidate))
		same := comparisonText(field.selected) == comparisonText(field.candidate)
		if field.label == "DOI" {
			same = NormalizeDOI(field.selected) == NormalizeDOI(field.candidate)
		}
		if !same {
			differences = append(differences, fmt.Sprintf("%s: selected %q; provider %q", field.label, available(field.selected), available(field.candidate)))
		}
	}
	if len(differences) == 0 {
		differences = append(differences, "No supplied metadata differences.")
	}
	sections := []string{"Match status: " + assessment.Label(), "Reason: " + assessment.Reason}
	if reasons := ReviewReasons(work, candidate); len(reasons) > 0 {
		sections = append(sections, "Review required:\n"+strings.Join(reasons, "\n"))
	}
	sections = append(sections, "Differences:\n"+strings.Join(differences, "\n"),
		"Selected work:\n"+strings.Join(selected, "\n"), "Provider candidate:\n"+strings.Join(supplied, "\n"))
	return strings.Join(sections, "\n\n")
}

func available(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unavailable"
	}
	return value
}
