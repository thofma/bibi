package util

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

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
