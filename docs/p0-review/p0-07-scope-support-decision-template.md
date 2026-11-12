> **Preservation draft:** historical working evidence; factual statements and links require review. No approval or phase completion is implied. Machine-local paths and private coordination details have been omitted.

# P0-07 Scope, Support, and Resource Decision Record — Blank Template

**Status:** BLANK TEMPLATE — no architecture, support, resource, or phase-boundary decision is recorded here.

This form captures five separate owner decisions. Every choice and value remains blank for the authorized decision-maker and reviewers. Do not infer a production-supported platform from a candidate test target, and do not invent resource floors or phase assignments.

## Exact review inputs

- Repository:
- Canonical source baseline SHA (expected P0 planning snapshot: `0aad68e81138a0f226e83e401f90bae4d0b6bddc`; reviewer to verify):
- Exact revision of `docs/phases/P0-PLANNING.md`:
- Exact revision of `docs/ADR_REGISTER.md`:
- Exact revisions of `docs/ARCHITECTURE.md`, `docs/BLUEPRINT.md`, `docs/FEATURE_MATRIX.md`, `docs/ROADMAP.md`, `docs/INSTALLER_SPEC.md`, and `docs/TEST_STRATEGY.md`:
- Issue #3 exact body/acceptance-statement snapshot: read-only GitHub fetch at `2026-11-12 00:48 UTC`; title “P0-03: Finalize P1 MVP scope, initial Linux matrix and issue breakdown”; state `open`; `updated_at` `2026-10-26T01:36:08Z`; body SHA-256 `4169d2af61b19ad0500acf21a710a8971e912ba88028a53b976e59fcd8d3ab4d` (810 UTF-8 bytes, LF line breaks, no trailing newline). Exact acceptance text: “At least one full critical path to a safe test environment, with zero live-server edits and clear phase acceptance; UPDATES.md checkpoint updated.” Source: https://github.com/parmtech/Tong-Panel/issues/3. This is a source snapshot only; reviewer must reverify it against the issue when recording a disposition. It does not record issue acceptance or owner authority.
- Related P1/P2/P3 evidence revisions reviewed:
- Decision record revision / immutable commit:
- Evidence files, test reports, and artifact locations:

## Source-evidence annex for owner review — claims/conflicts to verify

The following are source observations at the pinned baseline, not selected options or owner decisions. Review the exact cited revisions and record outcomes only in the blank decision sections below. The P3 draft at PR #160 head `4a71768256a38d6c052461b6cd8871298e6f3435` is supplemental proposal evidence, not a canonical decision.

