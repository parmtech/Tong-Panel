# P2 Slice 7 — Schedule Policy Decision Brief

**State:** decision options for maintainer/security review only. Nothing here is an approved architecture or product contract.

**Work item:** [P2 issue #151](https://github.com/parmtech/Tong-Panel/issues/151).

**Gate:** no scheduler, endpoint, shell execution, privileged operation, or host-facing behavior may be implemented until the applicable P0 decisions and P1 entry gates are approved. Project safety rules exclude shell commands as browser/API authority; this brief does not propose relaxing that rule.

## Invariants before selecting a schedule policy

- A schedule belongs to a server-resolved tenant and site. The caller cannot supply a tenant authority, executable, command, OS user, working directory, environment, or host path.
- Each scheduled action resolves to a fixed registered handler with a validated parameter schema and becomes an idempotent durable job.
- The executor rechecks current membership, site state, task version, and authorization immediately before claim/dispatch. Stale or revoked work is denied and audited.
- Every run has a bounded execution time, explicit terminal state, auditable actor/job/task relationship, and redacted bounded diagnostics.
- Restart, duplicate wakeup, retry, clock adjustment, and concurrent scheduler instances cannot cause an untracked or duplicate logical execution.

## Decisions and alternatives for review

| Decision | Candidate choices | Trade-offs and required review |
|---|---|---|
| Scheduled task kinds | Small fixed set of P2 operations; or a reviewed registry that can grow by versioned handler | A fixed set narrows the initial product surface; a registry is extensible but needs schema/version compatibility and migration rules. Arbitrary shell is excluded. |
| Schedule expression | Fixed intervals; constrained five-field cron; or another explicitly versioned grammar | Intervals are easy to validate and reason about but cannot express common calendar schedules. Cron is familiar but needs a parser, canonical form, bounds, and clear unsupported-field errors. |
| Time basis | UTC instants/intervals; or local wall-clock schedules with an explicit IANA timezone | UTC is predictable across DST but may not match a user's intended local time. Wall-clock schedules match local intent but need persisted timezone identity and defined DST behavior. Display conversion must not silently change stored meaning. |
| DST gap/fold | Skip a nonexistent wall time or move it forward; run once at the earlier or later instant for a repeated time; or another explicit policy | The policy must be stable across restarts and timezone database updates. Tests must prove no ambiguous schedule silently runs twice or disappears without an observable state. |
| Overlap and missed runs | Skip while running; coalesce missed occurrences; or enqueue each occurrence within a strict backlog cap | Per-kind semantics may differ. The choice must bound queue growth, define which occurrence key is idempotent, and be visible to the user. |
| Failure and retry | Bounded retries with backoff; or terminal failure requiring an authorized retry | Retry can repeat external effects, so handlers need idempotency or effect verification. Define crash-after-effect behavior, timeout, cancellation, and retry exhaustion. |
| User controls | Create/edit/pause/run-now/inspect/delete permission matrix | Run-now must enqueue the same typed job and authorization checks, not execute inline. Edits create a new version; define what happens to an already queued occurrence. |
| Execution history | Bounded per-run status and structured logs with retention; or status-only history | More detail improves support but increases sensitive-data and storage risk. Set byte/line limits, expiry, redaction, and cursor behavior before API/UI design. |

No candidate is selected here. Reviewers should record one outcome per row, owner/security reviewer, date, rationale, and linked ADR/threat-model updates in the repository's decision process.

## Acceptance cases required after policy approval

Use an injected clock and pinned timezone data so tests do not rely on the host clock or ambient locale.

1. Valid schedule inputs persist in one canonical form; malformed, oversized, unsupported, duplicate, and out-of-range values fail before creating a task or job.
2. The accepted timezone and grammar are versioned. A stored schedule's meaning remains stable after process restart and explicit timezone database update.
3. Cover DST gap and fold, leap day/month boundary, clock rollback/forward, restart immediately before/after due time, and a timezone change while a task is enabled.
4. Two scheduler instances and duplicate wakeups create at most one job for the same logical occurrence; task edits/deletes and already queued jobs follow the selected version policy.
5. Overlap, missed-run backlog, timeout, cancellation, retry exhaustion, handler crash, and crash after external effect produce the selected bounded and observable states.
6. Tenant membership revocation, role downgrade, site disable/delete, and task-version change after enqueue but before claim deny or safely reconcile the run.
7. Cross-tenant IDs, idempotency keys, task versions, cursors, and run-now requests cannot disclose or mutate another tenant's task or run.
8. Logs expose neither shell/environment values nor secrets, are bounded under oversized output, and obey the selected retention and expiry rules.

These tests are specifications until the policy is approved and the relevant contracts are implemented. Existing PR #60/#61/#65 renderers and PR #80's config rollback lab cover adjacent components, not schedule semantics.

## Traceability

- [P2 Slice 7 plan](P2_CRON_LOGS_TEMPLATES_TEST_PLAN.md).
- [P2 two-tenant integrated acceptance plan](P2_TWO_TENANT_END_TO_END_ACCEPTANCE.md).
- [P2 phase plan](phases/P2-HOSTING-ALPHA.md), [roadmap](ROADMAP.md), and [phase index](PHASE_INDEX.md).
- [ADR register](ADR_REGISTER.md), [threat model](THREAT_MODEL.md), and [security blueprint](SECURITY.md).
