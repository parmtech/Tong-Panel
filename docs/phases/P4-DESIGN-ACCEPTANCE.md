# P4 hosting parity — gated design and acceptance plan

**Status:** BLOCKED / design only. This document does not implement or certify P4.  
**Branch:** `docs/p4-hosting-parity-design`  
**Baseline:** `main` at `0aad68e81138a0f226e83e401f90bae4d0b6bddc`  
**Plan source:** `docs/phases/P4-HOSTING-PARITY.md`, `docs/BLUEPRINT.md`, `docs/ROADMAP.md`, `docs/PHASE_INDEX.md`.  
**Scope boundary:** DNS/mail design, PostgreSQL/extensions, reseller and quota boundaries, migration dry-run/reconciliation and phase-specific acceptance specifications. No P6 work.

## Gate and dependency record

The phase index defines P4 as dependent on P2 and P3. P2 is itself gated on verified P1; P3 requires P2's secure site routing, auth/quotas, backups and deploy jobs. The current P1 exit report says P1 is incomplete and P2 entry is blocked. GitHub contains narrow passing P2 lab results and draft P2/P3 specifications, but no accepted P2 or P3 phase gate. Therefore every item below is preparatory; do not implement privileged service mutations or call any acceptance gate passed until maintainers record upstream evidence and security/ADR decisions.

Required handoffs before P4 implementation:
1. P1 exit evidence and explicit P2 entry decision, including approved trust/agent and installer gates.
2. P2 stable site/domain, typed-job, tenant authorization, quota, encrypted secret, backup/restore, and rollback contracts, each linked to exact verified commits/tests.
3. P3 build/runtime isolation, deployment/rollback, secret, and job contracts with hostile-app evidence.
4. A reviewed P4 architecture decision that resolves the relevant proposed ADRs, especially datastore strategy and optional DNS/mail profiles. Existing ADRs are proposals, not approvals.

### P4 capability-to-prerequisite traceability

Use this map to request exact producer evidence when the upstream phase gates reopen. It narrows the existing P4 feature and acceptance lists to the contracts each slice consumes; it does not authorize implementation ahead of the phase entry decision.

| P4 capability group | Required upstream contracts to accept first | P4 acceptance cases |
|---|---|---|
| DNS and mail (slices 1–3) | P2 verified-domain ownership/proof, TLS and service health, typed durable jobs with audit, tenant secret references, quotas/rate limits; P3 job retry/cancel state and reviewed egress/secret boundary where deploy/runtime integration is involved | 1 DNS provider matrix; 2 mail isolation/abuse; 6 reconciliation/cutover |
| PostgreSQL and extensions (slice 4) | P2 database ownership/grants, package/install policy, tenant resource quotas, encrypted backup/restore and upgrade/recovery contract; P3 typed job lifecycle, service/resource isolation and secret rotation semantics | 3 PostgreSQL profile; 7 restore evidence |
| Resellers, packages and quotas (slice 5) | P1 reviewed identity/RBAC and audit; P2 authoritative quota reservation/accounting at API and adapter, suspend/restore and ownership; P3 execution-time job reauthorization and resource accounting | 4 reseller/quota adversarial matrix |
| cPanel/WHM migration and reconciliation (slice 6) | P2 bounded upload/archive staging, site/domain/database/mail ownership, backup/restore and rollback/compensation; P3 artifact, runtime secret and deployment-state contracts for any imported app/release content | 5 archive adversarial corpus; 6 reconciliation/cutover; 7 restore evidence |
| Site operations and operator interfaces (slices 7–8) | P1 scoped API/CLI auth, token/session policy and audit; P2 approved cron/log/template/site lifecycle, backup/export and error states; P3 durable event/job lifecycle for deployment notifications and webhooks | Relevant slice-specific tests plus full reference-deployment release gate |

Before proposing an implementation issue, each dependency row must link the accepted contract version, exact source commit, relevant test run and artifact, reviewer/maintainer gate decision, and exclusions. A draft PR, component test, or unsigned VM smoke is not a substitute for any required contract.

