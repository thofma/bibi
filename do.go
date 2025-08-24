package main

import (
	_"encoding/json"
	_"fmt"
	_"io"
	_"net/http"
	_"net/url"
	_"os"
	_"reflect"
	"strings"
)

func cleanAndValidateDOI(doi string) (string, bool) {
	doi = strings.TrimSpace(doi)
	prefixes := [3] string{"https://doi.org/", "http://doi.org/", "doi.org/"}
	_doi := doi
	fl := false
	for i := 0; i < len(prefixes); i++ {
		_doi, fl = strings.CutPrefix(doi, prefixes[i])
		if fl {
			break;
		}
	}
	doi = _doi
	return doi, true
}
