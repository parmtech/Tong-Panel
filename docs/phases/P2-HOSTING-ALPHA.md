# P2 — Secure Single-Server Hosting Alpha
**Status:** PLANNED, not implemented. **Depends:** P1 auth, typed jobs/host agent, installer lab and secret handling.

**Entry gate:** recorded P0 owner/security approval and P1 exit approval are required before P2 implementation or host-facing tests. Until then, work is limited to unprivileged design, prototypes and contract tests. Slice 1's signed-package/bootstrap/recovery work is downstream of these prerequisites; it does not substitute for them.

See the [P2 acceptance traceability matrix](../P2_ACCEPTANCE_TRACEABILITY.md) for the mapping from all ten slices and must-pass exit criteria to the proposed scenario and required evidence. It is a design map only and does not change gate status.

## Implementation slices
1. Supported distro package/installer, preflight, journal and recovery.
2. Web service adapters: Nginx and PHP-FPM per site; runtime/version selectors only for tested versions.
3. Domains/aliases/subdomains, DNS ownership/preflight, redirects, vhost config and health checks.
4. Databases MariaDB/MySQL: scoped user provisioning/rotation, export/import and isolation.
5. Original secure Finder/Explorer-style files UI/API, quotas and metadata/ACL.
6. ACME certificates: HTTP-01 and DNS-01 provider capability, renewal, status, HTTPS redirects and real verification. DNS-01 provider work depends on slice 3's approved domain/DNS proof flow and reviewed provider-credential custody; no provider secrets or host effects before those decisions.
7. Cron, sites logs, versioned server templates; graceful service failure handling. See the [proposed Slice 7 acceptance and test plan](../P2_CRON_LOGS_TEMPLATES_TEST_PLAN.md) and [schedule policy decision brief](../P2_SLICE7_POLICY_DECISION_OPTIONS.md); these are design only and do not authorize implementation.
8. WordPress/pure PHP deploy recipe with pinned versions, checksums and restore.
9. Account backup schedules, offsite encrypted targets, retention and complete restore drills. Depends on approved key custody/recovery and offsite/retention policy, a reviewed durable scheduler/job contract (slice 7), and validated site/runtime, database and application state (slices 2–8). Restore must not reactivate stale scheduled executable state.
10. Multi-tenant adversarial testing and understandable UX/onboarding. See the [proposed two-tenant end-to-end acceptance plan](../P2_TWO_TENANT_END_TO_END_ACCEPTANCE.md); design only, not gate evidence.

## Must-pass end-to-end tests
- Two sites and two separate tenants cannot access each other's data via API, files, DB users or job logs.
- Unsupported OS, low disk and occupied ports fail **before** destructive mutations.
- Existing certificates detected; issuance/retry/renewal and HTTP->HTTPS state verified externally.
- Atomic config staged/syntax checked, service reload failure rolls back.
- Backup to offsite, destroy lab VM, restore on clean VM and verify data and settings.
- After component readiness, run the complete authenticated two-tenant workflow on each exact supported disposable guest; destroy the source overlay, restore sites/databases/settings from retained artifacts, then rerun cross-tenant denial checks. Component labs are supporting evidence, not phase-exit evidence.
- Upload of malicious archive/symlink cannot escape jailed user paths.
- Login/MFA/recovery doesn't allow bypass across users; audit logs redact secrets.
- Basic phone/tablet site/domain/file tasks complete with guidance.

## Alpha release gate
Signed experimental artifacts for published exact OS/architecture matrix. Every limitation visible. Not marketed as production ready; email/DNS authoritative host and Node/Python deferred to P3/P4.