### P4 architecture-decision and threat crosswalk

The ADR register and threat model are still proposals, not approvals. Resolve the applicable decisions through the repository review process before any privileged adapter or external-service mutation is implemented.

| P4 capability group | Existing decision proposals to review | Threats and minimum P4 evidence |
|---|---|---|
| DNS and mail | ADR-006 optional module profiles; ADR-007 typed host boundary, secret encryption and rotation | T08/T09/T13; prove scoped provider credentials, DNS ownership and TLS challenge results, no webhook replay/SSRF, authenticated mail submission, no open relay, quota/rate enforcement, and honest port-25/deliverability status (cases 1, 2, 6) |
| PostgreSQL and extensions | ADR-003 datastore/job boundaries; ADR-005 supported OS/package matrix; ADR-007 privileged package and credential boundary | T02/T03/T10/T15; prove per-tenant grants, allowlisted extension/package request, denial of arbitrary host operations, bounded restore and recovery from interrupted jobs (cases 3, 7) |
| Resellers, packages and quotas | ADR-007 authorization at the typed adapter; P1 authentication/RBAC decision record; P2 authoritative quota contract | T01/T02/T15; prove delegated capability ceilings, request/job/adapter reauthorization, atomic quota accounting and safe suspend/restore under retries (case 4) |
| cPanel/WHM migration | ADR-007 safe staging/privilege boundary; ADR-008 artifact provenance and supported-host/recovery policy | T02/T04/T10/T11/T15; prove hostile archive rejection, no dry-run writes, bounded unprivileged staging, source preservation and resumable compensation limits (cases 5–7) |
| Operator API, CLI, events and webhooks | ADR-002 API boundary; ADR-007 typed operations and audit; P1 authentication/token decision record | T01/T02/T08/T15; prove scope checks at enqueue and execution, replay-resistant webhook handling, idempotent token rotation, audit attribution and bounded failure recovery (cross-cutting cases 1–7) |

If review rejects or changes one of these proposals, update the affected P4 contract and tests before implementation; do not infer an accepted decision from the presence of this crosswalk.

## Non-overlapping P4 workstreams

### DNS and mail design

- Keep authoritative DNS, external DNS APIs, and mail independently optional capabilities. A basic website profile must continue to work when either is absent.
- Define a provider-neutral typed desired-state contract for zones/records and mail domains/mailboxes. Every operation must bind tenant, domain ownership evidence, capability, idempotency key, expected revision and audit/job identity.
- DNS design must cover authoritative and externally hosted zones, import/export, TTL, DNSSEC state, split DNS, propagation observations, scoped provider credentials, timeout/retry bounds, provider rate limits, drift detection and safe conflict handling. Treat propagation as asynchronous evidence, not a successful API write.
- Mail design must cover Postfix/Dovecot profile isolation, mailbox/alias/forwarder/auto-reply lifecycle, authenticated submission, abuse/rate quotas, queue diagnostics, external relay, TLS, DKIM rotation and SPF/DMARC/MX/PTR checks. State clearly that port 25 and deliverability are provider/network dependent and never guaranteed.
- Avoid storing plaintext provider tokens, mailbox passwords, DKIM private keys or message content in jobs/logs. Specify envelope encryption/key rotation ownership, redaction tests, minimal retention and export controls before implementation.
- Failed DNS or mail dependencies must fail closed for mutations, report partial state and preserve a compensating action. Do not claim rollback for external effects that cannot be reversed; use reconciliation and explicit operator review.

### PostgreSQL and extensions

- Treat PostgreSQL as an optional separately versioned service profile, not an implied replacement for panel metadata storage. The architecture ADR-003 currently leaves that choice unresolved.
- Define tenant database/user ownership, per-database least privilege, credential rotation, extension allowlists by tested OS/package profile, explicit version compatibility, connection/resource limits, backup/restore and upgrade windows.
- Extension installation must use a reviewed package allowlist and declarative typed request; reject arbitrary SQL, package names, filesystem paths and shell fragments from tenant inputs. Separate privileged package changes from tenant database grants.
- Acceptance must cover extension policy denial, unsupported version/OS, partial package failure, role/database isolation, credential rotation, backup/restore, upgrade compatibility and audit redaction.

