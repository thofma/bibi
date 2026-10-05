package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/thofma/bibi/internal/httpclient"
	"github.com/thofma/bibi/lib/bibliography"
	"github.com/thofma/bibi/lib/crossref"
	"github.com/thofma/bibi/util"
)

func TestRecoveredRequestsExportOrSaveExactlyOnce(t *testing.T) {
	for _, commandName := range []string{"search", "add"} {
		t.Run(commandName, func(t *testing.T) {
			requests := make(map[string]int)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				requests[req.URL.Path]++
				if requests[req.URL.Path] == 1 {
					w.WriteHeader(503)
					return
				}
				if strings.HasSuffix(req.URL.Path, "/transform") {
					io.WriteString(w, `@article{Recovered,title={Recovered work},author={Doe, Jane},year=1999,journal={Journal},doi={10.1000/recovered}}`)
				} else {
					io.WriteString(w, `{"message":{"DOI":"10.1000/recovered","title":["Recovered work"],"type":"journal-article"}}`)
				}
			}))
			defer server.Close()
			backend := &crossref.Backend{BaseURL: server.URL, HTTPClient: server.Client()}
			services := func() searchServices {
				return searchServices{discovery: map[string]bibliography.Discoverer{"crossref": backend}, bib: map[string]bibliography.Provider{"crossref": backend}}
			}
			choose := func(util.ChooserRequest) (int, error) {
				t.Error("verified match requested confirmation")
				return 0, nil
			}
			var command *cobra.Command
			args := []string{"10.1000/recovered"}
			path := filepath.Join(t.TempDir(), "references.bib")
			if commandName == "search" {
				command = newSearchCommand(services, choose)
			} else {
				command = newAddCommand(services, choose)
				args = append(args, path)
			}
			addDebugFlag(command)
			var stdout, stderr bytes.Buffer
			command.SetOut(&stdout)
			command.SetErr(&stderr)
			command.SetArgs(append(args, "--discovery", "crossref", "--bib", "crossref"))
			if err := command.Execute(); err != nil {
				t.Fatal(err)
			}
			text := stdout.String()
			if commandName == "add" {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				text = string(data)
				if stdout.Len() != 0 {
					t.Fatalf("add wrote data to stdout: %q", stdout.String())
				}
			}
			if strings.Count(text, "@article{") != 1 || !strings.Contains(text, "Recovered") || strings.Contains(text, "retrying") ||
				strings.Count(stderr.String(), "attempt 2/3") != 2 || strings.Contains(stderr.String(), "\x1b") ||
				requests["/works/10.1000/recovered"] != 2 || requests["/works/10.1000/recovered/transform"] != 2 {
				t.Fatalf("requests=%v stdout=%q stderr=%q saved=%q", requests, stdout.String(), stderr.String(), text)
			}
		})
	}
}

func TestRateLimitedAddLeavesFileUnchanged(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(429)
	}))
	defer server.Close()
	backend := &crossref.Backend{BaseURL: server.URL, HTTPClient: server.Client()}
	services := searchServices{discovery: map[string]bibliography.Discoverer{"crossref": backend}, bib: map[string]bibliography.Provider{"crossref": backend}}
	path := filepath.Join(t.TempDir(), "references.bib")
	before := []byte("% Keep this comment\n@article{Existing,title={Old work}}\n")
	if err := os.WriteFile(path, before, 0600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := runAddTest(t, []string{"10.1000/new", path, "--discovery", "crossref", "--bib", "crossref"}, services, nil)
	after, readErr := os.ReadFile(path)
	if err == nil || readErr != nil || stdout != "" || !bytes.Equal(before, after) || !strings.Contains(stderr, "retry after") {
		t.Fatalf("stdout=%q stderr=%q error=%v read=%v file=%q", stdout, stderr, err, readErr, after)
	}
}

type contextualTestProvider struct{ called bool }

func (*contextualTestProvider) BibTeX(bibliography.Work) ([]bibliography.Record, error) {
	return nil, fmt.Errorf("used context-free provider")
}
func (provider *contextualTestProvider) BibTeXContext(ctx context.Context, _ bibliography.Work) ([]bibliography.Record, error) {
	provider.called = true
	return nil, ctx.Err()
}

func TestProviderReceivesCommandCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	command := &cobra.Command{}
	command.SetContext(ctx)
	command.SetErr(io.Discard)
	provider := &contextualTestProvider{}
	_, err := retrieveProviderCitation(command, bibliography.Work{}, "mr", provider, nil)
	if !provider.called || !errors.Is(err, context.Canceled) {
		t.Fatalf("called=%t error=%v", provider.called, err)
	}
}

func TestBatchContinuesAfterRetryExhaustion(t *testing.T) {
	calls := 0
	command := newGetCommand(func() getServices {
		return getServices{doi: func(context.Context, string) (bibliography.Record, error) {
			calls++
			if calls == 1 {
				return bibliography.Record{}, &httpclient.Error{Service: "DOI service", Attempts: 3, Cause: fmt.Errorf("HTTP 503 Service Unavailable")}
			}
			return searchRecord("AfterFailure", "10.1000/second"), nil
		}}
	}, nil)
	addDebugFlag(command)
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	command.SetArgs([]string{"10.1000/first", "10.1000/second"})
	err := command.Execute()
	if err == nil || calls != 2 || strings.Count(stdout.String(), "@article{") != 1 || !strings.Contains(stdout.String(), "AfterFailure") || !strings.Contains(stderr.String(), "3 attempts") {
		t.Fatalf("calls=%d stdout=%q stderr=%q error=%v", calls, stdout.String(), stderr.String(), err)
	}
}
