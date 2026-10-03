package zb

import (
	"fmt"
	"strings"

	"github.com/thofma/bibi/lib/bibliography"
)

const licensePlaceholder = "zbMATH Open Web Interface contents unavailable due to conflicting licenses."

// The API uses this notice in place of individual metadata fields. A notice in
// a review or reference does not imply that the citation itself is restricted.
func isLicensePlaceholder(value string) bool {
	value = strings.TrimSuffix(strings.TrimSpace(value), ".")
	return strings.EqualFold(value, strings.TrimSuffix(licensePlaceholder, "."))
}

func availableText(value string) string {
	if isLicensePlaceholder(value) {
		return ""
	}
	return strings.TrimSpace(value)
}

func restrictedCitationMetadata(item Item) bool {
	values := []string{item.Title.Title, item.Year, item.Source.Pages}
	for _, contributors := range [][]Author{item.Contributors.Authors, item.Contributors.Editors} {
		for _, contributor := range contributors {
			values = append(values, contributor.Name)
		}
	}
	for _, series := range item.Source.Series {
		values = append(values, series.Title, series.ShortTitle, series.Volume, series.Issue, series.Year)
		for _, issn := range series.ISSN {
			values = append(values, issn.Number)
		}
	}
	for _, book := range item.Source.Book {
		values = append(values, book.Title, book.Publisher, book.Year)
		for _, isbn := range book.ISBN {
			values = append(values, isbn.Number)
		}
	}
	for _, link := range item.Links {
		if strings.EqualFold(link.Type, "doi") || strings.EqualFold(link.Type, "arxiv") {
			values = append(values, link.Identifier, link.URL)
		}
	}
	for _, value := range values {
		if isLicensePlaceholder(value) {
			return true
		}
	}
	return false
}

func restrictedMetadataNotice(item Item) string {
	notice := "zbMATH Open withholds citation metadata because of license restrictions."
	if doi, _ := ItemGetDOI(item); doi != "" {
		if doi, ok := bibliography.DOIQuery(doi); ok {
			return notice + fmt.Sprintf(" Retry with: bibi search %s --discovery crossref --bib crossref", doi)
		}
	}
	return notice + " Try your search with --discovery crossref --bib crossref."
}