### Reseller, package and quota boundaries

- Resellers may delegate only capability subsets granted by the platform owner; child accounts must never acquire server-admin or sibling tenant authority.
- Make package limits explicit per resource (sites/domains, storage, database count/size, mailbox count/size, DNS zones/records, API/job rates, runtime CPU/RAM) with documented units, rounding and enforcement semantics.
- Enforce limits at request admission, durable job revalidation and the typed host adapter. UI validation is informational only.
- Define reservation/release semantics for concurrent creates, retries, cancellation, suspension and restoration. Use an atomic reservation or equivalent so two concurrent requests cannot oversubscribe. Define grace/overage reporting; do not silently delete or suspend data when a limit is reached.
- Every object and audit event needs immutable owner/reseller lineage and tenant scope. Test IDOR, changed reseller grants, revoked membership, cross-package transfer and stale queued work.

### Migration dry-run and reconciliation

- Treat every source archive as hostile data. Never execute bundled scripts or trust paths, symlinks, ownership, package metadata, serialized objects or declared sizes.
- Dry-run must be read-only and produce a stable, versioned manifest: source fingerprint, parser version, detected accounts/domains/sites/databases/mail/DNS/backups, conflicts, unsupported items, estimates, required destination capabilities, planned writes, secret reset requirements and warnings. Redact credential values while identifying reset/rotation actions.
- Bound archive entry count, expanded bytes, compression ratio, nesting, path length, parser time and temporary disk. Reject traversal, absolute paths, device nodes, symlink/hardlink escapes, duplicate/conflicting paths, decompression bombs, malformed encodings and ambiguous ownership.
- Stage extracted data under a disposable unprivileged identity and tenant-scoped directory. Verify checksums where present; never restore directly to production paths from an uploaded archive.
- Reconciliation compares desired manifest with destination inventory under a pinned revision, shows create/update/skip/conflict actions and exact pending writes, and requires a fresh review if the source or destination revision changes.
- Execution, once later authorized by upstream gates, should be resumable and idempotent by migration ID and step; record a durable journal, validate post-import services, preserve source, and provide rollback/compensation boundaries. DNS cutover is a separate explicit operator decision with TTL/propagation checks and reversal plan.

#### Candidate dry-run report contract (review only)

The following shape makes the already-required manifest fields and safe preview behavior testable. It is a **proposal for review**, not an approved API or execution schema. Keep identifiers opaque, use stable reason codes, and never include passwords, tokens, DKIM keys, serialized source secrets, or host filesystem paths.

```json
{
  "schema_version": "tongpanel.migration-dry-run/v1",
  "source": {
    "format": "cpanel-whm",
    "archive_sha256": "<64 lowercase hex characters>"
  },
  "parser": { "id": "cpanel-whm", "version": "<pinned parser version>" },
  "destination": { "inventory_revision": "<pinned opaque revision>" },
  "generated_at": "<RFC 3339 UTC timestamp>",
  "result": "blocked",
  "inventory": [
    { "source_ref": "<opaque ref>", "kind": "mailbox", "classification": "supported" }
  ],
  "planned_actions": [
    {
      "step_ref": "<stable ref>",
      "action": "manual_review",
      "source_ref": "<opaque ref>",
      "destination_ref": null,
      "reason_codes": ["destination_conflict"],
      "depends_on": []
    }
  ],
  "required_capabilities": ["mail.profile.v1"],
  "secret_actions": [
    { "source_ref": "<opaque ref>", "kind": "mailbox_credential", "action": "reset" }
  ],
  "unsupported_items": [],
  "warnings": [],
  "resource_estimates": { "expanded_bytes": 0, "database_bytes": 0, "file_bytes": 0 },
  "limits_observed": { "entries": 0, "expanded_bytes": 0, "elapsed_ms": 0 },
  "plan_fingerprint": "<sha256 of canonical plan fields>"
}
```

