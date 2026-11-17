# Current-role authorization for SQLite jobs

The development SQLite queue now rechecks mutation permission inside its data
transactions, in addition to the API's initial authorization. `CreateFor` checks
the active database identity, tenant, membership and `jobs:create` policy before
looking up an idempotency key. An identical retry is not a way to retain a role
that has since been withdrawn. `CancelFor` similarly checks `jobs:cancel` before
returning a repeated cancellation or changing queued state. Cancellation denial
continues to use `ErrNotFound` so other requesters' jobs are not disclosed.

Create authorization denial uses the generic `ErrRequesterDenied` sentinel.
The API maps it to HTTP 403 even when an earlier authenticated principal still
holds an allowed role. A regression test captures that principal, commits the
downgrade through another database handle, then verifies the post-authentication
handler creates neither a job nor an audit event. The test models this boundary
directly rather than relying on a timing-sensitive HTTP race.

Claim selection and every lease transition use one shared SQL role predicate
for the requester. Owner, admin, reseller and customer can mutate the current
no-op job kind; auditor cannot. Tests compare this predicate and the transactional
requester helper with `rbac.Principal.Can` for every current role. Changes to the
RBAC policy or role schema must update and revalidate this parity.

An auditor's own-job `GetFor` and `ListFor` access remains permitted. Mutation
checks deliberately do not replace the membership-only helper used by reads.
Tenant scoping, active-account checks, lease token hashes, expiry fencing and
transactional attempt/audit records remain in place.

A downgrade before claim makes the job ineligible without blocking another
eligible tenant's work. A downgrade after claim makes start, renewal, completion
and failure transitions return `ErrLeaseLost` without changing the job, attempt
or audit state. This is **current-role authorization**, not an irreversible
revocation epoch: restoring a permitted role can make an unexpired lease usable
again. Ineligible queued or expired jobs remain parked; this change does not
invent a terminal-state or cleanup policy for revoked requesters.

The original defect was reproduced by a failing test: create as owner, change
membership to auditor, then claim still succeeded. Regression tests now also
cover new/idempotent create and cancel, current-role parity, read preservation,
mid-lease downgrade, restoration, unchanged denied-operation state, and fairness
when an older queued or expired job has become ineligible.

This queue still accepts only `validation.noop`. No executing worker, host
adapter, arbitrary command, public authentication flow or privileged action is
added. Database lease fencing alone cannot fence an external side effect; host
effect authorization and monotonic fencing still require separate reviewed work.
