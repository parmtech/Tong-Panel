# Advisory architecture and threat-model review — 2026-10-29

**Reviewer:** Codex technical review, based on the repository drafts and current progress journal. This is an advisory implementation review, not an owner decision, independent security assessment, penetration test, or approval. No maintainer/security reviewer has been identified in the evidence examined; all formal decisions remain **pending**.

## Recommendations

| Item | Advisory outcome | Evidence-based finding and next acceptance evidence |
|---|---|---|
| ADR-001 — project/provenance | Needs revision | The original-codebase direction and attribution rule are reasonable. The decision needs a maintained dependency/source provenance ledger and a license compatibility review tied to release artifacts before distribution. |
| ADR-002 — language/boundaries | Needs revision | Unprivileged Go API plus a separate privileged agent is the right boundary to preserve. Define the Unix peer-identity mechanism, typed operation/version contract, socket ownership, and failure/rollback behavior before enabling root writes. |
| ADR-003 — storage/jobs | Needs revision | SQLite is a plausible single-server choice. Pin migration compatibility, single-writer/lock/restart semantics, idempotency and backup consistency; test restore from a real consistent SQLite snapshot and document the limits of filesystem/NFS placement. |
| ADR-004 — execution isolation | Needs revision | The no-root-mount/no-Docker-socket rule is sound. PHP-FPM tenant isolation evidence is useful but does not prove hostile application containment. Define supported runtime profiles and test process, filesystem, network, resource and secret boundaries with adversarial workloads. |
| ADR-005 — distro/package scope | Needs revision | Ubuntu 24.04 and Debian 12 are test targets, not certified support. Keep that distinction; require clean install, upgrade/downgrade, interrupted-operation recovery, package signature verification and restore on each declared architecture before support claims. |
| ADR-006 — mail/DNS scope | Needs revision | Optional mail and external DNS reduce initial host exposure. Specify the supported first-release feature boundary and ownership for DNS credentials, ACME challenge issuance/renewal, expiry alarms and failure rollback; current previews and challenge-path checks do not establish issuance or renewal. |
| ADR-007 — privilege/secrets | Needs revision | This is the highest-risk boundary. Do not enable production root operations until peer authentication, per-operation authorization, canonical path/race defenses, audit/redaction, external key custody, rotation, recovery and adversarial tests are reviewed together. Encrypted backup primitives do not by themselves define safe key custody. |
| ADR-008 — installer/supply chain | Needs revision | Signed immutable releases and preflight-before-privilege are appropriate. Provide an independently verifiable signing-key custody/rotation/revocation and recovery plan, then verify tamper rejection and rollback on clean supported VMs. Current package fixtures are unsigned developer previews. |
| ADR-009 — authentication | Needs revision | The draft calls out the right major controls and explicitly blocks public login. Before integration, benchmark Argon2 admission on the smallest intended VM, decide proxy/origin configuration and bootstrap/recovery/key backup, implement recovery-code lifecycle, and test the full tenant/role/job matrix and session rotation/revocation. |
| Threat model | Needs revision | The register is a useful starting point, but its own exit gate is unmet: assign an owner and issue/test contract to every high-impact threat. Expand explicit cases for database/key backup restore, signing-key compromise/recovery, session fixation/replay, proxy misconfiguration, malicious restore input and dependency-update response. Map each claim to evidence and residual risk. |

## P0 disposition

No item above is marked Accepted or Rejected. Recommended status is **Needs revision** for the technical drafts; formal ADR and threat-model owner/security decisions remain **pending** until a responsible human reviewer records outcomes, date, rationale and evidence in the ADR register and linked issues. This assessment alone cannot close that gate.
