# Tong Panel — Feature Matrix and Acceptance Specification
**Revision:** 0.1 | **Status:** aspirational backlog, not implemented.  
**Legend:** P0 = foundation; P1 = hosting MVP; P2 = modern apps; P3 = full-hosting; P4 = scale; P5 = stable. Priority is not availability. Phase references are from ROADMAP.

| Domain | Requirements | Priority | Minimum acceptance / proof |
|---|---|---|---|
| Installer | supported OS check, signed packages, one-command entrypoint, rollback | P0 | clean VM install/failed-preflight simulation |
| Panel access | admin bootstrap, MFA, recovery, web sessions, role permissions | P0 | authn/authz/rate-limit tests |
| Privileged operations | command allowlist, typed jobs, audit, least privilege | P0 | cannot run arbitrary root shell through API |
| User interface | icon-grid, search, responsive, accessibility, tours | P0 | keyboard + mobile + a11y verification |
| Sites/domains | create, alias, subdomains, redirects, vhosts | P1 | idempotent create/remove, cross-tenant tests |
| Web stack | Nginx, PHP-FPM, per-site PHP version and logs | P1 | concurrent PHP sites isolated |
| Files | Finder/Explorer tree, drag-drop, trash, archive, edit | P1 | no path traversal/symlink breakouts |
| Database | MariaDB create/export/users/rotation | P1 | private DB ownership and restore proof |
| SSL | ACME cert issue/renew, HTTPS redirects, status | P1 | expiration/renewal and challenge failures |
| Backups | local/offsite encrypted, restore drills, retention | P1 | new-host recovery from backup |
| Applications | WordPress, PHP templates, basic marketplace contracts | P1 | pinned install + safe removal |
| Node.js | runtime versions, builds, process health, domain proxy | P2 | Express/Next app deployed/rolled back |
| Python | venv builds, WSGI/ASGI, secrets, proxy, logs | P2 | Flask/Django/FastAPI sample deployments |
| Git deploy | OAuth/keys, webhook signatures, build history | P2 | Git-based deployment and rollback |
| Containers | isolated workloads, Compose profile, resource caps | P2 | no exposed docker socket/host files |
| App catalog | vetted recipes, upgrades, signatures, permissions | P2 | signed recipe and rollback tests |
| Mail | Postfix/Dovecot, mailboxes, DKIM/SPF/DMARC checks, webmail | P3 | send/receive/queue/reputation diagnostics |
| DNS | BIND/PowerDNS, provider APIs, DNSSEC, zone validation | P3 | authoritative/external-record tests |
| Advanced databases | PostgreSQL + optional Redis profiles | P3 | least-privileged provisioning and backup |
| Resellers | packages, quotas, team roles, suspension, accounting | P3 | malicious boundary tests, quota races |
| Migrations | cPanel import dry-run then supervised restore | P3 | safe staging + reconciliation report |
| API/CLI | versioned OpenAPI, scoped tokens, event hooks | P3 | contract, permission and pagination tests |
| Monitoring | service, logs, alerts, resource insights | P3 | anomaly/alert notification runbooks |
| Multi-node | secure agent enrolment, node jobs and topology | P4 | partial outage and revocation drills |
| Advanced app catalog | community submissions + moderation + signatures | P4 | threat model and review gates |
| Server migration | incremental site movement and DNS validation | P4 | controlled cutover/reverse plan |
| Hardening | external audit, penetration tests, SBOM, signed release | P5 | auditable security checklist |
| Reliability | load, fail/retry, chaos, resource exhaustion, upgrade | P5 | reference SLO and known limits |
| Docs | quickstart, API reference, tutorials, migrations, tours | P5 | independent newcomer install exercise |

## Application recipe catalogue (proposed)
- CMS: WordPress, Drupal, Joomla, Ghost, 84PAD (only after available), Grav.
- Frameworks: Laravel, Symfony, Django, Flask, FastAPI, Express, NestJS, Next.js, Nuxt, Rails.
- eCommerce: WooCommerce, PrestaShop, OpenCart, Magento/Adobe Commerce only where licensing/requirements allow.
- Collaboration: Nextcloud, Gitea/Forgejo, Wiki.js, BookStack, Matomo.
- Databases/services: PostgreSQL, MariaDB, Redis/Valkey, MongoDB where deployable; RabbitMQ.
- Developer tools: n8n, Grafana, Prometheus, Uptime Kuma, Plausible (verify licences).
**Important:** This is a proposed integration list, NOT bundled images, compatibility certification or authorization to redistribute proprietary assets. Recipe maintainers must review each application's license, resource demands, image supply chain, maintenance status and official install method.

## Every integration requires
1. Owner and upstream version/license recorded.
2. Compatibility matrix (OS/CPU/RAM/ports/storage).
3. Immutable reference + SHA/digest, package signature, SBOM where available.
4. Least-privileged filesystem/network capabilities and secrets handling.
5. Health checks, observability, upgrade/rollback and backup/restore.
6. Failure-mode docs and continuous integration test coverage.
7. Security patch SLA and removal/deprecation path.

## UX acceptance criteria
- No role sees controls it can never use without explanation.
- Main tasks reachable from Home grid and search in <= 3 navigational actions (excluding forms).
- Destructive action communicates scope, confirmation and recoverability.
- Every long-running task shows progress/status, logs and remediation path.
- Files provides accessible list/grid/tree views; bulk operations; breadcrumbs; keyboard shortcuts; no surprising deletion.
- First visit offers dismissible guided tour; comprehensive inline help; language and dark/light preferences.
- Performance targets to be established from baseline measurement, not guessed in planning.
