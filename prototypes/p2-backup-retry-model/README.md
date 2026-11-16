# P2 backup generation fence model (non-normative)

This standalone Python-standard-library, in-memory model exercises one
provider-neutral part of the P2 backup contract: a synthetic backup ID,
generation, and expected SHA-256 digest are bound to an attempt; verification
is required before publication; and an older attempt cannot replace or delete a
newer current generation. A stale attempt reaches the explicit
`stale_reconciliation_required` terminal state. Tests also cover digest
mismatch, same-generation digest conflict, and immutable identity binding.

The model stores no fixture payload, contacts no provider, and performs no
filesystem, network, database, key, API, authorization, quota, or production
operation. It is non-normative and is not a backup service or P2 acceptance
evidence. It does not model process restart or durable persistence, retries or
backoff, transfer limits or concurrency, remote partial-object cleanup,
retention, provider behavior, key custody, tenant authorization, source
quiescence, or complete database/service restore. Those contracts and policy
choices remain open; no values or provider semantics are selected here.
The stale-delete check only fences an older attempt when a newer generation is
current. Non-stale deletion raises an out-of-scope result without mutating state;
the model does not implement deletion or select retention behavior.
The positive integer generation is only this model's ordering representation;
the production generation type, encoding, and persisted format remain undecided.

## Local checks

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s . -p 'test_*.py' -v
```

The offline-model workflow runs this test suite using the runner's Python
interpreter without installing third-party packages. Passing these model tests
does not satisfy a P2 gate.
