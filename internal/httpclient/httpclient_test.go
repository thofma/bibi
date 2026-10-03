package httpclient

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/thofma/bibi/internal/diagnostics"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

type fakeClock struct {
	sync.Mutex
	current time.Time
	waits   []time.Duration
}

func testContext() (context.Context, *session, *fakeClock) {
	s := newSession()
	clock := &fakeClock{current: time.Now().UTC()}
	s.now = func() time.Time { clock.Lock(); defer clock.Unlock(); return clock.current }
	s.wait = func(ctx context.Context, delay time.Duration) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		clock.Lock()
		defer clock.Unlock()
		clock.waits = append(clock.waits, delay)
		clock.current = clock.current.Add(delay)
		return nil
	}
	s.jitter = func() time.Duration { return 0 }
	return context.WithValue(context.Background(), sessionKey{}, s), s, clock
}
func request(t *testing.T, ctx context.Context, endpoint string) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}
func response(body string) *http.Response {
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

type temporaryError struct{}

func (temporaryError) Error() string   { return "temporarily unavailable" }
func (temporaryError) Timeout() bool   { return false }
func (temporaryError) Temporary() bool { return true }

func TestStatusPolicyAndRecovery(t *testing.T) {
	for _, code := range []int{408, 429, 500, 502, 503, 504, 400, 401, 403, 404, 406, 501, 505} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			ctx, _, clock := testContext()
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls++
				if req.Header.Get("Accept") != "application/json" {
					t.Error("lost Accept header")
				}
				if calls <= 2 {
					w.WriteHeader(code)
					io.WriteString(w, "provider response")
					return
				}
				io.WriteString(w, "recovered")
			}))
			defer server.Close()
			req := request(t, ctx, server.URL)
			req.Header.Set("Accept", "application/json")
			resp, body, err := Do(server.Client(), req, "Crossref")
			if err != nil {
				t.Fatal(err)
			}
			if retryStatus(code) {
				if calls != 3 || resp.StatusCode != 200 || string(body) != "recovered" || len(clock.waits) != 2 || clock.waits[0] != time.Second || clock.waits[1] != 2*time.Second {
					t.Fatalf("recovery: calls=%d status=%d body=%q waits=%v", calls, resp.StatusCode, body, clock.waits)
				}
			} else if calls != 1 || resp.StatusCode != code || string(body) != "provider response" || len(clock.waits) != 0 {
				t.Fatalf("permanent response changed: calls=%d status=%d body=%q waits=%v", calls, resp.StatusCode, body, clock.waits)
			}
		})
	}
}

func TestRetryAfterAndPersistentCooldown(t *testing.T) {
	for _, format := range []string{"seconds", "date"} {
		t.Run(format, func(t *testing.T) {
			ctx, s, clock := testContext()
			calls := 0
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				resp := response("busy")
				resp.StatusCode = 429
				value := "60"
				if format == "date" {
					value = s.now().Add(time.Minute).Format(http.TimeFormat)
				}
				resp.Header.Set("Retry-After", value)
				return resp, nil
			})}
			_, _, err := Do(client, request(t, ctx, "https://example.org/first"), "Crossref")
			var failed *Error
			if !errors.As(err, &failed) || failed.Attempts != 1 || failed.RetryAt.IsZero() || calls != 1 || len(clock.waits) != 0 {
				t.Fatalf("long wait: calls=%d waits=%v error=%v", calls, clock.waits, err)
			}
			_, _, err = Do(client, request(t, ctx, "https://example.org/next"), "Crossref")
			if !errors.As(err, &failed) || failed.Attempts != 0 || calls != 1 {
				t.Fatalf("cooldown bypassed: calls=%d error=%v", calls, err)
			}
			clock.Lock()
			clock.current = clock.current.Add(time.Minute)
			clock.Unlock()
			client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return response("ok"), nil })
			_, body, err := Do(client, request(t, ctx, "https://example.org/next"), "Crossref")
			if err != nil || string(body) != "ok" || calls != 2 {
				t.Fatalf("expired cooldown: calls=%d body=%q error=%v", calls, body, err)
			}
		})
	}
	ctx, _, clock := testContext()
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		resp := response("ok")
		if calls == 1 {
			resp.StatusCode = 503
			resp.Header.Set("Retry-After", "7")
		}
		return resp, nil
	})}
	_, _, err := Do(client, request(t, ctx, "https://example.org"), "Crossref")
	if err != nil || calls != 2 || len(clock.waits) != 1 || clock.waits[0] != 7*time.Second {
		t.Fatalf("short Retry-After: calls=%d waits=%v error=%v", calls, clock.waits, err)
	}
}

