> **Preservation draft:** historical working evidence; factual statements and links require review. No approval or phase completion is implied. Machine-local paths and private coordination details have been omitted.

# P0-10 Design Readiness and P1 Code/CI Go/No-Go Record

**Status:** BLANK TEMPLATE — no decision, approval, security signoff, or phase-gate result is recorded here.

> Complete this record only from review of the exact candidate revision and its cited evidence. “Approve” applies only to the named change and stated scope. This record does not accept ADRs by implication, approve production use, authorize merge/deployment, certify an OS, pass P1 exit, or authorize P2 entry.

## 1. Candidate change and immutable review target

| Field | Recorded value |
|---|---|
| Repository | parmtech/Tong-Panel |
| Candidate change title / purpose | _Reviewer to determine_ |
| Pull request URL and number | _Reviewer to determine_ |
| Candidate branch | _Reviewer to determine_ |
| Candidate head SHA | _Reviewer to determine_ |
| Base branch and SHA | _Reviewer to determine_ |
| Changed paths and scope reviewed | _Reviewer to determine_ |
| Exact CI/check names, run IDs/URLs, and conclusion; for each check record its PR-head association and actual checkout ref/SHA/base | _Reviewer to determine_ |
| Checks missing, skipped, stale, or on another SHA | _Reviewer to determine_ |
| Review packet/source revisions used | _Reviewer to determine_ |
| Decision record date and time (UTC) | _Reviewer to determine_ |

Do not transfer results from another head, base, branch, or artifact without recording why they apply to this candidate.

## 2. Decision authority and independent review

| Role | Name / account | Authority or review scope | Exact revision reviewed | Date (UTC) |
|---|---|---|---|---|
| Project owner / maintainer decision-maker | _Reviewer to determine_ | Design-readiness review (approval authority to be identified) | _Reviewer to determine_ | _Reviewer to determine_ |
| Authorized code/CI go-no-go decision-maker (role/authority to be identified) | _Reviewer to determine_ | Named P1 change decision after design-readiness review | _Reviewer to determine_ | _Reviewer to determine_ |
| Independent security reviewer | _Reviewer to determine_ | Threat model, privileged boundary, residual risks, security evidence | _Reviewer to determine_ | _Reviewer to determine_ |
| Qualified licence/trademark reviewer, when applicable | _Reviewer to determine_ | Distribution scope, provenance, notices, brand/trademark use | _Reviewer to determine_ | _Reviewer to determine_ |
| QA/release or other required reviewer | _Reviewer to determine_ | Reproducibility, check evidence, package/VM scope | _Reviewer to determine_ | _Reviewer to determine_ |

A proposed workstream, author, issue reporter, automated helper, green check, or relay is not an assigned or independent reviewer unless the authorized reviewer is explicitly identified and their disposition is recorded.

## 3. P0-01–P0-10 evidence and decision status

Use a factual status for each row (for example, verified, partial, missing, or not applicable); cite exact revisions, files/lines, issue/PR/check IDs and artifact locations. Leave unresolved decisions unresolved.

| P0 item | Evidence and exact references | Status | Remaining decision/evidence | Accountable owner and reviewer |
|---|---|---|---|---|
| P0-01 — Blueprint, feature matrix, baseline roadmap | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| P0-02 — UX and installer specifications / UX acceptance | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| P0-03 — Upstream survey and licence warnings | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| P0-04 — Agent handover protocol and progress log | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| P0-05 — ADR review and outcomes | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| P0-06 — Formal threat model and test contracts | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| P0-07 — Minimum P1 architecture, OS/runtime and resource assumptions | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| P0-08 — Owned backlog, reviewers and dependencies | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| P0-09 — Licence and trademark provenance review | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| P0-10 — This exact P1 code/CI change after design-readiness review | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |

### P0-01–04 artifact state versus issue #1–#4 acceptance

The P0 task checkbox records artifact preparation only. Record issue acceptance separately; a checked task or existing document does not satisfy an issue or P0 exit criterion. For issue #1, cite the exact body/revision and record whether the ADR_REGISTER vocabulary is aligned with P0-05; issue #1 asks for review status but defines no status labels. For issue #2, cite the exact body/revision and map the security-design disposition to the issue's explicit Acceptance clause: reviewed security design linked to acceptance tests and P1 work items, findings in `UPDATES.md`, and no untested security claims. Record installer/OS/firewall-recovery plan disposition separately as a listed issue #2 Deliverable, not as text from that Acceptance clause. Determine the overall issue status against the explicit Acceptance clause and any authorized owner-adopted criterion or updated issue revision; do not add installer recovery as an issue-closure criterion unless so adopted or updated. Keep both slices separate below and cross-reference P1 work-item IDs/revisions and linked acceptance-test contracts from P0-02/P0-06 without merging security and installer/recovery evidence. For issue #3, capture owner-approved OS/architecture/resource requirements plus the accepted package-to-issue dependency order and linked acceptance/test plan; keep proposal-only mappings separate. For issue #4, record UX specification separately from usability/accessibility findings.

