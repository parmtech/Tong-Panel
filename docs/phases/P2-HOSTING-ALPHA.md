# P2 — Secure Single-Server Hosting Alpha
**Status:** IN PROGRESS. Component prototypes exist on unmerged draft PRs, but P2 is not an integrated or installable alpha. **Depends:** P1 auth, reviewed typed jobs/host agent, installer lab and secret handling.

## Implementation slices
1. Supported distro package/installer, preflight, journal and recovery.
2. Web service adapters: Nginx and PHP-FPM per site; runtime/version selectors only for tested versions.
3. Domains/aliases/subdomains, DNS ownership/preflight, redirects, vhost config and health checks.
4. Databases MariaDB/MySQL: scoped user provisioning/rotation, export/import and isolation.
5. Original secure Finder/Explorer-style files UI/API, quotas and metadata/ACL.
6. ACME certificates: HTTP-01 and DNS-01 provider capability, renewal, status, HTTPS redirects and real verification.
7. Cron, sites logs, versioned server templates; graceful service failure handling.
8. WordPress/pure PHP deploy recipe with pinned versions, checksums and restore.
9. Account backup schedules, offsite encrypted targets, retention and complete restore drills.
10. Multi-tenant adversarial testing and understandable UX/onboarding.

## Must-pass end-to-end tests
- Two sites and two separate tenants cannot access each other's data via API, files, DB users or job logs.
- Unsupported OS, low disk and occupied ports fail **before** destructive mutations.
- Existing certificates detected; issuance/retry/renewal and HTTP->HTTPS state verified externally.
- Atomic config staged/syntax checked, service reload failure rolls back.
- Backup to offsite, destroy lab VM, restore on clean VM and verify data and settings.
- Upload of malicious archive/symlink cannot escape jailed user paths.
- Login/MFA/recovery doesn't allow bypass across users; audit logs redact secrets.
- Basic phone/tablet site/domain/file tasks complete with guidance.

## Alpha release gate
Signed experimental artifacts for published exact OS/architecture matrix. Every limitation visible. Not marketed as production ready; email/DNS authoritative host and Node/Python deferred to P3/P4.


## Evidence-based implementation checkpoint — 29 October 2026
Verified component prototypes on unmerged drafts include Nginx/PHP-FPM renderers, two-tenant clean-VM HTTP/TLS checks, SQLite site/domain/database metadata, isolated MariaDB grant tests, one-time domain proof, authenticated loopback site/domain APIs and site alias create/remove. They are not wired into an installer or production host adapters. The full P2 acceptance below remains outstanding wherever lifecycle integration, rollback, data recovery, certificate issue/renew, UI or owner/security approval is required.

**Additional evidence:** PR #69 adds verified-domain preview filtering and PR #70 exposes the tenant-scoped read-only preview API; their exact-head Go, daemon, archive and Ubuntu/Debian VM checks pass. The API remains disabled without explicit server-injected paths and does not apply host configuration.

**Files UI checkpoint (draft PR #73):** a read-only accessible Explorer can navigate folders and download regular files through the tenant-scoped API. It uses same-origin credentials, safe text rendering, path/entry validation, no link following, a 16 MiB download cap and bounded requests. Local Node 20/20 and TypeScript 5.9 checks passed; hosted UI passed at `b25ab4060c47b994d28e8595a496ef594ad49eb1`. Other exact-head Go/daemon/archive/VM jobs were pending at report time. Writes remain unavailable without an approved browser CSRF/session path. This does not change the phase status: installer/recovery, app-driven database/ACME lifecycle, backup restore, integrated two-tenant acceptance and P0 owner/security decisions remain open.
