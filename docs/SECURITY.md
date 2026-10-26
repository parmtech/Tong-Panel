# Tong Panel — Security and Trust Blueprint
**Status:** design requirements only. No implementation has yet been audited.

## Threat model actors
- Internet unauthenticated visitor; credential thief; compromised customer account; tenant running malicious PHP/Node/Python; malicious dependency/app recipe/plugin; compromised reseller; compromised worker; privileged host insider; backup thief.
- Assets: root host integrity, customer files, domain control, databases, DNS, mailboxes, credentials, OAuth/webhook secrets, backups, signing keys, audit trail.

## Highest-risk trust boundaries
1. Web/API request -> control plane auth and resource scope.
2. Control plane -> root-like local host agent.
3. User uploaded files -> filesystem, symlinks, archive extraction.
4. User app deployment -> build runner, runtime and network.
5. External integrations -> provider credentials/webhooks.
6. Marketplace manifest/image -> installed executable.
7. Backup archive and migration package -> restore staging.
8. Controller -> remote agents over network.

## Mandatory controls before first public alpha
- Admin MFA (TOTP and ideally passkeys), non-default bootstrap, short-lived reset flows, rate limits, credential rotation, safe recovery and session invalidation.
- Argon2id or equivalent modern password hashing; secure cookie/session flags, CSRF mitigation where applicable, server-side session revocation, CSP, SSRF protection, secure origin/TLS and webhook signature checks.
- Per-request + per-job tenant-scoped RBAC/ABAC. No client-supplied tenant ID authorization, all objects checked.
- Small privileged agent surface: allowlisted typed commands, no shell interpolation, audited args/results, temporary file and process isolation. Permission to restart one vhost is not permission for arbitrary host command.
- Safe file tree operations using openat2 or equivalent, path base and inode checks, symlink defense, no untrusted archive path escape, atomic writes and strict per-tenant UID/GID.
- Scoped access to databases, backup destinations and DNS providers; envelope encryption, key rotation, secret redaction; no long-lived raw credentials in logs.
- Build/deploy workloads sandboxed (non-root, limited network/resources, no host root mounts, no Docker socket); image digests pinned and verified.
- SSH key management, firewall defaults and rollback planning (do not lock out the operator), fail2ban, audit log, security headers.
- Signed release artifacts and authenticated distribution; dependency governance, vulnerability monitoring, SBOM and reproducible build goals.
- Exportable backups encrypted, retention/deletion verified, test restored on separate host; neither logs nor restores expose other tenants' data.
- One active critical infrastructure config change per affected service; stage/check/apply/health/revert with idempotent job state.

## Security test suite specification
- Authorization matrix all endpoints and asynchronous workers across roles/tenants.
- Fuzz host-agent input, configs, archive extraction and file paths.
- Attack cases: directory traversal, symlink race, malicious archive, command injection, SSRF, CSRF, XSS, credential stuffing, tenant escape, Docker socket access, malicious app manifest, replayed webhook, package substitution.
- Kill processes during package changes, TLS certificate issuance, backup and database restore; verify consistent reconciliation.
- Dependency scanning, SAST, dynamic tests, penetration testing before stable release.
- Distribution install on fresh VMs; verify log scrubbing, bootstrap expiry, firewall connectivity and rollback.
- Threat review before shipping ANY feature that executes customer-supplied software.

## Security disclosure policy (proposed)
Report vulnerabilities privately through GitHub Security Advisories once enabled, not public issues with exploit details. Until configured, coordinate a vetted contact method with project maintainers. Do NOT invent an email inbox. SECURITY.md here is design status, not a statement that an audited vulnerability program exists.

## Licence/provenance is also a security issue
Archived or insecure upstream components are not safe defaults even when open-source. In September 2026 the original File Browser project archived its repository and warned of unresolved session and command execution classes: treat as research/UI inspiration only, NOT production dependency, without a separately maintained audited fork.

## Incident readiness
Define severity and impact, key revocation, affected-host quarantine, config snapshot, immutable evidence, tenant notice and restoration procedure; establish independent incident access path if panel unavailable.

## Production release blocker list
- Known unmitigated high/critical vulnerabilities.
- Cross-tenant access in any module.
- Unbounded root-command surface.
- Backup untested, missing or readable cross-tenant.
- Default credentials/secrets or unsigned binary distribution.
- Unknown migration/destructive update behavior.
