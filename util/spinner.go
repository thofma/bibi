package util

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/thofma/bibi/internal/diagnostics"
	"github.com/thofma/bibi/internal/httpclient"
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
	mu      sync.Mutex
	stopped bool
}

// StartSpinner starts a terminal-only spinner. It does nothing for redirected
// output so that non-interactive callers receive no control characters.
func StartSpinner(writer io.Writer, message string) *Spinner {
	return newSpinner(writer, message, isTerminal(writer) && !diagnostics.Enabled(), spinnerInterval)
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
		spinner.mu.Lock()
		spinner.stopped = true
		spinner.mu.Unlock()
		if spinner.stop == nil {
			return
		}
		close(spinner.stop)
		<-spinner.done
		_, _ = fmt.Fprint(spinner.writer, "\r\033[2K")
	})
}

// Context directs retry progress to this spinner, or plain stderr when the
// spinner is disabled. Debug tracing already reports each retry separately.
func (spinner *Spinner) Context(ctx context.Context) context.Context {
	return httpclient.WithObserver(ctx, func(event httpclient.Event) {
		if diagnostics.Enabled() {
			return
		}
		spinner.mu.Lock()
		defer spinner.mu.Unlock()
		if spinner.stopped {
			return
		}
		if spinner.stop != nil {
			spinner.message = event.String()
		} else if spinner.writer != nil {
			_, _ = fmt.Fprintln(spinner.writer, event.String())
		}
	})
}

func (spinner *Spinner) render(frame int) {
	spinner.mu.Lock()
	defer spinner.mu.Unlock()
	_, _ = fmt.Fprintf(spinner.writer, "\r\033[2K%s %s", spinnerFrames[frame], spinner.message)
}

func isTerminal(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
