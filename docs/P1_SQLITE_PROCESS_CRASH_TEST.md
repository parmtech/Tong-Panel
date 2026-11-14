# SQLite queue worker-process crash test

The Linux-only subprocess test in `internal/jobs/sqlite_process_crash_test.go`
tests the durable queue library with real process death and natural lease expiry.
It adds no daemon flags, clock override, worker operation or production behavior.

The parent creates isolated synthetic tenant/job metadata and closes its database.
A child opens the real SQLite store, claims and starts the no-op job with a short
supported lease, signals only after the start transaction commits, then blocks.
The parent kills that process with SIGKILL and verifies signal termination.
It inspects the persisted running attempt, waits for the recorded lease deadline
using real time, then starts a fresh recovery child. That child claims, starts and
completes the same job. Final checks cover attempt history, audit actors, cleared
lease fields, idempotency and absence of additional claimable work.

The test does not edit lease timestamps, substitute the store clock, remove SQLite
WAL/shared-memory files, or recover using an inherited open database connection.
Subprocesses, pipes and waits have bounds and cleanup. Fixtures live in a private
temporary directory; no host command or customer job is executed.

This is process-crash evidence for the queue library. It does not prove the actual
`tongd` worker's 30-second lease behavior, a host executor's exactly-once effects,
kernel/VM power loss, disk failure, backup recovery or production readiness.
The existing daemon smoke still uses an explicitly expired fixture after a fatal
audit failure; these are separate tests with separate claims.

Run on Linux:

```sh
go test -race ./internal/jobs -run '^TestSQLiteProcessCrashRecovery$' -count=1
```

See `UPDATES.md` for authored versus actually executed evidence.
