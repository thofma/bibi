package util

import (
	"errors"
	"io"
	"os"
	"runtime"
)

// ErrInteractiveTerminalUnavailable means that a required choice cannot be made.
var ErrInteractiveTerminalUnavailable = errors.New("interactive terminal unavailable")

// Open a separate terminal handle so pickers never consume piped identifiers.
// These are the same devices used by Bubble Tea's WithInputTTY option.
func openChooserTerminal() (io.ReadCloser, error) {
	if runtime.GOOS == "windows" {
		return os.OpenFile("CONIN$", os.O_RDWR, 0)
	}
	return os.Open("/dev/tty")
}