| Issue | Related P0 task(s) | Exact issue acceptance statement/revision | Artifact and evidence revisions | Issue disposition / status | Remaining evidence or gate | Owner and reviewer authority/date |
|---|---|---|---|---|---|---|
| #1 — ADR decisions | | | | | | |
| #2a — Security design | | | | Compare with the exact issue #2 Acceptance clause | | |
| #2b — Installer/OS/firewall recovery | | | | Issue #2 Deliverable plan disposition; not an extra Acceptance clause absent an authorized owner decision or updated issue text | | |
| #3 — Scope/backlog | | | | | | |
| #4 — UX/usability/accessibility | | | | | | |

## 4. ADR dispositions (record each separately)

For every ADR, record an explicit owner outcome, qualified review where relevant, exact reviewed version, rationale, alternatives, evidence, security/licence impact, migration/removal consequences, and residual risk. Use the canonical ADR_REGISTER outcomes (`Accepted / Rejected / Needs revision`). Record `Deferred` as a formal outcome only if the authorized register owner adds that status to the exact register revision; otherwise leave the ADR outcome blank and record a temporary hold separately, as described in P0-05. Blank fields mean no disposition is recorded.

| ADR | Exact revision reviewed | Owner outcome (Accepted / Rejected / Needs revision; formal Deferred only if added to ADR_REGISTER, otherwise record separate temporary hold per P0-05) | Rationale, alternatives and evidence | Reviewer(s), date, residual risks |
|---|---|---|---|---|
| ADR-001 | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| ADR-002 | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| ADR-003 | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| ADR-004 | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| ADR-005 | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| ADR-006 | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| ADR-007 | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| ADR-008 | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |

### ADR-009 treatment

- Exact source revision and whether the proposal is canonical or supplemental: _Reviewer to determine_
- Owner decision on inclusion, indexing, and issue/backlog scope: _Reviewer to determine_
- ADR-009 outcome, qualified security review, rationale, evidence, and unresolved items: _Reviewer to determine_

Do not count a branch-only proposal as a canonical ADR or an accepted decision without an explicit owner record.

## 5. Required P0 exit criteria

For each criterion, cite exact evidence and record the disposition, accountable owner, reviewer, and remaining limitation. A passing code check does not substitute for a decision or human review.

| P0 exit criterion | Evidence and exact references | Status / disposition | Owner and reviewer | Remaining limitation |
|---|---|---|---|---|
| No contradictory stack/install claims in README, ADRs, and architecture | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| Each risky design decision has owner, rationale, alternatives, and review status | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| Threats map to planned tests; privileged-agent boundary is approved by maintainer | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| Distro/runtime promise remains proposed until clean-VM CI passes | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| No unreviewed third-party source is copied; known trademark limits are recorded | _Reviewer to determine_ (cite P0-09 record ID, exact revision/hash, frozen source/distribution snapshot, covered artifact paths, and qualified reviewer plus maintainer dispositions; reconcile every §9 Item ID against its scope selection and reviewer disposition, and carry every unknown, open, excluded, not-reviewed, or out-of-scope item into remaining limitations. An aggregate P0-09 gate outcome does not clear an unresolved item.) | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| P1 issue backlog has dependency order, acceptance, and linked test plan | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| UPDATES checkpoint is accurate and documentation links work | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |

## 6. Specific evidence gates for P0-06 through P0-09

