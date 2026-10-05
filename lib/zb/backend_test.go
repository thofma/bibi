package zb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/lib/bibliography"
)

func TestSearchPagesRetainNativeRecordsAcrossPages(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		query := r.URL.Query()
		if query.Get("search_string") != "free search" || query.Get("results_per_page") != "33" {
			t.Errorf("request = %s", r.URL)
		}
		var items []string
		switch query.Get("page") {
		case "0":
			for id := 1; id <= 33; id++ {
				items = append(items, fmt.Sprintf(`{"id":%d,"document_type":{"code":"j"},"title":{"title":"Work %d"}}`, id, id))
			}
		case "1":
			items = append(items, `{"id":34,"document_type":{"code":"j"},"title":{"title":"Later work"}}`)
		default:
			t.Errorf("unexpected page: %s", r.URL)
		}
		fmt.Fprintf(w, `{"result":[%s],"status":{"nr_total_results":34}}`, strings.Join(items, ","))
	}))
	defer server.Close()
	useZBTestAPI(t, server.URL, server.Client())
	backend := &Backend{}
	backend.SetPageSize(33)
	first, err := backend.SearchPage(context.Background(), "free search", "")
	if err != nil || len(first.Works) != 33 || first.NextToken != "1" || first.Total != 34 {
		t.Fatalf("first = %+v, error = %v", first, err)
	}
	last, err := backend.SearchPage(context.Background(), "free search", first.NextToken)
	if err != nil || len(last.Works) != 1 || last.NextToken != "" {
		t.Fatalf("last = %+v, error = %v", last, err)
	}
	for _, work := range []bibliography.Work{first.Works[0], last.Works[0]} {
		records, err := backend.BibTeX(work)
		if err != nil || len(records) != 1 || records[0].Title != work.Title {
			t.Fatalf("wrong cached record: %+v, error = %v", records, err)
		}
	}
	if requests != 2 {
		t.Fatalf("native exports triggered more requests: %d", requests)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := backend.SearchPage(ctx, "free search", "1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled request error = %v", err)
	}
}

func TestSearchLaterPageWithObjectAuthorReferences(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		query := r.URL.Query()
		if query.Get("search_string") != "number theory" || query.Get("results_per_page") != "47" {
			t.Errorf("unexpected search request: %s", r.URL)
		}
		start := 1
		switch query.Get("page") {
		case "0":
		case "1":
			start = 48
		default:
			t.Errorf("unexpected page: %s", r.URL)
		}
		var items []string
		for id := start; id < start+47; id++ {
			contributors := `{"authors":[{"name":"Doe, Jane"}],"author_references":[]}`
			if id == 92 {
				// Index 44 on the second page reproduces the reported parse failure.
				contributors = `{"authors":[{"name":"Doe, Jane"}],"author_references":[{"aliases":[],"checked":"1","codes":["rosenlicht.maxwell"],"name":"Rosenlicht, Maxwell"}]}`
			}
			items = append(items, fmt.Sprintf(`{"id":%d,"document_type":{"code":"j"},"title":{"title":"Work %d"},"contributors":%s}`, id, id, contributors))
		}
		fmt.Fprintf(w, `{"result":[%s],"status":{"nr_total_results":94}}`, strings.Join(items, ","))
	}))
	defer server.Close()
	useZBTestAPI(t, server.URL, server.Client())
	backend := &Backend{}
	backend.SetPageSize(47)
	first, err := backend.SearchPage(context.Background(), "number theory", "")
	if err != nil || len(first.Works) != 47 || first.NextToken != "1" {
		t.Fatalf("first page=%+v error=%v", first, err)
	}
	second, err := backend.SearchPage(context.Background(), "number theory", first.NextToken)
	if err != nil || len(second.Works) != 47 || second.NextToken != "" {
		t.Fatalf("second page=%+v error=%v", second, err)
	}
	work := second.Works[44]
	if work.Title != "Work 92" || len(work.Authors) != 1 || work.Authors[0] != "Doe, Jane" {
		t.Fatalf("wrong later-page metadata: %+v", work)
	}
	records, err := backend.BibTeX(work)
	if err != nil || len(records) != 1 || records[0].Entry.Fields["author"].String() != "Doe, Jane" {
		t.Fatalf("later-page export=%+v error=%v", records, err)
	}
	if requests != 2 {
		t.Fatalf("cached later-page export triggered extra requests: %d", requests)
	}
}

func TestItemWorkShowsSourceTypeAndEditionNotes(t *testing.T) {
	item := Item{ID: 42, Database: "Zbl", Identifier: "1156.11046", Title: Title{Title: "Work", Addition: "2nd ed."},
		DocumentType: DocumentType{Code: "j", Description: "journal article"},
		Source:       Source{Source: "Full journal citation"},
		Links:        []Link{{Type: "arxiv", Identifier: "arXiv:1234.5678"}}}
	work := ItemWork(item)
	if work.Type != "journal article" || work.Notes != "2nd ed." || work.Venue != "Full journal citation" || work.IDs["zbl"] != "1156.11046" || work.IDs["arxiv"] != "1234.5678" {
		t.Fatalf("source metadata = %+v", work)
	}
	if work.Type == "preprint" {
		t.Fatal("arXiv link changed publication status")
	}
}

