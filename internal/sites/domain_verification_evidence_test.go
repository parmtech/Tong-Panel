package sites

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/rbac"
)

func newEvidenceCatalog(t *testing.T) *Catalog {
	t.Helper()
	c := newVerificationCatalog(t)
	if _, err := c.db.Exec(`INSERT INTO tenants(id,name) VALUES('tenant-b','Tenant B');
		INSERT INTO identities(id,username) VALUES('user-a','alice'),('user-b','bob');
		INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','user-a','owner'),('tenant-b','user-b','owner');`); err != nil {
		t.Fatal(err)
	}
	return c
}

func beginEvidenceProof(t *testing.T, c *Catalog, domain string, authorized bool) DomainVerificationInstruction {
	t.Helper()
	var instruction DomainVerificationInstruction
	var err error
	if authorized {
		instruction, err = c.BeginDomainVerificationFor(context.Background(), "tenant-a", "site-a", domain, "user-a", nil)
	} else {
		instruction, err = c.BeginDomainVerification(context.Background(), "tenant-a", "site-a", domain)
	}
	if err != nil {
		t.Fatal(err)
	}
	return instruction
}

func completeEvidenceProof(c *Catalog, ctx context.Context, domain string, instruction DomainVerificationInstruction, authorized bool, lookup TXTLookup) error {
	if authorized {
		return c.CompleteDomainVerificationFor(ctx, "tenant-a", "site-a", domain, instruction.Value, "user-a", nil, lookup)
	}
	return c.CompleteDomainVerification(ctx, "tenant-a", "site-a", domain, instruction.Value, lookup)
}

func matchedEvidenceProof(instruction DomainVerificationInstruction) TXTLookup {
	return txtLookupFunc(func(context.Context, string) ([]string, error) { return []string{instruction.Value}, nil })
}

type storedDomainEvidence struct {
	state              string
	observed, verified sql.NullInt64
	id, token          sql.NullString
}

func readStoredDomainEvidence(t *testing.T, c *Catalog, domain string) storedDomainEvidence {
	t.Helper()
	var value storedDomainEvidence
	if err := c.db.QueryRow(`SELECT ownership_state,ownership_observed_at,ownership_verified_at,ownership_verification_id,verification_token_sha256
		FROM site_domains WHERE domain=?`, domain).Scan(&value.state, &value.observed, &value.verified, &value.id, &value.token); err != nil {
		t.Fatal(err)
	}
	return value
}

func assertNoDomainEvidence(t *testing.T, c *Catalog) {
	t.Helper()
	value := readStoredDomainEvidence(t, c, "example.com")
	if value.state != "pending" || value.observed.Valid || value.verified.Valid || value.id.Valid || !value.token.Valid {
		t.Fatal("failed proof changed pending state or created fresh evidence")
	}
}

func TestDomainEvidenceRecordsSuccessfulProofOnlyAndSurvivesReplay(t *testing.T) {
	for _, authorized := range []bool{false, true} {
		t.Run(map[bool]string{false: "trusted", true: "authorized"}[authorized], func(t *testing.T) {
			c := newEvidenceCatalog(t)
			instruction := beginEvidenceProof(t, c, "example.com", authorized)
			assertNoDomainEvidence(t, c)
			if err := completeEvidenceProof(c, context.Background(), "example.com", instruction, authorized, matchedEvidenceProof(instruction)); err != nil {
				t.Fatal(err)
			}
			stored := readStoredDomainEvidence(t, c, "example.com")
			if stored.state != "verified" || !stored.observed.Valid || stored.observed.Int64 != 2000 || !stored.verified.Valid || stored.verified.Int64 != 2000 ||
				!stored.id.Valid || !validVerificationID(stored.id.String) || stored.token.Valid || strings.Contains(instruction.Value, stored.id.String) {
				t.Fatal("successful proof did not retain bounded server-authored evidence")
			}
			if err := completeEvidenceProof(c, context.Background(), "example.com", instruction, authorized, matchedEvidenceProof(instruction)); !errors.Is(err, ErrDomainVerificationState) {
				t.Fatalf("replay: %v", err)
			}
			if got := readStoredDomainEvidence(t, c, "example.com"); got != stored {
				t.Fatal("replay refreshed or replaced successful evidence")
			}
			view, err := c.DomainVerificationEvidenceFor(context.Background(), "tenant-a", "site-a", "user-a", nil, time.Hour)
			if err != nil || view.TenantID != "tenant-a" || view.SiteID != "site-a" || len(view.Domains) != 1 || view.Domains[0].VerificationID != stored.id.String {
				t.Fatalf("evidence snapshot: %v", err)
			}
			view.Domains[0].VerificationID = "modified"
			again, err := c.DomainVerificationEvidenceFor(context.Background(), "tenant-a", "site-a", "user-a", nil, time.Hour)
			if err != nil || again.Domains[0].VerificationID != stored.id.String {
				t.Fatal("caller mutated persisted evidence")
			}
		})
	}
}

