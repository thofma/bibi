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
	"reflect"
	"strings"
	"testing"

	"github.com/nickng/bibtex"
	"github.com/thofma/bibi/lib/bibliography"
	"github.com/thofma/bibi/lib/doi"
	"github.com/thofma/bibi/util"
)

func getOutputKeys(t *testing.T, output string) []string {
	t.Helper()
	parsed, err := bibtex.Parse(strings.NewReader(output))
	if err != nil {
		t.Fatalf("invalid BibTeX: %v\n%s", err, output)
	}
	var keys []string
	for _, entry := range parsed.Entries {
		keys = append(keys, entry.CiteName)
	}
	return keys
}

func TestGetBatchArgumentsAndImplicitStdin(t *testing.T) {
	for _, test := range []struct {
		name  string
		args  []string
		input string
	}{
		{
			name: "arguments",
			args: []string{"get", "10.1000/FIRST", "https://arxiv.org/pdf/2301.12345v2.pdf", "doi:10.1000/first",
				"https://arxiv.org/abs/2301.12345v2?tracking=1", "2301.12345v1", "2301.12345", "https://doi.org/10.1000/SECOND"},
		},
		{
			name:  "stdin with CRLF blank lines and no final newline",
			args:  []string{"get"},
			input: " \r\n10.1000/FIRST\r\n\thttps://arxiv.org/pdf/2301.12345v2.pdf \r\n\ndoi:10.1000/first\nhttps://arxiv.org/abs/2301.12345v2?tracking=1\n2301.12345v1\n2301.12345\nhttps://doi.org/10.1000/SECOND",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []string
			services := getServices{
				arxiv: func(_ context.Context, id string) (bibliography.Record, error) {
					calls = append(calls, "arxiv:"+id)
					record := getPreprint("10.1000/first")
					record.IDs["arxiv"] = id
					record.Entry.CiteName = fmt.Sprintf("Entry%d", len(calls))
					record.Entry.AddField("eprint", bibtex.NewBibConst(id))
					return record, nil
				},
				doi: func(_ context.Context, id string) (bibliography.Record, error) {
					calls = append(calls, "doi:"+id)
					return searchRecord(fmt.Sprintf("Entry%d", len(calls)), id), nil
				},
			}
			initialized := 0
			root := debugTestRoot(newGetCommand(func() getServices {
				initialized++
				return services
			}, nil))
			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetIn(strings.NewReader(test.input))
			root.SetArgs(test.args)
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			wantCalls := []string{"doi:10.1000/first", "arxiv:2301.12345v2", "arxiv:2301.12345v1", "arxiv:2301.12345", "doi:10.1000/second"}
			if !reflect.DeepEqual(calls, wantCalls) || initialized != 1 || stderr.Len() != 0 {
				t.Fatalf("calls=%v initialized=%d stderr=%q", calls, initialized, stderr.String())
			}
			if got := getOutputKeys(t, stdout.String()); !reflect.DeepEqual(got, []string{"Entry1", "Entry2", "Entry3", "Entry4", "Entry5"}) {
				t.Fatalf("output keys=%v", got)
			}
			parsed, _ := bibtex.Parse(strings.NewReader(stdout.String()))
			for i, want := range []string{"2301.12345v2", "2301.12345v1", "2301.12345"} {
				if got := parsed.Entries[i+1].Fields["eprint"].String(); got != want {
					t.Errorf("eprint=%q, want %q", got, want)
				}
			}
		})
	}
}

type getFailedReader struct {
	err   error
	reads int
}

func (reader *getFailedReader) Read([]byte) (int, error) {
	reader.reads++
	return 0, reader.err
}

func TestGetArgumentsTakePrecedenceOverStdin(t *testing.T) {
	input := &getFailedReader{err: errors.New("stdin must not be read")}
	root := debugTestRoot(newGetCommand(func() getServices {
		return getServices{doi: func(_ context.Context, id string) (bibliography.Record, error) {
			return searchRecord("Argument", id), nil
		}}
	}, nil))
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(io.Discard)
	root.SetIn(input)
	root.SetArgs([]string{"get", "10.1000/argument"})
	if err := root.Execute(); err != nil || input.reads != 0 {
		t.Fatalf("error=%v stdin reads=%d", err, input.reads)
	}
	assertMRBibTeX(t, stdout.String(), "Argument")
}

