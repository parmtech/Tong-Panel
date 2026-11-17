// Package acmeprototype evaluates an isolated HTTP-01 provider contract.
// It is not connected to Tong Panel, an ACME server, or a network listener.
package acmeprototype

import (
	"context"
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
	ErrClosed   = errors.New("challenge attempt is closed")
	label       = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
	base64url   = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

// Provider holds at most one active challenge for one trusted configured host.
// Construction is not domain authorization: a future caller must enforce that.
type Provider struct {
	mu             sync.Mutex
	attempt        context.Context
	stop           func() bool
	closed         bool
	host           string
	ttl            time.Duration
	now            func() time.Time
	token, keyAuth string
	expires        time.Time
}

var _ challenge.Provider = (*Provider)(nil)
var _ http.Handler = (*Provider)(nil)

func New(host string, ttl time.Duration) (*Provider, error) {
	return NewForAttempt(context.Background(), host, ttl)
}

// NewForAttempt binds this provider to one caller-owned attempt. Canceling ctx
// clears the challenge and prevents later presentation or HTTP admission. It
// does not cancel an ACME client's order, transport or previously admitted bytes.
// Call Close when finished to release the cancellation registration.
func NewForAttempt(ctx context.Context, host string, ttl time.Duration) (*Provider, error) {
	if ctx == nil {
		return nil, ErrInput
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(host) > 253 || !strings.Contains(host, ".") || net.ParseIP(host) != nil || ttl <= 0 || ttl > 10*time.Minute {
		return nil, ErrInput
	}
	for _, part := range strings.Split(host, ".") {
		if !label.MatchString(part) {
			return nil, ErrInput
		}
	}
	p := &Provider{attempt: ctx, host: host, ttl: ttl, now: time.Now}
	// The callback may start during registration. Publish its stop handle under
	// the same lock used by Close, then check for cancellation during creation.
	p.mu.Lock()
	p.stop = context.AfterFunc(ctx, p.Close)
	if err := ctx.Err(); err != nil {
		stop := p.closeLocked()
		p.unlock(stop)
		return nil, err
	}
	p.mu.Unlock()
	return p, nil
}

// Close permanently ends the attempt and clears its state. It is idempotent.
// Responses admitted before closure may already hold a copy of the challenge.
func (p *Provider) Close() {
	p.mu.Lock()
	stop := p.closeLocked()
	p.unlock(stop)
}

func (p *Provider) closeLocked() func() bool {
	p.closed = true
	p.token, p.keyAuth, p.expires = "", "", time.Time{}
	stop := p.stop
	p.stop = nil
	return stop
}

// Cancellation is checked synchronously as well as by AfterFunc, so a delayed
// callback cannot admit new activity after the attempt context is canceled.
func (p *Provider) fenceLocked() func() bool {
	if !p.closed && p.attempt.Err() != nil {
		return p.closeLocked()
	}
	return nil
}

func (p *Provider) unlock(stop func() bool) {
	p.mu.Unlock()
	if stop != nil {
		stop()
	}
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
		p.token, p.keyAuth, p.expires = "", "", time.Time{}
	}
}

// Present rejects concurrent replacement; identical retries do not extend expiry.
func (p *Provider) Present(domain, token, keyAuth string) error {
	if !p.valid(domain, token, keyAuth) {
		return ErrInput
	}
	p.mu.Lock()
	stop := p.fenceLocked()
	defer p.unlock(stop)
	if p.closed {
		return ErrClosed
	}
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
	stop := p.fenceLocked()
	defer p.unlock(stop)
	if p.closed {
		return nil
	}
	p.expire()
	if p.token == "" {
		return nil
	}
	if p.token != token || p.keyAuth != keyAuth {
		return ErrMismatch
	}
	p.token, p.keyAuth, p.expires = "", "", time.Time{}
	return nil
}

func (p *Provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	validHost := true
	if strings.Contains(host, ":") {
		var err error
		host, _, err = net.SplitHostPort(host)
		if err != nil {
			validHost = false
		}
	}
	p.mu.Lock()
	stop := p.fenceLocked()
	p.expire()
	keyAuth := ""
	if !p.closed && validHost && r.Method == http.MethodGet && host == p.host && p.token != "" && r.URL.Path == http01.ChallengePath(p.token) && r.URL.RawQuery == "" {
		keyAuth = p.keyAuth
	}
	p.unlock(stop)

	// Never hold the state lock across client-controlled ResponseWriter methods.
	// Close fences subsequent admissions but cannot recall this admitted copy.
	w.Header().Set("Cache-Control", "no-store")
	if keyAuth == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(keyAuth))
}
