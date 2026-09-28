package zb

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func useZBTestAPI(t *testing.T, baseURL string, client *http.Client) {
	t.Helper()
	originalBaseURL := zbAPIBaseURL
	originalClient := zbHTTPClient
	zbAPIBaseURL = baseURL
	zbHTTPClient = client
	t.Cleanup(func() {
		zbAPIBaseURL = originalBaseURL
		zbHTTPClient = originalClient
	})
}

func TestGetZBResponseAnythingBuildsRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/v1/document/_search"; got != want {
			t.Errorf("request path = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("search_string"), "Tommy Hofmann"; got != want {
			t.Errorf("search_string = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("Accept"), "application/json"; got != want {
			t.Errorf("Accept = %q, want %q", got, want)
		}
		_, _ = io.WriteString(w, `{"result":[]}`)
	}))
	defer server.Close()

	useZBTestAPI(t, server.URL+"/v1/document", server.Client())
	body, err := getZBResponseAnything("Tommy Hofmann")
	if err != nil {
		t.Fatalf("getZBResponseAnything() error = %v", err)
	}
	if got, want := body, `{"result":[]}`; got != want {
		t.Errorf("getZBResponseAnything() = %q, want %q", got, want)
	}
}

func TestGetZBResponseAnythingReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	useZBTestAPI(t, server.URL, server.Client())
	_, err := getZBResponseAnything("test")
	if err == nil {
		t.Fatal("getZBResponseAnything() error = nil, want HTTP status error")
	}
	if !strings.Contains(err.Error(), "503 Service Unavailable") || !strings.Contains(err.Error(), "temporarily unavailable") {
		t.Errorf("getZBResponseAnything() error = %q, want status and response body", err)
	}
}

func TestGetZBResponseAnythingHonorsClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = io.WriteString(w, `{"result":[]}`)
	}))
	defer server.Close()

	useZBTestAPI(t, server.URL, &http.Client{Timeout: 10 * time.Millisecond})
	_, err := getZBResponseAnything("test")
	if err == nil {
		t.Fatal("getZBResponseAnything() error = nil, want timeout error")
	}
	if !strings.Contains(err.Error(), "request zbMath") {
		t.Errorf("getZBResponseAnything() error = %q, want request context", err)
	}
}

func TestZBAnythingParsesTypedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{
			"result": [{
				"contributors": {"authors": [{"name": "Hofmann, Tommy"}]},
				"links": [{"type": "doi", "identifier": "10.1000/example"}],
				"title": {"title": "A typed response"},
				"year": "2026"
			}]
		}`)
	}))
	defer server.Close()

	useZBTestAPI(t, server.URL, server.Client())
	entries, err := ZBAnything("typed response")
	if err != nil {
		t.Fatalf("ZBAnything() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ZBAnything() returned %d entries, want 1", len(entries))
	}
	entry := entries[0]
	if entry.Title != "A typed response" || entry.Year != "2026" {
		t.Errorf("ZBAnything() entry = %+v, want parsed title and year", entry)
	}
	if len(entry.Authors) != 1 || entry.Authors[0] != "Hofmann, Tommy" {
		t.Errorf("ZBAnything() authors = %v, want parsed author", entry.Authors)
	}
	if entry.Doi == nil || *entry.Doi != "10.1000/example" {
		t.Errorf("ZBAnything() DOI = %v, want parsed DOI", entry.Doi)
	}
}

func TestParseZBResponseHandlesMissingResult(t *testing.T) {
	_, _, _, _, err := parseZBResponse(`{"result":[]}`)
	if err == nil {
		t.Fatal("parseZBResponse() error = nil, want missing-result error")
	}
}
