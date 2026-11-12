# P2 Slice 7 — Cron, Site Logs, and Versioned Templates

**State:** proposed design and test specification only. No scheduler, host action, log reader, or configuration applier is implemented by this document.

**Phase:** P2 secure single-server hosting alpha. The roadmap and P2 plan require site-scoped cron, site logs, versioned server templates, and graceful service-failure handling. The current P2 gate report identifies this slice as lacking an integrated lifecycle design and evidence.

**Entry gates:** do not implement host-facing behavior until the P0 ADR/threat review and P1 authentication, typed job/agent, installer/recovery, and secret-handling gates have authoritative approval and evidence. Until then, work may cover pure validation, mock-free contract tests, or isolated non-privileged prototypes only. In particular, this plan does not authorize a public endpoint, shell execution, privileged service changes, or a real-host test.

## Goals and boundaries

This slice covers three P2 capabilities for a tenant-owned PHP site:

1. Define and inspect recurring site tasks through a server-owned allowlist of typed task kinds.
2. Read bounded, tenant-scoped site access and error logs with useful status and retention behavior.
3. Render versioned Nginx/PHP-FPM configuration, validate it before activation, and recover the last known-good version when validation or reload fails.

It does not add DNS provider management, mail, reseller policy, arbitrary user commands, general purpose log search, or a new service manager. Those remain outside this P2 slice or require separate reviewed scope.

## Existing draft work to reuse

These drafts are unmerged component evidence, not accepted product behavior:

- [PR #60](https://github.com/parmtech/Tong-Panel/pull/60), [PR #61](https://github.com/parmtech/Tong-Panel/pull/61), and [PR #65](https://github.com/parmtech/Tong-Panel/pull/65) cover pure Nginx/PHP-FPM/HTTPS rendering and selected clean-guest checks. Do not recreate their renderers as part of this slice; future tests should focus on immutable template-version selection and its association with the site/job state.
- [PR #80](https://github.com/parmtech/Tong-Panel/pull/80) covers staged Nginx syntax validation, activation, simulated reload failure, and rollback in a disposable guest. Reuse that lab or its contract after review. Additional coverage should prove the template-version and typed-job path invokes the already-tested lifecycle and reports its result; it should not duplicate the standalone rollback scenarios.

These branches remain drafts and their presence does not prove the end-to-end P2 gate.

## Candidate contracts for review

These are requirements for a future reviewed interface, not approved architecture decisions.

### Scheduled site task

- The caller never supplies a shell command, executable path, environment, user ID, working directory, or tenant ID as authority. The service derives tenant and site ownership from the authenticated principal and stored site record.
- A task is one registered typed operation with a validated parameter schema. The executor resolves the operation to a fixed handler; it must not interpolate request data into a shell string.
- A definition records its tenant/site ownership, task kind, validated schedule, enabled state, bounded timeout, overlap policy, timestamps, and version. Updates use optimistic versioning or an equivalent reviewed concurrency rule.
- Due work becomes an idempotent durable job. Authorization and current site state are revalidated when the job is claimed/executed, not only when it is created. Cancellation, retry exhaustion, and site deletion produce explicit terminal states.
- The schedule timezone and daylight-saving behavior must be explicit and stable across restarts. The reviewer must choose and document the policy before implementation. Ambiguous or nonexistent local times must never cause duplicate executions or silently skipped tasks.

### Site logs

- Access and error logs are selected from server-owned site mappings; clients do not choose arbitrary paths or tenant roots.
- Reads are bounded by bytes, lines, and time. The contract defines pagination/cursor behavior, whether a partial final line is returned, how rotation is handled, and how an expired cursor is reported.
- Authorization is tenant/site scoped and checked on each read. A site owner cannot read another tenant's logs by changing an ID, query parameter, cursor, or filename.
- Secrets and authorization material are redacted before response and diagnostic surfaces. Responses do not reveal host paths, other sites' names, or raw process environment.
- Retention and deletion policy is explicit. The panel does not promise recovery of entries already removed by the host's log rotation policy.

### Versioned server templates

- A rendered configuration is associated with a template identifier/version and the exact site facts used to render it. Template versions are immutable after publication; changing a template creates a new version.
- Tenant-provided values are validated as data and escaped for the target format. They cannot introduce directives, extra server blocks, file paths, socket paths, or include directives.
- Apply is a staged lifecycle: render, write to a private same-filesystem stage, run the target service's syntax check, preserve the current known-good version, activate atomically, reload, and verify health. Failure at any step leaves or restores the prior known-good configuration and records a redacted outcome.
- A successful syntax check alone is not proof of successful reload or healthy site behavior. State distinguishes rendered, validated, activated, reload-failed, health-failed, and rolled-back outcomes.
- The future adapter is a typed host operation with fixed arguments and audited actor/job attribution. It does not accept a free-form path or command from the API.

## Acceptance and test specification

| Area | Required evidence | Negative and failure cases |
|---|---|---|
| Schedule validation | Table-driven tests for accepted syntax, canonical persistence, timezone, minimum interval, timeout, and overlap rules | Reject shell metacharacters in task kind/parameters, malformed schedule, oversized values, unknown timezone, duplicate/ambiguous fields, and unsupported task kinds |
| Tenant authorization | Owner/Admin/Customer/Auditor behavior is specified and tested for each operation; site ownership is always server-derived | Cross-tenant IDs, stale membership/role, suspended tenant, disabled site, guessed job IDs, and replayed cursors are denied without existence leakage |
| Durable execution | Repeated delivery is idempotent; task and job state transitions survive restart; bounded timeout and retry exhaustion are observable | Crash before claim, after claim, and during handler; duplicate wakeup; cancellation; site deletion; permission revocation between enqueue and claim |
| Time behavior | Deterministic tests use an injected clock and fixed timezone database | DST gap/fold, leap day, clock rollback/forward, restart near due time, and concurrent scheduler instances cannot duplicate a logical run |
| Log isolation | Two-site fixture proves each authorized tenant sees only its own bounded access/error stream | Path traversal, symlink swap, cross-tenant ID/cursor, deleted/rotated file, oversized line, malformed encoding, partial write, and secret-bearing fixture |
| Template-version selection | Existing PR #60/#61/#65 renderer tests remain the component baseline; add only checks that an immutable version is selected and recorded with the site/job state | Unknown/deprecated version, missing required facts, or a tenant-injected version selector cannot choose a different tenant's config or publish output |
| Integrated config lifecycle | Reuse PR #80's isolated Nginx rollback lab; add the authenticated site/job-to-template-version path and verify the typed operation's terminal result against that lab | A version conflict, stale site, revoked authorization, failed validation/reload/health result, or job retry cannot silently mark the new version active; retain the prior known-good version |
| Audit and diagnostics | Actor, tenant, site, job, template version, transition, and stable error class are attributable without secrets | Tokens, passwords, environment values, raw host paths, and unbounded upstream output never appear in logs or API errors |
| UX/accessibility | Responsive site task/log views expose state, next action, retention limits, and permission errors using keyboard-accessible controls | Empty, loading, stale, denied, partially rotated, and failed states are understandable without color alone |

## Test execution gates

1. **Before P1 entry approval:** tests may cover pure schedule/value validation and template rendering with in-memory fixtures only. Do not add privileged integration tests or enable behavior in the daemon.
2. **After typed job and agent review:** exercise a fake typed adapter that records its structured operation; assert no free-form command/path crosses the boundary.
3. **After installer and supported-OS policy approval:** run service lifecycle checks only in fresh disposable guests for each exact supported OS/architecture. Use a temporary service instance and guest-local fixture paths; never modify the runner or developer's system Nginx/PHP-FPM/cron services.
4. **Before P2 acceptance:** combine authenticated API, current membership, job execution, tenant logs, template activation/rollback, and two-tenant denial in one end-to-end guest scenario. Unit tests and separate component labs do not satisfy this gate.

## Open decisions before implementation

- Which task kinds belong in P2, and what exact parameters may each accept?
- Which cron grammar, timezone source, DST fold/gap rule, minimum interval, timeout, and overlap policy are supportable?
- Which actor roles may create, edit, pause, run-now, inspect, and delete a task?
- What log retention, redaction, pagination, rotation, and maximum response limits apply?
- Which configuration changes require a service reload, how is health verified, and what is the bounded rollback/retry policy?
- Which template versions and OS package/service versions are supported, and how are prior versions retained and migrated?
- How do backup and restore preserve task definitions, template versions, and audit provenance without restoring stale executable state?

These decisions require the repository's owner/security review process. This proposal does not record approval for any ADR or threat-model item.

The [schedule policy decision brief](P2_SLICE7_POLICY_DECISION_OPTIONS.md) compares candidate task, schedule, timezone, overlap, retry, permission, and log-retention policies for maintainer review. It selects no policy and does not expand the implementation authorization above.

## Traceability

- [Phase 2 plan](phases/P2-HOSTING-ALPHA.md), implementation slice 7 and its must-pass tests.
- [Roadmap](ROADMAP.md), P2 scope and two-tenant/restore exit gate.
- [Test strategy](TEST_STRATEGY.md), isolated VM, fault injection, authorization, and recovery requirements.
- [Security blueprint](SECURITY.md), tenant-scoped authorization, typed host actions, safe paths, and secret redaction.
- [Latest Phase 1 entry-gate evidence](https://github.com/parmtech/Tong-Panel/blob/feature/p1-file-quota-request-context/docs/P1_EXIT_GATE_REPORT_2026-10-27.md) and [Phase 2 exit-gate evidence](https://github.com/parmtech/Tong-Panel/blob/feature/p1-file-quota-request-context/docs/P2_EXIT_GATE_REPORT_2026-10-29.md): P1/P2 release gates remain open; component prototypes do not establish integrated acceptance.
