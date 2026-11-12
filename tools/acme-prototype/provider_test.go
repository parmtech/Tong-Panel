package acmeprototype

import (
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
