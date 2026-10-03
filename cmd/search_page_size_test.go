package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/thofma/bibi/lib/bibliography"
	"github.com/thofma/bibi/lib/crossref"
	"github.com/thofma/bibi/util"
)

func TestSearchRequestsEnoughResultsAndSelectsBeyondTen(t *testing.T) {
	searches, exports := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/works":
			searches++
			query := r.URL.Query()
			rows, err := strconv.Atoi(query.Get("rows"))
			if err != nil || rows < 17 || query.Get("query.bibliographic") != "number theory" {
				t.Errorf("search request does not fill the selector: %s", r.URL)
			}
			var items []string
			for id := 1; id <= rows; id++ {
				items = append(items, fmt.Sprintf(`{"DOI":"10.1000/%d","title":["Work %d"],"type":"journal-article"}`, id, id))
			}
			fmt.Fprintf(w, `{"message":{"items":[%s],"total-results":100,"next-cursor":"next"}}`, strings.Join(items, ","))
		case "/works/10.1000/17/transform":
			exports++
			_, _ = io.WriteString(w, `@article{Selected17, title={Work 17}, author={Doe, Jane}, journal={Journal}, year={2020}, doi={10.1000/17}}`)
		default:
			t.Errorf("unexpected request: %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	backend := &crossref.Backend{BaseURL: server.URL, HTTPClient: server.Client()}
	services := searchServices{
		discovery: map[string]bibliography.Discoverer{"crossref": backend},
		bib:       map[string]bibliography.Provider{"crossref": backend},
	}
	picks := 0
	root := debugTestRoot(newSearchCommand(func() searchServices { return services }, func(request util.ChooserRequest) (int, error) {
		picks++
		if len(request.Choices) < 17 || request.NextToken != "next" || request.LoadPage == nil {
			t.Fatalf("selector still has too few results or lost pagination: %+v", request)
		}
		return 16, nil
	}))
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"search", "number", "theory", "--discovery", "crossref", "--bib", "crossref"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	assertMRBibTeX(t, stdout.String(), "Selected17")
	if searches != 1 || exports != 1 || picks != 1 || stderr.Len() != 0 {
		t.Fatalf("searches=%d exports=%d picks=%d stderr=%q", searches, exports, picks, stderr.String())
	}
}

func TestSearchKeepsAllReturnedWorksAndProviderCandidates(t *testing.T) {
	works := make([]bibliography.Work, 20)
	for i := range works {
		works[i] = bibliography.Work{Title: fmt.Sprintf("Work %d", i+1)}
	}
	services := searchServices{
		discovery: map[string]bibliography.Discoverer{"zb": fakeDiscovery(func(string) ([]bibliography.Work, error) { return works, nil })},
		bib: map[string]bibliography.Provider{"mr": fakeProvider(func(work bibliography.Work) ([]bibliography.Record, error) {
			if work.Title != "Work 20" {
				t.Fatalf("selected wrong discovery result: %+v", work)
			}
			records := make([]bibliography.Record, 20)
			for i := range records {
				records[i] = searchRecord(fmt.Sprintf("Candidate%d", i+1), "")
				records[i].Work = work
			}
			return records, nil
		})},
	}
	picks := 0
	root := debugTestRoot(newSearchCommand(func() searchServices { return services }, func(request util.ChooserRequest) (int, error) {
		picks++
		if len(request.Choices) != 20 || request.Confirmation != (picks == 2) {
			t.Fatalf("truncated discovery results or provider candidates: %+v", request)
		}
		return 19, nil
	}))
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"search", "number theory", "--bib", "mr"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	assertMRBibTeX(t, stdout.String(), "Candidate20")
	if picks != 2 {
		t.Fatalf("opened %d selectors, want discovery and confirmation", picks)
	}
}
