# Tong Panel — Master Product Blueprint
**Project:** Tong Panel (TongPanel) | **Publisher:** ParmTech (https://parmtech.au)  
**Project type:** Free/open-source Linux hosting control panel and cPanel/WHM alternative  
**License:** GNU AGPL-3.0 for original Tong Panel code; all third-party dependencies retain their own licences.  
**Status:** PRODUCT PLANNING — no deployable Tong Panel exists yet  
**Updated:** 2026-10-26 | **Blueprint revision:** 0.1

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
