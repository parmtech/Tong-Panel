package auth

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/parmtech/tong-panel/internal/database"
)

func emptyBootstrapDB(t *testing.T) (string, *sql.DB) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "owner.db")
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return path, db
}

func TestLocalOwnerBootstrapCreatesSingleAuditedArgon2idOwner(t *testing.T) {
	_, db := emptyBootstrapDB(t)
	password := []byte("correct horse battery staple 2026")
	result, err := BootstrapOwner(context.Background(), db, "first_owner", "First Hosting Tenant", password)
	if err != nil {
		t.Fatal(err)
	}
	if result.IdentityID == "" || result.TenantID == "" || result.Username != "first_owner" {
		t.Fatalf("bootstrap returned bad identity metadata: %+v", result)
	}
	var phc, role, auditKind, auditUser string
	if err := db.QueryRow("SELECT password_phc FROM identities WHERE id=?", result.IdentityID).Scan(&phc); err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(password, phc) || phc == string(password) {
		t.Fatal("bootstrap failed to store a protected Argon2id password hash")
	}
	if err := db.QueryRow("SELECT role FROM memberships WHERE tenant_id=? AND identity_id=?",
		result.TenantID, result.IdentityID).Scan(&role); err != nil || role != "owner" {
		t.Fatalf("first user was not tenant owner: role=%s err=%v", role, err)
	}
	if err := db.QueryRow("SELECT actor_kind,actor_identity_id FROM audit_events WHERE action='owner.bootstrap'").
		Scan(&auditKind, &auditUser); err != nil || auditKind != "user" || auditUser != result.IdentityID {
		t.Fatalf("first owner audit was not attributed: kind=%s actor=%s err=%v", auditKind, auditUser, err)
	}
	if _, err := BootstrapOwner(context.Background(), db, "other_owner", "Other", password); !errors.Is(err, ErrBootstrapUnavailable) {
		t.Fatalf("second owner bootstrap accepted: %v", err)
	}
	for _, table := range []string{"tenants", "identities", "memberships", "audit_events"} {
		var count int
		if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 1 {
			t.Fatalf("bootstrap modified %s on retry: %d %v", table, count, err)
		}
	}
}

func TestLocalOwnerBootstrapRefusesPartialAndInvalidInput(t *testing.T) {
	_, db := emptyBootstrapDB(t)
	password := []byte("valid bootstrapping passphrase")
	for _, tc := range []struct{ name, tenant string; password []byte }{
		{"Owner", "Tenant", password},
		{"root/admin", "Tenant", password},
		{"one", "   ", password},
		{"alice", "bad\ncompany", password},
		{"alice", "Tenant", []byte("short")},
	} {
		if _, err := BootstrapOwner(context.Background(), db, tc.name, tc.tenant, tc.password); err == nil {
			t.Fatalf("invalid owner input accepted: %q / %q", tc.name, tc.tenant)
		}
	}
	if _, err := db.Exec("INSERT INTO tenants(id,name) VALUES('existing','Imported')"); err != nil {
		t.Fatal(err)
	}
	if _, err := BootstrapOwner(context.Background(), db, "realowner", "Tenant", password); !errors.Is(err, ErrBootstrapUnavailable) {
		t.Fatalf("partial import allowed privileged first owner: %v", err)
	}
	var identities int
	if err := db.QueryRow("SELECT count(*) FROM identities").Scan(&identities); err != nil || identities != 0 {
		t.Fatalf("partial DB bootstrap wrote identity: %d %v", identities, err)
	}
}

func TestLocalOwnerBootstrapAuditFailureRollsBackAllTables(t *testing.T) {
	_, db := emptyBootstrapDB(t)
	if _, err := db.Exec("CREATE TRIGGER deny_first_owner BEFORE INSERT ON audit_events BEGIN SELECT RAISE(ABORT,'audit unavailable'); END"); err != nil {
		t.Fatal(err)
	}
	password := []byte("correct horse battery staple")
	if _, err := BootstrapOwner(context.Background(), db, "first", "Tenant", password); err == nil {
		t.Fatal("owner account created without mandatory audit")
	}
	for _, table := range []string{"tenants", "identities", "memberships", "audit_events"} {
		var count int
		if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("failed audit left partial %s: %d %v", table, count, err)
		}
	}
	if _, err := db.Exec("DROP TRIGGER deny_first_owner"); err != nil {
		t.Fatal(err)
	}
	if _, err := BootstrapOwner(context.Background(), db, "first", "Tenant", password); err != nil {
		t.Fatalf("retry after rollback failed: %v", err)
	}
}

func TestLocalOwnerBootstrapRaceAllowsOnlyOneOwner(t *testing.T) {
	path, db := emptyBootstrapDB(t)
	other, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	type outcome struct{ result BootstrapResult; err error }
	responses := make(chan outcome, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, handle := range []*sql.DB{db, other} {
		wg.Add(1)
		go func(id int, store *sql.DB) {
			defer wg.Done()
			<-start
			result, err := BootstrapOwner(context.Background(), store, "owner"+string(rune('a'+id)), "Company", []byte("long-enough-bootstrap-password"))
			responses <- outcome{result, err}
		}(i, handle)
	}
	close(start)
	wg.Wait()
	close(responses)
	successes, denied := 0, 0
	for item := range responses {
		if item.err == nil {
			successes++
		} else if errors.Is(item.err, ErrBootstrapUnavailable) {
			denied++
		} else {
			t.Fatalf("unexpected concurrency error: %v", item.err)
		}
	}
	if successes != 1 || denied != 1 {
		t.Fatalf("bootstrap race winners=%d refused=%d", successes, denied)
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM identities").Scan(&count); err != nil || count != 1 {
		t.Fatalf("bootstrap race persisted %d identities: %v", count, err)
	}
}
