# P2 Two-Tenant End-to-End Acceptance Plan

**State:** proposed acceptance design only. This plan records no owner/security approval and is not evidence that any P2 gate has passed.

**Work items:** [P2 issue #150](https://github.com/parmtech/Tong-Panel/issues/150) tracks staged ACME and WordPress integration; [P2 issue #152](https://github.com/parmtech/Tong-Panel/issues/152) tracks scheduled offsite backup and retention; [P2 issue #153](https://github.com/parmtech/Tong-Panel/issues/153) tracks database lifecycle and credential-rotation acceptance.

**Purpose:** turn the P2 two-tenant and recovery exit criteria into one reviewable clean-guest scenario. The current P2 gate report says component checks exist, but no end-to-end panel-to-agent suite covers API, files, database users, domains, and job logs together.

**Evidence freshness:** the P2 exit report prepared 2026-11-07 lists PR #91's WordPress VM execution as pending. At PR #91 head `01ae2237b8bdb4240fb79ca45842f9e6d4f549a1`, the [Ubuntu clean-VM job](https://github.com/parmtech/Tong-Panel/actions/runs/42633862164/job/129216767234) and [Debian clean-VM job](https://github.com/parmtech/Tong-Panel/actions/runs/42633862164/job/129216766868) have since passed. Their workflow step runs `scripts/clean_vm_smoke.sh`, whose source includes the disposable WordPress backup-source and fresh-VM restore-target scripts. This updates that component-evidence detail only; PR #91 remains a draft and these jobs do not prove a complete P2 install, lifecycle, or integrated two-tenant restore gate.

**Entry gate:** the latest Phase 1 report says Phase 2 entry is blocked. Do not enable or test public authentication, privileged host operations, installer mutations, or customer-like production workloads until P0 decisions and P1 security, identity, typed-job/agent, signed-package, and recovery gates have recorded approval and evidence. Before that, only prepare fixtures and pure contract tests in disposable, non-privileged environments.

## Existing draft evidence to reuse

All items below are unmerged prototypes; they are useful inputs to a future integrated suite, not accepted product behavior:

- [PR #40](https://github.com/parmtech/Tong-Panel/pull/40) covers a two-root tenant Files lifecycle fixture; later Files drafts cover bounded upload, deletion, quotas, and maintenance barriers.
- [PR #62](https://github.com/parmtech/Tong-Panel/pull/62) exercises two rendered PHP-FPM/Nginx tenants in clean Ubuntu/Debian guests.
- [PR #64](https://github.com/parmtech/Tong-Panel/pull/64) checks MariaDB grants deny cross-tenant reads; [PR #79](https://github.com/parmtech/Tong-Panel/pull/79) adds tenant-scoped dump/restore checks.
- [PR #70](https://github.com/parmtech/Tong-Panel/pull/70) exercises an authenticated read-only vhost preview; [PR #73](https://github.com/parmtech/Tong-Panel/pull/73) adds a read-only Files UI.
- [PR #74](https://github.com/parmtech/Tong-Panel/pull/74) is an unsigned disposable package lifecycle lab; [PR #80](https://github.com/parmtech/Tong-Panel/pull/80) tests staged Nginx rollback in a disposable guest.
- [PR #84](https://github.com/parmtech/Tong-Panel/pull/84) exercises encrypted MariaDB restore to a fresh guest; [PR #91](https://github.com/parmtech/Tong-Panel/pull/91) extends the WordPress backup/restore lab.

The integrated suite must consume reviewed interfaces and reuse these fixtures where possible. It should add only the missing panel-to-job-to-agent and cross-subsystem assertions, not repeat the component-level tests above.

## Required test environment

- Start each acceptance run from a fresh disposable VM image whose source URL, version, architecture, and digest are recorded and verified before boot.
- Run only against the exact supported distro/package matrix approved by maintainers. Existing Ubuntu/Debian draft guest results are not certification.
- Keep all test tenants, domains, credentials, service instances, package state, and data inside the disposable guest or runner-private temporary storage. Never use a production host or real customer data.
- Use a deterministic private DNS fixture for domain ownership and a certificate-authority staging endpoint for ACME behavior. Do not make the suite depend on production DNS zones or production certificates.
- Capture command exit status, guest image digest, package and binary hashes, test report, sanitized service logs, and restore comparisons. For backups, include the approved disposable target/interface identity, artifact IDs and ciphertext hashes, schedule/retention policy version, non-secret key reference, and upload/verification/deletion outcomes. Redact secrets and avoid raw environment dumps.
- If a prerequisite is missing or unknown, assert a safe denial before the first host mutation and stop the run; do not silently skip the gate.

## End-to-end scenario

Create two isolated tenants, `tenant-a` and `tenant-b`, with one site, one database, and distinct users per tenant. Use the roles and authentication flows approved for the release candidate; this document does not define new role policy.

| Step | Action | Required assertions |
|---|---|---|
| 1. Preflight and install | Run the exact supported package's non-mutating preflight, then install only after every required fact passes | Unsupported OS, insufficient disk, occupied required port, unknown reading, or missing policy stops before package metadata, service files, or state directories are created; a passing report records the exact facts and policy version |
| 2. Authentication and tenant setup | Bootstrap and authenticate the two test identities using the approved MFA/session flow | No shared/default credential; revoked, expired, disabled, or cross-tenant identity is denied; audit records contain actor and tenant attribution without secrets |
| 3. Domain, certificate, and site lifecycle | Create one site per tenant, prove each test domain through the deterministic DNS fixture, issue certificates only through the configured staging CA, then exercise renewal and approved site activation | A tenant cannot claim or mutate the other tenant's site/domain/certificate; pending or failed proof never enters an active vhost; each certificate SAN matches only its verified tenant domains; the runner validates the chain and hostname through the staging trust anchor; failed renewal preserves the last known-good certificate and serving state; duplicate and stale requests are deterministic |
| 4. Web runtime and WordPress recipe | Use each tenant's real disposable Nginx/PHP-FPM instance, install the exact checksum-pinned WordPress release through the approved recipe, and create distinct tenant-owned content markers | Core checksum mismatch, archive traversal/link entries, or an unverified domain prevents publication; each HTTPS domain serves only its own marker and WordPress content; each install uses its own database and runtime identity; direct requests, traversal, symlink swaps, hidden files, alternate Host headers, and guessed site IDs cannot read the other tenant's files or private runtime data |
| 5. Database lifecycle | Create app-managed database/user pairs for each tenant through an approved typed interface; exercise credential rotation, scoped export/import, and authorized site teardown | Each principal can access only its own schema; the old credential is denied after the reviewed rotation cutover and the new credential works only for its tenant; active connections/config updates follow the approved transition policy; grant/revoke/secret-handoff failure never reports success or leaves broader grants; teardown and retries reach an explicit terminal or recoverable state; credentials appear only through approved secret custody, never normal API responses, logs, support bundles, or audit details |
| 6. Files and quota | Exercise the supported Files upload/read/delete/archive workflow through authenticated tenant-scoped APIs and UI | Tenant A cannot list/read/overwrite/delete tenant B paths; traversal, symlinks, hostile archives, decompression limits, quota overflow, cancellation, and concurrent publication fail without partial publication or cross-tenant effects |
| 7. Jobs, cron, and logs | Enqueue one approved typed operation per tenant, inspect status/logs, and exercise the approved scheduled-task contract if implemented | No arbitrary command/path crosses the API; each job and log is visible only to authorized principals for its tenant; membership revocation before execution denies the operation; audit/log output is bounded and redacted |
| 8. Config apply and failure recovery | Select the versioned template, stage config, syntax-check, activate, reload, and probe each site; inject failure at each transition | The test identifies the active template version. Syntax, write, reload, or health failure retains/restores the prior known-good config, emits a redacted terminal job result, and leaves both tenants' serving state explainable |
| 9. Scheduled offsite backup and clean restore | Schedule encrypted backups for both tenants to an approved disposable offsite target; verify each remote artifact; exercise retention expiry/deletion; destroy the source overlay; boot a new guest and restore from retained artifacts, including sites/databases/settings and WordPress content | Only ciphertext reaches the target and key material stays outside it; schedule/run identity is idempotent and attributable; transfer or remote verification failure is not reported as success; retention never deletes a newer valid backup; deletion is verified and safely retried; wrong tenant/key, tamper, truncation, missing artifact, and partial restore fail closed; restore does not resurrect stale executable tasks or broaden tenant access; restored certificates are revalidated or safely reissued under the reviewed policy; the new guest contains no source-VM state |
| 10. UI and accessibility | Complete common create/view/recovery tasks at phone/tablet viewport sizes with keyboard-only navigation | No horizontal overflow or keyboard trap; labels, focus, status/errors, loading/empty/denied states, and recovery guidance are available without color alone; automated checks are supplemented by recorded human accessibility review |

## Cross-tenant denial matrix

For every resource above, run both an authorized positive case and a negative case using tenant B's principal against tenant A's object. Repeat with:

- altered path, site/database/job/domain ID, pagination cursor, alias, or template version;
- a stale authorization snapshot, role downgrade, membership removal, disabled identity, or suspended tenant between request and job execution;
- duplicate concurrent create/delete/update requests, replayed proof, idempotency-key reuse across tenants, and cancellation during a writer or host-operation wait;
- DNS challenge name/token, ACME order/certificate ID, certificate file mapping, or renewal task belonging to the other tenant;
- database/user identifier, credential reference, rotation job, secret-handoff result, or deprovision request belonging to the other tenant;
- backup schedule/run ID, remote object key, tenant/key reference, or retention/deletion request belonging to the other tenant;
- an API read, UI action, background job, support bundle, and audit/log view where the resource can be surfaced.

Where the public API contract intentionally conceals object existence, assert the same external response for absent and foreign-tenant objects while retaining an appropriately attributed internal denial audit. A denial must not partially mutate metadata, files, database grants, service config, backup state, or job state.

## Failure injection and recovery evidence

Inject process termination, timeout, disk-full, permission denial, service reload failure, wrong DNS TXT value, delayed DNS propagation, ACME staging rejection/rate limit/timeout, failed certificate renewal, offsite-target throttling/timeout/partial upload, artifact-verification failure, retention deletion failure, database unavailable, database grant/revoke failure, credential rotation race or secret-handoff failure, deprovision interruption, source stream error, and restore interruption at documented boundaries. For every case, record:

1. the phase/job state before and after failure;
2. whether any external effect occurred and how it was detected;
3. the idempotent retry or rollback action and its bounded retry policy;
4. the final filesystem, database, service, and tenant-authorization snapshots;
5. proof that secrets and other tenants' data were absent from user-visible errors and captured diagnostics.

## Exit evidence checklist

- [ ] P0 ADR/security decisions and threat review have recorded owner outcomes.
- [ ] P1 exit report shows all authentication, identity, typed-job/agent, signed package, clean install/rollback/recovery, supply-chain, and accessibility gates verified.
- [ ] The exact supported OS/architecture matrix is published and matches the guest image hashes used.
- [ ] Every P2 implementation slice has linked component evidence and is present in the integrated guest run.
- [ ] Staging-CA issuance and renewal are exercised for both tenants; DNS/ACME failures retain the prior known-good certificate, and no production CA/account is used.
- [ ] The pinned WordPress archive checksum and bounded no-link extraction are verified before publication; both tenant installs restore posts and settings without exposing private configuration or sharing databases.
- [ ] Each tenant's app-managed database lifecycle covers create, rotation, scoped use/export/import and authorized deprovisioning; secret custody is approved, old credentials are rejected after cutover, and no cross-tenant grant or credential leak occurs.
- [ ] Scheduled offsite backups are verified as ciphertext at a disposable approved target; key material remains separate and retention deletion is verified without removing a newer valid artifact.
- [ ] Source-VM destruction precedes restore from the retained offsite artifact; restored tenant sites, databases, settings and WordPress content are checked against recorded hashes and cross-tenant denial cases.
- [ ] Every cross-tenant positive/negative case passes across API, files, database, domains, jobs/logs, and backup/restore.
- [ ] Failure injection proves safe rollback or an explicit recoverable state; no mutation is silently reported as successful.
- [ ] A fresh-VM restore contains sites, databases, configuration, and settings, with tenant boundaries re-tested after restore.
- [ ] UI phone/tablet workflows and accessibility review are complete.
- [ ] Test reports and sanitized logs identify exact code/package/image hashes; no unsigned developer preview is described as a supported release.

Until all required evidence exists, the P2 exit gate remains open. A passing component test, a PR's green checks, or this test plan does not constitute Phase 2 completion.

## Traceability

- [Phase 2 plan](phases/P2-HOSTING-ALPHA.md), slices 1–10 and must-pass end-to-end tests.
- [Roadmap](ROADMAP.md), P2 scope and exit gate.
- [Phase index](PHASE_INDEX.md), evidence-based phase transitions.
- [QA strategy](TEST_STRATEGY.md), disposable VM, authorization, fault-injection, browser, and recovery layers.
- [ADR register](ADR_REGISTER.md), specifically proposed ADR-007 — Privilege boundary and secrets. It is still unapproved; the database credential flow in step 5 cannot be implemented until owners/security approve secret custody and the typed host boundary.
- [Phase 1 entry gate](https://github.com/parmtech/Tong-Panel/blob/feature/p1-file-quota-request-context/docs/P1_EXIT_GATE_REPORT_2026-10-27.md) and [Phase 2 exit gate](https://github.com/parmtech/Tong-Panel/blob/feature/p1-file-quota-request-context/docs/P2_EXIT_GATE_REPORT_2026-10-29.md).
