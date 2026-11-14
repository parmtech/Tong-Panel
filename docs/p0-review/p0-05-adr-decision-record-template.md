> **Preservation draft:** historical working evidence; factual statements and links require review. No approval or phase completion is implied. Machine-local paths and private coordination details have been omitted.

# P0-05 ADR Disposition Record — Blank Template

**Status:** BLANK TEMPLATE — no ADR outcome or implementation approval is recorded here.

Use this form for an authorized owner/maintainer decision after the required technical, security, or licence review. All names, dates, outcomes, and rationale below are intentionally blank for the owner and reviewers to determine. Do not treat earlier proposals, automated analysis, or conditional recommendations as accepted decisions.

## Review session and authority

- Repository:
- Exact source snapshot / commit:
- ADR register path and exact revision reviewed:
- Related P0 planning revision:
- Related threat-model revision:
- Issue #1 exact body snapshot: read-only GitHub fetch at `2026-11-12 00:51 UTC`; title “P0-01: Review Tong Panel architecture ADRs and upstream license strategy”; state `open`; `updated_at` `2026-10-26T01:36:04Z`; body SHA-256 `fe7a2b1772647029857a1b5855f2e17d56c4b8ceb162b45a89bdc5a75c2142f7` (850 UTF-8 bytes, LF line breaks, no trailing newline). Exact acceptance text: “Each ADR states context, alternatives, security/licence trade-offs, selected proposal and review status; update docs and UPDATES.md. Never mistake this design issue for implementation.” Source: https://github.com/parmtech/Tong-Panel/issues/1. Snapshot is evidence only; reviewer must reverify it when recording any decision.
- Issue #6 exact body snapshot (ADR-009 conditional dependency): read-only GitHub fetch at `2026-11-12 00:51 UTC`; title “P1-02: Define auth, recovery and MFA architecture”; state `open`; `updated_at` `2026-10-27T23:02:01Z`; body SHA-256 `57937bc967d10cbb185bd9c11935e7a95c492477eaf17c6d9a048c1640399a21` (9,145 UTF-8 bytes). Exact acceptance text: “approved security design, executable negative tests; no hardcoded default password.” Source: https://github.com/parmtech/Tong-Panel/issues/6. Current body says ADR-009 is proposed/not owner-security approved; verify exact bytes/revision and recheck at disposition time. Snapshot is not acceptance.
- PR #160 exact proposal revision for package 03 dependencies (if relied on):
- Review date/time (UTC):
- Decision owner name and role:
- Required security reviewer name, role, and authority:
- Required licence/trademark reviewer name, role, and authority (where applicable):
- Additional reviewers and authority:
- Reviewer independence / conflict statement:
- Evidence locations and review method:

The canonical register currently lists `Accepted / Rejected / Needs revision`; issue #1 requests an ADR review status but defines no labels. These current canonical outcomes may be recorded without a vocabulary change. This form also offers `Deferred` as an optional status: the authorized canonical writer needs to add it to `docs/ADR_REGISTER.md` only if the owner chooses a formal Deferred outcome. Until then, leave the ADR outcome blank for a temporary hold and use the linked deferral record below; do not map the hold to another status. Issue #1 remains the discussion/signoff location for each exact disposition. No outcome is selected here.

**Optional register wording if the owner chooses a formal Deferred outcome (proposal only; not a decision):** “For each ADR, record **Accepted / Rejected / Needs revision / Deferred**, decision date, reviewers, evidence, alternatives considered, security impact, migration story, and linked PR/issues. **Deferred** is a distinct temporary hold pending a stated condition and revisit; it is not **Accepted** or **Needs revision**.” Adding this option is not a prerequisite for recording outcomes already listed in the canonical register; no register change is made by this form.

## Required cross-gate evidence references

For each ADR, link the exact relevant revision and evidence/decision section from the [P0-02 installer-spec/recovery record](p0-02-installer-spec-acceptance-template.md), [P0-06 independent security review record](p0-06-independent-security-review-template.md), [P0-07 scope/support record](p0-07-scope-support-decision-template.md), [P0-08 owner/backlog record](p0-08-owner-assignment-template.md), and [P0-09 provenance review record](p0-09-provenance-review-template.md). ADR-005 and ADR-008 must cite the applicable P0-02 installer-spec disposition and, where in scope, the separate issue #2 installer/OS/firewall-recovery plan disposition; otherwise provide a scoped N/A rationale. P0-02 plan review is not executed installation evidence, security-design acceptance, OS support, or overall issue #2 acceptance; execution evidence remains at the applicable implementation/CI gate. For P0-08, cite the threat-contract routing crosswalk revision/hash and each applicable repeated slice-row ID, its route status and phase-entry gate, or a justified N/A; for `owner-confirmed` or `deferred`, cite the exact confirming owner decision and authority reference. Route status is separate from P0-06 mitigation disposition and the ADR outcome; it does not establish an assignment or accepted test contract. If any gate is not applicable, give the rationale; if it is open, deferred, or unreviewed, record that state without treating the cross-reference as acceptance. These references do not replace the separate reviewer or owner dispositions.

