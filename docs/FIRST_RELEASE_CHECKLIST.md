# Tong Panel — first release checklist

**Working proposal, 2026-11-12; not approved scope or release-ready certification.** This checklist proposes a limited first production release on a single supported Linux server. Owner/maintainers must decide its exact feature set, supported OS and whether the first publication is an alpha or stable 1.0. The current `ROADMAP.md` calls P2 an experimental hosting alpha and reserves stable 1.0 for P6 after broader work; this checklist does not silently change or waive those gates. Entries marked candidate are not selected or installed. No checkboxes below are acceptance claims.

## 1. Decide scope before building

- [ ] Owner approves target users, release name, supported OS/version/architecture, system requirements, support policy and exactly which features are in/out.
- [ ] Complete P0 architecture, threat-model, dependency/provenance and independent security decision records. Reconcile differing phase numbers in `ROADMAP.md` and `FEATURE_MATRIX.md`.
- [ ] Specify whether this is single-tenant admin hosting or multi-tenant shared hosting. Do not market multi-tenant support until isolation is tested.
- [ ] Set measurable feature acceptance, recovery targets and known limitations; agree who can approve release.

## 2. Minimum working product (proposed)

- [ ] **Installer and upgrades:** clean supported OS preflight, port/resource checks, signed/pinned packages, clear failures, upgrade, rollback and uninstall/recovery instructions. Fail safely on unsupported platforms.
- [ ] **Admin access:** secure bootstrap, TLS, login/logout, sessions, MFA and recovery, rate limits, account recovery and audited administrative actions. Independently assess whether an external identity service such as [Authelia](https://github.com/authelia/authelia) fits; do not assume it is integrated or that it replaces the panel's authorization.
- [ ] **Authorization and isolation:** roles, tenant/site ownership, server-side enforcement, restricted privileged agent and allowlisted typed jobs; no arbitrary root commands. Prove cross-tenant denial, revocation, path and symlink protections.
- [ ] **Websites and domains:** create/list/update/remove a site with safe validation, verified domain ownership, isolated filesystem and usable status/errors. Manage an externally installed [NGINX](https://github.com/nginx/nginx) and PHP-FPM through reviewed adapters with config validation and rollback.
- [ ] **Files and databases:** restricted file browsing/upload/download/restore as scoped, and MariaDB database/user lifecycle with credentials stored safely. Evaluate [phpMyAdmin](https://github.com/phpmyadmin/phpmyadmin) only if owner wants an optional separate database UI. Include safe deletion, limits and tenant isolation.
- [ ] **SSL:** issue, install, renew and recover certificates with domain rechecks, safe key/account storage, challenge cleanup and clear errors. [lego](https://github.com/go-acme/lego) is a candidate. Local test CA and read-only preflight exist; live issuance is not integrated. See `tools/acme-prototype/LOCAL_E2E.md` and `docs/P1_ACME_PREFLIGHT.md`.
- [ ] **Backups:** select encrypted local/offsite backup tool, protect keys and schedules, support failure recovery and restore to a fresh host. The [restic](https://github.com/restic/restic) synthetic drill passed but its tested binary has unresolved scanner findings; patched build investigation is blocked. Do not ship it until reviewed. See `tools/restic-prototype/README.md`.
- [ ] **User interface:** original responsive interface, accessible keyboard/screen-reader flows, status and actionable errors, consistent roles and no controls that falsely imply unfinished capabilities. Current SSL snapshot UI is read-only, not an issuance feature.
- [ ] **Service operations:** logs with secret redaction, health and disk/resource alerts, safe restart/recovery and audit visibility. Prometheus has only an opt-in development marker, not complete production monitoring; [Prometheus](https://github.com/prometheus/prometheus) or [Netdata](https://github.com/netdata/netdata) are candidates.
- [ ] **Applications:** if WordPress is in first-release scope, pin package sources and verify install, upgrade, rollback and backup/restore in clean disposable VMs. No unverified marketplace execution.

## 3. Optional candidates — not required unless owner includes them

- [ ] **DNS:** separately operated [PowerDNS](https://github.com/PowerDNS/pdns) or BIND; validate zone permissions, DNSSEC and recovery.
- [ ] **Email:** Postfix/Dovecot plus optional [Roundcube](https://github.com/roundcube/roundcubemail); validate send/receive, spam controls, DNS and port-25 limitations.
- [ ] **File transfer:** [SFTPGo Community](https://github.com/drakkan/sftpgo) or another tool; review per-component terms, especially its restricted UI theme, and verify isolation. Use original Tong Panel UI.
- [ ] **Malware and intrusion tools:** [ClamAV](https://github.com/Cisco-Talos/clamav), [Fail2ban](https://github.com/fail2ban/fail2ban), [Coraza](https://github.com/corazawaf/coraza) or [CrowdSec](https://github.com/crowdsecurity/crowdsec) only after false-positive, update, resource and recovery testing.
- [ ] **Git and modern app deployment:** assess [Coolify](https://github.com/coollabsio/coolify) as a separate integration or reference; isolate builds, secrets, ports and rollback.
- [ ] **Multi-node, marketplace, resellers, cPanel import and high availability:** separately scoped later work unless explicitly selected and tested for this release. Do not imply parity with cPanel.

## 4. Release evidence — mandatory for the selected scope

- [ ] Dependency inventory with exact versions, source, hashes, license/copyright/NOTICE and SBOM; security advisory triage and documented maintenance/update plan. Do not silently ignore scanner errors or findings.
- [ ] Automated unit, integration, contract and browser/accessibility checks against the release commit. Verify actual hosted CI results for that exact commit; Windows-only or cross-compilation is not Linux runtime evidence.
- [ ] Clean Linux VM install, normal operations, failed/interrupted install, upgrade/rollback and unsupported-OS tests on each advertised platform.
- [ ] Two-user/tenant adversarial tests where multi-tenancy is claimed, including revoked access, credential leakage, privilege boundaries, path traversal and malicious archives.
- [ ] Restore an encrypted backup to a clean new host, verify data and service recovery, and document timing and key-recovery process.
- [ ] Independent security assessment and owner review; remediate critical/high exploitable issues before stable release, document residual risks and follow the published decision process.
- [ ] Reproducible/signed build and provenance, release checksums, change log, installation/upgrade/restore docs, privacy and open-source notices, support matrix and known limitations.
- [ ] Explicit owner go/no-go after reviewing the evidence. No production deployment or release is authorized by creating this list.

## 5. Current starting point and order

1. Read `docs/RESUME_HANDOFF_2026-11-12.md`, `docs/BLUEPRINT.md`, `docs/ROADMAP.md` and `docs/OPEN_SOURCE_RESEARCH_REPORT_2026-11-12.md`; check current draft PR #166 and exact-head CI.
2. Resolve P0 decisions and agree first-release scope. Preserve existing work and private evidence boundaries.
3. Finish foundation/auth/isolation and a clean installer; implement a small complete site-to-restore path in disposable environments.
4. Integrate approved upstream components one at a time with provenance, tests and rollback; mark unfinished work as drafts.
5. Run all selected release checks, document remaining issues and seek the existing owner/security decisions. A research candidate or passing prototype alone does not satisfy release acceptance.
