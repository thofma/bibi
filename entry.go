package main

import (
	_"encoding/json"
	"fmt"
	_"io"
	_"net/http"
	_"net/url"
	_"log"
	_"flag"
	_"os"
	_"log"
	_"reflect"
	"strings"
	"bytes"
	"github.com/nickng/bibtex"
)

/*
    Stuff for the menu
														*/

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

func DOIQuery(doi string) []*entry {
	// let's always return an array
	return ZBQueryWithDOI(doi)
}

func ZBQueryWithDOI(doi string) []*entry {
	// very disappointing, that zb has no API to retrieve the bibtex entry
	authors, year, title, _ := doiToAYT(doi)
	mrq := MRQueryAYT(authors[1], year, title)
	return mrq
}

func ArxivQueryWithIdentifier(identifier string) []*entry {
	body, _ := getArxivResponse(identifier)
	title, authors, id, year, _ := parseArxivResponse(body)
	resp, _ := arxivBibtex(title, authors, id, year)
	bib, _ := bibtex.Parse(bytes.NewReader([]byte(resp)))
	p := entry{mrbibtex: bib.Entries[0]}
	p.authors = authors
	p.title = title
	p.year = year
  p.arxiv = identifier
	result := []*entry{&p}
	return result
}

//func entryFillMRBib(e *entry) error {
//	if e.authors == nil {
//		if e.doi != nil {
//			authors, year, title, _ := doiToAYT(*e.doi)
//			e.authors = authors
//			e.year = year
//			e.title = title
//		}
//	}
//	if e.authors == nil {
//		panic(1)
//	}
//	resp, _ := mrMultiResponseFromDOI(*e.doi)
//	if len(resp) > 1 {
//		panic(1)
//	}
//	e.mrbibtex = &resp[0]
//	b := []byte(*e.mrbibtex)
//	bib, _ := bibtex.Parse(bytes.NewReader(b))
//	fmt.Printf("%T\n", bib.Entries[0])
//	for k, v := range bib.Entries[0].Fields { 
//    fmt.Printf("key[%s] value[%s]\n", k, v)
//		delete(bib.Entries[0].Fields, k)
//		bib.Entries[0].Fields[strings.ToLower(k)] = v
//	}
//	fmt.Println(bib.Entries[0].Fields)
//	fmt.Println(bib.Entries[0].Fields["author"])
//	//fmt.Printf("%T", bib.Entries[0])
//	fmt.Println(bib.Entries[0])
//	fmt.Println("authors: ", fmt.Sprint(bib.Entries[0].Fields["author"]))
//	return nil
//}

func entyArrayFromAYT(author string, year string, title string) []*entry {
	resp, _ := mrMultiResponseFromAYT(author, year, title)
	res := make([]*entry, len(resp))
	return res
}
