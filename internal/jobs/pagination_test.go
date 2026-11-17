package jobs

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
)

func pageID(number int) string { return fmt.Sprintf("%032x", number) }

func TestJournalPagesBoundedOrderedAndTenantScoped(t *testing.T) {
	store := &Store{}
	for index := 205; index > 0; index-- {
		store.items = append(store.items, Job{ID: pageID(index), TenantID: "tenant-a", Kind: KindNoop})
	}
	store.items = append(store.items, Job{ID: pageID(0), TenantID: "tenant-b"})
	var ids []string
	after := ""
	for {
		items, next, err := store.ListPage(context.Background(), "tenant-a", after, 100)
		if err != nil || len(items) > 100 {
			t.Fatal("invalid journal page")
		}
		for _, job := range items {
			if job.TenantID != "tenant-a" || job.ID <= after {
				t.Fatal("journal page escaped scope or cursor")
			}
			ids = append(ids, job.ID)
		}
		if next == "" {
			break
		}
		if next != items[len(items)-1].ID {
			t.Fatal("cursor is not final returned ID")
		}
		after = next
	}
	if len(ids) != 205 {
		t.Fatal("journal paging lost history")
	}
	for index, id := range ids {
		if id != pageID(index+1) {
			t.Fatal("journal pagination duplicated or misordered history")
		}
	}
	if got := store.List("tenant-a"); len(got) != 205 || got[0].ID != pageID(205) {
		t.Fatal("paged reads changed legacy journal behavior")
	}
	items, next, err := store.ListPage(context.Background(), "missing", "", 10)
	if err != nil || items == nil || len(items) != 0 || next != "" {
		t.Fatal("empty journal page shape incorrect")
	}
}