func TestDomainEvidenceFreshnessBoundariesAndInvalidStoredEvidence(t *testing.T) {
	for _, tc := range []struct {
		name     string
		observed any
		verified any
		id       any
		now      time.Time
		maxAge   time.Duration
		want     error
	}{
		{"fresh", int64(2000), int64(2000), strings.Repeat("a", 32), time.Unix(2000, 999999999), time.Second, nil},
		{"at boundary", int64(2000), int64(2000), strings.Repeat("a", 32), time.Unix(2001, 0), time.Second, ErrDomainEvidenceUnavailable},
		{"stale", int64(2000), int64(2000), strings.Repeat("a", 32), time.Unix(2002, 0), time.Second, ErrDomainEvidenceUnavailable},
		{"legacy unknown", nil, nil, nil, time.Unix(2000, 0), time.Hour, ErrDomainEvidenceUnavailable},
		{"future observation", int64(2001), int64(2001), strings.Repeat("a", 32), time.Unix(2000, 0), time.Hour, ErrDomainEvidenceUnavailable},
		{"future commit", int64(1999), int64(2001), strings.Repeat("a", 32), time.Unix(2000, 0), time.Hour, ErrDomainEvidenceUnavailable},
		{"invalid ordering", int64(2000), int64(1999), strings.Repeat("a", 32), time.Unix(2000, 0), time.Hour, ErrDomainEvidenceUnavailable},
		{"zero observation", int64(0), int64(2000), strings.Repeat("a", 32), time.Unix(2000, 0), time.Hour, ErrDomainEvidenceUnavailable},
		{"negative observation", int64(-1), int64(2000), strings.Repeat("a", 32), time.Unix(2000, 0), time.Hour, ErrDomainEvidenceUnavailable},
		{"missing observation", nil, int64(2000), strings.Repeat("a", 32), time.Unix(2000, 0), time.Hour, ErrDomainEvidenceUnavailable},
		{"missing receipt", int64(2000), int64(2000), nil, time.Unix(2000, 0), time.Hour, ErrDomainEvidenceUnavailable},
		{"malformed receipt", int64(2000), int64(2000), strings.Repeat("G", 32), time.Unix(2000, 0), time.Hour, ErrDomainEvidenceUnavailable},
		{"empty policy", int64(2000), int64(2000), strings.Repeat("a", 32), time.Unix(2000, 0), 0, ErrInvalidSpec},
		{"negative policy", int64(2000), int64(2000), strings.Repeat("a", 32), time.Unix(2000, 0), -time.Second, ErrInvalidSpec},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newEvidenceCatalog(t)
			// Deliberately corrupt this disposable fixture to verify reader defense
			// independently of the migration's write-time constraints.
			if _, err := c.db.Exec("DROP TRIGGER site_domains_evidence_state_update; PRAGMA ignore_check_constraints=ON"); err != nil {
				t.Fatal(err)
			}
			if _, err := c.db.Exec("UPDATE site_domains SET ownership_state='verified',ownership_observed_at=?,ownership_verified_at=?,ownership_verification_id=? WHERE domain='example.com'", tc.observed, tc.verified, tc.id); err != nil {
				t.Fatal(err)
			}
			c.now = func() time.Time { return tc.now }
			view, err := c.DomainVerificationEvidenceFor(context.Background(), "tenant-a", "site-a", "user-a", nil, tc.maxAge)
			if !errors.Is(err, tc.want) {
				t.Fatalf("freshness result: %v, want %v", err, tc.want)
			}
			if tc.want != nil && !reflect.DeepEqual(view, DomainVerificationEvidence{}) {
				t.Fatal("rejected evidence returned partial domain data")
			}
		})
	}
}