func TestTransportFailuresAndRetryLimit(t *testing.T) {
	for _, test := range []struct {
		name  string
		cause error
		retry bool
	}{
		{"EOF", io.EOF, true}, {"reset", syscall.ECONNRESET, true}, {"timeout", context.DeadlineExceeded, true},
		{"temporary connection", temporaryError{}, true},
		{"temporary DNS", &net.DNSError{IsTemporary: true}, true}, {"permanent DNS", &net.DNSError{IsNotFound: true}, false},
		{"certificate", x509.UnknownAuthorityError{}, false}, {"cancellation", context.Canceled, false}, {"invalid transport", errors.New("invalid URL"), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, _, clock := testContext()
			calls := 0
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, test.cause })}
			_, body, err := Do(client, request(t, ctx, "https://example.org"), "zbMATH Open")
			want := 1
			if test.retry {
				want = 3
			}
			var failed *Error
			if !errors.As(err, &failed) || !errors.Is(err, test.cause) || calls != want || failed.Attempts != want || body != nil || len(clock.waits) != want-1 {
				t.Fatalf("calls=%d waits=%v body=%q error=%v", calls, clock.waits, body, err)
			}
		})
	}
}

func TestNonGETRequestsAreNotReplayed(t *testing.T) {
	ctx, _, clock := testContext()
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		resp := response("busy")
		resp.StatusCode = 503
		return resp, nil
	})}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://example.org", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = Do(client, req, "Crossref")
	if err == nil || calls != 1 || len(clock.waits) != 0 {
		t.Fatalf("non-GET replayed: calls=%d waits=%v error=%v", calls, clock.waits, err)
	}
}

type brokenBody struct{ closed bool }

func (*brokenBody) Read(buffer []byte) (int, error) {
	return copy(buffer, "partial"), io.ErrUnexpectedEOF
}
func (body *brokenBody) Close() error { body.closed = true; return nil }

func TestInterruptedBodyIsDiscardedAndClosed(t *testing.T) {
	ctx, _, _ := testContext()
	broken := &brokenBody{}
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		resp := response("complete")
		if calls == 1 {
			resp.Body = broken
		}
		return resp, nil
	})}
	_, body, err := Do(client, request(t, ctx, "https://example.org"), "MR Lookup")
	if err != nil || calls != 2 || string(body) != "complete" || !broken.closed {
		t.Fatalf("body=%q calls=%d closed=%t error=%v", body, calls, broken.closed, err)
	}
}

func TestCancellationDuringBackoffAndRequest(t *testing.T) {
	for _, during := range []string{"backoff", "request"} {
		t.Run(during, func(t *testing.T) {
			ctx, _, _ := testContext()
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			calls := 0
			ctx = WithObserver(ctx, func(Event) { cancel() })
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if during == "request" {
					cancel()
					<-req.Context().Done()
					return nil, req.Context().Err()
				}
				resp := response("busy")
				resp.StatusCode = 503
				return resp, nil
			})}
			_, _, err := Do(client, request(t, ctx, "https://example.org"), "Crossref")
			if !errors.Is(err, context.Canceled) || calls != 1 {
				t.Fatalf("calls=%d error=%v", calls, err)
			}
		})
	}
}

