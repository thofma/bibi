package phd

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nickng/bibtex"
)

const singleMGPFixture = `<!doctype html>
<html><body>
<h2 style="text-align: center; margin-bottom: 0.5ex; margin-top: 1ex">
  Emmy  Noether
</h2>
<div style="line-height: 30px; text-align: center">
  <span style="margin-right: 0.5em">Dr. phil.
    <span style="color: #006633; margin-left: 0.5em">Georg-August-Universit&auml;t G&ouml;ttingen</span> 1907
  </span>
</div>
<div style="text-align: center"><span style="font-style:italic" id="thesisTitle">
  Invariantentheorie
</span></div>
</body></html>`

func useMGPTestAPI(t *testing.T, searchURL, entryURL string, client *http.Client) {
	t.Helper()
	originalSearchURL := mgpSearchURL
	originalEntryURL := mgpEntryURL
	originalClient := mgpHTTPClient
	mgpSearchURL = searchURL
	mgpEntryURL = entryURL
	mgpHTTPClient = client
	t.Cleanup(func() {
		mgpSearchURL = originalSearchURL
		mgpEntryURL = originalEntryURL
		mgpHTTPClient = originalClient
	})
}

func TestMGPResponseParsesSingleHit(t *testing.T) {
	entries, err := MGPResponse(singleMGPFixture)
	if err != nil {
		t.Fatalf("MGPResponse() error = %v", err)
	}
	if got, want := len(entries), 1; got != want {
		t.Fatalf("entry count = %d, want %d", got, want)
	}
	entry := entries[0]
	if got, want := entry.Author, "Emmy Noether"; got != want {
		t.Errorf("author = %q, want %q", got, want)
	}
	if got, want := entry.Title, "Invariantentheorie"; got != want {
		t.Errorf("title = %q, want %q", got, want)
	}
	if got, want := entry.University, "Georg-August-Universität Göttingen"; got != want {
		t.Errorf("university = %q, want %q", got, want)
	}
	if got, want := entry.Year, "1907"; got != want {
		t.Errorf("year = %q, want %q", got, want)
	}
	if entry.BibTeX == nil {
		t.Fatal("single result has no BibTeX entry")
	}
	if _, err := bibtex.Parse(strings.NewReader(entry.BibTeX.PrettyString())); err != nil {
		t.Fatalf("generated BibTeX is invalid: %v\n%s", err, entry.BibTeX.PrettyString())
	}
	if got, want := entry.BibTeX.CiteName, "EmmyNoether1907"; got != want {
		t.Errorf("citation key = %q, want %q", got, want)
	}
	if !strings.Contains(entry.BibTeX.PrettyString(), "Georg-August-Universität Göttingen") {
		t.Errorf("BibTeX = %q, want school field", entry.BibTeX.PrettyString())
	}
}

func TestMGPResponseParsesMultipleHits(t *testing.T) {
	const response = `Your search has found 2 records
<table>
  <tr><td><a href="id.php?id=101">Noether, Emmy</a></td>
  <td>Georg-August-Universit&auml;t G&ouml;ttingen</td><td>1907</td></tr>
  <tr><td><a href="id.php?id=102">Noether, Fritz</a></td>
  <td>Universit&auml;t Erlangen</td><td>1911</td></tr>
</table>`

	entries, err := MGPResponse(response)
	if err != nil {
		t.Fatalf("MGPResponse() error = %v", err)
	}
	if got, want := len(entries), 2; got != want {
		t.Fatalf("entry count = %d, want %d", got, want)
	}
	if got, want := entries[0], (MGPEntry{ID: "101", Author: "Noether, Emmy", University: "Georg-August-Universität Göttingen", Year: "1907"}); got != want {
		t.Errorf("first entry = %#v, want %#v", got, want)
	}
	if got, want := entries[1], (MGPEntry{ID: "102", Author: "Noether, Fritz", University: "Universität Erlangen", Year: "1911"}); got != want {
		t.Errorf("second entry = %#v, want %#v", got, want)
	}
}

