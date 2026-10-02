// Package diagnostics provides opt-in tracing for a single CLI invocation.
package diagnostics

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

var output struct {
	sync.Mutex
	writer io.Writer
}

// SetOutput enables tracing to writer and returns a function restoring the
// previous output. Passing nil disables tracing. Callers should defer restoration.
func SetOutput(writer io.Writer) func() {
	output.Lock()
	previous := output.writer
	output.writer = writer
	output.Unlock()
	return func() {
		output.Lock()
		output.writer = previous
		output.Unlock()
	}
}

func Enabled() bool {
	output.Lock()
	defer output.Unlock()
	return output.writer != nil
}

func Printf(format string, args ...any) {
	output.Lock()
	defer output.Unlock()
	if output.writer != nil {
		_, _ = fmt.Fprintf(output.writer, "[debug] "+format+"\n", args...)
	}
}

// Preview quotes a bounded response excerpt so malformed provider responses can
// be diagnosed without flooding the terminal or emitting control characters.
func Preview(label, body string) {
	if !Enabled() {
		return
	}
	const limit = 2048
	truncated := len(body) > limit
	if truncated {
		body = body[:limit]
	}
	Printf("%s response=%q truncated=%t", label, body, truncated)
}

// Do logs request metadata and response timing without changing the request or
// reading its body. Headers other than Accept are not included in the trace.
func Do(client *http.Client, request *http.Request) (*http.Response, error) {
	if !Enabled() {
		return client.Do(request)
	}
	Printf("HTTP %s %s accept=%q", request.Method, request.URL.Redacted(), request.Header.Get("Accept"))
	start := time.Now()
	response, err := client.Do(request)
	elapsed := time.Since(start).Round(time.Millisecond)
	if err != nil {
		Printf("HTTP failed after %s: %v", elapsed, err)
		return response, err
	}
	Printf("HTTP response %s in %s", response.Status, elapsed)
	if response.Request != nil && response.Request.URL.String() != request.URL.String() {
		Printf("HTTP final URL %s", response.Request.URL.Redacted())
	}
	return response, nil
}