func TestDomainEvidenceAliasScopeAndRecreatedClaimHasNewID(t *testing.T) {
	c := newEvidenceCatalog(t)
	ctx := context.Background()
	for _, domain := range []string{"z.example.test", "pending.example.test"} {
		if _, err := c.AddAliasDomain(ctx, "tenant-a", "site-a", domain); err != nil {
			t.Fatal(err)
		}
	}
	for _, domain := range []string{"example.com", "z.example.test"} {
		instruction := beginEvidenceProof(t, c, domain, true)
		if err := completeEvidenceProof(c, ctx, domain, instruction, true, matchedEvidenceProof(instruction)); err != nil {
			t.Fatal(err)
		}
	}
	view, err := c.DomainVerificationEvidenceFor(ctx, "tenant-a", "site-a", "user-a", nil, time.Hour)
	if err != nil || len(view.Domains) != 2 || view.Domains[0].Domain != "example.com" || view.Domains[1].Domain != "z.example.test" {
		t.Fatal("verified alias selection changed or pending alias leaked")
	}
	oldID := view.Domains[1].VerificationID
	if err := c.RemoveAliasDomainFor(ctx, "tenant-a", "site-a", "z.example.test", "user-a", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AddAliasDomainFor(ctx, "tenant-a", "site-a", "z.example.test", "user-a", nil); err != nil {
		t.Fatal(err)
	}
	if got := readStoredDomainEvidence(t, c, "z.example.test"); got.id.Valid || got.observed.Valid || got.verified.Valid {
		t.Fatal("recreated claim inherited previous evidence")
	}
	instruction := beginEvidenceProof(t, c, "z.example.test", true)
	if err := completeEvidenceProof(c, ctx, "z.example.test", instruction, true, matchedEvidenceProof(instruction)); err != nil {
		t.Fatal(err)
	}
	if got := readStoredDomainEvidence(t, c, "z.example.test"); !got.id.Valid || got.id.String == oldID {
		t.Fatal("recreated claim reused verification identity")
	}
	if _, err := c.db.Exec("UPDATE site_domains SET ownership_state='verified' WHERE domain='pending.example.test'"); err != nil {
		t.Fatal(err)
	}
	if view, err := c.DomainVerificationEvidenceFor(ctx, "tenant-a", "site-a", "user-a", nil, time.Hour); !errors.Is(err, ErrDomainEvidenceUnavailable) || len(view.Domains) != 0 {
		t.Fatal("legacy verified alias was silently accepted or excluded")
	}
	if _, err := c.ACMEHTTP01PreflightFor(ctx, "tenant-a", "site-a", "user-a", nil); err != nil {
		t.Fatal("informational public preflight unexpectedly requires new evidence")
	}
}

func TestDomainEvidenceUsesCurrentReadAuthorityAndSession(t *testing.T) {
	c := newEvidenceCatalog(t)
	ctx := context.Background()
	instruction := beginEvidenceProof(t, c, "example.com", true)
	if err := completeEvidenceProof(c, ctx, "example.com", instruction, true, matchedEvidenceProof(instruction)); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		tenant, user string
		want         error
	}{
		{"tenant-a", "user-b", ErrSiteReadDenied},
		{"tenant-b", "user-b", ErrSiteNotFound},
	} {
		if view, err := c.DomainVerificationEvidenceFor(ctx, tc.tenant, "site-a", tc.user, nil, time.Hour); !errors.Is(err, tc.want) || len(view.Domains) != 0 {
			t.Fatal("cross-tenant evidence was disclosed")
		}
	}
	// Reading evidence preserves sites:read policy; it is not a new operation grant.
	for _, role := range []string{"owner", "admin", "reseller", "customer", "auditor"} {
		if _, err := c.db.Exec("UPDATE memberships SET role=? WHERE tenant_id='tenant-a' AND identity_id='user-a'", role); err != nil {
			t.Fatal(err)
		}
		if _, err := c.DomainVerificationEvidenceFor(ctx, "tenant-a", "site-a", "user-a", nil, time.Hour); err != nil {
			t.Fatal("existing read role denied evidence")
		}
	}
	sessions, err := auth.NewSQLiteSessionStore(c.db)
	if err != nil {
		t.Fatal(err)
	}
	token, _, _, err := sessions.Issue(rbac.Principal{TenantID: "tenant-a", UserID: "user-a", Role: rbac.Auditor}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	proof, ok := sessions.ReadProof(token)
	if !ok {
		t.Fatal("missing session proof")
	}
	if _, err := c.DomainVerificationEvidenceFor(ctx, "tenant-a", "site-a", "user-a", &proof, time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Revoke(token); err != nil {
		t.Fatal(err)
	}
	if view, err := c.DomainVerificationEvidenceFor(ctx, "tenant-a", "site-a", "user-a", &proof, time.Hour); !errors.Is(err, ErrSiteSessionDenied) || len(view.Domains) != 0 {
		t.Fatal("revoked session disclosed evidence")
	}
	if _, err := c.db.Exec("UPDATE identities SET disabled=1 WHERE id='user-a'"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DomainVerificationEvidenceFor(ctx, "tenant-a", "site-a", "user-a", nil, time.Hour); !errors.Is(err, ErrSiteReadDenied) {
		t.Fatal("disabled identity read evidence")
	}
}

func TestDomainEvidenceReadDenialsAndSessionClock(t *testing.T) {
	for _, mode := range []string{"membership removed", "unknown role", "suspended tenant", "absolute expiry", "idle expiry", "malformed expiry", "foreign pool", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			c := newEvidenceCatalog(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			// Trusted setup has no user audit row retaining the membership;
			// deletion below must exercise current authority, not an audit FK.
			instruction := beginEvidenceProof(t, c, "example.com", false)
			if err := completeEvidenceProof(c, ctx, "example.com", instruction, false, matchedEvidenceProof(instruction)); err != nil {
				t.Fatal(err)
			}
			stored := readStoredDomainEvidence(t, c, "example.com")
			sessions, err := auth.NewSQLiteSessionStore(c.db)
			if err != nil {
				t.Fatal(err)
			}
			token, _, _, err := sessions.Issue(rbac.Principal{TenantID: "tenant-a", UserID: "user-a", Role: rbac.Owner}, time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			proof, ok := sessions.ReadProof(token)
			if !ok {
				t.Fatal("missing session proof")
			}
			want := ErrSiteSessionDenied
			switch mode {
			case "membership removed":
				_, err = c.db.Exec("DELETE FROM memberships WHERE tenant_id='tenant-a' AND identity_id='user-a'")
			case "unknown role":
				_, err = c.db.Exec("PRAGMA ignore_check_constraints=ON; UPDATE memberships SET role='unknown' WHERE tenant_id='tenant-a' AND identity_id='user-a'")
			case "suspended tenant":
				_, err = c.db.Exec("UPDATE tenants SET status='suspended' WHERE id='tenant-a'")
			case "absolute expiry":
				// The catalog clock is Unix 2000. This expiry lies after that
				// clock but before the session store's current trusted clock.
				_, err = c.db.Exec("UPDATE sessions SET expires_at='2000-01-01T00:00:00Z'")
			case "idle expiry":
				_, err = c.db.Exec("UPDATE sessions SET idle_deadline_unix=2001")
			case "malformed expiry":
				_, err = c.db.Exec("UPDATE sessions SET expires_at='invalid-time'")
			case "foreign pool":
				other := newEvidenceCatalog(t)
				foreign, openErr := auth.NewSQLiteSessionStore(other.db)
				if openErr != nil {
					t.Fatal(openErr)
				}
				proof, ok = foreign.ReadProof(token)
				if !ok {
					t.Fatal("missing foreign session proof")
				}
				want = ErrCatalogUnavailable
			case "canceled":
				cancel()
				want = context.Canceled
			}
			if err != nil {
				t.Fatal(err)
			}
			view, err := c.DomainVerificationEvidenceFor(ctx, "tenant-a", "site-a", "user-a", &proof, time.Hour)
			if !errors.Is(err, want) || !reflect.DeepEqual(view, DomainVerificationEvidence{}) {
				t.Fatalf("session denial: %v, want %v with no data", err, want)
			}
			if mode == "membership removed" || mode == "unknown role" || mode == "suspended tenant" {
				view, err := c.DomainVerificationEvidenceFor(ctx, "tenant-a", "site-a", "user-a", nil, time.Hour)
				if !errors.Is(err, ErrSiteReadDenied) || !reflect.DeepEqual(view, DomainVerificationEvidence{}) {
					t.Fatal("membership-only read retained stale authority")
				}
			}
			if mode == "absolute expiry" || mode == "idle expiry" {
				var revoked bool
				if err := c.db.QueryRow("SELECT revoked_at IS NOT NULL FROM sessions").Scan(&revoked); err != nil || !revoked {
					t.Fatal("observed expired session was not permanently revoked")
				}
			}
			if readStoredDomainEvidence(t, c, "example.com") != stored {
				t.Fatal("rejected read changed stored evidence")
			}
		})
	}
}

func TestDomainEvidenceExpiredSessionReadCancelsDuringWriterWait(t *testing.T) {
	for _, absolute := range []bool{true, false} {
		t.Run(map[bool]string{true: "absolute", false: "idle"}[absolute], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "evidence.db")
			c := openCatalog(t, path)
			c.now = func() time.Time { return time.Unix(2000, 0) }
			if _, err := c.db.Exec("INSERT INTO identities(id,username) VALUES('user-a','alice'); INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','user-a','owner')"); err != nil {
				t.Fatal(err)
			}
			if err := c.Create(context.Background(), SiteSpec{TenantID: "tenant-a", SiteID: "site-a", PrimaryDomain: "example.com", DocumentRoot: "sites/site-a/public"}); err != nil {
				t.Fatal(err)
			}
			instruction := beginEvidenceProof(t, c, "example.com", false)
			if err := completeEvidenceProof(c, context.Background(), "example.com", instruction, false, matchedEvidenceProof(instruction)); err != nil {
				t.Fatal(err)
			}
			stored := readStoredDomainEvidence(t, c, "example.com")
			sessions, err := auth.NewSQLiteSessionStore(c.db)
			if err != nil {
				t.Fatal(err)
			}
			token, _, _, err := sessions.Issue(rbac.Principal{TenantID: "tenant-a", UserID: "user-a", Role: rbac.Owner}, time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			proof, ok := sessions.ReadProof(token)
			if !ok {
				t.Fatal("missing proof")
			}
			digest := proof.Digest()
			if absolute {
				_, err = c.db.Exec("UPDATE sessions SET expires_at='2000-01-01T00:00:00Z' WHERE token_sha256=?", digest[:])
			} else {
				_, err = c.db.Exec("UPDATE sessions SET idle_deadline_unix=2001 WHERE token_sha256=?", digest[:])
			}
			if err != nil {
				t.Fatal(err)
			}
			peer, err := database.Open(context.Background(), path)
			if err != nil {
				t.Fatal(err)
			}
			defer peer.Close()
			writer, err := peer.Conn(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer writer.Close()
			if _, err := writer.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
				t.Fatal(err)
			}
			defer writer.ExecContext(context.Background(), "ROLLBACK")
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			started := time.Now()
			view, err := c.DomainVerificationEvidenceFor(ctx, "tenant-a", "site-a", "user-a", &proof, time.Hour)
			if !errors.Is(err, context.DeadlineExceeded) || !reflect.DeepEqual(view, DomainVerificationEvidence{}) {
				t.Fatalf("canceled expired-session read: %v, want deadline with no evidence", err)
			}
			if elapsed := time.Since(started); elapsed > time.Second {
				t.Fatalf("expired-session read cancellation took too long: %v", elapsed)
			}
			var revoked bool
			if err := c.db.QueryRow("SELECT revoked_at IS NOT NULL FROM sessions WHERE token_sha256=?", digest[:]).Scan(&revoked); err != nil || revoked {
				t.Fatalf("canceled read changed session: revoked=%t err=%v", revoked, err)
			}
			var busyTimeout int
			if err := c.db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil || busyTimeout != 5000 {
				t.Fatalf("busy_timeout=%d err=%v, want restored 5000ms", busyTimeout, err)
			}
			if _, err := writer.ExecContext(context.Background(), "ROLLBACK"); err != nil {
				t.Fatal(err)
			}
			view, err = c.DomainVerificationEvidenceFor(context.Background(), "tenant-a", "site-a", "user-a", &proof, time.Hour)
			if !errors.Is(err, ErrSiteSessionDenied) || !reflect.DeepEqual(view, DomainVerificationEvidence{}) {
				t.Fatalf("retried expired-session read: %v, want session denial with no evidence", err)
			}
			if err := c.db.QueryRow("SELECT revoked_at IS NOT NULL FROM sessions WHERE token_sha256=?", digest[:]).Scan(&revoked); err != nil || !revoked {
				t.Fatalf("retried read did not permanently revoke session: revoked=%t err=%v", revoked, err)
			}
			if readStoredDomainEvidence(t, c, "example.com") != stored {
				t.Fatal("expired-session read changed domain evidence")
			}
		})
	}
}

func TestDomainEvidenceCompletionRechecksSessionAfterDNS(t *testing.T) {
	for _, mode := range []string{"revoked", "expired", "membership removed"} {
		t.Run(mode, func(t *testing.T) {
			c := newEvidenceCatalog(t)
			instruction := beginEvidenceProof(t, c, "example.com", false)
			sessions, err := auth.NewSQLiteSessionStore(c.db)
			if err != nil {
				t.Fatal(err)
			}
			token, _, _, err := sessions.Issue(rbac.Principal{TenantID: "tenant-a", UserID: "user-a", Role: rbac.Owner}, time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			proof, ok := sessions.ReadProof(token)
			if !ok {
				t.Fatal("missing session proof")
			}
			want := ErrSiteSessionDenied
			lookup := txtLookupFunc(func(context.Context, string) ([]string, error) {
				switch mode {
				case "revoked":
					err = sessions.Revoke(token)
				case "expired":
					_, err = c.db.Exec("UPDATE sessions SET expires_at='2000-01-01T00:00:00Z'")
				case "membership removed":
					_, err = c.db.Exec("DELETE FROM memberships WHERE tenant_id='tenant-a' AND identity_id='user-a'")
					want = ErrSiteReadDenied
				}
				return []string{instruction.Value}, err
			})
			err = c.CompleteDomainVerificationFor(context.Background(), "tenant-a", "site-a", "example.com", instruction.Value, "user-a", &proof, lookup)
			if !errors.Is(err, want) {
				t.Fatalf("changed session: %v, want %v", err, want)
			}
			assertNoDomainEvidence(t, c)
		})
	}
}

func TestDomainEvidenceAuditFailureAndAuthorizationChangeRollBackStamp(t *testing.T) {
	for _, mode := range []string{"audit failure", "role revoked during DNS", "canceled during DNS", "DNS failure", "DNS mismatch", "expired during DNS", "replaced during DNS", "clock rollback"} {
		t.Run(mode, func(t *testing.T) {
			c := newEvidenceCatalog(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			instruction := beginEvidenceProof(t, c, "example.com", true)
			if mode == "audit failure" {
				if _, err := c.db.Exec(`CREATE TRIGGER fail_evidence_audit BEFORE INSERT ON audit_events WHEN NEW.action='site.domain_verified' BEGIN SELECT RAISE(ABORT,'synthetic audit failure'); END`); err != nil {
					t.Fatal(err)
				}
			}
			lookup := txtLookupFunc(func(context.Context, string) ([]string, error) {
				switch mode {
				case "role revoked during DNS":
					if _, err := c.db.Exec("UPDATE memberships SET role='auditor' WHERE tenant_id='tenant-a' AND identity_id='user-a'"); err != nil {
						t.Fatal(err)
					}
				case "canceled during DNS":
					cancel()
				case "DNS failure":
					return nil, errors.New("synthetic lookup failure")
				case "DNS mismatch":
					return []string{"unrelated-record"}, nil
				case "expired during DNS":
					c.now = func() time.Time { return time.Unix(instruction.ExpiresAt, 0) }
				case "replaced during DNS":
					beginEvidenceProof(t, c, "example.com", true)
				case "clock rollback":
					calls := 0
					c.now = func() time.Time {
						calls++
						if calls == 1 {
							return time.Unix(2000, 0)
						}
						return time.Unix(1999, 0)
					}
				}
				return []string{instruction.Value}, nil
			})
			err := completeEvidenceProof(c, ctx, "example.com", instruction, true, lookup)
			if err == nil {
				t.Fatal("failed completion unexpectedly succeeded")
			}
			want := map[string]error{
				"role revoked during DNS": ErrSiteReadDenied,
				"canceled during DNS":     context.Canceled,
				"DNS mismatch":            ErrDomainProofMismatch,
				"expired during DNS":      ErrDomainVerificationExpired,
				"replaced during DNS":     ErrDomainVerificationExpired,
				"clock rollback":          ErrDomainEvidenceUnavailable,
			}[mode]
			if want != nil && !errors.Is(err, want) {
				t.Fatalf("completion: %v, want %v", err, want)
			}
			assertNoDomainEvidence(t, c)
			var audits int
			if err := c.db.QueryRow("SELECT count(*) FROM audit_events WHERE action='site.domain_verified'").Scan(&audits); err != nil || audits != 0 {
				t.Fatal("failed completion retained success audit")
			}
		})
	}
}

func TestDomainEvidenceWriterDelayDoesNotFreshenObservation(t *testing.T) {
	for _, authorized := range []bool{false, true} {
		t.Run(map[bool]string{false: "trusted", true: "authorized"}[authorized], func(t *testing.T) {
			runDomainEvidenceWriterWait(t, authorized, false)
		})
	}
}

func TestDomainEvidenceWriterCancellationDoesNotStamp(t *testing.T) {
	for _, authorized := range []bool{false, true} {
		t.Run(map[bool]string{false: "trusted", true: "authorized"}[authorized], func(t *testing.T) {
			runDomainEvidenceWriterWait(t, authorized, true)
		})
	}
}

func runDomainEvidenceWriterWait(t *testing.T, authorized, cancelWrite bool) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "evidence.db")
	c := openCatalog(t, path)
	if _, err := c.db.Exec("INSERT INTO identities(id,username) VALUES('user-a','alice'); INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','user-a','owner')"); err != nil {
		t.Fatal(err)
	}
	var clock atomic.Int64
	clock.Store(2000)
	c.now = func() time.Time { return time.Unix(clock.Load(), 0) }
	if err := c.Create(context.Background(), SiteSpec{TenantID: "tenant-a", SiteID: "site-a", PrimaryDomain: "example.com", DocumentRoot: "sites/site-a/public"}); err != nil {
		t.Fatal(err)
	}
	instruction := beginEvidenceProof(t, c, "example.com", authorized)
	peer, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	writer, err := peer.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	defer writer.ExecContext(context.Background(), "ROLLBACK")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	locked := make(chan struct{})
	lookup := txtLookupFunc(func(context.Context, string) ([]string, error) {
		if _, err := writer.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
			return nil, err
		}
		close(locked)
		return []string{instruction.Value}, nil
	})
	result := make(chan error, 1)
	go func() { result <- completeEvidenceProof(c, ctx, "example.com", instruction, authorized, lookup) }()
	select {
	case <-locked:
	case err := <-result:
		t.Fatalf("proof stopped before DNS observation: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("proof did not reach DNS observation")
	}
	deadline := time.Now().Add(2 * time.Second)
	for c.db.Stats().InUse == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if c.db.Stats().InUse == 0 {
		t.Fatal("proof did not reserve final update connection")
	}
	clock.Store(2010)
	if cancelWrite {
		cancel()
		select {
		case err := <-result:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("writer cancellation: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("writer wait ignored cancellation")
		}
	}
	if _, err := writer.ExecContext(context.Background(), "ROLLBACK"); err != nil {
		t.Fatal(err)
	}
	if cancelWrite {
		assertNoDomainEvidence(t, c)
	} else {
		select {
		case err := <-result:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("proof did not finish after writer release")
		}
		stored := readStoredDomainEvidence(t, c, "example.com")
		if stored.observed.Int64 != 2000 || stored.verified.Int64 != 2010 {
			t.Fatal("writer delay changed DNS observation time")
		}
		if _, err := c.DomainVerificationEvidenceFor(context.Background(), "tenant-a", "site-a", "user-a", nil, 10*time.Second); !errors.Is(err, ErrDomainEvidenceUnavailable) {
			t.Fatal("delayed commit refreshed expired observation")
		}
	}
	var busyTimeout int
	if err := c.db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil || busyTimeout != 5000 {
		t.Fatal("domain proof writer did not restore SQLite timeout")
	}
}
