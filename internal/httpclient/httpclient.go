// Package httpclient recovers transient failures of read-only provider requests.
package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/thofma/bibi/internal/diagnostics"
)

const (
	maxAttempts    = 3
	attemptTimeout = 15 * time.Second
	requestTimeout = 45 * time.Second
	arxivInterval  = 3 * time.Second
)

// Event describes a wait before an attempt. Observers run synchronously and
// should update the command's progress display, rather than writing to stdout.
type Event struct {
	Service string
	Attempt int
	Delay   time.Duration
	Reason  string
}

func (event Event) String() string {
	action := "retrying"
	if event.Attempt == 1 {
		action = "waiting"
	}
	return fmt.Sprintf("%s: %s; %s in %s — attempt %d/%d", event.Service, event.Reason, action,
		event.Delay.Round(100*time.Millisecond), event.Attempt, maxAttempts)
}

type sessionKey struct{}
type observerKey struct{}

type session struct {
	sync.Mutex
	cooldowns    map[string]time.Time
	arxivSlot    chan struct{}
	nextArxiv    time.Time
	now          func() time.Time
	wait         func(context.Context, time.Duration) error
	jitter       func() time.Duration
	budget       time.Duration
	attemptLimit time.Duration
}

func newSession() *session {
	return &session{cooldowns: make(map[string]time.Time), arxivSlot: make(chan struct{}, 1), now: time.Now,
		wait: wait, jitter: func() time.Duration { return time.Duration(rand.Int64N(int64(250 * time.Millisecond))) },
		budget: requestTimeout, attemptLimit: attemptTimeout}
}

// WithSession shares rate-limit cooldowns and arXiv pacing for one invocation.
func WithSession(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Value(sessionKey{}).(*session); ok {
		return ctx
	}
	return context.WithValue(ctx, sessionKey{}, newSession())
}

// WithObserver replaces progress reporting for a spinner or interactive picker.
func WithObserver(ctx context.Context, observer func(Event)) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, observerKey{}, observer)
}

func report(ctx context.Context, event Event) {
	diagnostics.Printf("%s", event.String())
	if observer, ok := ctx.Value(observerKey{}).(func(Event)); ok && observer != nil {
		observer(event)
	}
}

// Error preserves the transport cause while giving a bounded, actionable error.
type Error struct {
	Service  string
	Attempts int
	Cause    error
	RetryAt  time.Time
}

func (err *Error) Error() string {
	message := err.Service + " request failed before sending"
	if err.Attempts > 0 {
		plural := "s"
		if err.Attempts == 1 {
			plural = ""
		}
		message = fmt.Sprintf("%s request failed after %d attempt%s", err.Service, err.Attempts, plural)
	}
	message += fmt.Sprintf(": %v", err.Cause)
	if !err.RetryAt.IsZero() {
		return message + "; retry after " + err.RetryAt.UTC().Format(time.RFC3339)
	}
	return message + "; check your connection or try again later"
}
func (err *Error) Unwrap() error { return err.Cause }

