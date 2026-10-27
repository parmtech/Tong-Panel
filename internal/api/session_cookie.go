package api

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"strings"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/rbac"
)

// cookiePrincipal authenticates a session that was issued by a separate,
// reviewed login flow. The API intentionally does not issue cookies itself.
// HTTPS is required from the actual TLS connection, never X-Forwarded-Proto.
// A cookie must not be combined with developer bearer credentials.
func (s *Server) cookiePrincipal(r *http.Request, c *http.Cookie) (rbac.Principal, int, string) {
	if s.Sessions == nil || r.TLS == nil || r.Header.Get("Authorization") != "" {
		return rbac.Principal{}, http.StatusUnauthorized, "unauthorized"
	}
	// Duplicate cookie names have ambiguous parsing semantics. Reject them.
	count := 0
	for _, cookie := range r.Cookies() {
		if cookie.Name == auth.BrowserCookieName {
			count++
		}
	}
	if count != 1 {
		return rbac.Principal{}, http.StatusUnauthorized, "unauthorized"
	}
	p, ok := s.Sessions.Check(c.Value)
	if !ok {
		return rbac.Principal{}, http.StatusUnauthorized, "unauthorized"
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
		if !sameTLSOrigin(r) || !s.Sessions.CheckCSRF(c.Value, r.Header.Get("X-CSRF-Token")) {
			return rbac.Principal{}, http.StatusForbidden, "csrf_failed"
		}
	}
	return p, 0, ""
}

// sameTLSOrigin enforces the actual Host and HTTPS scheme. Never rely on
// caller-controlled proxy headers as proof of HTTPS or the trusted domain.
func sameTLSOrigin(r *http.Request) bool {
	if r.TLS == nil || r.Host == "" {
		return false
	}
	raw := r.Header.Get("Origin")
	if raw == "" || raw != strings.TrimSpace(raw) {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil ||
		u.Path != "" || u.RawPath != "" || u.RawQuery != "" || u.Fragment != "" ||
		u.Opaque != "" {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

// Explicit compile-time assertion of TLS-only trust source in API tests.
var _ *tls.ConnectionState
