# Tong Panel open-source research report

Research snapshot: 2026-11-12. Audience: maintainers and the next worker. **This is a candidate shortlist, not an exhaustive internet search, authorization to copy code, proof of security, or completed integration.** Repository-level license labels are preliminary; check the precise release, subcomponents, bundled rules, trademarks and notices before use. Phase references follow `ROADMAP.md`; the feature matrix calls its backlog aspirational.

## Priorities and candidates

| Priority / blueprint need | Project and upstream | Preliminary repository license | Proposed use / integration boundary | Checks before implementing |
| --- | --- | --- | --- | --- |
| P1 certificate issue/renew | [lego](https://github.com/go-acme/lego) | MIT | Compare a separately managed ACME client with a Go library adapter. Reference tag v4.28.1 resolves to `b704b26e6c49a540cbb5dd51c2f3a9d70053c86c`; this is not a selected production version. | Challenge isolation, DNS credentials, account keys, renewal failure, expiry alerts, rollback and local ACME test server. Verify exact version, transitive licenses and advisories. |
| P1 encrypted backup/restore | [restic](https://github.com/restic/restic) | BSD-2-Clause | Evaluate separately installed backup executable through a restricted typed job adapter; avoid assuming all workloads are crash-consistent. | Database-consistent snapshots, key custody, untrusted output handling, restore to a new host, retention, interruption and remote storage tests. |
| P1 HTTP and reverse proxy | [NGINX Open Source](https://github.com/nginx/nginx) | Verify exact release and each bundled module; repository license alone is insufficient | Manage OS-provided service through validated config and least-privilege adapter; no source copy required. | Version/OS compatibility, atomic config validation and rollback, tenant isolation, upgrade and TLS tests. |
| P2 apps and Git deployment | [Coolify](https://github.com/coollabsio/coolify) | Apache-2.0 repository; check specific files, images and dependencies | Study deployment workflows or evaluate a separately operated integration; do not merge its control plane wholesale. | Threat model, webhooks, build isolation, credentials, container privilege, rollback and exact package licensing. |
| P3 authoritative DNS | [PowerDNS](https://github.com/PowerDNS/pdns) | GPL-2.0 family; upstream says inspect NOTICE and exceptions | Evaluate separately packaged authoritative DNS service via a scoped API adapter. Do not assume source-code mixing is compatible with Tong Panel. | Exact component license, DNSSEC, API credentials, tenant boundaries, zone validation, backup and recovery. |
| P3 monitoring | [Prometheus](https://github.com/prometheus/prometheus) | Verify exact release and component licenses | Evaluate separately operated metrics collection and documented interface. | Sensitive labels, tenant isolation, resource limits, retention and alerts. |
| P3 optional cache | [Valkey](https://github.com/valkey-io/valkey) | Verify exact release and component licenses | Optional independently managed service, not a mandatory P0 dependency. | Authentication, network isolation, persistence, memory quotas and package lifecycle. |
| Later hardening | [Coraza](https://github.com/corazawaf/coraza) | Apache-2.0 repository; rulesets and connectors may differ | Evaluate WAF only after product need and traffic placement are defined. Observed main `8538b97cc1d7b0cc03384704f1b5cfb035e4c83c` is not a release pin. | Ruleset licenses, false positives, per-tenant policies, limits, security advisories and fail-open/fail-closed behavior. |
| Later hardening | [CrowdSec Security Engine](https://github.com/crowdsecurity/crowdsec) | MIT repository; independently verify remediation components, rules and services | Consider optional external security integration, not automatic blocking. | Privacy/data sharing, licensing, local-only behavior, privileges, false positives and recovery from lockouts. |
| Hosting UX / feature research | [HestiaCP](https://github.com/hestiacp/hestiacp), [1Panel Community](https://github.com/1Panel-dev/1Panel) | Both upstream repositories identify GPL-3.0; verify exact files and versions | Research workflows and service boundaries. No wholesale fork or direct code import without provenance and compatibility review; do not assume commercial editions are open source. | Component-specific license and copyright, independent implementation, security boundary and owner architecture decision. |

## Recommended order

1. Finish Phase Zero decisions and privilege/security boundaries; none of these candidates substitutes for them.
2. Evaluate lego and restic in isolated prototypes against defined certificate and restore acceptance tests. Select exact versions only after checking maintenance, advisories, transitive dependencies, licensing and packaging.
3. Assess external service adapters for NGINX and later PowerDNS using typed operations and configuration validation. Never grant arbitrary root shell access.
4. Evaluate deployment, monitoring and hardening candidates only when their roadmap capabilities and operating costs justify them.

## Required integration record

For each actual import record the upstream URL, exact release and commit, checksum, copyright/license and notices, transitive dependencies and SBOM, security advisory review, owner/security review, supported platforms, minimal privileges, failure and rollback tests, upgrade plan and removal path. Verify GPL/AGPL compatibility for the exact combination; a separate service and copied/linked code may have different implications. No third-party source, package, ruleset, secrets or proprietary cPanel assets were added as part of this research.

## Current status

The earlier candidate note records the lego and Coraza leads. This expanded document adds backup, service, DNS, monitoring, security and hosting-panel leads. It does not claim to cover every candidate or complete implementation; consult `UPSTREAM_RESEARCH.md`, `FEATURE_MATRIX.md`, and `ADR_REGISTER.md` for existing requirements and unresolved decisions.