// Do reads and closes the complete response before returning it to a parser.
// Nonretryable HTTP statuses are returned unchanged for provider-specific handling.
// Retries replay GET requests only; they never repeat parsing, exports or writes.
func Do(client *http.Client, request *http.Request, service string) (*http.Response, []byte, error) {
	parent := request.Context()
	s, ok := parent.Value(sessionKey{}).(*session)
	if !ok {
		s = newSession()
	}
	ctx, cancel := context.WithTimeout(parent, s.budget)
	defer cancel()
	deadline := s.now().Add(s.budget)
	if earlier, ok := parent.Deadline(); ok && earlier.Before(deadline) {
		deadline = earlier
	}
	var response *http.Response
	var body []byte
	var cause error
	var retryAt time.Time
	attempts := 0
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			if parent.Err() != nil {
				return nil, nil, parent.Err()
			}
			return nil, nil, &Error{service, attempts, err, retryAt}
		}
		if service == "arXiv" {
			select {
			case s.arxivSlot <- struct{}{}:
			case <-ctx.Done():
				if parent.Err() != nil {
					return nil, nil, parent.Err()
				}
				return nil, nil, &Error{service, attempts, ctx.Err(), retryAt}
			}
		}
		release := func() {
			if service == "arXiv" {
				<-s.arxivSlot
			}
		}
		s.Lock()
		ready := s.cooldowns[request.URL.Host]
		waitReason := "server cooldown"
		if service == "arXiv" && s.nextArxiv.After(ready) {
			ready = s.nextArxiv
			waitReason = "respecting arXiv request interval"
		}
		s.Unlock()
		if ready.After(s.now()) {
			delay := ready.Sub(s.now())
			if !ready.Before(deadline) {
				release()
				return nil, nil, &Error{service, attempts, fmt.Errorf("server wait exceeds request deadline"), ready}
			}
			report(ctx, Event{service, attempt, delay, waitReason})
			if err := s.wait(ctx, delay); err != nil {
				release()
				if parent.Err() != nil {
					return nil, nil, parent.Err()
				}
				return nil, nil, &Error{service, attempts, err, ready}
			}
		}
		if err := ctx.Err(); err != nil {
			release()
			if parent.Err() != nil {
				return nil, nil, parent.Err()
			}
			return nil, nil, &Error{service, attempts, err, retryAt}
		}
		if service == "arXiv" {
			s.Lock()
			s.nextArxiv = s.now().Add(arxivInterval)
			s.Unlock()
		}
		attempts++
		attemptCtx, attemptCancel := context.WithTimeout(ctx, s.attemptLimit)
		response, cause = diagnostics.Do(client, request.Clone(attemptCtx))
		body = nil
		if response != nil && response.Body != nil {
			if cause == nil {
				body, cause = io.ReadAll(response.Body)
			}
			response.Body.Close()
		}
		attemptCancel()
		release()
		if parent.Err() != nil {
			return nil, nil, parent.Err()
		}
		if ctx.Err() != nil {
			return nil, nil, &Error{service, attempts, ctx.Err(), retryAt}
		}
		retry := transient(cause)
		reason := "connection interrupted"
		if cause == nil {
			if response == nil {
				return nil, nil, &Error{service, attempts, errors.New("empty HTTP response"), time.Time{}}
			}
			retry = retryStatus(response.StatusCode)
			if !retry {
				return response, body, nil
			}
			cause = fmt.Errorf("HTTP %d %s", response.StatusCode, http.StatusText(response.StatusCode))
			reason = cause.Error()
		} else if errors.Is(cause, context.DeadlineExceeded) {
			reason = "request timed out"
		}
		if response != nil && response.StatusCode >= 400 && !retryStatus(response.StatusCode) {
			retry = false
		}
		delay := time.Duration(1<<uint(attempt-1))*time.Second + s.jitter()
		retryAt = time.Time{}
		if response != nil && retryStatus(response.StatusCode) {
			if serverDelay, valid := retryAfter(response.Header.Get("Retry-After"), s.now()); valid {
				delay = max(delay, serverDelay)
			}
			retryAt = s.now().Add(delay)
			s.Lock()
			for _, host := range []string{request.URL.Host, responseHost(response)} {
				if host != "" && retryAt.After(s.cooldowns[host]) {
					s.cooldowns[host] = retryAt
				}
			}
			s.Unlock()
		}
		if service == "arXiv" {
			s.Lock()
			delay = max(delay, s.nextArxiv.Sub(s.now()))
			s.Unlock()
		}
		if !retry || request.Method != http.MethodGet || attempt == maxAttempts {
			if len(body) > 0 {
				diagnostics.Preview(service, string(body))
			}
			return response, nil, &Error{service, attempts, cause, retryAt}
		}
		if !s.now().Add(delay).Before(deadline) {
			return response, nil, &Error{service, attempts, fmt.Errorf("%w; retry wait exceeds request deadline", cause), retryAt}
		}
		report(ctx, Event{service, attempt + 1, delay, reason})
		if err := s.wait(ctx, delay); err != nil {
			if parent.Err() != nil {
				return nil, nil, parent.Err()
			}
			return nil, nil, &Error{service, attempts, err, retryAt}
		}
	}
	panic("unreachable")
}

func responseHost(response *http.Response) string {
	if response.Request != nil && response.Request.URL != nil {
		return response.Request.URL.Host
	}
	return ""
}

func retryStatus(code int) bool {
	switch code {
	case 408, 429, 500, 502, 503, 504:
		return true
	}
	return false
}

func transient(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return dns.IsTimeout || dns.IsTemporary
	}
	var network net.Error
	if errors.As(err, &network) && (network.Timeout() || network.Temporary()) {
		return true
	}
	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.EPIPE)
}

func retryAfter(value string, now time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if date, err := http.ParseTime(value); err == nil {
		return max(0, date.Sub(now)), true
	}
	// Parse decimal seconds without allowing negatives or overflowing durations.
	var seconds int64
	for _, char := range value {
		if char < '0' || char > '9' || seconds > (int64((1<<63-1)/time.Second)-int64(char-'0'))/10 {
			return 0, false
		}
		seconds = seconds*10 + int64(char-'0')
	}
	return time.Duration(seconds) * time.Second, true
}

func wait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}
