# Phase 5 prerequisite assessment and acceptance test specification

**Assessment date:** 2026-11-10 (Australia/Sydney)
**State:** design and test specification only; Phase 5 entry is **BLOCKED**.
**Evidence baseline:** GitHub [`main` commit `0aad68e81138a0f226e83e401f90bae4d0b6bddc`](https://github.com/parmtech/Tong-Panel/commit/0aad68e81138a0f226e83e401f90bae4d0b6bddc); preserved Phase 4 checkpoint [`0029b39aa5ebb2056506892b18b400a31402977b`](https://github.com/parmtech/Tong-Panel/commit/0029b39aa5ebb2056506892b18b400a31402977b) on `docs/p4-hosting-parity-design`.

This document translates the existing Phase 5 plan into independently verifiable gates and test scenarios. It does not approve architecture, authorize privileged work, assert implementation, or change any entry gate. A passing prototype test cannot substitute for the prerequisite evidence below.

## Independently checked entry gates

| Prerequisite | Current finding | Evidence required before dependent implementation |
|---|---|---|
| P0 architecture and threat review | **Not met.** The current P1 exit report says ADR-001 through ADR-009 and the threat model remain unapproved. | Recorded maintainer/security decisions, threat-model review, and approved boundaries for identity, secrets, agent trust, plugin capabilities, and recovery. |
| P1 foundation | **Not met.** The latest P1 exit report says Phase 1 is incomplete and Phase 2 entry is blocked. It lists public authentication/key custody, identity policy and agent binding, host-effect recovery, signed installer/recovery, and UI/security acceptance as open. PR #143 exact head `482017005f9872c1fe7f2969ebfb477ea7de30dd` passed both unsigned preview guests in run #42781985275. Later draft PR #149 head `8764d0fef218a36a4cc9411e5481656d014cdf0c` has a clean-VM rerun active in run #42785167173; its prior run recorded an Ubuntu package-mirror size failure and a Debian homepage timeout, so those failures are not counted as passing. Even a successful rerun would be preview-slice evidence only, not a signed installer, supported production install, complete upgrade/recovery, or P1 exit. | Accepted P1 exit report with exact-head evidence for all required authentication, authorization, agent, job, audit, backup, installer, recovery, accessibility, and security criteria. Draft PR or unit-test success alone is insufficient. |
| P2 production-like single-server recovery | **Not met; narrow positive lab evidence exists.** P1 report says Phase 2 entry remains blocked. Draft PR #89 run #42631444449 passed Ubuntu 24.04 and Debian 12 guest smoke for isolated WordPress installs and tenant DB separation. Draft PR #91 run #42633862164 passed both guests for an encrypted WordPress restore slice with homepage/upload checks and private-config/privileged-DB denials. These are unsigned preview workflows and do not prove signed install, full lifecycle, complete restore, rollback, or P2 exit. | Supported disposable-host install/upgrade/rollback, two-tenant service lifecycle and isolation, complete backup/restore validation, and incident/recovery runbooks with measured outcomes. |
| P3 app execution and isolation | **Not met.** Latest checked P3 evidence branch head [`0931292d30fcc68210eb9f5383dd1ccace4ac80a`](https://github.com/parmtech/Tong-Panel/commit/0931292d30fcc68210eb9f5383dd1ccace4ac80a) retains the explicit **P3 entry: NOT PROVEN** decision. No runtime/build/host-mutation implementation is authorized. P3 depends on accepted P2. | Hostile-app isolation, deploy/rollback, secrets separation, resource/port conflict and recovery evidence on supported disposable hosts, plus accepted P1/P2 producer contracts. |
| P4 operational parity and incident procedures | **Not met.** The Phase 4 checkpoint remains [`0029b39aa5ebb2056506892b18b400a31402977b`](https://github.com/parmtech/Tong-Panel/commit/0029b39aa5ebb2056506892b18b400a31402977b), explicitly **PLANNED, NOT IMPLEMENTED**. Latest checked continuation branch head [`34d0076bf89502d965878bb3c6b2c4e78a63b71f`](https://github.com/parmtech/Tong-Panel/commit/34d0076bf89502d965878bb3c6b2c4e78a63b71f) still says P4 is BLOCKED / design only, with P1/P2/P3 handoffs pending. P4 requires tested mail/DNS/database/migration/quotas and a full-hosting reference deployment. | Accepted P4 exit evidence for eligible hosts, service failure modes, tenant/reseller limits, migration rollback, backup restoration, audit/export, and incident procedures. |
| Phase 5-specific approval | **Not met.** P5 is explicitly planned, with no accepted implementation or entry review recorded. Existing open P1 drafts were searched; no separate P5 branch or pull request was found. Draft P5 design PR #147 and issue #148 are ours and have no human review recorded. | Reviewed P5 threat analysis, interfaces, data ownership and recovery contracts; approved scoped issue(s), owner, dependencies, rollback strategy, and non-production test environment. |

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

Use deterministic, local-only fakes for control-plane logic, a disposable isolated network for protocol/integration tests, and synthetic tenant/node data. Every test must record the exact source commit and configuration. Tests involving persistence, time, rotation, revocation, retry, or migration should use controllable clocks and failpoints. No test may use real server credentials, contact a production node, mutate a host, or claim HA from a simulated component. The standalone `prototypes/p5-outcome-model` package explores only conservative outcome reporting after a lost remote response; `prototypes/p5-plugin-capability-model` explores exact-scope plugin authorization and disable behavior. Both are non-normative, offline-only, and have no product-runtime integration.

### Candidate fault-injection oracle for scenario 5 (not a protocol contract)

| Injected failure point | Coordinator's safe view | Candidate retry/recovery rule |
|---|---|---|
| Before dispatch is attempted | Not dispatched | Dispatch only after current authorization and node eligibility are checked. |
| Dispatch attempted; no response received | Unknown | Reconcile by stable operation identity. Replay only the same key, node, tenant, action and immutable request when worker-side deduplication is durable. |
| Node accepted the operation; acceptance response lost | Unknown | Do not issue a new operation identity; query/replay under the reviewed deduplication contract. |
| Effect may have happened; completion response lost | Unknown | Do not infer failure from timeout or node offline status. Verify/reconcile before terminal success or another effect. |
| Worker reports completion but external effect is not verified | Unknown | Keep a pending/uncertain status and expose the missing verification evidence. |
| Required external effect is independently verified | Succeeded | Record verification source, timestamp, operation identity, and actor; late stale responses must not downgrade the record. |
| Node revocation occurs while disconnected or work is in flight | No new dispatch; in-flight outcome remains unknown until resolved | Reauthorize queued work on reconnect. Apply the reviewed cancellation/quiescence rule to running work; never treat disconnection as cancellation proof. |

The implementation must define a correlation mechanism that makes duplicate, delayed, and reordered responses attributable to the same operation without allowing stale evidence to overwrite a newer revocation or verified result. Selection of that mechanism, the authoritative effect verifier, and the exact state names require P0/P1/P5 review. The table is an acceptance-design aid only.

## Phase 5 acceptance scenarios

### Node identity, enrollment, rotation, and revocation

1. **One-time enrollment:** accept one valid, short-lived enrollment proof; reject reuse, expiry, malformed proof, wrong control-plane audience, and a proof issued for another node. A failed attempt must not leave a registered or partially trusted node.
2. **Mutual identity:** accept only the expected node identity and trust root; reject unknown, expired, not-yet-valid, wrong-purpose, and wrong-node certificates. Do not fall back to bearer-only or unauthenticated transport.
3. **Rotation:** exercise overlap and retirement windows using a fake clock; prove old material is rejected after retirement and no rotation secret is logged or persisted in plaintext.
4. **Revocation:** revoke a node while connected and while disconnected; immediately stop new control-plane dispatches, and after reconnection deny new jobs and credentials. Reauthorize queued jobs at execution. Exercise the reviewed policy for already-running work; if its effect cannot be stopped or verified, report unknown rather than cancelled or successful. Verify stale cached authorization cannot restore access, revoked authorization denies replay, and revocation is auditable. Do not treat a disconnected node as proof that in-flight work stopped.
5. **Control-plane partition:** partition control-plane-to-node traffic before, during, and after a lease. Verify no ghost-success, no unbounded authorization lease, explicit unknown outcome for uncertain effects, and safe reconnection reconciliation. Replay after a lost response must retain the same idempotency key, target node, tenant, action, and immutable request; changed bindings or missing durable node deduplication must deny replay. The offline candidate classifier is in `prototypes/p5-outcome-model`.
6. **Node partition/offline:** mark health and job state honestly when a node is offline; retries must not duplicate non-idempotent work. The UI/API must distinguish unreachable, pending, failed, and externally verified success.

#### Review-only enrollment and revocation evidence contract for scenarios 1–5

Use synthetic identities, certificates, clocks, queues, and node state. The owner/security review must first approve the enrollment proof, trust-root/key custody, rotation overlap, authorization lease, and already-running-job policy. These cases define observable outcomes without selecting a certificate authority, transport library, or lease duration.

| Sequence or fault | Expected test result | Evidence to retain |
|---|---|---|
| Two requests redeem the same enrollment proof concurrently, or a completed proof is replayed | Exactly one node binding is created; the losing/replayed attempt creates no partial identity or usable credential. | Proof ID, bound node/audience, request order, resulting identity inventory, and redacted denial/audit record. |
| Credential is valid but bound to another node, control-plane audience, or trust purpose | Reject before job dispatch; no fallback identity mode or remote side effect. | Synthetic identity fields, presented/expected principal, denial reason, and dispatch/side-effect counters. |
| Rotation acknowledgment is lost and the request is retried | Retry resolves to one rotation outcome; overlap ends at the reviewed boundary and retired credentials cannot regain authority. | Credential version timeline, fake-clock readings, retry correlation, and acceptance/denial results. |
| Node is revoked while connected, then receives queued or replayed work | Stop new dispatch, deny stale credentials/replay, reauthorize queued work at execution, and audit the revocation decision. | Revocation time, queue/dispatch trace, authorization version, node response, and audit correlation. |
| Node is partitioned when revocation occurs with work possibly in flight | No new work is sent; on reconnect, queued work is reauthorized and uncertain in-flight effects stay unknown until independently reconciled. | Partition/revoke/reconnect ordering, last verified operation state, external effect check, and final reconciliation. |

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

#### Review-only migration evidence contract for scenarios 11–15

Before migration implementation, approve how writes are handled during transfer, which side owns authority at each stage, what constitutes application-level validation, and who authorizes cutover or rollback. The cases below specify evidence and safety outcomes without choosing a replication or cutover mechanism.

| Fault or stage | Expected test result | Evidence to retain |
|---|---|---|
| Staging before cutover | Compare source and destination file/database inventories, checksums, and application-level invariants; destination is not authoritative before explicit validation. | Source commit/config, synthetic fixture manifest, verification commands/results, and data ownership state. |
| Partition or process restart during transfer | Preserve a durable last verified stage; retry/recovery is attributable to the same migration and cannot trigger duplicate cutover or source retirement. | Fault point, checkpoint lineage, restart order, resumed/aborted decision, and source/destination authority. |
| Route/DNS cutover with stale resolvers or partial health | Retain a verified recovery path; record which destination answered and passed application checks. Do not report complete cutover while supported resolver paths still target an unvalidated endpoint. | TTL/propagation fixture, resolver observations, health evidence, and cutover approval/audit event. |
| Rollback after writes occur on either side | Detect divergence and preserve both sides for explicit reconciliation; never silently discard an acknowledged write or claim rollback restored consistency without verification. | Write sequence, authority/fencing evidence, divergence report, reconciliation decision, and post-recovery invariants. |
| Credential rotation or abort cleanup | Deny revoked credentials at every reachable stage and verify transfer secrets, temporary archives, and partial destination material follow the reviewed cleanup policy. | Credential versions/revocation time, synthetic secret markers, cleanup inventory, and denied replay attempts. |

### Plugins, marketplace, and ecosystem

16. **Manifest and signature policy:** reject unsigned, untrusted, mutable/unpinned, expired, wrong-target, downgraded, or incompatible plugin artifacts before installation. Verify dependency and licence metadata is present and reviewable.
17. **Capability sandbox:** grant a plugin only declared scoped capabilities; prove denied node, tenant, file, secret, network, and API access cannot be reached through direct or confused-deputy paths. The offline candidate model checks exact plugin/install/tenant/resource/capability matching only; it is not sandbox evidence.
18. **Disable and uninstall:** disable/revoke capabilities immediately, preserve user-owned data with an explicit recovery path, and prove panel boot and unrelated plugins continue. Uninstall must not delete unowned data. The offline prototype tests only that an in-memory disabled grant denies repeat requests; persistence, cache invalidation, data preservation, and boot behavior remain untested.
19. **Update and rollback:** stage a signed update, validate compatibility, inject failure, and restore the prior known-good version without restoring revoked permissions or secrets.
20. **Registry outage/compromise:** test unavailable registry, stale metadata, key revocation, and an artifact whose digest differs from reviewed metadata. Fail closed and preserve audit evidence.

#### Review-only supply-chain test details for scenarios 16, 19, and 20

These checks define observable outcomes only. The owner/security review must still choose and approve the metadata format, signer trust model, key custody, revocation source, offline policy, and rollback authority before implementation.

| Fixture or fault | Expected test result | Evidence to retain |
|---|---|---|
| Artifact bytes differ from the reviewed immutable digest, or a reference resolves to different bytes on a second fetch | Reject before install/update side effects; keep the active known-good version unchanged. | Reviewed manifest/reference, expected and observed digests, resolver response, and proof no activation occurred. |
| Unknown, expired, not-yet-valid, wrong-purpose, or revoked signer; revoked signer data arrives after cached metadata | Deny new install/update and prevent revoked trust from being restored by stale cache or reconnect order. | Trust-state fixture, fake clock/cache sequence, decision reason, and redacted audit event. |
| Dependency is missing, mutable/unpinned, incompatible, cyclic, or lacks required licence/provenance metadata | Reject the complete staged graph before activation; do not leave partial dependencies enabled. | Full synthetic dependency graph, per-artifact identity, validation report, and post-failure active-version inventory. |
| Registry is unavailable or returns stale/conflicting metadata | Apply only an explicitly reviewed offline policy; absent valid cached authorization, deny install/update and preserve the active version. | Cache age, last verified metadata, network fault timeline, policy version, and observed decision. |
| Update fails after staging or activation begins | Recover the previous known-good version only if its artifact remains trusted; preserve revocations and narrowed permissions, then verify panel boot and unrelated plugins. | Failure injection point, before/after digests and grants, recovery steps, and boot/health results. |

### Usage, billing integrations, and HA claims

21. **Usage accounting:** reconcile synthetic usage across node restart, delayed/offline reports, duplicate events, clock skew, and tenant transfer. Keep source measurements and derived totals attributable; do not emit billing actions from unverified usage.
22. **External billing/provisioning:** use a fake provider and signed webhook fixtures; test replay, duplicate delivery, timeout, reordered events, and idempotent reconciliation. Billing failure must not silently grant privileges or destroy hosted data.
23. **Mirrored services:** for each proposed DNS/mail/database topology, declare consistency and failure semantics, then test split brain, stale replica, quorum loss, failover, and recovery. Document when an operation is unavailable or read-only.
24. **HA boundary:** publish no HA claim unless workload-specific failover and recovery tests establish explicit topology, failure domain, measured RPO/RTO, and known limitations. Management across several nodes alone is not HA evidence.

#### Review-only usage accounting evidence for scenarios 21–22

The implementation must first approve its usage source, event identity, time attribution, correction policy, tenant-transfer boundary, and period-finalization rules. The following fixtures test conservation and auditability without requiring a particular event table, counter format, or billing provider.

| Fixture or fault | Expected test result | Evidence to retain |
|---|---|---|
| Same usage event is delivered repeatedly with identical content | Apply its contribution once; retries remain attributable and idempotent. | Source identity, payload digest, accepted count before/after, and replay decision. |
| Same event identity arrives with conflicting quantity or tenant/resource binding | Quarantine the conflict for explicit reconciliation; do not silently replace the first record or sum both. | Both payloads, conflict reason, unchanged billable total, and audit lineage. |
| Delayed/offline usage arrives after later reports or after node restart | Reconcile according to reviewed source-time and restart-generation rules; keep receipt time separate from usage attribution and avoid double counting. | Source and receipt timestamps, restart boundary, pre/post totals, and reconciliation decision. |
| Node clock is skewed or tenant ownership changes near a reporting boundary | Apply the reviewed attribution/transfer policy deterministically; record any adjustment and source evidence. | Fake-clock offset, ownership timeline, boundary rule, and reproducible calculation. |
| Report is unverified, incomplete, or still inside a correction window | Do not create an external billing/provisioning action until the reviewed finalization rule is met; provider retries remain idempotent. | Verification status, period state, billing request key, fake-provider interactions, and final reconciliation. |

#### Review-only evidence contract for scenarios 23–24

For each service and supported topology, the test record should name the exact implementation/configuration, node and witness roles, independent failure domains, authoritative write path, consistency guarantee, and supported read/write behavior. Before fault injection, record which observations count as an acknowledged durable write and how service health and data correctness will be verified after recovery. These are evidence fields, not a prescribed topology or replication algorithm.

| Fault or claim | Required observation and evidence | Claim limit |
|---|---|---|
| Control-plane or inter-node partition | Record reachability in each direction, authoritative role before/during/after partition, accepted/rejected writes, stale reads, and the evidence used to prevent or detect competing writers. Rejoin must reconcile or quarantine divergence explicitly. | A simulated partition validates only the tested harness/configuration; it does not prove every network or host failure mode. |
| Stale replica, quorum loss, or witness loss | Record which reads/writes remain available, what authority is retained, whether promotion is denied, and how stale data is detected before resuming writes. | Do not infer safe promotion or consistency from node health alone. |
| Failover and recovery | Record fault injection time, detection/decision/recovery milestones, service-level verification, acknowledged writes compared with recovered data, and any manual action or data repair. Repeat across each claimed failure domain and supported topology. | Report observations and sample count; one successful drill does not establish a general availability percentage. |
| RPO/RTO statement | Derive RPO from the observed gap between acknowledged durable writes and verified recovered state; derive RTO from injected failure to independently verified service recovery. Include workload, topology, repetitions, percentile or range, and exclusions. | Publish only measured scope and conditions; label targets separately from observed results. |
| Management availability vs service HA | Test control-plane loss separately from workload service behavior, including operation queueing, authorization expiry/revocation, and reconciliation on reconnect. | Multi-node inventory/orchestration alone supports only a management capability claim, not workload HA. |

## Evidence and completion record

Each accepted test should retain: scenario ID, source commit, test harness/environment, synthetic fixture description, expected and observed result, logs with secrets removed, cleanup result, and reviewer. Failures and retries remain in the record. A P5 release decision additionally requires the plan's partition/offline/revocation, wrong-node/tenant, migration consistency/rollback, plugin disable/recovery, supply-chain, and HA-boundary criteria to pass on the reviewed implementation. No completion claim is made by this specification.

## Source references

- [Blueprint](https://github.com/parmtech/Tong-Panel/blob/main/docs/BLUEPRINT.md) — staged scale and trust-boundary intent.
- [Roadmap](https://github.com/parmtech/Tong-Panel/blob/main/docs/ROADMAP.md) and [phase index](https://github.com/parmtech/Tong-Panel/blob/main/docs/PHASE_INDEX.md) — phase dependencies and gates.
- [Canonical P5 plan](https://github.com/parmtech/Tong-Panel/blob/main/docs/phases/P5-MULTISERVER-ECOSYSTEM.md) — scope and acceptance.
- [P4 plan at the preserved checkpoint](https://github.com/parmtech/Tong-Panel/blob/docs/p4-hosting-parity-design/docs/phases/P4-HOSTING-PARITY.md) — P4 dependency and acceptance.
- [P1 exit report on the active draft stack](https://github.com/parmtech/Tong-Panel/blob/feature/p1-file-quota-request-context/docs/P1_EXIT_GATE_REPORT_2026-10-27.md), [PR #143](https://github.com/parmtech/Tong-Panel/pull/143), [PR #149](https://github.com/parmtech/Tong-Panel/pull/149), its [first clean-VM run #42785167173](https://github.com/parmtech/Tong-Panel/actions/runs/42785167173), and the earlier [PR #143 clean-VM run #42781985275](https://github.com/parmtech/Tong-Panel/actions/runs/42781985275) — PR #143's unsigned preview passed both guests; PR #149's first run failed and the exact-head rerun remains in progress at this assessment. P1 remains incomplete per the exit report.
- [Draft PR #146](https://github.com/parmtech/Tong-Panel/pull/146) and the [latest P4 continuation at commit `34d0076bf89502d965878bb3c6b2c4e78a63b71f`](https://github.com/parmtech/Tong-Panel/blob/34d0076bf89502d965878bb3c6b2c4e78a63b71f/docs/phases/P4-DESIGN-ACCEPTANCE.md) — the P4 assessment remains blocked/design-only. The original checkpoint above remains preserved.
- [P3 entry/test plan at commit `0931292d30fcc68210eb9f5383dd1ccace4ac80a`](https://github.com/parmtech/Tong-Panel/blob/0931292d30fcc68210eb9f5383dd1ccace4ac80a/docs/P3_ENTRY_AND_ACCEPTANCE_TEST_PLAN.md) on `docs/p3-prerequisite-evidence` — P3 entry remains unproven; this is a design artifact, not P3 implementation.
- [P2 draft PR #89](https://github.com/parmtech/Tong-Panel/pull/89), [P2 draft PR #91](https://github.com/parmtech/Tong-Panel/pull/91), [guest run #42631444449](https://github.com/parmtech/Tong-Panel/actions/runs/42631444449), and [guest run #42633862164](https://github.com/parmtech/Tong-Panel/actions/runs/42633862164) — narrow unsigned preview evidence; exclusions are recorded above.
- [`UPDATES.md` on GitHub `main`](https://github.com/parmtech/Tong-Panel/blob/main/UPDATES.md) — committed journal baseline; active draft-branch outcomes must be rechecked before gate decisions.
