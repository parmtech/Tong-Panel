# Tong Panel — Master Product Blueprint
**Project:** Tong Panel (TongPanel) | **Publisher:** ParmTech (https://parmtech.au)  
**Project type:** Free/open-source Linux hosting control panel and cPanel/WHM alternative  
**License:** GNU AGPL-3.0 for original Tong Panel code; all third-party dependencies retain their own licences.  
**Status:** PRODUCT BLUEPRINT + UNMERGED DEV PROTOTYPES — no production-ready or supported deployable Tong Panel exists. For the actual code/CI state read [AI agent A–Z handoff](AI_AGENT_START_HERE.md).  
**Original blueprint:** 2026-10-26 | **Current engineering status linked:** 2026-11-17 | **Blueprint revision:** 0.1 (targets preserved)

## Current implementation reality and next agent entrypoint (PR #171 head 203179; status refreshed 2026-11-17)

**Published PR/ref:** PR #171 is open/draft/unmerged at `203179dccdbcb9c07089c82f5644ba6b049c8093` (tree `b83dfc22980abc2bf3cdfc3579faf2e59404972b`) against main `0aad68e81138a0f226e83e401f90bae4d0b6bddc`; `integration/pr-reconciliation-20261113` matches. All 17 exact-SHA Actions runs and all 28 checks passed: 13 PR-event runs/24 checks and 4 push runs/4 checks. Go, Current API contract, P1 gate inventory, P1 developer smoke, unsigned developer package, browser, and P2 workflows passed. P2 run #44067810602 passed Debian job #133985940788 and Ubuntu job #133985941365 through restore CLI, guest smoke, and job completion. No raw VM or serial logs were read. This establishes CI results for this source only.

**Current worktree:** The root review is in progress. The uncommitted `internal/api/files_delete_quarantine_cancellation_test.go` is a test-only Linux regression and is uncompiled/unrun because Go, gofmt, and WSL are unavailable locally; it needs exact-SHA Linux verification. The uncommitted test-only `scripts/test_p4_compatibility_matrix_audit.py` follow-up passed 30/30 locally, outside the hosted SHA. The corrected `tools/ui-browser/jobs.spec.mjs` regression keeps preview content as the intermediate Tab stop before pagination in a phone-viewport real Files handler flow; Node syntax and diff-check passed; the browser run is unrun because `TONG_UI_FIXTURE_BINARY` and Go are unavailable. None of these three follow-ups is covered by the 203179 results.

**Tracker and gates:** Issue #172 remains open at 0/12 DoD; all nine approvals and `product_support_approval` are false. P0-06 and owner/security, production authentication/key custody, privileged-host, signed installer/recovery, integrated two-tenant restore, accessibility, merge, deployment, and release gates remain open. CI and the P1 gate inventory do not grant approval or phase acceptance. No production-ready or supported deployable Tong Panel exists. Read the [A?Z handoff](AI_AGENT_START_HERE.md) for resume order.

### Historical exact-head snapshot — e9a6c70 (superseded by f220226; CI observed 2026-11-17T05:51:37Z)

**Published PR/ref:** PR #171 remains open/draft/unmerged at `e9a6c700022e4d2834074980e9e82bac40a1b4d6` (tree `12891d236860bf2479ac37a84ca4166dbc2bfb3a`, parent `34db65c97131d16a1cda3fe5e5a7a8ddee04a5ee`) against main `0aad68e81138a0f226e83e401f90bae4d0b6bddc`; `integration/pr-reconciliation-20261113` matches.

**Exact-head CI:** all 13 Actions runs returned for e9a6 (all `pull_request` events) and all 24 check-runs completed successfully, with no failure or pending result. This result applies only to e9a6. P2 run #44024677223 passed: Debian #133842055840 completed guest smoke at 05:45:29Z and the post setup-go, post checkout, and Complete steps by 05:45:30Z; Ubuntu #133842055630 completed guest smoke, post setup-go, post checkout, and Complete at 05:50:35Z, with job success at 05:50:38Z. No raw guest or serial logs were read. The legacy aggregate commit-status API remains `pending` with zero contexts and is separate from named checks.

**Historical e9a6-era local follow-up (later published in f220 and included in c86):** e9a6 included the test-only reciprocal tenant-binding denial in `scripts/test_backup_restore_site_lab_cli.py`; its then-local continuation called `inspect_empty_inventory(directory)` after authenticated traversal/symlink archive denials. It asserted that the post-failure inventory observed zero archive stages while retaining no-destination and unchanged-sentinel assertions. This continuation was not part of e9a6, but was later published in f220 and is in c86. The c86 exact-head CI is green, though its metadata does not isolate this individual assertion. The check does not inject a cleanup failure or test ambiguous publication and changes no production behavior or policy.

**Tracker and gates:** Issue #172 remains open at 0/12 DoD checked; its body states all nine approval fields false and `product_support_approval:false`. The PR and issue openings were refreshed to e9a6 after its CI result; their GitHub `updated_at` values are 05:54:11Z and 05:54:16Z, respectively. P0-06, owner/security decisions, production authentication/key custody, privileged-host actions, signed installer/recovery, integrated two-tenant restore, accessibility, merge, deployment, and release gates remain open. CI is not approval or phase acceptance; no production-ready or supported deployable Tong Panel exists. Read the [A–Z handoff](AI_AGENT_START_HERE.md) for the current resume order.

### Historical exact-head snapshot — 7f3c1c6 (superseded by 34db and e9a6; metadata read 2026-11-17T03:06:06Z)

At PR head `7f3c1c6a9fbeebfc1f1484b24ce447ef71308d13`, 9/13 PR-event Actions runs and 17/23 PR checks succeeded; across PR and push events, 11/16 Actions runs and 19/26 check-runs succeeded, with no pending results. P2 run #44004387562 passed on Debian #133776021340 and Ubuntu #133776021100. Failure annotations then included the P2 synthetic digest lookup, P3 model pointer mismatch, and P4 formatting; Go and scanner annotations pointed to the P3 compile error. At that snapshot, commits 3ff61ebe489342c418b2590670123a6fd4f55c44 and 16038e2818f7eea3b177d954a4d02423420f5976 were local follow-ups; they were subsequently included in the source history published through 34db. These 7f results are historical and do not describe e9a6.

### Historical verified CI snapshot — f804 (superseded by 10f666a)

The product scope below is the target, not a completed-feature list. PR #171 was open/draft/unmerged at remote head f804127fdef6b46fec16c27beeb357398e066cba against main 0aad68e81138a0f226e83e401f90bae4d0b6bddc, per the PR/ref read at 2026-11-16 20:29:23 UTC. Exact-head CI, read at 2026-11-16 20:26:10 UTC, passed all 13 Actions runs and 22 check runs. P2 run #43945726657 passed on Debian job #133574748060 and Ubuntu job #133574748605; both passed step 18, post setup/checkout, and Complete.

The exact-head run IDs are Go #43945726529, govulncheck #43945726676, gosec #43945726585, unsigned package #43945727158, P1 developer smoke #43945726443, isolated prototypes #43945726402, UI #43945726533, SQLite #43945726799, offline models #43945726483, blocker inventory #43945726665, API contract #43945726530, Jobs browser #43945726491, and P2 clean VM #43945726657. No raw guest or serial logs were read. The earlier db97 Debian failure cause remains unknown; the 5257 and f804 successes do not explain it.

P4 schema-v2 source commit a5e1787832cf74754e45750f47b512ac9e81a721 is included in f804127. The root verified 27/27 focused tests, the offline workflow path coverage is verified, and no filter change is pending. This remains structural inventory only: product_support_approval:false and no P4 acceptance or product support claim.

Local candidate is branch head `a2988b377d9a5c9656be310cab50fb58b5bbaed7` (tree `819becb152a32d620a0a4616c97aaabe145712dd`), based on last published head `f804127fdef6b46fec16c27beeb357398e066cba`; it is unpushed. Unpushed commits: `f3d02ac841655a5320f5dd3b602310ef3cc5757f` adds focused synthetic UI tests (2/2 passed locally: Sites details at 390x844 with pending/catalog-only metadata and axe scoped to #site-details; empty Files-folder keyboard round-trip at 320x640); `90becaf720556fe28cffadf18813dfc2d7e4682a` adds the synthetic P2 backup-retry model/workflow (6/6 root-verified local tests); `a2988b377d9a5c9656be310cab50fb58b5bbaed7` adds a test-only SQLite job-kind allowlist regression, reviewed but not run locally because Go/gofmt are unavailable. No hosted CI covers any of these commits. Code and workflow changes are committed; only the refreshed handoff documents are modified and uncommitted. After a fresh pre-push PR/ref/issue read and normal PR sync, expect 13 workflows and 23 checks, including both P2 guests and the offline-model job; this is an expectation, not a CI result.

The PR/ref snapshot is separate from tracker state. Issue #172 was read at 2026-11-16 20:29:23 UTC and remains open, updated_at 2026-11-16T16:12:59Z, with 0/12 DoD items checked and all nine approvals false. P0-06, owner/security decisions, production authentication/key custody, privileged host actions, signed installation/recovery, real two-tenant restore, accessibility and release approval remain open.

**Historical next action in the pre-2f 022 snapshot (superseded by pushed 2f):** Finish reviewing the 14-file handoff correction, re-read PR #171/ref and issue #172 immediately before publication, then publish the local branch by normal fast-forward and inspect exact-head CI on the new docs SHA. The 022 run was terminal and green at that snapshot; a later docs SHA still required its own result. Preserve issue #172 at 0/12 with nine approvals false, P0-06 unchecked, product_support_approval:false, and every owner/security/production/restore/accessibility/signing/merge/release gate open. No production-ready or supported deployable Tong Panel exists.

## Historical implementation checkpoint — 2054/3490235 (superseded 17 November 2026)

The **product scope below is the target**, not a completed-feature list. Published PR #171 snapshot read 2026-11-16 12:36 UTC: open/draft/unmerged at head 2054a425108abd4db8df6e84b29c421dafe17dd0 against unchanged main 0aad68e81138a0f226e83e401f90bae4d0b6bddc; 16/17 workflows and 24/25 checks succeeded, with the one P2 failure and zero pending. Attempt 1 ran both Debian and Ubuntu and step 17 passed on both. Attempt 2 reran only failed Debian: step 17 passed and step 18 failed; Ubuntu's attempt-1 success was retained, not rerun. Root cause remains unknown and umask unconfirmed. Candidate 34902351b90c0bb7fe8f67d6681a15031f24e5d1 (tree 61335cc7c6078aaa65daf6f6dab50cf9cd8724e3) has 11 source commits after published source 61fdb; with prior docs-only commit 02a1cfd and two P0 crosswalk docs, it is 14 commits after 2054. Two SQLite test commits add TestSQLiteCompletionCommitProcessCrashBeforeAck (execution unverified). A draft P0 threat-to-test crosswalk is also present (P0-06 unchecked, all 43 references resolve, no tests run for the crosswalk). Earlier local evidence: Python 19/19, bash -n, 11 cleanup scenarios and diff check passed. Go/gofmt/WSL/govulncheck are unavailable; OpenRootAt, ACME and SQLite Go tests did not run. No exact-head hosted CI has run on the candidate. Read [AI agent A–Z handoff](AI_AGENT_START_HERE.md) for the exact candidate and next action.

Source candidate `61fdb4b8faf520ec277cb5aeef82d4ac3510fe88` adds lease-expiry/restart coverage, forced-colors and reduced-motion journeys, an amd64-only package guard, phone Sites/Files targets, theme-aware focus colors with 15 focus selectors sampled in both themes (30 selector/theme pairs), forced-colors navigation focus, bounded allowlisted PHP-FPM diagnostics, atomic unsigned-package publication under a trusted output-parent assumption without crash-durability guarantees, test-only OpenRootAt owner/private-mode tests, and newline-preserving package path capture with an exact-parent staging regression. Local responsive tests passed 17/17 and PHP-FPM diagnostic unit tests passed 16/16; the publication failure/path shim suite passed on Git Bash and its regression failed against the old builder with exit 95. The Linux-only Go tests were not run locally because Go/WSL are unavailable; hosted Go race/vet and security checks passed at exact PR head 2054. Step 17 passed on both guest jobs in attempt 1 and on Debian's failed-job rerun in attempt 2; Ubuntu's attempt-1 success was retained, not rerun. Debian step 18 guest smoke failed with generic exit 1 twice, while Ubuntu passed on attempt 1; its successful result was retained, not rerun on attempt 2. The cause remains unknown; a static PID-file/umask hypothesis is unconfirmed. The FPM identity/diagnostic/stage-classification follow-up is in unpublished 3490235 but does not establish root cause. Guest PHP-FPM values are unverified. True Linux symlink/mode checks and successful Debian guest lifecycle remain incomplete. The focus tests are scoped synthetic evidence, not overall WCAG conformance. Master tracker #172 coordinates P0-P6; all 12 DoD items remain unchecked and all nine approval fields remain false. The branch has unsigned developer-only packages and isolated Go/VM/ACME/backup/migration models, not a supported hosting panel. No ADR/security signoff, public login, root host mutation, production signing/installer or supported release is approved. Keep vision separate from verified implementation.

## 1. Mission and non-negotiables

Create an original, beginner-friendly but professional control panel with familiar, discoverable UX for shared hosting, VPS and dedicated Linux servers. The UI uses a home-screen grid, searchable apps, Windows/macOS-style file management, contextual guidance and first-run tours. It is NOT a visual or source-code clone of proprietary cPanel or of any other product.

Design priorities in order: **safety of customer data and server**, correct workflows, clear feedback, easy installation, usability/accessibility, broad capability, predictable upgrades, speed, extensibility. Feature richness is an end-state goal, not permission to ship insecure or unfinished subsystems.

- Free to use, self-host and modify. Full AGPL-compliant source distribution and appropriate attribution.
- One documented installer entrypoint on supported *fresh* Linux systems, with interactive and unattended modes. Transactional preflight; never claim support for any arbitrary distro.
- Server admin, reseller and hosting-customer roles with true account isolation.
- Reliable traditional PHP hosting **and** modern Node.js, Python, Ruby, static sites, containers and Git-based deployments.
- Hostnames, DNS, mail, databases, SSL/TLS, backups, migration, monitoring, security, automation, APIs and app marketplace.
- Single-server first, multi-server orchestration later via secure workers/agents.
- Optional services remain optional: mail, authoritative DNS and Docker must not be required to host a basic website.
- No hidden subscriptions/paid-feature gates for first-party core functionality. Third-party DNS, cloud storage, VPS, registrar, CA, mail relays etc. can charge fees.
- First-release direction: integrated third-party software for panel features should be free and open source, with no mandatory paid or paid-only components. Use suitable open-source components and original open-source Tong Panel implementation as needed, subject to license, compatibility and security review. Do not claim a suitable component exists until evaluated. Softaculous is not selected. Infrastructure and optional external providers may charge independently.

## 2. Audiences and experience

**Personas:** new site owner; developer; small-business admin; hosting reseller; infrastructure operator.  
**Entry points:** (a) guided Site Creation wizard, (b) app-icon Home, (c) searchable command palette, (d) admin/server centre, (e) expert CLI/API.
**Familiar design conventions:** mobile/tablet icon-grid home; file explorer left tree and breadcrumbs; clear Save/Cancel, undo where safe; item context menus; drag-drop uploads; multi-select; familiar notification centre and system settings. Responsive desktop/tablet/mobile; keyboard-accessible, screen-reader labels, high contrast, reduced motion, language packs.

**Home areas:** Websites, Domains, Apps, Files, Databases, Emails, DNS, SSL, Backups, Security, Server, Monitoring, Users, Packages, Marketplace, Settings, Help. Cards hide or explain unavailable capabilities based on role/server services; never create dead links.
**Cross-cutting:** global search; progress/log output for jobs; rollback or compensating action if possible; plain-language errors plus expert details; context tips, guided interactive tours, docs, diagnostics.

## 3. Complete capability catalogue

### Hosting/domains/web
- Domains, subdomains, aliases, redirects, parked domains; DNS ownership checks and IP/vhost validation.
- Auto-created virtual hosts, document roots, custom error pages, URL rewrites, reverse proxies, HTTP/2/3 where supported, gzip/Brotli, cache policies and security headers.
- Nginx-first supported profile with PHP-FPM; optional Apache interoperability adapter in subsequent phases. Do not install conflicting active web-stack configurations.
- PHP version selector per app/site; per-site process pool, resource isolation, extension management, safe php.ini policies, logs and cron.
- SSH/SFTP access, tightly scoped file permissions, quotas; no unrestricted web-root ownership by panel service.
- Opt-in browser terminal for authorized accounts in their own isolated environment, with session revocation, bounded resources and audit. It is a planned first-release feature; no unrestricted root shell or cross-tenant access. See the [terminal implementation proposal](P1_BROWSER_TERMINAL_PROPOSAL.md) for unresolved scope and acceptance tests.

### Application development/deployment
- PHP (WordPress, Laravel, 84PAD, etc.), HTML/static, Node.js (Express, Next.js build/runtime), Python (Django, Flask, FastAPI), Ruby, Go binary and generic container/Compose profiles.
- One-click templates with signed/pinned manifests, environment variables, secrets, dependency and runtime versions, health checks, ports, build logs, git webhooks, deploy history/rollback.
- GitHub/GitLab/Gitea repositories via scoped OAuth/app/SSH deploy keys; push-to-deploy optional. Build worker isolation and explicit network, memory and CPU limits.
- Process supervisor / systemd for native non-container apps; isolated rootless-container profiles for container apps. Shared ports and reverse-proxy routes assigned centrally; no exposed unintended container sockets.
- Install/update/backup WordPress and approved apps; marketplace extensible for community contributions after security review.
- Job queue; app states and deployment state machine; safe delete requiring confirmation and backup/export.

### Database/storage
- MariaDB/MySQL, PostgreSQL first; Redis optional; later MongoDB etc. via isolated supported profiles.
- Provision, rotate credentials, grant least privilege, import/export, query UI with guardrails, connection troubleshooting, per-user quotas where feasible, backups/PITR planning, diagnostics.
- File explorer: tree/grid/list, breadcrumbs, search, sort, upload with resumable chunks, downloads, archive, extract with path validation, editor with versioning, preview, drag/drop, copy/move, permissions, trash/recovery, SFTP browser, activity log. Strict per-tenant jailed path access; never execute uploaded content in panel context.
- External storage interfaces: S3-compatible, SFTP and rclone adapters, opt-in with scoped encrypted credentials.

### Email
- Optional first-class self-hosted stack: SMTP submission/delivery (Postfix), IMAP/POP (Dovecot), webmail adapter (Roundcube/SnappyMail subject to license/version review), queues, limits, aliases, forwarders, auto-responders, filters, spam/AV.
- SPF, DKIM, DMARC, PTR guidance, TLS, DNS checks, mailbox migration, provider relay, catch-all warnings, delivery/reputation dashboards, quarantine and logs.
- Clarify outbound mail needs port/network permission and reputation; no promise of deliverability on every VPS.

### DNS / SSL / certificates
- Managed BIND/PowerDNS optional, external provider API adapters, DNSSEC, record editor, templates, zone import/export, validation and propagation view.
- ACME certificates, wildcard via DNS-01, HTTP-01/TLS-ALPN-01 when feasible, automatic renewal with observable lifecycle/retry/alerts; cPanel-equivalent HTTPS redirect toggles with external verification; multi-domain/SAN.
- Certificates for web, mail and control panel separately; avoid duplicative issuing where valid existing certificates exist; safe renewal and post-issuance chain checks.

### User/hosting business
- Roles: platform owner, server admin, reseller, customer, team member, read-only auditor. Strong RBAC with object-level authorization on every service.
- Hosting packages/feature flags/quotas, usage/accounting, domain ownership, suspension/unsuspension, account transfer, notification centre.
- Tenant isolation, per-tenant logs, account backups, secure impersonation requiring auditable grant and warnings.
- Billing integration later through external systems (e.g. interoperable APIs/webhooks) rather than building a payment processor into root-privileged daemon.

### Infra/reliability
- Service manager, updates, security patch warnings, health/resource monitoring, firewall, fail2ban, SSH-hardening, audit trails, suspicious login alerts, jobs, notification sinks.
- Backup policies: schedules, retention, encryption at rest, S3/SFTP, point-in-time possibilities, per-site and whole-host exports, restore verification drills, disaster-recovery runbook. Backups must be offsite-capable and immutable when provider supports it.
- One-click supported fresh-server install and safe updates; preflight for resources, architecture, ports, clean OS, hostname, DNS, storage, privileges and supported packages.
- cPanel/WHM (and subsequently Plesk/DirectAdmin) migration via parsers with documented limits and dry-run. Never execute source-archive scripts. Restore in constrained staging.
- Multi-server control plane, remote workers, node labels, service placement, DNS and mail clusters, secrets, replication, failover as distinct advanced milestones.

### Developer/extensibility
- Versioned REST/OpenAPI API, events/webhooks, CLI, SDK, declarative desired-state adapters, scoped service permissions.
- Signed app/manifests and audited plugin permissions; no arbitrary administrator-equivalent marketplace code by default.
- Docs portal, troubleshooting recipes, tours, system-status pages, contribution guides, translated docs, sandbox/demo mode.

## 4. Delivery strategy / release boundaries

**Gate 0 Planning:** architecture, threat model, matrix, ADRs, legal provenance and agent rules. This repo is currently in Gate 0.
**Gate 1 Foundation:** preflight/installer, auth, admin shell, settings, jobs, log/audit, daemon boundary, tests.
**Gate 2 Hosting MVP:** Nginx/PHP-FPM, domains, files, MariaDB, SSL, backups, WordPress install. Secure isolated single-host alpha.
**Gate 3 Modern Apps:** Node.js, Python, Git deploy, deployments, env secrets, isolated build, app templates, optional containers.
**Gate 4 Full Hosting:** optional mail/DNS, PostgreSQL, enhanced backups, migration, resellers/packages and CLI/API coverage.
**Gate 5 Scale:** multi-server, orchestration, HA designs, rich marketplace, observability/ops and additional distros.
**Gate 6 Stable 1.0:** independent security audit, migration drills, restore drills, cross-distro matrices, load tests, accessibility and user docs.

A phase is not complete just because UI renders. It needs backend tests, safe permissions, integration and rollback tests, manual acceptance, docs and changelog evidence.

### Phase-by-phase integration plan and checkpoint (2026-11-13)

The following follows the Gate 0–6 numbering above and `ROADMAP.md`. `FEATURE_MATRIX.md` now follows the same P0–P6 phases and records the legacy-label crosswalk for historical notes. Upstream components are candidates, not approved production selections; retain their licenses/notices and review exact versions, dependencies, security, tenant boundaries, rollback and maintenance before release.

| Gate | Planned use and current state | Next acceptance work |
|---|---|---|
| 0 — Planning | Complete owner decisions, architecture, threat model and provenance. The [research report](OPEN_SOURCE_RESEARCH_REPORT_2026-11-12.md) maps possible upstream components; decisions remain open. | Record actual owner/security decisions, component review and phase gates. Draft prototypes do not approve a design. |
| 1 — Foundation | An opt-in Prometheus Go client integration exposes only a local development marker; see [development metrics](P1_DEVELOPMENT_METRICS.md). Guarded-read session-expiry writes now honor request cancellation during SQLite contention. Authentication, jobs and agent boundaries remain broader foundation work. | Define real metrics, credential and proxy policy, tenant isolation, secure transport and tests before production monitoring; retain current authority checks and fail-closed expiry behavior. |
| 2 — Hosting MVP | Evaluate [lego](https://github.com/go-acme/lego) for SSL/ACME, Nginx for web serving, and [restic](https://github.com/restic/restic) or another reviewed tool for backup. A local challenge test, authorized read-only preflight, recorded domain-proof evidence and cancellation of a prototype challenge attempt exist; certificate issuance is not wired into the panel. The restic fixture restore passed, but its tested binary has unresolved scanner findings and is not approved. | Resolve the explicit decisions in the [certificate operation and key-custody contract](P1_CERTIFICATE_OPERATION_CONTRACT.md), then implement reviewed permissions, custody, renewal and rollback with local-CA evidence. Resolve backup tool findings and test tenant isolation, failure recovery and full restore before use. |
| 3 — Modern apps | Evaluate [Coolify](https://github.com/coollabsio/coolify) workflows or a separately operated integration for Git/app deployment; Node/Python and optional isolated containers remain planned. | Review credentials, build isolation, resource limits, deployment/rollback and compatible integration boundaries; do not assume its control plane is already integrated. |
| 4 — Full hosting | Evaluate PowerDNS/BIND for optional DNS, Postfix/Dovecot for optional mail, PostgreSQL, enhanced backups, reseller controls and supervised migration. | Test scoped service adapters, mail delivery/DNS failure cases, recovery, tenant separation and migration reconciliation. |
| 5 — Scale | Plan node enrollment, placement, revocation, marketplace permissions and cross-server operations. | Prove partial failure, credential revocation, placement and recovery in disposable environments. |
| 6 — Stable release | Complete independent security review, signed releases, SBOM, support matrix, accessibility and recovery evidence. Optional [Coraza](https://github.com/corazawaf/coraza) or [CrowdSec](https://github.com/crowdsecurity/crowdsec) hardening requires its own evaluation. | Triage scanner findings, run supported-OS and restore drills, complete security review and document residual limitations. |

The [complete PR reconciliation](PR_RECONCILIATION_2026-11-13.md) preserves the original 149 draft heads in one development review, including the separate P0/P2/P3/P4/P5 models and packets. SQL job reads now recheck current cookie-session authority in their data snapshot; isolated model checks remain separate from product acceptance. Consolidation does not advance a phase or select a policy.

The [P2 responsive Sites and Files journey plan](P2_RESPONSIVE_SITES_FILES_JOURNEYS.md), prepared for issue #155, records proposed novice/admin journeys, viewport/state coverage, and human accessibility evidence. The last fully verified PR source b411 passed its hosted browser workflow. Source candidate 61fdb4b8faf520ec277cb5aeef82d4ac3510fe88 (tree 019c3f734a1743f91e1bdb3203013cbc154bec2c) adds phone-size target checks, forced-colors and reduced-motion keyboard journeys, theme-aware focus colors and 15 focus selectors sampled in both themes (30 selector/theme pairs); its responsive browser file passed 17/17 locally and the exact-head browser workflow passed at 2054. The focus checks are scoped synthetic evidence; integrated end-to-end implementation, human review, P0/P1 entry, and P2 acceptance remain pending.

The detailed current paths, test evidence and step-by-step resume instructions are in [the resume handoff](RESUME_HANDOFF_2026-11-12.md); precise component provenance and blockers are in the individual prototype READMEs linked there. The separate [ACME lab](../tools/acme-prototype/LOCAL_E2E.md#interrupted-order-response-and-reconciliation) now tests recovery of the same order after a lost finalize response, while keys remain in RAM. That is protocol evidence for uncertain outcomes, not durable recovery or panel issuance. This is a draft plan, not evidence that the listed services are installed or phases completed.

For the owner-directed first-release feature target and the evidence required before launch, see [the first release checklist](FIRST_RELEASE_CHECKLIST.md). The feature selection is recorded; supported platforms, exact acceptance criteria and the alpha/stable release designation remain open. This target does not replace the roadmap's stable 1.0 gates.

## 5. Proposed user journeys (acceptance)

1. New VPS: preflight identifies incompatible distro or occupied ports BEFORE making changes; fresh supported distro provisions panel and serves TLS admin UI.
2. New admin sets passkey/TOTP MFA, installs a PHP site, maps domain, HTTPS auto-enables, log display works; interrupted install is safely diagnosable.
3. Customer logs in and cannot enumerate/open other users' files, databases, mailboxes, tokens or job logs.
4. Nontechnical customer uses Explorer to upload site files, preview, rename, bulk move to trash and restore; accidental uploads do not grant execution.
5. Developer deploys a Node.js app from Git, sets secrets and domain, watches build logs, rolls back to prior known-good release; no shell root exposure.
6. Developer deploys a Python FastAPI app independently of other sites' Python versions and confirms reverse proxy/health checks.
7. Operator sets scheduled encrypted offsite backup, simulates lost VPS, restores and validates every selected account.
8. Mail operator creates mailbox with DKIM, runs DNS reputation preflight, checks delivery queue; blocked port 25 produces an actionable warning.
9. Reseller provisions limited customer package; limit enforced in both API and daemon, even against concurrent requests.
10. Importer inspects a cPanel backup in dry-run, sees unsupported features clearly, reviews planned resource allocations, executes controlled migration with validation.

## 6. Differentiators

- Unified **classic hosting + modern app PaaS** without making container deployment mandatory.
- True novice-to-expert design: friendly home grid and Finder/Explorer-style Files backed by robust API/CLI.
- Explainable jobs, recovery-first actions, first-run learning path, effective preview/dry-run.
- No premium-only features in the Tong Panel first-party core; open contribution model.
- Transparent capability status, graceful missing-service states and per-distro support matrix.

## 7. Definition of production-ready

No known critical/high exploitable security defects; zero unintended cross-tenant access in adversarial tests; trusted package signatures and reproducible builds; reliable clean install, safe upgrade/revert; repeatable restore drill; documented SLOs and support boundaries; API contract tests; accessible workflows; verified licence notices; signed release artifacts. Never promise 100% bug-free software.

## 8. Decisions and open questions

**Fixed:** Name Tong Panel; publisher ParmTech; fully open source; AGPL-3.0 selected; intuitive grid + desktop file-explorer UX; one-command supported-install goal; extensive features; stage planning before implementation.
**Proposals for ADR:** Original control plane (rather than unreviewed mixed forks); Go privileged agent + TypeScript web UI; PostgreSQL panel metadata; Debian 12/13 and Ubuntu 24.04 LTS as initial targets after CI validation; Nginx primary; optional mail, DNS and container engine; release channels alpha/beta/stable.
**Not approved as guaranteed:** all Linux distros, cPanel full compatibility, any specific release date, third-party source-code incorporation, zero-maintenance installs.

## 9. Canonical references
See [FEATURE_MATRIX.md](FEATURE_MATRIX.md), [ARCHITECTURE.md](ARCHITECTURE.md), [ROADMAP.md](ROADMAP.md), [SECURITY.md](SECURITY.md), [UPSTREAM_RESEARCH.md](UPSTREAM_RESEARCH.md), [../UPDATES.md](../UPDATES.md), and [../AGENTS.md](../AGENTS.md).
