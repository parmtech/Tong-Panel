# P2 retention-delete reconciliation model (synthetic only)

This standard-library, in-memory fixture models a delete intent for one
preselected artifact that the fixture declares previously verified. The
immutable target is identified by tenant and opaque artifact ID; an immutable
operation ID is bound to that exact target. Replaying that operation ID for the
same target returns its current intent, while reusing it for a different tenant
or artifact is denied. Operation IDs share one namespace across the single and
batch APIs, so an ID cannot be changed from one intent kind to the other.

The model performs no deletion. Tests inject a `DeleteObservation` for the exact
target. `PRESENT`, `UNKNOWN`, or `UNAVAILABLE` observations leave the intent in
`reconciliation_required`; replaying its request does not clear that state. Only
an exact `VERIFIED_ABSENT` observation can move it to `verified_absent`. A
completed intent is terminal, and a mismatched observation cannot retarget it.

The artifact and observation types are test fixtures, not independent proof of
artifact integrity or remote absence. The model does not choose which artifacts
are retention-eligible, implement a retention policy, contact a provider, hold
credentials, perform a delete, model provider-specific versioning, or cover
authorization, scheduling, persistence, restart, host behavior, or real
reconciliation. It is acceptance preparation only; it does not satisfy P2
acceptance, close issue #152, or change owner/security gates.

## Fixed-set batch reconciliation

The batch API is separate from the single-artifact API. The caller supplies a
nonempty tuple of previously verified fixture artifacts. The model rejects
mutable collections, duplicate artifacts, and malformed identities, then
stores the set in deterministic order. An operation ID already used by a
single-artifact intent cannot be reused for a batch. It does not decide which
artifacts are eligible for retention or deletion.

Each batch observation must be a nonempty tuple with exactly one
`DeleteObservation` for every bound artifact. Missing, extra, duplicate,
foreign, malformed, or retargeted observations are denied without changing
batch state. A complete observation containing any `PRESENT`, `UNKNOWN`, or
`UNAVAILABLE` item keeps the batch in `reconciliation_required`. Only an exact
full-set observation reporting `VERIFIED_ABSENT` for every item completes the
batch. Exact absence replay is idempotent; a contradictory later observation
cannot reopen a completed batch.

The batch state and observations remain synthetic, in-memory fixture data.
The model performs no deletion, provider lookup, retention selection, or
independent absence verification.

Run the focused tests from the repository root with:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s prototypes/p2-retention-delete-reconciliation-model -p 'test_*.py' -v
```
