// Package auth includes experimental authentication primitives.
// Sessions here are in-memory only: they are NOT an authenticated login flow.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/parmtech/tong-panel/internal/rbac"
)

// BrowserCookieName is host-bound when served over HTTPS (no Domain attribute).
const BrowserCookieName = "__Host-tong_session"
const MaxSessionLifetime = 24 * time.Hour
const maxSessionEntries = 10000

var ErrInvalidSession = errors.New("invalid session request")
var ErrSessionCapacity = errors.New("session capacity reached")

type sessionRecord struct {
	principal rbac.Principal
	csrfHash  [32]byte
	expires   time.Time
}

// SessionManager stores only SHA-256 token hashes and provides revocation.
// It is a safe primitive for tests, but it intentionally has no persistence,
// MFA enrollment, credential verification, distributed locking, or login HTTP route.
// Do not use it as production authentication without those capabilities.
type SessionManager struct {
	mu       sync.Mutex
	sessions map[[32]byte]sessionRecord
	now      func() time.Time
}

func NewSessionManager() *SessionManager {
	return &SessionManager{sessions: make(map[[32]byte]sessionRecord), now: time.Now}
}

// Issue creates two independent 256-bit random secrets: a bearer session token
// and a CSRF token. The caller must expose session tokens only via HTTPS and
// secure HttpOnly host-only cookies, and never log either secret.
func (m *SessionManager) Issue(p rbac.Principal, lifetime time.Duration) (token, csrf string, expires time.Time, err error) {
	if m == nil || p.UserID == "" || p.TenantID == "" || !validRole(p.Role) || lifetime <= 0 || lifetime > MaxSessionLifetime {
		return "", "", time.Time{}, ErrInvalidSession
	}
	var secret, csrfSecret [32]byte
	if _, err = rand.Read(secret[:]); err != nil {
		return "", "", time.Time{}, err
	}
	if _, err = rand.Read(csrfSecret[:]); err != nil {
		return "", "", time.Time{}, err
	}
	tokenHash := sha256.Sum256(secret[:])
	csrfHash := sha256.Sum256(csrfSecret[:])
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	// Bound unexpired session storage; fail closed instead of evicting another user.
	for key, value := range m.sessions {
		if !now.Before(value.expires) {
			delete(m.sessions, key)
		}
	}
	if len(m.sessions) >= maxSessionEntries {
		return "", "", time.Time{}, ErrSessionCapacity
	}
	expires = now.Add(lifetime)
	m.sessions[tokenHash] = sessionRecord{principal: p, csrfHash: csrfHash, expires: expires}
	return hex.EncodeToString(secret[:]), hex.EncodeToString(csrfSecret[:]), expires, nil
}

func sessionKey(token string) ([32]byte, bool) {
	var empty [32]byte
	if len(token) != 64 {
		return empty, false
	}
	raw, err := hex.DecodeString(token)
	if err != nil || len(raw) != 32 {
		return empty, false
	}
	return sha256.Sum256(raw), true
}

// Check returns no principal for unknown, malformed, expired or revoked tokens.
func (m *SessionManager) Check(token string) (rbac.Principal, bool) {
	if m == nil {
		return rbac.Principal{}, false
	}
	key, ok := sessionKey(token)
	if !ok {
		return rbac.Principal{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r, found := m.sessions[key]
	if !found {
		return rbac.Principal{}, false
	}
	if !m.now().Before(r.expires) {
		delete(m.sessions, key)
		return rbac.Principal{}, false
	}
	return r.principal, true
}

// CheckCSRF compares the independently generated CSRF secret for an active session.
// Applications must also verify Origin/Referer for cookie-authenticated mutations.
func (m *SessionManager) CheckCSRF(token, csrf string) bool {
	if m == nil {
		return false
	}
	key, ok := sessionKey(token)
	if !ok {
		return false
	}
	csrfKey, ok := sessionKey(csrf)
	if !ok {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.sessions[key]
	if !ok || !m.now().Before(r.expires) {
		return false
	}
	return subtle.ConstantTimeCompare(r.csrfHash[:], csrfKey[:]) == 1
}

// Revoke invalidates a single token immediately. Safe to call repeatedly.
func (m *SessionManager) Revoke(token string) {
	if m == nil {
		return
	}
	key, ok := sessionKey(token)
	if !ok {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, key)
}

// RevokeUser invalidates all sessions for a known user, e.g. after password reset.
func (m *SessionManager) RevokeUser(userID string) {
	if m == nil || userID == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for key, r := range m.sessions {
		if r.principal.UserID == userID {
			delete(m.sessions, key)
		}
	}
}

// SecureSessionCookie deliberately requires HTTPS: never set Secure=false.
func SecureSessionCookie(token string, expires time.Time) (*http.Cookie, error) {
	if _, ok := sessionKey(token); !ok || expires.IsZero() {
		return nil, ErrInvalidSession
	}
	return &http.Cookie{Name: BrowserCookieName, Value: token, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode, Expires: expires}, nil
}

// ExpiredSessionCookie clears the host-only cookie without weakening its flags.
func ExpiredSessionCookie() *http.Cookie {
	return &http.Cookie{Name: BrowserCookieName, Path: "/", Value: "", MaxAge: -1, Expires: time.Unix(0, 0), Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode}
}
