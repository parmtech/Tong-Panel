package acmeprototype

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-acme/lego/v4/challenge/http01"
)

const host = "tenant.example.test"

var token = strings.Repeat("a", 32)
var auth = token + "." + strings.Repeat("b", 43)

func fixture(t *testing.T) *Provider {
	t.Helper()
	p, err := New(host, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

func request(p *Provider, method, authority, path string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://"+authority+path, nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, r)
	return w
}

func TestChallengeLifecycle(t *testing.T) {
	p := fixture(t)
	path := http01.ChallengePath(token)
	if got := request(p, "GET", host, path); got.Code != 404 {
		t.Fatalf("unpresented: %d", got.Code)
	}
	if err := p.Present(host, token, auth); err != nil {
		t.Fatal(err)
	}
	for _, authority := range []string{host, host + ":80"} {
		got := request(p, "GET", authority, path)
		if got.Code != 200 || got.Body.String() != auth || got.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("challenge: %d %q", got.Code, got.Body.String())
		}
	}
	for _, bad := range []struct{ method, authority, path string }{
		{"GET", "other.example.test", path}, {"POST", host, path}, {"HEAD", host, path},
		{"GET", host, path + "?extra=1"}, {"GET", host, path + "/"}, {"GET", host, "/"},
		{"GET", host, http01.ChallengePath("unknown")},
	} {
		if got := request(p, bad.method, bad.authority, bad.path); got.Code != http.StatusNotFound {
			t.Fatalf("unexpected access for %+v: %d", bad, got.Code)
		}
	}
	if err := p.CleanUp(host, token, auth); err != nil {
		t.Fatal(err)
	}
	if err := p.CleanUp(host, token, auth); err != nil {
		t.Fatal(err)
	}
	if got := request(p, "GET", host, path); got.Code != 404 {
		t.Fatalf("cleanup: %d", got.Code)
	}
}

func TestExpiryRetryAndStaleCleanup(t *testing.T) {
	p := fixture(t)
	now := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)
	p.now = func() time.Time { return now }
	if err := p.Present(host, token, auth); err != nil {
		t.Fatal(err)
	}
	now = now.Add(30 * time.Second)
	if err := p.Present(host, token, auth); err != nil {
		t.Fatal(err)
	}
	second := strings.Repeat("c", 32)
	secondAuth := second + "." + strings.Repeat("d", 43)
	if err := p.Present(host, second, secondAuth); !errors.Is(err, ErrBusy) {
		t.Fatalf("replacement: %v", err)
	}
	now = now.Add(30 * time.Second)
	if got := request(p, "GET", host, http01.ChallengePath(token)); got.Code != 404 {
		t.Fatalf("expiry: %d", got.Code)
	}
	if err := p.Present(host, second, secondAuth); err != nil {
		t.Fatal(err)
	}
	if err := p.CleanUp(host, token, auth); !errors.Is(err, ErrMismatch) {
		t.Fatalf("stale cleanup: %v", err)
	}
	if got := request(p, "GET", host, http01.ChallengePath(second)); got.Body.String() != secondAuth {
		t.Fatal("stale cleanup removed new challenge")
	}
}

func TestInvalidInput(t *testing.T) {
	for _, name := range []string{"", "*.example.test", "../example.test", "127.0.0.1", "EXAMPLE.test", "a..test", "-a.test", strings.Repeat("a", 64) + ".test"} {
		if _, err := New(name, time.Minute); err == nil {
			t.Errorf("accepted host %q", name)
		}
	}
	for _, ttl := range []time.Duration{0, -time.Second, 11 * time.Minute} {
		if _, err := New(host, ttl); err == nil {
			t.Errorf("accepted ttl %s", ttl)
		}
	}
	p := fixture(t)
	for _, input := range []struct{ domain, token, auth string }{
		{"other.example.test", token, auth}, {host, "../file", auth}, {host, "short", auth},
		{host, token, "invalid"}, {host, token, token + "." + strings.Repeat("b", 42)},
		{host, token, token + "." + strings.Repeat("!", 43)},
	} {
		if err := p.Present(input.domain, input.token, input.auth); !errors.Is(err, ErrInput) {
			t.Errorf("Present accepted %+v", input)
		}
		if err := p.CleanUp(input.domain, input.token, input.auth); !errors.Is(err, ErrInput) {
			t.Errorf("CleanUp accepted %+v", input)
		}
	}
}

