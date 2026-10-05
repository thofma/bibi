package zb

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nickng/bibtex"
	"github.com/thofma/bibi/lib/bibliography"
)

// ItemToBibEntry converts an individual zbMath item into a BibTeX entry.
// Related items may be supplied to resolve metadata for an enclosing book.
func ItemToBibEntry(item Item, relatedItems ...Item) (*bibtex.BibEntry, error) {
	if restrictedCitationMetadata(item) {
		return nil, fmt.Errorf("cannot export zbMATH record %d: %s", item.ID, restrictedMetadataNotice(item))
	}
	switch item.DocumentType.Code {
	case "j":
		return ItemToArticle(item), nil
	case "a":
		return ItemToProceedingsArticle(item, relatedItems...)
	case "b":
		return ItemToBook(item), nil
	case "p":
		return ItemToPreprint(item), nil
	case "":
		if isArXivPreprint(item) {
			return ItemToPreprint(item), nil
		}
	default:
		return nil, fmt.Errorf("unsupported zbMath document type %q", item.DocumentType.Code)
	}

	return nil, fmt.Errorf("unsupported zbMath document type %q", item.DocumentType.Code)
}

func ItemToArticle(item Item) *bibtex.BibEntry {
	entry := newBibEntry("article", item)
	addCommonFields(entry, item)

	addBibTextField(entry, "journal", ItemGetSeriesTitle(item))
	if series, ok := firstSeries(item); ok {
		addBibTextField(entry, "fjournal", series.Title)
	}
	addBibField(entry, "issn", ItemGetSeriesISSN(item))
	addBibField(entry, "volume", ItemGetSeriesVolume(item))
	addBibField(entry, "number", ItemGetSeriesIssue(item))
	addBibField(entry, "year", firstNonEmpty(item.Year, ItemGetSeriesYear(item)))

	return entry
}

func ItemToProceedingsArticle(item Item, relatedItems ...Item) (*bibtex.BibEntry, error) {
	entry := newBibEntry("inproceedings", item)
	addCommonFields(entry, item)

	relatedBook := relatedBookItem(item, relatedItems)
	bookTitle := ItemGetBookTitle(item)
	if bookTitle == "" && relatedBook != nil {
		bookTitle = ItemGetTitle(*relatedBook)
	}
	addBibTitleField(entry, "booktitle", bookTitle)
	editors := ItemGetEditors(item)
	if relatedBook != nil {
		editors = firstNonEmpty(editors, ItemGetEditors(*relatedBook))
	}
	addBibTextField(entry, "editor", editors)

	publisher := ItemGetBookPublisher(item)
	isbn := ItemGetBookISBN(item)
	year := ItemGetBookYear(item)
	if relatedBook != nil {
		publisher = firstNonEmpty(publisher, ItemGetBookPublisher(*relatedBook))
		isbn = firstNonEmpty(isbn, ItemGetBookISBN(*relatedBook))
		year = firstNonEmpty(year, ItemGetBookYear(*relatedBook), relatedBook.Year)
	}

	addBibTextField(entry, "publisher", publisher)
	addBibField(entry, "isbn", isbn)
	addBibField(entry, "year", firstNonEmpty(year, item.Year))

	seriesItem := item
	if !hasSeries(seriesItem) && relatedBook != nil {
		seriesItem = *relatedBook
	}
	addBibTextField(entry, "series", ItemGetSeriesTitle(seriesItem))
	addBibField(entry, "issn", ItemGetSeriesISSN(seriesItem))
	addBibField(entry, "volume", ItemGetSeriesVolume(seriesItem))

	return entry, nil
}

func ItemToBook(item Item) *bibtex.BibEntry {
	entry := newBibEntry("book", item)
	addCommonFields(entry, item)

	addBibTextField(entry, "editor", ItemGetEditors(item))
	addBibTextField(entry, "publisher", ItemGetBookPublisher(item))
	addBibField(entry, "isbn", ItemGetBookISBN(item))
	addBibTextField(entry, "series", ItemGetSeriesTitle(item))
	addBibField(entry, "issn", ItemGetSeriesISSN(item))
	addBibField(entry, "volume", ItemGetSeriesVolume(item))
	addBibField(entry, "year", firstNonEmpty(ItemGetBookYear(item), item.Year, ItemGetSeriesYear(item)))

	return entry
}

// ItemToPreprint returns a BibTeX misc entry for a zbMath preprint record.
// The public API currently omits document_type for some arXiv records.
func ItemToPreprint(item Item) *bibtex.BibEntry {
	entry := newBibEntry("misc", item)
	addCommonFields(entry, item)
	addBibField(entry, "year", item.Year)

	arXivID, arXivURL := ItemGetArXiv(item)
	if arXivID != "" {
		addBibField(entry, "eprint", arXivID)
		addBibField(entry, "archiveprefix", "arXiv")
	} else {
		addBibTextField(entry, "howpublished", item.Source.Source)
	}
	addBibField(entry, "url", arXivURL)

	return entry
}

func newBibEntry(entryType string, item Item) *bibtex.BibEntry {
	return bibtex.NewBibEntry(entryType, fmt.Sprintf("zbMATH%d", item.ID))
}

func addCommonFields(entry *bibtex.BibEntry, item Item) {
	addBibTextField(entry, "author", ItemGetAuthors(item))
	addBibTitleField(entry, "title", ItemGetTitle(item))
	addBibField(entry, "pages", pageRangePattern.ReplaceAllString(ItemGetSourcePages(item), "$1--$2"))

	doi, _ := ItemGetDOI(item)
	addBibField(entry, "doi", doi)
	if item.ID != 0 {
		addBibField(entry, "zbmath", fmt.Sprintf("%d", item.ID))
	}
}