| Topic | Source references at `main@0aad68e81138a0f226e83e401f90bae4d0b6bddc` | Claim/conflict to verify; owner choice remains blank |
|---|---|---|
| Metadata datastore | `docs/BLUEPRINT.md` §8 / lines 120–124 proposes PostgreSQL panel metadata; `docs/ADR_REGISTER.md` ADR-003 / lines 20–24 proposes small-server SQLite first and PostgreSQL only after testing; `docs/ARCHITECTURE.md` §control-plane storage / line 32 permits PostgreSQL or SQLite and defers finalization to benchmarks and backup/upgrade tests. | The default is unresolved across the source documents. Verify exact scope (metadata/jobs), single-writer and local-filesystem assumptions, benchmarks, migration and restore evidence. Do not preselect SQLite or PostgreSQL here. |
| Debian 13 and architecture proof order | `docs/INSTALLER_SPEC.md` “Initial compatibility plan” / lines 6–9 groups Debian 12/13 with Ubuntu 24.04 x86_64 for validation; `docs/ADR_REGISTER.md` ADR-005 / line 33 places Debian 13 after verified CI; `docs/TEST_STRATEGY.md` / line 37 names Ubuntu 24.04 and Debian 12 as candidate lab targets and says add Debian 13 based on evidence; `docs/ARCHITECTURE.md` “Supported platform promise” / line 69 says first prove specific fresh Debian/Ubuntu x86_64/arm64 combinations. | The candidate test sequence and architecture proof order are underspecified across these sources. In particular, ADR-005's “arm64 after verified CI” needs reconciliation with ARCHITECTURE's Debian/Ubuntu x86_64/arm64 proof requirement, including the exact OS/architecture matrix and what “verified CI” means. This does not make any distro supported or certified. |
| P1 foundation operations vs zero-customer boundary | `docs/ROADMAP.md` lines 14–16 lists API, auth/session/MFA/RBAC, instance identity, metadata migrations, durable jobs and simulated operations in P1; the pinned PR #160 proposal at reconciliation §“P3 minimum P1 scope/backlog cross-check” describes a zero-customer foundation with hosting lifecycle downstream. | No direct contradiction is established: simulated tenant/identity/job foundations can differ from live customer-facing hosting. The owner should explicitly map permitted P1 operations and test fixtures to the canonical roadmap, say whether “zero-customer” excludes all live/customer-facing hosting, and record which auth, tenant, and job work is simulated versus installable or production-facing. Keep the decision and entry gate open. |
| Optional mail and DNS scope (ADR-006) | `docs/BLUEPRINT.md` lines 18–20 and Gates 2/4 / lines 85–90 include broad hosting features while describing mail and authoritative DNS as optional; `docs/ADR_REGISTER.md` ADR-006 / lines 38–42 proposes Nginx/PHP/MariaDB as a minimal profile, external DNS first, with Postfix/Dovecot and BIND/PowerDNS separately installable and optional; `docs/ROADMAP.md` P2/P4 / lines 19–30 places core single-server hosting in P2 and managed DNS/optional mail in P4; `docs/FEATURE_MATRIX.md` / lines 23–25 labels mail and DNS P3 and describes advanced databases under P3. | Exact service inclusion and phase placement remain unresolved across these documents. Record a separate owner choice for mail, external DNS integration, and managed authoritative DNS, including include/exclude/defer outcome, phase/entry gate, alternatives, rationale, exact source revisions, and supporting evidence. Do not imply an operational or supported service profile from these aspirational labels. |
| Phase vocabulary and Feature Matrix priority labels | `docs/FEATURE_MATRIX.md` header / line 5 calls the column “Priority,” while its legend / line 3 calls P0 foundation, P1 hosting MVP, P2 modern apps, P3 full hosting, P4 scale and P5 stable, and says phase references come from ROADMAP. `docs/ROADMAP.md` “Milestone definitions” / lines 10–38 and `docs/PHASE_INDEX.md` / lines 6–12 define seven phases P0 planning, P1 foundation, P2 hosting, P3 apps, P4 full hosting, P5 scale and P6 stable. `docs/phases/P0-PLANNING.md` task sequence / lines 9–19 separately uses P0-01 through P0-10 as task IDs. | The phase crosswalk is inconsistent/ambiguous: clarify whether FEATURE_MATRIX “Priority” values are priority tiers or phase IDs, designate the canonical operational phase index, and keep P0-01 through P0-10 identified as task IDs. Do not infer scope from the matrix labels. |
| Numeric resources and runtime profile | `docs/INSTALLER_SPEC.md` lines 8, 12, and 34 calls for candidate targets, CPU/memory/storage preflight and tested resource profiles but supplies no approved numeric floors or runtime versions; `docs/TEST_STRATEGY.md` lines 10 and 37 calls for workload/resource testing and candidate OS profiles but gives no numeric minimum; `docs/FEATURE_MATRIX.md` line 62 says performance targets should come from baseline measurement, not guesses. P3 PR #160 draft likewise leaves CPU/RAM/free-disk/concurrency and supported runtime versions unresolved. | No approved CPU, RAM, free-disk, concurrency, or general supported runtime/toolchain minimum is recorded in these sources. Measure against an owner-defined profile or explicitly mark unknown/out of scope; do not invent values. |
| PR #14 preflight thresholds and test coverage | Open draft PR #14 exact head [`a246ac7950d3e99ff5794333acbf584f3d8bd53d`](https://github.com/parmtech/Tong-Panel/commit/a246ac7950d3e99ff5794333acbf584f3d8bd53d): [`doctor.go`](https://github.com/parmtech/Tong-Panel/blob/a246ac7950d3e99ff5794333acbf584f3d8bd53d/internal/preflight/doctor.go#L132) warns only below 1,048,576 kB RAM; [line 139](https://github.com/parmtech/Tong-Panel/blob/a246ac7950d3e99ff5794333acbf584f3d8bd53d/internal/preflight/doctor.go#L139) warns only below 8,589,934,592 bytes free root disk. [`doctor_test.go`](https://github.com/parmtech/Tong-Panel/blob/a246ac7950d3e99ff5794333acbf584f3d8bd53d/internal/preflight/doctor_test.go#L48-L50) exercises a known-resource fixture at 2 GiB RAM/15 GiB disk; [lines 65–72](https://github.com/parmtech/Tong-Panel/blob/a246ac7950d3e99ff5794333acbf584f3d8bd53d/internal/preflight/doctor_test.go#L65-L72) cover unknown readings, but not known-low values, below/at/above boundaries, or disk-full behavior. [`READ_ONLY_DOCTOR.md`](https://github.com/parmtech/Tong-Panel/blob/a246ac7950d3e99ff5794333acbf584f3d8bd53d/docs/READ_ONLY_DOCTOR.md#L18) describes the figures as preliminary development guidance. | These are observed provisional code thresholds, not measured product minima, supported floors, or approved support policy. Equality passes both strict-less-than checks. The known test does not validate threshold behavior; documentation claiming low-resource tests exceed the test coverage. No resource/support choice is made here. |
| PR #14 Go module/docs/local/CI versions | At the same PR #14 head, [`go.mod`](https://github.com/parmtech/Tong-Panel/blob/a246ac7950d3e99ff5794333acbf584f3d8bd53d/go.mod#L3) declares Go 1.23, [`READ_ONLY_DOCTOR.md`](https://github.com/parmtech/Tong-Panel/blob/a246ac7950d3e99ff5794333acbf584f3d8bd53d/docs/READ_ONLY_DOCTOR.md#L7) says Go 1.23+, the PR description self-reports local Go 1.23.2 (not pinned in this source record), and [CI](https://github.com/parmtech/Tong-Panel/blob/a246ac7950d3e99ff5794333acbf584f3d8bd53d/.github/workflows/go-checks.yml#L22) selects Go 1.27.3. | The pinned module/docs claims are compatible; CI version-matrix coverage is limited to 1.27.3. The mutable PR-body self-report is not treated as pinned execution evidence. This indicates a coverage gap, not demonstrated incompatibility or an owner-selected runtime profile. |
| PR #149 candidate-lab image/toolchain inputs | Exact run 42790473632, PR #149 candidate head 4dab4c60cc86186935e9a7d987b9fe1ca97a1f23, base 482017005f9872c1fe7f2969ebfb477ea7de30dd, merge ref da44246220147560b6d7c1510f32cc1edcae3af7. Ubuntu guest job 129742295654: Ubuntu 24.04.5 LTS, x86_64/amd64; image https://cloud-images.ubuntu.com/releases/server/24.04/release-20261017/ubuntu-24.04-server-cloudimg-amd64.img, SHA-256 04e8f2461a16a44e78a370284471f29616cdd9baf1893df25807b71f985b6c94. Debian guest job 129742295031: Debian GNU/Linux 12 (bookworm), x86_64/amd64; image https://cloud.debian.org/images/cloud/bookworm/20261026-2643/debian-12-genericcloud-amd64-20261026-2643.qcow2, SHA-512 0464b5d4ae3aa4f658468d88da83af7857b71548b30d46dcb6282770955a747fb95a983476db90d25afd0653af607224c7ebd05122ab71faa106fd712aaad652. At exact PR head, go.mod blob 9737ad41c6eb33929aa3dc98d75606d792e95181 declares Go 1.23.0; CI workflow blob 5c965562342ed4a4dff8de945bb7d0fabb016fc7 selects Go 1.27.3. | Candidate-lab inputs/results only: the workflow reports check/run results associated with the candidate head, while the clean-VM jobs actually checked out the recorded merge ref. This is not head-only execution evidence. It does not measure CPU, RAM, disk, or concurrency; cover arm64 or Debian 13; establish a broader package/image pinning policy; or approve support, runtime, or resource profiles. See P0-10 §8 for the run/check-to-checkout distinction. |