func TestSQLitePagesOwnedHistoryAndCurrentReadAuthority(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	ctx := context.Background()
	if _, err := store.db.Exec("INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','user-b','owner')"); err != nil {
		t.Fatal(err)
	}
	for index := 107; index > 0; index-- {
		if _, err := store.db.Exec("INSERT INTO jobs(id,tenant_id,requested_by,kind,idempotency_key) VALUES(?,'tenant-a','user-a','validation.noop',?)", pageID(index), fmt.Sprintf("page-key-%04d", index)); err != nil {
			t.Fatal(err)
		}
	}
	for _, fixture := range []struct {
		tenant, user string
		id           int
	}{{"tenant-a", "user-b", 200}, {"tenant-b", "user-b", 201}} {
		if _, err := store.db.Exec("INSERT INTO jobs(id,tenant_id,requested_by,kind,idempotency_key) VALUES(?,?,?,'validation.noop',?)", pageID(fixture.id), fixture.tenant, fixture.user, fmt.Sprintf("foreign-key-%04d", fixture.id)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.db.Exec("UPDATE memberships SET role='auditor' WHERE tenant_id='tenant-a' AND identity_id='user-a'"); err != nil {
		t.Fatal(err)
	}
	var ids []string
	after := ""
	for {
		items, next, err := store.ListPageFor(ctx, "tenant-a", "user-a", after, 40)
		if err != nil || len(items) > 40 {
			t.Fatal("auditor owned page unavailable")
		}
		for _, job := range items {
			if job.RequestedBy != "user-a" || job.TenantID != "tenant-a" {
				t.Fatal("page leaked another requester or tenant")
			}
			ids = append(ids, job.ID)
		}
		if next == "" {
			break
		}
		after = next
	}
	if len(ids) != 107 {
		t.Fatal("owned pagination lost history")
	}
	for index, id := range ids {
		if id != pageID(index+1) {
			t.Fatal("owned pagination order or uniqueness incorrect")
		}
	}
	items, next, err := store.ListPageFor(ctx, "tenant-a", "user-a", pageID(999), 100)
	if err != nil || items == nil || len(items) != 0 || next != "" {
		t.Fatal("empty SQL page shape incorrect")
	}
	for _, change := range []string{"UPDATE identities SET disabled=1 WHERE id='user-a'", "UPDATE identities SET disabled=0 WHERE id='user-a';UPDATE tenants SET status='suspended' WHERE id='tenant-a'"} {
		if _, err := store.db.Exec(change); err != nil {
			t.Fatal(err)
		}
		if _, _, err := store.ListPageFor(ctx, "tenant-a", "user-a", "", 40); !errors.Is(err, ErrNotFound) {
			t.Fatal("inactive requester read accepted")
		}
	}
	if _, _, err := store.ListPageFor(ctx, "tenant-b", "user-a", "", 40); !errors.Is(err, ErrNotFound) {
		t.Fatal("missing tenant membership read accepted")
	}
	var attempts, audits int
	if err := store.db.QueryRow("SELECT (SELECT count(*) FROM job_attempts),(SELECT count(*) FROM audit_events)").Scan(&attempts, &audits); err != nil || attempts != 0 || audits != 0 {
		t.Fatal("pagination changed job ledgers")
	}
}

func TestPagesValidationCancellationAndErrors(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	journal := &Store{}
	for _, call := range []func(context.Context, string, int) ([]Job, string, error){
		func(ctx context.Context, after string, limit int) ([]Job, string, error) {
			return journal.ListPage(ctx, "tenant-a", after, limit)
		},
		func(ctx context.Context, after string, limit int) ([]Job, string, error) {
			return store.ListPageFor(ctx, "tenant-a", "user-a", after, limit)
		},
	} {
		for _, page := range []struct {
			after string
			limit int
		}{{"", 0}, {"", 101}, {"../other", 1}, {"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", 1}, {pageID(1) + "?x", 1}} {
			if _, _, err := call(context.Background(), page.after, page.limit); !errors.Is(err, ErrInvalidPage) {
				t.Fatal("invalid pagination accepted")
			}
		}
		if _, _, err := call(nil, "", 1); !errors.Is(err, ErrInvalidPage) {
			t.Fatal("nil context accepted")
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, _, err := call(ctx, "", 1); !errors.Is(err, context.Canceled) {
			t.Fatal("canceled request not preserved")
		}
	}
	if err := store.db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.ListPageFor(context.Background(), "tenant-a", "user-a", "", 1); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatal("closed database did not preserve backend failure distinct from missing authority")
	}
}

func TestJournalPagesDocumentConcurrentInsertBoundary(t *testing.T) {
	store := &Store{items: []Job{{ID: pageID(2), TenantID: "t"}, {ID: pageID(4), TenantID: "t"}, {ID: pageID(6), TenantID: "t"}}}
	_, next, err := store.ListPage(context.Background(), "t", "", 2)
	if err != nil || next != pageID(4) {
		t.Fatal("initial boundary incorrect")
	}
	store.items = append(store.items, Job{ID: pageID(3), TenantID: "t"}, Job{ID: pageID(5), TenantID: "t"})
	items, cursor, err := store.ListPage(context.Background(), "t", next, 2)
	if err != nil || len(items) != 2 || cursor != "" || !reflect.DeepEqual([]string{items[0].ID, items[1].ID}, []string{pageID(5), pageID(6)}) {
		t.Fatal("continuation crossed its ID boundary")
	}
	fresh, _, err := store.ListPage(context.Background(), "t", "", 100)
	if err != nil || len(fresh) != 5 {
		t.Fatal("fresh listing did not include earlier inserted row")
	}
}

func TestSQLitePagesKeepIDBoundaryAcrossIndependentWrites(t *testing.T) {
	store, path := openTestSQLiteStore(t)
	writer, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	insert := func(number int) {
		t.Helper()
		if _, err := writer.Exec("INSERT INTO jobs(id,tenant_id,requested_by,kind,idempotency_key) VALUES(?,'tenant-a','user-a','validation.noop',?)", pageID(number), fmt.Sprintf("boundary-key-%04d", number)); err != nil {
			t.Fatal(err)
		}
	}
	for _, number := range []int{2, 4, 6} {
		insert(number)
	}
	_, next, err := store.ListPageFor(context.Background(), "tenant-a", "user-a", "", 2)
	if err != nil || next != pageID(4) {
		t.Fatal("initial SQL boundary incorrect")
	}
	insert(3)
	insert(5)
	if _, err := writer.Exec("DELETE FROM jobs WHERE id=?", next); err != nil {
		t.Fatal(err)
	}
	items, cursor, err := store.ListPageFor(context.Background(), "tenant-a", "user-a", next, 2)
	if err != nil || len(items) != 2 || cursor != "" || items[0].ID != pageID(5) || items[1].ID != pageID(6) {
		t.Fatal("SQL continuation failed after boundary deletion or independent inserts")
	}
	fresh, _, err := store.ListPageFor(context.Background(), "tenant-a", "user-a", "", 100)
	if err != nil || len(fresh) != 4 || fresh[1].ID != pageID(3) {
		t.Fatal("refresh did not expose earlier inserted job")
	}
}

func TestSQLitePageRequestContextBoundsConnectionWait(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	connection, err := store.db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	before := store.db.Stats().WaitCount
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, _, err := store.ListPageFor(ctx, "tenant-a", "user-a", "", 10); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("page connection wait ignored request deadline")
	}
	if store.db.Stats().WaitCount <= before {
		t.Fatal("fixture did not exercise a blocked connection wait")
	}
	if store.ctx.Err() != nil {
		t.Fatal("page request mutated shared store context")
	}
	if err := connection.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.ListPageFor(context.Background(), "tenant-a", "user-a", "", 10); err != nil {
		t.Fatal("canceled page poisoned later requests")
	}
}
