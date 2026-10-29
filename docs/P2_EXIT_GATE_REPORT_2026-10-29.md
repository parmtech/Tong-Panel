# Tong Panel — Phase 2 Exit Gate Report
**Prepared:** 2026-10-29 (Australia/Sydney)  
**Verdict:** **P2 IN PROGRESS — EXIT GATE NOT PASSED; NOT PRODUCTION READY.**  
**Basis:** `docs/phases/P2-HOSTING-ALPHA.md`, current draft PRs, hosted CI evidence, and advisory review `docs/ADR_SECURITY_REVIEW_2026-10-29.md`.

This report distinguishes component evidence from integrated product acceptance. All cited work remains on unmerged draft branches. No real VPS, production tenant, or customer data was changed. A green unit test or disposable VM lab does not constitute support certification or security approval.

## Implementation slices

| P2 slice | Current evidence | Exit status / remaining acceptance |
|---|---|---|
| 1. Supported package/installer, preflight, journal and recovery | PR #74 unsigned Debian package lab tests install without auto-start, loopback non-root service, upgrade/downgrade and state-marker retention on Ubuntu 24.04/Debian 12. PR #80 tests isolated Nginx staged-config rollback. | **Open.** No signed/approved installer, privileged install lifecycle, migration/restore recovery guarantee, or full interruption/rollback drill. Neither distro is certified. |
| 2. Nginx/PHP-FPM service adapters | PRs #61/#65 preview renderers; PR #80 rollback lab; clean guests run isolated two-tenant PHP-FPM/Nginx fixtures. | **Open.** Preview and lab code are not wired to a reviewed privileged adapter or customer host. No integrated apply, health-check, rollback and reconciliation lifecycle. |
| 3. Domains, DNS ownership, redirects and vhosts | PRs #58–#70 implement metadata, proof and preview checkpoints; clean guest checks exercise tenant host routing and redirects. | **Open.** No production DNS/provider integration, full lifecycle, external ownership/certificate verification, or reviewed host apply path. |
| 4. MariaDB provisioning, rotation, export/import and isolation | PRs #75/#77 provide credential-free metadata APIs. PR #79's isolated tenant-grant/export/import test passes Ubuntu/Debian. PR #84 adds encrypted dump and fresh-host restore; code head `95257da5d1faaae5cc07f27125b64c20b24c8e49` passes Go/race/vet, daemon, archive, and Ubuntu [job 123172500141](https://github.com/parmtech/Tong-Panel/actions/runs/40789979352/job/123172500141) / Debian [job 123172500626](https://github.com/parmtech/Tong-Panel/actions/runs/40789979352/job/123172500626) clean-VM restore checks. | **Open.** No app-managed MariaDB creation/removal/credential rotation or approved secret custody. PR #84's runner temp handoff is not offsite storage; no schedule, retention/deletion, full multi-tenant/settings restore, or recovery guarantee. Latest PR head adds an owner-read permission negative test; its latest CI is pending. |
| 5. Files API/UI, quotas and ACL | PR #53 quota metadata; PR #73 read-only Explorer/download UI; tenant-scoped path/entry guards and download limits. | **Open.** No upload/edit/delete flow, integrated auth/CSRF session path, privileged filesystem adapter, per-operation quotas/ACL lifecycle, or browser accessibility acceptance. |
| 6. ACME issue/renew/status/redirect | HTTPS renderer/redirect and HTTP-01 challenge-path fixture checks exist. | **Open.** No real ACME issuance/renewal, DNS-01 provider, certificate lifecycle/secret custody, expiry alert or external certificate verification. |
| 7. Cron, site logs and templates | No integrated lifecycle evidence found in the reviewed P2 checkpoint set. | **Open.** Define tenant authorization, safe schedule validation, log isolation/retention, template versioning and failure recovery; implement and test on clean guests. |
| 8. WordPress/PHP deploy and restore | No pinned/checksummed deploy recipe and clean-VM restore evidence found in the reviewed P2 checkpoint set. | **Open.** Add verified pinned artifacts, isolated setup, tenant-specific configuration, upgrade/rollback and restore tests. |
| 9. Scheduled offsite encrypted backups, retention and complete restore | PRs #81–#83 implement streaming envelope, private atomic staging and key re-encryption; PR #84 verifies fresh-VM MariaDB restore on both target guest OSes. | **Open.** No durable offsite provider, backup scheduler, retention/deletion policy, production external-key custody/rotation/recovery, panel job integration, or complete settings/sites/databases restore. |
| 10. Multi-tenant adversarial tests and usable onboarding | Component tests cover many cross-tenant cases; disposable guests verify selected MariaDB, PHP-FPM/Nginx and HTTPS isolation cases. PR #73 is read-only Explorer. | **Open.** No end-to-end adversarial panel-to-agent suite across files, DB, domains and job logs; no integrated phone/tablet task-completion or accessibility acceptance. |