## Conditional owner-review recommendations — proposals only

The following review positions make the evidence actionable; they are not selections, approvals, or numeric policy. The owner must complete the blank decision sections below against exact revisions.

| Decision | Proposal to evaluate | Minimum record/evidence before disposition |
|---|---|---|
| 0 — P1 baseline and P1/P2 boundary | Evaluate a zero-customer foundation, but map every permitted operation to ROADMAP P1's API, auth/session, identity, jobs and simulated-ops scope. State explicitly whether zero-customer excludes live/customer-facing hosting and which tenant/auth/job paths are simulated versus installable or production-facing. Defer customer/site lifecycle, mail and authoritative DNS unless the owner expands scope with separate tests and threat review. Clarify separately whether P0 installer specification, P1 non-production foundation work, and the first installable/release package occur at different phase gates. | Exact included operations/exclusions and installer phase per deliverable; P1/P2 entry gates; simulated vs live/customer-facing boundaries; dependencies; ADR/threat revisions; rationale; owner and reviewer decisions. Issue #3 additionally needs the safe-test critical path and updated `UPDATES.md`. |
| 1 — Metadata/job store and workload semantics | Do not adopt SQLite, PostgreSQL, or Redis as a default from conflicting draft text. Compare SQLite and PostgreSQL against a named single-host workload; keep panel metadata/jobs distinct from customer/site backups. | Data classes; writer and filesystem limits; queue durability/idempotency; concurrency and workload; migration, crash/full-disk, retry, backup/restore and post-restore session criteria; test method/plan, planned environment, owner-set pass/fail limits, and any available results. P0 records the decision evidence needed for the choice; execute required package tests at the applicable P1 acceptance gate. |
| 2 — OS/architecture order vs support | Consider Ubuntu 24.04 and Debian 12 as the first candidate-lab sequence, then Debian 13 only after the owner defines and accepts a CI evidence bar. Reconcile that order with ARCHITECTURE's requirement to prove specific fresh Debian/Ubuntu x86_64/arm64 combinations. Do not label any target supported from candidate jobs alone. | Immutable image/source digests; exact OS/architecture test order and “verified CI” threshold; how the proof sequence satisfies both ADR-005 and ARCHITECTURE; clean install, upgrade, rollback and recovery evidence; separate candidate/tested/supported status and owner/QA disposition. |
| 3 — Resource/runtime profile | Do not adopt PR #14's strict-below 1 GiB RAM and 8 GiB free-disk warnings as product floors. Measure an owner-defined workload/profile or record unknown/out-of-scope without implying issue #3 acceptance; the Go 1.27.3-only CI choice is a coverage gap, not demonstrated incompatibility. | Named workload and image; CPU/RAM/disk/concurrency/toolchain method and limits; below/at/above threshold and disk-full evidence if preflight is relied on; owner and QA/performance review. |
| 4 — Phase/feature vocabulary | Establish the canonical phase index and clarify whether FEATURE_MATRIX values are priority tiers or phase IDs; keep P0-01–10 explicitly labeled as planning task IDs. | Exact phase definitions and feature/package mapping, P1/P2 boundary, dependencies, entry/exit evidence, and owner-approved change plan for roadmap, matrix, issues and handoffs. |