| ADR | P0-02 installer-spec and (where applicable) issue #2 recovery-plan record/revision/disposition or scoped N/A | P0-06 threat/security record, threat IDs, exact revision/disposition (or N/A rationale) | P0-07 scope/resource/OS profile record, exact revision/disposition (or N/A rationale) | P0-08 threat-contract route, TIDs, applicable repeated slice-row IDs, exact crosswalk revision/hash and each row's status/phase gate; exact owner decision/authority reference for owner-confirmed/deferred routes (or N/A rationale) | P0-09 provenance record, frozen scope and qualified finding/disposition (or N/A rationale) |
|---|---|---|---|---|---|
| 001 | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| 002 | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| 003 | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| 004 | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| 005 | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| 006 | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| 007 | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| 008 | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| 009 (conditional) | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |

## ADR-001 — Original control panel or wholesale fork?

- Exact ADR document/path and revision reviewed:
- Route selected by the owner after review (leave all blank during evaluation): [ ] original implementation with independently written integrations  [ ] fork/direct reuse from specifically identified upstream  [ ] hybrid, itemized by component
- Exact upstream project, repository, tag/commit, and proposed reused files/assets (if any):
- Intended use and distribution scope to be reviewed (source/binaries/network service/derivative or combined work/other; reviewer to determine applicability):
- Exact related source/dependency/asset snapshot and artifact hashes:
- Original-code provenance and contributor-rights evidence:
- For each third-party component: exact version/source/files, applicable license/exception, LICENSE/NOTICE/source-offer/attribution obligations, modifications, and combined-work compatibility review:
- Separate asset/brand record (icons, fonts, screenshots, artwork, product names/marks) with source, rights/permission, intended use, and reviewer disposition:
- Declared repository license boundary: README/LICENSE declare GNU AGPL-3.0 for original Tong Panel code. This declaration alone does not establish rights or compatibility for any third-party code/assets or a combined distribution; reviewer to state the applicable scope and obligations:
- Related P0-09 record ID and exact revision/hash; owner-selected frozen distribution scope, artifact classes/channels and digests; component/asset findings; qualified licence/trademark reviewer disposition; and the separate maintainer distribution disposition (or unresolved gap). Preview-only SBOM evidence, Debian-builder omissions, and the caller-supplied WordPress runtime destination are source observations only; they do not select distribution scope or inclusion:
- Decision owner name and role:
- Required reviewers, roles, and review authority:
- Date (UTC):
- Outcome: [ ] Accepted  [ ] Rejected  [ ] Needs revision  [ ] Deferred
- Rationale:
- Alternatives considered:
- Evidence reviewed:
- Security impact:
- Licence / copyright / trademark impact:
- Migration or removal story:
- Residual risk:
- Residual-risk acceptor/authority and expiry or revisit trigger:
- Linked issue / PR and follow-up work:
- Recorded disposition reference:

## ADR-002 — Language and service boundaries

- Exact ADR document/path and revision reviewed:
- Selected language/runtime/build role by component (API, privileged agent, UI) and UI framework, or explicit alternate/defer choice:
- Exact supported language/compiler/toolchain versions by component; pin/update policy and reproducible-build evidence or limits:
- API↔agent IPC/protocol/schema revision; version negotiation, compatibility/deprecation and upgrade behavior:
- Named maintenance owner/workstream and support horizon for each component:
- Required API/agent/UI compatibility and boundary tests tied to exact revisions, commands and evidence:
- Migration/removal plan if a component, protocol or framework changes:
- Decision owner name and role:
- Required reviewers, roles, and review authority:
- Date (UTC):
- Outcome: [ ] Accepted  [ ] Rejected  [ ] Needs revision  [ ] Deferred
- Rationale:
- Alternatives considered:
- Evidence reviewed:
- Security impact:
- Licence / copyright / trademark impact:
- Migration or removal story:
- Residual risk:
- Residual-risk acceptor/authority and expiry or revisit trigger:
- Linked issue / PR and follow-up work:
- Recorded disposition reference:

## ADR-003 — Storage and jobs

- Exact ADR document/path and revision reviewed:
- Exact owner-selected product/workload profile and linked P0-07 Decision 1/3 revisions:
- Store scope by data class (panel metadata, durable jobs/queue, other state); explicitly exclude customer/site/offsite backups unless separately included and reviewed:
- Datastore and queue/transaction model selected per scope, including writer/concurrency/locking, leases, retry/idempotency/fencing behavior and owner-defined limits:
- Filesystem/mount assumptions and explicit network-filesystem/NFS allowance or exclusion:
- Schema/version migration, compatibility, failed-upgrade rollback and removal story:
- Metadata/job snapshot and backup/restore consistency, recovery procedure and evidence:
- Session/token validity or invalidation policy after metadata restore:
- Exact migration, concurrent-writer, crash/kill, full-disk, retry/idempotency, partial or unknown external-effect recovery/reconciliation, and backup/restore test contracts (profile/dataset/mount, code SHA, command, owner-defined pass/fail limits, results/artifacts):
- Decision owner name and role:
- Required reviewers, roles, and review authority:
- Date (UTC):
- Outcome: [ ] Accepted  [ ] Rejected  [ ] Needs revision  [ ] Deferred
- Rationale:
- Alternatives considered:
- Evidence reviewed:
- Security impact:
- Licence / copyright / trademark impact:
- Migration or removal story:
- Residual risk:
- Residual-risk acceptor/authority and expiry or revisit trigger:
- Linked issue / PR and follow-up work:
- Recorded disposition reference:

## ADR-004 — Application execution

- Exact ADR document/path and revision reviewed:
- Execution mode by workload/profile (leave blank until selected): [ ] native per-site identity/systemd scope  [ ] rootless container profile  [ ] hybrid, itemized by workload  [ ] defer customer workload/build execution
- Exact operations in scope (build/deploy/run/stop/update/log/restore or other) and phase for each:
- Workload identity and tenant/site boundary; customer files, secrets, database credentials, control-plane data, backup keys, host paths/sockets, and network egress available or denied:
- Runtime versions, OS/profile, process/capability/namespace/cgroup/UID and resource/time/disk limits:
- Build-script location, identity, egress, artifact privileges and isolation (or explicit deferral):
- Explicit host-root mount, privileged-container, Docker-socket, panel/agent-identity, and cross-tenant access denials:
- Exact hostile-workload threat/test contract revision and proposed profile to review at P0:
- Later P1 implementation evidence (actual test run, immutable code/image SHA, command, result, evidence artifact; leave blank until applicable implementation/CI gate):
- Phase boundary and exclusions (P1 zero-customer/P2/P3/P4) and linked owner decision:
- Decision owner name and role:
- Required reviewers, roles, and review authority:
- Date (UTC):
- Outcome: [ ] Accepted  [ ] Rejected  [ ] Needs revision  [ ] Deferred
- Rationale:
- Alternatives considered:
- Evidence reviewed:
- Security impact:
- Licence / copyright / trademark impact:
- Migration or removal story:
- Residual risk:
- Residual-risk acceptor/authority and expiry or revisit trigger:
- Linked issue / PR and follow-up work:
- Recorded disposition reference:

## ADR-005 — Distro and packaging scope

- Exact ADR document/path and revision reviewed:
- Exact P0-07 Decision 2 scope/support record revision and disposition linked to this ADR:
- Ordered candidate OS/version/architecture test matrix, distinct from any supported/certified matrix:
- For each candidate or claimed supported row: immutable source/image digest, installer/service profile and enabled optional capabilities, commands, required scenarios, owner-defined pass/fail limits and exact evidence:
- Explicit unsupported/untested profile exclusions and user-facing wording:
- Support decision invalidation triggers (source/image/profile change, failed/regressed check, newly discovered limitation) and required re-review/recovery:
- Decision owner name and role:
- Required reviewers, roles, and review authority:
- Date (UTC):
- Outcome: [ ] Accepted  [ ] Rejected  [ ] Needs revision  [ ] Deferred
- Rationale:
- Alternatives considered:
- Evidence reviewed:
- Security impact:
- Licence / copyright / trademark impact:
- Migration or removal story:
- Residual risk:
- Residual-risk acceptor/authority and expiry or revisit trigger:
- Linked issue / PR and follow-up work:
- Recorded disposition reference:

## ADR-006 — Mail and DNS modules