- **P0-06 threat/security:** threat assets and boundaries, mitigations, linked test contracts, responsible owners, residual risks, independent reviewer findings, and explicit maintainer disposition of the privileged-agent boundary: _Reviewer to determine_.
- **P0-07 scope/support:** P1/P2 boundary; exact candidate OS/architecture and runtime profile; measured resource assumptions or explicit unknowns; candidate-vs-supported wording; owner decision and clean-VM evidence scope: _Reviewer to determine_.
- **P0-08 backlog:** accepted canonical package ID ↔ issue mapping for packages 01–10, including duplicate/gap/split disposition; owner-confirmed dependency order; named accountable assignees with responsibility acceptance and named independent reviewers/authority/disposition; per-package acceptance/test plans and security/licence checklist with reviewer disposition or reviewed N/A; cite exact issue/P0-08 record revisions. For design-readiness, cite the exact P0-08 threat-contract routing crosswalk revision/hash and account for every T01–T15 route and each applicable boundary-slice row, including its covered portion, uncovered/downstream portion, route status, and phase-entry gate. For any `owner-confirmed` or `deferred` slice route, cite the exact confirming owner decision and authority reference; P2/P3 proposal text alone cannot establish it. Treat routing entries as evidence only; they do not establish assignments, accepted test contracts, issue acceptance, or owner decisions. Mark the P0-08 gate outcome separately and state its effect on this candidate. PR #160 and the blank P0-08 matrix are proposal/preparation only, not accepted assignments or backlog: _Reviewer to determine_.
- **P0-09 provenance:** exact record ID/revision/hash; frozen distribution snapshot and package paths; component/source/version/hash/SPDX/NOTICE inventory; generated/source/UI/assets provenance; SBOM/provenance/signature coverage and exclusions; contributor-rights evidence; qualified licence/trademark reviewer disposition and any legal-counsel referral. Reconcile every P0-09 §9 Item ID against its §10 scope coverage and maintainer decision; carry unknown, open, excluded, not-reviewed, and out-of-scope items into this record's remaining limitations. A satisfied aggregate P0-09 gate does not clear any unresolved item. State explicitly whether NOT VERIFIED/open or an actual recorded disposition: _Reviewer to determine_.

## 7. Named P1 code/CI change decision

This decision is separate from ADR dispositions, security review, eventual P1 exit, and P2 entry.

- Change and exact candidate SHA: _Reviewer to determine_
- Scope approved for this decision, including explicit exclusions: _Reviewer to determine_
- Required checks and exact results on that SHA: _Reviewer to determine_
- Known failures, skipped checks, limitations, and residual risks: _Reviewer to determine_
- Mandatory-check rule and disposition for a failed, missing, skipped, stale, or wrong-SHA result; any exception authority and exact recorded exception: _Reviewer to determine_
- Decision invalidation triggers (including changed head/base or later check regression), required re-review, and hold/rollback/recovery action: _Reviewer to determine_
- Cross-gate scope/evidence invalidation triggers: identify whether each design-readiness and code/CI disposition must be repeated after a change to P0-05–P0-09 decisions/evidence, especially ADR-001 route/upstream source/assets, ADR-004 execution mode/workload/build-isolation scope, P0-07 Decision 0/2/3, ADR-006 service/profile choices, ADR-008 artifact classes/channel/signing/bootstrap plan, and the P0-09 frozen distribution/artifact scope or inclusion modes; record affected scope, exact revisions, required re-review, authority, and date: _Reviewer to determine_
- Unresolved P0-05–P0-09 decisions/evidence relevant to this candidate and their effect (block / condition / defer / other), with authorized rationale: _Reviewer to determine_
- If a required ADR outcome or ADR-009 scope/index choice is Deferred, cite its P0-05 deferral/scope record; identify each resulting unmet P0/P1 predecessor and candidate scope held or limited. Any separate scope exception requires its own authorized rationale, exact scope, authority, and date. Deferred is not Accepted and satisfies no required predecessor: _Reviewer to determine_
- Design-readiness outcome (**Approve / Hold / Reject**): _Reviewer to determine_
- Change-specific code/CI go/no-go (**Approve / Hold / Reject**): _Reviewer to determine_

### Separate decision records

Record the design-readiness decision first. Record the change-specific code/CI decision separately and only after design readiness has an explicit disposition; do not treat the two as simultaneous or interchangeable gates. A code/CI **Approve** outcome requires design readiness to be **Approve**, its required conditions to be satisfied, and all required P0 predecessors to be met. If design readiness is **Hold** or **Reject**, the code/CI outcome must be **Hold** or **Reject**. Any exception must be recorded separately by its authorized decision-maker with exact scope and rationale; it does not relabel an unmet P0 predecessor as satisfied.

**Design-readiness decision**

- Design-readiness decision-maker identity and authority: _Reviewer to determine_
- Exact evidence/revisions and P0 exit criteria reviewed for this decision: _Reviewer to determine_
- Design-readiness rationale, conditions, limitations, and date (UTC): _Reviewer to determine_
- Durable record/reference for this disposition: _Reviewer to determine_

