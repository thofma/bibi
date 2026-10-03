package crossref

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/lib/bibliography"
)

func TestSearchCursorPagesAndSourceDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("query.bibliographic") != "free search" || q.Get("rows") != "10" || q.Has("offset") {
			t.Errorf("request = %s", r.URL)
		}
		var items []string
		switch q.Get("cursor") {
		case "*":
			for id := 1; id <= 10; id++ {
				items = append(items, fmt.Sprintf(`{"DOI":"10.1000/%d","title":["Work %d"]}`, id, id))
			}
		case "opaque+/= token":
			items = append(items, `{"DOI":"10.1000/11","title":["Later work"],"container-title":["Full Journal Name"],"type":"book","edition-number":"2","subtype":"translation"}`)
		default:
			t.Errorf("unexpected cursor %q", q.Get("cursor"))
		}
		fmt.Fprintf(w, `{"message":{"items":[%s],"total-results":11,"next-cursor":"opaque+/= token"}}`, strings.Join(items, ","))
	}))
	defer server.Close()
	backend := &Backend{BaseURL: server.URL, HTTPClient: server.Client()}
	first, err := backend.SearchPage(context.Background(), "free search", "")
	if err != nil || len(first.Works) != 10 || first.NextToken != "opaque+/= token" || first.Total != 11 {
		t.Fatalf("first = %+v, error = %v", first, err)
	}
	last, err := backend.SearchPage(context.Background(), "free search", first.NextToken)
	if err != nil || len(last.Works) != 1 || last.NextToken != "" {
		t.Fatalf("last = %+v, error = %v", last, err)
	}
	work := last.Works[0]
	if work.Venue != "Full Journal Name" || work.Type != "book" || work.Edition != "2" || work.Notes != "Subtype: translation" {
		t.Fatalf("source metadata = %+v", work)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := backend.SearchPage(ctx, "free search", first.NextToken); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled request error = %v", err)
	}
}

func TestWorkRecognizesOnlyExplicitPreprintSubtype(t *testing.T) {
	for _, test := range []struct {
		subtype, want string
	}{
		{"preprint", "preprint"},
		{"", "posted-content"},
		{"other", "posted-content"},
	} {
		work := (item{Type: "posted-content", Subtype: test.subtype}).work()
		if work.Type != test.want {
			t.Fatalf("subtype=%q type=%q, want %q", test.subtype, work.Type, test.want)
		}
	}
}

func TestSearchMapsFreeTextToWork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works" || r.URL.Query().Get("query.bibliographic") != "serre local fields 1979" || r.URL.Query().Get("rows") != "10" {
			t.Errorf("request = %s", r.URL.String())
		}
		if r.Header.Get("Accept") != "application/json" || !strings.Contains(r.Header.Get("User-Agent"), "bibi") {
			t.Errorf("headers = %v", r.Header)
		}
		_, _ = io.WriteString(w, `{"message":{"items":[{
			"DOI":"10.1000/example", "title":["Local fields"],
			"author":[{"family":"Serre","given":"Jean-Pierre"},{"name":"Research group"}],
			"published":{"date-parts":[[1979]]}, "created":{"date-parts":[[2026]]}
		}]}}`)
	}))
	defer server.Close()
	backend := &Backend{BaseURL: server.URL, HTTPClient: server.Client()}
	works, err := backend.Search("serre local fields 1979")
	if err != nil {
		t.Fatal(err)
	}
	want := bibliography.Work{Title: "Local fields", Authors: []string{"Serre, Jean-Pierre", "Research group"}, Year: "1979", DOI: "10.1000/example"}
	if len(works) != 1 || !reflect.DeepEqual(works[0], want) {
		t.Fatalf("works = %+v, want %+v", works, want)
	}
}

func TestSearchDOIUsesExactRoute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works/10.1000/example" || r.URL.RawQuery != "" {
			t.Errorf("request = %s", r.URL.String())
		}
		_, _ = io.WriteString(w, `{"message":{"DOI":"10.1000/example","title":["Work"],"issued":{"date-parts":[[1999]]}}}`)
	}))
	defer server.Close()
	backend := &Backend{BaseURL: server.URL, HTTPClient: server.Client()}
	works, err := backend.Search("https://doi.org/10.1000/EXAMPLE")
	if err != nil || len(works) != 1 || works[0].Year != "1999" {
		t.Fatalf("works = %+v, error = %v", works, err)
	}
}

