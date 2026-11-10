package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/rbac"
	"github.com/parmtech/tong-panel/internal/sites"
)

type heldProofLookup struct {
	entered chan struct{}
	release chan struct{}
	value   string
}

func (l *heldProofLookup) LookupTXT(ctx context.Context, _ string) ([]string, error) {
	close(l.entered)
	select {
	case <-l.release:
		return []string{l.value}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestDomainProofReauthorizesAfterUnlockedDNS(t *testing.T) {
	for _, update := range []string{"UPDATE memberships SET role='auditor' WHERE tenant_id='tenant-a' AND identity_id='u1'", "UPDATE identities SET disabled=1 WHERE id='u1'", "UPDATE tenants SET status='suspended' WHERE id='tenant-a'"} {
		t.Run(update, func(t *testing.T) {
			s, _, db := setupSiteAPI(t)
			if r := createTestSite(t, s.Handler(), secret, "example", "primary.example.test"); r.Code != 201 {
				t.Fatal(r.Body.String())
			}
			p := rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer}
			started, err := s.SiteCatalog.BeginDomainVerificationFor(context.Background(), p.TenantID, "example", "primary.example.test", p.UserID, nil)
			if err != nil {
				t.Fatal(err)
			}
			lookup := &heldProofLookup{entered: make(chan struct{}), release: make(chan struct{}), value: started.Value}
			s.DomainTXTLookup = lookup
			result := make(chan *httptest.ResponseRecorder, 1)
			go func() {
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"value":"`+started.Value+`"}`))
				req.Header.Set("Content-Type", "application/json")
				s.siteDomainVerificationComplete(w, req, p, "example", "primary.example.test")
				result <- w
			}()
			select {
			case <-lookup.entered:
			case <-time.After(3 * time.Second):
				t.Fatal("DNS not entered")
			}
			// This succeeds while DNS is held, demonstrating no pool/writer lock is held.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if _, err := db.ExecContext(ctx, update); err != nil {
				close(lookup.release)
				t.Fatalf("write blocked during DNS: %v", err)
			}
			close(lookup.release)
			select {
			case w := <-result:
				if w.Code != 403 {
					t.Fatalf("withdrawn authority %d %s", w.Code, w.Body.String())
				}
			case <-time.After(3 * time.Second):
				t.Fatal("completion stuck")
			}
			var state string
			var n int
			if err := db.QueryRow("SELECT ownership_state FROM site_domains WHERE domain='primary.example.test'").Scan(&state); err != nil || state != "pending" {
				t.Fatalf("state %s %v", state, err)
			}
			if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='site.domain_verified'").Scan(&n); err != nil || n != 0 {
				t.Fatal("denied completion audited")
			}
		})
	}
}

func TestDomainProofAuditFailureRollsBackBeginAndComplete(t *testing.T) {
	s, lookup, db := setupSiteAPI(t)
	if r := createTestSite(t, s.Handler(), secret, "example", "primary.example.test"); r.Code != 201 {
		t.Fatal(r.Body.String())
	}
	path := "/api/v1/sites/example/domains/primary.example.test/verification"
	if _, err := db.Exec(`CREATE TRIGGER fail_begin BEFORE INSERT ON audit_events WHEN NEW.action='site.domain_verification_started' BEGIN SELECT RAISE(ABORT,'fail'); END`); err != nil {
		t.Fatal(err)
	}
	if w := call(s.Handler(), http.MethodPost, path, secret, "", ""); w.Code != 503 {
		t.Fatalf("begin audit failure %d %s", w.Code, w.Body.String())
	}
	var digest any
	if err := db.QueryRow("SELECT verification_token_sha256 FROM site_domains WHERE domain='primary.example.test'").Scan(&digest); err != nil || digest != nil {
		t.Fatalf("failed begin retained digest %v %v", digest, err)
	}
	if _, err := db.Exec("DROP TRIGGER fail_begin"); err != nil {
		t.Fatal(err)
	}
	w := call(s.Handler(), http.MethodPost, path, secret, "", "")
	if w.Code != 201 {
		t.Fatalf("begin %d %s", w.Code, w.Body.String())
	}
	var instruction sites.DomainVerificationInstruction
	if err := json.Unmarshal(w.Body.Bytes(), &instruction); err != nil {
		t.Fatal(err)
	}
	lookup.set(instruction.Name, instruction.Value)
	if _, err := db.Exec(`CREATE TRIGGER fail_complete BEFORE INSERT ON audit_events WHEN NEW.action='site.domain_verified' BEGIN SELECT RAISE(ABORT,'fail'); END`); err != nil {
		t.Fatal(err)
	}
	w = call(s.Handler(), http.MethodPost, path+"/complete", secret, "", `{"value":"`+instruction.Value+`"}`)
	if w.Code != 503 {
		t.Fatalf("complete audit failure %d %s", w.Code, w.Body.String())
	}
	var state string
	if err := db.QueryRow("SELECT ownership_state FROM site_domains WHERE domain='primary.example.test'").Scan(&state); err != nil || state != "pending" {
		t.Fatalf("failed audit verified %s %v", state, err)
	}
}

func TestDomainProofCookieCSRFAuthorization(t *testing.T) {
	s, lookup, db := setupSiteAPI(t)
	if r := createTestSite(t, s.Handler(), secret, "example", "primary.example.test"); r.Code != 201 {
		t.Fatal(r.Body.String())
	}
	store, err := auth.NewSQLiteSessionStore(db)
	if err != nil {
		t.Fatal(err)
	}
	s.Sessions = store
	token, csrf, _, err := store.Issue(rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/sites/example/domains/primary.example.test/verification"
	if w := cookieCall(s.Handler(), http.MethodPost, path, token, "", "https://panel.test", ""); w.Code != 403 {
		t.Fatalf("missing csrf %d", w.Code)
	}
	w := cookieCall(s.Handler(), http.MethodPost, path, token, csrf, "https://panel.test", "")
	if w.Code != 201 {
		t.Fatalf("begin %d %s", w.Code, w.Body.String())
	}
	var instruction sites.DomainVerificationInstruction
	if err := json.Unmarshal(w.Body.Bytes(), &instruction); err != nil {
		t.Fatal(err)
	}
	lookup.set(instruction.Name, instruction.Value)
	body := `{"value":"` + instruction.Value + `"}`
	if w := cookieCall(s.Handler(), http.MethodPost, path+"/complete", token, "", "https://panel.test", body); w.Code != 403 {
		t.Fatalf("complete missing csrf %d", w.Code)
	}
	if w := cookieCall(s.Handler(), http.MethodPost, path+"/complete", token, csrf, "https://panel.test", body); w.Code != 200 {
		t.Fatalf("complete %d %s", w.Code, w.Body.String())
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action IN ('site.domain_verification_started','site.domain_verified') AND actor_identity_id='u1' AND subject_id='example/primary.example.test' AND instr(subject_id,?)=0", instruction.Value).Scan(&n); err != nil || n != 2 {
		t.Fatalf("audit attribution %d %v", n, err)
	}
}

func TestDomainProofReplacementAndSessionRevocationDuringDNS(t *testing.T) {
	for _, revoke := range []bool{false, true} {
		t.Run(map[bool]string{false: "replacement", true: "session revoked"}[revoke], func(t *testing.T) {
			s, _, db := setupSiteAPI(t)
			if r := createTestSite(t, s.Handler(), secret, "example", "primary.example.test"); r.Code != 201 {
				t.Fatal(r.Body.String())
			}
			p := rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer}
			store, err := auth.NewSQLiteSessionStore(db)
			if err != nil {
				t.Fatal(err)
			}
			token, _, _, err := store.Issue(p, time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			proof, ok := store.ReadProof(token)
			if !ok {
				t.Fatal("proof")
			}
			started, err := s.SiteCatalog.BeginDomainVerificationFor(context.Background(), p.TenantID, "example", "primary.example.test", p.UserID, &proof)
			if err != nil {
				t.Fatal(err)
			}
			lookup := &heldProofLookup{entered: make(chan struct{}), release: make(chan struct{}), value: started.Value}
			done := make(chan error, 1)
			go func() {
				done <- s.SiteCatalog.CompleteDomainVerificationFor(context.Background(), p.TenantID, "example", "primary.example.test", started.Value, p.UserID, &proof, lookup)
			}()
			select {
			case <-lookup.entered:
			case <-time.After(3 * time.Second):
				t.Fatal("DNS not entered")
			}
			if revoke {
				if err := store.Revoke(token); err != nil {
					close(lookup.release)
					t.Fatal(err)
				}
			} else {
				if _, err := s.SiteCatalog.BeginDomainVerificationFor(context.Background(), p.TenantID, "example", "primary.example.test", p.UserID, &proof); err != nil {
					close(lookup.release)
					t.Fatal(err)
				}
			}
			close(lookup.release)
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("stale proof accepted")
				}
			case <-time.After(3 * time.Second):
				t.Fatal("completion stuck")
			}
			var n int
			if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='site.domain_verified'").Scan(&n); err != nil || n != 0 {
				t.Fatal("stale verification audited")
			}
		})
	}
}

func TestDomainProofSessionExpiresDuringDNS(t *testing.T) {
	s, _, db := setupSiteAPI(t)
	if r := createTestSite(t, s.Handler(), secret, "example", "primary.example.test"); r.Code != 201 {
		t.Fatal(r.Body.String())
	}
	p := rbac.Principal{TenantID: "tenant-a", UserID: "u1", Role: rbac.Customer}
	store, err := auth.NewSQLiteSessionStore(db)
	if err != nil {
		t.Fatal(err)
	}
	token, _, _, err := store.Issue(p, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	proof, ok := store.ReadProof(token)
	if !ok {
		t.Fatal("proof")
	}
	started, err := s.SiteCatalog.BeginDomainVerificationFor(context.Background(), p.TenantID, "example", "primary.example.test", p.UserID, &proof)
	if err != nil {
		t.Fatal(err)
	}
	lookup := &heldProofLookup{entered: make(chan struct{}), release: make(chan struct{}), value: started.Value}
	done := make(chan error, 1)
	go func() {
		done <- s.SiteCatalog.CompleteDomainVerificationFor(context.Background(), p.TenantID, "example", "primary.example.test", started.Value, p.UserID, &proof, lookup)
	}()
	select {
	case <-lookup.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("DNS not entered")
	}
	if _, err := db.Exec("UPDATE sessions SET idle_deadline_unix=0 WHERE revoked_at IS NULL"); err != nil {
		close(lookup.release)
		t.Fatal(err)
	}
	close(lookup.release)
	select {
	case err := <-done:
		if err != sites.ErrSiteSessionDenied {
			t.Fatalf("expired session: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("completion stuck")
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='site.domain_verified'").Scan(&n); err != nil || n != 0 {
		t.Fatal("expired session verification audited")
	}
}
