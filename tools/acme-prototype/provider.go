// Package acmeprototype evaluates an isolated HTTP-01 provider contract.
// It is not connected to Tong Panel, an ACME server, or a network listener.
package acmeprototype

import (
	"errors"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/http01"
)

var (
	ErrInput    = errors.New("invalid challenge input")
	ErrBusy     = errors.New("a different challenge is active")
	ErrMismatch = errors.New("cleanup does not match the active challenge")
	label       = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
	base64url   = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

// Provider holds at most one active challenge for one trusted configured host.
// Construction is not domain authorization: a future caller must enforce that.
type Provider struct {
	mu             sync.Mutex
	host           string
	ttl            time.Duration
	now            func() time.Time
	token, keyAuth string
	expires        time.Time
}

var _ challenge.Provider = (*Provider)(nil)
var _ http.Handler = (*Provider)(nil)

func New(host string, ttl time.Duration) (*Provider, error) {
	if len(host) > 253 || !strings.Contains(host, ".") || net.ParseIP(host) != nil || ttl <= 0 || ttl > 10*time.Minute {
		return nil, ErrInput
	}
	for _, part := range strings.Split(host, ".") {
		if !label.MatchString(part) {
			return nil, ErrInput
		}
	}
	return &Provider{host: host, ttl: ttl, now: time.Now}, nil
}

func (p *Provider) valid(domain, token, keyAuth string) bool {
	if domain != p.host || len(token) < 22 || len(token) > 256 || !base64url.MatchString(token) {
		return false
	}
	thumbprint, ok := strings.CutPrefix(keyAuth, token+".")
	return ok && len(thumbprint) == 43 && base64url.MatchString(thumbprint)
}

func (p *Provider) expire() {
	if !p.now().Before(p.expires) {
		p.token, p.keyAuth = "", ""
	}
}

// Present rejects concurrent replacement; identical retries do not extend expiry.
func (p *Provider) Present(domain, token, keyAuth string) error {
	if !p.valid(domain, token, keyAuth) {
		return ErrInput
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.expire()
	if p.token != "" {
		if p.token == token && p.keyAuth == keyAuth {
			return nil
		}
		return ErrBusy
	}
	p.token, p.keyAuth, p.expires = token, keyAuth, p.now().Add(p.ttl)
	return nil
}

// CleanUp is idempotent after expiry/removal and cannot delete another challenge.
func (p *Provider) CleanUp(domain, token, keyAuth string) error {
	if !p.valid(domain, token, keyAuth) {
		return ErrInput
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.expire()
	if p.token == "" {
		return nil
	}
	if p.token != token || p.keyAuth != keyAuth {
		return ErrMismatch
	}
	p.token, p.keyAuth = "", ""
	return nil
}

func (p *Provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	host := r.Host
	if strings.Contains(host, ":") {
		var err error
		host, _, err = net.SplitHostPort(host)
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.expire()
	if r.Method != http.MethodGet || host != p.host || p.token == "" || r.URL.Path != http01.ChallengePath(p.token) || r.URL.RawQuery != "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(p.keyAuth))
}
