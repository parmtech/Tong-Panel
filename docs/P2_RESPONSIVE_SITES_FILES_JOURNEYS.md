# P2 issue #155 — Responsive Sites and Files journey acceptance plan

**State:** design proposal only, based on PR #171 source `dff0a230f3c7840df118efec63fa00011435e9a4`. This document adds acceptance criteria; it does not implement or verify the journeys, approve a role policy, or close P2 issue #155.

**Issue:** [#155 — P2 acceptance: define mobile/tablet site and Files journeys](https://github.com/parmtech/Tong-Panel/issues/155).

## Purpose and boundaries

Define observable novice and administrator journeys for site/domain/HTTPS setup and tenant Files use across phone, tablet, and desktop layouts. Automated browser checks and recorded human usability/accessibility review are both required. This plan adds no role, permission, production route, host action, supported-browser promise, or service contract.

The P0/P1 entry gates are dependencies, not steps this plan can waive. Before exercising authenticated host-facing operations, owners must approve the architecture and threat model; security must review identity/session and tenant boundaries; and P1 must establish production authentication, MFA recovery, CSRF/session policy, the typed tenant-bound host interface, audit and recovery behavior. Domain/HTTPS work additionally requires reviewed DNS evidence, staging-CA/key handling, and known-good configuration/certificate rollback. Files mutations additionally require the approved tenant-root and quota/publication contract. Until those interfaces exist, use synthetic browser fixtures and label all incomplete actions planned or disabled. A mock response is never end-to-end acceptance evidence.

Do not use production tenants, customer data, public DNS zones, public certificate authorities, a live server, or privileged host operations. Later integrated runs must use approved synthetic identities and disposable guests only.

## Baseline evidence and uncovered work

The inventory below was checked against `tools/ui-browser/jobs.spec.mjs`, `web/test/`, and the UI at the baseline SHA above. It describes existing component evidence only.

| Existing evidence | Current coverage | Gap for issue #155 |
|---|---|---|
| Home launcher browser test | Grid widths 1440, 1024, 768, 390, and 320 CSS px; checks columns and horizontal overflow. | It does not complete a Sites or Files task. |
| Jobs mobile browser tests | At 390×844, checks job/details overflow and scoped axe WCAG 2.2 A/AA rules. Keyboard tests cover job details. | Does not cover Sites or Files; no human screen-reader review. |
| Sites and domain tests | Real loopback TLS/SQLite fixture tests catalog selection, site details, prefix pagination, tenant/session changes, stale responses, and authorization denials. The certificate snapshot and Nginx preview have read-only browser cases. | Current tests are not run across the responsive viewport matrix and do not complete site creation, DNS proof, certificate issuance, activation, or rollback. |
| Files tests | Browser cases cover tenant-scoped Linux handler pages/previews, prefix filtering, 401/403 clearing, pagination, stale folder actions, and retry/failure behavior. Other cases use intercepted responses. | No phone/tablet/desktop Files matrix, touch review, screen-reader review, or full upload/edit/move/delete/archive journey. |
| Accessibility checks | Some Chromium/axe checks cover Jobs and the read-only certificate snapshot. | No recorded human accessibility/usability findings for Sites or Files; no review evidence for screen-reader use, contrast in all themes, forced-colors, reduced motion, or touch-target operation. |

Playwright currently uses Chromium and a disposable loopback TLS fixture. These tests do not establish behavior on physical phones/tablets, other browser engines, production authentication, or a supported hosting installation.

## Personas and journeys

Persona names are test roles, not a proposed authorization model. A persona may see or act on only resources authorized by the approved server-side policy. “Admin” does not imply blanket cross-tenant access.

### N1 — Novice site owner: find a site, establish a domain, understand HTTPS

1. Start from Home and reach Sites or search for the site within three navigation steps, excluding form completion.
2. Select one of the persona’s own synthetic sites and understand its primary domain, ownership state, and whether hosting is actually provisioned. Catalog metadata must not imply a live site.
3. Follow the approved domain-proof instructions. See distinct pending, verified, stale, and failed states with a clear next safe step.
4. Where an approved HTTPS workflow exists, request or review setup and follow its queued, challenge, certificate-validation, apply, and health-check states. Show whether the last known-good HTTPS configuration remains active after a failure.
5. Recover from a DNS or staging-CA failure using the displayed guidance. A retry must not duplicate an operation or claim success before the server confirms it.

Until those services and permissions exist, the journey ends at a truthful read-only/planned state. Do not present the current domain snapshot or Nginx preview as DNS verification, certificate issuance, or enabled HTTPS.

### A1 — Administrator/operator persona: inspect state and recover safely

1. Find an authorized synthetic site, filter or page the catalog, and inspect its domain and service status.
2. Identify a stale ownership proof, failed staging-CA request, or failed config/health check from the state text and operation history.
3. Determine whether the operation is still running, safely retryable, rolled back, or requires supervised recovery. Do not infer the outcome from a timeout or a visible file alone.
4. When apply/rollback interfaces become approved and implemented, inject failure at the documented transitions. Verify that the previous known-good configuration and certificate continue serving, or that the UI clearly reports an explicit recoverable/manual-review state.
5. Verify that attempts to inspect or operate on another tenant’s site are denied without revealing protected details or leaving stale site/domain data visible.

This persona must use the explicit support/operator permissions approved for the release. The UI cannot invent a support override or use a hidden tenant selector.

### N1/A1 — Tenant Files: browse and recover without crossing tenant roots

1. Enter Files from Home, identify the current tenant/site root, and use breadcrumbs to navigate folders.
2. Use the available prefix filter, Next/Previous paging, and refresh while retaining a clear location and page status.
3. Open an available regular-file preview or download. Unsupported, binary, or oversized preview content must produce a bounded explanation rather than stale or partial text.
4. Exercise empty folders, no search matches, a long list, an expired session, a permission denial, and a transient service error. Provide a safe retry for read operations; clear previously displayed tenant data on authentication or authorization loss.
5. Hold a response from tenant A, switch to tenant B or revoke the session, then release it. A’s rows, preview, paths, page cursor, and status must not reappear in B’s view.
6. Request A’s path/site ID with B’s authenticated synthetic session, including altered URL, cursor, and stale UI actions. The server must deny according to the approved object-concealment contract; neither tenant’s data may be partially changed.

The current UI is a development read-only view. Upload, edit, create, move, delete, archive, trash, and permission changes remain planned until their backend, authorization, quota, audit, and recovery paths are implemented and approved. Do not make those controls appear operational for this acceptance run.

## Responsive viewport and interaction matrix

Run each named journey at each layout class; use synthetic data and the same test identities. Record CSS viewport dimensions, orientation, browser/OS version, zoom, and whether the test used emulation or a physical device.

| Class | Required viewport examples | Required observations |
|---|---|---|
| Phone | 320×640 and 390×844 CSS px, portrait; repeat one supported width in landscape. | Primary task remains discoverable; navigation, forms, status, lists, and recovery guidance fit without page-level horizontal scrolling; no action is available only by hover or right-click. |
| Tablet | 768×1024 portrait and 1024×768 landscape. | Sites details and Files navigation remain usable without clipped controls, misleading nested scroll, or loss of context. |
| Desktop | 1280×800 and 1440×900. | Dense site/domain and Files information remains scannable; keyboard and pointer workflows reach equivalent actions. |

Also test text enlargement/reflow at 200% and a narrow 320 CSS-pixel content width. No required status or action may be clipped, hidden behind fixed navigation, or available only after horizontal page scrolling. If an internal region must scroll, give it a clear accessible name and preserve keyboard access to its contents. Verify portrait/landscape changes do not reset a task or apply an unintended action.

## State, authorization, and recovery assertions

Use controlled delays, server fixtures, and explicit failure injection. Do not infer successful host work from mocked UI state.

| Area | Required states and transitions | Pass condition |
|---|---|---|
| Sites/domains | Initial, loading, empty, populated, pending/unverified, verified, stale proof, validation error, 401/403/404, conflict, timeout/503, long-running, completion, failure, rollback, unknown outcome. | State text explains what is known and what safe action is available. Stale responses cannot overwrite the newly selected site or session. Unknown or partially applied mutation is not shown as success and is not blindly retried. |
| HTTPS/config | Challenge pending, staging-CA rejection/timeout, certificate mismatch, apply/reload failure, health-check failure, successful verified completion, prior-known-good recovery. | No public CA or real hostname is used. Failure preserves and reports last known-good state or an explicit recovery requirement. Success is shown only after the approved server-side operation and verification confirm it. |
| Files | Loading, empty, no matches, page transition, refresh, preview/download loading, denial, revoked session, wrong tenant, unavailable/binary/oversized content, network error, retry, held/stale response. | Auth loss clears prior rows, previews, path and cursor. Transient same-scope read failures show an explicit retry and identify if retained content may be stale. No old request can update a new folder, filter, site, or tenant view. |
| Retry/rollback | Duplicate click, repeated request, cancellation, timeout after possible mutation, failure before/after publication or activation. | Read retries are bounded and do not expose old-scope data. Mutating retries require an approved idempotency/operation-state contract. Rollback is independently verified; ambiguous outcomes remain pending/manual review rather than being mislabeled as rolled back. |

For any response that intentionally conceals cross-tenant object existence, assert the same external result for absent and foreign-tenant IDs. Check that denial does not partially change site configuration, certificate state, Files data, URLs, browser history, or visible job/status text.

## Keyboard, screen reader, touch, motion, and visual acceptance

- **Keyboard:** complete both journeys with keyboard only. Check logical Tab/Shift+Tab order, visible focus, Enter/Space activation, escape/return behavior for dialogs and drawers, focus after pagination and errors, and absence of traps. Every essential touch action needs an equivalent keyboard path.
- **Screen reader:** conduct and record a human review on one declared desktop screen-reader/browser pair and one declared phone pair. Record those combinations as the tested scope; do not imply broader support. Verify landmarks, headings, labels, control names/roles/states, selected tenant and folder, pagination position, verification/HTTPS states, loading completion, denial, and recovery announcements. Dynamic updates must be announced without repeated or noisy live-region output. Record actual AT/OS/browser versions and the reviewer’s findings.
- **Touch:** test the primary site, domain, HTTPS, navigation, Files row, breadcrumb, paging, filter, preview, close, and recovery actions on a physical touch device when available; otherwise label the test emulated. Meet WCAG 2.2 SC 2.5.8 target size and spacing. As a stronger project usability target, aim for 44×44 CSS-pixel targets for primary touch controls; identify and justify any dense secondary controls that cannot meet that target.
- **Contrast and non-color cues:** test light and dark themes and forced-colors/high-contrast mode. Measure normal text at 4.5:1, large text at 3:1, and visual component boundaries/state indicators at 3:1 where WCAG 2.2 AA requires it. Pending, verified, failed, selected, focused, and disabled states need text, shape, or another non-color cue. Check focus rings and error text separately; axe is supplemental, not a substitute for contrast measurement or human review.
- **Reduced motion:** with `prefers-reduced-motion: reduce`, confirm nonessential transitions/animation are removed or reduced and no status, focus, loading, or success information depends on movement. Verify the preference does not suppress task completion announcements.
- **Zoom and magnification:** verify text enlargement and browser zoom do not obscure navigation, confirmation, denial, retry, or rollback guidance. Record any intentional internal scroll region and why it is needed.

Standards references: [WCAG 2.2](https://www.w3.org/TR/WCAG22/), [SC 2.5.8 Target Size (Minimum)](https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum), [SC 1.4.3 Contrast (Minimum)](https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum), and [SC 1.4.11 Non-text Contrast](https://www.w3.org/WAI/WCAG22/Understanding/non-text-contrast). The 44×44 project target above is a usability goal, not the AA minimum in SC 2.5.8.

## Test method and evidence record

1. **Design/contract phase:** keep inaccessible or unimplemented operations disabled/planned. Use deterministic, synthetic API fixtures to test layout, keyboard, accessibility tree, loading, error, denial, stale-response and retry presentation. Mark all mocked behavior as component evidence only.
2. **Authorized integration phase:** only after P0/P1 decisions and reviewed auth/session policy, use disposable synthetic tenants A and B and actual handlers. Test current session/membership checks, denial, revocation and stale UI behavior. No real identities, host mutations, or service credentials.
3. **P2 service phase:** only after the typed host operations and DNS/ACME/config/rollback interfaces are approved, use a fresh disposable guest, private DNS fixtures, staging CA, and recorded image digest. Inject each documented failure and compare previous/current configuration, certificate, serving state, and tenant data.
4. **Human phase:** have at least one representative novice and one administrator/operator complete the journeys without facilitator rescue. Record task success/failure, navigation steps, time, help requested, wrong selections, error comprehension, and whether recovery was safe. Obtain participant consent; use synthetic fixture data.

For every run, capture:

- source commit and whether the browser used mocks, actual handlers, a disposable VM, or a physical device;
- role/persona, tenant fixture IDs, browser/OS/AT versions, viewport, orientation, zoom, theme, contrast mode, and motion preference;
- task steps and expected/observed result, test command, pass/fail count, sanitized logs/screenshots, and any skipped prerequisite;
- human usability/accessibility reviewer, findings with severity and affected criterion, linked fix, and exact retest commit/result.

Do not capture real hostnames, secrets, customer files, tokens, or unrestricted environment dumps. A screenshot, axe report, component test, or passing PR workflow alone does not satisfy issue #155.

## Exit criteria for issue #155

- [ ] Named novice and administrator/operator site-domain-HTTPS journeys and tenant Files journeys complete at phone, tablet, and desktop sizes without facilitator rescue, when all required layers are implemented and approved.
- [ ] Each layout/state matrix passes, including absent/expired session, foreign-tenant ID/path, delayed stale response, network failure, retry, long operation, and rollback/unknown outcome.
- [ ] No cross-tenant site, domain, file, path, cursor, preview, status, or stale response is disclosed; all denials follow the approved public response policy and have no partial effect.
- [ ] Keyboard-only and human screen-reader reviews cover the actual flows; focus and announcements remain usable through loading, page changes, denial, retry, and recovery.
- [ ] Contrast, forced-colors, reduced-motion, touch target/spacing, zoom, reflow, and phone/tablet orientation checks have recorded results and no unresolved critical/high accessibility blocker.
- [ ] All defects have linked fixes and exact-commit retests; automated reports are supplemented by human findings and a recorded accessibility/usability review.
- [ ] Incomplete APIs, privileged operations, certificates, or Files mutations remain disabled/planned and are not represented as accepted end-to-end work.
- [ ] P0/P1 dependencies have owner/security decisions and evidence; this plan itself does not approve or satisfy those gates.

Until every item has linked evidence, issue #155 and the P2 exit gate remain open. This plan is a proposed acceptance contract, not a test report or accessibility conformance claim.

## Traceability

- [UX/UI specification](UX_UI_SPEC.md), responsive experience, Files Explorer, accessibility, and three-step discoverability.
- [Test strategy](TEST_STRATEGY.md), browser matrix, disposable environments, human role acceptance, and evidence record.
- [P2 two-tenant acceptance plan](P2_TWO_TENANT_END_TO_END_ACCEPTANCE.md), especially the Files/quota, domain/certificate, UI/accessibility, denial matrix, and exit checklist sections.
- [P2 acceptance traceability matrix](P2_ACCEPTANCE_TRACEABILITY.md), issue #155 and remaining integrated evidence.
- [P2 issue #155](https://github.com/parmtech/Tong-Panel/issues/155), which remains open until automation and human evidence are attached.
