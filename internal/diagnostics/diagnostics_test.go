package diagnostics

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestHTTPTracingPreservesRequestAndBody(t *testing.T) {
	var trace bytes.Buffer
	defer SetOutput(&trace)()
	request, err := http.NewRequest(http.MethodGet, "https://example.org/works?query=local+fields", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer private-header")
	client := &http.Client{Transport: roundTripFunc(func(got *http.Request) (*http.Response, error) {
		if got != request || got.Header.Get("Authorization") != "Bearer private-header" {
			t.Fatal("tracing changed the request")
		}
		return &http.Response{StatusCode: 200, Status: "200 OK", Body: io.NopCloser(strings.NewReader("original body")), Request: got}, nil
	})}
	response, err := Do(client, request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil || string(body) != "original body" {
		t.Fatalf("response body = %q, error = %v", body, err)
	}
	for _, want := range []string{"[debug] HTTP GET https://example.org/works?query=local+fields", `accept="application/json"`, "HTTP response 200 OK in"} {
		if !strings.Contains(trace.String(), want) {
			t.Errorf("trace = %q, want %q", trace.String(), want)
		}
	}
	for _, absent := range []string{"private-header", "original body"} {
		if strings.Contains(trace.String(), absent) {
			t.Errorf("trace unexpectedly includes %q", absent)
		}
	}
}

func TestHTTPTracingPropagatesFailures(t *testing.T) {
	var trace bytes.Buffer
	defer SetOutput(&trace)()
	want := errors.New("connection unavailable")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, want })}
	request, _ := http.NewRequest(http.MethodGet, "https://example.org/", nil)
	if _, err := Do(client, request); !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
	if !strings.Contains(trace.String(), "HTTP failed after") || !strings.Contains(trace.String(), want.Error()) {
		t.Fatalf("trace = %q", trace.String())
	}
}

func TestTracingCanBeDisabledAndRestored(t *testing.T) {
	var trace bytes.Buffer
	defer SetOutput(&trace)()
	restore := SetOutput(nil)
	if Enabled() {
		t.Fatal("tracing remained enabled")
	}
	Printf("hidden")
	Preview("hidden", "response")
	restore()
	Printf("visible")
	if trace.String() != "[debug] visible\n" {
		t.Errorf("trace = %q", trace.String())
	}
}

func TestResponsePreviewIsBoundedAndQuoted(t *testing.T) {
	var trace bytes.Buffer
	defer SetOutput(&trace)()
	Preview("provider", "\n\x1b"+strings.Repeat("x", 3000)+"hidden tail")
	if strings.Contains(trace.String(), "hidden tail") || len(trace.String()) > 2150 || strings.Contains(trace.String(), "\x1b") {
		t.Fatalf("preview not bounded or quoted: %q", trace.String())
	}
	if !strings.Contains(trace.String(), `response="\n\x1b`) || !strings.Contains(trace.String(), "truncated=true") {
		t.Fatalf("preview = %q", trace.String())
	}
}