func TestAttemptTimeoutAndOverallBudget(t *testing.T) {
	ctx, s, _ := testContext()
	s.attemptLimit = time.Millisecond
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls < 3 {
			<-req.Context().Done()
			return nil, req.Context().Err()
		}
		return response("ok"), nil
	})}
	_, body, err := Do(client, request(t, ctx, "https://example.org"), "Crossref")
	if err != nil || calls != 3 || string(body) != "ok" {
		t.Fatalf("attempt timeout: calls=%d error=%v", calls, err)
	}
	ctx, s, _ = testContext()
	s.budget = 5 * time.Millisecond
	calls = 0
	client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	_, _, err = Do(client, request(t, ctx, "https://example.org"), "Crossref")
	if !errors.Is(err, context.DeadlineExceeded) || calls != 1 {
		t.Fatalf("overall budget: calls=%d error=%v", calls, err)
	}
}

func TestArXivPacingIncludesRetriesAndNextLookup(t *testing.T) {
	ctx, s, clock := testContext()
	var starts []time.Time
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		starts = append(starts, s.now())
		resp := response("ok")
		if len(starts) == 1 {
			resp.StatusCode = 503
		}
		return resp, nil
	})}
	for _, path := range []string{"first", "second"} {
		if _, _, err := Do(client, request(t, ctx, "https://export.arxiv.org/"+path), "arXiv"); err != nil {
			t.Fatal(err)
		}
	}
	if len(starts) != 3 {
		t.Fatalf("starts=%v", starts)
	}
	for i := 1; i < len(starts); i++ {
		if starts[i].Sub(starts[i-1]) < 3*time.Second {
			t.Fatalf("requests too close: %v waits=%v", starts, clock.waits)
		}
	}
}

func TestArXivWaitingForConnectionIsCancellable(t *testing.T) {
	ctx, _, _ := testContext()
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		close(started)
		<-release
		return response("ok"), nil
	})}
	go func() {
		_, _, err := Do(client, request(t, ctx, "https://export.arxiv.org/first"), "arXiv")
		done <- err
	}()
	<-started
	second, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()
	_, _, err := Do(client, request(t, second, "https://export.arxiv.org/second"), "arXiv")
	close(release)
	if firstError := <-done; firstError != nil {
		t.Fatal(firstError)
	}
	if !errors.Is(err, context.DeadlineExceeded) || calls != 1 {
		t.Fatalf("connection wait: calls=%d error=%v", calls, err)
	}
}

func TestRetryAfterParsing(t *testing.T) {
	now := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		value string
		want  time.Duration
		valid bool
	}{
		{"0", 0, true}, {"12", 12 * time.Second, true}, {now.Add(5 * time.Second).Format(http.TimeFormat), 5 * time.Second, true},
		{now.Add(-time.Minute).Format(http.TimeFormat), 0, true}, {"", 0, false}, {"-1", 0, false}, {"1.5", 0, false}, {"9999999999999999999999", 0, false},
	} {
		got, valid := retryAfter(test.value, now)
		if got != test.want || valid != test.valid {
			t.Errorf("Retry-After %q: %s %t", test.value, got, valid)
		}
	}
}

func TestRetriesPreserveRedirectsAndDebugTrace(t *testing.T) {
	ctx, _, _ := testContext()
	calls := 0
	var trace bytes.Buffer
	defer diagnostics.SetOutput(&trace)()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/start" {
			http.Redirect(w, req, "/final", 302)
			return
		}
		calls++
		if calls == 1 {
			w.WriteHeader(502)
			return
		}
		io.WriteString(w, "ok")
	}))
	defer server.Close()
	resp, body, err := Do(server.Client(), request(t, ctx, server.URL+"/start"), "DOI service")
	if err != nil || resp.Request.URL.Path != "/final" || string(body) != "ok" || calls != 2 || !strings.Contains(trace.String(), "attempt 2/3") || !strings.Contains(trace.String(), "HTTP final URL") {
		t.Fatalf("redirect recovery: calls=%d body=%q error=%v trace=%q", calls, body, err, trace.String())
	}
}
