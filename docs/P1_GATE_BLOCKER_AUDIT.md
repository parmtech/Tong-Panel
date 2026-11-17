# P1 gate blocker audit (development only)

This is a deterministic, **fail-closed inventory check**, not a signer, phase approval, or release gate attestation.

- `docs/P1_GATE_SNAPSHOT.json` lists nine currently blocked P0/P1/P2 prerequisite decisions and source/issue references. Each reference is a **proposal/evidence pointer**, not a completed acceptance record.
- `python3 scripts/p1_gate_audit.py` reports `result: blocked` and exits **2** by default. An ambiguous, missing, altered or forged-approval manifest exits **3**; it never reports ready or approved.
- `python3 scripts/p1_gate_audit.py --expect-blocked` exits **0 only when the manifest still explicitly says blocked**. The corresponding GitHub workflow is a test of that invariant. A green workflow is **not** evidence that Tong Panel may be deployed.
- `python3 -m unittest discover -s scripts -p test_p1_gate_audit.py -v` exercises normal blocked reporting, missing/duplicate/extra gates, falsified approval/release claims, absent/outside evidence and malformed/oversized JSON.

Release readiness will require an independently reviewed policy, real owner/security signoff, specific verified artifacts/versions/OS matrix, trusted signed release provenance, host rollback and adversarial acceptance. Source-controlled metadata cannot prove those facts, and **this script has no mechanism for authorizing release**. A later approved release gate must be a separate reviewed implementation with authenticated external evidence rather than a checkbox flip.

See [P1 Security Acceptance](P1_SECURITY_ACCEPTANCE_2026-11-13.md), [the ADR register](ADR_REGISTER.md), [PR #171](https://github.com/parmtech/Tong-Panel/pull/171) and [master tracker #172](https://github.com/parmtech/Tong-Panel/issues/172).
