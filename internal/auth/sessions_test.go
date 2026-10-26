package auth

import (
	"github.com/parmtech/tong-panel/internal/rbac"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func principal(id string) rbac.Principal {
	return rbac.Principal{UserID: id, TenantID: "tenant-1", Role: rbac.Customer}
}
func TestSessionIssueCheckRevokeExpiry(t *testing.T) {
	s := NewSessionManager()
	fakeNow := time.Unix(120000, 0)
	s.now = func() time.Time { return fakeNow }
	token, csrf, expiry, err := s.Issue(principal("one"), time.Hour)
	if err != nil || token == csrf || len(token) != 64 || len(csrf) != 64 || expiry != fakeNow.Add(time.Hour) {
		t.Fatalf("issue %v", err)
	}
	p, ok := s.Check(token)
	if !ok || p.UserID != "one" || !s.CheckCSRF(token, csrf) {
		t.Fatal("valid session denied")
	}
	if s.CheckCSRF(token, token) || s.CheckCSRF(token, strings.Repeat("0", 64)) {
		t.Fatal("csrf bypass")
	}
	if _, ok = s.Check("bad"); ok {
		t.Fatal("bad token accepted")
	}
	fakeNow = fakeNow.Add(time.Hour)
	if _, ok = s.Check(token); ok || s.CheckCSRF(token, csrf) {
		t.Fatal("expired session accepted")
	}
	token, _, _, err = s.Issue(principal("one"), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	s.Revoke(token)
	if _, ok = s.Check(token); ok {
		t.Fatal("revoked accepted")
	}
}
func TestSessionRoleAndLifetimeValidation(t *testing.T) {
	s := NewSessionManager()
	for _, d := range []time.Duration{0, -time.Second, MaxSessionLifetime + time.Second} {
		if _, _, _, err := s.Issue(principal("one"), d); err == nil {
			t.Fatal("bad lifetime accepted")
		}
	}
	for _, p := range []rbac.Principal{{TenantID: "t", Role: rbac.Customer}, {UserID: "u", Role: rbac.Customer}, {UserID: "u", TenantID: "t", Role: "root"}} {
		if _, _, _, err := s.Issue(p, time.Minute); err == nil {
			t.Fatal("bad principal accepted")
		}
	}
}
func TestSessionRevokeUserAndIsolation(t *testing.T) {
	s := NewSessionManager()
	a, _, _, _ := s.Issue(principal("a"), time.Hour)
	b, _, _, _ := s.Issue(principal("b"), time.Hour)
	s.RevokeUser("a")
	if _, ok := s.Check(a); ok {
		t.Fatal("user logout failed")
	}
	if p, ok := s.Check(b); !ok || p.UserID != "b" {
		t.Fatal("another user removed")
	}
}
func TestSessionConcurrentAndUnique(t *testing.T) {
	s := NewSessionManager()
	var wg sync.WaitGroup
	tok := make(chan string, 40)
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, _, _, err := s.Issue(principal("user"), time.Hour)
			if err != nil {
				t.Error(err)
				return
			}
			tok <- v
		}()
	}
	wg.Wait()
	close(tok)
	seen := map[string]bool{}
	for x := range tok {
		if seen[x] {
			t.Fatal("repeated token")
		}
		seen[x] = true
		if _, ok := s.Check(x); !ok {
			t.Fatal("concurrent lost session")
		}
	}
}
func TestSecureCookiesAlwaysHostBound(t *testing.T) {
	s := NewSessionManager()
	token, _, ex, _ := s.Issue(principal("u"), time.Hour)
	c, err := SecureSessionCookie(token, ex)
	if err != nil || !c.Secure || !c.HttpOnly || c.Domain != "" || c.Path != "/" || c.SameSite != http.SameSiteStrictMode || !strings.HasPrefix(c.Name, "__Host-") {
		t.Fatal("insecure cookie settings")
	}
	old := ExpiredSessionCookie()
	if !old.Secure || !old.HttpOnly || old.MaxAge != -1 {
		t.Fatal("insecure logout cookie")
	}
	if _, err := SecureSessionCookie("junk", ex); err == nil {
		t.Fatal("accepted invalid token")
	}
}