func TestBibTeXRetrievesCrossrefExport(t *testing.T) {
	const source = `@article{CrossrefKey,
		author = {Hofmann, Tommy and Zhang, Yinan},
		title = {Valuations of {p}-adic regulators},
		doi = {10.1000/EXAMPLE},
		year = {2016},
		month = Dec,
	}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works/10.1000/example/transform" || r.Header.Get("Accept") != "application/x-bibtex" {
			t.Errorf("request = %s, headers = %v", r.URL.String(), r.Header)
		}
		_, _ = io.WriteString(w, source)
	}))
	defer server.Close()
	backend := &Backend{BaseURL: server.URL, HTTPClient: server.Client()}
	records, err := backend.BibTeX(bibliography.Work{DOI: "doi:10.1000/example"})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Entry.CiteName != "CrossrefKey" || records[0].Entry.Fields["title"].String() != "Valuations of {p}-adic regulators" {
		t.Fatalf("records = %+v", records)
	}
	if got := records[0].Entry.Fields["month"].String(); got != "December" {
		t.Errorf("month = %q, want December", got)
	}
	if _, err := backend.BibTeX(bibliography.Work{Title: "No DOI"}); err == nil || !strings.Contains(err.Error(), "requires a DOI") {
		t.Fatalf("missing DOI error = %v", err)
	}
}

func TestBibTeXRetainsJournalNamesFromDiscovery(t *testing.T) {
	for _, query := range []string{"free text", "10.1000/example"} {
		t.Run(query, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				const metadata = `{"DOI":"10.1000/example","title":["Work"],"container-title":["Algebra & Geometry"],"short-container-title":["Alg. Geom."]}`
				switch r.URL.Path {
				case "/works":
					_, _ = io.WriteString(w, `{"message":{"items":[`+metadata+`]}}`)
				case "/works/10.1000/example":
					_, _ = io.WriteString(w, `{"message":`+metadata+`}`)
				case "/works/10.1000/example/transform":
					_, _ = io.WriteString(w, `@article{Export,title={Work},journal={Algebra \& Geometry},doi={10.1000/example}}`)
				default:
					t.Errorf("unexpected request: %s", r.URL)
				}
			}))
			defer server.Close()
			backend := &Backend{BaseURL: server.URL, HTTPClient: server.Client()}
			works, err := backend.Search(query)
			if err != nil || len(works) != 1 {
				t.Fatalf("works = %+v, error = %v", works, err)
			}
			records, err := backend.BibTeX(works[0])
			if err != nil || len(records) != 1 {
				t.Fatalf("records = %+v, error = %v", records, err)
			}
			want := bibliography.JournalNames{Full: `Algebra \& Geometry`, Short: "Alg. Geom."}
			if records[0].Journals != want || requests != 2 {
				t.Errorf("journals = %+v, requests = %d", records[0].Journals, requests)
			}
			if got := records[0].Entry.Fields["journal"].String(); got != want.Full {
				t.Errorf("native export was changed: %q", got)
			}
		})
	}
}

func TestCrossrefFailures(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
		bib    bool
		empty  bool
	}{
		{"rate limited", 429, "retry later", false, false},
		{"invalid JSON", 200, "{", false, false},
		{"missing DOI", 404, "missing", true, true},
		{"invalid export", 200, "not BibTeX", true, false},
		{"different DOI export", 200, `@article{x,doi={10.1000/wrong}}`, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			backend := &Backend{BaseURL: server.URL, HTTPClient: server.Client()}
			var err error
			if test.bib {
				var records []bibliography.Record
				records, err = backend.BibTeX(bibliography.Work{DOI: "10.1000/example"})
				if test.empty && len(records) != 0 {
					t.Fatalf("records = %+v, want none", records)
				}
			} else {
				_, err = backend.Search("free text")
			}
			if (err == nil) != test.empty {
				t.Fatalf("error = %v, empty = %v", err, test.empty)
			}
		})
	}
}

func TestCrossrefTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = io.WriteString(w, `{"message":{"items":[]}}`)
	}))
	defer server.Close()
	backend := &Backend{BaseURL: server.URL, HTTPClient: &http.Client{Timeout: time.Millisecond}}
	if _, err := backend.Search("test"); err == nil {
		t.Fatal("request ignored client timeout")
	}
}

func TestBibTeXDebugExplainsCrossrefFailures(t *testing.T) {
	for _, test := range []struct {
		name   string
		doi    string
		status int
		body   string
		want   string
	}{
		{"missing DOI", "", 200, "", "stage=lookup failed: selected work has no valid DOI; export was not requested"},
		{"not found", "10.1000/example", 404, "missing", "stage=lookup failed: HTTP 404"},
		{"malformed export", "10.1000/example", 200, "not BibTeX", "stage=parse failed"},
		{"wrong DOI", "10.1000/example", 200, `@article{x,doi={10.1000/wrong}}`, "stage=matching failed: selected DOI="},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if test.doi == "" {
					t.Error("requested an export without a DOI")
				}
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			var trace bytes.Buffer
			defer diagnostics.SetOutput(&trace)()
			backend := &Backend{BaseURL: server.URL, HTTPClient: server.Client()}
			records, _ := backend.BibTeX(bibliography.Work{DOI: test.doi})
			if len(records) != 0 || !strings.Contains(trace.String(), test.want) {
				t.Fatalf("records = %+v, trace = %q", records, trace.String())
			}
		})
	}
}
