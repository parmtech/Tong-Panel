package jobs

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/parmtech/tong-panel/internal/database"
)

func openTestSQLiteStore(t *testing.T) (*SQLiteStore, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "panel.db")
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`INSERT INTO tenants(id,name) VALUES('tenant-a','A'),('tenant-b','B');
		INSERT INTO identities(id,username) VALUES('user-a','alice'),('user-b','bob');
		INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','user-a','owner'),('tenant-b','user-b','owner');`)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	return store, path
}

func TestSQLiteCreateIdempotencyScopeAndTransactions(t *testing.T) {
	store, path := openTestSQLiteStore(t)
	created, err := store.CreateFor("tenant-a", "user-a", KindNoop, "request-0001")
	if err != nil {
		t.Fatal(err)
	}
	retry, err := store.CreateFor("tenant-a", "user-a", KindNoop, "request-0001")
	if err != nil || retry.ID != created.ID {
		t.Fatalf("retry=%+v err=%v", retry, err)
	}
	if _, err := store.CreateFor("tenant-a", "user-a", "different.work", "request-0001"); err == nil {
		t.Fatal("unsupported work kind accepted")
	}
	if _, err := store.CreateFor("tenant-a", "user-b", KindNoop, "request-0001"); err != ErrConflict {
		t.Fatalf("different requester should conflict: %v", err)
	}
	if _, err := store.CreateFor("tenant-b", "user-b", KindNoop, "request-0001"); err != nil {
		t.Fatalf("idempotency key should be tenant-scoped: %v", err)
	}
	if _, err := store.CreateFor("tenant-a", "missing-user", KindNoop, "request-0002"); err == nil {
		t.Fatal("missing membership accepted")
	}
	if _, err := store.db.Exec("UPDATE tenants SET status='suspended' WHERE id='tenant-a'"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateFor("tenant-a", "user-a", KindNoop, "request-0003"); err == nil {
		t.Fatal("job created for suspended tenant")
	}
	if _, err := store.db.Exec("UPDATE tenants SET status='active' WHERE id='tenant-a'; UPDATE identities SET disabled=1 WHERE id='user-a'"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateFor("tenant-a", "user-a", KindNoop, "request-0004"); err == nil {
		t.Fatal("job created by disabled identity")
	}
	if _, err := store.db.Exec("UPDATE identities SET disabled=0 WHERE id='user-a'"); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := store.db.QueryRow("SELECT count(*) FROM jobs WHERE tenant_id='tenant-a'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("failed insertion left a row: count=%d err=%v", count, err)
	}
	if _, err := store.Get("tenant-b", created.ID); err != ErrNotFound {
		t.Fatalf("cross-tenant lookup was not hidden: %v", err)
	}
	other, err := store.List("tenant-b")
	if err != nil || len(other) != 1 {
		t.Fatalf("tenant-b list=%v err=%v", other, err)
	}
	missing, err := store.List("missing")
	if err != nil || len(missing) != 0 {
		t.Fatal("tenant-filtered list mismatch")
	}
	if _, err := store.Transition("tenant-a", created.ID, Done); err == nil {
		t.Fatal("invalid transition accepted")
	}
	if _, err := store.Transition("tenant-a", created.ID, Running); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Transition("tenant-a", created.ID, Done); err != nil {
		t.Fatal(err)
	}
	if err := store.db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	reopened, _ := NewSQLiteStore(db)
	got, err := reopened.Get("tenant-a", created.ID)
	if err != nil || got.State != Done {
		t.Fatalf("restart lost committed state: %+v err=%v", got, err)
	}
}

func TestSQLiteConcurrentIdempotentCreate(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	const workers = 24
	var wg sync.WaitGroup
	ids := make(chan string, workers)
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			j, err := store.CreateFor("tenant-a", "user-a", KindNoop, "request-race-1")
			if err != nil {
				errs <- err
				return
			}
			ids <- j.ID
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	want := ""
	for id := range ids {
		if want == "" {
			want = id
		}
		if id != want {
			t.Fatalf("concurrent retries produced multiple jobs: %s != %s", id, want)
		}
	}
	list, err := store.List("tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	if got := len(list); got != 1 {
		t.Fatalf("stored %d jobs, want exactly one", got)
	}
}
