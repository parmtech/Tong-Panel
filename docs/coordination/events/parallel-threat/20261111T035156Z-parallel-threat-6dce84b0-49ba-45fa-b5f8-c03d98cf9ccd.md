# Threat reproducibility milestone and source corrections

- Event ID: parallel-threat-20261111T035156Z-6dce84b0-49ba-45fa-b5f8-c03d98cf9ccd
- UTC time: 2026-11-11T03:51:56Z
- Writer/phase: Parallel threat reproducibility / P0 supplemental
- Kind: status
- Recipients: coordinator, relevant P0 workstreams

## Completed since adoption

Read back adoption event `parallel-threat-20261111T035006Z-a5b71c43-0a19-49ba-b2b1-68bb914133ca` from `docs/phase-coordination`; its create commit was `49f8fcb7f8a2762eab38970f0da996dc38145c31` and fetched blob `da213f6828e777e18a4ecdd76b33a88553927c99`.

Exact pointer resolution through read-only public GitHub sources found:
- `cabef25…` = `cabef25caa779db1650353fd67e8ff9c0fa70dda`, a SQLite queue contention change including `internal/jobs/sqlite_load_test.go`; it does not support the draft's T03 agent decoder fuzz attribution.
- `6c983a…` = `6c983a22de1e5b1ea0de133ea28c6d1daadc490c`, formatting-only change to `internal/api/session_cookie_test.go`; it does not establish cookie/origin coverage.
- Session lifecycle `6ae402888141e76aa83f38af5f96805598da5966` and internal login `c2cfadc0d6e85be13b5dfab7083a77405505caf6` point to the corresponding auth test files.
- All seven draft-cited Actions IDs resolve completed/success: `40598738018`, `40598737975`, `40598738139`, `40598737988`, `40591534457`, `40591534497`, `40591534190`. Job/step summaries were inspected for the four `6ae402...` associated runs; they show Go formatting/tests/vet, SQLite migration constraint verification, local daemon/bootstrap dry-run, and nonproduction unsigned archive checks. Step summaries do not expose exact command text or output.
- The draft's referenced PRs are open. Current PR-head SHA pointers are recorded in the local matrix; they are not merged-main evidence. The two helper reviews confirm the draft's evidence-preparation framing is appropriate but its broad proposed contracts still lack deterministic, runnable case setup/assertions/artifacts/teardown.

Local artifact: `threat-test-reproducibility.md`, SHA-256 `E9C604CBBCFCB026291BFA442B8548D09AE47E8E1C05D0B9A2EDB97AC91489CA`. No tests were executed locally; no test outcomes were invented.

## Staffing and state

Both assigned read-only helper subtasks completed; successful starts used explicit `gpt-6-luna`, low reasoning. Their private task IDs/settings are available in the chat to the coordinator. No third helper was needed.

## Scope and next action

This event corrects evidence interpretation; it does not itself alter the P2 draft, accept threats, approve mitigation, or establish phase acceptance. P0 remains in progress. No production/privileged action, merge, deployment, or test execution occurred.

Next: canonical owner should correct the misleading T03 and T08 source descriptions and add stable case-level reproducibility fields. If safe test rerun proceeds later, first pin the exact public code/tree and command; distinguish PR-only evidence from merged code. No sibling acknowledgment is implied.

## Read receipt

Coordination protocol/roster were read at extension commit `0a30a21677276b340f8da530384d89e7b6b09b02`. Adoption event above was fetched from remote; no other event IDs have been read. This status replies to/extends the adoption event only.
