# Phase 5 prerequisite assessment and acceptance test specification

**Assessment date:** 2026-11-10 (Australia/Sydney)
**State:** design and test specification only; Phase 5 entry is **BLOCKED**.
**Evidence baseline:** GitHub `main` at `0aad68e81138a0f226e83e401f90bae4d0b6bddc`; Phase 4 checkpoint `docs/p4-hosting-parity-design` at `0029b39aa5ebb2056506892b18b400a31402977b`.

This document translates the existing Phase 5 plan into independently verifiable gates and test scenarios. It does not approve architecture, authorize privileged work, assert implementation, or change any entry gate. A passing prototype test cannot substitute for the prerequisite evidence below.

## Independently checked entry gates

| Prerequisite | Current finding | Evidence required before dependent implementation |
|---|---|---|
| P0 architecture and threat review | **Not met.** The current P1 exit report says ADR-001 through ADR-009 and the threat model remain unapproved. | Recorded maintainer/security decisions, threat-model review, and approved boundaries for identity, secrets, agent trust, plugin capabilities, and recovery. |
| P1 foundation | **Not met.** The 2026-11-10 journal describes active draft P1 work and explicitly says Phase 1 is not complete. The report lists public authentication/key custody, identity policy and agent binding, host-effect recovery, signed installer/recovery, and UI/security acceptance as open. P1 PR #143's non-VM checks passed, but its clean Ubuntu/Debian smoke jobs in run #42781985275 were still in progress when independently checked. | Accepted P1 exit report with exact-head evidence for all required authentication, authorization, agent, job, audit, backup, installer, recovery, accessibility, and security criteria. Draft PR or unit-test success alone is insufficient. |
| P2 production-like single-server recovery | **Not met; narrow positive lab evidence exists.** P1 report says Phase 2 entry remains blocked. Draft PR #89 run #42631444449 passed Ubuntu 24.04 and Debian 12 guest smoke for isolated WordPress installs and tenant DB separation. Draft PR #91 run #42633862164 passed both guests for an encrypted WordPress restore slice with homepage/upload checks and private-config/privileged-DB denials. These are unsigned preview workflows and do not prove signed install, full lifecycle, complete restore, rollback, or P2 exit. | Supported disposable-host install/upgrade/rollback, two-tenant service lifecycle and isolation, complete backup/restore validation, and incident/recovery runbooks with measured outcomes. |
| P3 app execution and isolation | **Not met.** The current `docs/p3-prerequisite-evidence` branch adds a detailed entry/test plan and explicitly records **P3 entry: NOT PROVEN**; it authorizes no runtime/build/host-mutation implementation. Its source review confirms P1/P2 gaps remain. P3 depends on accepted P2. | Hostile-app isolation, deploy/rollback, secrets separation, resource/port conflict and recovery evidence on supported disposable hosts, plus accepted P1/P2 producer contracts. |
| P4 operational parity and incident procedures | **Not met.** The Phase 4 plan at the pinned checkpoint is explicitly **PLANNED, NOT IMPLEMENTED** and depends on P2/P3. Draft PR #146 refreshes gate evidence, records the narrow P2 runs above, and still states P4 is blocked/design-only. P4 requires tested mail/DNS/database/migration/quotas and a full-hosting reference deployment. | Accepted P4 exit evidence for eligible hosts, service failure modes, tenant/reseller limits, migration rollback, backup restoration, audit/export, and incident procedures. |
| Phase 5-specific approval | **Not met.** P5 is explicitly planned, with no accepted implementation or entry review recorded. Existing open P1 drafts were searched; no P5 PR or P5 branch was returned by the connected GitHub searches at assessment time. | Reviewed P5 threat analysis, interfaces, data ownership and recovery contracts; approved scoped issue(s), owner, dependencies, rollback strategy, and non-production test environment. |

**Decision:** Restrict current P5 work to this design/test specification and isolated non-privileged prototypes that cannot access real nodes, credentials, production networks, or privileged operations. Recheck every gate against GitHub before implementation begins; this assessment is a point-in-time snapshot.

## Scope-to-evidence traceability

This matrix maps the canonical P5 scope bullets to prerequisite evidence and the scenario IDs above. “Depends on” names required foundations, not authorization to start implementation.

