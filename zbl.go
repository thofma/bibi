package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	_"os"
	_"reflect"
	_"strings"
)

func getZBResonseAnything(search string) (string, error) {
	encodedString := url.QueryEscape(search)
	apiURL := fmt.Sprintf("https://api.zbmath.org/v1/document/_search?search_string=%v", encodedString)
	//https://api.zbmath.org/v1/document/_search?search_string=hofmann%20zhang%20p-adic&results_per_page=5
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		panic(err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return string(body), nil
}

func getZBResponse(doi string) (string, error) {
	encodedDOI := url.QueryEscape(doi)

	// Should do it the same as in mr.go
	apiURL := fmt.Sprintf("https://api.zbmath.org/v1/document/_structured_search?page=0&results_per_page=1&DOI=%s", encodedDOI)
//https://zbmath.org/bibtexoutput/?q=au:john+ti:surface
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return string(body), nil
}

func parseZBMultiResponse(body string) ([]*entry, error) {
	var res []*entry
	var dat map[string]interface{}
	b := []byte(body)
	if err := json.Unmarshal(b, &dat); err != nil {
		panic(err)
	}
	if dat["result"] == nil {
		return res, nil
		//return author, year, title, doi, fmt.Errorf("No bibliography entry found for doi in zbMath")
	}
	// check whether it is the correct entry
	dd := dat["result"].([]interface{})

	res, _ = ZBParseJSONInternal(dd)

	return res, nil
}

func ZBAnything(search string) ([]*entry, error) {
	resp, _ := getZBResonseAnything(search)
	res, _ := parseZBMultiResponse(resp)
	return res, nil
}

func ZBParseJSONInternal(d []interface{}) ([]*entry, error) {
	var doi string
	var year string
	var title string
	res := make([]*entry, len(d))
	for i := 0; i < len(d); i++ {
		dd := d[i].(map[string]interface{})//.([]interface{})[0]
		links := dd["links"].([]interface{})
		for i := 0; i < len(links); i++ {
			linktype := links[i].(map[string]interface{})["type"].(string)
			if linktype == "doi" {
				doi = links[i].(map[string]interface{})["identifier"].(string)
			}
		}

		authors := make([]string, len(dd["contributors"].(map[string]interface{})["authors"].([]interface{})))
		for i := 0; i < len(authors); i++ {
			authors[i] = dd["contributors"].(map[string]interface{})["authors"].([]interface{})[i].(map[string]interface{})["name"].(string)
		}
		year = dd["year"].(string)
		title = dd["title"].(map[string]interface{})["title"].(string)
	  fmt.Println(authors, year, title, doi)
		res[i] = &entry{authors: authors, year: year, title: title, doi: &doi}
	}
	return res, nil
}


func parseZBResponse(body string) ([]string, string, string, string, error) {
	// Now comes the json nightmare
	var dat map[string]interface{}

	author := []string{}
	year := ""
	title := ""
	doi := ""

	b := []byte(body)

	if err := json.Unmarshal(b, &dat); err != nil {
		panic(err)
	}
	if dat["result"] == nil {
		return author, year, title, doi, fmt.Errorf("No bibliography entry found for doi in zbMath")
	}
	// check whether it is the correct entry
	dd := dat["result"].([]interface{})[0].(map[string]interface{})//.([]interface{})[0]

	links := dd["links"].([]interface{})
	for i := 0; i < len(links); i++ {
		linktype := links[i].(map[string]interface{})["type"].(string)
		if linktype == "doi" {
			doi = links[i].(map[string]interface{})["identifier"].(string)
		}
	}

	authors := make([]string, len(dd["contributors"].(map[string]interface{})["authors"].([]interface{})))
	for i := 0; i < len(authors); i++ {
		authors[i] = dd["contributors"].(map[string]interface{})["authors"].([]interface{})[i].(map[string]interface{})["name"].(string)
	}
	year = dd["year"].(string)
	title = dd["title"].(map[string]interface{})["title"].(string)
	return authors, year, title, doi, nil
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
	author, year, title, doifound, err = parseZBResponse(body);
	if doifound != doi {
		return author, year, title, fmt.Errorf("Mismatch between provided doi %s and found doi %s", doi, doifound)
	}
	return author, year, title, err
}	
