> **Preservation draft:** historical working evidence; factual statements and links require review. No approval or phase completion is implied. Machine-local paths and private coordination details have been omitted.

# P0-08 Backlog Ownership Confirmation Matrix — Blank Template

**Status:** BLANK OWNER-CONFIRMATION TEMPLATE. This is a crosswalk for review, not an approved backlog, assignment, test plan, phase decision, or authorization.

**Sources:** P3 draft PR #160 at head `4a71768256a38d6c052461b6cd8871298e6f3435` (base `main@0aad68e81138a0f226e83e401f90bae4d0b6bddc`) and fresh issue bodies #3 and #5–#12 read 2026-11-11. The package mapping below is the P3 draft’s coverage assessment, not accepted ownership or acceptance.

> Issue title labels P1-01…P1-08 are not canonical package IDs. Issue authorship, labels, proposed workstreams, generic mentions of “owner” or “security review,” coordinator events, or acknowledgments do not establish assignment or write authority.

## Known issue mapping and numbering

- **Issue #3** is the P0 backlog anchor; it asks for approved target/resource assumptions, package/service boundaries, separate tickets with test plans, security/licence checklist, owner/workstream, dependency graph, and a P1 foundation/P2 hosting split. It is not one of canonical P1 packages 01–10.
- **#5 / title P1-01** maps to canonical package 02 only for loopback liveness; #12 also supports API contract CI.
- **#6 / title P1-02** maps to canonical package **03 (Authentication)**, not canonical package 02.
- **#7 / title P1-03** maps to canonical package 05 (Durable jobs).
- **#8 / title P1-04** maps to canonical package 06 (Local agent).
- **#9 / title P1-05** maps to canonical package 08 (UI); it does not supply package 09 diagnostics/support acceptance.
- **#10 / title P1-06** combines canonical packages **04 (Identity/authorization) and 07 (Metadata/migrations)**.
- **#11 / title P1-07** covers canonical package 10 (Installer/package lab) and only the doctor/preflight slice of package 09.
- **#12 / title P1-08** spans canonical package 01 (repository/workflows and CI) and provides supporting CI for packages 02 and 10.
- No issue among #5–#12 is a complete one-to-one map to the ten canonical packages. #3 remains the backlog/scope anchor. This table maps proposed P1 package ownership only; P0 issue anchors #1 (ADR outcomes), #2 (threat model), and #4 (UX flow) are outside this package matrix and require their own owner/reviewer records. #4 is the UX anchor associated with P0-02.

## Evidence-only issue and acceptance crosswalk — 2026-11-11

PR #160 remains a P3 working draft at the cited head. A metadata refresh at 2026-11-11 05:39 UTC confirmed that issues #1–#12 remain open, with no assignees and zero comments; their `updated_at` timestamps remain in October 2026, unchanged from the 03:26 UTC snapshot. No issue body or comment records a P0/P1 owner decision. Existing issue acceptance text is narrower than the P3 proposal in several places; the proposed expansions below are not approved ticket changes.