func TestGetBatchContinuesAfterIndividualFailures(t *testing.T) {
	lookupErr := errors.New("DOI service unavailable")
	var calls []string
	root := debugTestRoot(newGetCommand(func() getServices {
		return getServices{doi: func(_ context.Context, id string) (bibliography.Record, error) {
			calls = append(calls, id)
			if id == "10.1000/unavailable" {
				return bibliography.Record{}, lookupErr
			}
			return searchRecord(strings.TrimPrefix(id, "10.1000/"), id), nil
		}}
	}, nil))
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetIn(strings.NewReader("10.1000/first\n\ninvalid prose\n10.1000/unavailable\n10.1000/last\nDOI:10.1000/UNAVAILABLE\n"))
	root.SetArgs([]string{"get"})
	err := root.Execute()
	var batchErr *getBatchError
	if !errors.Is(err, lookupErr) || !errors.As(err, &batchErr) || len(batchErr.failures) != 2 || batchErr.written != 2 {
		t.Fatalf("error=%v", err)
	}
	if !reflect.DeepEqual(calls, []string{"10.1000/first", "10.1000/unavailable", "10.1000/last"}) {
		t.Fatalf("calls=%v", calls)
	}
	if got := getOutputKeys(t, stdout.String()); !reflect.DeepEqual(got, []string{"first", "last"}) {
		t.Fatalf("output keys=%v", got)
	}
	for _, want := range []string{`stdin line 3 ("invalid prose")`, `stdin line 4 ("10.1000/unavailable")`, "DOI service unavailable", "2 inputs failed; 2 BibTeX entries written"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("missing %q in stderr=%q", want, stderr.String())
		}
	}
	if strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("lookup failure printed usage: %s", stderr.String())
	}
}

func TestGetBatchRecoversAfterMalformedProviderBibTeX(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/10.1000/broken" {
			_, _ = io.WriteString(writer, "@article{broken, title={unfinished")
			return
		}
		_, _ = io.WriteString(writer, `@article{Valid, title={{ABC} and $p$-adic fields}, author={Doe, Jane}, journal={Journal}, year=2020, doi={10.1000/valid}}`)
	}))
	defer server.Close()
	backend := &doi.Backend{BaseURL: server.URL, HTTPClient: server.Client()}
	root := debugTestRoot(newGetCommand(func() getServices {
		return getServices{doi: backend.Lookup}
	}, nil))
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"get", "10.1000/broken", "10.1000/valid"})
	if got := commandExitCode(root); got != 1 {
		t.Fatalf("exit code=%d", got)
	}
	assertMRBibTeX(t, stdout.String(), "Valid")
	if !strings.Contains(stderr.String(), "parse DOI BibTeX") || !strings.Contains(stderr.String(), "1 input failed; 1 BibTeX entry written") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestGetPublishedBatchReportsIncompatibleAndMissingDOIs(t *testing.T) {
	var calls []string
	root := debugTestRoot(newGetCommand(func() getServices {
		return getServices{
			arxiv: func(_ context.Context, id string) (bibliography.Record, error) {
				calls = append(calls, "arxiv:"+id)
				if id == "2301.12345" {
					return getPreprint(""), nil
				}
				return getPreprint("10.1000/published"), nil
			},
			doi: func(_ context.Context, id string) (bibliography.Record, error) {
				calls = append(calls, "doi:"+id)
				return searchRecord("Published", id), nil
			},
		}
	}, nil))
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"get", "10.1000/example", "2301.12345", "2301.12346", "--published"})
	if got := commandExitCode(root); got != 1 {
		t.Fatalf("exit code=%d", got)
	}
	if !reflect.DeepEqual(calls, []string{"arxiv:2301.12345", "arxiv:2301.12346", "doi:10.1000/published"}) {
		t.Fatalf("calls=%v", calls)
	}
	assertMRBibTeX(t, stdout.String(), "Published")
	for _, want := range []string{"--published requires an arXiv", "supplies no publication DOI", "2 inputs failed; 1 BibTeX entry written"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("missing %q in stderr=%q", want, stderr.String())
		}
	}
}

func TestGetStdinReadErrorsAndEmptyInput(t *testing.T) {
	readErr := errors.New("input failed")
	for _, test := range []struct {
		name   string
		input  io.Reader
		want   string
		lookup bool
	}{
		{"empty", strings.NewReader(""), "stdin contained no", false},
		{"blank lines", strings.NewReader(" \r\n\t\n"), "stdin contained no", false},
		{"read failure", &getFailedReader{err: readErr}, "read identifiers from stdin", false},
		{"read failure after an entry", io.MultiReader(strings.NewReader("10.1000/first\n"), &getFailedReader{err: readErr}), "read identifiers from stdin", true},
		{"oversized line", strings.NewReader(strings.Repeat("x", bufioMaxTokenForTest)), "token too long", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			called := false
			root := debugTestRoot(newGetCommand(func() getServices {
				if !test.lookup {
					t.Fatal("empty or unreadable input initialized services")
				}
				return getServices{doi: func(_ context.Context, id string) (bibliography.Record, error) {
					called = true
					return searchRecord("First", id), nil
				}}
			}, nil))
			var stdout bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(io.Discard)
			root.SetIn(test.input)
			root.SetArgs([]string{"get"})
			err := root.Execute()
			if err == nil || !strings.Contains(err.Error(), test.want) || called != test.lookup {
				t.Fatalf("error=%v called=%t", err, called)
			}
			if strings.HasPrefix(test.name, "read failure") && !errors.Is(err, readErr) {
				t.Fatalf("underlying input error lost: %v", err)
			}
			if test.lookup {
				assertMRBibTeX(t, stdout.String(), "First")
			} else if stdout.Len() != 0 {
				t.Fatalf("stdout=%q", stdout.String())
			}
		})
	}
}

