package acmeprototype

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// attemptTransport is a test-fixture adapter, not a production ACME transport.
// The caller supplies the complete base transport, including target/TLS policy.
type attemptTransport struct {
	attempt context.Context
	base    http.RoundTripper
}

func newAttemptTransport(ctx context.Context, base http.RoundTripper) (*attemptTransport, error) {
	if ctx == nil || base == nil {
		return nil, errors.New("attempt context and base transport are required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &attemptTransport{attempt: ctx, base: base}, nil
}

func (t *attemptTransport) RoundTrip(original *http.Request) (*http.Response, error) {
	if original == nil {
		return nil, errors.New("attempt transport requires a request")
	}
	caller := original.Context()
	predispatchError := func() error {
		if err := caller.Err(); err != nil {
			return err
		}
		return t.attempt.Err()
	}
	closeRequestBody := func() {
		if original.Body != nil {
			_ = original.Body.Close()
		}
	}
	if err := predispatchError(); err != nil {
		closeRequestBody()
		return nil, err
	}

	var merged context.Context
	var cancel context.CancelFunc
	deadline, hasDeadline := t.attempt.Deadline()
	if hasDeadline {
		// WithDeadline retains an earlier caller deadline and caller values.
		merged, cancel = context.WithDeadline(caller, deadline)
	} else {
		merged, cancel = context.WithCancel(caller)
	}
	stop := context.AfterFunc(t.attempt, func() {
		if hasDeadline && t.attempt.Err() == context.DeadlineExceeded {
			// The merged deadline independently expires at this time or earlier.
			// Do not race its DeadlineExceeded result with a plain Cancel call.
			return
		}
		cancel()
	})
	var releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() {
			stop()
			cancel()
		})
	}
	currentError := func() error {
		// An AfterFunc callback can be delayed. Recheck both sources directly
		// before dispatch and before admitting a returned response.
		if err := predispatchError(); err != nil {
			return err
		}
		return merged.Err()
	}
	// Clone headers, URL and context without modifying the caller's request.
	clone := original.Clone(merged)
	if err := currentError(); err != nil {
		release()
		closeRequestBody()
		return nil, err
	}

	// Request/body ownership transfers to the base only when it is called.
	response, err := t.base.RoundTrip(clone)
	if err == nil {
		err = currentError()
	}
	if err != nil {
		release()
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return nil, err
	}
	if response == nil {
		release()
		return nil, errors.New("base transport returned no response or error")
	}
	if response.Body == nil {
		release()
		return response, nil
	}
	// Do not defer release here: RoundTrip returns after headers, while body
	// reads may still need the merged context and attempt cancellation callback.
	response.Body = &attemptResponseBody{body: response.Body, release: release}
	return response, nil
}

type attemptResponseBody struct {
	body      io.ReadCloser
	release   func()
	closeOnce sync.Once
	closeErr  error
}

func (b *attemptResponseBody) Read(dst []byte) (int, error) {
	n, err := b.body.Read(dst)
	if err != nil {
		b.release()
	}
	return n, err
}

func (b *attemptResponseBody) Close() error {
	// Cancel before closing the base body so blocked reads can finish promptly.
	b.release()
	b.closeOnce.Do(func() { b.closeErr = b.body.Close() })
	return b.closeErr
}

type attemptTestRoundTripFunc func(*http.Request) (*http.Response, error)

func (f attemptTestRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type attemptTestBody struct {
	read   func([]byte) (int, error)
	closed atomic.Int32
}

func (b *attemptTestBody) Read(dst []byte) (int, error) { return b.read(dst) }
func (b *attemptTestBody) Close() error {
	b.closed.Add(1)
	return nil
}

// Expose registration accounting through Context's documented AfterFunc hook.
// Values come from the caller's request, not this attempt-only context.
type trackedAttemptContext struct {
	context.Context
	active, released atomic.Int32
	registered       func()
	callbackGate     <-chan struct{}
}

func (c *trackedAttemptContext) Value(any) any { return nil }

func (c *trackedAttemptContext) AfterFunc(f func()) func() bool {
	c.active.Add(1)
	var finished sync.Once
	finish := func() {
		finished.Do(func() {
			c.released.Add(1)
			c.active.Add(-1)
		})
	}
	stop := context.AfterFunc(c.Context, func() {
		if c.callbackGate != nil {
			<-c.callbackGate
		}
		finish()
		f()
	})
	if c.registered != nil {
		c.registered()
	}
	return func() bool {
		stopped := stop()
		if stopped {
			finish()
		}
		return stopped
	}
}

func attemptTestRequest(t *testing.T, ctx context.Context, body io.Reader) *http.Request {
	t.Helper()
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://fixture.acme.test/order", body)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func attemptTestTransport(t *testing.T, ctx context.Context, base http.RoundTripper) *attemptTransport {
	t.Helper()
	transport, err := newAttemptTransport(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	return transport
}

func waitAttemptTest(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("attempt transport operation did not finish")
	}
}

func waitAttemptRegistrationReleased(t *testing.T, tracked *trackedAttemptContext) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for tracked.active.Load() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("attempt cancellation registration was not released")
		}
		time.Sleep(time.Millisecond)
	}
	if tracked.released.Load() != 1 {
		t.Fatal("attempt cancellation registration was not released exactly once")
	}
}

