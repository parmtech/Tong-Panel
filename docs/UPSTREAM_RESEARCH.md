# Tong Panel — Upstream Capabilities, Licences and Provenance
**Research snapshot:** 2026-10-26. Revalidate upstream licensing, supported versions and security advisories immediately before integrating code.

This is feature/architecture research, NOT permission to copy software without its licence and attribution obligations. Upstream features are not yet implemented by Tong Panel.

| Project | Best pattern to study | Upstream licence (verify exact component) | Integration strategy |
|---|---|---|---|
| HestiaCP | Classic sites, PHP-FPM, DNS/mail, SSL, backups, easy installer | GPL-3.0 | Feature research; prefer protocols/independent compatible adapters; direct source reuse needs counsel/provenance review |
| Virtualmin GPL/Webmin | Web hosting automation, robust domain provisioning, mail, CLI/API, migrations, plugins | GPL-3.0 and some component-specific notices | Reference behaviors/API, adapter feasibility; avoid wholesale blend |
| ISPConfig | Multi-server, reseller permissions, mail/DNS separation | BSD (check exact code paths) | Architectural study and possible permissively licensed snippets with attribution |
| 1Panel Community | Modern server UX, app catalog and Docker patterns | GPL-3.0 for upstream community repository | UX inspiration and protocol integrations; **do not assume Pro features are open source** |
| Coolify | Git deploys, app/container lifecycle, secrets, health and rollback | Apache-2.0 repository; verify versions/dependencies | Design research, separate external integration where useful |
| CloudPanel | User-friendly PHP/Node/Python site provisioning | check upstream terms: NOT assumed fully open source | Inspiration only unless licence explicitly grants reuse |
| lego (go-acme) | ACME certificate lifecycle, many DNS provider plugins | check exact current module before vendoring | Candidate dependency/service interface |
| File Browser | Explorer-style files UI | Apache-2.0, upstream archived 2026-09-01 | **NOT a default dependency** due to maintenance/security warnings |
| Postfix / Dovecot / BIND / PowerDNS | Independent proven mail/DNS services | component-specific licences | Package-management and documented configuration adapters |
| Nginx / PHP-FPM / MariaDB / PostgreSQL | Core services | individual licences | Manage external processes/packages; comply with notice/distribution duties |

### Verified source URLs
- HestiaCP features: https://hestiacp.com/features
- HestiaCP repository: https://github.com/hestiacp/hestiacp
- Virtualmin GPL repository: https://github.com/virtualmin/virtualmin-gpl
- ISPConfig services: https://www.ispconfig.org/ispconfig/services-and-functions/
- ISPConfig docs: https://docs.ispconfig.org/about/
- 1Panel repository: https://github.com/1Panel-dev/1Panel
- 1Panel app store: https://github.com/1Panel-dev/appstore
- Coolify repository: https://github.com/coollabsio/coolify
- Coolify applications: https://coolify.io/docs/applications/
- Coolify database operations: https://coolify.io/docs/databases/
- CloudPanel features: https://www.cloudpanel.io/features/
- cPanel Application Manager as **feature benchmark only**: https://docs.cpanel.net/cpanel/software/application-manager/
- lego ACME: https://github.com/go-acme/lego
- File Browser maintenance warning: https://github.com/filebrowser/filebrowser

### Mandatory integration review checklist
For each proposed dependency or code segment record:
1. Precise URL, version/tag, commit SHA, component and licence SPDX ID.
2. Original copyright, NOTICE, source offer and AGPL/GPL compatibility analysis.
3. Whether the component is separate software/service vs linked/combined work; check derivative-work risks with knowledgeable legal advice.
4. Whether use of code, icons, screenshots, brand names and design assets is independently authorized. **Do not copy cPanel trademarks/assets/proprietary UI.**
5. CVE/security posture, maintainer activity, supported versions and release cadence.
6. Supply chain pin, package hash/signature, SBOM, tests and rollback strategy.
7. A migration/removal path if upstream becomes abandoned.
8. Security and license checklist reviewed in PR before merge.

### Original implementation policy
Tong Panel starts as an **original AGPL-3.0 application**; study multiple upstream tools, use Linux standard interfaces, package-manager services and well-licensed independent libraries where compatible. Do not copy large chunks or merge GPL/AGPL code arbitrarily. AGPLv3 section 13 discusses combining GPLv3-covered works, but this does NOT automatically clear every upstream asset, licence exception, plugin, design or trademark. Record decisions in ADR and provenance manifest.

### Unsupported assumptions
- “Open source” does not mean a commercial edition's complete feature set can be copied.
- “Free to use” does not always mean redistributable or AGPL-compatible.
- A marketplace recipe is not a guarantee its third-party app is free, maintained or compatible.
- Cross-platform Linux support must be proven per distribution and architecture.
