package mr

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	_"log"
	_"flag"
	"os"
	_"log"
	_"reflect"
	"strings"
	"bytes"
	"github.com/nickng/bibtex"
	"github.com/thofma/bibi/util"
)

/*
    Stuff for the menu
														*/

// func mrMultiResponseFromDOI(doi string) ([]string, error) {
// 	author, year, title, err := doiToAYT(doi)
// 	if err != nil {
// 		return []string{}, err
// 	}
//   // TODO: check error
// 
// 	return mrMultiResponseFromAYT(author[1], year, title)
// }
// 
// func mrSingleResponseFromDOI(doi string) (string, error) {
// 	res, err := mrMultiResponseFromDOI(doi)
// 	if err != nil {
// 		return "", err
// 	}
// 	// TODO: check err
// 	return res[0], nil
// }

// getBibTeX queries MRLookup with author and year and returns the BibTeX entry
func mrMultiResponseFromAYT(author, year, title string) ([]string, error) {
	empty := []string{}

	// Fix the name
	// MR wants "LAST, FIRST"
	author, err := fixName(author)
	if err != nil {
		return empty, err
	}

	baseURL := "https://mathscinet.ams.org/mathscinet/api/freetools/mrlookup"
	data := url.Values{}
	data.Add("author", author)
	data.Set("year", year)
	data.Set("journal", "")
	data.Set("firstPage", "")
	data.Set("lastPage", "")
	data.Set("title", title)

	reqURL := baseURL + "?" + data.Encode()

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

	// Now comes the json nightmare
	var dat map[string]interface{}

	b := []byte(text)

	if err := json.Unmarshal(b, &dat); err != nil {
		panic(err)
	}

	dd := dat["all"].(map[string]interface{})
	resu := dd["results"].([]interface {})
	ress := make([]string, len(resu))

	for i := 0; i < len(resu); i++ {
		//fmt.Println("NEXT RESULT")
		//fmt.Printf("Type resu[i]: %T \n", resu[i])
		resucast := ((resu[i].(map[string]interface{}))["bibTexFormat"]).(string)
		ress[i] = strings.TrimSpace(resucast)
		//fmt.Println(resu[i])
	}
	//fmt.Println("2 \n")
	return ress, nil
}

func mrSingleResponseFromAYT(author, year, title string) (string, error) {
	res, _ := mrMultiResponseFromAYT(author, year, title)
	// TODO: check err
	return res[0], nil
}

func fixName(name string) (string, error) {
	// We only accept "LAST,FIRST" or "LAST"
	// If there are more commas, we abort
	if strings.Contains(name, ",") {
		splitted := strings.Split(name, ",")
		if len(splitted) > 2 {
			return "", fmt.Errorf("Malformed name. Should be of the form \"LAST\" or \"LAST,FIRST\"")
		}
		name = splitted[0] + ", " + splitted[1]
	}
	return name, nil
}
type entry struct {
	doi *string
	authors []string
	title string
	year string
	doctype string
	arxiv string
	mrbibtex *bibtex.BibEntry
	zbbibtex *bibtex.BibEntry
}

func MRQueryAYT(author string, year string, title string) []*entry {
	resp, _ := mrMultiResponseFromAYT(author, year, title)
	result := make([]*entry, len(resp))
	for i := 0; i < len(result); i++ {
		b := []byte(resp[i])
		bib, _ := bibtex.Parse(bytes.NewReader(b))
		p := entry{mrbibtex: bib.Entries[0]}
		p.authors = ExtractAuthorsFromBibtex(p.mrbibtex)
		p.title = ExtractTitleFromBibtex(p.mrbibtex)
		p.year = ExtractYearFromBibtex(p.mrbibtex)
		doi := ExtractDOIFromBibtex(p.mrbibtex)
		p.doi = &doi
		result[i] = &p
	}
	return result
}

func ExtractFieldFromBibtex(bib *bibtex.BibEntry, field string) string {
	for k, v := range bib.Fields { 
		if strings.ToLower(k) == strings.ToLower(field) {
			return strings.TrimSpace(fmt.Sprint(v))
		}
	}
	return ""
}

func ExtractAuthorsFromBibtex(bib *bibtex.BibEntry) []string {
	authors := strings.Split(ExtractFieldFromBibtex(bib, "author"), " and ")
	return authors
}

func ExtractTitleFromBibtex(bib *bibtex.BibEntry) string {
	return ExtractFieldFromBibtex(bib, "title")
}

func ExtractYearFromBibtex(bib *bibtex.BibEntry) string {
	return ExtractFieldFromBibtex(bib, "year")
}

func ExtractDOIFromBibtex(bib *bibtex.BibEntry) string {
	return ExtractFieldFromBibtex(bib, "doi")
}

func entryFromDoi(doi string) *entry {
	p := entry{doi: &doi}
	return &p
}
//
//func DOIQuery(doi string) []*entry {
//	// let's always return an array
//	return ZBQueryWithDOI(doi)
//}
//
// func ZBQueryWithDOI(doi string) []*entry {
// 	// very disappointing, that zb has no API to retrieve the bibtex entry
// 	authors, year, title, _ := doiToAYT(doi)
// 	mrq := MRQueryAYT(authors[1], year, title)
// 	return mrq
// }
//
//func ArxivQueryWithIdentifier(identifier string) []*entry {
//	body, _ := getArxivResponse(identifier)
//	title, authors, id, year, _ := parseArxivResponse(body)
//	resp, _ := arxivBibtex(title, authors, id, year)
//	bib, _ := bibtex.Parse(bytes.NewReader([]byte(resp)))
//	p := entry{mrbibtex: bib.Entries[0]}
//	p.authors = authors
//	p.title = title
//	p.year = year
//  p.arxiv = identifier
//	result := []*entry{&p}
//	return result
//}

func Main(args []string) {
	if len(args) > 3 {
		fmt.Println("expected at most three arguments")
		os.Exit(1)
	}
	if len(args) == 0 {
		fmt.Println("expected at least one argument")
		os.Exit(1)
	}
	author := args[0]
	var title string
	var year string
	//bibtex, _ := mrMultiResponseFromAYT(author, year, title)
	if len(args) >= 2 {
		title = args[1]
	}
	if len(args) == 3 {
		year = args[2]
	}

	if author == "-" {
		author = ""
	}

	if title == "-" {
		title = ""
	}

	if year == "-" {
		year = ""
	}

	bibtex := MRQueryAYT(author, year, title)

	if len(bibtex) == 0 {
		fmt.Println("No entry found!")
		os.Exit(1)
	}
	if len(bibtex) > 1 {
		choices := make([]string, len(bibtex))
		for i := 0; i < len(bibtex); i++ {
			choices[i] = bibtex[i].authors[0]
			if len(bibtex[i].authors) > 0 {
				choices[i] = choices[i] + " et al."
			}
			choices[i] = choices[i] + ", " + bibtex[i].year + ", " + bibtex[i].title
		}
		choice := util.RunChooser(choices)
		fmt.Print(bibtex[choice].mrbibtex)
	} else {
		fmt.Print(bibtex[0].mrbibtex)
	}
	os.Exit(0)
}
