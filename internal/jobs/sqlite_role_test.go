package jobs

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/rbac"
)

func setRequesterRole(t *testing.T, store *SQLiteStore, role rbac.Role) {
	t.Helper()
	if _, err := store.db.Exec("UPDATE memberships SET role=? WHERE tenant_id='tenant-a' AND identity_id='user-a'", role); err != nil {
		t.Fatal(err)
	}
}

// Membership changes are excluded; every persisted queue/attempt/audit field
// must remain identical when a mutation is denied. Never print lease hashes.
func queueRoleSnapshot(t *testing.T, store *SQLiteStore) map[string]string {
	t.Helper()
	return queueRoleSnapshotForJob(t, store, "")
}

func queueRoleSnapshotForJob(t *testing.T, store *SQLiteStore, jobID string) map[string]string {
	t.Helper()
	result := map[string]string{}
	for _, table := range []string{"jobs", "job_attempts", "audit_events"} {
		query := "SELECT * FROM " + table
		var arguments []any
		if jobID != "" {
			column := map[string]string{"jobs": "id", "job_attempts": "job_id", "audit_events": "subject_id"}[table]
			query += " WHERE " + column + "=?"
			arguments = append(arguments, jobID)
		}
		rows, err := store.db.Query(query+" ORDER BY id", arguments...)
		if err != nil {
			t.Fatal(err)
		}
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			t.Fatal(err)
		}
		var contents strings.Builder
		for rows.Next() {
			values := make([]any, len(columns))
			destinations := make([]any, len(columns))
			for i := range values {
				destinations[i] = &values[i]
			}
			if err := rows.Scan(destinations...); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			fmt.Fprintf(&contents, "%#v\n", values)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		rows.Close()
		result[table] = contents.String()
	}
	return result
}

func requireQueueRoleSnapshot(t *testing.T, store *SQLiteStore, before map[string]string) {
	t.Helper()
	if !reflect.DeepEqual(before, queueRoleSnapshot(t, store)) {
		t.Fatal("denied operation changed persisted job, attempt or audit state")
	}
}

