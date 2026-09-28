package zb

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thofma/bibi/lib/mr"
)

const (
	defaultZBAPIBaseURL    = "https://api.zbmath.org/v1/document"
	defaultZBBibTeXBaseURL = "https://zbmath.org/bibtexoutput/"
	zbHTTPTimeout          = 15 * time.Second
	// MaxSearchResults bounds the number of results requested for an interactive search.
	MaxSearchResults = 10
)

var (
	zbAPIBaseURL    = defaultZBAPIBaseURL
	zbBibTeXBaseURL = defaultZBBibTeXBaseURL
	zbHTTPClient    = &http.Client{Timeout: zbHTTPTimeout}
)

type zbHTTPError struct {
	statusCode int
	status     string
	body       string
}

func (err *zbHTTPError) Error() string {
	return fmt.Sprintf("zbMath request returned %s: %s", err.status, strings.TrimSpace(err.body))
}

func getZBResponseAnything(search string) (string, error) {
	query := url.Values{}
	query.Set("search_string", search)
	query.Set("results_per_page", strconv.Itoa(MaxSearchResults))
	return getZBAPI("_search", query)
}

func getZBResponse(doi string) (string, error) {
	query := url.Values{}
	query.Set("page", "0")
	query.Set("results_per_page", "1")
	query.Set("DOI", doi)
	return getZBAPI("_structured_search", query)
}

func getZBAPI(path string, query url.Values) (string, error) {
	endpoint, err := url.Parse(zbAPIBaseURL)
	if err != nil {
		return "", fmt.Errorf("parse zbMath API URL: %w", err)
	}

	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/" + strings.TrimLeft(path, "/")
	endpoint.RawQuery = query.Encode()
	return getZBURL(endpoint.String(), "application/json")
}

func getZBURL(endpoint string, accept string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create zbMath request: %w", err)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}

	resp, err := zbHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request zbMath: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read zbMath response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", &zbHTTPError{
			statusCode: resp.StatusCode,
			status:     resp.Status,
			body:       string(body),
		}
	}

	return string(body), nil
}

// Search retrieves and decodes a zbMath response for the supplied query.
func Search(search string) (Response, error) {
	search = strings.TrimSpace(search)
	if search == "" {
		return Response{}, fmt.Errorf("zbMath search query cannot be empty")
	}

	body, err := getZBResponseAnything(search)
	if err != nil {
		if response, ok := zbNoResultsResponse(err); ok {
			return response, nil
		}
		return Response{}, err
	}
	response, err := ParseToStruct(body)
	if err != nil {
		return Response{}, fmt.Errorf("parse zbMath search response: %w", err)
	}
	return response, nil
}

func zbNoResultsResponse(err error) (Response, bool) {
	var httpErr *zbHTTPError
	if !errors.As(err, &httpErr) || httpErr.statusCode != http.StatusNotFound {
		return Response{}, false
	}

	response, parseErr := ParseToStruct(httpErr.body)
	if parseErr != nil || !strings.Contains(strings.ToLower(response.Status.InternalCode), "no results found") {
		return Response{}, false
	}
	return response, true
}

func ZBAnything(search string) ([]*mr.Entry, error) {
	response, err := Search(search)
	if err != nil {
		return nil, err
	}
	return ZBParseJSONInternal(response.Result)
}

func ZBGetBibtex(id string) (string, error) {
	endpoint, err := url.Parse(zbBibTeXBaseURL)
	if err != nil {
		return "", fmt.Errorf("parse zbMath BibTeX URL: %w", err)
	}
	query := url.Values{}
	query.Set("q", id)
	endpoint.RawQuery = query.Encode()
	return getZBURL(endpoint.String(), "text/plain")
}

func ZBParseJSONInternal(items []Item) ([]*mr.Entry, error) {
	entries := make([]*mr.Entry, 0, len(items))
	for _, item := range items {
		authors := make([]string, 0, len(item.Contributors.Authors))
		for _, author := range item.Contributors.Authors {
			if author.Name != "" {
				authors = append(authors, author.Name)
			}
		}

		doi, _ := ItemGetDOI(item)
		entry := &mr.Entry{
			Authors: authors,
			Year:    item.Year,
			Title:   item.Title.Title,
			Doi:     &doi,
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func parseZBResponse(body string) ([]string, string, string, string, error) {
	response, err := ParseToStruct(body)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("parse zbMath DOI response: %w", err)
	}
	if len(response.Result) == 0 {
		return nil, "", "", "", fmt.Errorf("no bibliography entry found for DOI in zbMath")
	}

	item := response.Result[0]
	authors := make([]string, 0, len(item.Contributors.Authors))
	for _, author := range item.Contributors.Authors {
		if author.Name != "" {
			authors = append(authors, author.Name)
		}
	}
	doi, _ := ItemGetDOI(item)
	return authors, item.Year, item.Title.Title, doi, nil
}

func doiToAYT(doi string) ([]string, string, string, error) {
	body, err := getZBResponse(doi)
	author := []string{}
	year := ""
	title := ""
	doifound := ""
	if err != nil {
		return author, year, title, err
	}
	author, year, title, doifound, err = parseZBResponse(body)
	if err != nil {
		return author, year, title, err
	}
	if doifound != doi {
		return author, year, title, fmt.Errorf("Mismatch between provided doi %s and found doi %s", doi, doifound)
	}
	return author, year, title, err
}
