package zb

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/lib/bibliography"
)

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
	if records[0].Entry.Fields["number"].String() != "7" || records[0].Journals != (bibliography.JournalNames{Full: "Journal", Short: "J."}) {
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
