package util

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

const spinnerInterval = 100 * time.Millisecond

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner displays progress while a network request is in flight.
type Spinner struct {
	writer  io.Writer
	message string
	stop    chan struct{}
	done    chan struct{}
	once    sync.Once
}

// StartSpinner starts a terminal-only spinner. It does nothing for redirected
// output so that non-interactive callers receive no control characters.
func StartSpinner(writer io.Writer, message string) *Spinner {
	return newSpinner(writer, message, isTerminal(writer), spinnerInterval)
}

func newSpinner(writer io.Writer, message string, enabled bool, interval time.Duration) *Spinner {
	spinner := &Spinner{writer: writer, message: message}
	if !enabled || writer == nil {
		return spinner
	}

	spinner.stop = make(chan struct{})
	spinner.done = make(chan struct{})
	spinner.render(0)
	go func() {
		defer close(spinner.done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		frame := 1
		for {
			select {
			case <-spinner.stop:
				return
			case <-ticker.C:
				spinner.render(frame)
				frame = (frame + 1) % len(spinnerFrames)
			}
		}
	}()
	return spinner
}

// Stop clears the spinner line. It is safe to call more than once.
func (spinner *Spinner) Stop() {
	if spinner == nil {
		return
	}
	spinner.once.Do(func() {
		if spinner.stop == nil {
			return
		}
		close(spinner.stop)
		<-spinner.done
		_, _ = fmt.Fprint(spinner.writer, "\r\033[2K")
	})
}

func (spinner *Spinner) render(frame int) {
	_, _ = fmt.Fprintf(spinner.writer, "\r%s %s", spinnerFrames[frame], spinner.message)
}

func isTerminal(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