## Must-pass end-to-end acceptance

| Requirement | Current evidence | Status |
|---|---|---|
| Two tenants isolated through API, files, DB users and job logs | Separate component/VM checks exist for selected database and PHP-FPM/Nginx paths; no single integrated panel-to-agent suite. | **Not passed end-to-end.** |
| Unsupported OS, disk and port preflight before destructive changes | Read-only `tongctl` preflight denies occupied ports in a disposable guest; there is no destructive installer flow to prove preflight ordering. | **Partial; gate open.** |
| Existing cert detection, real issue/retry/renew and external HTTP→HTTPS verification | Renderer fixtures and HTTP redirect checks only; no real certificate. | **Not passed.** |
| Staged config syntax validation and reload rollback | Standalone isolated Nginx rollback lab passes. | **Component passed; integrated host lifecycle open.** |
| Offsite backup, source destruction, clean-VM restore of data and settings | Fresh pristine-overlay MariaDB data restore passes both Ubuntu 24.04 and Debian 12 at the cited code head; the transfer is runner temp staging and no Tong settings/sites restore is included. | **Partial; not offsite or complete.** |
| Malicious archive/symlink cannot escape tenant paths | Selected path and file API tests exist; no complete jailed upload/extraction workflow and hostile archive suite. | **Not passed.** |
| Login/MFA/recovery isolation and secret-redacted audit | Internal auth/session primitives and logout route exist in drafts; no public production login/session issuance, recovery flow, trusted deployment proxy, or sign-off. | **Not passed.** |
| Phone/tablet site/domain/files tasks with guidance | Read-only Explorer component exists; no end-to-end responsive usability/accessibility acceptance. | **Not passed.** |

## P0 prerequisites and disposition

The 2026-10-29 advisory review recommends **Needs revision** for ADR-001–009 and the draft threat model, with concrete implementation evidence still required. It is not an owner decision, independent security review, penetration test or approval. Human owner/security reviewer identity, formal outcome/date and linked evidence remain **pending**. ADR-007 privilege/key custody and ADR-009 production authentication are unresolved dependencies; no public insecure login is enabled.

## Exit decision

P2 remains **IN PROGRESS**. Do not merge draft PRs, deploy to a real VPS, claim distro support/security approval, or declare Phase 1/Phase 2 production-ready on this evidence. Revisit this report only when each open slice and must-pass test has authoritative integrated evidence and the responsible human owners record the required architecture/security decisions.


## P2 package preflight ordering lab — draft PR #86 (29 October 2026)

A follow-up disposable-package test now extracts the unsigned `tongctl` binary without installing it, proves low-disk and occupied-port policies deny, and asserts package metadata/service files/state directories remain absent before the first `dpkg --install`. It then proves the supported clean guest satisfies the lab policy immediately before installation. This adds test-harness ordering evidence only; it does not create a supported installer or prove upgrade/recovery authorization. Exact-head Ubuntu/Debian CI is pending, so the gate remains **partial and open**.