- Exact ADR document/path and revision reviewed:
- Phase/profile disposition for each separately: external DNS-provider workflows, authoritative PowerDNS/BIND, and mail/SMTP services (leave unselected until owner decision):
- Alternative under review: [ ] defer mail and authoritative DNS through P2; decide separately whether its domain preflight and HTTP-01/DNS-01 challenge flows remain  [ ] retain P2 domain ownership/preflight and DNS-01 provider challenge capability, while deferring hosted-zone management, authoritative DNS and mail to later scope  [ ] earlier, explicitly scoped optional-service profile with revised phase gates  [ ] other, specify:
- Enabled-by-default / opt-in / disabled behavior and opt-out effect on the selected web/core profile:
- Domain ownership/authorization, DNS provider principal/scope, zone/record/DNSSEC/TTL duties and credential custody/rotation/revocation:
- SMTP ingress/egress ports, blocked TCP 25 behavior, relay authentication, queue/rate/disk bounds, abuse response and external relay credentials:
- Exact failure/recovery cases and expected state (provider timeout, stale ACME challenge, partial DNS/config update, service reload failure, revoked credential, queue/disk exhaustion):
- Supported OS/profile, service identity, resource envelope, package/update/rollback boundaries and test evidence revision:
- Related P0-06 threat IDs/review record (including applicable T09/T13), P0-07 service/profile decision, and P0-09 provenance/notice record for included services/assets; exact revisions/dispositions or N/A rationale:
- Decision owner name and role:
- Required reviewers, roles, and review authority:
- Date (UTC):
- Outcome: [ ] Accepted  [ ] Rejected  [ ] Needs revision  [ ] Deferred
- Rationale:
- Alternatives considered:
- Evidence reviewed:
- Security impact:
- Licence / copyright / trademark impact:
- Migration or removal story:
- Residual risk:
- Residual-risk acceptor/authority and expiry or revisit trigger:
- Linked issue / PR and follow-up work:
- Recorded disposition reference:

## ADR-007 — Privilege boundary and secrets

- Exact ADR document/path and revision reviewed:
- Explicitly denied by default; exact first permitted typed verb(s), if any, and per-verb arguments/resources (leave blank until owner/security decision):
- API-to-agent protocol/schema revision, versioning and compatibility rule:
- Unix socket path, service identities/permissions and OS peer-authentication mechanism/profile:
- Principal/tenant/job/resource binding, authorization time (including immediately before each host effect), expiry, replay/idempotency/fencing and revocation behavior:
- Allowed path/file-descriptor model, canonicalization, symlink/TOCTOU and race protections, host-root/Docker-socket/shell denials:
- External-effect staging, idempotency, rollback/reconciliation and audit linkage/redaction/failure behavior:
- Secret types in scope and proposed custody/handoff/rotation/revocation/loss-recovery responsibilities (architecture to be determined; do not assume external custody):
- Required negative/positive test contracts, exact code SHA/OS/profile/commands, expected denial/effect results and artifact/log references:
- Separate maintainer privileged-agent boundary decision and reference to independent P0-06 review (do not collapse their dispositions):
- Decision owner name and role:
- Required security reviewers, roles, and review authority:
- Date (UTC):
- Outcome: [ ] Accepted  [ ] Rejected  [ ] Needs revision  [ ] Deferred
- Rationale:
- Alternatives considered:
- Evidence reviewed:
- Security impact:
- Licence / copyright / trademark impact:
- Migration or removal story:
- Residual risk:
- Residual-risk acceptor/authority and expiry or revisit trigger:
- Linked issue / PR and follow-up work:
- Recorded disposition reference:
- If accepted, exact allowed scope, explicit exclusions, and conditions/expiry/revisit trigger:

## ADR-008 — Installer and supply chain