Issue #3 remains open, unassigned, and without comments in the 2026-11-11 read. Its minimum resource/support acceptance cannot be marked met by an unknown profile unless the authorized issue owner revises that criterion. All proposals above leave the owner-choice fields blank.

| Owner decision / evidence field (complete only after review) | Blank record |
|---|---|
| Sources and exact revisions independently rechecked | _Reviewer to determine_ |
| Conflict resolved / unresolved and rationale | _Reviewer to determine_ |
| Decision owner, authority, reviewers, and date | _Reviewer to determine_ |
| Decision-record link and follow-up evidence | _Reviewer to determine_ |

## Decision authority and review

- Decision owner / maintainer name and role:
- Required architecture/security reviewers, names, roles, and authority:
- QA/release reviewer, name, role, and authority:
- Additional reviewers and authority:
- Reviewer independence / conflict statement:
- Decision/review date and time (UTC):
- Review method and evidence reviewed:
- Overall record status: [ ] Complete  [ ] Needs revision  [ ] Deferred
- Conditions and follow-up issues:

## Decision 0 — Minimum P1 architecture baseline

This decision records whether the minimum P1 architecture is locked for a named scope. A datastore choice alone does not establish the full component and interface baseline.

Record candidate profiles and alternatives as provisional review inputs while required architecture ADRs, threat decisions, and applicable security findings remain unresolved. Do not select **Locked for named scope** until the exact supporting ADR revisions and required P0-06 decisions are recorded; then reconcile the baseline against those decisions. If the selected scope differs from the reviewed provisional profile, reopen the affected P0-06/P0-09 evidence and decisions before locking or accepting dependent ADRs.

