package phd

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/nickng/bibtex"
)

const (
	defaultMGPSearchURL = "https://www.genealogy.math.ndsu.nodak.edu/quickSearch.php"
	defaultMGPEntryURL  = "https://www.genealogy.math.ndsu.nodak.edu/id.php"
	mgpHTTPTimeout      = 15 * time.Second
)

var (
	mgpSearchURL  = defaultMGPSearchURL
	mgpEntryURL   = defaultMGPEntryURL
	mgpHTTPClient = &http.Client{Timeout: mgpHTTPTimeout}

	resultCountPattern = regexp.MustCompile(`(?is)Your\s+search\s+has\s+found\s+(?:(\d+)|no)\s+records?`)
	resultRowPattern   = regexp.MustCompile(`(?is)<tr\b[^>]*>\s*<td\b[^>]*>\s*<a\b[^>]*href\s*=\s*["'][^"']*id\.php\?id=([^&"']+)[^"']*["'][^>]*>(.*?)</a>\s*</td>\s*<td\b[^>]*>(.*?)</td>\s*<td\b[^>]*>(.*?)</td>\s*</tr>`)
	headingPattern     = regexp.MustCompile(`(?is)<h2\b[^>]*>(.*?)</h2>`)
	thesisTitlePattern = regexp.MustCompile(`(?is)<span\b[^>]*\bid\s*=\s*(?:"thesisTitle"|'thesisTitle')[^>]*>(.*?)</span>`)
	universityPattern  = regexp.MustCompile(`(?is)<span\b[^>]*color\s*:\s*#006633[^>]*>(.*?)</span>`)
	yearPattern        = regexp.MustCompile(`\b(?:1[5-9]|20|21)\d{2}\b`)
	htmlTagPattern     = regexp.MustCompile(`(?is)<[^>]*>`)
)

// MGPEntry is a thesis record returned by the Mathematics Genealogy Project.
type MGPEntry struct {
	Author     string
	Year       string
	University string
	ID         string
	Title      string
	BibTeX     *bibtex.BibEntry
}

// MGPQuery retrieves the raw MGP search response for author.
func MGPQuery(author string) (string, error) {
	author = strings.TrimSpace(author)
	if author == "" {
		return "", fmt.Errorf("MGP search terms are required")
	}

	return mgpGet(mgpSearchURL, url.Values{
		"searchTerms": {author},
		"Submit":      {"Search"},
	})
}

// MGPQueryAndResponse retrieves and parses MGP search results.
func MGPQueryAndResponse(author string) ([]MGPEntry, error) {
	response, err := MGPQuery(author)
	if err != nil {
		return nil, err
	}
	entries, err := MGPResponse(response)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// MGPResponse parses either a single MGP thesis page or a multiple-result page.
func MGPResponse(text string) ([]MGPEntry, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("parse MGP response: response is empty")
	}

	lowerText := strings.ToLower(text)
	if strings.Contains(lowerText, "thesistitle") {
		entry, err := MGPEntryGetFromSingleHit(text)
		if err != nil {
			return nil, fmt.Errorf("parse MGP single result: %w", err)
		}
		entry.BibTeX = CreateBibEntryForThesis(entry.Author, entry.Year, entry.Title, entry.University)
		return []MGPEntry{entry}, nil
	}
	if strings.Contains(lowerText, "your search has found") {
		return parseMGPResultList(text)
	}

	return nil, fmt.Errorf("parse MGP response: unrecognized result page")
}

func parseMGPResultList(text string) ([]MGPEntry, error) {
	countMatch := resultCountPattern.FindStringSubmatch(text)
	if len(countMatch) != 2 {
		return nil, fmt.Errorf("parse MGP response: could not read result count")
	}

	count := 0
	if countMatch[1] != "" {
		var err error
		count, err = strconv.Atoi(countMatch[1])
		if err != nil {
			return nil, fmt.Errorf("parse MGP response: invalid result count %q: %w", countMatch[1], err)
		}
	}
	if count == 0 {
		return []MGPEntry{}, nil
	}

	rowMatches := resultRowPattern.FindAllStringSubmatch(text, -1)
	entries := make([]MGPEntry, 0, len(rowMatches))
	for _, row := range rowMatches {
		if len(row) != 5 {
			return nil, fmt.Errorf("parse MGP response: malformed search-result row")
		}
		entry := MGPEntry{
			ID:         cleanHTMLText(row[1]),
			Author:     cleanHTMLText(row[2]),
			University: cleanHTMLText(row[3]),
			Year:       cleanHTMLText(row[4]),
		}
		if entry.ID == "" || entry.Author == "" {
			return nil, fmt.Errorf("parse MGP response: search-result row is missing an id or author")
		}
		entries = append(entries, entry)
	}

	if len(entries) != count {
		return nil, fmt.Errorf("parse MGP response: expected %d records, parsed %d", count, len(entries))
	}
	return entries, nil
}