- Exact ADR document/path and revision reviewed:
- Artifact classes covered (source archive, installer/bootstrapper, OS packages, embedded runtimes/UI, metadata/SBOM/notices; mark each included/excluded/separate for the intended release):
- Signing identity/organization and trust root; exact artifact/metadata binding and authenticated release-channel policy:
- Key generation, custody/storage, access separation, audit, backup, rotation, expiry, revocation, compromise/quarantine and recovery authority:
- Build inputs/toolchain pins, source commit/workflow provenance and reproducibility evidence or explicit limits:
- Per-artifact digest/signature, SBOM and NOTICE/licence/source-offer evidence bound to immutable hashes:
- Installer verification order before elevation, failure behavior for substitution/revoked/expired/malformed signatures, and operator notification:
- Downgrade/anti-rollback policy, authorized recovery target and reversible upgrade/restore behavior:
- Exact positive/negative package-swap, key-compromise/revocation, upgrade and recovery tests (code SHA, disposable profile, commands, results/artifacts):
- Evidence-only candidate pointer (if relevant; must not be treated as signed-release evidence):
- Related P0-09 record ID and exact revision/hash; owner-selected frozen distribution scope, artifact classes/channels and per-artifact digests; qualified licence/trademark reviewer findings/dispositions; separate maintainer distribution disposition; per-artifact notice/source-offer mapping; and unresolved gaps. Preview-only SBOM evidence, Debian-builder omissions, and the caller-supplied WordPress runtime destination are source observations only; they do not select distribution scope or inclusion:
- Decision owner name and role:
- Required security/release reviewers, roles, and review authority:
- Date (UTC):
- Outcome: [ ] Accepted  [ ] Rejected  [ ] Needs revision  [ ] Deferred
- Rationale:
- Alternatives considered:
- Evidence reviewed:
- Security impact:
- Licence / copyright / trademark impact:
- Migration or removal story:
- Residual risk:
- Residual-risk acceptor/authority and expiry or revisit trigger:
- Linked issue / PR and follow-up work:
- Recorded disposition reference:

### Reusable hold / deferral record — not an ADR outcome by itself

A temporary hold records that no formal ADR outcome has been reached; leave the canonical outcome blank unless/until an authorized owner records one of the register's statuses. If the register later adds **Deferred** and the owner selects it, link that exact status to this record. A hold differs from **Needs revision** (the proposal needs changes before a decision) and **Out of scope** (the owner excludes scope rather than deferring its decision). Complete a separate copy for each held ADR and link it from issue #1 or another durable follow-up; leave unused copies blank. A hold does not count as acceptance or close any gate.

- ADR and exact revision deferred:
- Decision/scope held and explicit exclusions:
- Downstream gate/package consequence while deferred (blocked, limited, or conditional; state exact effect):
- Condition or evidence required before reconsideration:
- Revisit trigger:
- Target revisit date (UTC):
- Accountable follow-up owner and authority:
- Decision owner/reviewer required to lift or revise the deferral:
- Linked issue and durable deferral/disposition record:

## Separate owner choice — ADR-009 scope and index status

ADR-009 is a separate proposal and is not assumed to be part of the existing P0-05 “eight ADR” scope. The owner must make and record this scope/indexing choice; no choice is preselected. If the scope/index choice is deferred, or ADR-009 is kept outside P0-05 or the indexed ADR set, a separate canonical authentication/session disposition must be recorded in the ADR register and issue #1 before the proposed P1 authentication package 03 in PR #160 at head `4a71768256a38d6c052461b6cd8871298e6f3435` can proceed. If ADR-009 is added to the P0 decision set, only an Accepted disposition for an exact revision that explicitly accepts the required authentication/session policy can satisfy that predecessor; Rejected, Needs revision, or Deferred leaves it unmet. An indexing or scope choice alone neither approves authentication nor authorizes login implementation.

**Available proposal evidence:** `docs/ADR-009-AUTHENTICATION.md` was fetched from P1 proposal commit `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23` (blob `deff69603093b7d443d1c51d01907a6019d330a4`). It labels itself PROPOSED/unapproved, says no login route or production credential store is approved, and lists explicit benchmark, key-custody, recovery, proxy/origin and session questions. This source pointer makes the proposal reviewable; it does not decide whether to add/index it or accept its content.

- Exact ADR-009 document/path and revision considered:
- Current ADR index/issue scope revision considered:
- Decision owner name and role:
- Required reviewers and authority:
- Date (UTC):
- Scope choice: [ ] Add ADR-009 to P0-05 / issue #1 and index  [ ] Keep ADR-009 outside current P0-05 scope  [ ] Defer scope/index decision
- If scope/index is deferred or ADR-009 is outside the P0-05 decision set, reference the separate canonical authentication/session disposition in the ADR register and issue #1 required before package 03 proceeds. If ADR-009 is added, reference its exact full disposition and package dependency; only Accepted with the required policy recorded satisfies the predecessor:
- Alternatives considered:
- Evidence reviewed:
- Security impact:
- Licence impact, if any:
- Residual risk and acceptor/expiry, if applicable:
- Linked issue / PR and required index/backlog follow-up:
- ADR-009's stated issue #6 acceptance condition: exact issue #6 body/revision reviewed, plus the issue #6 record of the owner/security disposition linked to the corresponding ADR-009 and issue #1 records (leave blank until recorded):
- The ADR-009/predecessor disposition is design evidence only; it does not establish issue #6 acceptance or completion of issue #6's executable negative-test requirement.
- Recorded scope decision reference:

If the owner adds ADR-009 to the decision set, record its Accepted / Rejected / Needs revision outcome using a separate full disposition block with the same fields above. A formal Deferred outcome may be used only if the authorized canonical writer has added that status to the register; otherwise leave the outcome blank and use the linked hold record. Only an Accepted outcome for the exact revision that records the required authentication/session policy meets package 03’s ADR predecessor; the other outcomes or a pending hold leave it unmet until a separate canonical disposition is recorded in the ADR register and issue #1. Adding it to scope does not approve its content or authorize a login implementation.

## Source-evidence and reviewer agenda — proposals only

**No ADR outcome is recorded by this agenda.** The recommendations below are conditional review positions for discussion against pinned `main@0aad68e81138a0f226e83e401f90bae4d0b6bddc` and the supplemental revisions cited. They do not change the blank outcome fields above, constitute owner decisions, or authorize implementation. Reviewers must verify exact revisions and evidence before any disposition.

| ADR | Recommended review position (agenda only) | Evidence/action needed before an outcome | Source references |
|---|---|---|---|
| 001 — Original code and provenance | Compare at least (a) original implementation with independently written compatible integrations and (b) fork/direct reuse from a specifically named upstream revision; an itemized hybrid or deferral may also be considered. No route or disposition is selected. The pinned ADR-001 proposes an original Tong Panel AGPL-3.0 codebase and compatible integrations to standard Linux services; that proposal alone does not establish third-party rights, combined-work compatibility, or asset/mark clearance. | For the selected intended scope, itemize first-party provenance/contributor rights, exact third-party versions/files/licenses/notices/source obligations and combined-work review, separate asset/brand permissions, security/maintenance, and migration/removal strategy. Before P0-09 review, cite the owner-reviewed P0-07 Decision 0/2 profile and ADR-008 artifact/channel scope to freeze the intended distribution surface. Reconcile P4's bounded inventory and unresolved archive/notice evidence; obtain a qualified licence/trademark disposition. | Baseline `docs/ADR_REGISTER.md` §ADR-001; `docs/UPSTREAM_RESEARCH.md` integration checklist; P0-07 Decisions 0/2 and ADR-008 artifact/channel fields, when reviewed; P4 packet `docs/p0-review/p4-provenance-license-readiness.md` at PR #158 head `8b05989bf94ef596c33142bade980f97565b7b4f`, blob `20e1a2f3dccb154da9f5239e48ef896973d129bd`; local evidence crosswalk, if reviewed. |
| 002 — Runtime/service boundary | Proposal to evaluate the Go unprivileged API, separate narrow Go agent, and TypeScript/React UI direction against the register-listed Laravel/PHP monolith, Rust daemon, and single-process Node alternatives; no architecture selection recorded. | Record chosen UI framework (React is proposed in the ADR while architecture leaves the framework flexible), toolchains, owning roles, protocol/version policy and maintenance horizon. Independently review the API/agent boundary; cite P0-06's independent threat/privileged-boundary findings and P0-07 Decision 0/3 profile or state explicitly which evidence remains unknown. Comparative team capability and maintenance evidence are not established by the technology proposals and must be supplied or remain unknown. | Baseline `docs/ADR_REGISTER.md` §ADR-002; `docs/ARCHITECTURE.md` proposed implementation technologies, lines 28–32; P0-06 and P0-07 Decisions 0/3, when reviewed. |
| 003 — Metadata and jobs | Evaluate SQLite-first and PostgreSQL datastore profiles for a named workload; explicitly consider ADR-003’s PostgreSQL-only and Redis-only-queue alternatives as tradeoffs, not selected designs. No store or queue selection is recorded. Reconcile BLUEPRINT and ADR-003 before recommending a default. | Resolve the metadata-store conflict; define data scope, transaction/job semantics, writer/filesystem limits, migration and failed-upgrade recovery, metadata/job backup/restore and post-restore session validity. Link exact P0-07 profile/decision records; require owner-set contention, migration, crash/retry and restore evidence. This does not select a P1 customer/site/offsite backup lifecycle; broader scope needs its own decision/evidence. | Baseline `docs/ADR_REGISTER.md` §ADR-003; `docs/BLUEPRINT.md` §8; `docs/ARCHITECTURE.md` control-plane storage; this template's P0-07 Decisions 1/3, if reviewed. |
| 004 — Workload execution | Compare native per-site identities/systemd scopes, compatible rootless-container profiles, a workload-specific hybrid, or deferring customer builds/execution beyond the P3-proposed zero-customer P1 boundary if the owner adopts it. Explicitly assess the register-listed container-only and native-only alternatives as tradeoffs; no runtime mode or scope is selected. | Decide exact operations and phase by workload; identities/data/secret/network/host-path boundaries; build-script isolation; resource limits; and a reviewer-dispositioned hostile-workload threat/test contract with its proposed profile for P0 review. Record actual adversarial run, code/image SHA, command and result only at the later applicable implementation/CI gate. Do not accept host-effect scope before its required P0-06 security review. The P2 threat draft recommends deferring app-build/isolation acceptance until downstream scope is explicit; separately, the canonical P3 plan specifies Node/Python applications. Align ADR-004's runtime order with those proposals without treating them as an adopted boundary. Cite the exact P2 threat/test proposal and P0-06 threat disposition plus P0-07 Decision 0/3 workload/profile choices; keep every unresolved profile or execution mode explicitly unknown. | Baseline `docs/ADR_REGISTER.md` §ADR-004; `docs/ARCHITECTURE.md` workload boundaries; `docs/phases/P3-APP-PLATFORM.md` Node/Python scope; P2 threat/test proposal at exact commit `317c72058b2609a4127f253e7e2a0f98d0688cd8` (proposal only); P0-06 and P0-07 Decisions 0/3, when reviewed; P3 scope backlog at PR #160 head `4a71768256a38d6c052461b6cd8871298e6f3435` (proposal only). |
| 005 — OS/package scope | Consider Ubuntu 24.04 and Debian 12 x86_64 for initial lab evaluation, subject to owner choice; no supported-OS decision or claim recorded. Debian 13 evaluation timing remains unresolved: INSTALLER_SPEC lists it among initial validation targets, while ADR-005/TEST_STRATEGY place it after or contingent on earlier CI evidence. Architecture also calls for proving specific Debian/Ubuntu x86_64/arm64 combinations first, while ADR-005 puts arm64 after verified CI. Arm64 remains an evaluation candidate only; no support decision is recorded. Also review the register-listed universal-installer and Docker-only alternatives as scope tradeoffs, not accepted deployment modes. | Resolve the candidate architecture/test-order conflicts and link the exact P0-07 Decision 2 matrix; keep architecture/test order distinct from supported/certified scope. For each claimed supported row, record immutable source/image and installer profile, its exact P0-02 installer-spec disposition, owner-set acceptance limits/scenarios/evidence, exclusions and triggers for re-review after a changed image/source or failed check. Measure minima on the owner-chosen profile or keep values unknown; P0 plan review does not represent executed installation results. | Baseline `docs/ADR_REGISTER.md` §ADR-005; `docs/ARCHITECTURE.md` §Supported platform promise; `docs/INSTALLER_SPEC.md` initial compatibility plan; `docs/TEST_STRATEGY.md` supported OS matrix; this template's P0-07 Decision 2, if reviewed. |
| 006 — Mail/DNS | Compare deferring mail/authoritative DNS through P2 while deciding whether P2's proposed domain preflight and HTTP-01/DNS-01 challenge flows remain; retaining P2's proposed domain ownership/preflight and DNS-01 provider challenge capability while deferring hosted-zone management, authoritative DNS and mail; or introducing a precisely scoped optional profile earlier with revised gates. No service or phase is selected. | Record phase/profile for each service, domain authorization, ports/credential ownership, default-off/opt-out behavior, abuse limits and failure/recovery evidence. P3 proposes a zero-customer P1 boundary; if the owner adopts it, assess P2's proposed per-site Nginx/PHP-FPM, domain ownership/preflight and ACME challenge flows at their proposed phase, and P3's proposed exclusion of mail and authoritative DNS from P1. P2/P3 proposals do not set canonical scope. The P2 threat draft identifies mail as optional P4 scope and does not set an authoritative-DNS phase; the canonical P4 plan specifies optional mail and authoritative DNS. Any earlier inclusion requires revised scope, dependencies and acceptance. | Baseline `docs/ADR_REGISTER.md` §ADR-006; canonical P1/P2/P4 scope plans at `main@0aad68e81138a0f226e83e401f90bae4d0b6bddc`, including `docs/phases/P4-HOSTING-PARITY.md` mail/DNS scope; P3 scope/backlog draft PR #160 at head `4a71768256a38d6c052461b6cd8871298e6f3435` (proposal only); `docs/INSTALLER_SPEC.md`; `docs/SECURITY.md`; P2 T09/T13 proposal at commit `317c72058b2609a4127f253e7e2a0f98d0688cd8` (proposal only). |
| 007 — Privilege and secrets | Require independent security review; evaluate whether the identified control gaps require ADR revision. Proposal for review: keep privileged-operation authorization unresolved pending identity, authorization, replay, path, audit and secret-custody/handoff decisions; no security disposition recorded. Preserve the register’s explicit rejection of a root web server or unrestricted sudo as a rejected alternative, not an approved mode. | Decide exact typed verbs/denials, OS peer/socket identity, principal/tenant/job/resource binding and effect-time reauthorization, expiry/replay/fencing, path/race safety, durable audit/reconciliation, secret custody/handoff/rotation/revocation/recovery (without prespecifying external custody). Map P2 T02/T03/T04/T06/T10/T15 and issue #153 into reviewed contracts; issue #153 is design-only and does not approve secret custody or host operations. Keep the maintainer privileged-boundary decision separate from independent security findings. Do not enable a privileged operation before the required recorded review; any deferral needs an owner/security decision and explicit gate revision. | Baseline `docs/ADR_REGISTER.md` §ADR-007; `docs/THREAT_MODEL.md` privileged-agent boundary; P2 threat/test readiness at `317c72058b2609a4127f253e7e2a0f98d0688cd8`; P3 stage 7 (canonical package 06 / issue #8) in PR #160; linked issue #153, if reviewed. |
| 008 — Installer/release integrity | Evaluate the release-security proposal against owner and security-review criteria, including whether revision is needed. Preserve the register’s stated rejection of piping a moving main branch to sudo bash without provenance/integrity; do not present that alternative as accepted. PR #149's open draft has an `UNSIGNED-developer-preview` artifact and candidate check/VM-smoke results only; they do not establish a signed trust chain, supported OS, or release approval. | Decide signing identity, trust root and key custody/authority; bind exact artifact classes and immutable source/build provenance to digests/signatures/SBOM/notices; require verification before elevation; define compromise/revocation, anti-downgrade and upgrade/recovery plan. Cite the exact P0-02 installer-spec disposition and, where in scope, issue #2 recovery-plan disposition (or scoped N/A); leave actual installation/recovery results to the later applicable implementation/CI gate. | Baseline `docs/ADR_REGISTER.md` §ADR-008; `docs/INSTALLER_SPEC.md` installer chain, failure cases and CI validation; bounded PR #149 evidence at head `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23`. |
| 009 — Authentication/session (conditional content review; separate from scope/index choice) | Treat exact P1 ADR-009 at commit `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23` as a proposed policy for owner/security review, not an approved design. Keep approval of public login and production credential-store scope blocked until an exact canonical policy and reviewed threat/test contracts are recorded. No setting or outcome is selected here. | Review the candidate against SECURITY.md's public-alpha authentication controls and THREAT_MODEL T01/T02/T08; obtain the P0-06 independent security/threat disposition and identify the P0-07 Decision 3 workload/runtime/resource profile. If profile values are unknown or out of scope, record that explicitly and keep affected package gates unmet. Schedule KDF/concurrency measurements and control execution evidence at the applicable P1 gate before public-login scope advances. Exact policy settings and owner outcome remain for the authorized owner; the independent reviewer records a separate review disposition. Preserve the separate ADR-009 scope/index choice above; scope/index alone does not approve content. | P1 `docs/ADR-009-AUTHENTICATION.md` at exact commit `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23`, blob `deff69603093b7d443d1c51d01907a6019d330a4`; baseline `docs/SECURITY.md` lines 18–20; baseline `docs/THREAT_MODEL.md` T01/T02/T08; P0-06 review record; P0-07 Decision 3; issues #1 and #6. |


ADR-009 remains a separate scope/index question as recorded in the owner-choice section above; this agenda does not resolve that question or determine ADR-009's content outcome.

## Consolidated record and boundaries

- Decision-record revision / immutable commit:
- ADRs with recorded outcomes:
- ADRs still pending or deferred:
- Conditions and follow-up owners/issues:
- Owner confirmation that outcomes are recorded in the canonical ADR register and linked decision issue:
- Exact `UPDATES.md` findings entry/anchor and revision satisfying issue #1's documentation requirement:
- Independent security/licence review evidence references:

**Authorization boundary:** Completing this template is not itself an ADR acceptance, security or legal approval, implementation authorization, merge approval, deployment approval, production authorization, or approval to mutate a host. Any such decision must be separately and explicitly recorded by the authorized decision-maker for its exact scope.