- Named P1 product profile and permitted scope:
- Selected minimum P1 components and their responsibilities:
- Required component interfaces and trust boundaries:
- Provisional/pending ADRs and threat decisions relevant to this profile, with exact source revisions and status:
- Related P0-08 threat-contract routing crosswalk exact revision/hash; cite each applicable repeated slice-row ID with its covered/uncovered portion, route status, and phase-entry gate. For any `owner-confirmed` or `deferred` slice route, cite the exact confirming owner decision/authority reference. Distinguish proposed routing from canonical THREAT_MODEL assignments and owner decisions:
- Exact accepted ADR IDs and source revisions supporting this baseline:
- Alternatives considered and rationale:
- Unresolved architecture choices or explicitly out-of-scope components:
- Evidence or constraints reviewed:
- Outcome: [ ] Locked for named scope  [ ] Unresolved  [ ] Out of scope  [ ] Deferred
- Decision owner and required reviewer dispositions, names/roles/date:
- Linked issue / PR / follow-up documentation:

If any required ADR or threat decision above remains pending, select **Unresolved** and keep this profile provisional; do not mark it locked. Record the final lock only after those required inputs are decided and the reviewed P0-06/P0-09 scope still matches.

## Decision 1 — Metadata/job datastore and storage constraints

### Owner decision

- Exact datastore default for the named product profile:
- Datastore scope (metadata, jobs, other state):
- Alternatives considered:
- Chosen transaction and concurrency model:
- Single-writer behavior and limits:
- Filesystem/storage assumptions (including local filesystem requirements or exclusions):
- Backup, upgrade, migration, and recovery constraints:
- Source-session/token validity after metadata snapshot restore, including treatment of stale or revoked credentials (decision and exact canonical record reference):
- Explicit out-of-scope profiles or workloads:
- Outcome: [ ] Accepted  [ ] Rejected  [ ] Needs revision  [ ] Deferred

### Evidence and review

- Exact ADR/source revisions:
- Decision-relevant benchmarks/evidence and exact revisions; label test contracts as planned and distinguish any available executed results:
- Full-disk/journal fault-injection scenario and owner-approved acceptance criteria for metadata and durable-job state (consistency, attributable/recoverable failure, partial or unknown external-effect handling, planned test/environment/code revision; actual result if already available):
- Required implementation/CI gate for executing this test before any dependent package is accepted:
- Security impact:
- Data-integrity and recovery implications:
- Migration/removal story:
- Residual risks and rationale:
- Residual-risk acceptor/authority and expiry/revisit trigger:
- Decision owner and required reviewer dispositions, names/roles/date:
- Linked issue / PR / follow-up documentation:

## Decision 2 — Candidate OS/architecture test order versus supported certification

### Candidate test profile and order