| P5 scope | Required upstream evidence before implementation | Acceptance scenarios | Remaining review/design output |
|---|---|---|---|
| Remote agent enrollment, per-server identity, mutual TLS, revocation, rotation, scoped actions | P0 trust/secret decisions; P1 identity, authorization, audit and durable-job acceptance; P2/P4 operational incident and recovery procedures | 1–8, 10 | Reviewed enrollment lifecycle, identity binding, expiry/revocation propagation, lease limits, key custody and incident response. |
| Node inventory, placement, health, labels and maintenance | Accepted P1 job authorization and cancellation/reconciliation; P2/P3 workload/resource inventory and supported-host behavior | 6–10 | Inventory freshness contract, capacity facts, eligibility semantics and behavior for unknown/stale data. |
| Workload moves, DNS/SSL cutover, app/DB backup handoff and rollback | P2 measured backup/restore and service lifecycle; P3 deploy/rollback and app isolation; P4 DNS, database, migration and quota acceptance | 11–15 | Migration state ownership, consistency checks, dual-write/divergence treatment, cutover authority, rollback limits and measured RPO/RTO. Existing PR #91 restore evidence is one narrow unsigned lab slice and does not satisfy this dependency. |
| Mirrored DNS, mail and database designs | P4 tested single-host service contracts and failure handling; approved service-specific consistency and recovery decisions | 5–6, 13, 15, 23–24 | Separate topology and failure-boundary review per service; no generic HA claim from control-plane replication. |
| Marketplace templates and third-party plugin framework | P0 supply-chain/privilege decisions; P1 auth/audit foundations; P3 app isolation; reviewed licensing and artifact provenance | 16–20 | Separate declarative app-recipe and executable panel-plugin threat models, permission review, provenance, moderation, deprecation and recovery policy. |
| Usage reporting, tenant analytics and billing/provisioning APIs | P1 tenant identity/audit; P2/P3/P4 stable ownership and resource-accounting sources | 7, 21–22 | Define event authority, correction/reconciliation policy, privacy/retention boundaries and ensure accounting cannot directly cause privileged provisioning without authorization. |
| Notifications, audit export and ecosystem APIs | P1 reviewed identity, authorization, durable events and audit integrity; P4 operational event contracts | 7–8, 21–22 | Versioned contract, tenant scope, replay/idempotency, redaction, retention and delivery-failure semantics. |
| Additional distributions and arm64 | Existing supported-platform matrix and phase-specific automated install, upgrade, rollback, recovery and workload tests | 2, 11–15, 24 | A distinct evidence package for each OS/architecture pair; no inference from another target. |

The traceability map is a planning aid, not a substitute for prerequisite closure. At this assessment, the Phase 1 report still describes Phase 1 as incomplete; P3's new design branch still says entry is not proven; the P4 plan remains planned/not implemented; and the pinned Phase 4 branch is a design checkpoint rather than exit evidence. The two P2 clean-guest runs above are useful limited evidence, not phase acceptance. Therefore every row remains **not authorized for production implementation**.

## Proposed test architecture (for review)

Use deterministic, local-only fakes for control-plane logic, a disposable isolated network for protocol/integration tests, and synthetic tenant/node data. Every test must record the exact source commit and configuration. Tests involving persistence, time, rotation, revocation, retry, or migration should use controllable clocks and failpoints. No test may use real server credentials, contact a production node, mutate a host, or claim HA from a simulated component.

## Phase 5 acceptance scenarios

### Node identity, enrollment, rotation, and revocation

1. **One-time enrollment:** accept one valid, short-lived enrollment proof; reject reuse, expiry, malformed proof, wrong control-plane audience, and a proof issued for another node. A failed attempt must not leave a registered or partially trusted node.
2. **Mutual identity:** accept only the expected node identity and trust root; reject unknown, expired, not-yet-valid, wrong-purpose, and wrong-node certificates. Do not fall back to bearer-only or unauthenticated transport.
3. **Rotation:** exercise overlap and retirement windows using a fake clock; prove old material is rejected after retirement and no rotation secret is logged or persisted in plaintext.
4. **Revocation:** revoke a node while connected and while disconnected; after reconnection, deny new jobs and credentials. Verify stale cached authorization cannot restore access and revocation is auditable.
5. **Control-plane partition:** partition control-plane-to-node traffic before, during, and after a lease. Verify no ghost-success, no unbounded authorization lease, explicit unknown outcome for uncertain effects, and safe reconnection reconciliation.
6. **Node partition/offline:** mark health and job state honestly when a node is offline; retries must not duplicate non-idempotent work. The UI/API must distinguish unreachable, pending, failed, and externally verified success.

### Authorization and placement

7. **Cross-node/tenant denial:** submit a valid user/job for a different node or tenant and assert denial before dispatch, no metadata leakage, no remote side effect, and an attributed audit record.
8. **Policy change in flight:** narrow permissions or remove placement eligibility between enqueue and execution; execution-time authorization must deny, with a stable reason code and no arbitrary command path.
9. **Placement determinism:** given inventory, labels, constraints, health, and capacity, verify deterministic eligible-node selection; unknown/stale inventory and conflicting constraints fail closed.
10. **No shell escape:** fuzz the typed job/action protocol with unknown action, extra fields, path/argument injection, oversized frames, duplicate keys, replayed IDs, and malformed encodings. No generic shell/exec operation is accepted.

