> **Preservation draft:** historical working evidence; factual statements and links require review. No approval or phase completion is implied. Machine-local paths and private coordination details have been omitted.

# P0-06 Independent Security Review Record — Blank Template

**Status:** BLANK TEMPLATE — no review or approval is recorded here.

This form is for the authorized maintainer and independent security reviewer to complete. Preparation of this template is not a threat review, security approval, or owner decision. It authorizes no merge, deployment, host mutation, production use, or Phase 2 entry.

## Exact review inputs and scope

- Repository and reviewed source snapshot:
- Canonical P0 planning file revision: `docs/phases/P0-PLANNING.md` — expected baseline `0aad68e81138a0f226e83e401f90bae4d0b6bddc`; reviewer to verify:
- Canonical threat model revision: `docs/THREAT_MODEL.md` — expected baseline `0aad68e81138a0f226e83e401f90bae4d0b6bddc`; reviewer to verify:
- Canonical security requirements revision: `docs/SECURITY.md` — expected baseline `0aad68e81138a0f226e83e401f90bae4d0b6bddc`; reviewer to verify exact revision and cross-check relevant requirements against threats, controls, and contracts (or record a scoped N/A rationale):
- Canonical test strategy revision: `docs/TEST_STRATEGY.md` — expected baseline `0aad68e81138a0f226e83e401f90bae4d0b6bddc`; reviewer to verify exact revision and cross-check relevant acceptance/test requirements against the threat contracts (or record a scoped N/A rationale):
- P2 threat-to-test packet revision: [`docs/p0-review/p2-threat-test-readiness.md` at `317c72058b2609a4127f253e7e2a0f98d0688cd8`](https://github.com/parmtech/Tong-Panel/blob/317c72058b2609a4127f253e7e2a0f98d0688cd8/docs/p0-review/p2-threat-test-readiness.md), blob `e98df44416185538a40705b5a93b6704c040d70f` — retrieved read-only at this exact ref; reviewer to verify its claims against underlying source revisions:
- Related P2 acceptance-plan revision/path, if included:
- Issue #2 security-design / installer-recovery URL: https://github.com/parmtech/Tong-Panel/issues/2
- Issue #2 exact body revision evidence: read-only GitHub fetch at `2026-11-12 00:52 UTC`; title “P0-02: Build detailed security threat model and validation strategy”; state `open`; `updated_at` `2026-10-27T02:11:43Z`; body SHA-256 `be131c8a2be8427fca07ce72e8495863d0b1620007608a161ff78f5c625e19ae` (UTF-8 body). Exact Acceptance text: “Reviewed security design linked to acceptance tests and P1 work items. Update UPDATES.md; no untested security claims.” Source: https://github.com/parmtech/Tong-Panel/issues/2. Snapshot is source evidence only; verify at review/disposition time.
- Issue #2 acceptance statement and required `UPDATES.md` findings linkage reviewed at exact body revision; keep distinct from installer/recovery Deliverable disposition:
- Exact `UPDATES.md` security-design findings path/anchor and revision, linked issue #2 acceptance result/disposition, and evidence reference (keep separate from P0-02 installer findings):
- Issue #153 URL: https://github.com/parmtech/Tong-Panel/issues/153
- Issue #153 relationship to this review and body snapshot: read-only fetch at `2026-11-12 00:53 UTC`; title “P2 acceptance: specify database provisioning and credential rotation lifecycle”; state `open`; `updated_at` `2026-11-10T09:05:32Z`; body SHA-256 `13632facc364188e1e4444a09839cc19f874588e1b95c9380f15d741eab0a3bf` (2,453 UTF-8 bytes); source https://github.com/parmtech/Tong-Panel/issues/153. Body scopes issue #153 as a Phase 2 design/test specification and states it implies no production credential path, host-facing implementation, or gate approval. Reverify exact revision when reviewing.
- Related ADR IDs and exact revisions reviewed:
- P0-08 threat-contract routing crosswalk exact record revision/hash; for each canonical T01–T15, compare its threat-model assignment with every P0-08 boundary-slice row, verify the covered and uncovered/downstream portions and phase-entry gates, identify package/issue routes or explicit unmapped/unassigned status, record mismatches and unknowns, and distinguish proposed from owner-confirmed routing:
- Related P1 code/CI commit(s), if in scope:
- Supported OS / architecture / runtime / resource profile in scope:
- Interfaces, trust boundaries, and phase scope reviewed:
- Data-flow diagram(s): exact path/revision/hash and reviewer-verified coverage of browser, API, jobs, root agent, applications, backups, and datastore; list any omitted flows/domains and their disposition:
- Explicit exclusions and their later gate/trigger:
- Evidence artifacts and locations:

## Reviewer authority and overall disposition

- Maintainer / decision owner name and role:
- Independent security reviewer name, role, and review authority:
- Reviewer qualifications / experience relevant to this threat and privileged-boundary scope; competence basis and any limits/referral:
- Reviewer independence / conflict statement:
- Review date and time (UTC):
- Review method and evidence examined:
- Canonical threat-model owner disposition for the exact stated scope: [ ] Approve  [ ] Approve with conditions  [ ] Needs revision  [ ] Reject  [ ] Defer
- Threat-model decision owner name/role, authority, rationale, date, and evidence reference:
- Independent security reviewer disposition for the exact stated scope: [ ] Accept  [ ] Accept with findings/conditions  [ ] Needs revision  [ ] Reject  [ ] Defer
- Independent reviewer identity/qualification, authority, date, and evidence reference:
- Exact scope covered by this disposition:
- Conditions, required follow-up issues, or unresolved questions:
- Evidence link(s) for the recorded decision:

## Evidence-only annex: reported component pointers for reviewer verification

Source: P2 threat-to-test draft `docs/p0-review/p2-threat-test-readiness.md` at exact head `317c72058b2609a4127f253e7e2a0f98d0688cd8`, table rows 22–36 and proposed contract groups 42–51. These are reported/component/proposal pointers only. Verify the exact underlying artifact and its scope before use. They do not fill or satisfy the per-threat review matrix below and do not establish accepted tests, reviewed mitigations, risk decisions, ownership, or security approval.

| TID | Existing pointer as reported in P2 | Remaining contract gap; proposed contract reference |
|---|---|---|
| T01 | Internal session test commit `6ae402888141e76aa83f38af5f96805598da5966`; login prototype `c2cfadc0d6e85be13b5dfab7083a77405505caf6`; several Actions runs listed in P2 T01 row. Reported component evidence only. | No hardened public login/cookie/logout/recovery integration; P2 proposed contract L42. |
| T02 | P1-reported dirty/uncommitted cross-tenant test; PRs #62/#64/#79/#89/#40. Reported/partial component pointers. | No end-to-end API→job→agent effect-time denial matrix; P2 proposed contract L43. |
| T03 | At snapshot `cabef25caa779db1650353fd67e8ff9c0fa70dda`, [`internal/agent/agent_linux_test.go`](https://github.com/parmtech/Tong-Panel/blob/cabef25caa779db1650353fd67e8ff9c0fa70dda/internal/agent/agent_linux_test.go) (blob `21567b0029346db2a1f8de119a0f8b8ec32f1e4b`) contains ping-only decoder, duplicate-operation, peer-UID, and socket-parent tests; [`internal/agent/agent_linux.go`](https://github.com/parmtech/Tong-Panel/blob/cabef25caa779db1650353fd67e8ff9c0fa70dda/internal/agent/agent_linux.go) (blob `7b5f50d871e5eddce26a10587768c6ee84074d33`) is the corresponding code snapshot. Decoder changes were introduced in `1ded105f680c10d6c5243201045350b3222800e5`. These are reported code/unit pointers; P2 provides no test-run ID. | No accepted privileged boundary, effect/idempotency policy, or integrated adversarial evidence; verify executed run and exact environment. P2 proposed contract L44. |
| T04 | PR #40 traversal/bounded extraction and #49 concurrent create-only extraction tests. Reported filesystem component evidence. | No full API/daemon, per-tenant UID/quota, or race-adversarial coverage; P2 proposed contract L45. |
| T05 | No hostile application execution test identified in the reviewed P0 evidence. | Canonical `THREAT_MODEL.md` assigns T05 (malicious Git build/runtime escape) to P3. P2 groups T05 with T07 in proposed contract L46 and asks to defer later execution/feature acceptance until P3/P5 scope is explicit; preserve that as a proposal, not a phase reassignment. P0 still needs a reviewer-dispositioned planned test contract, issue/workstream mapping, and explicit P1/P2 exclusion plus P3 entry gate. Do not treat the P2 deferral proposal as an accepted contract or as an exception to P0 threat-to-test planning. |
| T06 | PR #95 facts-only support-bundle redaction; #81 backup envelope with caller-supplied external KEK. Reported component evidence only. | No accepted full secret lifecycle contract; P2 proposed contract L47. |
| T07 | No signed catalogue/update acceptance identified. | Canonical `THREAT_MODEL.md` assigns T07 (malicious marketplace plugin/update) to P3/P5. P2 proposed contract L46 likewise asks to defer later execution/feature acceptance until P3/P5 scope is explicit. P0 still needs a reviewer-dispositioned planned test contract, issue/workstream mapping, and explicit exclusion plus P3/P5 entry gate. Do not treat the P2 deferral proposal as an accepted contract or as an exception to P0 threat-to-test planning. |
| T08 | [`internal/api/session_cookie_test.go`](https://github.com/parmtech/Tong-Panel/blob/6c983a22de1e5b1ea0de133ea28c6d1daadc490c/internal/api/session_cookie_test.go) at snapshot `6c983a22de1e5b1ea0de133ea28c6d1daadc490c` (blob `0815c7002e8ae0e4325da50eb47af93369ba38b8`), a formatting-only commit. It contains cookie, CSRF/origin, TLS ambiguity, duplicate-origin, role-change, session-revocation, and concurrency tests. This is a code-state pointer, not test-introduction or execution provenance; P2 supplies no run ID. | No external review or integrated public-route/proxy policy acceptance; verify executed run and exact environment. Proposed with T01 at P2 L42. |
| T09 | DNS challenge #66 and resolver test #85. Reported component proofs. | P2 proposed service-failure contract L48. |
| T10 | PRs #81/#82/#84/#91/#79; issue #105 reports unresolved stale-session behavior in a SQLite snapshot. Reported component restore evidence. | No complete cross-product restore/authorization acceptance; P2 proposed contract L49. |
| T11 | PR #74 unsigned disposable-archive build/extract/smoke; #86 package preflight; image checksums reported. Reported component evidence. | No signed provenance; P2 proposed contract L50. |
| T12 | No qualifying host-mutation/rollback evidence identified. | P2 proposed disposable-VM safety contract L50. |
| T13 | No mail acceptance evidence identified; mail is optional P4 scope. | P2 proposed service-failure contract with T09 at L48. |
| T14 | No evidence beyond a ping-only local decoder/agent prototype; remote-agent system is future P5 scope. | P2 proposed future-agent contract L51. |
| T15 | P1-reported SQLite job tests #127/#128; restore components also referenced for T10. Reported component evidence only. | No queue→privileged effect→audit integration for full-disk, power-loss/reboot, or network failure; P2 proposed contract L49. Process termination/job-worker restart evidence alone does not establish hard-power interruption or reboot recovery. |

Across all rows, P2 contains no filled accepted test contract, threat-specific named accountable owner, reviewed mitigation disposition, residual-risk decision, or independent reviewer disposition. Its document-level “P2 chat, temporarily reassigned to P0” is not a per-threat person assignment. Proposed contracts remain for the authorized owner/reviewer to accept, revise, or reject. The per-threat matrix, STRIDE labels, likelihood/impact, mitigation and risk dispositions, owner fields, privileged-agent decision, and P0-06 gate below remain blank for reviewer completion.

### Required per-threat evidence layer and observed-result record

For each threat, distinguish the actual evidence layer and result from the accepted test-contract revision and its expected outcome. A reported component pointer, unit/CI pass, or clean-VM lab pass is not integrated end-to-end acceptance unless the threat-relevant interface and effect path were exercised. Cite the immutable source/head and artifact, record “not run” where applicable, and state the remaining integration gap. Do not infer acceptance from a passing workflow or draft proposal.

| TID | Evidence layer/status (reported component / unit or CI / clean-VM lab / integrated end-to-end / not run; reviewer classifies) | Exact source and revision/head (workflow, run, artifact, or report) | Actual observed result and scope, or “not run” basis | Remaining integration/acceptance gap |
|---|---|---|---|---|
| T01 | | | | |
| T02 | | | | |
| T03 | | | | |
| T04 | | | | |
| T05 | | | | |
| T06 | | | | |
| T07 | | | | |
| T08 | | | | |
| T09 | | | | |
| T10 | | | | |
| T11 | | | | |
| T12 | | | | |
| T13 | | | | |
| T14 | | | | |
| T15 | | | | |

**T15 failure-mode separation:** canonical T15 includes power failure. Record host/process crash, full disk, network failure, power interruption, and reboot as separate cases. For each in-scope case, identify the injection method, durable job/effect/audit state before failure, expected safe state, restart/replay/idempotency behavior, external-effect reconciliation, and exact command/environment/result evidence. A process-kill or worker-restart test is not evidence of power-loss durability. If power interruption/reboot is not tested or is out of scope, record that status and owner rationale explicitly; do not infer a result.

**Additional P1 candidate evidence pointer for reviewer verification:** PR #149 head `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23` on base `482017005f9872c1fe7f2969ebfb477ea7de30dd`; PR-head-associated clean-VM run `42790473632`, [Debian 12 job](https://github.com/parmtech/Tong-Panel/actions/runs/42790473632/job/129742295031) and [Ubuntu 24.04 job](https://github.com/parmtech/Tong-Panel/actions/runs/42790473632/job/129742295654). The guest jobs checked out merge ref `da44246220147560b6d7c1510f32cc1edcae3af7`, so they are not exact-head-only execution evidence. The workflow returned no downloadable VM artifacts, but job logs are accessible. Reviewer must classify only threat-relevant checks and record limitations; this pointer is not a security review or integrated acceptance.

## Per-threat review matrix

Complete one row for every canonical threat. Use additional linked detail where a cell cannot hold the rationale. All labels, scores, dispositions, owners, and risk acceptances are for the reviewer to determine. For routing, cite the exact P0-08 crosswalk revision and every applicable slice row and route status for that TID; any `owner-confirmed` or `deferred` slice route must cite its exact confirming owner decision and authority reference. A proposed route is not a mitigation disposition or phase assignment.

| TID | Canonical threat text / revision | Related ADR / P0-05 exact revision and status/dependency (or N/A rationale) | Assets affected | Data flow and trust boundary | STRIDE labels (reviewer confirms/revises) | Likelihood scale, value, rationale | Impact scale, value, rationale | Mitigation disposition and control/revision | Linked issue + accepted test-contract revision | Test command/script + immutable code SHA + OS/arch/profile | Expected positive/negative result and evidence artifact | Named accountable owner (or explicitly unassigned); workstream | Residual risk and rationale | Risk acceptor/authority + expiry/revisit trigger | Reviewer disposition/date |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| T01 | | | | | | | | | | | | | | | |
| T02 | | | | | | | | | | | | | | | |
| T03 | | | | | | | | | | | | | | | |
| T04 | | | | | | | | | | | | | | | |
| T05 | | | | | | | | | | | | | | | |
| T06 | | | | | | | | | | | | | | | |
| T07 | | | | | | | | | | | | | | | |
| T08 | | | | | | | | | | | | | | | |
| T09 | | | | | | | | | | | | | | | |
| T10 | | | | | | | | | | | | | | | |
| T11 | | | | | | | | | | | | | | | |
| T12 | | | | | | | | | | | | | | | |
| T13 | | | | | | | | | | | | | | | |
| T14 | | | | | | | | | | | | | | | |
| T15 | | | | | | | | | | | | | | | |

### Threat-specific mitigation disposition and deferral record

Record the authorized owner decision separately from the independent review finding. Use an explicit status for every threat: `accepted`, `needs revision`, `deferred`, `rejected`, or `not reviewed`. A deferral must name its approving authority, excluded scope, required completion gate, and revisit trigger; it is not an accepted mitigation or test.

| TID | Exact proposed control / mitigation revision | Owner disposition (accepted / needs revision / deferred / rejected / not reviewed) | Decision owner, authority, rationale, and date (UTC) | Independent reviewer finding/date and evidence reference | If deferred: explicit exclusion, completion gate, owner, and revisit trigger | Linked issue/PR or closure record |
|---|---|---|---|---|---|---|
| T01 | | | | | | |
| T02 | | | | | | |
| T03 | | | | | | |
| T04 | | | | | | |
| T05 | | | | | | |
| T06 | | | | | | |
| T07 | | | | | | |
| T08 | | | | | | |
| T09 | | | | | | |
| T10 | | | | | | |
| T11 | | | | | | |
| T12 | | | | | | |
| T13 | | | | | | |
| T14 | | | | | | |
| T15 | | | | | | |

### Threat-model and contract closure / change-control record

| Field | Record |
|---|---|
| Exact threat-model revision/hash and scope proposed for closure | _Reviewer to determine_ |
| Exact accepted mitigation and test-contract revisions by TID | _Reviewer to determine_ |
| Linked change issue/PR, source/base/head, and changed TIDs/flows/controls | _Reviewer to determine_ |
| Closure evidence: exact commands, profiles, results, artifacts/log hashes, and reviewer findings | _Reviewer to determine_ |
| Owner decision, authority, rationale, and date; independent review disposition/date | _Reviewer to determine_ |
| Change/invalidation triggers requiring re-review (scope, asset/flow, control, contract, code, profile, or evidence change) | _Reviewer to determine_ |
| Required re-review roles, hold/reopen action, and closure/reopen record reference | _Reviewer to determine_ |

## Installer / OS / firewall recovery matrix and failure-injection QA plan

Issue #2 requires a recovery matrix and failure-injection plan. Complete a row for each selected OS/architecture/image profile and each in-scope lifecycle/failure point, or record the exact reason it is excluded/deferred and the gate where it must be completed. A scenario name or generic threat reference alone is not a recovery result. Define the expected before/after state and evidence before any test; do not imply that a proposed scenario has been executed.

| OS / architecture / exact image or profile | Lifecycle stage and failure point | Fault or interruption injected | Expected pre-state and safe failure state | Expected rollback/recovery and post-state | Preservation checks (SSH/access, firewall, services, data) | Command/environment and exact result/artifact reference | Owner / independent reviewer disposition (tested, not tested, deferred, excluded) |
|---|---|---|---|---|---|---|---|
| | Fresh install / preflight / dependency check | | | | | | |
| | Port conflict / firewall rule apply or restore | | | | | | |
| | SSH/access preservation during install, upgrade, or rollback | | | | | | |
| | Interruption at each privileged/mutating checkpoint | | | | | | |
| | Post-install health-check failure / service-start failure | | | | | | |
| | Upgrade, failed migration, rollback, or recovery | | | | | | |
| | Other profile-specific failure point | | | | | | |

- QA plan revision and linked scenario inventory:
- Required profiles and evidence sources; untested/deferred/excluded profiles and rationale:
- Failure-injection boundary selection and expected-state oracle owner/reviewer:
- Known destructive/out-of-scope tests and required disposable-environment controls:
- QA disposition and remaining conditions: [ ] Reviewed  [ ] Needs revision  [ ] Not reviewed
- Reviewer authority, scope, date, and evidence reference:

## Explicit privileged-agent boundary decision

- Exact agent/API protocol, schema, and revision reviewed:
- OS peer-authentication mechanism and supported profile:
- Socket path, ownership, permissions, and service identities:
- Privilege separation and capability/resource limits:
- Allowed typed operations and explicitly denied operations:
- Tenant, principal, job, resource, expiry, and replay/idempotency bindings:
- Filesystem/path confinement and race protections:
- External effects, staging, rollback/recovery, and idempotency rules:
- Audit linkage, secret redaction, and failure behavior:
- Linked negative/positive test contract, exact code SHA, command, and environment:
- Residual risks and exclusions:
- Maintainer boundary disposition: [ ] Approve only for named scope  [ ] Needs revision  [ ] Reject  [ ] Defer
- Maintainer name/role, date, and decision evidence:
- Independent security reviewer disposition, name/role, date, and evidence:

## P0-06 gate result

- P0-06 disposition: [ ] Met  [ ] Not met  [ ] Deferred
- P0-06 gate decision owner/authority, rationale, date, and evidence reference:
- Evidence basis for each P0-06 criterion:
- Unresolved threat IDs / decision dependencies:
- Remaining conditions and linked follow-up issues:
- Separate P0-05/P0-07/P0-08/P0-09/P0-10 decisions still required:

**Gate boundary:** Completing this form does not itself authorize merge, deployment, production use, host mutation, or P2 entry. Those actions require their own recorded approvals.