- Candidate OS/distribution and exact version(s):
- Candidate architecture(s):
- Installer/service profile and enabled optional capabilities for each OS/image row (web-only / standard / app-developer or owner-defined profile; list enabled/disabled optional services):
- Test order and rationale:
- VM image/source identifiers and immutable digests required for evidence:
- OS package repositories/snapshot identifiers, exact package versions or immutable snapshot manifest, and pin/update policy for each OS/image profile:
- Package-to-OS compatibility evidence and re-review triggers for repository, snapshot, or package-version changes:
- Required clean-VM scenarios and acceptance criteria:
- Evidence required before changing candidate status:

### Issue #3 safe-environment critical-path evidence

- Exact critical-path issue/work-item reference and reviewed revision:
- End-to-end path from clean test environment through expected safe result, including the zero-live-server-edit boundary:

| Scenario / step | OS, architecture, image digest, and profile | Exact command / procedure revision | Expected result / oracle | Actual result / status | Immutable log or artifact reference and digest | Reviewer / date |
|---|---|---|---|---|---|---|
| | | | | | | |
| | | | | | | |
| | | | | | | |

- Failure, rollback, cleanup, or recovery evidence and open limitations:
- Owner acceptance of the complete critical path and exact evidence revision:

### Separate support/certification decision

- OS/architecture officially claimed as supported, if any:
- Certification decision and exact evidence revision:
- Unsupported or untested profiles and user-facing wording:
- Conditions or exclusions (including host/container assumptions):
- Support outcome: [ ] Supported for named scope  [ ] Candidate lab only  [ ] Not supported  [ ] Deferred

### Decision owner and review

- Alternatives considered:
- Security/installer/recovery implications:
- Residual risks and acceptor/authority/expiry:
- Decision owner and required reviewer dispositions, names/roles/date:
- Linked issue / PR / follow-up documentation:

## Decision 3 — Resource floors and runtime/toolchain profile

For each quantity, enter a measured value with method/evidence, or record it as unknown or out of scope. A measured test-environment setting is not automatically a proposed product minimum or a supported floor. Record the intended use of each profile and link any product-minimum/support claim to exact owner-approved acceptance evidence. Leave the decision blank until the owner chooses; do not estimate without evidence.

| Profile / workload | CPU minimum | RAM minimum | Free disk minimum | Concurrency / workload ceiling | Measurement method and exact environment | Runtime / toolchain versions | Evidence state (measured / unknown / out of scope) | Intended use (test fixture only / proposed product minimum / owner-claimed supported floor / unknown / out of scope) |
|---|---|---|---|---|---|---|---|---|
| Panel profile (owner to define) | | | | | | | | |
| Agent / host-service profile (if included) | | | | | | | | |
| Build / tenant-app profile, if included | | | | | | | | |
| Other named profile | | | | | | | | |
| Authentication/KDF workload, if included | | | | | | | | |

For a datastore comparison or capacity claim, capture the workload dimensions below for every profile or mark each unknown/out of scope. These are inputs to a repeatable test, not prescribed product minima:

- Test dataset size and shape (record types/counts, representative job mix and growth assumptions):
- Tenant/site count represented by the dataset:
- Concurrent API writers and operation mix:
- Concurrent active jobs, queued jobs, and job mix:
- Filesystem and mount used by the test dataset, including whether it is local or networked:
- Owner-defined latency, error, resource, and recovery pass/fail limits, or explicitly “not set”:

- Which runtime/toolchain versions are selected for each profile:
- Required benchmark or workload evidence and exact revision:
- KDF parameter decision (if authentication is in scope), measured latency/concurrency on the smallest owner-approved VM with exact image digest/method, or explicit unknown/out-of-scope disposition; canonical ADR-009 or equivalent decision reference:
- Related P0-08 threat-contract routes for each profile: exact crosswalk revision/hash, TIDs, route status and phase-entry gate; identify unmapped/downstream contracts, including distinct T15 power interruption, reboot, network interruption, process termination, and full-disk cases and their evidence status, without implying execution or an accepted assignment:
- Alternatives considered:
- Security, capacity, upgrade, and compatibility implications:
- Explicit unknowns or exclusions and revisit triggers:
- Residual risks and acceptor/authority/expiry:
- Decision outcome: [ ] Accepted  [ ] Rejected  [ ] Needs revision  [ ] Deferred
- Decision owner and required reviewer dispositions, names/roles/date:
- Linked issue / PR / follow-up documentation:

