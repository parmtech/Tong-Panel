package jobs

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/parmtech/tong-panel/internal/database"
)

type idempotencyKeyCase struct {
	name string
	key  string
	want bool
}

func idempotencyKeyBoundaryCases() []idempotencyKeyCase {
	return []idempotencyKeyCase{
		{name: "below minimum", key: strings.Repeat("a", 7)},
		{name: "minimum", key: strings.Repeat("a", 8), want: true},
		{name: "maximum", key: strings.Repeat("b", 128), want: true},
		{name: "above maximum", key: strings.Repeat("c", 129)},
		{name: "non-ASCII at minimum byte length", key: strings.Repeat("é", 4)},
	}
}

func TestStoreCreateEnforcesASCIIIdempotencyKeyByteBounds(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "jobs.json"))
	if err != nil {
		t.Fatal(err)
	}

	wantJobs := 0
	for _, tc := range idempotencyKeyBoundaryCases() {
		t.Run(tc.name, func(t *testing.T) {
			job, err := store.Create("tenant-a", KindNoop, tc.key)
			if tc.want {
				if err != nil || job.IdempotencyKey != tc.key {
					t.Fatalf("Create key length=%d returned job=%+v err=%v", len(tc.key), job, err)
				}
				wantJobs++
			} else if err == nil {
				t.Fatalf("Create accepted invalid key %q (%d bytes)", tc.key, len(tc.key))
			}
			if got := len(store.List("tenant-a")); got != wantJobs {
				t.Fatalf("job count=%d after key length=%d; want %d", got, len(tc.key), wantJobs)
			}
		})
	}
}

func TestSQLiteCreateEnforcesASCIIIdempotencyKeyByteBounds(t *testing.T) {
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`INSERT INTO tenants(id,name) VALUES('tenant-a','A');
		INSERT INTO identities(id,username) VALUES('user-a','alice');
		INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-a','user-a','owner');`); err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}

	wantJobs := 0
	for _, tc := range idempotencyKeyBoundaryCases() {
		t.Run(tc.name, func(t *testing.T) {
			job, err := store.CreateFor("tenant-a", "user-a", KindNoop, tc.key)
			if tc.want {
				if err != nil || job.IdempotencyKey != tc.key {
					t.Fatalf("CreateFor key length=%d returned job=%+v err=%v", len(tc.key), job, err)
				}
				wantJobs++
			} else if err == nil {
				t.Fatalf("CreateFor accepted invalid key %q (%d bytes)", tc.key, len(tc.key))
			}
			jobs, err := store.List("tenant-a")
			if err != nil || len(jobs) != wantJobs {
				t.Fatalf("job count=%d err=%v after key length=%d; want %d", len(jobs), err, len(tc.key), wantJobs)
			}
		})
	}
}
