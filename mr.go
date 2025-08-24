package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	_"os"
	_"reflect"
	"strings"
)

func mrMultiResponseFromDOI(doi string) ([]string, error) {
	author, year, title, err := doiToAYT(doi)
	if err != nil {
		return []string{}, err
	}
  // TODO: check error

	return mrMultiResponseFromAYT(author[1], year, title)
}

func mrSingleResponseFromDOI(doi string) (string, error) {
	res, err := mrMultiResponseFromDOI(doi)
	if err != nil {
		return "", err
	}
	// TODO: check err
	return res[0], nil
}

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
