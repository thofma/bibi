package zb

import (
	"fmt"
	"strings"

	"github.com/nickng/bibtex"
)

// ItemToBibEntry converts an individual zbMath item into a BibTeX entry.
// Related items may be supplied to resolve metadata for an enclosing book.
func ItemToBibEntry(item Item, relatedItems ...Item) (*bibtex.BibEntry, error) {
	switch item.DocumentType.Code {
	case "j":
		return ItemToArticle(item), nil
	case "a":
		return ItemToProceedingsArticle(item, relatedItems...)
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

	addBibField(entry, "journal", ItemGetSeriesTitle(item))
	addBibField(entry, "issn", ItemGetSeriesISSN(item))
	addBibField(entry, "volume", ItemGetSeriesVolume(item))
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
	if bookTitle == "" {
		return nil, fmt.Errorf("zbMath proceedings item %d has no book title", item.ID)
	}
	addBibField(entry, "booktitle", bookTitle)

	publisher := ItemGetBookPublisher(item)
	isbn := ItemGetBookISBN(item)
	year := ItemGetBookYear(item)
	if relatedBook != nil {
		publisher = firstNonEmpty(publisher, ItemGetBookPublisher(*relatedBook))
		isbn = firstNonEmpty(isbn, ItemGetBookISBN(*relatedBook))
		year = firstNonEmpty(year, ItemGetBookYear(*relatedBook), relatedBook.Year)
	}

	addBibField(entry, "publisher", publisher)
	addBibField(entry, "isbn", isbn)
	addBibField(entry, "year", firstNonEmpty(year, item.Year))

	seriesItem := item
	if !hasSeries(seriesItem) && relatedBook != nil {
		seriesItem = *relatedBook
	}
	addBibField(entry, "series", ItemGetSeriesTitle(seriesItem))
	addBibField(entry, "issn", ItemGetSeriesISSN(seriesItem))
	addBibField(entry, "volume", ItemGetSeriesVolume(seriesItem))

	return entry, nil
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
		addBibField(entry, "howpublished", item.Source.Source)
	}
	addBibField(entry, "url", arXivURL)

	return entry
}

func newBibEntry(entryType string, item Item) *bibtex.BibEntry {
	return bibtex.NewBibEntry(entryType, fmt.Sprintf("zbMATH%d", item.ID))
}

func addCommonFields(entry *bibtex.BibEntry, item Item) {
	addBibField(entry, "author", ItemGetAuthors(item))
	addBibField(entry, "title", ItemGetTitle(item))
	addBibField(entry, "pages", ItemGetSourcePages(item))

	doi, _ := ItemGetDOI(item)
	addBibField(entry, "doi", doi)
	if item.ID != 0 {
		addBibField(entry, "zbmath", fmt.Sprintf("%d", item.ID))
	}
}

func addBibField(entry *bibtex.BibEntry, name, value string) {
	if value = strings.TrimSpace(value); value != "" {
		entry.AddField(name, bibtex.NewBibConst(value))
	}
}

func ItemGetID(item Item) int {
	return item.ID
}

func ItemGetTitle(item Item) string {
	return strings.TrimSpace(item.Title.Title)
}

func ItemGetAuthors(item Item) string {
	names := make([]string, 0, len(item.Contributors.Authors))
	for _, author := range item.Contributors.Authors {
		if name := strings.TrimSpace(author.Name); name != "" {
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
	return strings.TrimSpace(item.Source.Pages)
}

func ItemGetSeriesISSN(item Item) string {
	series, ok := firstSeries(item)
	if !ok {
		return ""
	}
	for _, issn := range series.ISSN {
		if number := strings.TrimSpace(issn.Number); number != "" {
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
	return strings.TrimSpace(series.Volume)
}

func ItemGetSeriesYear(item Item) string {
	series, ok := firstSeries(item)
	if !ok {
		return ""
	}
	return strings.TrimSpace(series.Year)
}

func ItemGetBookTitle(item Item) string {
	book, ok := firstBook(item)
	if !ok {
		return ""
	}
	return strings.TrimSpace(book.Title)
}

func ItemGetBookPublisher(item Item) string {
	book, ok := firstBook(item)
	if !ok {
		return ""
	}
	return strings.TrimSpace(book.Publisher)
}

func ItemGetBookYear(item Item) string {
	book, ok := firstBook(item)
	if !ok {
		return ""
	}
	return strings.TrimSpace(book.Year)
}

func ItemGetBookISBN(item Item) string {
	book, ok := firstBook(item)
	if !ok {
		return ""
	}

	isbns := make([]string, 0, len(book.ISBN))
	for _, isbn := range book.ISBN {
		if number := strings.TrimSpace(isbn.Number); number != "" {
			isbns = append(isbns, number)
		}
	}
	return strings.Join(isbns, "; ")
}

func ItemGetDOI(item Item) (string, string) {
	for _, link := range item.Links {
		if strings.EqualFold(strings.TrimSpace(link.Type), "doi") {
			return strings.TrimSpace(link.Identifier), strings.TrimSpace(link.URL)
		}
	}
	return "", ""
}

// ItemGetArXiv returns the arXiv identifier and canonical URL, if present.
func ItemGetArXiv(item Item) (string, string) {
	for _, link := range item.Links {
		if strings.EqualFold(strings.TrimSpace(link.Type), "arxiv") {
			return normalizeArXivID(link.Identifier), strings.TrimSpace(link.URL)
		}
	}

	identifier := strings.TrimSpace(item.Identifier)
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
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
