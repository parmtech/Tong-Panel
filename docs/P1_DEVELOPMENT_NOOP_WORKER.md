# Opt-in development no-op worker

The loopback development daemon can run an explicitly enabled SQLite worker for
`validation.noop`. `TONG_DEV_RUN_NOOP_WORKER=yes` is accepted only when
`TONG_DEV_SQLITE_DB` is configured; the existing developer-token requirement for
that mode remains. Empty means disabled. Every other nonempty value fails with
a generic configuration error before opening a database or journal. No identity,
tenant or credential is created by enabling the worker.

The worker claims a 30-second lease, starts the job, checks that its authoritative
kind remains `validation.noop`, and completes it using the existing transactional
attempt/audit APIs. It checks the kind before start as well, and refuses future
kinds rather than turning into a generic executor. There is no callback, command,
path, shell, plugin or host adapter. Its only effect is SQLite queue metadata.
The fixed `local-dev-noop` audit reference is a development label, not a registered
or authenticated worker identity. Lease tokens and database error details are not
logged by this worker.

The worker copies the store and binds the copy to its cancellation context; the
API store's context is unchanged. Empty queues and withdrawn lease authority use
a cancellable 100-millisecond wait. Existing role/identity/tenant checks, lease
fencing, bounded attempt recovery and transactional audits remain authoritative.
Role-ineligible jobs stay parked. Audit/storage failures stop the worker with a
generic failure. A failed completion audit leaves the running attempt unfinished
for the existing expiry/retry mechanism; it never falsely records completion.

The daemon binds its IPv4 loopback listener successfully before starting a
worker. HTTP exit, a fatal worker error or shutdown cancels supervision, shuts
down HTTP (forcing close after a five-second grace period if necessary), and
joins both goroutines before returning to the caller that closes SQLite. Normal
signal shutdown exits successfully; fatal supervision errors exit nonzero.
The default-disabled path preserves queued-job/cancellation behavior.

Tests cover persisted completion and worker audit provenance, independent-worker
contention, role-ineligible fairness, expired-attempt recovery/stale capabilities,
audit rollback, unsupported kinds and cancellation without poisoning the API
store. Daemon tests cover strict opt-in, listener failure before claim, disabled
execution, supervision, cancellation and shutdown. Compiled-process CI is
configured to run `scripts/smoke_noop_worker.py` against built and extracted
binaries. It enqueues over HTTP, checks persisted attempts/actor provenance and
role-ineligible parking, injects a failed completion audit, then restarts the
real daemon against an explicitly expired fixture lease. This is not a physical
host crash, an elapsed-lease-duration test or proof of external-effect recovery.

This is not a production execution service. It grants no root or host capability,
and database fencing does not fence an external side effect. Reviewed worker
registration, typed host-effect authorization, monotonic external-effect fencing,
resource policy, privileged adapters and the remaining phase gates are still open.