func TestAttemptTransportRejectsInvalidInputs(t *testing.T) {
	base := attemptTestRoundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid input dispatched")
		return nil, nil
	})
	for _, input := range []struct {
		ctx  context.Context
		base http.RoundTripper
	}{{nil, base}, {context.Background(), nil}} {
		if transport, err := newAttemptTransport(input.ctx, input.base); transport != nil || err == nil {
			t.Fatal("invalid constructor input accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if transport, err := newAttemptTransport(ctx, base); transport != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled construction: %v", err)
	}
	transport := attemptTestTransport(t, context.Background(), base)
	if response, err := transport.RoundTrip(nil); response != nil || err == nil {
		t.Fatal("nil request accepted")
	}
}

func TestAttemptTransportPredispatchCancellationClosesRequest(t *testing.T) {
	for _, source := range []string{"attempt", "caller", "delayed-callback"} {
		t.Run(source, func(t *testing.T) {
			attempt, endAttempt := context.WithCancel(context.Background())
			defer endAttempt()
			caller, endCaller := context.WithCancel(context.Background())
			defer endCaller()
			gate := make(chan struct{})
			var releaseGateOnce sync.Once
			releaseGate := func() { releaseGateOnce.Do(func() { close(gate) }) }
			defer releaseGate()
			tracked := &trackedAttemptContext{Context: attempt}
			transport := attemptTestTransport(t, tracked, attemptTestRoundTripFunc(func(*http.Request) (*http.Response, error) {
				t.Error("canceled request reached the base transport")
				return nil, errors.New("unexpected dispatch")
			}))
			switch source {
			case "attempt":
				endAttempt()
			case "caller":
				endCaller()
			case "delayed-callback":
				tracked.callbackGate = gate
				tracked.registered = endAttempt
			}
			body := &attemptTestBody{read: func([]byte) (int, error) { return 0, io.EOF }}
			request := attemptTestRequest(t, caller, body)
			response, err := transport.RoundTrip(request)
			if response != nil || !errors.Is(err, context.Canceled) || body.closed.Load() != 1 {
				t.Fatalf("predispatch result: response=%v err=%v closes=%d", response, err, body.closed.Load())
			}
			if source == "delayed-callback" {
				releaseGate()
				waitAttemptRegistrationReleased(t, tracked)
			} else if tracked.active.Load() != 0 || tracked.released.Load() != 0 {
				t.Fatal("pre-canceled source registered an attempt callback")
			}
		})
	}
}

