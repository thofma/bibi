package arxiv

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/thofma/bibi/lib/bibliography"
)

func atomEntry(id, doi string) string {
	return fmt.Sprintf(`<feed xmlns="http://www.w3.org/2005/Atom" xmlns:arxiv="http://arxiv.org/schemas/atom">
  <entry>
    <id>http://arxiv.org/abs/%s</id>
    <title>
      Galois groups of \(GL_2(K)\) &amp; 100%% results
    </title>
    <published>2023-01-20T12:30:00Z</published>
    <updated>2024-03-10T12:30:00Z</updated>
    <author><name>Per Brinch Hansen</name></author>
    <author><name>Léo Ducas</name></author>
    <arxiv:primary_category term="math.NT"/>
    <arxiv:doi>%s</arxiv:doi>
    <arxiv:journal_ref>Example Journal 12 (2024), 1–9</arxiv:journal_ref>
  </entry>
</feed>`, id, doi)
}

func TestExactLookupPreservesRequestedVersionAndTeX(t *testing.T) {
	for _, test := range []struct {
		query, id, returned string
	}{
		{"https://arxiv.org/pdf/2301.12345v2.pdf", "2301.12345v2", "2301.12345v2"},
		{"2301.12345", "2301.12345", "2301.12345v3"},
		{"math/0303109v1", "math/0303109v1", "math/0303109v1"},
	} {
		t.Run(test.query, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if q := r.URL.Query(); q.Get("id_list") != test.id || q.Has("search_query") {
					t.Errorf("not an exact lookup: %s", r.URL)
				}
				if r.Header.Get("Accept") != "application/atom+xml" || !strings.Contains(r.Header.Get("User-Agent"), "bibi") {
					t.Errorf("headers = %v", r.Header)
				}
				_, _ = io.WriteString(w, atomEntry(test.returned, "10.1000/published"))
			}))
			defer server.Close()
			backend := &Backend{BaseURL: server.URL, HTTPClient: server.Client()}
			record, err := backend.Lookup(context.Background(), test.query)
			if err != nil {
				t.Fatal(err)
			}
			if requests != 1 || record.Type != "preprint" || record.Year != "2023" || record.IDs["arxiv"] != test.id || record.DOI != "10.1000/published" {
				t.Fatalf("requests=%d work=%+v", requests, record.Work)
			}
			if record.Entry.Type != "misc" || len(bibliography.MissingFields(record.Entry)) != 0 {
				t.Fatalf("unusable preprint: %+v", record.Entry)
			}
			for name, want := range map[string]string{
				"title":  `{Galois groups of \(GL_2(K)\) \& 100\% results}`,
				"author": "Per Brinch Hansen and Léo Ducas", "year": "2023",
				"eprint": test.id, "archiveprefix": "arXiv", "primaryclass": "math.NT",
				"url":          "https://arxiv.org/abs/" + test.id,
				"howpublished": "arXiv preprint arXiv:" + test.id,
			} {
				if field := record.Entry.Fields[name]; field == nil || field.String() != want {
					t.Errorf("field %s = %v, want %q", name, field, want)
				}
			}
			if record.Entry.Fields["doi"] != nil || record.Entry.Fields["journal"] != nil {
				t.Fatal("publication metadata leaked into the preprint entry")
			}
		})
	}
}

func TestArXivLookupFailures(t *testing.T) {
	const empty = `<feed xmlns="http://www.w3.org/2005/Atom"/>`
	for _, test := range []struct {
		name, query, body, want string
		status                  int
	}{
		{"missing", "2301.12345", empty, "not found", 200},
		{"HTTP missing", "2301.12345", "missing", "not found", 404},
		{"rate limit", "2301.12345", "retry later", "429", 429},
		{"malformed", "2301.12345", "<feed", "parse arXiv", 200},
		{"wrong ID", "2301.12345", atomEntry("2301.99999v1", ""), "different identifier", 200},
		{"wrong version", "2301.12345v2", atomEntry("2301.12345v3", ""), "different identifier", 200},
		{"bad date", "2301.12345", strings.ReplaceAll(atomEntry("2301.12345v1", ""), "2023-01-20T12:30:00Z", "unknown"), "publication date", 200},
		{"no authors", "2301.12345", strings.ReplaceAll(strings.ReplaceAll(atomEntry("2301.12345v1", ""), "Per Brinch Hansen", ""), "Léo Ducas", ""), "missing a title or authors", 200},
		{"API error", "2301.12345", `<feed xmlns="http://www.w3.org/2005/Atom"><entry><id>http://arxiv.org/api/errors#bad_id</id><summary>Identifier rejected</summary></entry></feed>`, "Identifier rejected", 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			backend := &Backend{BaseURL: server.URL, HTTPClient: server.Client()}
			record, err := backend.Lookup(context.Background(), test.query)
			if err == nil || !strings.Contains(err.Error(), test.want) || record.Entry != nil {
				t.Fatalf("record=%+v error=%v; want %q", record, err, test.want)
			}
		})
	}
}

func TestInvalidPublicationDOIDoesNotPreventPreprintRetrieval(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, atomEntry("2301.12345v1", "not a DOI"))
	}))
	defer server.Close()
	backend := &Backend{BaseURL: server.URL, HTTPClient: server.Client()}
	record, err := backend.Lookup(context.Background(), "2301.12345")
	if err != nil || record.Entry == nil || record.DOI != "" {
		t.Fatalf("record=%+v error=%v", record, err)
	}
}

func TestArXivLookupCancellationAndTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(30 * time.Millisecond)
		_, _ = io.WriteString(w, atomEntry("2301.12345v1", ""))
	}))
	defer server.Close()
	backend := &Backend{BaseURL: server.URL, HTTPClient: &http.Client{Timeout: time.Millisecond}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := backend.Lookup(ctx, "2301.12345"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error=%v", err)
	}
	if _, err := backend.Lookup(context.Background(), "2301.12345"); err == nil {
		t.Fatal("request ignored client timeout")
	}
}
