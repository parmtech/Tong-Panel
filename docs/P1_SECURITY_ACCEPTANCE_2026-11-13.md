# P1 security acceptance — owner/reviewer decision packet

**Prepared:** 2026-11-13 (Australia/Sydney)  
**Basis:** Tong Panel integration draft [#171](https://github.com/parmtech/Tong-Panel/pull/171), exact tested source `4837d0c76286bc58db86cc00a98733292af09317`.  
**Decision:** **P1 EXIT NOT APPROVED. P2 ENTRY BLOCKED. DO NOT MERGE/DEPLOY/INSTALL.**

This is a factual checklist to make unresolved decisions actionable. It is **not** a security approval or completed third-party audit. No owner, security reviewer, release signer or supported production distro has been designated here. The GitHub integration review supersedes 149 historical draft PRs; historical failure/cancellation and missing checks are preserved in [the reconciliation report](PR_RECONCILIATION_2026-11-13.md).

**Additional lab evidence (13 Nov 2026):** the [ADR-008 offline signature-verification model](../prototypes/p1-release-manifest-model/README.md) tests separately pinned Ed25519 public keys, exact source/version/target metadata and signed artifact digests using ephemeral keys and synthetic bytes. This is non-product contract evidence only; it does **not** select a production trust anchor, sign a release, prove provenance or change any gate below. Its distinct GitHub CI outcome must be checked on the new published head.

**Additional offline failure-recovery design evidence:** the [P1 inert signed-release model](../prototypes/p1-release-manifest-model/README.md) now also simulates a bounded upgrade journal, restart/uncertain effects, fresh signature validation, health/active-version reconciliation and rollback *recommendations*. It contains no durable journal, host effects or trusted health observations. These tests **do not** resolve ADR-008 signing or actual installer/recovery acceptance.\n\n**Additional dpkg failure-lifecycle evidence:** a [rootless isolated dpkg lab](P1_DPKG_FAILURE_RECOVERY_LAB.md) tests synthetic `preinst`/`postinst` failures, explicit half-configured recovery, corrupt archive denial and unexpected old-`prerm` fallback. This is not systemd, production package, signed installer, power-loss or customer recovery evidence. Its new-head CI is separate and must be verified before claiming any hosted pass.

**New-agent entrypoint (13 November 2026):** read the [complete A–Z handoff](AI_AGENT_START_HERE.md) and [state snapshot](AI_AGENT_STATE.json) before any implementation. The consolidated [PR #171](https://github.com/parmtech/Tong-Panel/pull/171) has extensive development evidence, but **all independent P0/security/release approval gates below remain open**. A passing blocker-audit workflow is not release authorization.\n\n## Verified baseline (development only)

The original integration head passed **12/12 GitHub workflows**, including root Go race/fuzz/vet, API contract, UI/browser, schema, static-analysis reporting, vulnerability checks, isolated ACME/Prometheus, offline P3/P4/P5 models, unsigned package, developer smoke and both Ubuntu 24.04/Debian 12 disposable VM jobs. See [PR #171](https://github.com/parmtech/Tong-Panel/pull/171) for exact run links. No workflow verifies formal P0 architecture acceptance, real operator identity, production issuance/key custody, supported installer signatures or an independent red-team assessment. Reporting success does not close untriaged scanner findings.

## Decision and acceptance ledger

| Gate | Current evidence | Missing approval/verification required | Responsible decision |
|---|---|---|---|
| P0 architecture ADR-001–009 | [ADR register](ADR_REGISTER.md) describes proposals; [issue #1](https://github.com/parmtech/Tong-Panel/issues/1) open | Record **Accepted / Rejected / Needs revision** per ADR with date, real named approvers, impact, migration and link; reconcile threat-model assumptions | Owner/maintainer + independent security reviewer (unassigned) |
| Threat model and tenant privilege boundary | [Security](SECURITY.md), [threat model](THREAT_MODEL.md), ping-only Unix-socket agent and negative tests | Independent review of API↔agent identity, TOCTOU/symlink, replay, scope isolation and host effects; prohibit root actions until approved | Independent security reviewer (unassigned) |
| Authentication and recovery — ADR-009 | Argon2id, TOTP, revocable SQLite sessions, CSRF/Origin and post-auth job-read fixes; [issue #6](https://github.com/parmtech/Tong-Panel/issues/6) open | Review public login design, MFA recovery/lockout, external secret storage, TLS/proxy boundary, bootstrap, smallest-VM password hashing budget; run adversarial session/tenant tests before enabling a login endpoint | Owner + auth/security reviewer (unassigned) |
| Certificates — ADR-007 | Fresh DNS evidence and read-only ACME preflight; local test-CA interruption/reconciliation | Approve issue/renew authority, freshness policy, service identity, account-vs-certificate key scope, encrypted persistent custody, restart reconciliation, bounded retry/renew, audit and no-privilege host installation/rollback contract | Owner + security/key-custody reviewer (unassigned) |
| Browser terminal T0 | [isolated terminal proposal](P1_BROWSER_TERMINAL_PROPOSAL.md); no shipped shell | Decide tenant/runtime scope, PTY process owner, sandbox/quotas, attach auth, revoke/reap and persistent audit **before** allowing a process; root host shell forbidden | Owner + security reviewer (unassigned) |
| Supply chain and release — ADR-008 | unsigned developer artifact, checksums, SBOM and two disposable OS guests | Review provenance/licences and SAST findings; sign immutable tagged package with trusted key, verify bootstrap signature, enforce clean-install/upgrade/uninstall/rollback and corrupted/compromised-key negative tests | Release maintainer + independent reviewer (unassigned) |
| Support/restore | Ubuntu 24.04 and Debian 12 disposable fixture smokes, encrypted synthetic restore | Owner-approved support matrix; complete authenticated two-tenant site/DB/files/SSL/backup recovery across destructive restart, restore and cross-tenant adversarial tests on supported clean guests | Owner + QA/release reviewer (unassigned) |
| P1 UX accessibility | Node/browser/keyboard/mobile scoped tests, working read-only shell | Production login UX, independent keyboard/screen-reader/WCAG checks and beginner/operator acceptance; no falsely active controls | UX/accessibility reviewer (unassigned) |

## Review order and blocking rules

1. **Owner action:** on [issue #1](https://github.com/parmtech/Tong-Panel/issues/1), assign actual ADR decision owners and independent security reviewer; record decisions on ADR-001–009 and security threat model with timestamps and supporting evidence. Lack of a response means **unapproved**, not implied approval.
2. **Auth action:** on [issue #6](https://github.com/parmtech/Tong-Panel/issues/6), review login/MFA recovery, bootstrap, secret custody and TLS/proxy/session boundaries. Existing internal session primitives are not permission to expose public login.
3. **Certificate/terminal action:** resolve [certificate contract](P1_CERTIFICATE_OPERATION_CONTRACT.md) and [terminal T0](P1_BROWSER_TERMINAL_PROPOSAL.md) explicitly. Continue isolated negative tests only; no persistent CA keys, public issuer or tenant shell until approved.
4. **Release action:** triage security findings/provenance; design and independently verify signed artefacts, installer/rollback/recovery on exact distro images. Distinguish developer packages from installable releases.
5. **Gate decision:** update this ledger with actual evidence and explicit reviewer conclusions. Re-run every relevant gate on the new exact code head; a documentation-only edit does not inherit an approved production release or human signoff.

## Current disposition

- **No additional PR required** solely for planning: this packet lives in the sole draft [#171](https://github.com/parmtech/Tong-Panel/pull/171).
- **Not approved:** P0 ADRs, public authentication, root host adapter, persistent SSL key custody, browser terminal T0, supported installer, P1 phase exit, P2 entry, customer deployment.
- **Next single actionable step:** obtain a real recorded ADR/security decision on issue #1, beginning with the privilege/key-custody boundary in ADR-007 and authentication in ADR-009, then convert approved requirements into narrow tested changes.
