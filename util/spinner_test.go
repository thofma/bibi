package util

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/thofma/bibi/internal/diagnostics"
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