func TestMGPResponseReportsMalformedPages(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "missing title",
			body: `<h2>Emmy Noether</h2><span id="thesisTitle"></span>`,
			want: "thesis title",
		},
		{
			name: "missing rows",
			body: "Your search has found 2 records",
			want: "expected 2 records, parsed 0",
		},
		{
			name: "unknown page",
			body: "not an MGP response",
			want: "unrecognized result page",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := MGPResponse(test.body); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("MGPResponse() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestMGPResponseHandlesNoResults(t *testing.T) {
	entries, err := MGPResponse("Your search has found no records")
	if err != nil {
		t.Fatalf("MGPResponse() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("entry count = %d, want 0", len(entries))
	}
}

func TestMGPQueryAndEntryLookup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search":
			if got, want := r.URL.Query().Get("searchTerms"), "Emmy Noether"; got != want {
				t.Errorf("searchTerms = %q, want %q", got, want)
			}
			if got, want := r.URL.Query().Get("Submit"), "Search"; got != want {
				t.Errorf("Submit = %q, want %q", got, want)
			}
			_, _ = io.WriteString(w, singleMGPFixture)
		case "/entry":
			if got, want := r.URL.Query().Get("id"), "42"; got != want {
				t.Errorf("id = %q, want %q", got, want)
			}
			_, _ = io.WriteString(w, singleMGPFixture)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	useMGPTestAPI(t, server.URL+"/search", server.URL+"/entry", server.Client())

	entries, err := MGPQueryAndResponse(" Emmy Noether ")
	if err != nil {
		t.Fatalf("MGPQueryAndResponse() error = %v", err)
	}
	if got, want := len(entries), 1; got != want || entries[0].BibTeX == nil {
		t.Fatalf("search entries = %#v, want one result with BibTeX", entries)
	}

	bib, err := MGPEntryGetBibtex(MGPEntry{ID: "42"})
	if err != nil {
		t.Fatalf("MGPEntryGetBibtex() error = %v", err)
	}
	if got, want := bib.CiteName, "EmmyNoether1907"; got != want {
		t.Errorf("citation key = %q, want %q", got, want)
	}
}

func TestMGPHTTPAndEntryErrors(t *testing.T) {
	t.Run("empty search", func(t *testing.T) {
		if _, err := MGPQuery("  "); err == nil || !strings.Contains(err.Error(), "search terms") {
			t.Fatalf("MGPQuery() error = %v, want empty-search error", err)
		}
	})

	t.Run("transport failure", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network unavailable")
		})}
		useMGPTestAPI(t, "https://example.test/search", "https://example.test/entry", client)

		if _, err := MGPQuery("Noether"); err == nil || !strings.Contains(err.Error(), "request MGP") {
			t.Fatalf("MGPQuery() error = %v, want transport error", err)
		}
	})

	t.Run("HTTP status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		}))
		defer server.Close()
		useMGPTestAPI(t, server.URL, server.URL, server.Client())

		if _, err := MGPQuery("Noether"); err == nil || !strings.Contains(err.Error(), "503 Service Unavailable") {
			t.Fatalf("MGPQuery() error = %v, want HTTP-status error", err)
		}
	})

	t.Run("missing result id", func(t *testing.T) {
		if _, err := MGPEntryGetBibtex(MGPEntry{}); err == nil || !strings.Contains(err.Error(), "has no id") {
			t.Fatalf("MGPEntryGetBibtex() error = %v, want missing-id error", err)
		}
	})
}

func TestBibtexEncodeTitle(t *testing.T) {
	if got, want := BibtexEncodeTitle("An ABC theorem"), "{A}n {ABC} theorem"; got != want {
		t.Errorf("BibtexEncodeTitle() = %q, want %q", got, want)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