`result` is limited to `ready`, `blocked`, `partial`, or `rejected`; `ready` means the preview is fully classified, not that any import ran. Only `ready` may present a candidate create/update plan, and that plan still requires a separate later authorized approval before execution. `blocked`, `partial` and `rejected` reports contain no executable actions. Sort inventory, actions, reason codes and capability lists deterministically. Compute `plan_fingerprint` from canonical source/parser/destination revisions, inventory classifications, planned actions, secret actions, warnings and estimates; exclude `generated_at`, temporary staging names and the fingerprint itself. A new source digest or destination revision invalidates approval and requires a fresh preview. Estimates and limits must state units and reflect observed data, never source-declared sizes alone.

Candidate contract tests: identical archive bytes, parser version and destination revision yield the same fingerprint; changing source bytes, parser version or destination revision changes it; action dependencies form an acyclic graph and every inventory reference resolves; all planned creates/updates appear in the review list; unsupported/manual-review items cannot be executed; credential canaries never appear in serialized output or logs; limit overflow rejects without an executable plan; and a dry-run leaves durable destination state unchanged and removes temporary staging on success, failure and cancellation.

## P4 acceptance specification (not yet executed)

1. **DNS provider matrix:** in isolated test doubles and a disposable authoritative-DNS lab, verify create/update/delete/import/export, DNSSEC invalid states, split-horizon behavior, stale revisions, idempotent retry, provider timeout/rate-limit, partial failure, propagation pending/failure, credential scope and drift reconciliation.
2. **Mail isolation/abuse:** disposable mail lab with synthetic identities verifies send/receive only through authenticated submission; tenant mailbox boundaries; relay denial; per-mailbox/domain rate and queue bounds; full queue/disk behavior; SPF/DKIM/DMARC/MX/TLS/PTR checks including stale/mismatch results; port-25 unavailable warning; key rotation; secret/log redaction. No public delivery claim.
3. **PostgreSQL profile:** supported-version matrix per OS/architecture; prove per-tenant DB/role separation, least-privilege grants, rotation/revocation, allowlisted extensions only, package failure behavior, quota races, dump/restore correctness and cross-tenant denial.
4. **Reseller/quota adversarial matrix:** nested reseller attempts, forbidden role escalation, sibling tenant IDs, concurrent final-slot allocation, job replay, cancellation/retry races, package downgrade/transfer, suspended/revoked principals and stale jobs. Assert both API and adapter denial, unchanged state on rejection, attributed audit and no secret values.
5. **Archive adversarial corpus:** traversal, absolute path, symlink/hardlink escape, duplicate entries, oversized metadata, expansion bomb, malformed/truncated archive, forged checksums, unsupported formats, credential-bearing config, duplicate domain/account, database incompatibility and resource exhaustion. Assert bounded execution, no writes during dry-run, precise reject/unsupported report and clean staging teardown.
6. **Reconciliation/cutover:** deterministic manifest round-trip; complete pending-write list; source/destination revision drift prompts re-review; interrupted staged restore resumes without duplicate accounts; validation failure leaves source intact and reports compensation limits; DNS cutover remains manual.
7. **Restore evidence:** restore synthetic mail, PostgreSQL and site data to a separate disposable host; verify ownership, permissions, service health and data integrity. This is a lab test, not a production host action.

Every eventual result must record exact source commit, OS/architecture/service profile, command and test artifact, counts, failures, reviewer and exclusions. Authored specifications are not tests passed.

## Current evidence and remaining gates

