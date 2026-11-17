> **Preservation draft:** historical working evidence; factual statements and links require review. No approval or phase completion is implied. Machine-local paths and private coordination details have been omitted.

# P0-02 UX Acceptance Evidence Record — Blank Template

**Status:** BLANK TEMPLATE — no UX acceptance, accessibility conformance, implementation, or phase approval is recorded here.

This record captures reviewed UX/design evidence against the canonical P0-02 UX requirements and the linked issue #4. The issue title says “P0-04”; that issue is a UX-flow anchor, while this form uses the P0 planning task number P0-02. The label difference does not change the canonical P0 task numbering. Complete only against exact design revisions and observed review evidence. Leave every result and disposition for the authorized reviewer to determine.

## 1. Exact review inputs and scope

- Repository:
- Canonical baseline SHA: `0aad68e81138a0f226e83e401f90bae4d0b6bddc` (verify before review):
- `docs/phases/P0-PLANNING.md` exact revision reviewed:
- `docs/UX_UI_SPEC.md` exact revision reviewed:
- `docs/FEATURE_MATRIX.md` exact revision reviewed:
- `docs/PHASE_INDEX.md` / relevant roadmap revision, if used:
- Issue #4 exact body snapshot: read-only GitHub fetch at `2026-11-12 00:57 UTC`; title “P0-04: Validate UX flows for Tong Panel's grid home and File Explorer”; state `open`; `updated_at` `2026-10-26T01:36:10Z`; body SHA-256 `7f6a9abe6c3669114016a0f22cf3ac5dde4b8d7723681d1cb95fe959d1983493` (713 UTF-8 bytes, LF line breaks, no trailing newline). Exact Acceptance text: “Documented, reviewed usability tasks and accessible navigation flows; report findings in UPDATES.md. Design artifacts are not backend implementation.” Source: https://github.com/parmtech/Tong-Panel/issues/4. This is source evidence only; reverify before a disposition. It records no UX acceptance or owner review.
- Exact design prototype, wireframe, build, and artifact revisions/URLs:
- Included user roles, workflows, product areas, and platforms:
- Explicitly excluded or planned-only flows and the later gate for reconsideration:
- Review date/time (UTC):

### Existing source evidence context — no UX disposition