const bufioMaxTokenForTest = 64 * 1024

func TestGetRedirectedFileIsNotATerminal(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "identifiers")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := io.WriteString(file, "10.1000/file\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	root := debugTestRoot(newGetCommand(func() getServices {
		return getServices{doi: func(_ context.Context, id string) (bibliography.Record, error) {
			return searchRecord("File", id), nil
		}}
	}, nil))
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetIn(file)
	root.SetArgs([]string{"get"})
	if err := root.Execute(); err != nil || stderr.Len() != 0 {
		t.Fatalf("error=%v stderr=%q", err, stderr.String())
	}
	assertMRBibTeX(t, stdout.String(), "File")
}

func TestGetCitationKeyCollisionsDoNotCorruptBatch(t *testing.T) {
	root := debugTestRoot(newGetCommand(func() getServices {
		return getServices{doi: func(_ context.Context, id string) (bibliography.Record, error) {
			key := "Shared"
			if id == "10.1000/second" {
				key = "SHARED"
			} else if id == "10.1000/third" {
				key = "Third"
			}
			return searchRecord(key, id), nil
		}}
	}, nil))
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"get", "10.1000/first", "10.1000/second", "10.1000/third"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected a citation key collision")
	}
	if got := getOutputKeys(t, stdout.String()); !reflect.DeepEqual(got, []string{"Shared", "Third"}) {
		t.Fatalf("output keys=%v", got)
	}
	if !strings.Contains(stderr.String(), `citation key "SHARED" is already used by doi 10.1000/first`) {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

type getFailedWriter struct{ err error }

func (writer getFailedWriter) Write([]byte) (int, error) { return 0, writer.err }

func TestGetBatchStopsOnBrokenOutput(t *testing.T) {
	writeErr := errors.New("output closed")
	calls := 0
	root := debugTestRoot(newGetCommand(func() getServices {
		return getServices{doi: func(_ context.Context, id string) (bibliography.Record, error) {
			calls++
			return searchRecord("First", id), nil
		}}
	}, nil))
	root.SetOut(getFailedWriter{err: writeErr})
	root.SetErr(io.Discard)
	root.SetArgs([]string{"get", "10.1000/first", "10.1000/second"})
	if err := root.Execute(); !errors.Is(err, writeErr) || calls != 1 {
		t.Fatalf("error=%v calls=%d", err, calls)
	}
}

func TestGetBatchCancellationStopsRemainingLookups(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	root := debugTestRoot(newGetCommand(func() getServices {
		return getServices{doi: func(_ context.Context, id string) (bibliography.Record, error) {
			calls++
			if calls == 2 {
				cancel()
			}
			return searchRecord(fmt.Sprintf("Entry%d", calls), id), nil
		}}
	}, nil))
	root.SetContext(ctx)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"get", "10.1000/first", "10.1000/second", "10.1000/third"})
	if err := root.Execute(); !errors.Is(err, context.Canceled) || calls != 2 {
		t.Fatalf("error=%v calls=%d", err, calls)
	}
	assertMRBibTeX(t, stdout.String(), "Entry1")
}

func TestGetBatchStopsWhenProviderSelectionIsCancelled(t *testing.T) {
	calls := 0
	root := debugTestRoot(newGetCommand(func() getServices {
		return getServices{
			doiMetadata: func(_ context.Context, id string) (bibliography.Work, error) {
				calls++
				return bibliography.Work{DOI: id, Title: "Selected work"}, nil
			},
			bib: map[string]bibliography.Provider{"mr": fakeProvider(func(work bibliography.Work) ([]bibliography.Record, error) {
				return []bibliography.Record{searchRecord("First", work.DOI), searchRecord("Second", work.DOI)}, nil
			})},
		}
	}, func(util.ChooserRequest) (int, error) { return -1, nil }))
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"get", "10.1000/first", "10.1000/second", "--bib", "mr"})
	if err := root.Execute(); !errors.Is(err, errSelectionCancelled) || calls != 1 || stdout.Len() != 0 {
		t.Fatalf("error=%v calls=%d stdout=%q", err, calls, stdout.String())
	}
}