- **Current gate reports:** the [P1 exit report on PR #149's branch](https://github.com/parmtech/Tong-Panel/blob/feature/p1-file-mutation-context/docs/P1_EXIT_GATE_REPORT_2026-10-27.md) still says **NOT PHASE 1 COMPLETE; NOT PUBLISH/INSTALL READY; PHASE 2 ENTRY BLOCKED**. The [P2 exit report at the same branch](https://github.com/parmtech/Tong-Panel/blob/feature/p1-file-mutation-context/docs/P2_EXIT_GATE_REPORT_2026-10-29.md) says **P2 IN PROGRESS — EXIT GATE NOT PASSED; NOT PRODUCTION READY**. These explicit decisions remain controlling despite component checks below.
- **P1 exact-head evidence:** draft PR #143 (`feature/p1-file-quota-request-context`, head `482017005f9872c1fe7f2969ebfb477ea7de30dd`) passed its six non-VM checks and both Ubuntu 24.04/Debian 12 clean-VM jobs in [run #42781985275](https://github.com/parmtech/Tong-Panel/actions/runs/42781985275). Newer draft PR #149 (`feature/p1-file-mutation-context`, head `4d42ca024a7628e424a9738aa258d70927122f02`) has six non-VM checks passing; its Ubuntu/Debian clean-VM run [#42785167173](https://github.com/parmtech/Tong-Panel/actions/runs/42785167173) was still in progress at this snapshot. Both are unsigned developer-preview branches, not P1 exit evidence.
- **P2 narrow lab evidence has advanced:** draft PR #89 exact head `51e12abb99d919e5e8cb10a69e868dc65c4ed9a0` passed Ubuntu 24.04 and Debian 12 clean-VM run [#42631444449](https://github.com/parmtech/Tong-Panel/actions/runs/42631444449); logs report two isolated WordPress installs, tenant-scoped DB separation, and successful dump/restore checks. Draft PR #91 exact head `01ae2237b8bdb4240fb79ca45842f9e6d4f549a1` passed both guests in [#42633862164](https://github.com/parmtech/Tong-Panel/actions/runs/42633862164); logs report encrypted WordPress SQL/config/content restored on a fresh guest, homepage/upload verification, and denials for private-config and privileged-DB reads. These are meaningful P2 lab results, not full P2 acceptance.
- The observed workflow explicitly calls the artifact an **unsigned developer preview**. It does not establish a signed supported installer, service install/upgrade/rollback, fully authenticated user lifecycle, full two-tenant hosting acceptance, or maintainer/security gate approval. Keep P1 incomplete and P2 in progress until the complete phase exit matrix is evidenced.
- **Related planning work (avoid duplication):** draft PR #145 (`docs/p3-prerequisite-evidence`, current head `ccbadf2c33e18e34745593c53da144bb8c01bee6`) now includes a dependency-ordered P3 review packet, crash/restart checkpoints and synthetic fixtures; its body still labels P3 entry unproven and these fixtures as narrow evidence. Draft PR #144 (`phase2/cron-logs-templates-test-plan-20261110`, current head `80308251b13d562a3ff4aa24659d6583b251c474`) adds a proposed P2 integrated two-tenant acceptance sequence and tracks staging-CA/WordPress restore gaps in issue #150, alongside its Slice 7 tests. Both remain design-only and unmerged. Reference their upstream gate work rather than duplicating it; no P3 hostile-app phase acceptance is evidenced.
- Before P4 implementation, re-fetch the referenced PRs, exact heads and CI/review state; link each required P1/P2/P3 acceptance artifact and explicit maintainer gate decisions. Passing component or lab tests do not alone satisfy upstream phase gates.
- This continuation preserves the original P4 design and test specifications and refines only the dependency evidence. No DNS, mail, migration, host-service or live-host mutation was performed. P4 remains **BLOCKED / design only** pending P1/P2/P3 handoffs and relevant architecture/security approvals.

## Next action

Recheck P1/P2/P3 gates from GitHub before each new P4 session. Ask maintainers to link exact verified P2/P3 contracts and acceptance artifacts, plus relevant ADR decisions, before translating this plan into implementation issues. Until then, continue only with evidence-led design and test-spec clarification; do not start service mutations or claim phase acceptance.