## Decision 4 — Canonical phase vocabulary and P1/P2 boundary

### Owner decision

- Canonical names and meaning for each affected phase:
- P1 scope, including whether it is a zero-customer foundation and its exact permitted operations:
- P2 scope and hosting features that begin only after its entry gate:
- Work explicitly excluded from P1 and the gate/phase where it may be considered:
- Canonical source of phase labels and mapping for feature/backlog IDs:
- Required dependency order and entry/exit evidence:
- Exact blank P0-08 T01–T15 threat-contract crosswalk revision/hash; owner-review each route against the canonical THREAT_MODEL phase assignment and record explicit downstream/unmapped status and phase-entry gate. Reconcile these mixed/unresolved slices individually: T02 request-vs-job authorization; T03 API/agent vs host-effect scope; T04 upload/extraction vs restore paths; T06 token vs backup-key handling; T08 endpoint/CSRF vs SSRF/webhook replay; T09 ownership/preflight vs DNS/ACME service flows; T11 installer vs release/signature workflow; T12 firewall vs SSH/network rollback; and T15 full disk, power interruption, reboot, network interruption, and process termination. Also track T05/T07/T13/T14 phase slices, T04 scope, and T10 broad-restore. This enumeration is a review checklist, not a phase reassignment or approval; do not assign a phase from a TID or proposed crosswalk alone.
- Threat-boundary slices with different or unresolved phase ownership: record each slice separately; do not assign a phase from a TID alone.

| Threat/boundary slice | Candidate phase(s) or unassigned | Canonical phase/issue ID | Dependencies | Entry evidence | Exit evidence | Assignment status (unassigned / proposed / owner-decided) | Owner/reviewer |
|---|---|---|---|---|---|---|---|
| | | | | | | | |
| | | | | | | | |
| | | | | | | | |

- Outcome: [ ] Accepted  [ ] Rejected  [ ] Needs revision  [ ] Deferred

### Review and change control

- Conflicting documents/labels reviewed at exact revisions:
- Alternatives considered and rationale:
- Security impact and risk of phase-boundary ambiguity:
- FEATURE_MATRIX `Priority` meaning: [ ] priority tier  [ ] phase ID  [ ] other (specify):
- Explicit mapping of each FEATURE_MATRIX priority value to the canonical phase/priority system:
- Confirm P0-01 through P0-10 remain planning task IDs, with any separate phase mapping recorded here:
- Evidence and exact source revisions for the semantics/mapping:
- Migration/reconciliation plan for roadmap, feature matrix, issues, and handoffs:
- Residual risks and acceptor/authority/expiry:
- Decision owner and required reviewer dispositions, names/roles/date:
- Linked issue / PR / follow-up documentation:

### ADR-006 service-scope choice — owner decision remains blank

Record each service separately; choices below do not imply support or implementation. For each, specify include, exclude, or defer; the phase and entry gate; alternatives and rationale; exact source revisions/evidence; and owner/reviewer dispositions.

| Capability | Owner choice (include / exclude / defer) | Phase and entry gate | Alternatives and rationale | Exact sources/evidence | Owner and reviewer disposition/date |
|---|---|---|---|---|---|
| Mail services (including Postfix/Dovecot, if considered) | | | | | |
| External DNS provider integration | | | | | |
| Managed authoritative DNS (including BIND/PowerDNS, if considered) | | | | | |

## Consolidated owner decision

- Decision owner name, role, and authority:
- Reviewer names, roles, authority, and dates:
- Decisions accepted / rejected / revised / deferred (identify each separately):
- Conditions, open questions, and accountable follow-up references:
- Canonical documents updated only after the recorded owner decision:
- Exact record revision / immutable commit:

**Authorization boundary:** Preparing or completing this form is not by itself acceptance of an ADR, support certification, implementation authorization, merge approval, deployment approval, security signoff, or permission to mutate a host. Candidate test results do not constitute supported-OS certification. Any implementation or phase transition requires its own explicit, recorded gate decision.