**Change-specific code/CI decision**

- Code/CI decision-maker identity and authority: _Reviewer to determine_
- Exact candidate SHA/base, required-check conclusions, and exceptions reviewed for this decision: _Reviewer to determine_
- Code/CI rationale, conditions, limitations, and date (UTC): _Reviewer to determine_
- Durable record/reference for this disposition: _Reviewer to determine_

### Independent security disposition

- Exact revision and evidence reviewed: _Reviewer to determine_
- Security disposition (**Approve / Conditional / Hold / Reject / Not reviewed**): _Reviewer to determine_
- Findings, residual risks, conditions, and required follow-up: _Reviewer to determine_
- Independent reviewer identity/qualification, authority, and date (UTC): _Reviewer to determine_
- Exact P0-06 record ID/revision/hash documenting reviewer independence/conflicts and the related threat-model/maintainer dispositions: _Reviewer to determine_

### Separate phase status

- P0 exit decision and evidence record: _Reviewer to determine_
- P1 exit decision (separate later gate): _Reviewer to determine_
- P2 entry decision (separate later gate): _Reviewer to determine_

## 8. Scope, exclusions, and limitations

- Included code, documentation, workflows, artifacts, and interfaces reviewed: _Reviewer to determine_
- Excluded paths, platforms, environments, threat cases, or distributions: _Reviewer to determine_
- Evidence not independently reproduced or unavailable: _Reviewer to determine_
- Required follow-up owners, reviewers, issues, and due conditions: _Reviewer to determine_

### Bounded source-diff inventory — PR #149 (not a review disposition)

This inventory compares exact PR #149 head `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23` with exact base `482017005f9872c1fe7f2969ebfb477ea7de30dd`: GitHub compare reports 3 commits ahead, 0 behind, and these eight changed paths. Blob IDs below are from the exact head.

| Risk area | Changed path and exact-head blob | Scope note / exact-head test source |
|---|---|---|
| Tenant file API, authorization boundary, cancellation | `internal/api/files_api.go` — `962479c794149d182200eec2d30e24e7f098ba4c` | File handlers retain `files:read`/`files:write` checks against the authenticated principal's tenant and pass request context into mutations. |
| Linux file API adapter | `internal/api/files_linux.go` — `ba61708eb1c7330beb49379a49b51feb55767e5b` | Context-aware tenant-root write/remove adapter. |
| Tenant-root selection | `internal/files/tenant_roots_linux.go` — `ae6afc4ab5c03f7aa1bfa009c750aa1f1cf26b60` | Carries context through server-selected pinned tenant roots. |
| Pinned-root file operations | `internal/files/root_linux.go` — `2a187e4f4ebc3a28f951b3b532c740ca62c2e447` | Linux bounded staged write/removal cancellation checks; source describes this package as unprivileged and not wired to the agent. |
| API boundary and quota/cancellation tests | `internal/api/files_api_test.go` — `7afe281840926f2fb049dd5f002c3dc0322673a3` | Linux-only exact-head tests cover authenticated tenant isolation, role/CSRF and invalid-path cases, cancellation before body consumption/mutation, uncertain publication accounting, and concurrent create-only writes using test adapters. |
| Pinned-root filesystem tests | `internal/files/root_linux_test.go` — `06f94062c6b97087c08c63b413d910745da09d23` | Linux-only exact-head tests include cancellation-at-EOF without publication and canceled removal preserving the file, alongside existing path/symlink/limit/staging tests. |
| Documentation | `UPDATES.md` — `8f0571e0be13c30f9fe789ee3ae5b8d9545ae6d7`; `docs/P1_EXIT_GATE_REPORT_2026-10-27.md` — `c90063d933d715e2a84f39dc9edd0f1acce26c46` | Status and evidence narrative only. |

#### Recorded workflow-to-test linkage (status not refreshed)

This table maps immutable workflow sources to invocation scope. Run/job outcomes below are the previously recorded evidence in this packet; they were not refreshed for this inventory.

