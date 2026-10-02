package bibliography

import (
	"strings"

	"github.com/nickng/bibtex"
)

// JournalNames contains only names actually supplied by the BibTeX provider.
// An empty name means the provider has not supplied that form.
type JournalNames struct {
	Full  string
	Short string
}

// JournalNamesFromEntry reads the full and abbreviated names in MR-style exports.
func JournalNamesFromEntry(entry *bibtex.BibEntry) JournalNames {
	return JournalNames{Full: bibField(entry, "fjournal"), Short: bibField(entry, "journal")}
}

// WithJournalName returns an independent entry, preserving field spelling and
// all other contents. Missing names leave the provider's journal unchanged.
func WithJournalName(entry *bibtex.BibEntry, names JournalNames, style string) (*bibtex.BibEntry, string) {
	copy := &bibtex.BibEntry{Type: entry.Type, CiteName: entry.CiteName, Fields: make(map[string]bibtex.BibString, len(entry.Fields))}
	for key, value := range entry.Fields {
		copy.Fields[key] = value
	}
	if style == "source" || (!strings.EqualFold(entry.Type, "article") && bibField(entry, "journal") == "") {
		return copy, ""
	}
	name := names.Full
	if style == "short" {
		name = names.Short
	}
	if strings.TrimSpace(name) == "" {
		return copy, "requested " + style + " journal name is unavailable; keeping the provider's journal"
	}
	key := "journal"
	for existing := range copy.Fields {
		if strings.EqualFold(existing, key) {
			key = existing
			break
		}
	}
	copy.AddField(key, bibtex.NewBibConst(name))
	return copy, ""
}

// MissingFields checks the standard BibTeX required fields, without treating
// optional identifiers, issue numbers, or page ranges as mandatory.
func MissingFields(entry *bibtex.BibEntry) []string {
	var required []string
	switch strings.ToLower(entry.Type) {
	case "article":
		required = []string{"author", "title", "journal", "year"}
	case "book", "inbook":
		required = []string{"author or editor", "title", "publisher", "year"}
		if strings.EqualFold(entry.Type, "inbook") {
			required = append(required, "chapter or pages")
		}
	case "incollection":
		required = []string{"author", "title", "booktitle", "publisher", "year"}
	case "inproceedings", "conference":
		required = []string{"author", "title", "booktitle", "year"}
	case "proceedings":
		required = []string{"title", "year"}
	case "phdthesis", "mastersthesis":
		required = []string{"author", "title", "school", "year"}
	case "techreport":
		required = []string{"author", "title", "institution", "year"}
	case "booklet", "manual":
		required = []string{"title"}
	case "unpublished":
		required = []string{"author", "title", "note"}
	case "misc":
		// BibTeX requires no fields for misc; bibi still needs a usable title.
		required = []string{"title"}
	}
	var missing []string
	for _, requirement := range required {
		present := false
		for _, field := range strings.Split(requirement, " or ") {
			if strings.Trim(bibField(entry, field), "{} \t\r\n") != "" {
				present = true
				break
			}
		}
		if !present {
			missing = append(missing, requirement)
		}
	}
	return missing
}

func bibField(entry *bibtex.BibEntry, name string) string {
	for key, value := range entry.Fields {
		if strings.EqualFold(key, name) && value != nil {
			return strings.TrimSpace(value.String())
		}
	}
	return ""
}
