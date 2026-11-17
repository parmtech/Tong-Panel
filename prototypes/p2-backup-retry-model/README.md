# P2 backup generation fence model (non-normative)

This standalone Python-standard-library, in-memory model exercises synthetic
backup identity and generation fencing. A backup ID, generation, and expected
SHA-256 digest are bound to an attempt; verification is required before
publication; and an older attempt cannot replace or delete a newer current
generation. A stale attempt reaches the explicit
`stale_reconciliation_required` terminal state. Tests also cover digest
mismatch, same-generation digest conflict, and immutable identity binding.

The `TransferSimulator` models bounded transfer outcomes without connecting to
a provider or storing payloads. Every simulator receives caller-injected size,
duration, attempt-count, backoff, and simultaneous-transfer limits, plus a
deterministic monotonic clock. Transfer progress is also caller-reported. A
retry reuses the same immutable identity and declared size; its start is gated
by the injected attempt cap and backoff schedule. The schedule must provide one
delay for each permitted retry. Size and duration are checked per attempt;
attempt count and backoff are scoped to one in-memory transfer operation, and
the concurrency cap applies to active attempts in one simulator instance.
These are test inputs, not production defaults or selected provider semantics.

`TransferSimulator.checkpoint()` can capture the coupled `Model` and simulator
as an immutable in-memory Python value, and `SyntheticCheckpoint.reconstruct()`
can rebuild that fixture under the same supplied assumptions. It preserves
identities and digests, sizes, current artifacts, attempt and operation history,
counters, retry limits and backoff, reconciliation states, and the clock
anchor. `checkpoint()` requires a non-empty synthetic epoch string and a
previously observed finite clock anchor. Active or inconsistent state is
rejected. Reconstruction requires the exact same epoch string and a supplied
clock whose first reading exactly equals the saved anchor, so elapsed downtime
has no modeled meaning. This is deterministic fixture reconstruction only:
actual process restart, durable encoding/recovery, provider behavior, and product
behavior remain unmodeled. The checkpoint has no disk, database, JSON, or
production persistence format. The 12-path candidate remains uncommitted and unpushed, and has no hosted CI. Sol fixed the stale successful same-generation conflicting-digest and substituted-later-publisher findings while preserving exact-time ambiguity; Luna QA found no counterexample. Root review passed and added a narrow huge-integer overflow validation guard and regression, independently QA-reviewed. Root coordinator reran the final suite and confirmed 39/39, including clock-failure atomicity. This is not P2 acceptance evidence.

Only a transfer with the declared size and reported byte count, content matching
the identity's expected SHA-256 digest, and passing generation fence can reach
`VERIFIED`. A failed transfer with reported remote bytes reaches `CLEANED` only
when `cleanup_verified=True` is injected; otherwise it reaches
`reconciliation_required`. That boolean is an observation supplied by the
test, not an object-removal operation or independent verification. An
unresolved partial transfer cannot be retried in this model. Oversize requests
are recorded as failures before transfer activity.

The model remains non-normative and is not a backup service or P2 acceptance
evidence. It stores no fixture payload and performs no filesystem, network,
database, key, API, authorization, quota, provider, or production operation.
Actual process restart, durable persistence and recovery, retention/deletion
policy, provider behavior, key custody, tenant authorization, source
quiescence, and complete database/service restore remain unmodeled. These
contracts and policy choices remain open. The stale-delete check only fences
an older attempt when a newer generation is current. Non-stale deletion raises
an out-of-scope result without mutating state; the model does not implement
deletion or select retention behavior. The positive integer generation is only
this model's ordering representation; the production generation type, encoding,
and persisted format remain undecided.

## Local checks

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s . -p 'test_*.py' -v
```

The offline-model workflow runs this test suite using the runner's Python
interpreter without installing third-party packages. Passing these model tests
does not satisfy a P2 gate.
