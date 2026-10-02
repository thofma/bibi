package mr

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

func TestProviderUsesSelectedMetadataAndPreservesMRBibTeX(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("author") != "Serre" || r.URL.Query().Get("title") != "A course in arithmetic" || r.URL.Query().Get("year") != "1973" {
			t.Errorf("request = %s", r.URL.String())
		}
		_, _ = io.WriteString(w, mrLookupJSON(t, validMRBibTeX()))
	}))
	defer server.Close()
	useMRTestAPI(t, server.URL, server.Client())
	records, err := (Provider{}).BibTeX(bibliography.Work{
		Title: "A course in arithmetic", Authors: []string{"Serre, Jean-Pierre"}, Year: "1973",
	})
	if err != nil || len(records) != 1 {
		t.Fatalf("records = %+v, error = %v", records, err)
	}
	if records[0].Entry.CiteName != "MR1" || records[0].DOI != "10.1000/example" || records[0].IDs["mr"] != "MR1" {
		t.Fatalf("record = %+v", records[0])
	}
	if got := records[0].Entry.Fields["author"].String(); got != "Serre, Jean-Pierre" {
		t.Errorf("exported author = %q, want the complete MR author name", got)
	}
}

func TestProviderDebugExplainsMRFailures(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want string
	}{
		{"no matches", `{"all":{"results":[]},"notice":"No matches for this query"}`, "bib mr lookup found no records for the author/title/year query"},
		{"malformed response", `<html>Service unavailable</html>`, "MR stage=response-parse failed"},
		{"missing export", `{"all":{"results":[{"bibTexFormat":""}]}}`, "MR stage=export failed: API result 1 has no bibTexFormat"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			useMRTestAPI(t, server.URL, server.Client())
			var trace bytes.Buffer
			defer diagnostics.SetOutput(&trace)()
			records, _ := (Provider{}).BibTeX(bibliography.Work{Title: "Work", Authors: []string{"Doe, Jane"}, Year: "1999", DOI: "10.1000/example"})
			if len(records) != 0 {
				t.Fatalf("unexpected records: %+v", records)
			}
			for _, want := range []string{test.want, `author="Doe" title="Work" year="1999"`, `author from discovery="Doe, Jane" lookup_author="Doe"`, "it is not sent to MR Lookup", "response="} {
				if !strings.Contains(trace.String(), want) {
					t.Errorf("trace = %q, want %q", trace.String(), want)
				}
			}
		})
	}
}

func TestProviderQueriesKnownFamilyNameOnly(t *testing.T) {
	for _, test := range []struct {
		name string
		want string
	}{
		{"Fesenko, I. B.", "Fesenko"},
		{"  Fesenko, I. B.  ", "Fesenko"},
		{"van der Waerden, B. L.", "van der Waerden"},
		{"de la Vallée Poussin, Charles", "de la Vallée Poussin"},
		{"Serre, Jean-Pierre", "Serre"},
		{"Fesenko", "Fesenko"},
		{"Jean-Pierre Serre", "Jean-Pierre Serre"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.URL.Query().Get("author"); got != test.want {
					t.Errorf("MR author = %q, want %q", got, test.want)
				}
				if r.URL.Query().Get("title") != "Local fields and their extensions" || r.URL.Query().Get("year") != "2002" {
					t.Errorf("MR title/year changed: %s", r.URL.String())
				}
				_, _ = io.WriteString(w, mrLookupJSON(t, validMRBibTeX()))
			}))
			defer server.Close()
			useMRTestAPI(t, server.URL, server.Client())
			_, err := (Provider{}).BibTeX(bibliography.Work{Title: "Local fields and their extensions", Authors: []string{test.name, "Vostokov, S. V."}, Year: "2002"})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