func TestAttemptTransportClonesRequestAndRetainsBodyContext(t *testing.T) {
	for _, earlier := range []string{"caller", "attempt", "none"} {
		t.Run(earlier, func(t *testing.T) {
			type valueKey struct{}
			caller := context.WithValue(context.Background(), valueKey{}, "caller-value")
			attempt := context.Background()
			wantDeadline := time.Now().Add(time.Hour)
			if earlier != "none" {
				callerDeadline, attemptDeadline := wantDeadline, wantDeadline.Add(time.Hour)
				if earlier == "attempt" {
					callerDeadline, attemptDeadline = attemptDeadline, callerDeadline
				}
				var cancelCaller, cancelAttempt context.CancelFunc
				caller, cancelCaller = context.WithDeadline(caller, callerDeadline)
				attempt, cancelAttempt = context.WithDeadline(attempt, attemptDeadline)
				defer cancelCaller()
				defer cancelAttempt()
			} else {
				var cancel context.CancelFunc
				attempt, cancel = context.WithCancel(attempt)
				defer cancel()
			}
			tracked := &trackedAttemptContext{Context: attempt}
			reader := strings.NewReader("fixture-body")
			body := &attemptTestBody{read: reader.Read}
			request := attemptTestRequest(t, caller, nil)
			request.Header.Set("X-Fixture", "original")
			var dispatched *http.Request
			transport := attemptTestTransport(t, tracked, attemptTestRoundTripFunc(func(clone *http.Request) (*http.Response, error) {
				dispatched = clone
				clone.Header.Set("X-Fixture", "base-only")
				clone.URL.Path = "/base-only"
				return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
			}))
			response, err := transport.RoundTrip(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if dispatched == request || request.Context() != caller || request.Header.Get("X-Fixture") != "original" || request.URL.Path != "/order" {
				t.Fatal("base transport mutated the original request")
			}
			if dispatched.Context().Value(valueKey{}) != "caller-value" || dispatched.Context().Err() != nil || tracked.active.Load() != 1 {
				t.Fatal("caller value or live body cancellation registration was lost")
			}
			deadline, ok := dispatched.Context().Deadline()
			if (earlier == "none" && ok) || (earlier != "none" && (!ok || !deadline.Equal(wantDeadline))) {
				t.Fatalf("merged deadline: %v, exists=%v", deadline, ok)
			}
			got, err := io.ReadAll(response.Body)
			if err != nil || string(got) != "fixture-body" {
				t.Fatalf("body was canceled when headers returned: %q, %v", got, err)
			}
			if tracked.active.Load() != 0 || tracked.released.Load() != 1 || dispatched.Context().Err() != context.Canceled {
				t.Fatal("EOF did not release the derived context/registration exactly once")
			}
			if caller.Err() != nil || attempt.Err() != nil {
				t.Fatal("body cleanup canceled an input context")
			}
			response.Body.Close()
			response.Body.Close()
			if body.closed.Load() != 1 || tracked.released.Load() != 1 {
				t.Fatal("repeated body Close repeated cleanup")
			}
		})
	}
}

func TestAttemptTransportCancelsInFlightRequest(t *testing.T) {
	attempt, cancel := context.WithCancel(context.Background())
	defer cancel()
	tracked := &trackedAttemptContext{Context: attempt}
	entered, done := make(chan struct{}), make(chan struct{})
	requestBody := &attemptTestBody{read: func([]byte) (int, error) { return 0, io.EOF }}
	forceExit := make(chan struct{})
	defer close(forceExit)
	transport := attemptTestTransport(t, tracked, attemptTestRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		defer request.Body.Close()
		close(entered)
		select {
		case <-request.Context().Done():
			return nil, request.Context().Err()
		case <-forceExit:
			return nil, errors.New("test cleanup forced dispatch exit")
		}
	}))
	var err error
	go func() {
		defer close(done)
		_, err = transport.RoundTrip(attemptTestRequest(t, context.Background(), requestBody))
	}()
	waitAttemptTest(t, entered)
	cancel()
	waitAttemptTest(t, done)
	if !errors.Is(err, context.Canceled) || requestBody.closed.Load() != 1 {
		t.Fatalf("in-flight cancellation: %v; request closes=%d", err, requestBody.closed.Load())
	}
	waitAttemptRegistrationReleased(t, tracked)
}

type attemptContextBody struct {
	ctx     context.Context
	entered chan struct{}
	closed  chan struct{}
	once    sync.Once
}

func (b *attemptContextBody) Read([]byte) (int, error) {
	b.once.Do(func() { close(b.entered) })
	select {
	case <-b.ctx.Done():
		return 0, b.ctx.Err()
	case <-b.closed:
		return 0, io.ErrClosedPipe
	}
}

func (b *attemptContextBody) Close() error {
	close(b.closed)
	return nil
}

