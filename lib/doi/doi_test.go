package doi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/thofma/bibi/lib/bibliography"
)

const export = `@article{NativeKey,
  author = {Ducas, Léo},
  title = {Galois groups of $GL_2(K)$},
  journal = {Full Journal Name},
  year = 2023,
  month = Dec,
  DOI = {10.1000/Example},
  note = {Keep \LaTeX{} & native fields}
}`

func TestDOILookupFollowsRegistrationAgencyRedirect(t *testing.T) {
	exports, redirects := 0, 0
	agency := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		exports++
		if r.URL.Path != "/registered/10.1000/example" || r.Header.Get("Accept") != "application/x-bibtex" {
			t.Errorf("agency request = %s %v", r.URL, r.Header)
		}
		_, _ = io.WriteString(w, export)
	}))
	defer agency.Close()
	resolver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirects++
		if r.URL.Path != "/10.1000/example" || !strings.Contains(r.Header.Get("User-Agent"), "bibi") {
			t.Errorf("resolver request = %s %v", r.URL, r.Header)
		}
		http.Redirect(w, r, agency.URL+"/registered"+r.URL.Path, http.StatusFound)
	}))
	defer resolver.Close()
	backend := &Backend{BaseURL: resolver.URL, HTTPClient: resolver.Client()}
	record, err := backend.Lookup(context.Background(), "https://doi.org/10.1000/EXAMPLE")
	if err != nil {
		t.Fatal(err)
	}
	if redirects != 1 || exports != 1 || record.DOI != "10.1000/example" || record.Entry.CiteName != "NativeKey" {
		t.Fatalf("redirects=%d exports=%d record=%+v", redirects, exports, record)
	}
	for field, want := range map[string]string{
		"title": "Galois groups of $GL_2(K)$", "author": "Ducas, Léo",
		"note": `Keep \LaTeX{} & native fields`, "DOI": "10.1000/Example", "month": "December",
	} {
		if got := record.Entry.Fields[field].String(); got != want {
			t.Errorf("native field %s = %q, want %q", field, got, want)
		}
	}
	if record.Journals != (bibliography.JournalNames{Full: "Full Journal Name"}) {
		t.Errorf("journals = %+v", record.Journals)
	}
}

func TestDOILookupPreservesReservedSuffixCharacters(t *testing.T) {
	const id = "10.1000/a_b(2);z?value#part"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+id || r.URL.RawQuery != "" || r.URL.Fragment != "" {
			t.Errorf("DOI suffix changed: %s", r.URL)
		}
		_, _ = io.WriteString(w, `@misc{exact,title={Exact work}}`)
	}))
	defer server.Close()
	backend := &Backend{BaseURL: server.URL, HTTPClient: server.Client()}
	record, err := backend.Lookup(context.Background(), id)
	if err != nil || record.DOI != id {
		t.Fatalf("record=%+v error=%v", record, err)
	}
	if record.Entry.Fields["doi"] != nil {
		t.Fatal("resolver verification rewrote the native export")
	}
}

func TestDOIMetadataUsesCSLForExplicitProviderLookup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.citationstyles.csl+json" {
			t.Errorf("requested another provider's BibTeX instead of metadata: %v", r.Header)
		}
		_, _ = io.WriteString(w, `{"DOI":"10.1000/EXAMPLE","title":"A selected article","container-title":"Example Journal","type":"article-journal","author":[{"family":"van der Waerden","given":"B. L."},{"literal":"Example Collaboration"}],"editor":[{"family":"Doe","given":"Jane"}],"issued":{"date-parts":[[2024,5]]}}`)
	}))
	defer server.Close()
	backend := &Backend{BaseURL: server.URL, HTTPClient: server.Client()}
	work, err := backend.Metadata(context.Background(), "10.1000/example")
	if err != nil {
		t.Fatal(err)
	}
	want := bibliography.Work{DOI: "10.1000/example", Title: "A selected article", Venue: "Example Journal", Type: "article-journal", Year: "2024",
		Authors: []string{"van der Waerden, B. L.", "Example Collaboration"}, Editors: []string{"Doe, Jane"}}
	if !reflect.DeepEqual(work, want) {
		t.Fatalf("metadata=%+v, want %+v", work, want)
	}
}

func TestDOIMetadataPreservesEditionAndTranslationNotes(t *testing.T) {
	for _, edition := range []string{`"2"`, `2`, `null`} {
		t.Run(edition, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, `{"title":"Book","type":"book","edition":`+edition+`,"note":"Translation of the original"}`)
			}))
			defer server.Close()
			backend := &Backend{BaseURL: server.URL, HTTPClient: server.Client()}
			work, err := backend.Metadata(context.Background(), "10.1000/book")
			want := "2"
			if edition == "null" {
				want = ""
			}
			if err != nil || work.Edition != want || work.Notes != "Translation of the original" {
				t.Fatalf("metadata=%+v error=%v", work, err)
			}
		})
	}
}

func TestDOILookupFailures(t *testing.T) {
	for _, test := range []struct {
		name, body, want string
		status           int
		metadata         bool
	}{
		{"not found", "missing", "not found", 404, false},
		{"unsupported format", "unsupported", "cannot supply", 406, false},
		{"rate limit", "retry later", "429", 429, false},
		{"not BibTeX", "<html>Article landing page</html>", "expected one", 200, false},
		{"malformed BibTeX", "@article{broken, title=,}", "parse DOI BibTeX", 200, false},
		{"multiple entries", "@misc{a,title={A}} @misc{b,title={B}}", "expected one", 200, false},
		{"different DOI", "@misc{x,doi={10.1000/wrong}}", "different DOI", 200, false},
		{"malformed metadata", "{", "parse DOI metadata", 200, true},
		{"different metadata DOI", `{"DOI":"10.1000/wrong","title":"Wrong"}`, "different DOI", 200, true},
		{"empty metadata", "{}", "no title", 200, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			backend := &Backend{BaseURL: server.URL, HTTPClient: server.Client()}
			var err error
			if test.metadata {
				_, err = backend.Metadata(context.Background(), "10.1000/example")
			} else {
				record, lookupErr := backend.Lookup(context.Background(), "10.1000/example")
				err = lookupErr
				if record.Entry != nil {
					t.Fatal("failed lookup returned BibTeX")
				}
			}
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want %q", err, test.want)
			}
		})
	}
}

func TestDOICancellationAndTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(30 * time.Millisecond)
		_, _ = io.WriteString(w, export)
	}))
	defer server.Close()
	backend := &Backend{BaseURL: server.URL, HTTPClient: &http.Client{Timeout: time.Millisecond}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := backend.Lookup(ctx, "10.1000/example"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error=%v", err)
	}
	if _, err := backend.Lookup(context.Background(), "10.1000/example"); err == nil {
		t.Fatal("request ignored client timeout")
	}
}
