# P1 — SQLite Queued-Job Cancellation (Draft)

**State:** implementation proposed in an isolated stacked review branch, not merged or wired into the dev API.

## Behavior
- `SQLiteStore.CancelFor(tenant, requester, jobID)` withdraws only a **queued** `validation.noop` job created by the **same active requester in the same active tenant**.
- Worker leases and cancellation use the same SQLite `BEGIN IMMEDIATE` transaction lock. Only one may win; there is no state in which a cancelled job can acquire a new lease.
- Repeated cancellation is idempotent and does not repeat the `job.cancelled` audit event.
- Cancellation and the audit entry commit or roll back together; loss of the audit table causes cancellation to fail closed.
- A leased, running, completed or failed job returns `ErrJobNotCancelable`. This prevents a user interface from falsely indicating that an external side effect has stopped.
- Invalid tenant/requester/job identifiers and disabled users or suspended tenants are denied without leaking other people's jobs.
- This function never contacts the privileged Linux agent, does not execute workloads and is **not** a host-side cancellation mechanism.

## Testing
- Same-owner queued cancellation and repeated idempotency.
- Cross-tenant/cross-requester denial; tenant suspension and user disablement.
- Rejection of leased, running and completed jobs.
- Audit failure rolls back status.
- Concurrent worker claim and user cancellation over separate SQLite connections: only one succeeds.

## Known limitations
- Does not add an HTTP DELETE endpoint, production login integration, worker stop request, resource lock, client retry ID or host-side fencing.
- A running operation cannot be stopped safely merely by changing its database state; proper host-side fencing and an effect acknowledgement protocol are required first.
- Audit currently attributes worker lease events to the requesting user in other PR #17 code. Review and correct actor provenance before production use.
- PR #14–#17 are still stacked drafts; merge parent PRs in order only after security/design review. This branch should target PR #17, not main.
- Complete P1 acceptance, privileged agent security review, OS lab testing and signed installation remain pending.

See `UPDATES.md` and PR links for actual CI evidence.
