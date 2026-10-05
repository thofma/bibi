package bibliography

import (
	"strings"

	"github.com/nickng/bibtex"
)

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
