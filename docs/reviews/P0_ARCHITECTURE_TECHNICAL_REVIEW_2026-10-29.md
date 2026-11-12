# P0 Architecture and Threat Model — Technical Review Notes

**Review date:** 2026-10-29 (Australia/Sydney)  
**Reviewer:** Codex engineering assistant (independent desk review; not the project owner or an authorized security approver)  
**Sources:** `docs/ADR_REGISTER.md`, `docs/ADR-009-AUTHENTICATION.md`, and `docs/THREAT_MODEL.md` at the stacked P2 development source.  
**Authority:** Recommendations only. No ADR is accepted or rejected by this note. Owner/security review remains **PENDING** and the threat model remains **DRAFT / UNREVIEWED**.

## Summary

The proposals establish useful guardrails: an unprivileged panel, typed privileged operations, tenant checks, no default credentials, signed releases, and explicit negative-test evidence. They are not ready to authorize production or host-mutating work. The strongest unresolved cross-cutting dependencies are privileged-agent protocol and confinement, external key custody and recovery, signed-release trust and rotation, and real failure/restore evidence on declared supported guests.

Recommended disposition for ADR-001 through ADR-009: **Needs revision before owner/security decision**. This is a technical recommendation, not an owner decision. Preserve each ADR's formal status as proposed until the responsible human records the decision, date, reviewer and evidence in the register and issue #1/#6 as applicable.

## ADR-by-ADR findings

| ADR | Recommended technical disposition | Evidence needed before an owner decision |
| --- | --- | --- |
| 001 — Original panel or fork | **Needs revision.** The clean-room direction is reasonable, but the required dependency provenance and licence/attribution review are process promises rather than a present inventory. | Dependency/SBOM inventory, source provenance, licence compatibility review and an explicit maintenance/support owner for imported components. |
| 002 — Language and service boundaries | **Needs revision.** Separating the Go API and privileged agent is the right boundary; the local socket concept alone does not define the agent's authenticated wire contract or downgrade behavior. | Versioned typed request schema, peer identity and socket ownership across target distros, request limits/timeouts, replay/idempotency rules, error redaction and denial tests. Keep privileged actions disabled until independently reviewed. |
| 003 — Storage and jobs | **Needs revision.** SQLite is plausible for a single-node first release, but the backup/upgrade story and single-writer operational envelope are not demonstrated. | Representative concurrency/contention tests, crash during migration/commit drills, verified consistent backup and fresh-guest restore, integrity checks, and explicit limits (including no NFS). |
| 004 — Application execution | **Needs revision.** The prohibition on host-root mounts and Docker socket access is clear; the supported isolation profile for hostile tenant code is not specified enough to enforce. | Per-profile UID, filesystem, network, cgroup/resource and syscall policy; escape/denial tests; cleanup after interrupted builds; documented exclusions for unsupported workloads. |
| 005 — Distro and packaging | **Needs revision.** The target matrix is explicitly not certified. The disposable unsigned package lab (#74) is useful but does not establish a supported installation, rollback or recovery guarantee. | Defined support policy and pinned image/package baseline; clean install, interrupted install, upgrade, rollback and recovery on each declared guest; reproducible artifacts and clear unsupported-host behavior. |
| 006 — Mail and DNS | **Needs revision.** Optional capability boundaries are sensible, but no concrete per-module trust, credential scope, outbound behavior or failure policy is specified. | Threat boundaries and least-privilege provider credentials, DNS/MX failure and abuse tests, opt-out behavior, resource estimates, and operational recovery per enabled module. |
| 007 — Privilege and secrets | **Needs revision.** The right goals are stated, but encryption, key custody, rotation, backup separation and loss/recovery are still open; these block safe credential-bearing provisioning. | Key source and separation from SQLite backups, rotation/revocation procedure, restore and key-loss drills, secret redaction checks, canonical-path/symlink/TOCTOU adversarial tests, and an independent review of every agent operation. |
| 008 — Installer and supply chain | **Needs revision.** Signature verification before privileged execution is necessary, but the trust root and lifecycle for signing keys and downgrade prevention are not defined. | Offline signing-key custody and rotation/revocation, immutable provenance/SBOM, signature/checksum tamper tests, rollback/downgrade policy, and install/recovery tests on supported guests. |
| 009 — Authentication and recovery | **Needs revision.** The draft itself correctly prohibits production use. Public login must remain blocked until the stated owner review, smallest-VM KDF/concurrency measurements, external MFA/recovery key design and integrated session/CSRF/proxy tests are evidenced. | Resolve the draft's outstanding questions (KDF budget and input semantics, key custody and restore, proxy/origin trust, recovery, limiter persistence and dependency baseline); demonstrate generic failures, replay denial, revocation and browser CSRF/Origin behavior; record owner/security decision. |

## Threat model review

The initial model is a useful inventory of actors, assets, boundaries and high-impact failures. It correctly states that the register is not evidence of mitigation. Recommended disposition: **Needs revision before security approval**.

1. Add concrete abuse cases and data-flow diagrams for authentication/bootstrap/recovery, privileged-agent calls, uploads/extraction, outbound DNS/ACME, backups/restores, and package updates. Identify the component that enforces each control.
2. Extend coverage for SSRF/DNS rebinding and outbound egress; resource exhaustion across KDF, API bodies, SQLite, file storage and background jobs; symlink races and archive bombs; credential/key loss or compromise; log/support-bundle retention and redaction; and partial host changes that strand SSH or TLS.
3. For every mitigation, name a responsible workstream and executable negative/fault test. Distinguish implemented controls, prototype-only controls, planned controls and external operational dependencies.
4. Define residual-risk acceptance and incident response (key compromise, tenant cross-read, malicious release, failed restore). The draft currently has no assigned accountable owner or acceptance thresholds.

## Gate and limits

- No public login route, tenant DB provisioning, privileged host writes, production install or real VPS deployment is authorized by this technical note.
- PR #75 is metadata-only and does not resolve ADR-007 credential custody or authorize MariaDB creation.
- P0 owner/security decisions are still required. A maintainer with authority must record decisions, reviewer identity, date and linked evidence; this assistant cannot supply that signoff.
- Phase 1 and Phase 2 remain incomplete. This review is not security approval, penetration testing, or release certification.
