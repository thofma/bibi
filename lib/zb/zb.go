package zb

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/thofma/bibi/lib/mr"
)

const (
	defaultZBAPIBaseURL    = "https://api.zbmath.org/v1/document"
	defaultZBBibTeXBaseURL = "https://zbmath.org/bibtexoutput/"
	zbHTTPTimeout          = 15 * time.Second
)

var (
	zbAPIBaseURL    = defaultZBAPIBaseURL
	zbBibTeXBaseURL = defaultZBBibTeXBaseURL
	zbHTTPClient    = &http.Client{Timeout: zbHTTPTimeout}
)

func getZBResponseAnything(search string) (string, error) {
	query := url.Values{}
	query.Set("search_string", search)
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
		return "", fmt.Errorf("zbMath request returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return string(body), nil
}

func parseZBMultiResponse(body string) ([]*mr.Entry, error) {
	response, err := ParseToStruct(body)
	if err != nil {
		return nil, fmt.Errorf("parse zbMath search response: %w", err)
	}
	return ZBParseJSONInternal(response.Result)
}

func ZBAnything(search string) ([]*mr.Entry, error) {
	response, err := getZBResponseAnything(search)
	if err != nil {
		return nil, err
	}
	return parseZBMultiResponse(response)
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

func Main(args []string) {
	body := strings.Join(args, " ")
	resp, err := ParseToStruct(body)
	if err != nil {
		fmt.Println("invalid zbMath response:", err)
		return
	}
	if len(resp.Result) == 0 {
		fmt.Println("no zbMath results")
		return
	}
	ItemToBibEntry(resp.Result[0])
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
