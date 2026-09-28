package mr

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func useMRTestAPI(t *testing.T, baseURL string, client *http.Client) {
	t.Helper()
	originalBaseURL := mrAPIBaseURL
	originalClient := mrHTTPClient
	mrAPIBaseURL = baseURL
	mrHTTPClient = client
	t.Cleanup(func() {
		mrAPIBaseURL = originalBaseURL
		mrHTTPClient = originalClient
	})
}

func TestMRQueryAYTParsesResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Method, http.MethodGet; got != want {
			t.Errorf("method = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("author"), "Serre, Jean-Pierre"; got != want {
			t.Errorf("author = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("title"), "A course in arithmetic"; got != want {
			t.Errorf("title = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("year"), "1973"; got != want {
			t.Errorf("year = %q, want %q", got, want)
		}
		_, _ = io.WriteString(w, mrLookupJSON(t, validMRBibTeX()))
	}))
	defer server.Close()
	useMRTestAPI(t, server.URL, server.Client())

	entries, err := MRQueryAYT("Serre,Jean-Pierre", "1973", "A course in arithmetic")
	if err != nil {
		t.Fatalf("MRQueryAYT() error = %v", err)
	}
	if got, want := len(entries), 1; got != want {
		t.Fatalf("entry count = %d, want %d", got, want)
	}
	entry := entries[0]
	if entry.BibTeX == nil || entry.BibTeX.CiteName != "MR1" {
		t.Fatalf("BibTeX = %#v, want MR1 entry", entry.BibTeX)
	}
	if got, want := entry.Authors, []string{"Serre, Jean-Pierre"}; len(got) != len(want) || got[0] != want[0] {
		t.Errorf("authors = %v, want %v", got, want)
	}
	if got, want := entry.Title, "A course in arithmetic"; got != want {
		t.Errorf("title = %q, want %q", got, want)
	}
	if entry.Doi == nil || *entry.Doi != "10.1000/example" {
		t.Errorf("DOI = %v, want 10.1000/example", entry.Doi)
	}
}

func TestMRQueryAYTReturnsEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, mrLookupJSON(t))
	}))
	defer server.Close()
	useMRTestAPI(t, server.URL, server.Client())

	entries, err := MRQueryAYT("Serre", "", "")
	if err != nil {
		t.Fatalf("MRQueryAYT() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("entry count = %d, want 0", len(entries))
	}
}

func TestMRQueryAYTParsesMultipleResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, mrLookupJSON(t,
			validMRBibTeX(),
			`@book{MR2,
  author = {Noether, Emmy},
  title = {Collected papers},
  year = {1983},
}`,
		))
	}))
	defer server.Close()
	useMRTestAPI(t, server.URL, server.Client())

	entries, err := MRQueryAYT("Noether", "", "")
	if err != nil {
		t.Fatalf("MRQueryAYT() error = %v", err)
	}
	if got, want := len(entries), 2; got != want {
		t.Fatalf("entry count = %d, want %d", got, want)
	}
	if entries[0].BibTeX == nil || entries[0].BibTeX.CiteName != "MR1" {
		t.Errorf("first result = %#v, want MR1", entries[0].BibTeX)
	}
	if entries[1].BibTeX == nil || entries[1].BibTeX.CiteName != "MR2" {
		t.Errorf("second result = %#v, want MR2", entries[1].BibTeX)
	}
	if got, want := entries[1].Authors, []string{"Noether, Emmy"}; len(got) != len(want) || got[0] != want[0] {
		t.Errorf("second-result authors = %v, want %v", got, want)
	}
}

func TestMRQueryAYTReportsHTTPAndJSONErrors(t *testing.T) {
	t.Run("HTTP status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
		}))
		defer server.Close()
		useMRTestAPI(t, server.URL, server.Client())

		_, err := MRQueryAYT("Serre", "", "")
		if err == nil || !strings.Contains(err.Error(), "503 Service Unavailable") {
			t.Fatalf("MRQueryAYT() error = %v, want HTTP status", err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, `{`)
		}))
		defer server.Close()
		useMRTestAPI(t, server.URL, server.Client())

		_, err := MRQueryAYT("Serre", "", "")
		if err == nil || !strings.Contains(err.Error(), "parse MR Lookup response") {
			t.Fatalf("MRQueryAYT() error = %v, want JSON parse context", err)
		}
	})
}

func TestMRQueryAYTRejectsInvalidResults(t *testing.T) {
	t.Run("missing BibTeX", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, mrLookupJSON(t, ""))
		}))
		defer server.Close()
		useMRTestAPI(t, server.URL, server.Client())

		_, err := MRQueryAYT("Serre", "", "")
		if err == nil || !strings.Contains(err.Error(), "has no BibTeX") {
			t.Fatalf("MRQueryAYT() error = %v, want missing BibTeX error", err)
		}
	})

	t.Run("invalid BibTeX", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, mrLookupJSON(t, "not a BibTeX entry"))
		}))
		defer server.Close()
		useMRTestAPI(t, server.URL, server.Client())

		_, err := MRQueryAYT("Serre", "", "")
		if err == nil || !strings.Contains(err.Error(), "MR BibTeX result 1") {
			t.Fatalf("MRQueryAYT() error = %v, want invalid BibTeX error", err)
		}
	})
}

func TestExtractAuthorsFromBibtexHandlesMissingAuthor(t *testing.T) {
	if authors := ExtractAuthorsFromBibtex(nil); len(authors) != 0 {
		t.Errorf("authors = %v, want none", authors)
	}
}

func TestFixNameRejectsTooManyParts(t *testing.T) {
	_, err := fixName("Serre, Jean-Pierre, Jr.")
	if err == nil {
		t.Fatal("fixName() error = nil, want malformed-name error")
	}
}

func mrLookupJSON(t *testing.T, entries ...string) string {
	t.Helper()
	payload := struct {
		All struct {
			Results []lookupResult `json:"results"`
		} `json:"all"`
	}{}
	for _, entry := range entries {
		payload.All.Results = append(payload.All.Results, lookupResult{BibTeXFormat: entry})
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal MR lookup response: %v", err)
	}
	return string(encoded)
}

func validMRBibTeX() string {
	return `@article{MR1,
  author = {Serre, Jean-Pierre},
  title = {A course in arithmetic},
  year = {1973},
  doi = {10.1000/example},
}`
}
