package main

import (
	_"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"html"
	_"os"
	_"reflect"
	"strings"
	"unicode"
	"strconv"
	"github.com/nickng/bibtex"
)

type MGPEntry struct {
	author string
	year string
	uni string
	id string
	title string
	bib *bibtex.BibEntry
}

func MGPQuery(author string) (string, error) {
	baseURL := "https://www.genealogy.math.ndsu.nodak.edu/quickSearch.php"
	data := url.Values{}
	data.Add("searchTerms", author)
	data.Set("Submit", "Search")

	reqURL := baseURL + "?" + data.Encode()

	result := ""

	// Make the request
	resp, err := http.Get(reqURL)
	if err != nil {
		return result, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("non-OK HTTP status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("failed to read response body: %v", err)
	}

	return string(body), nil
}

func MGPQueryAndResponse(author string) ([]MGPEntry, error) {
	resp, _ := MGPQuery(author)
	res, _ := MGPResponse(resp)
	return res, nil
}

func MGPResponse(text string) ([]MGPEntry, error) {
	var result []MGPEntry

	if strings.Contains(text, "Dissertation") {
		ent := MGPEntryGetFromSingleHit(text)
		//fmt.Println(ent)
		ent.bib = CreateBibEntryForThesis(ent.author, ent.year, ent.title, ent.uni)
		return []MGPEntry{ent}, nil
	}

	if strings.Contains(text, "Your search has found") {
		_, after, _ := strings.Cut(text, "Your search has found")
		//fmt.Println(after)
		nresult, _, _ := strings.Cut(after, "records")
		nnresult, _ := strconv.Atoi(strings.TrimSpace(nresult))
		//fmt.Println(nnresult)
		if nnresult > 0 {
			result = make([]MGPEntry, nnresult)
			splitted := strings.Split(text, "<tr><td>")
			// <a href="id.php?id=27377">Hofmann, Bernd</a></td>
			// <td>Georg-August-Universit&auml;t G&ouml;ttingen</td>
			// <td>1997</td></tr>\n\n\n
			for i := 1; i < len(splitted); i++ {
				id := strings.SplitN(splitted[i], "\"", 3)[1]
				id = strings.Split(id, "=")[1]
				id  = strings.TrimSpace(id)
				_, name, _ := strings.Cut(splitted[i], ">")
				name, _, _ = strings.Cut(name, "<")
				name  = html.UnescapeString(strings.TrimSpace(name))
				year, uni, _ := strings.Cut(splitted[i], "<td>")
				uni, year, _ = strings.Cut(uni, "</td>\n<td>")
				year, _, _ = strings.Cut(year, "</td></tr>")
				year = html.UnescapeString(strings.TrimSpace(year))
				uni = html.UnescapeString(strings.TrimSpace(uni))
				//fmt.Println("id: ", id, " name ", name)
				//fmt.Println("uni: ", uni, " year ", year)
				ent := MGPEntry{id: id, author: name, uni: uni, year: year}
				result[i - 1] = ent
			}
		}
	}
	//fmt.Println(result)

	return result, nil
}

func MGPEntryGetFromSingleHit(text string) MGPEntry {
	// only one hit and we found it
	_, after, _ := strings.Cut(text, "<div style=\"text-align: center\"><span style=\"color: #000066\">")
	_, after, _ = strings.Cut(after, ":</span> <span style=\"font-style:italic\" id=\"thesisTitle\">")
	title, _, _ := strings.Cut(after, "</span></div>")
	title = html.UnescapeString(strings.TrimSpace(title))
	_, after, _ = strings.Cut(text, "<h2 style=\"text-align: center; margin-bottom: 0.5ex; margin-top: 1ex\">")
	var uni string
	author, after, _ := strings.Cut(after, "</h2>")
	// Unescape HTML and remove spurious double spaces
	author = strings.Join(strings.Fields(html.UnescapeString(author)), " ")
	uni, after, _ = strings.Cut(after, "</span>")
	year, after, _ := strings.Cut(after, "</span")
	year = strings.TrimSpace(year)
	lastInd := strings.LastIndex(uni, ">")
	uni = uni[lastInd + 1:]
	// Unescape HTML
	uni = html.UnescapeString(strings.TrimSpace(uni))
	return MGPEntry{author: author, title: title, year: year, uni: uni}
}

func MGPEntryGetBibtex(entry MGPEntry) (*bibtex.BibEntry, error) {
	if entry.bib != nil {
		return entry.bib, nil
	}
	//fmt.Println("bibentry not nil")
	baseURL := "https://www.genealogy.math.ndsu.nodak.edu/id.php"
	data := url.Values{}
	data.Set("id", entry.id)

	reqURL := baseURL + "?" + data.Encode()

	result := bibtex.BibEntry{}

	// Make the request
	resp, err := http.Get(reqURL)
	if err != nil {
		return &result, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &result, fmt.Errorf("non-OK HTTP status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &result, fmt.Errorf("failed to read response body: %v", err)
	}

	text := string(body)

	//fmt.Println(text)

	_, after, _ := strings.Cut(text, "<div style=\"text-align: center\"><span style=\"color: #000066\">")
	_, after, _ = strings.Cut(after, ":</span> <span style=\"font-style:italic\" id=\"thesisTitle\">")
	title, _, _ := strings.Cut(after, "</span></div>")
	title = html.UnescapeString(strings.TrimSpace(title))

	res := CreateBibEntryForThesis(entry.author, entry.year, title, entry.uni)
	//fmt.Println(kind)
	//fmt.Println(title)
	return res, nil
}

func CreateBibEntryForThesis(author string, year string, title string, university string) *bibtex.BibEntry {
	a, _, _ := strings.Cut(author, ",")
	a = strings.TrimSpace(a)
	entry := bibtex.NewBibEntry("thesis", fmt.Sprintf("%v%v", a, year))
	entry.AddField("author", bibtex.NewBibConst(author))
	entry.AddField("title", bibtex.NewBibConst(BibtexEncodeTitle(title)))
	entry.AddField("year", bibtex.NewBibConst(year))
	entry.AddField("school", bibtex.NewBibConst(university))
	return entry
}

func BibtexEncodeTitle(title string) string {
	words := strings.Split(title, " ")
	res := ""
	for i := 0; i < len(words); i++ {
		 res = res + " " + BibtexifyWord(words[i])
	}
	res = strings.TrimSpace(res)
	return res
}

func BibtexifyWord(word string) string {
	wrunes := []rune(word)
	res := ""
	upper := false	
	for i := 0; i < len(wrunes); i++ {
		if unicode.IsUpper(wrunes[i]) && !upper {
			res = res + "{" + string(wrunes[i])
			upper = true
			if i == len(wrunes) - 1 {
				res = res + "}"
			}
		} else if unicode.IsUpper(wrunes[i]) && upper {
			res = res + string(wrunes[i])
			if i == len(wrunes) - 1 {
				res = res + "}"
			}
		} else if !unicode.IsUpper(wrunes[i]) && upper {
			res = res + "}" + string(wrunes[i])
			upper = false
		} else {
			res = res + string(wrunes[i])
		}
	}
	return res
}

// func MGPEntryGetBibtex
// use id to retrieve Bibtex
//
// 					https://www.genealogy.math.ndsu.nodak.edu/id.php?id=204187
// 
// 					response
//
// <div id="paddingWrapper">
// <script>
// 	MathJax = {
// tex: {
// 	inlineMath: [['$', '$'], ['\\(', '\\)']]
// },
// svg: {
// 	fontCache: 'global'
// }
// };
// </script>
// <script id="MathJax-script" async="" src="https://cdn.jsdelivr.net/npm/mathjax@3/es5/tex-mml-chtml.js"></script>
// 
// 
// <h2 style="text-align: center; margin-bottom: 0.5ex; margin-top: 1ex">
// Tommy  Hofmann </h2>
// 
// <p style="text-align: center; margin-top: 0; margin-bottom: 0px; font-size: small">
//  <a href="http://www.ams.org/mathscinet/MRAuthorID/1074375">MathSciNet</a>
// </p>
// 
// <div style="margin-left: auto; margin-right: auto; width: 300px"><hr style="width: 300px; height: 0; border-style: solid; border-width: 2px 0 0 0; color: gray; background-color: gray"></div>
// 
// <div style="line-height: 30px; text-align: center; margin-bottom: 1ex">
//   <span style="margin-right: 0.5em">Dr. rer. nat. <span style="color:
//   #006633; margin-left: 0.5em">Technische Universität Kaiserslautern</span> 2016</span>
// 
// <img src="img/flags/Germany.gif" alt="Germany" width="50" height="30" style="border: 0; vertical-align: middle" title="Germany">
// </div>
// 
// 
// <div style="text-align: center"><span style="color: #000066">Dissertation:</span> <span style="font-style:italic" id="thesisTitle">
// 
// Integrality of representations of finite groups</span></div>
// 
// <p style="text-align: center; line-height: 2.75ex">Advisor 1: <a href="id.php?id=28657">Claus  Fieker</a><br></p><p style="text-align: center">No students known.</p>
// <p style="font-size: small; text-align: center">If you have additional information or
//  corrections regarding this mathematician, please use the <a href="submit-data.php?id=204187&amp;edit=0">update form</a>. To submit students of this
//  mathematician, please use the <a href="submit-data.php?id=NEW&amp;edit=0">new
// 	    data form</a>, noting this mathematician's MGP ID of 204187 for the advisor ID.</p>
// 
// </div>
// //
