# Tong Panel P1: truthful worker and system audit provenance

**Status:** Experimental security bug fix, stacked on draft PR #20. Not merged, not installed on servers.

## Problem
The original queue wrote every event through `appendAudit` using `job.RequestedBy`. Consequently a worker lease, worker completion or automated lease-expiry could appear in the permanent audit history as an action performed by the human job requester. This falsifies attribution and breaks incident response.

## Remediation
- SQLite migration v3 adds `actor_kind` and `actor_ref` with check constraints and write/update triggers.
- New user-originated job creation and cancellation events keep `actor_identity_id` and `actor_kind=user`.
- Lease, start, renewal, explicit failure, retry and completion record `actor_kind=worker`, `actor_ref=<validated worker ID>`, and **NULL human identity**.
- Lease expiry and exhausted retry reconciliation record `actor_kind=system`, `actor_ref=lease-reconciler`, and **NULL human identity**.
- Previously saved, ambiguous worker-looking events from v1/v2 are relabelled `actor_kind=unknown` with NULL human identity. They are preserved, not rewritten as evidence about a worker that cannot be identified.
- The audit stores no claim token or lease token hash. Worker identifiers are length-limited and restricted to alphanumerics, dash, dot and underscore to avoid log injection.
- Audit writing occurs inside the same SQLite transaction as job mutation. A failed audit write must roll back the job action.
- Database schema migrations are serialized by the existing `BEGIN IMMEDIATE` mechanism. Future versions are rejected.

## Tests
Added regression tests covering user/worker/system roles, legacy v2 backfill without invented identities, worker ID input restrictions, direct SQL attempts to spoof actor kind, and audit token leakage checks. Existing full Go race + database schema test suites must pass in GitHub Actions before this is approved.

## Remaining safety constraints
This does NOT authenticate workers to the OS or prevent impersonation by a malicious peer with queue access. True worker enrollment, host-side monotonic effect fencing, rollback after interrupted operations, complete role policy, external security review, and a secure production job executor are still required. At this stage the queue accepts only `validation.noop` and **cannot modify the host**.

**Review order:** parent draft PR #14 -> #15 -> #16 -> #17 -> #20 -> this worker provenance PR. Other draft PRs #18/#19 require independent integration.
