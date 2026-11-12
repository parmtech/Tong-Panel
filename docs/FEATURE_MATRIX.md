# Tong Panel — Feature Matrix and Acceptance Specification
**Revision:** 0.2 | **Status:** requirements backlog; each capability needs its own implementation and acceptance evidence.

**Canonical phases:** [ROADMAP.md](ROADMAP.md) defines P0 planning, P1 foundation, P2 hosting alpha, P3 modern apps beta, P4 full hosting, P5 scale and P6 stable readiness. The phase column below names the intended milestone, not current availability or approval.

## Phase numbering and release scope

Earlier revisions of this matrix omitted planning and numbered implementation phases one lower than the roadmap. Use the following crosswalk when reading historical notes; it does not move work through a gate or certify a feature.

| Earlier matrix label | Canonical roadmap phase |
|---|---|
| No corresponding label | P0 — Planning, product and trust boundary |
| P0 — Foundation | P1 — Core platform foundation |
| P1 — Hosting MVP | P2 — Secure single-server hosting alpha |
| P2 — Modern apps | P3 — Developer apps beta |
| P3 — Full hosting | P4 — Full hosting parity |
| P4 — Scale | P5 — Scaling and ecosystem |
| P5 — Stable | P6 — Stable 1.0 release readiness |

The owner's [first-release target](FIRST_RELEASE_CHECKLIST.md) includes capabilities across P1–P4, including DNS, mail, Node.js/Python apps, an isolated browser terminal, resellers/quotas and cPanel migration. Release selection and phase numbering are separate: the acceptance, isolation, recovery and release gates still apply. Integrated panel software must be free and open source, with no mandatory paid components; Softaculous is not selected. The exact supported platforms, component selections and alpha/stable designation remain undecided.

Installer and API foundations begin in P1; installable hosting packages are a P2 requirement and stable release certification is P6. Monitoring and documentation also develop earlier than their comprehensive P4/P6 targets. A later phase label does not postpone controls needed by an earlier feature.

## Feature requirements

| Domain | Requirements | Canonical phase | Minimum acceptance / proof |
|---|---|---|---|
| Installer | supported OS check, signed packages, one-command entrypoint, rollback | P1 foundation; P2 hosting package; P6 stable release | clean VM install/failed-preflight simulation |
| Panel access | admin bootstrap, MFA, recovery, web sessions, role permissions | P1 | authn/authz/rate-limit tests |
| Privileged operations | command allowlist, typed jobs, audit, least privilege | P1 | cannot run arbitrary root shell through API |
| User interface | icon-grid, search, responsive, accessibility, tours | P1 | keyboard + mobile + a11y verification |
| Sites/domains | create, alias, subdomains, redirects, vhosts | P2 | idempotent create/remove, cross-tenant tests |
| Web stack | Nginx, PHP-FPM, per-site PHP version and logs | P2 | concurrent PHP sites isolated |
| Files | Finder/Explorer tree, drag-drop, trash, archive, edit | P2 | no path traversal/symlink breakouts |
| Database | MariaDB create/export/users/rotation | P2 | private DB ownership and restore proof |
| SSL | ACME cert issue/renew, HTTPS redirects, status | P2 | expiration/renewal and challenge failures |
| Backups | local/offsite encrypted, restore drills, retention | P2 | new-host recovery from backup |
| Applications | WordPress, PHP templates, basic marketplace contracts | P2 | pinned install + safe removal |
| Node.js | runtime versions, builds, process health, domain proxy | P3 | Express/Next app deployed/rolled back |
| Python | venv builds, WSGI/ASGI, secrets, proxy, logs | P3 | Flask/Django/FastAPI sample deployments |
| Git deploy | OAuth/keys, webhook signatures, build history | P3 | Git-based deployment and rollback |
| Containers | isolated workloads, Compose profile, resource caps | P3 | no exposed docker socket/host files |
| App catalog | vetted recipes, upgrades, signatures, permissions | P3 | signed recipe and rollback tests |
| Mail | Postfix/Dovecot, mailboxes, DKIM/SPF/DMARC checks, webmail | P4 | send/receive/queue/reputation diagnostics |
| DNS | BIND/PowerDNS, provider APIs, DNSSEC, zone validation | P4 | authoritative/external-record tests |
| Advanced databases | PostgreSQL + optional Redis profiles | P4 | least-privileged provisioning and backup |
| Resellers | packages, quotas, team roles, suspension, accounting | P4 | malicious boundary tests, quota races |
| Migrations | cPanel import dry-run then supervised restore | P4 | safe staging + reconciliation report |
| API/CLI | versioned OpenAPI, scoped tokens, event hooks | P1 foundation; P4 comprehensive coverage | contract, permission and pagination tests |
| Monitoring | service, logs, alerts, resource insights | P4 | anomaly/alert notification runbooks |
| Multi-node | secure agent enrolment, node jobs and topology | P5 | partial outage and revocation drills |
| Advanced app catalog | community submissions + moderation + signatures | P5 | threat model and review gates |
| Server migration | incremental site movement and DNS validation | P5 | controlled cutover/reverse plan |
| Hardening | external audit, penetration tests, SBOM, signed release | P6 | auditable security checklist |
| Reliability | load, fail/retry, chaos, resource exhaustion, upgrade | P6 | reference SLO and known limits |
| Docs | quickstart, API reference, tutorials, migrations, tours | P6 | independent newcomer install exercise |

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
