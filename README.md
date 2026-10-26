# Tong Panel
**A modern, friendly and fully open-source Linux web hosting control panel — a future cPanel/WHM alternative.**

**Publisher:** [ParmTech](https://parmtech.au)  
**Project:** [Tong Panel](https://github.com/parmtech/Tong-Panel)  
**License:** [GNU AGPL-3.0](LICENSE) for original Tong Panel code  
**Current status:** **Planning / blueprint only** — there is NO downloadable working panel or installer yet.

## What we are building
A capable but easy-to-use hosting dashboard for self-managed supported Linux VPS and dedicated servers. Our design combines conventional hosting tools (web domains, PHP, mail, DNS, database, SSL, backups, resellers) with modern app deployment (Node.js, Python, Git-based apps, isolated containers), without copying proprietary cPanel designs or software.

- **Simple, familiar UX:** responsive tablet-like icon grid; global search; desktop-like file explorer with tree/list/grid, multi-select, drag/drop and preview; accessible light/dark modes; tours and contextual docs.
- **Websites and apps:** PHP, WordPress, Laravel, static sites, Node.js, Python and eventually more; environment variables, safe deployments, logs and rollbacks.
- **Hosting management:** domains and redirects, SSL/ACME, MariaDB/PostgreSQL, email, DNS, backups, file manager, cron, SSH/SFTP, users and quotas.
- **Server and developer operations:** service health, hardening, audit, migrations, API/CLI, app marketplace and future multi-server.
- **Free software:** first-party core functions remain open source. External servers, cloud services, optional infrastructure and third-party applications may incur their own costs.

This is the *target blueprint*, not a list of completed functions.

## Documentation — start here
| File | What it contains |
|---|---|
| [Master Blueprint](docs/BLUEPRINT.md) | Product vision, end-to-end feature definition, user journeys and differentiators |
| [Feature Matrix](docs/FEATURE_MATRIX.md) | Detailed functional modules, priorities and acceptance criteria |
| [Architecture](docs/ARCHITECTURE.md) | Proposed control plane, privileged agent, database/jobs, security boundaries |
| [Roadmap](docs/ROADMAP.md) | Milestones P0–P6 and exit conditions |
| [UX/UI Specification](docs/UX_UI_SPEC.md) | Launcher-style home, Explorer/Finder-like file manager, accessibility and tours |
| [Installer Specification](docs/INSTALLER_SPEC.md) | Verified fresh-Linux setup, one-command goal, recovery and update strategy |
| [Security](docs/SECURITY.md) | Threat model, privilege separation, safe file handling, release gates |
| [Open-source Research](docs/UPSTREAM_RESEARCH.md) | HestiaCP, Virtualmin, ISPConfig, 1Panel, Coolify and third-party licence cautions |
| [Agent Handover](AGENTS.md) | How a developer/AI agent can resume work safely |
| [Latest Progress](UPDATES.md) | What is **actually** complete, blockers, history and exact next step |
| [Contributing](CONTRIBUTING.md) | Pull request, testing and review expectations |

## Current milestone
**P0 Planning:** Initial specification and source research recorded. **Next:** review proposed architecture/licensing decisions (ADRs), threat model and P1 backlog before implementation. See [UPDATES.md](UPDATES.md) for the authoritative checkpoint. There are currently no tested production installers, binaries or release artifacts.

## Installation
**Not available yet.** Please do not run unofficial install scripts claiming to be Tong Panel. A signed, versioned one-command installer for a tested OS matrix is a future requirement, not a current feature. Existing third-party shared hosting does not permit replacement of the provider's cPanel without root-level server administration.

## Source reuse and attribution
Tong Panel plans an original AGPL-3.0 codebase and compatible integrations to standard open-source server services. Upstream projects are researched for capability and design patterns, not automatically imported. Every dependency/code snippet needs source, licence, security and provenance review. cPanel and WHM are third-party trademarks and not affiliated with Tong Panel.

## Contributors
Read [AGENTS.md](AGENTS.md), [UPDATES.md](UPDATES.md) and [CONTRIBUTING.md](CONTRIBUTING.md) before starting or resuming work. Record true test status; do not treat design docs as working features.