func TestConcurrentProviderUse(t *testing.T) {
	p := fixture(t)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if err := p.Present(host, token, auth); err != nil {
					t.Error(err)
				}
				got := request(p, "GET", host, http01.ChallengePath(token))
				if got.Code != 200 && got.Code != 404 {
					t.Errorf("status %d", got.Code)
				}
				if err := p.CleanUp(host, token, auth); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
}

func assertAttemptClosed(t *testing.T, p *Provider) {
	t.Helper()
	p.mu.Lock()
	closed := p.closed
	empty := p.token == "" && p.keyAuth == "" && p.expires.IsZero() && p.stop == nil
	p.mu.Unlock()
	if !closed || !empty {
		t.Fatal("closed attempt retained state or cancellation registration")
	}
	for _, value := range []string{token, strings.Repeat("c", 32)} {
		keyAuth := value + "." + strings.Repeat("b", 43)
		if err := p.Present(host, value, keyAuth); !errors.Is(err, ErrClosed) {
			t.Fatalf("closed attempt accepted Present: %v", err)
		}
		if err := p.CleanUp(host, value, keyAuth); err != nil {
			t.Fatalf("closed attempt cleanup was not idempotent: %v", err)
		}
		if got := request(p, "GET", host, http01.ChallengePath(value)); got.Code != http.StatusNotFound {
			t.Fatalf("closed attempt served a new request: %d", got.Code)
		}
	}
}

func waitForClosedAttempt(t *testing.T, p *Provider) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		p.mu.Lock()
		closed := p.closed
		p.mu.Unlock()
		if closed {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("cancellation callback did not close the idle attempt")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestAttemptRejectsInvalidContext(t *testing.T) {
	if p, err := NewForAttempt(nil, host, time.Minute); p != nil || !errors.Is(err, ErrInput) {
		t.Fatalf("nil context: provider=%v error=%v", p, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if p, err := NewForAttempt(ctx, host, time.Minute); p != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled context: provider=%v error=%v", p, err)
	}
	ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if p, err := NewForAttempt(ctx, host, time.Minute); p != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expired context: provider=%v error=%v", p, err)
	}
}

func TestClosePermanentlyEndsAttempt(t *testing.T) {
	p := fixture(t)
	if err := p.Present(host, token, auth); err != nil {
		t.Fatal(err)
	}
	p.Close()
	p.Close()
	assertAttemptClosed(t, p)
}

func TestAttemptCancellationClearsIdleState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p, err := NewForAttempt(ctx, host, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	if err := p.Present(host, token, auth); err != nil {
		t.Fatal(err)
	}
	cancel()
	// Do not invoke provider methods while waiting: the callback must clear an
	// idle challenge even if no HTTP, Present or CleanUp call follows cancel.
	waitForClosedAttempt(t, p)
	assertAttemptClosed(t, p)
}

func TestAttemptCancellationFencesBeforeCallback(t *testing.T) {
	for _, entrypoint := range []string{"present", "cleanup", "http"} {
		t.Run(entrypoint, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			p, err := NewForAttempt(ctx, host, time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(p.Close)
			if err := p.Present(host, token, auth); err != nil {
				t.Fatal(err)
			}
			// Detach the asynchronous callback to deterministically exercise the
			// synchronous cancellation fence before any callback can clear state.
			p.mu.Lock()
			stop := p.stop
			p.mu.Unlock()
			if !stop() {
				t.Fatal("cancellation callback was already running")
			}
			cancel()
			switch entrypoint {
			case "present":
				if err := p.Present(host, token, auth); !errors.Is(err, ErrClosed) {
					t.Fatalf("Present after cancellation: %v", err)
				}
			case "cleanup":
				if err := p.CleanUp(host, token, auth); err != nil {
					t.Fatal(err)
				}
			case "http":
				if got := request(p, "GET", host, http01.ChallengePath(token)); got.Code != http.StatusNotFound {
					t.Fatalf("GET after cancellation: %d", got.Code)
				}
			}
			assertAttemptClosed(t, p)
		})
	}
}

type blockingResponseWriter struct {
	recorder *httptest.ResponseRecorder
	write    bool
	entered  chan struct{}
	release  chan struct{}
	once     sync.Once
}

func (w *blockingResponseWriter) Header() http.Header { return w.recorder.Header() }

func (w *blockingResponseWriter) block() {
	w.once.Do(func() { close(w.entered) })
	<-w.release
}

func (w *blockingResponseWriter) WriteHeader(code int) {
	if !w.write {
		w.block()
	}
	w.recorder.WriteHeader(code)
}

func (w *blockingResponseWriter) Write(body []byte) (int, error) {
	if w.write {
		w.block()
	}
	return w.recorder.Write(body)
}

func TestBlockedHTTPWriterDoesNotBlockClose(t *testing.T) {
	for _, blockWrite := range []bool{true, false} {
		name := map[bool]string{true: "challenge-body", false: "not-found-header"}[blockWrite]
		t.Run(name, func(t *testing.T) {
			p := fixture(t)
			if err := p.Present(host, token, auth); err != nil {
				t.Fatal(err)
			}
			w := &blockingResponseWriter{recorder: httptest.NewRecorder(), write: blockWrite, entered: make(chan struct{}), release: make(chan struct{})}
			method := http.MethodGet
			if !blockWrite {
				method = http.MethodPost
			}
			r := httptest.NewRequest(method, "http://"+host+http01.ChallengePath(token), nil)
			handlerDone := make(chan struct{})
			var closeDone chan struct{}
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(w.release) }) }
			t.Cleanup(func() {
				// Release the writer even after Fatal, then join both goroutines so
				// the pre-fix deadlock cannot leak into following tests.
				release()
				for _, done := range []chan struct{}{handlerDone, closeDone} {
					if done == nil {
						continue
					}
					select {
					case <-done:
					case <-time.After(3 * time.Second):
						t.Error("HTTP/Close goroutine did not stop after writer release")
					}
				}
			})
			go func() {
				defer close(handlerDone)
				p.ServeHTTP(w, r)
			}()
			select {
			case <-w.entered:
			case <-time.After(3 * time.Second):
				t.Fatal("HTTP writer did not reach its blocking operation")
			}
			closeDone = make(chan struct{})
			go func() {
				p.Close()
				close(closeDone)
			}()
			select {
			case <-closeDone:
			case <-time.After(3 * time.Second):
				t.Fatal("blocked HTTP writer prevented Close")
			}
			assertAttemptClosed(t, p)
			release()
			<-handlerDone
			if blockWrite {
				// A response admitted before Close may finish sending its copy.
				if w.recorder.Code != http.StatusOK || w.recorder.Body.String() != auth {
					t.Fatal("already-admitted response did not finish consistently")
				}
			} else if w.recorder.Code != http.StatusNotFound {
				t.Fatalf("rejected request status: %d", w.recorder.Code)
			}
		})
	}
}

func TestConcurrentAttemptCancellationAndClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p, err := NewForAttempt(ctx, host, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 40; j++ {
				if err := p.Present(host, token, auth); err != nil && !errors.Is(err, ErrClosed) {
					t.Error(err)
				}
				got := request(p, "GET", host, http01.ChallengePath(token))
				if got.Code != http.StatusOK && got.Code != http.StatusNotFound {
					t.Errorf("unexpected status: %d", got.Code)
				}
				if err := p.CleanUp(host, token, auth); err != nil {
					t.Error(err)
				}
				if j == 5 {
					cancel()
				}
				if j >= 10 {
					p.Close()
				}
			}
		}()
	}
	close(start)
	wg.Wait()
	p.Close()
	assertAttemptClosed(t, p)
}

func TestCancellationDuringAttemptConstruction(t *testing.T) {
	for i := 0; i < 30; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		type result struct {
			provider *Provider
			err      error
		}
		started := make(chan struct{})
		created := make(chan result, 1)
		go func() {
			close(started)
			p, err := NewForAttempt(ctx, host, time.Minute)
			created <- result{p, err}
		}()
		<-started
		cancel()
		got := <-created
		if got.err != nil {
			if got.provider != nil || !errors.Is(got.err, context.Canceled) {
				t.Fatalf("construction cancellation: provider=%v error=%v", got.provider, got.err)
			}
			continue
		}
		if got.provider == nil {
			t.Fatal("successful construction returned nil provider")
		}
		waitForClosedAttempt(t, got.provider)
		assertAttemptClosed(t, got.provider)
	}
}
