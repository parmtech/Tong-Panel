package jobs

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/auth"
	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/rbac"
)

func jobSessionProof(t *testing.T, s *SQLiteStore) *auth.SessionReadProof {
	t.Helper()
	sessions, err := auth.NewSQLiteSessionStore(s.db)
	if err != nil {
		t.Fatal(err)
	}
	token, _, _, err := sessions.Issue(rbac.Principal{TenantID: "tenant-a", UserID: "user-a", Role: rbac.Owner}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	proof, ok := sessions.ReadProof(token)
	if !ok {
		t.Fatal("missing session read proof")
	}
	return &proof
}

func readJobSessionCase(s *SQLiteStore, ctx context.Context, operation, id string, proof *auth.SessionReadProof) ([]Job, string, error) {
	switch operation {
	case "list":
		items, err := s.ListForSession(ctx, "tenant-a", "user-a", proof)
		return items, "", err
	case "page":
		return s.ListPageForSession(ctx, "tenant-a", "user-a", "", 1, proof)
	default:
		job, err := s.GetForSession(ctx, "tenant-a", "user-a", id, proof)
		if job == (Job{}) {
			return nil, "", err
		}
		return []Job{job}, "", err
	}
}

func TestSQLiteSessionReadsPreserveOwnedScopeOrderAndPages(t *testing.T) {
	s, _ := openTestSQLiteStore(t)
	proof := jobSessionProof(t, s)
	var expectedIDs []string
	for index := range 5 {
		job, err := s.CreateFor("tenant-a", "user-a", KindNoop, fmt.Sprintf("session-owned-job-%d", index))
		if err != nil {
			t.Fatal(err)
		}
		expectedIDs = append(expectedIDs, job.ID)
	}
	if _, err := s.db.Exec("INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','user-b','customer')"); err != nil {
		t.Fatal(err)
	}
	other, err := s.CreateFor("tenant-a", "user-b", KindNoop, "session-other-user")
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := s.CreateFor("tenant-b", "user-b", KindNoop, "session-other-tenant")
	if err != nil {
		t.Fatal(err)
	}
	// Current auditors retain their existing own-job read permission even
	// when the session was originally issued to an owner.
	if _, err := s.db.Exec("UPDATE memberships SET role='auditor' WHERE identity_id='user-a'"); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"list", "page", "detail"} {
		want, wantNext, err := readJobSessionCase(s, context.Background(), operation, expectedIDs[0], nil)
		if err != nil {
			t.Fatal(err)
		}
		got, next, err := readJobSessionCase(s, context.Background(), operation, expectedIDs[0], proof)
		if err != nil || !reflect.DeepEqual(got, want) || next != wantNext {
			t.Fatalf("%s changed trusted result: err=%v", operation, err)
		}
	}
	sort.Strings(expectedIDs)
	var seen []string
	after := ""
	for turn := 0; turn < 4; turn++ {
		page, next, err := s.ListPageForSession(context.Background(), "tenant-a", "user-a", after, 2, proof)
		if err != nil || len(page) > 2 {
			t.Fatalf("page bound: len=%d err=%v", len(page), err)
		}
		for _, job := range page {
			if job.TenantID != "tenant-a" || job.RequestedBy != "user-a" || job.ID <= after {
				t.Fatal("session page crossed actor/tenant/cursor scope")
			}
			seen = append(seen, job.ID)
		}
		if next == "" {
			break
		}
		if len(page) != 2 || next != page[len(page)-1].ID {
			t.Fatal("invalid continuation")
		}
		after = next
	}
	if !reflect.DeepEqual(seen, expectedIDs) {
		t.Fatal("session pagination repeated or omitted jobs")
	}
	for _, id := range []string{other.ID, foreign.ID} {
		if got, err := s.GetForSession(context.Background(), "tenant-a", "user-a", id, proof); !errors.Is(err, ErrNotFound) || got != (Job{}) {
			t.Fatalf("foreign job returned data: %v", err)
		}
	}
	for _, scope := range []struct{ tenant, user string }{{"tenant-a", "user-b"}, {"tenant-b", "user-b"}} {
		if got, err := s.ListForSession(context.Background(), scope.tenant, scope.user, proof); !errors.Is(err, ErrSessionDenied) || got != nil {
			t.Fatalf("proof authorized another principal: %v", err)
		}
	}
	if got, next, err := s.ListPageForSession(context.Background(), "tenant-a", "user-a", strings.Repeat("f", 32), 2, proof); err != nil || got == nil || len(got) != 0 || next != "" {
		t.Fatalf("terminal page changed: %v", err)
	}
	if got, next, err := s.ListPageForSession(context.Background(), "tenant-a", "user-a", "invalid", 2, proof); !errors.Is(err, ErrInvalidPage) || got != nil || next != "" {
		t.Fatalf("invalid session cursor accepted: %v", err)
	}
}