func TestItemWorkUsesSuppliedDocumentCodeWithoutGuessingFromLinks(t *testing.T) {
	for _, test := range []struct {
		code, want string
	}{
		{"j", "journal article"}, {"b", "book"}, {"a", "proceedings article"}, {"p", "preprint"}, {"", ""},
	} {
		work := ItemWork(Item{DocumentType: DocumentType{Code: test.code}, Title: Title{Title: "Preprint, second edition"},
			Links: []Link{{Type: "arxiv", Identifier: "arXiv:1234.5678"}}})
		if work.Type != test.want || work.Edition != "" {
			t.Fatalf("code=%q work=%+v", test.code, work)
		}
	}
}

func TestBackendRetainsNativeRecordsForBibTeX(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = io.WriteString(w, `{"result":[{
			"id":1, "document_type":{"code":"j"}, "title":{"title":"Work"},
			"year":"1999", "contributors":{"authors":[{"name":"Doe, Jane"}]},
			"source":{"series":[{"title":"Journal","short_title":"J.","volume":"42","issue":"7"}],"pages":"1--10"}
		}]}`)
	}))
	defer server.Close()
	useZBTestAPI(t, server.URL, server.Client())
	backend := &Backend{}
	works, err := backend.Search("Doe Work 1999")
	if err != nil || len(works) != 1 {
		t.Fatalf("works = %+v, error = %v", works, err)
	}
	records, err := backend.BibTeX(works[0])
	if err != nil || len(records) != 1 {
		t.Fatalf("records = %+v, error = %v", records, err)
	}
	if requests != 1 || records[0].Entry.Fields["volume"].String() != "42" {
		t.Fatalf("requests = %d, fields = %+v", requests, records[0].Entry.Fields)
	}
	if records[0].Entry.Fields["number"].String() != "7" || records[0].Entry.Fields["journal"].String() != "J." || records[0].Entry.Fields["fjournal"].String() != "Journal" {
		t.Fatalf("issue number or journal names were lost: %+v", records[0])
	}
	if exact, _ := bibliography.Match(works[0], records[0].Work, "zb"); !exact {
		t.Fatal("native identifier did not establish identity")
	}
}

func TestBackendDebugExplainsRejectedNativeRecords(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want string
	}{
		{"conflicting DOI", `{"result":[{"id":7,"title":{"title":"Other work"},"links":[{"type":"doi","identifier":"10.1000/wrong"}]}]}`, "stage=matching failed: all 1 native records had conflicting DOIs"},
		{"unsupported type", `{"result":[{"id":7,"document_type":{"code":"x"},"title":{"title":"Work"},"links":[{"type":"doi","identifier":"10.1000/example"}]}]}`, `stage=conversion failed: ID=7 type="x"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			useZBTestAPI(t, server.URL, server.Client())
			var trace bytes.Buffer
			defer diagnostics.SetOutput(&trace)()
			records, _ := (&Backend{}).BibTeX(bibliography.Work{DOI: "10.1000/example"})
			if len(records) != 0 || !strings.Contains(trace.String(), test.want) || !strings.Contains(trace.String(), "lookup returned 1 native records") {
				t.Fatalf("records = %+v, trace = %q", records, trace.String())
			}
		})
	}
}

func TestBackendMapsWorkFromAnotherServiceByDOI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_structured_search" || r.URL.Query().Get("DOI") != "10.1000/example" {
			t.Errorf("request = %s", r.URL.String())
		}
		_, _ = io.WriteString(w, `{"result":[{
			"id":7, "document_type":{"code":"j"}, "title":{"title":"Work"},
			"links":[{"type":"doi","identifier":"10.1000/example"}]
		}]}`)
	}))
	defer server.Close()
	useZBTestAPI(t, server.URL, server.Client())
	records, err := (&Backend{}).BibTeX(bibliography.Work{DOI: "https://doi.org/10.1000/EXAMPLE"})
	if err != nil || len(records) != 1 || records[0].Entry.CiteName != "zbMATH7" {
		t.Fatalf("records = %+v, error = %v", records, err)
	}
}

func TestBackendDOIDiscoveryRejectsWrongDOI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"result":[{"id":7,"links":[{"type":"doi","identifier":"10.1000/wrong"}]}]}`)
	}))
	defer server.Close()
	useZBTestAPI(t, server.URL, server.Client())
	works, err := (&Backend{}).Search("10.1000/example")
	if err != nil || len(works) != 0 {
		t.Fatalf("works = %+v, error = %v", works, err)
	}
}
