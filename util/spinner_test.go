package util

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/internal/httpclient"
)

func TestStartSpinnerSkipsDebugOutput(t *testing.T) {
	file, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	defer diagnostics.SetOutput(file)()
	spinner := StartSpinner(file, "Searching")
	defer spinner.Stop()
	if spinner.stop != nil {
		t.Fatal("debug mode started an animated spinner")
	}
}

func TestStartSpinnerSkipsRedirectedOutput(t *testing.T) {
	var output bytes.Buffer
	spinner := StartSpinner(&output, "Searching")
	spinner.Stop()
	if output.Len() != 0 {
		t.Errorf("redirected output = %q, want empty", output.String())
	}
}

func TestSpinnerRendersAndClearsLine(t *testing.T) {
	var output bytes.Buffer
	spinner := newSpinner(&output, "Searching", true, time.Hour)
	spinner.Stop()
	spinner.Stop()

	if got := output.String(); !strings.Contains(got, "⠋ Searching") {
		t.Errorf("spinner output = %q, want initial frame", got)
	}
	if got := output.String(); !strings.HasSuffix(got, "\r\033[2K") {
		t.Errorf("spinner output = %q, want cleared line", got)
	}
}

func TestIsTerminalRejectsNonFileWriters(t *testing.T) {
	if isTerminal(&bytes.Buffer{}) {
		t.Error("isTerminal() = true for bytes.Buffer")
	}
}

func TestSpinnerReportsRecoveredRequestWithoutDuplicateDebugNotice(t *testing.T) {
	for _, mode := range []string{"animated", "redirected", "debug"} {
		t.Run(mode, func(t *testing.T) {
			var output bytes.Buffer
			if mode == "debug" {
				defer diagnostics.SetOutput(&output)()
			}
			spinner := newSpinner(&output, "Searching", mode == "animated", 10*time.Millisecond)
			defer spinner.Stop()
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				if calls == 1 {
					w.WriteHeader(503)
					return
				}
				io.WriteString(w, "recovered")
			}))
			defer server.Close()
			req, err := http.NewRequestWithContext(spinner.Context(context.Background()), http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			_, body, err := httpclient.Do(server.Client(), req, "Crossref")
			spinner.Stop()
			text := output.String()
			if err != nil || string(body) != "recovered" || calls != 2 || !strings.Contains(text, "attempt 2/3") {
				t.Fatalf("calls=%d body=%q error=%v output=%q", calls, body, err, text)
			}
			if mode != "animated" && (strings.Count(text, "attempt 2/3") != 1 || strings.Contains(text, "\x1b")) {
				t.Fatalf("duplicated notice or terminal escapes: %q", text)
			}
		})
	}
}