| Workflow/source blob | Test invocation and recorded run | Scope qualification |
|---|---|---|
| `.github/workflows/go-checks.yml` — `5c965562342ed4a4dff8de945bb7d0fabb016fc7` | `go test -race ./...`, followed by `go test ./internal/agent -run '^$' -fuzz '^FuzzDecodeRequestOnlyAllowsPing$' -fuzztime=10s`; recorded `test` check run `42790473198` | Runs on `ubuntu-latest`; Linux build tags include both cited Linux test files. The 10-second fuzz invocation is bounded workflow coverage, not a sustained fuzz campaign. |
| `.github/workflows/p2-clean-vm-smoke.yml` — `65d270e349e5817aeab0721750d76e234168b9cd`; `scripts/clean_vm_smoke.sh` — `fd9a9cec8f0b692bcaee573a829802fe04ccd30a` | Builds Linux/amd64 `files.test`, `api.test`, and `sites.test` with `go test -c`, then runs all three full test binaries; recorded Ubuntu/Debian jobs `129742295654` and `129742295031` in run `42790473632` | The recorded guest logs show checkout of merge ref `da44246220147560b6d7c1510f32cc1edcae3af7`, with the exact candidate head and base as its parents; this is candidate-with-base evidence, not a head-only checkout. |
| `.github/workflows/api-contract-checks.yml` — `deb3b6baebb10c27fa73874cb3979da10bcd3acc` | `go test ./internal/api -run '^TestCurrentOpenAPIContractScenarios$'`; recorded contract run `42790473182` | Compiles the API test package but selects only the OpenAPI contract test; it does not execute the added file test functions. |
| `.github/workflows/p1-preview-package.yml` — `8f4d53bda62f595f6d0e12a3d606e516d1b12970`; `.github/workflows/p1-smoke.yml` — `e29ce1f335fcef022c1a1629f6f6b82eda27a6e6`; `.github/workflows/go-vulnerability-checks.yml` — `cc18a9c45d0a77ecc005ff12b3d458ecee9fca09` | Recorded runs: preview package `42790473328`, developer smoke `42790473311`, vulnerability `42790473467` | Build/smoke scripts and `govulncheck -test` analysis do not execute the named API/filesystem test functions. |

No auth/session subsystem, agent or privileged host-action, jobs/audit/recovery, installer/package/signing/SBOM, or UI implementation paths changed in this comparison. This audit did not run tests. The cited test sources do not establish cross-process durable quota/barrier integration; this inventory establishes no host/filesystem/kernel support matrix, independent review, or production evidence. It is a bounded source-diff map only, not design approval, security review, code/CI go/no-go, P0/P1 completion, or Phase 1 authorization. All decision fields above remain blank for authorized reviewers.

## 9. Final record

- Decision summary and rationale: _Reviewer to determine_
- Exact evidence links and retained artifact locations: _Reviewer to determine_
- Recorded by / authority: _Reviewer to determine_
- Date (UTC): _Reviewer to determine_

**No approval is recorded by this blank template.** Do not treat its existence, completed technical checks, proposed workstream roles, or unfilled fields as permission to merge, release, deploy, modify a production host, claim support, complete P0/P1, or enter P2.

## Evidence-only annex — PR #149 (reviewer disposition still required)

This annex records PR evidence for reviewer consideration; it makes no design-readiness, code go/no-go, security, or phase-gate decision. PR #149 head `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23` is confirmed against `refs/pull/149/head` on `origin`; the supplied PR metadata states base `482017005f9872c1fe7f2969ebfb477ea7de30dd`. The bounded diagnostics read recorded at 2026-11-11 03:25 UTC in the reconciliation reports seven completed successful workflow runs, comprising eight named check/job results associated with the PR head (the six standard checks plus the separate Ubuntu 24.04 and Debian 12 clean-VM jobs). This describes run/check metadata association, not a verified checkout of the head for every workflow. There are no additional standalone status checks in that snapshot. The clean-VM workflow returned no downloadable artifacts; its GitHub job logs are available and were reviewed. The logs show selected smoke steps, but are not a downloadable guest-artifact bundle or complete retained VM evidence. The workflow matrix contains Ubuntu 24.04 amd64 and Debian 12 amd64 only; it has no Debian 13 job. A separate unsigned developer-preview package artifact is available and was confirmed downloadable in the 2026-11-11 05:39 UTC refresh; it is listed to expire 2026-11-17 and is not a substitute for VM artifacts or support certification.

### Hosted checks associated with exact head (VM smoke uses PR merge ref)

The supplied evidence reports eight passing checks on `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23`, grouped under these run records:

| Check group / scope | Run and reported result |
|---|---|
| `test` — Go checks | [Actions run 42790473198](https://github.com/parmtech/Tong-Panel/actions/runs/42790473198) — completed successfully |
| `contract` — API contract | [Actions run 42790473182](https://github.com/parmtech/Tong-Panel/actions/runs/42790473182) — completed successfully |
| `unsigned-dev-archive` | [Actions run 42790473328](https://github.com/parmtech/Tong-Panel/actions/runs/42790473328) — completed successfully |
| `daemon-dry-run` | [Actions run 42790473311](https://github.com/parmtech/Tong-Panel/actions/runs/42790473311) — completed successfully |
| `govulncheck` | [Actions run 42790473467](https://github.com/parmtech/Tong-Panel/actions/runs/42790473467) — completed successfully |
| `browser` | [Actions run 42790473233](https://github.com/parmtech/Tong-Panel/actions/runs/42790473233) — completed successfully |
| Ubuntu 24.04 amd64 clean VM smoke | [Actions job 129742295654](https://github.com/parmtech/Tong-Panel/actions/runs/42790473632/job/129742295654) — succeeded |
| Debian 12 amd64 clean VM smoke | [Actions job 129742295031](https://github.com/parmtech/Tong-Panel/actions/runs/42790473632/job/129742295031) — succeeded |

Seven workflow runs are recorded as successful and associated with the proposed PR head in the run/check metadata, producing eight named successful check/job results. The recorded guest logs independently identify the clean-VM checkout as merge ref `da44246220147560b6d7c1510f32cc1edcae3af7` against base `482017005f9872c1fe7f2969ebfb477ea7de30dd`; this is evidence for the proposed head tested with that base, not a head-only checkout. The non-VM workflow checkout SHAs were not independently recorded in this packet. These workflows use `pull_request` triggers and their checkout steps omit an explicit `ref`; GitHub documents that the default checkout for this event uses the PR merge branch ([GitHub Actions `pull_request` event](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#pull_request)). The clean-VM workflow has Ubuntu 24.04 amd64 and Debian 12 amd64 jobs only; both succeeded, including the “Boot clean guest and run the extracted preview smoke” step. Logs are accessible through the job links above although no downloadable workflow artifacts were returned. They include image verification/checksum handling, clean guest boot, restore-test and restored-config results, homepage/upload smoke, and negative checks denying private-config and privileged database access. These selected smoke results are candidate-lab/component evidence; they do not establish complete P1 acceptance, support, or certification. The separately available unsigned developer-preview artifact is not a VM log or support certification.

### Historical clean-VM evidence and attempt limits

The PR body reports that [Actions run 42785167173](https://github.com/parmtech/Tong-Panel/actions/runs/42785167173) initially failed in both guests: Ubuntu 24.04 reportedly encountered a mirror package-index size mismatch before product smoke; Debian 12 reportedly passed Nginx/PHP-FPM and WordPress isolation, then a sample homepage request timed out. This failure narrative is **PR-body-reported, not independently confirmed**. The currently available jobs and logs for this run are for the latest attempt. They show both Debian job `129729063671` and Ubuntu job `129729064012` completed successfully, checking out PR merge ref `4674c84899e4cebb4bbba41bea339fd5ebbec5d7` with parents `4d42ca024a7628e424a9738aa258d70927122f02` and base `482017005f9872c1fe7f2969ebfb477ea7de30dd`. Latest-attempt logs show successful real homepage and upload smoke on both platforms. The wrapper provides no per-attempt API selector or attempt-1 artifacts/logs; these latest-attempt results do not resolve or independently confirm the reported first-attempt failures or prove their retry linkage. The reported Ubuntu mismatch and Debian timeout remain unverified.

The documentation follow-up head `8764d0fef218a36a4cc9411e5481656d014cdf0c` has a reported successful clean-VM run [42787324539](https://github.com/parmtech/Tong-Panel/actions/runs/42787324539): Ubuntu job `129731713114` and Debian job `129731712883` show successful completion of the guest boot/product-smoke step. This supports a passing run on that different head; available API output does not establish that it was specifically a retry of failed jobs. The historical run query does not provide its commit/base SHA, so it cannot be tied to PR #149's exact head/base through available API evidence. Separately, run `42790473632` is associated with the PR #149 head; its Ubuntu 24.04 and Debian 12 jobs checked out the recorded merge ref as detailed above.

### Explicit status and limits

The supplied PR record states **P1 remains incomplete**. These passing checks do not establish P0 design readiness, independent security approval, supported-OS certification, production support, P1 exit, or P2 entry. The P2-named workflow is a clean disposable-VM smoke check; its name does not grant phase entry. No production VPS evidence was examined. Any approval, reviewer identity, owner disposition, or phase decision remains for the authorized reviewer to enter in the blank fields above.
