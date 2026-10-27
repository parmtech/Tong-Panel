# Tong Panel — Architecture Decision Register
**Revision:** 2026-10-26 | **Status:** DRAFT DECISIONS — proposals, not approved.  
Approval requires owner/maintainer review and a recorded accepted/rejected outcome. Changes to high-risk architecture must not be merged into production before review. All choices below can be revised in issue #1.

## ADR-001 — Original control panel or wholesale fork?
**Proposed:** Original Tong Panel AGPL-3.0 codebase; standard Linux OS services managed through typed adapters; only include specifically reviewed permissive/copyleft-compatible libraries and preserve attribution.  
**Rejected for now:** stitching full Hestia/Virtualmin/ISPConfig forks together; copying their panels' artwork or cPanel trademarks.  
**Why:** conflicting interfaces, privilege boundaries, operational assumptions and possible licence exceptions; a clean interface lets us separately swap upstream service adapters.  
**Trade-off:** more engineering time; greater responsibility to validate parity and security.  
**Verification:** dependency provenance ledger required before importing code; review legal compatibility of each combined component.

## ADR-002 — Language and service boundaries
**Proposed:** Go API (unprivileged) + small separate Go privileged host agent + TypeScript/React UI. Standard JSON/OpenAPI contract and typed host commands.  
**Alternatives:** Laravel/PHP monolith (familiar, but Linux root boundary still essential), Rust for daemon (strong, slower team onboarding), single-process Node (root privilege coupling risk).  
**Why:** Go binary deployments, reliable low-level OS interfaces; React supports interactive file tree, search and responsive components.  
**Review questions:** code-owner expertise, release support versions, signed packages and maintenance burden.  
**Safe prototype:** Go localhost health endpoint, no host writes and no root agent functionality.

## ADR-003 — Storage and jobs
**Proposed:** small-server-first embedded SQLite for panel metadata with a transactional durable job table and file-backed backup/upgrade strategy; offer PostgreSQL datastore profile only after testing.  
**Alternatives:** PostgreSQL-only (robust concurrency but larger footprint); Redis-only queue (risk of lost job lifecycle without durable store).  
**Why:** one-command install on minimal VPS and simplified initial backup.  
**Hazards:** SQLite contention and network filesystems; **single writer/reconciliation** and no SQLite-over-NFS guarantee; workload benchmarks decide final choice.  
**Acceptance:** migration/retry/idempotency/concurrency/kill-process tests.

## ADR-004 — Application execution
**Proposed:** PHP-FPM/site Unix identities first, systemd scoped Node/Python native profiles next; optional rootless-container sandbox for apps with compatible platform support. Builds separately isolated.  
**Alternatives:** container-only (high footprint; difficult multi-user mail/PHP support); native-only (weaker sandbox for arbitrary apps).  
**Hard rule:** don't mount host root, grant privileged containers or give tenants Docker socket.  
**Acceptance:** hostile workload denial of cross-tenant file/credential/process access.

## ADR-005 — Distro/packaging scope
**Proposed:** fresh Ubuntu 24.04 LTS and Debian 12 as first test targets, x86_64; Debian 13/arm64 follow verified CI; add AlmaLinux/Rocky later. Pin OS package versions and record compatibility.  
**Alternatives:** universal installer (unverifiable package/SELinux/firewall assumptions); Docker-only host (breaks conventional hosting).  
**Status:** **target matrix only — no distro currently certified**.  
**Acceptance:** clean/failed preflight, port collision, package rollback and interrupted install tests in each supported VM.

## ADR-006 — Mail and DNS modules
**Proposed:** Nginx/PHP/MariaDB minimal profile; external DNS supported first; Postfix+Dovecot and BIND/PowerDNS as separately installable capabilities; mail/DNS optional.  
**Why:** deliverability/DNS complexity and unnecessary exposure on tiny VPS.  
**Alternatives:** force full hosting mail stack on everyone (unacceptable resource and network assumptions).  
**Acceptance:** opt-out still allows website; mail and authoritative DNS independently tested in compatible VPS.

## ADR-007 — Privilege boundary and secrets
**Proposed:** panel API as unprivileged UID, root agent via tightly permissioned local Unix socket, peer identity check, allowlisted typed operations with canonical paths, staged config, validation and audit; per-tenant secrets encrypted at rest, key separation and documented rotation.  
**Alternatives:** root web server / unrestricted sudo (unacceptable).  
**Acceptance:** fuzzed malformed inputs, cross-tenant denial, symlink/TOCTOU protection, redaction and replay prevention; adversarial review before allowing any root writes.

## ADR-008 — Installer/supply chain
**Proposed:** immutable signed tagged releases, installer bootstrap verifies signature/checksums and OS preflight BEFORE privileged execution; journals/rollback and short-lived admin bootstrap; unattended mode opt-in.  
**Alternatives:** curl a moving main branch to sudo bash (unacceptable without provenance/integrity).  
**Acceptance:** package swap/key compromise simulation, clean install/upgrade/recovery in exact supported OS test matrix.

## ADR-009 — Authentication, sessions and recovery
**Proposed:** `docs/ADR-009-AUTHENTICATION.md` defines Argon2id PHC policy, bounded verification, generic failures, rate limits, MFA replay/recovery, revocable SQLite sessions, CSRF/Origin checks, tenant revalidation and bootstrap.
**Status:** DRAFT / NOT APPROVED. The isolated Argon2id and SQLite session primitives are prototypes only; do not wire a login route until owner/security approval and the listed acceptance evidence exist.
**Review:** issue #6 and issue #1; require benchmark evidence on the smallest supported VM, review of proxy/bootstrap/key handling and negative auth tests.

## Owner decision checklist
For each ADR record **Accepted / Rejected / Needs revision**, decision date, reviewers, evidence, alternatives considered, security impact, migration story and link to PR/issues. Until recorded here and in issue #1, all proposals remain non-binding. No code from cPanel/Hestia/Virtualmin should be imported merely because a proposed ADR exists.

## Change control
New architectural changes require ADR-009 onward, an issue/PR, threat/security and licence review, linked verification, UPDATES.md handoff. This register is the single index; issue #1 is the signoff discussion.
