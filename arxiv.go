//https://export.arxiv.org/api/query?id_list=2507.21975

package main

import (
	_"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	_"os"
	_"reflect"
	"strings"
	"github.com/clbanning/mxj/v2"
)

func getArxivResponse(identifier string) (string, error) {
	//empty := ""

//author, err := fixArxivIdentifier(author)
	baseURL := "https://export.arxiv.org/api/query"
	data := url.Values{}
	data.Set("id_list", identifier)

	reqURL := baseURL + "?" + data.Encode()
	//fmt.Println("reqURL", reqURL)

	empty := ""

	// Make the request
	resp, err := http.Get(reqURL)
	if err != nil {
		return empty, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return empty, fmt.Errorf("non-OK HTTP status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return empty, fmt.Errorf("failed to read response body: %v", err)
	}

	text := string(body)

	return text, nil
}

func arxivBibtex(title string, authors []string, id string, year string) (string, error) {
	author_string := ""
	key := ""
	for i, name := range authors {
		if i > 0 {
		  author_string = author_string + " and "
		}
		author_string = author_string + name
		key = key + name[:strings.Index(name,",")]
	}
	key = key + year
	//fmt.Println(author_string)
	//fmt.Println(key)

	bibtex := fmt.Sprintf("@misc{%v,\n" +
												"  archiveprefix = {arXiv},\n" +
												"  author = {%v},\n" +
												"  eprint = {%v},\n" +
												"  howpublished = {arXiv:%v},\n" +
												"  title = {%v},\n" +
												"  year = {%v}}", key, id, author_string, id, title, year)
	return bibtex, nil
}
//  @misc{nicholson2024cancellation,
//	archiveprefix = {arXiv},
//	author = {Nicholson, J.},
//	date-added = {2024-06-17 11:25:29 +0100},
//	date-modified = {2024-06-28 16:12:26 +0100},
//	eprint = {2406.08692},
//	howpublished = {arXiv:2406.08692},
//	primaryclass = {id='math.GR' full_name='Group Theory' is_active=True alt_name=None in_archive='math' is_general=False description='Finite groups, topological groups, representation theory, cohomology, classification and structure'},
//	title = {The cancellation property for projective modules over integral group rings},
//	year = {2024}}


func parseArxivResponse(body string) (string, []string, string, string, error) {
	b := []byte(body)
	mv, err := mxj.NewMapXml(b)
	if err != nil {
		fmt.Errorf("Error parsing arXiv response: %v", err)
	}
	title := mv["feed"].(map[string]interface{})["entry"].(map[string]interface{})["title"].(string)
	//fmt.Println("title:", title)
	aut := mv["feed"].(map[string]interface{})["entry"].(map[string]interface{})["author"].([]interface{})
	auts := make([]string,len(aut))
	for i := 0; i < len(aut); i++ {
		a := aut[i].(map[string]interface{})["name"].(string)
		// split at last occurence of " "
		lastInd := strings.LastIndex(a, " ")
		auts[i] = a[lastInd+1:] + ", " + a[:lastInd]
		//fmt.Println(auts[i])
	}
	id := mv["feed"].(map[string]interface{})["entry"].(map[string]interface{})["id"].(string)
	//fmt.Println("id: ", id)
	id, fl := strings.CutPrefix(id, "http://arxiv.org/abs/")
	if !fl {
		fmt.Errorf("Parsing arXiv id failed: %v", id)
	}
	year := mv["feed"].(map[string]interface{})["entry"].(map[string]interface{})["published"].(string)
	year = year[:strings.Index(year,"-")]
	//fmt.Println("year", year)
	return title, auts, id, year, nil
}

func arxivResponseFromIdentifier(identifier string) (string, error) {
	body, err := getArxivResponse(identifier)
	title, authors, id, year, err := parseArxivResponse(body)
	resp, err := arxivBibtex(title, authors, id, year)
	return resp, err
}

//func arxivSingleResponseFromIdentifier(identifier string) (string, error) {
//	res, err := mrMultiResponseFromDOI(doi)
//	if err != nil {
//		return "", err
//	}
//	// TODO: check err
//	return res[0], nil
//}
//
//// getBibTeX queries MRLookup with author and year and returns the BibTeX entry
//func mrMultiResponseFromAYT(author, year, title string) ([]string, error) {
//	empty := []string{}
//
//	// Fix the name
//	// MR wants "LAST, FIRST"
//	author, err := fixName(author)
//	if err != nil {
//		return empty, err
//	}
//
//	baseURL := "https://mathscinet.ams.org/mathscinet/api/freetools/mrlookup"
//	data := url.Values{}
//	data.Add("author", author)
//	data.Set("year", year)
//	data.Set("journal", "")
//	data.Set("firstPage", "")
//	data.Set("lastPage", "")
//	data.Set("title", title)
//
//	reqURL := baseURL + "?" + data.Encode()
//