### Migration and recovery

11. **Staged migration:** copy synthetic app/database state to a destination, verify checksums and application-level invariants before cutover, and preserve source state until explicit validation.
12. **Cutover interruption:** inject failure at every stage (snapshot, transfer, restore, validation, DNS/route switch, source retirement). Report the last verified stage; avoid claiming success when outcome is unknown.
13. **Rollback:** before and after cutover, exercise rollback with writes on both sides. Detect divergence and require reconciliation rather than silently discarding data. Verify documented RPO/RTO measurements are derived from observed tests.
14. **Credential rotation during move:** rotate/revoke node and service credentials during transfer; verify only the authorized stage continues and all temporary material is removed or invalidated on abort.
15. **Recovery after process restart:** restart coordinator and node agents at each migration checkpoint; reconcile durable state without duplicate cutover or loss of audit lineage.

### Plugins, marketplace, and ecosystem

16. **Manifest and signature policy:** reject unsigned, untrusted, mutable/unpinned, expired, wrong-target, downgraded, or incompatible plugin artifacts before installation. Verify dependency and licence metadata is present and reviewable.
17. **Capability sandbox:** grant a plugin only declared scoped capabilities; prove denied node, tenant, file, secret, network, and API access cannot be reached through direct or confused-deputy paths.
18. **Disable and uninstall:** disable/revoke capabilities immediately, preserve user-owned data with an explicit recovery path, and prove panel boot and unrelated plugins continue. Uninstall must not delete unowned data.
19. **Update and rollback:** stage a signed update, validate compatibility, inject failure, and restore the prior known-good version without restoring revoked permissions or secrets.
20. **Registry outage/compromise:** test unavailable registry, stale metadata, key revocation, and an artifact whose digest differs from reviewed metadata. Fail closed and preserve audit evidence.

### Usage, billing integrations, and HA claims

21. **Usage accounting:** reconcile synthetic usage across node restart, delayed/offline reports, duplicate events, clock skew, and tenant transfer. Keep source measurements and derived totals attributable; do not emit billing actions from unverified usage.
22. **External billing/provisioning:** use a fake provider and signed webhook fixtures; test replay, duplicate delivery, timeout, reordered events, and idempotent reconciliation. Billing failure must not silently grant privileges or destroy hosted data.
23. **Mirrored services:** for each proposed DNS/mail/database topology, declare consistency and failure semantics, then test split brain, stale replica, quorum loss, failover, and recovery. Document when an operation is unavailable or read-only.
24. **HA boundary:** publish no HA claim unless workload-specific failover and recovery tests establish explicit topology, failure domain, measured RPO/RTO, and known limitations. Management across several nodes alone is not HA evidence.

## Evidence and completion record

Each accepted test should retain: scenario ID, source commit, test harness/environment, synthetic fixture description, expected and observed result, logs with secrets removed, cleanup result, and reviewer. Failures and retries remain in the record. A P5 release decision additionally requires the plan's partition/offline/revocation, wrong-node/tenant, migration consistency/rollback, plugin disable/recovery, supply-chain, and HA-boundary criteria to pass on the reviewed implementation. No completion claim is made by this specification.

## Source references

- `docs/BLUEPRINT.md` — staged scale and trust-boundary intent.
- `docs/ROADMAP.md` and `docs/PHASE_INDEX.md` — phase dependencies and gates.
- `docs/phases/P5-MULTISERVER-ECOSYSTEM.md` — canonical P5 scope and acceptance.
- `docs/phases/P4-HOSTING-PARITY.md` at checkpoint `0029b39aa5ebb2056506892b18b400a31402977b` — P4 dependency and acceptance.
- `docs/P1_EXIT_GATE_REPORT_2026-10-27.md` on the current P1 draft stack — current gate evidence and remaining blockers.
- Draft PR #146 and its branch `docs/p4-hosting-parity-continuation` — refreshed Phase 4 gate snapshot and exact upstream evidence references.
- Branch `docs/p3-prerequisite-evidence`, `docs/P3_ENTRY_AND_ACCEPTANCE_TEST_PLAN.md` — P3 entry remains not proven; this is a design artifact, not P3 implementation.
- P2 draft PRs #89/#91 and clean-guest workflow runs #42631444449/#42633862164 — narrow unsigned preview evidence; exclusions are recorded above.
- `UPDATES.md` on GitHub `main` — latest committed journal baseline; active draft-branch outcomes must be rechecked before gate decisions.