// MGPEntryGetFromSingleHit parses one MGP thesis page.
func MGPEntryGetFromSingleHit(text string) (MGPEntry, error) {
	author := extractHTMLMatch(headingPattern, text)
	if author == "" {
		return MGPEntry{}, fmt.Errorf("could not find thesis author")
	}

	title := extractHTMLMatch(thesisTitlePattern, text)
	if title == "" {
		return MGPEntry{}, fmt.Errorf("could not find thesis title")
	}

	entry := MGPEntry{Author: author, Title: title}
	if match := universityPattern.FindStringSubmatchIndex(text); len(match) >= 4 {
		entry.University = cleanHTMLText(text[match[2]:match[3]])
		remainder := text[match[1]:]
		if closingSpan := strings.Index(strings.ToLower(remainder), "</span>"); closingSpan >= 0 {
			entry.Year = yearPattern.FindString(remainder[:closingSpan])
		}
	}

	return entry, nil
}

// MGPEntryGetBibtex retrieves a result page when necessary and creates its BibTeX entry.
func MGPEntryGetBibtex(entry MGPEntry) (*bibtex.BibEntry, error) {
	if entry.BibTeX != nil {
		return entry.BibTeX, nil
	}

	id := strings.TrimSpace(entry.ID)
	if id == "" {
		return nil, fmt.Errorf("MGP result has no id")
	}

	text, err := mgpGet(mgpEntryURL, url.Values{"id": {id}})
	if err != nil {
		return nil, fmt.Errorf("retrieve MGP result %q: %w", id, err)
	}

	detail, err := MGPEntryGetFromSingleHit(text)
	if err != nil {
		return nil, fmt.Errorf("parse MGP result %q: %w", id, err)
	}
	return CreateBibEntryForThesis(detail.Author, detail.Year, detail.Title, detail.University), nil
}

func mgpGet(baseURL string, values url.Values) (string, error) {
	endpoint, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse MGP URL: %w", err)
	}
	query := endpoint.Query()
	for key, values := range values {
		query.Del(key)
		for _, value := range values {
			query.Add(key, value)
		}
	}
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create MGP request: %w", err)
	}
	resp, err := mgpHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request MGP: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read MGP response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("MGP returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return string(body), nil
}

// CreateBibEntryForThesis creates a standard BibTeX phdthesis entry.
func CreateBibEntryForThesis(author, year, title, university string) *bibtex.BibEntry {
	key := thesisCitationKey(author, year)
	entry := bibtex.NewBibEntry("phdthesis", key)
	if author = strings.TrimSpace(author); author != "" {
		entry.AddField("author", bibtex.NewBibConst(author))
	}
	if title = strings.TrimSpace(title); title != "" {
		entry.AddField("title", bibtex.NewBibConst(BibtexEncodeTitle(title)))
	}
	if year = strings.TrimSpace(year); year != "" {
		entry.AddField("year", bibtex.NewBibConst(year))
	}
	if university = strings.TrimSpace(university); university != "" {
		entry.AddField("school", bibtex.NewBibConst(university))
	}
	return entry
}

func thesisCitationKey(author, year string) string {
	familyName, _, _ := strings.Cut(author, ",")
	familyName = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, familyName)
	if familyName == "" {
		familyName = "thesis"
	}
	return familyName + strings.TrimSpace(year)
}

// BibtexEncodeTitle preserves capitalized words in BibTeX title fields.
func BibtexEncodeTitle(title string) string {
	words := strings.Fields(title)
	for i, word := range words {
		words[i] = BibtexifyWord(word)
	}
	return strings.Join(words, " ")
}

func BibtexifyWord(word string) string {
	var result strings.Builder
	inUpperRun := false
	for _, r := range word {
		if unicode.IsUpper(r) {
			if !inUpperRun {
				result.WriteRune('{')
				inUpperRun = true
			}
			result.WriteRune(r)
			continue
		}
		if inUpperRun {
			result.WriteRune('}')
			inUpperRun = false
		}
		result.WriteRune(r)
	}
	if inUpperRun {
		result.WriteRune('}')
	}
	return result.String()
}

func extractHTMLMatch(pattern *regexp.Regexp, text string) string {
	match := pattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	return cleanHTMLText(match[1])
}

func cleanHTMLText(value string) string {
	value = htmlTagPattern.ReplaceAllString(value, " ")
	value = html.UnescapeString(value)
	return strings.Join(strings.Fields(value), " ")
}