var pageRangePattern = regexp.MustCompile(`([0-9]+)\s*[-–—]+\s*([0-9]+)`)

func addBibTextField(entry *bibtex.BibEntry, name, value string) {
	addBibField(entry, name, bibliography.EscapeTeXText(availableText(value)))
}

func addBibTitleField(entry *bibtex.BibEntry, name, value string) {
	if value = availableText(value); value != "" {
		addBibField(entry, name, bibliography.ProtectTitle(bibliography.EscapeTeXText(value)))
	}
}

func addBibField(entry *bibtex.BibEntry, name, value string) {
	if value = availableText(value); value != "" {
		entry.AddField(name, bibtex.NewBibConst(value))
	}
}

func ItemGetID(item Item) int {
	return item.ID
}

func ItemGetTitle(item Item) string {
	return availableText(item.Title.Title)
}

func ItemGetAuthors(item Item) string {
	return formatContributors(item.Contributors.Authors)
}

func ItemGetEditors(item Item) string {
	return formatContributors(item.Contributors.Editors)
}

func formatContributors(contributors []Author) string {
	names := make([]string, 0, len(contributors))
	for _, contributor := range contributors {
		if name := availableText(contributor.Name); name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, " and ")
}

func ItemGetSeriesTitle(item Item) string {
	series, ok := firstSeries(item)
	if !ok {
		return ""
	}
	return firstNonEmpty(series.ShortTitle, series.Title)
}

func ItemGetSourcePages(item Item) string {
	return availableText(item.Source.Pages)
}

func ItemGetSeriesISSN(item Item) string {
	series, ok := firstSeries(item)
	if !ok {
		return ""
	}
	for _, issn := range series.ISSN {
		if number := availableText(issn.Number); number != "" {
			return number
		}
	}
	return ""
}

func ItemGetSeriesVolume(item Item) string {
	series, ok := firstSeries(item)
	if !ok {
		return ""
	}
	return availableText(series.Volume)
}

func ItemGetSeriesIssue(item Item) string {
	series, ok := firstSeries(item)
	if !ok {
		return ""
	}
	return availableText(series.Issue)
}

func ItemGetSeriesYear(item Item) string {
	series, ok := firstSeries(item)
	if !ok {
		return ""
	}
	return availableText(series.Year)
}

func ItemGetBookTitle(item Item) string {
	book, ok := firstBook(item)
	if !ok {
		return ""
	}
	return availableText(book.Title)
}

func ItemGetBookPublisher(item Item) string {
	book, ok := firstBook(item)
	if !ok {
		return ""
	}
	return availableText(book.Publisher)
}

func ItemGetBookYear(item Item) string {
	book, ok := firstBook(item)
	if !ok {
		return ""
	}
	return availableText(book.Year)
}

func ItemGetBookISBN(item Item) string {
	book, ok := firstBook(item)
	if !ok {
		return ""
	}

	isbns := make([]string, 0, len(book.ISBN))
	for _, isbn := range book.ISBN {
		if number := availableText(isbn.Number); number != "" {
			isbns = append(isbns, number)
		}
	}
	return strings.Join(isbns, "; ")
}

func ItemGetDOI(item Item) (string, string) {
	for _, link := range item.Links {
		if strings.EqualFold(strings.TrimSpace(link.Type), "doi") {
			if doi := availableText(link.Identifier); doi != "" {
				return doi, availableText(link.URL)
			}
		}
	}
	return "", ""
}

// ItemGetArXiv returns the arXiv identifier and canonical URL, if present.
func ItemGetArXiv(item Item) (string, string) {
	for _, link := range item.Links {
		if strings.EqualFold(strings.TrimSpace(link.Type), "arxiv") {
			if id := availableText(link.Identifier); id != "" {
				return normalizeArXivID(id), availableText(link.URL)
			}
		}
	}

	identifier := availableText(item.Identifier)
	if strings.HasPrefix(strings.ToLower(identifier), "arxiv:") {
		return normalizeArXivID(identifier), ""
	}
	return "", ""
}

func isArXivPreprint(item Item) bool {
	if strings.EqualFold(strings.TrimSpace(item.Database), "arxiv") {
		return true
	}
	arXivID, _ := ItemGetArXiv(item)
	return arXivID != ""
}

func normalizeArXivID(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	if len(identifier) >= len("arxiv:") && strings.EqualFold(identifier[:len("arxiv:")], "arxiv:") {
		return strings.TrimSpace(identifier[len("arxiv:"):])
	}
	return identifier
}

func firstSeries(item Item) (Series, bool) {
	if len(item.Source.Series) == 0 {
		return Series{}, false
	}
	return item.Source.Series[0], true
}

func firstBook(item Item) (Book, bool) {
	if len(item.Source.Book) == 0 {
		return Book{}, false
	}
	return item.Source.Book[0], true
}

func hasSeries(item Item) bool {
	_, ok := firstSeries(item)
	return ok
}

func relatedBookItem(item Item, relatedItems []Item) *Item {
	for _, book := range item.Source.Book {
		if book.BookID == 0 {
			continue
		}
		for i := range relatedItems {
			if relatedItems[i].ID == book.BookID {
				return &relatedItems[i]
			}
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = availableText(value); value != "" {
			return value
		}
	}
	return ""
}