func TestSQLiteClaimRejectsRequesterRoleDowngrade(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	if _, err := store.CreateFor("tenant-a", "user-a", KindNoop, "role-downgrade-0001"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec("UPDATE memberships SET role='auditor' WHERE tenant_id='tenant-a' AND identity_id='user-a'"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ClaimNext("role-check-worker", time.Minute); !errors.Is(err, ErrNoJobs) {
		t.Fatalf("requester's withdrawn mutation authority still permitted claim: %v", err)
	}
}

func TestSQLiteRequesterDowngradeDeniesCreateRetryAndCancelButAllowsOwnReads(t *testing.T) {
	store, _ := openTestSQLiteStore(t)
	queued, err := store.CreateFor("tenant-a", "user-a", KindNoop, "role-existing-0001")
	if err != nil {
		t.Fatal(err)
	}
	cancelled, err := store.CreateFor("tenant-a", "user-a", KindNoop, "role-cancelled-0001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CancelFor("tenant-a", "user-a", cancelled.ID); err != nil {
		t.Fatal(err)
	}
	setRequesterRole(t, store, rbac.Auditor)
	before := queueRoleSnapshot(t, store)
	for _, key := range []string{"role-existing-0001", "role-new-request-0001"} {
		if _, err := store.CreateFor("tenant-a", "user-a", KindNoop, key); !errors.Is(err, ErrRequesterDenied) {
			t.Fatalf("create or idempotent retry retained withdrawn authority: %v", err)
		}
	}
	for _, id := range []string{queued.ID, cancelled.ID} {
		if _, err := store.CancelFor("tenant-a", "user-a", id); !errors.Is(err, ErrNotFound) {
			t.Fatalf("cancel or idempotent retry retained withdrawn authority: %v", err)
		}
	}
	if _, err := store.GetFor("tenant-a", "user-a", queued.ID); err != nil {
		t.Fatalf("auditor's own-job read was incorrectly blocked: %v", err)
	}
	if jobs, err := store.ListFor("tenant-a", "user-a"); err != nil || len(jobs) != 2 {
		t.Fatalf("auditor's own-job listing was incorrectly blocked: count=%d error=%v", len(jobs), err)
	}
	requireQueueRoleSnapshot(t, store, before)
}

func TestSQLiteJobMutationRolePredicateMatchesRBAC(t *testing.T) {
	for _, role := range []rbac.Role{rbac.Owner, rbac.Admin, rbac.Reseller, rbac.Customer, rbac.Auditor} {
		t.Run(string(role), func(t *testing.T) {
			store, _ := openTestSQLiteStore(t)
			if _, err := store.CreateFor("tenant-a", "user-a", KindNoop, "role-claim-0001"); err != nil {
				t.Fatal(err)
			}
			cancellable, err := store.CreateFor("tenant-a", "user-a", KindNoop, "role-cancel-0001")
			if err != nil {
				t.Fatal(err)
			}
			setRequesterRole(t, store, role)
			before := queueRoleSnapshot(t, store)
			principal := rbac.Principal{TenantID: "tenant-a", UserID: "user-a", Role: role}
			allowed := principal.Can("jobs:create", "tenant-a")
			var sqlAllowed bool
			if err := store.db.QueryRow("SELECT " + jobExecutionRolePredicate + " FROM memberships m WHERE m.tenant_id='tenant-a' AND m.identity_id='user-a'").Scan(&sqlAllowed); err != nil {
				t.Fatal(err)
			}
			if sqlAllowed != allowed {
				t.Fatal("execution SQL role predicate drifted from RBAC jobs:create")
			}
			if err := authorizedRequester(context.Background(), store.db, "tenant-a", "user-a", "jobs:create"); (err == nil) != allowed {
				t.Fatal("requester mutation authorization drifted from RBAC")
			}
			_, createErr := store.CreateFor("tenant-a", "user-a", KindNoop, "role-fresh-0001")
			_, cancelErr := store.CancelFor("tenant-a", "user-a", cancellable.ID)
			claim, claimErr := store.ClaimNext("role-parity-worker", time.Minute)
			if (createErr == nil) != allowed || (cancelErr == nil) != principal.Can("jobs:cancel", "tenant-a") || (claimErr == nil) != allowed {
				t.Fatal("create/cancel/claim disagreed with current role policy")
			}
			if !allowed {
				requireQueueRoleSnapshot(t, store, before)
				return
			}
			if _, err := store.Start(claim); err != nil {
				t.Fatal(err)
			}
			renewed, err := store.Renew(claim, time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.Complete(renewed); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSQLiteLeaseTransitionsRejectCurrentRoleDowngradeWithoutMutation(t *testing.T) {
	for _, action := range []string{"start", "renew", "complete", "fail"} {
		t.Run(action, func(t *testing.T) {
			store, _ := openTestSQLiteStore(t)
			if _, err := store.CreateFor("tenant-a", "user-a", KindNoop, "role-midlease-0001"); err != nil {
				t.Fatal(err)
			}
			claim, err := store.ClaimNext("midlease-role-worker", time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			if action != "start" {
				if _, err := store.Start(claim); err != nil {
					t.Fatal(err)
				}
			}
			run := func() error {
				switch action {
				case "start":
					_, err := store.Start(claim)
					return err
				case "renew":
					_, err := store.Renew(claim, time.Minute)
					return err
				case "complete":
					_, err := store.Complete(claim)
					return err
				default:
					_, err := store.Fail(claim, "validation_failed", false, 0)
					return err
				}
			}
			setRequesterRole(t, store, rbac.Auditor)
			before := queueRoleSnapshot(t, store)
			if err := run(); !errors.Is(err, ErrLeaseLost) {
				t.Fatalf("withdrawn mutation authority retained lease operation: %v", err)
			}
			requireQueueRoleSnapshot(t, store, before)
			// This is current-role authorization, not a permanent revocation
			// epoch. A still-current lease is usable after authority is restored.
			setRequesterRole(t, store, rbac.Owner)
			if err := run(); err != nil {
				t.Fatalf("restored authority could not use the unexpired lease: %v", err)
			}
		})
	}
}

func TestSQLiteRoleIneligibleOlderJobsDoNotStarveEligibleTenant(t *testing.T) {
	for _, expired := range []bool{false, true} {
		t.Run(fmt.Sprintf("expired-%v", expired), func(t *testing.T) {
			store, _ := openTestSQLiteStore(t)
			now := time.Date(2026, 11, 10, 1, 0, 0, 0, time.UTC)
			store.now = func() time.Time { return now }
			older, err := store.CreateFor("tenant-a", "user-a", KindNoop, "role-older-0001")
			if err != nil {
				t.Fatal(err)
			}
			if expired {
				claim, err := store.ClaimNext("old-role-worker", time.Second)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := store.Start(claim); err != nil {
					t.Fatal(err)
				}
			}
			setRequesterRole(t, store, rbac.Auditor)
			now = now.Add(2 * time.Second)
			eligible, err := store.CreateFor("tenant-b", "user-b", KindNoop, "role-newer-0001")
			if err != nil {
				t.Fatal(err)
			}
			before := queueRoleSnapshotForJob(t, store, older.ID)
			claim, err := store.ClaimNext("new-role-worker", time.Minute)
			if err != nil || claim.Job.ID != eligible.ID {
				t.Fatalf("ineligible older job starved eligible tenant: %v", err)
			}
			if !reflect.DeepEqual(before, queueRoleSnapshotForJob(t, store, older.ID)) {
				t.Fatal("role-ineligible job, attempts or audit were reconciled or mutated")
			}
		})
	}
}
