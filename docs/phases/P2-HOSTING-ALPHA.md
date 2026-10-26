# P2 — Secure Single-Server Hosting Alpha
**Status:** PLANNED, not implemented. **Depends:** P1 auth, typed jobs/host agent, installer lab and secret handling.

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
