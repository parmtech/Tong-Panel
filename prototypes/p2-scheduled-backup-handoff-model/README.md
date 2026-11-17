# P2 scheduled-run to backup-identity handoff model

This standard-library fixture composes two existing synthetic contracts:

- `p2-backup-schedule-run-model` derives an immutable run from a registered
  schedule and uses the occurrence key `(registered tenant, schedule ID,
  occurrence token)`.
- `p2-backup-retry-model` supplies the immutable backup identity
  `(tenant ID, backup ID, generation, expected SHA-256)` and bounded retry
  behavior.

The caller configures the handoff with the backup-retry model's canonical
`retry_contract.Identity` class and supplies the declared size. The constructor
accepts a caller-supplied frozen dataclass class; it does not itself hardwire
the retry module's class. With the canonical class configured, QA confirmed
that the value guard accepts only that exact class and validates its frozen,
hashable field shape before run creation. A well-shaped identity from another
dataclass is denied. This type/value check is fixture consistency, not
authorization. Exact replay returns the same immutable binding.
Rebinding the same occurrence to a different identity, size, or run revision is
denied without replacing the binding. The fixture preserves the identity values
from both input models. Its tenant consistency check is an in-memory fixture
identity check, not authorization or a security policy. No tenant value can be
supplied as an event override.

Retry limits are not selected or defaulted by this model. Tests pass explicit
size, duration, attempt, backoff, and concurrency bounds to the existing backup
retry simulator and verify that retries reuse the same handoff identity and
declared size while those supplied bounds remain enforced.

This code stores only synthetic fixture records in memory. It does not implement
a scheduler, queue, authentication, provider adapter, transfer, network or
filesystem operation, persistence, host action, key custody, or production
behavior. It is component acceptance preparation only; it does not satisfy P2
acceptance, complete issue #152, or change any gate.

Run from the repository root:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover \
  -s prototypes/p2-scheduled-backup-handoff-model -p 'test_*.py' -v
```