| Issue | Existing issue acceptance / dependency evidence | P3 proposal delta or unresolved mapping |
|---|---|---|
| [#5](https://github.com/parmtech/Tong-Panel/issues/5) — Go API liveness | Go tests/vet and non-leaking errors. | P3 proposes broader bounded OpenAPI/error surface and contract/security-boundary evidence; current issue covers liveness only. |
| [#6](https://github.com/parmtech/Tong-Panel/issues/6) — auth/recovery/MFA | Security design/review and executable negative tests; implementation follows design/review. | P3 proposes bootstrap, login/session, MFA/recovery, key custody, rate/lockout, and endpoint-wide authorization evidence. |
| [#7](https://github.com/parmtech/Tong-Panel/issues/7) — durable jobs/idempotency | Avoid duplicate host mutations on retry and worker bypass of tenant permissions; follows ADR-003 approval. | P3 proposes effect-time authorization, crash/restart, audit, restore, retry, and full-disk evidence. |
| [#8](https://github.com/parmtech/Tong-Panel/issues/8) — privileged Linux agent | Fuzz/abuse tests and independent review; follows ADR-007 and threat-model review. | P3 proposes principal/tenant/job/resource/replay/audit bindings and explicit shell/path denials. |
| [#9](https://github.com/parmtech/Tong-Panel/issues/9) — home grid/UI kit | Screenshot/automated UI evidence, accessibility, and first-time-task acceptance. | P3 proposes complete enabled-journey and human accessibility evidence; current issue does not define the whole enabled-journey matrix. |
| [#10](https://github.com/parmtech/Tong-Panel/issues/10) — tenant/resource model/RBAC | IDOR/privilege-escalation tests and reversible migration; references ADR-003. | P3 proposes restore/session, full-disk, concurrency, and API→job→agent evidence. This issue also serves package 07, so split vs explicit sub-acceptance remains unresolved. |
| [#11](https://github.com/parmtech/Tong-Panel/issues/11) — installer preflight/distro lab | Reject dangerous/unsupported hosts without changes and run in disposable VMs; references ADR-005/008. | P3 proposes signed-artifact, install, interruption, rollback, and recovery evidence. It covers only a subset of package 09 diagnostics/support. |
| [#12](https://github.com/parmtech/Tong-Panel/issues/12) — CI/supply chain/release | CI evidence with passing checks and known limitations. | P3 proposes reproducibility, dependency/security, API-compatibility, and provenance baselines; issue also overlaps packages 01, 02 and 10. |

Issue #3 remains the P0 scope/backlog anchor; it is not a person-level assignment. Issue bodies and P3 role labels do not name or confirm accountable package owners/reviewers. The existing issue records do not establish P3's full dependency graph as accepted linked dependencies. Package 09's missing support-bundle/redaction/evidence/recovery coverage, package 04/07 overlap at #10, any required issue split, P3 acceptance expansions, and the dependency order remain owner/reviewer decisions.

### Threat-contract routing crosswalk — blank owner review input

For every P2 threat/contract, record its proposed route using canonical package IDs (not P3 stage ordinals), or explicitly mark it downstream/unassigned with the applicable phase-entry gate. Use one row per distinct threat slice/boundary portion, and repeat the canonical TID as needed; do not let a route for one portion imply coverage or assignment of the whole canonical threat. For each row, state the portion covered, any uncovered/downstream portion and its entry gate, and the exact confirming owner decision/authority reference for confirmed/deferred status. P2/P3 proposal text alone cannot establish an owner-confirmed or deferred route. Link shared evidence once and cross-reference it from each relevant threat; do not imply that a route, issue, test, owner, or acceptance is confirmed by this draft.

| Canonical TID | Slice ID / threat-boundary portion | P2 proposed contract ID and exact revision/section | Portion covered by this row | Proposed canonical package ID(s) and GitHub issue(s), or explicitly unmapped | Uncovered/downstream portion and phase-entry gate | Shared test/evidence reference(s) or separate contract | Route status (proposed / owner-confirmed / deferred / unmapped / not reviewed) | Exact confirming owner decision and authority reference | Responsible workstream/person or explicitly unassigned |
|---|---|---|---|---|---|---|---|---|---|
| T01 | | | | | | | | | |
| T02 | | | | | | | | | |
| T03 | | | | | | | | | |
| T04 | | | | | | | | | |
| T05 | | | | | | | | | |
| T06 | | | | | | | | | |
| T07 | | | | | | | | | |
| T08 | | | | | | | | | |
| T09 | | | | | | | | | |
| T10 | | | | | | | | | |
| T11 | | | | | | | | | |
| T12 | | | | | | | | | |
| T13 | | | | | | | | | |
| T14 | | | | | | | | | |
| T15 | | | | | | | | | |

## Owner-confirmation matrix

For proposed dependency/critical-path text, retain the source citation and mark it “P3 proposal” until the owner confirms it. In each owner cell record both the named accountable person and their proposed/confirmed workstream role, then record that person's explicit acceptance; a role label or person name alone is not an assignment. Fill assignment and acceptance cells only after the actual accountable person and reviewers confirm. Use exact issue URLs, immutable revisions, test contracts, and evidence links.

| Canonical package ID / scope | Existing issue(s) and coverage | Proposed predecessor / interface (not accepted) | Critical-path position (proposal only) | Named accountable owner (person + workstream/role) | Reviewer, authority, and independence/conflicts where required or adopted | Acceptance / test contract and exact evidence | Owner disposition, exclusions, or child-ticket decision |
|---|---|---|---|---|---|---|---|
| 01 — Repository, workflows, package layout | #3 overlaps scope/layout; #12 covers CI/supply-chain subset | P3 draft proposes approved P0 scope/threat baseline, then repository/CI baseline; interfaces to API/schema/job/agent contracts | P3 proposed order 1–2 | Person + workstream/role to determine | Owner to determine | Owner to determine | Owner to determine |
| 02 — Control-plane API and loopback liveness | #5 covers loopback GET /healthz subset; #12 supports contract CI | P3 draft proposes package 01 and CI baseline first; interface to auth, tenant authorization, jobs and OpenAPI | P3 proposed order 3 | Person + workstream/role to determine | Owner to determine | Owner to determine | Owner to determine |
| 03 — Authentication, session and recovery | #6, titled P1-02; proposed ADR-009 is separate/unapproved | P3 draft proposes an accepted tenant/principal contract. The predecessor is met only by an Accepted exact ADR-009 revision adopting the required auth/session policy. If ADR-009 is outside scope/index, held/deferred, rejected, or needs revision, package 03 stays blocked until an alternative canonical auth/session policy is accepted and recorded in ADR_REGISTER and issue #1. An indexing/scope choice alone does not satisfy the predecessor. Also require reviewed TLS/origin/proxy, secret-custody, MFA/recovery and abuse/lockout decisions. | P3 proposed order 5 | Person + workstream/role to determine | Owner to determine | Owner to determine | Owner to determine |
| 04 — Identity and authorization | #10 combines this with package 07 | P3 draft proposes accepted ADR-003 schema/migration/recovery choice, threat model, and package 01 contract; interface to API, job worker and host-effect authorization | P3 proposed order 4 | Person + workstream/role to determine | Owner to determine | Owner to determine | Owner to determine |
| 05 — Durable jobs and audit | #7 | P3 draft proposes storage semantics from its order 4 (canonical packages 04 + 07) and principal/tenant authorization from orders 4–5 (canonical packages 03 + 04 + 07); interface to agent and external-effect reconciliation | P3 proposed order 6 | Person + workstream/role to determine | Owner to determine | Owner to determine | Owner to determine |
| 06 — Narrow local host-agent boundary | #8 | P3 draft proposes ADR-007/threat mitigations plus identity/job authorization and typed contracts from P3 orders 4–6 (canonical packages 03 + 04 + 05 + 07) | P3 proposed order 7 | Person + workstream/role to determine | Owner to determine | Owner to determine | Owner to determine |
| 07 — Metadata, migrations, snapshot/restore contract | #10 combines this with package 04 | P3 draft proposes accepted ADR-003 and package 01 schema choice; interface to identity, jobs, sessions and recovery | P3 proposed order 4, jointly scoped with 04 in draft | Person + workstream/role to determine | Owner to determine | Owner to determine | Owner to determine |
| 08 — UI foundation | #9 | P3 draft proposes stable reviewed API/auth/job contracts for enabled flows; otherwise controls stay disabled/design-only | P3 proposed order 8, alongside diagnostics | Person + workstream/role to determine | Owner to determine | Owner to determine | Owner to determine |
| 09 — Diagnostics, doctor/status/logs and support evidence | #11 covers doctor/preflight only; #9 is UI, not a diagnostics acceptance contract; no dedicated issue identified in PR #160 | Backlog owner may create a scoped child issue, add explicit package 09 sub-acceptance to #11 or UI-only scope to #9, or record complete nonduplicative coverage across existing issues. Exclude doctor/preflight under #11 and installer/package scope mapped partly to #11/#12; GitHub issue #10 maps canonical packages 04/07, not package 10. | P3 proposed order 8, alongside UI; coverage route and any child ticket are owner decisions | Person + workstream/role to determine | Owner to determine | Owner to determine | Owner to determine |
| 10 — Signed experimental package and disposable-VM lab | #11 + #12, partial/complementary | P3 draft proposes ADR-005/008 and scope/matrix decision plus tested contracts for each shipped operation | P3 proposed order 9 | Person + workstream/role to determine | Owner to determine | Owner to determine | Owner to determine |

### Assignment authority, scope, and acknowledgement record

Create one row per proposed package assignment. Keep a row unassigned until the assigning authority, named person, exact scope, and that person's response are recorded. A package/workstream label alone is not a file designation. Record the source of authority and the revision that authorizes the designation; do not infer it from a coordinator relay or acknowledgment.

| Package ID | Assignment state (unassigned / proposed / accepted / declined) | Named assignee and workstream/role | Exact files/paths, deliverables, and exclusions | Designated by (name, role, authority), exact authority source/revision, and date (UTC) | Owner response (accepted / declined / pending), accepted scope/conditions, and date (UTC) | Branch and PR/base/head (or explicitly not created) | Reviewer name, authority, scope/conflicts, response, and date (UTC) |
|---|---|---|---|---|---|---|---|
| 01 | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ |
| 02 | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ |
| 03 | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ |
| 04 | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ |
| 05 | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ |
| 06 | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ |
| 07 | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ |
| 08 | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ |
| 09 | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ |
| 10 | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ | _To determine_ |

If the assignee or designating authority is unknown, set the state to `unassigned` and record the missing identity/authority. Route a designation request only to a person whose authority to designate is documented; if no such authority is identified, keep assignment and shared-file write authority unresolved pending an authorized governance designation. A coordinator event may carry the request or link a designation record; it does not itself grant write authority. A reviewer response records receipt/scope only and does not grant assignment authority or approval.

### Per-package security/licence checklist record required by issue #3

Issue #3 requires each package ticket to carry a security/licence checklist. Complete this companion matrix only after the package owner and authorized reviewers confirm scope; cite the exact checklist/revision, evidence, reviewer disposition, or an explicit N/A rationale. Do not treat the row as approval by itself.

| Canonical package ID | Security/licence checklist revision and evidence links | Required reviewer disposition / authority, or N/A rationale |
|---|---|---|
| 01 | | |
| 02 | | |
| 03 | | |
| 04 | | |
| 05 | | |
| 06 | | |
| 07 | | |
| 08 | | |
| 09 | | |
| 10 | | |

## Cross-cutting confirmation

- **Issue assignment snapshot:** The 2026-11-11 05:39 UTC metadata refresh reconfirmed #1–#12 open with no assignees and zero comments; `updated_at` remains unchanged from 03:26 UTC. This matrix does not assign anyone; recheck before relying on this snapshot for a later decision.
- **Required person-level confirmation:** record the accountable individual, their acceptance of responsibility, named reviewer(s), review scope/authority, and the exact issue or record where assignment is maintained. Record reviewer independence/conflicts, any recusal/reassignment, or an explicit not-applicable rationale where independence is required by a source or adopted by the authorized owner; do not treat independence as a blanket P0-08 requirement.
- **Unknown assignment authority:** keep the row `unassigned` until a person with documented authority explicitly designates an assignment authority. Do not assume that the project owner has that authority without an authority source; do not infer assignment or write authority from a role, issue author, workstream label, coordinator event, or acknowledgment.
- **Acceptance:** do not copy proposed acceptance wording into a ticket as approved. Record test environment, command/script, exact source SHA, expected positive/negative result, artifact location, security/licence review needs, and known exclusions after owner/reviewer confirmation.
- **Dependencies/interfaces:** identify source contract revisions and blocking/predecessor relationships; do not treat P3’s order or a green component check as approval.
- **Disposition:** record whether each package is accepted, needs revision, deferred, split, combined, or excluded from P1, with owner, rationale, evidence and later gate/trigger.
- **ADR and phase boundaries:** this matrix does not accept ADR-001–008, decide ADR-009 inclusion/content, approve a threat model, approve P1 code/CI, pass P0/P1, authorize merge/deployment/production use, or authorize P2 entry.

## Record of confirmation

| Field | Owner/reviewer to complete |
|---|---|
| P0 baseline and matrix revision reviewed | Owner to determine |
| P3 draft PR #160 revision reviewed | Owner to determine |
| P0/backlog decision owner: name, role, authority source/revision, disposition scope (such as package mapping, dependency order, or issue split), and date (UTC) | Owner to determine |
| Reviewer, authority, independence/conflict declaration or rationale where required/adopted, and date (UTC) | Owner to determine |
| Canonical location for confirmed assignments | Owner to determine |
| Follow-up issues and evidence links | Owner to determine |

**No assignments, acceptance contracts, or approvals are recorded by this blank template.**