func TestSQLiteSessionReadsDenyEmptyAndMissingResults(t *testing.T) {
	for _, change := range []string{
		"UPDATE sessions SET revoked_at='2026-01-01T00:00:00Z'",
		"UPDATE sessions SET expires_at='malformed'",
		"DELETE FROM sessions",
		"DELETE FROM memberships WHERE identity_id='user-a'",
		"UPDATE identities SET disabled=1 WHERE id='user-a'",
		"UPDATE tenants SET status='suspended' WHERE id='tenant-a'",
	} {
		t.Run(change, func(t *testing.T) {
			s, _ := openTestSQLiteStore(t)
			proof := jobSessionProof(t, s)
			for _, operation := range []string{"list", "page", "detail"} {
				got, next, err := readJobSessionCase(s, context.Background(), operation, strings.Repeat("a", 32), proof)
				if operation == "detail" {
					if !errors.Is(err, ErrNotFound) || got != nil {
						t.Fatalf("missing job: %v", err)
					}
				} else if err != nil || got == nil || len(got) != 0 || next != "" {
					t.Fatalf("empty authorized %s: %v", operation, err)
				}
			}
			if _, err := s.db.Exec(change); err != nil {
				t.Fatal(err)
			}
			for _, operation := range []string{"list", "page", "detail"} {
				got, next, err := readJobSessionCase(s, context.Background(), operation, strings.Repeat("a", 32), proof)
				if !errors.Is(err, ErrSessionDenied) || got != nil || next != "" {
					t.Fatalf("withdrawn session accepted empty %s: %v", operation, err)
				}
			}
		})
	}
}

func TestSQLiteSessionReadsCancelDuringConnectionWait(t *testing.T) {
	s, _ := openTestSQLiteStore(t)
	proof := jobSessionProof(t, s)
	s.db.SetMaxOpenConns(1)
	blocker, err := s.db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Close()
	for _, operation := range []string{"list", "page", "detail"} {
		t.Run(operation, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			waits := s.db.Stats().WaitCount
			done := make(chan error, 1)
			go func() {
				got, next, err := readJobSessionCase(s, ctx, operation, strings.Repeat("a", 32), proof)
				if got != nil || next != "" {
					err = errors.New("canceled read returned data")
				}
				done <- err
			}()
			deadline := time.Now().Add(3 * time.Second)
			for s.db.Stats().WaitCount == waits && time.Now().Before(deadline) {
				time.Sleep(time.Millisecond)
			}
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) || s.db.Stats().WaitCount == waits {
					t.Fatalf("session read did not cancel its connection wait: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("session read did not stop after cancellation")
			}
		})
	}
}

func TestSQLiteSessionReadExpiryClosesRowsBeforeCancelableWriter(t *testing.T) {
	for _, absolute := range []bool{true, false} {
		t.Run(fmt.Sprintf("absolute=%t", absolute), func(t *testing.T) {
			s, path := openTestSQLiteStore(t)
			proof := jobSessionProof(t, s)
			// The job clock must not preserve a session expired on its own clock.
			s.now = func() time.Time { return time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC) }
			if _, err := s.CreateFor("tenant-a", "user-a", KindNoop, "session-expiry-writer"); err != nil {
				t.Fatal(err)
			}
			query := "UPDATE sessions SET idle_deadline_unix=0"
			if absolute {
				query = "UPDATE sessions SET expires_at='2000-01-01T00:00:00Z'"
			}
			if _, err := s.db.Exec(query); err != nil {
				t.Fatal(err)
			}
			s.db.SetMaxOpenConns(1)
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
			got, err := s.ListForSession(ctx, "tenant-a", "user-a", proof)
			if !errors.Is(err, context.DeadlineExceeded) || got != nil || time.Since(started) > time.Second {
				t.Fatalf("expired-session writer ignored caller deadline or returned data: %v", err)
			}
			var revoked bool
			if err := s.db.QueryRow("SELECT revoked_at IS NOT NULL FROM sessions").Scan(&revoked); err != nil || revoked {
				t.Fatalf("canceled read changed revocation: %t %v", revoked, err)
			}
			if _, err := writer.ExecContext(context.Background(), "ROLLBACK"); err != nil {
				t.Fatal(err)
			}
			retryCtx, retryCancel := context.WithTimeout(context.Background(), time.Second)
			defer retryCancel()
			got, err = s.ListForSession(retryCtx, "tenant-a", "user-a", proof)
			if !errors.Is(err, ErrSessionDenied) || got != nil {
				t.Fatalf("retry did not deny expiry after releasing rows: %v", err)
			}
			if err := s.db.QueryRow("SELECT revoked_at IS NOT NULL FROM sessions").Scan(&revoked); err != nil || !revoked {
				t.Fatalf("retry did not permanently revoke: %t %v", revoked, err)
			}
		})
	}
}