| Source | Exact revision | Verified content and boundary |
|---|---|---|
| [UX/UI specification](https://github.com/parmtech/Tong-Panel/blob/0aad68e81138a0f226e83e401f90bae4d0b6bddc/docs/UX_UI_SPEC.md) | `84c508becc9c67a7be77e716ee8c7f3287fefadb` | Labels itself proposed/nothing implemented. It defines proposed information architecture, file-explorer actions/safety, wizards, a WCAG 2.2 AA target, responsive/performance goals, 10 design-acceptance screens, and a usability validation protocol. These are requirements, not performed tests. |
| [Feature matrix](https://github.com/parmtech/Tong-Panel/blob/0aad68e81138a0f226e83e401f90bae4d0b6bddc/docs/FEATURE_MATRIX.md) | `94264d6ca720a5ae8e3da7a3acf890ef71e234cb` | Labels the feature list aspirational/not implemented; includes keyboard/mobile/accessibility acceptance expectations and no-role-inoperative-control requirement. |
| [P0 planning](https://github.com/parmtech/Tong-Panel/blob/0aad68e81138a0f226e83e401f90bae4d0b6bddc/docs/phases/P0-PLANNING.md) | `55fe6830ff0345b374bb08dd06f4ce95f4921bd0` | Marks the P0-02 specification and docs-entrypoint deliverable complete. This records document preparation, not usability validation or owner acceptance. |
| [Issue #4](https://github.com/parmtech/Tong-Panel/issues/4) | Live issue read 2026-11-12 01:41 UTC | Open, unassigned, zero comments; `updated_at` remains 2026-10-26T01:36:10Z. Acceptance calls for documented/reviewed usability tasks and accessible navigation flows with findings in `UPDATES.md`. No acceptance record was found in the reviewed sources. |
| [Historical Phase 1 report](https://github.com/parmtech/Tong-Panel/blob/0aad68e81138a0f226e83e401f90bae4d0b6bddc/docs/PHASE1_REPORT_2026-10-26.md) | `8502a96f3615f40c22c073c8d4cc080af215df22` | Describes an original responsive preview with search, dark/light mode, and tour. It says full responsive accessibility audit, keyboard/screenreader review, API error recovery, and frontend CI were lacking at that report snapshot. |

**Current UX evidence gap:** No representative beginner/admin task results, three-step discoverability observations, mobile/touch checks, keyboard or screenreader review, WCAG 2.2 AA audit, low-bandwidth/error-retry checks, large-list testing, or long-running-task recovery evidence was found in the sources reviewed. The source inventory is not proof that no such review happened elsewhere; issue #4 remains open and unverified here. This annex records no UX or accessibility disposition.

### Screen and feature coverage matrix for reviewer completion

The source UX specification lists ten design-acceptance screens at `docs/UX_UI_SPEC.md` §Screens (design acceptance), lines 35–45 at the inspected baseline. For each row, the reviewer should record one status—Covered, Explicitly excluded/deferred (with the later gate), or Not reviewed—then cite the exact design revision and evidence or rationale. Empty rows are unresolved; this matrix does not imply all screens are in the approved P1 scope.

| UX/UI spec screen | Coverage status (reviewer selects) | Role/journey examined | Exact design revision | Evidence or exclusion rationale / later gate | Finding / follow-up |
|---|---|---|---|---|---|
| Login/MFA/recovery and first-run tour | | | | | |
| Grid home and global search | | | | | |
| Websites, domain settings, SSL and DNS | | | | | |
| File Explorer (tree/list/grid/editor/multiselect) | | | | | |
| Application marketplace, install flow, deployment logs and rollback | | | | | |
| Node/Python/PHP runtime manager | | | | | |
| Database, email and backup workflows | | | | | |
| Admin users/resellers/packages, quotas and audit | | | | | |
| Server and service health, firewall, updates and alerts | | | | | |
| Help centre, guided walkthrough and troubleshooting | | | | | |

- Compact/expert density and theme review (source spec §Design language and §Accessibility and performance):
- Localization and RTL-ready layout review: exact languages/directions, mirrored layout behavior, overflow/focus checks, evidence, and findings (source spec §Responsive, mobile, accessibility and performance):
- Nonblocking notification behavior review: priority, dismissal, persistence, interruption, and accessible announcement (source spec §Responsive, mobile, accessibility and performance):
- Redacted diagnostic export review: exact exported fields, secret/PII redaction evidence, and reviewer disposition (source spec §Responsive, mobile, accessibility and performance):
- Save/Cancel/Undo consistency review across representative forms: unsaved-change handling, cancel semantics, undo window/scope, and evidence (source spec §Responsive, mobile, accessibility and performance):
- Reviewer confirmation that every source screen is covered, explicitly deferred to a named later gate, or marked not reviewed:

## 2. Reviewer authority and disposition

- UX decision owner / maintainer, name and role:
- UX reviewer, name, role, and authority:
- Accessibility reviewer, name, role, and authority:
- Representative beginner and administrator usability evaluation required by `UX_UI_SPEC.md` §Usability validation: record roles evaluated, method, and evidence. Issue #4 does not prescribe participant sessions; if sessions are used, record participant selection and consent. If another method is used, explain how it evaluates representative beginners and administrators:
- Session facilitator/test operator and observer identities, roles/authority, and independence/conflicts (or automated/no observer):
- Reviewer independence / conflicts:
- Review method and exact artifacts/devices examined:
- Overall UX disposition: [ ] Accepted for named design scope  [ ] Needs revision  [ ] Rejected  [ ] Deferred
- Scope and conditions covered by that disposition:
- Limitations, unresolved findings, and follow-up issue links:
- Recorded decision reference / immutable revision:

## 3. Sitemap and role × navigation/permission visibility

Record the expected and observed visibility for each role. “Visible” is a UI presentation result only; it is not proof that backend authorization enforces access.

| Role / permission profile | Home tiles, navigation and search results | Admin-only controls hidden or explained | Planned/unavailable controls clearly marked | Exact prototype/build and test account/profile | Expected vs observed result / finding |
|---|---|---|---|---|---|
| _Reviewer to determine_ | | | | | |
| _Reviewer to determine_ | | | | | |
| _Reviewer to determine_ | | | | | |

- Sitemap / navigation revision:
- Permission-dependent visibility map revision:
- Search synonyms and expected result destinations:
- Common-task navigation-step counts compared against the UX specification's three-step discovery target (forms excluded), with task, role, pass/fail, and any exception disposition:
- Excluded or deferred navigation tasks, rationale, decision owner, and later review gate:
- Any visible-but-inoperative or unexplained control, and disposition:
- Explicit statement of excluded admin/customer capabilities:

## 4. Home grid, global search, and usability task evidence

For each task, identify participant role, starting state, exact script, expected destination/actions, observed navigation steps, completion, errors, and evidence. Record usability findings rather than inferring backend functionality.

| Task ID | User role/profile | Exact task script and starting state | Expected outcome / navigation path | Observed result, steps, success/failure and notes | Artifact/reference |
|---|---|---|---|---|---|
| | | | | | |
| | | | | | |
| | | | | | |

### Proposed reviewer task scripts — preparation only, not run

The following scripts are draft review inputs derived from the exact source revisions above (`UX_UI_SPEC.md` `84c508becc9c67a7be77e716ee8c7f3287fefadb`; `FEATURE_MATRIX.md` `94264d6ca720a5ae8e3da7a3acf890ef71e234cb`). Before a session, the authorized reviewer must choose participant roles/profiles, exact design/build revision, starting state, expected behavior where the source is silent, and counting/measurement method. These scripts record no usability or accessibility result and do not prove backend behavior.

| Script ID | Proposed task and role to select | Source requirement / expected observation; source ambiguities to decide before testing | Evidence to capture |
|---|---|---|---|
| H1 — Home to Files | Select a representative beginner/customer profile. From Home, find and open File Explorer, then navigate to a named sample folder. | Record discoverability, labels and destination. Apply the source's ≤3 navigational-action target from Home, excluding form completion. Mark inactive capabilities as planned/design-only. | Role/profile, start state, exact artifact revision, ordered actions/count, completion/error, observer notes or screenshot. |
| H2 — Global search | Select a beginner profile. Search a source-listed synonym such as “new WordPress”; inspect the result and destination without submitting a deployment. | Record result relevance, destination and no-result/error behavior. Do not infer that search means backend provisioning exists; planned actions must be labeled. Apply ≤3 action target from the agreed starting point. | Query, role, results/order, path/count, empty/error state, exact artifact revision and notes. |
| H3 — Role visibility | Reviewer selects admin and non-admin profiles; compare Home tiles, navigation, search results and admin-only controls. | Record visible, hidden or explained controls separately. UI visibility does not prove server-side authorization. | Exact profiles, expected/observed visibility map, artifact revision, screenshots and findings. |
| F1 — Explorer navigation/views | Select a beginner profile and fixed sample tree/list. Navigate via tree/breadcrumbs, switch list/grid/details, then search/sort/filter. | Record orientation, location labels and selection retention. Include dataset size. No filesystem safety inference. | Data profile/size, steps, view/selection state, errors, artifact revision, notes/screenshots. |
| F2 — Invalid drag/drop and cancel | Reviewer defines a sample item, valid/invalid target, and a second drag to cancel. | UX source includes drag/drop but does not specify invalid-target or cancel semantics. Predefine expected feedback; a proposed safe expectation is clear rejection/cancellation without ambiguous move. Record source/destination state. | Exact target and expected result, observed feedback, before/after state, artifact revision. |
| F3 — Upload cancel/retry | Choose a sample file and device/network profile. Start chunked upload, cancel at a specified point, inspect displayed state, then follow the designed retry/resume path if present. | Source calls for progress/resumability but does not specify partial-data behavior on cancel. Do not assume deletion, retention or resumability; record the ambiguity and actual design-only behavior. | File size, network/device profile, cancel point, progress/state wording, retry path, artifact revision and observations. |
| F4 — Delete/restore/permanent-delete copy | Select a sample item, review trash/delete and restore steps, and inspect permanent-delete confirmation without executing a real deletion. | Record scope, confirmation, recoverability and irreversible-action distinction. No filesystem authorization or real data-integrity claim. | Selected item/scope, exact copy and sequence, restore path, artifact revision, notes/screenshots. |
| W1 — Backup/restore failure state | Select a suitable reviewer role. Walk select → preview/dry-run → verify → apply → validate; inject a scripted design failure at verification or validation and follow displayed remediation/retry. | UX source defines these steps and honest task states; determine whether recovery is implemented, design-only or unavailable. Require no premature success display. No backend restore/rollback claim. | Each step/state/message, failure point, remediation, displayed final state, exact artifact revision and notes. |
| A1/A2 — Keyboard and screen reader | Select browser/OS/assistive-technology versions. Traverse Home, navigation, search, Explorer and a dialog; inspect dynamic queued/failed status and error/remediation. | Record focus/order, labels, dialog focus/return and status/error announcements. Agree expected announcement behavior where the UX source is silent. Limited review is not WCAG 2.2 AA certification. | Keystrokes, focus/announcement sequence, AT/browser/OS versions, artifact revision, blockers and screenshots/video. |
| M1 — Mobile/touch | Select phone portrait/landscape and tablet profiles. Repeat H1/H2 and one Explorer operation; review drawer/bottom navigation, target reachability and context-menu alternatives. | Record clipping, touch access and alternatives. Apply ≤3 navigation actions only to the selected common task; state emulation vs physical-device limits. | Device or viewport/DPR/orientation, browser/OS, action sequence, target method, artifact revision, screenshots/video. |

- Home-grid discovery, labels, grouping, and empty/loading/error states reviewed:
- Global search behavior, synonyms, no-result/error states, and keyboard access reviewed:
- Task oracle/expected behavior source (UX specification or reviewer-defined), authorized setter, and date fixed before each participant session:
- First-run tour and help behavior reviewed, including dismiss/revisit:
- Beginner and administrator task coverage and participant limitations:
- Findings ranked by severity/impact, owner, follow-up issue, and retest result:

## 5. File Explorer / Finder interaction evidence

For each interaction, cite the exact wireframe/build and run the task script against that artifact. Record design usability separately from filesystem authorization, path confinement, and data safety implementation.

| Task ID | Exact script and selected role | View/interaction under test | Expected behavior | Observed behavior and evidence | Finding / follow-up |
|---|---|---|---|---|---|
| Navigate folder tree, breadcrumbs, recent/bookmarks | | | | | |
| Switch list/grid/details or preview; sort/filter/search | | | | | |
| Large file-list scale: exact item/byte counts, dataset profile, and interaction/response behavior | | | | | |
| Select/multi-select; copy/cut/paste, move, duplicate, rename | | | | | |
| Direct drag-and-drop between file-list/tree targets: valid and invalid target feedback, cancellation, and resulting state | | | | | |
| Create folder/file; upload/download; editor/preview | | | | | |
| Archive/compress/extract; trash/restore/permanent delete | | | | | |
| Permission display, storage usage, and recoverability cues | | | | | |

- Desktop wireframe/build revision and screenshots/video references:
- File-tree, list, grid, details/preview, search, filter, and selection states covered:
- Upload/download progress, cancellation, retry, interruption, and resumability states covered:
- Destructive-action scope, confirmation, undo/trash, and permanent-delete copy reviewed:
- Safety-sensitive interactions marked design-only until backend controls exist:
- No assertion of path traversal resistance, symlink protection, tenant root enforcement, or authorization based on UI review:

## 6. Wizard journeys and honest task/recovery states

List which journeys are in the reviewed design scope. For each journey, compare every source-defined step and decision point to expected and observed behavior, including explicitly planned-only steps. A designed journey does not mean its backend exists or is accepted.

| Journey | Exact script and scope | Steps/decision points reviewed | Expected final state and remediation | Observed result / evidence | Planned-only steps clearly identified? |
|---|---|---|---|---|---|
| Website/site: domain → runtime/template → optional database → import → HTTPS → review → deploy → health | | | | | |
| Node application | | | | | |
| Python application | | | | | |
| Mailbox/forwarding/relay and diagnostic warnings | | | | | |
| SSL/certificate: detect → challenge/validation → renewal → external verification | | | | | |
| Backup/restore: select → preview/dry-run → verify → apply → validate/recover | | | | | |

For every journey, add a step-by-step row for each step specified by `UX_UI_SPEC.md`; do not summarize a multi-step flow in one result. Record omitted/deferred steps with the owner's rationale and later gate.

| Journey ID | Exact source-defined step/decision point | Expected behavior/result | Observed behavior/evidence or planned-only status | Finding / owner disposition / retest |
|---|---|---|---|---|
| (add one row for every source-defined step) | | | | |

For each long-running task, record the exact UI state and user-visible wording for:

| State | Expected truthful meaning | Visible progress/log/remediation behavior | Evidence and findings |
|---|---|---|---|
| Queued | | | |
| Running | | | |
| Pending verification / verified | | | |
| Failed / interrupted / retrying | | | |
| Complete | | | |
| Recovery, rollback, or manual intervention required | | | |

- Are states tied to a real backend signal, or explicitly labeled prototype/design-only?
- Can refresh, navigation, duplicate action, or lost connection mislead the user about success or cause surprising data loss?
- Does each failure provide a safe next step, retry conditions, and any manual recovery requirement?
- Exact state diagrams, copy, and artifact revisions:

## 7. Keyboard, focus, shortcuts, dialogs, and accessibility

| Check | Browser/OS/assistive technology and version | Exact steps / control | Expected result | Observed result / evidence | Finding / retest |
|---|---|---|---|---|---|
| Keyboard-only navigation through header, sidebar/grid, search, and main content | | | | | |
| Visible focus, logical order, focus trap/return, and skip navigation | | | | | |
| Supported File Explorer shortcuts and discoverability | | | | | |
| Context menus, multi-select, drag alternatives, and touch/keyboard equivalent | | | | | |
| Dialog open/close, labels, Escape behavior, focus restore, and destructive confirmation | | | | | |
| Screen-reader names, roles, state/status announcements, and errors | | | | | |
| Contrast, zoom/reflow, reduced motion, and non-color cues | | | | | |

- Accessibility target assessed (the UX specification states a WCAG 2.2 AA target; this field records review scope, not a compliance claim):
- Screen reader(s), browser(s), operating system(s), versions, and assistive-technology settings:
- Keyboard and shortcut inventory / exceptions:
- Automated checker name/version/output, if used, and known limits:
- Manual accessibility reviewer findings and severity:
- Remaining barriers, exclusions, assigned follow-up, and retest evidence:

## 8. Mobile, viewport, and touch evidence

| Device/viewport class | Browser/OS/version | Exact journey and touch actions | Expected layout and target behavior | Observed result / evidence | Finding / retest |
|---|---|---|---|---|---|
| Phone portrait | | | | | |
| Phone landscape | | | | | |
| Tablet | | | | | |
| Desktop / resize and zoom | | | | | |

- Viewport sizes, device pixel ratio, orientation, and test method:
- Bottom/drawer navigation, touch-target spacing, long-press/context-menu alternatives:
- File Explorer dense lists, multi-select, upload, and destructive dialogs at small viewports:
- Low-bandwidth/loading/offline/error/retry behavior, tested profile and observed result, or explicit owner-approved defer/exclusion rationale and later gate:
- Large-file-list dataset sizes, interaction/response criteria, observed results, or explicit owner-approved defer/exclusion rationale and later gate:
- Screenshot/video references and limitations of emulation vs physical-device testing:

## 9. Original artwork and asset provenance

| Asset ID/path | Creator/source and creation date | License/permission evidence | Trademark/brand review | Distribution/use scope | Reviewer disposition |
|---|---|---|---|---|---|
| | | | | | |
| | | | | | |

- Confirmed original artwork/icons and retained source files:
- Third-party fonts, icons, illustrations, screenshots, logos, or copied UI patterns found:
- Exact component/source/version/hash/license/attribution or permission record for each non-original asset:
- cPanel or other third-party brand/trademark references reviewed and any required changes:
- Missing provenance, qualified licence/trademark reviewer, date, and follow-up:
- UX/artwork review is not legal clearance; record any counsel referral separately:

## 10. Findings, limitations, and P0-02 UX portion / issue #4 disposition

This section covers only the UX portion of P0-02 linked to issue #4. P0-PLANNING also includes installer specifications and the documentation entrypoint under P0-02; their evidence and disposition must be recorded separately.

| Finding ID / severity | Evidence and affected role/journey | Required change or accepted limitation | Accountable follow-up / issue | Retest condition and result |
|---|---|---|---|---|
| UX-EVID-20261111-01 / evidence gap | UX event `20261111T040519Z-parallel-ux-1e317387-1db3-43e6-a721-afb8fa579709` reports 13 passing fake-DOM interaction tests for prototype flows. Reported local artifact files are outside this checkout and were not independently verified. Real browser/headless, assistive-technology/accessibility, contrast/zoom, and participant usability checks are reported NOT RUN. | No acceptance or limitation disposition recorded. Obtain and review exact artifacts; record and perform the authorized review method for representative usability tasks and accessible navigation flows before claiming the criterion is met. Issue #4 does not itself require participant sessions; the unrun-session report is a method/evidence limitation unless the authorized review plan makes sessions necessary. | Issue #4; authorized UX/accessibility reviewer and maintainer. No assignment recorded. | Pending; no retest or issue #4 acceptance demonstrated. |
| | | | | |
| | | | | |

- UX reviewer disposition and date:
- Accessibility reviewer disposition and date:
- Maintainer/owner P0-02 UX portion / issue #4 disposition: [ ] Met  [ ] Not met  [ ] Deferred
- Evidence basis for each applicable P0-02 UX criterion:
- Open findings and blockers:
- UPDATES.md/checkpoint reference after authorized reconciliation:
- Exact final record revision / immutable commit:

## Boundary and limits

This is UX/design usability evidence only. It does not prove backend authorization, tenant isolation, filesystem safety, secure authentication, accessibility conformance beyond the tested scope, implementation readiness, a security review, ADR acceptance, P0 completion, P1 exit, P2 entry, or permission to merge, deploy, or mutate a host. Design artifacts must label unavailable backend functions as planned or design-only; visible controls do not establish enforcement. Every decision remains blank until the authorized owner and reviewers record it against exact revisions and evidence.
