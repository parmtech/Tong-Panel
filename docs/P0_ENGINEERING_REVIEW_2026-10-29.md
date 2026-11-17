# P0 Engineering Review for Maintainer Decision

**Date:** 2026-10-29 (Australia/Sydney)  
**Reviewer:** Codex engineering review (automated, non-owner)  
**Status:** Recommendations only. This is not owner acceptance, an independent security audit, or authorization to enable production features. Owner/security review remains pending in [issue #1](https://github.com/parmtech/Tong-Panel/issues/1) and [issue #2](https://github.com/parmtech/Tong-Panel/issues/2).

## Review outcome

The proposals are directionally coherent for a single-server alpha, but they are not ready for blanket acceptance. ADR-007 and ADR-008 define the highest-risk trust boundary and release path; missing operational contracts must be resolved before privileged host mutations. ADR-009 is detailed but still has explicit open questions and missing recovery/key-custody work. P0 remains open. No production login, installer, root action, or deployment is authorized by this review.

| Decision | Engineering recommendation | Conditions / evidence needed before owner decision |
| --- | --- | --- |
| ADR-001 — original codebase | Accept directionally | Keep provenance and per-component licence review as release gates; do not infer permission to copy upstream code. |
| ADR-002 — Go API, separate agent, TS UI | Needs revision | Record supported Go/Node toolchain and maintenance owners; freeze API/agent protocol and compatibility policy. |
| ADR-003 — SQLite and durable jobs | Accept for single-node prototype; needs revision for alpha | Define backup consistency, filesystem support, restore/version policy, concurrency/load limits, and prohibit SQLite on network filesystems. |
| ADR-004 — site execution profiles | Needs revision | Specify exact PHP-FPM identity/path/socket ownership, resource limits and denial tests before site provisioning; defer arbitrary app execution. |
| ADR-005 — Ubuntu 24.04 / Debian 12 targets | Accept as test targets only | Keep both uncertified until clean install, rollback, upgrade and recovery pass on pinned images/packages. |
| ADR-006 — mail/DNS optional | Accept directionally | Define the minimal alpha service matrix and ensure optional capabilities do not silently open ports or weaken site setup. |
| ADR-007 — unprivileged API and typed root agent | Needs revision; blocking | Specify socket peer authentication, action schema/versioning, per-job tenant/object authorization recheck, filesystem capability model, audit durability, cancellation, idempotency and rollback. Fuzz and adversarial review are required before any root write action. |
| ADR-008 — signed immutable releases | Needs revision; blocking | Name signing-key custody/rotation and recovery owners, installer trust anchor/bootstrap, reproducible build and provenance verification, downgrade policy, revocation, and interrupted-install rollback tests. |
| ADR-009 — authentication and recovery | Needs revision; blocking public login | Resolve KDF sizing on the smallest candidate VM, Unicode password semantics, key custody/rotation and backup, recovery codes/reset, trusted proxy/origin model, durable limiter policy, and dependency/licence advisories. Demonstrate integrated HTTPS login/logout only after these decisions and independent review. |

## Threat model review

The threat model identifies relevant P1/P2 risks and maps them to evidence, but it is a starting register, not a complete assessment. Before P2 host lifecycle work, assign an owner and concrete acceptance test for each applicable T02, T04, T09, T10, T12 and T15 item. Add explicit threats for certificate private-key exposure, cross-tenant database socket/credential access, web-service quiescence during file recovery, quota exhaustion, and backup restore onto an already-populated host. Record residual risk and incident response, not only mitigations.

## Repository evidence checked

- `docs/ADR_REGISTER.md`, `docs/ADR-009-AUTHENTICATION.md`, `docs/THREAT_MODEL.md`, `docs/ROADMAP.md`, `docs/FEATURE_MATRIX.md`, and `docs/SECURITY.md`.
- Issues #1 and #2 have no comments recording an owner/security decision as of this review.
- The repository has no reviewed production root-agent action set or installer lifecycle; developer diagnostics and unsigned VM previews do not satisfy those gates.

## Required owner/security action

For each ADR, record Accepted, Rejected, or Needs revision, reviewer name, date, evidence, residual risks, and issue/PR link in `docs/ADR_REGISTER.md` and issue #1. Record threat-model review and assigned mitigations in issue #2. Until then, this document changes no approval status and P0 stays blocked.