func TestAttemptTransportCancellationAfterHeaders(t *testing.T) {
	for _, source := range []string{"attempt", "caller"} {
		t.Run(source, func(t *testing.T) {
			attempt, endAttempt := context.WithCancel(context.Background())
			defer endAttempt()
			caller, endCaller := context.WithCancel(context.Background())
			defer endCaller()
			tracked := &trackedAttemptContext{Context: attempt}
			var body *attemptContextBody
			transport := attemptTestTransport(t, tracked, attemptTestRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				body = &attemptContextBody{ctx: r.Context(), entered: make(chan struct{}), closed: make(chan struct{})}
				return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
			}))
			response, err := transport.RoundTrip(attemptTestRequest(t, caller, nil))
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if body.ctx.Err() != nil {
				t.Fatal("response body context canceled at RoundTrip return")
			}
			done := make(chan struct{})
			var readErr error
			go func() {
				defer close(done)
				_, readErr = response.Body.Read(make([]byte, 1))
			}()
			waitAttemptTest(t, body.entered)
			if source == "attempt" {
				endAttempt()
			} else {
				endCaller()
			}
			waitAttemptTest(t, done)
			if !errors.Is(readErr, context.Canceled) {
				t.Fatalf("post-header body cancellation: %v", readErr)
			}
			waitAttemptRegistrationReleased(t, tracked)
		})
	}
}

func TestAttemptTransportEarlierDeadlineExpires(t *testing.T) {
	for _, earlier := range []string{"caller", "attempt"} {
		t.Run(earlier, func(t *testing.T) {
			short, long := 100*time.Millisecond, time.Minute
			callerDelay, attemptDelay := short, long
			if earlier == "attempt" {
				callerDelay, attemptDelay = long, short
			}
			caller, endCaller := context.WithTimeout(context.Background(), callerDelay)
			defer endCaller()
			attempt, endAttempt := context.WithTimeout(context.Background(), attemptDelay)
			defer endAttempt()
			transport := attemptTestTransport(t, attempt, attemptTestRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				select {
				case <-r.Context().Done():
					return nil, r.Context().Err()
				case <-time.After(3 * time.Second):
					return nil, errors.New("earlier deadline was not applied")
				}
			}))
			if response, err := transport.RoundTrip(attemptTestRequest(t, caller, nil)); response != nil || !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("earlier %s deadline: %v", earlier, err)
			}
		})
	}
}

func TestAttemptTransportCleansUpErrorsAndLateResponse(t *testing.T) {
	for _, scenario := range []string{"transport-error", "response-and-error", "read-error", "late-response", "nil-response", "nil-body"} {
		t.Run(scenario, func(t *testing.T) {
			attempt, cancel := context.WithCancel(context.Background())
			defer cancel()
			tracked := &trackedAttemptContext{Context: attempt}
			failure := errors.New("fixture failure")
			body := &attemptTestBody{read: func([]byte) (int, error) { return 0, failure }}
			var dispatched context.Context
			transport := attemptTestTransport(t, tracked, attemptTestRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				dispatched = r.Context()
				response := &http.Response{StatusCode: http.StatusOK, Body: body}
				switch scenario {
				case "transport-error":
					return nil, failure
				case "response-and-error":
					return response, failure
				case "late-response":
					cancel()
				case "nil-response":
					return nil, nil
				case "nil-body":
					response.Body = nil
				}
				return response, nil
			}))
			response, err := transport.RoundTrip(attemptTestRequest(t, context.Background(), nil))
			if scenario == "read-error" {
				if err != nil || response == nil {
					t.Fatal("read-error response was not admitted")
				}
				_, err = response.Body.Read(make([]byte, 1))
				response.Body.Close()
			}
			if scenario == "nil-body" {
				if response == nil || err != nil {
					t.Fatal("empty response was not admitted")
				}
			} else if err == nil {
				t.Fatal("expected failure was lost")
			}
			if (scenario == "response-and-error" || scenario == "late-response") && (response != nil || body.closed.Load() != 1) {
				t.Fatal("rejected response body was not closed")
			}
			if dispatched.Err() == nil {
				t.Fatal("finished transport retained its derived context")
			}
			waitAttemptRegistrationReleased(t, tracked)
		})
	}
}

func TestAttemptTransportConcurrentBodyCleanup(t *testing.T) {
	attempt, cancel := context.WithCancel(context.Background())
	defer cancel()
	tracked := &trackedAttemptContext{Context: attempt}
	body := &attemptTestBody{read: func([]byte) (int, error) { return 0, io.EOF }}
	transport := attemptTestTransport(t, tracked, attemptTestRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
	}))
	response, err := transport.RoundTrip(attemptTestRequest(t, context.Background(), nil))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = response.Body.Read(make([]byte, 1))
			_ = response.Body.Close()
		}()
	}
	wg.Wait()
	if tracked.active.Load() != 0 || tracked.released.Load() != 1 || body.closed.Load() != 1 {
		t.Fatal("concurrent EOF and Close did not release each resource exactly once")
	}
}
