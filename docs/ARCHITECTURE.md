# Tong Panel — Architecture Proposal (ADR candidates)
**Status:** proposed, NOT implemented; obtain review before coding.

## Topology

~~~text
Browser / Mobile Browser
        |
 TLS edge -> unprivileged API + session/auth/RBAC
        |           |       \
        |           |        -> control-plane metadata DB
        |           -> durable job queue & audit ledger
        |                   |
        |               policy engine
        |                   |
        |        narrow UNIX socket / mutual-auth channel
        |                   |
        |         privileged local host agent
        |            (typed allowlisted operations)
        |        /      |        |       |       \
      Web    DB/Files   ACME   Mail/DNS   Backups
        |
     App workloads (isolated Unix users / sandboxed containers)
~~~

No HTTP endpoint directly executes shell strings. The agent alone performs strictly validated actions requiring elevated permissions. UI/API services never receive the Docker socket or unrestricted root access.

## Proposed implementation technologies (to review)
- **Frontend:** TypeScript, React (or another proven accessible framework), responsive design system; no external CDN necessary for core panel.
- **API/control plane:** Go HTTP service with OpenAPI/JSON Schema and structured telemetry. Go offers a strong self-contained system service for Linux integration.
- **Privileged host agent:** separate minimal Go daemon, root only for specific allowed operations; Linux seccomp/AppArmor/systemd confinement where feasible, stringent file-path checks.
- **Control-plane storage:** PostgreSQL or SQLite (small-host profile) behind a migration interface; finalize after benchmarks and backup/upgrade tests. Do not accidentally require a large DB for a small-VPS profile.
- **Job queue:** transactional DB-backed queue initially with job lease, retry, lock, idempotency keys, cancellation state, exponential backoff; adapters later for scale.
- **Web:** Nginx and PHP-FPM; package-managed binary versions pinned to supported distribution profiles; optional Apache adapter later.
- **Deploy/build:** explicit native workloads under systemd and isolated build worker; optional rootless-container stack with safe networking.
- **Certificate management:** standard ACME client such as lego (license verification), separate issuance/renewal records; DNS/HTTP challenges.
- **File UI:** original Tong Panel frontend + hardened server filesystem primitives, not arbitrary embedding of legacy file managers.
- **Mail optional:** Postfix + Dovecot and independent webmail integration, security/deliverability checks.
- **DNS optional:** BIND or PowerDNS; provider adapters use scoped tokens.
- **Monitoring:** OpenTelemetry-compatible spans/structured logs, systemd/Prometheus exporters where feasible.

## Control-plane data model (minimum)

Server, ServiceInstance, OperatingSystemProfile, PackageProfile, AdminIdentity, Tenant, Membership, Role, Permission, HostingPackage, Quota, Domain, Site, Runtime, Application, Deployment, Build, EnvironmentSecretRef, DBInstance, DatabaseUser, MailDomain, Mailbox, DNSZone, DNSRecord, Certificate, StorageTarget, BackupPolicy, BackupArtifact, RestoreJob, PluginManifest, AuditEvent, Agent, Job, JobAttempt, HealthCheck, Notification, Webhook, ReconciliationDrift.

Every object has immutable ID, owner/tenant scope, revision/version and timestamps. Authorization enforced on all read/write operations and on worker execution, not only UI.

## Host mutation API contract
- Typed schema commands: CreateSite, ConfigureVhost, InstallRuntime, CreateDatabase, IssueCertificate, BackupTenant, RestoreTenant, etc.
- Validate caller capability, tenant scope, resource quotas, path ownership, port allocation and desired-state revision.
- Begin durable audited job -> stage temp configs -> syntax-check -> atomically swap -> reload only affected service -> health-check -> mark successful or safely revert.
- No caller-supplied shell snippets in privileged jobs; commands assembled from trusted constants and validated arguments; never interpolate user input into shells.
- Use distributed/host locks for domain, system package, and service configuration updates; idempotent retries and crash reconciliation.
- Audit who/what/when/before-after, redact secrets, retention controls, tamper evidence, support export.

## Isolation & risk tiers
Tier A: web customer app (untrusted content); Tier B: API/control plane (unprivileged); Tier C: host agent (privileged, allowlisted); Tier D: root-host operations (minimal, exceptional).
Use Linux users, ACLs, namespaces and cgroups; no shared readable credentials. Rootless containers preferred where practical. Prevent symlink/TOCTOU escapes using file descriptors/openat2 and careful Linux API use. Tenant-initiated deploys may execute arbitrary app code inside scoped workloads but never as panel/agent identity.

## Panel installation model
1. Signed pinned bootstrap release downloaded from official, HTTPS-verified source; provide documented checksums/signatures. Avoid unsafe curl-to-shell instructions without provenance verification.
2. Check OS/version/arch, kernel, root privileges, hostname/DNS, services/ports, memory/disk, apt repository safety, firewall/cloud security groups, TLS/challenge reachability.
3. Print precise plan; interactive confirm; non-interactive signed flags; no changes on unsupported host.
4. Stage packages/services, create system identities and database, initialize TLS/MFA bootstrap, preflight app stack, start services and health-check.
5. Roll back reversible steps, preserve diagnostic state; warn before irreversible package/network changes.
6. Present a unique short-lived bootstrap token out-of-band, and first-login workflow to choose credentials + MFA, not an embedded universal password.
7. Versioned upgrades, preflight checks, DB migration backup, health-check, snapshot/rollback and clear failure messages.

**Supported platform promise:** first prove specific fresh Debian/Ubuntu x86_64/arm64 combinations. Expand to AlmaLinux/Rocky after separate packaging/firewall/SELinux tests. Never market 'any Linux/VPS' without demonstrated support.

## Files / explorer security
Virtual root per tenant, safe relative path canonicalization, reject parent-directory traversal, control symlink and archive-extraction attacks, run privileged actions by fd/capability, no arbitrary URL fetching, size/virus/quota checks, resumable uploads, rate limits. Backups and temporary directories must not be reachable by customer HTTP.

## Extension and app store
Separate two extensibility types:
- **App recipes:** declarative manifests describing how to deploy isolated applications, with required permissions, provenance, immutable image digests, updates.
- **Panel plugins:** authenticated, narrow API extensions in sandbox/out-of-process workers, capability approval, version compatibility and audit; no direct access to host agent except typed API permissions.
Default deny; reviewed first-party catalog; community recipes require scanning and trust badges.

## Multi-server evolution
Agents register with explicit one-time scoped enrolment; mTLS cert rotation, revocation; encrypted reliable job dispatch, drift reconciliation, node discovery/labeling; datastore/queue topology chosen separately. Control plane cannot assume all nodes online. Initial releases must not promise HA.

## Nonfunctional gates
Threat model before privileged actions; reproducible CI artifact signing, dependency/OS vulnerability scans, SBOM, independent review, full-disk/space/kill-mid-job fault tests, migrations, upgrade/downgrade rules, backup/restore proof, role/a11y tests, performance and resource budgets.

## ADR queue
- ADR-001 Original implementation vs fork, with provenance/legal review.
- ADR-002 Go API/agent + TS web UI and build tooling.
- ADR-003 Metadata DB and durable job architecture.
- ADR-004 Native vs container app execution profiles.
- ADR-005 distro support and packaging.
- ADR-006 DNS and mail integration strategy.
- ADR-007 privilege separation/secret encryption keys and rotation.
- ADR-008 release signing and bootstrap chain of trust.

Consult SECURITY.md, ROADMAP.md, FEATURE_MATRIX.md and UPSTREAM_RESEARCH.md.
