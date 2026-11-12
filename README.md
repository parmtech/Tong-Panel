# Tong Panel
**A modern, friendly and fully open-source Linux web hosting control panel — a future cPanel/WHM alternative.**

**Publisher:** [ParmTech](https://parmtech.au)  
**Project:** [Tong Panel](https://github.com/parmtech/Tong-Panel)  
**License:** [GNU AGPL-3.0](LICENSE) for original Tong Panel code  
**Current status:** Development prototypes and review proposals are consolidated on `integration/pr-reconciliation-20261113`. P0/security and production release gates remain open. Read the [current handoff](docs/RESUME_HANDOFF_2026-11-12.md).

## What we are building
A capable but easy-to-use hosting dashboard for self-managed supported Linux VPS and dedicated servers. Our design combines conventional hosting tools (web domains, PHP, mail, DNS, database, SSL, backups, resellers) with modern app deployment (Node.js, Python, Git-based apps, isolated containers), without copying proprietary cPanel designs or software.

- **Simple, familiar UX:** responsive tablet-like icon grid; global search; desktop-like file explorer with tree/list/grid, multi-select, drag/drop and preview; accessible light/dark modes; tours and contextual docs.
- **Websites and apps:** PHP, WordPress, Laravel, static sites, Node.js, Python and eventually more; environment variables, safe deployments, logs and rollbacks.
- **Hosting management:** domains and redirects, SSL/ACME, MariaDB/PostgreSQL, email, DNS, backups, file manager, cron, SSH/SFTP, users and quotas.
- **Server and developer operations:** service health, hardening, audit, migrations, API/CLI, app marketplace and future multi-server.
- **Free software:** first-party core functions remain open source. External servers, cloud services, optional infrastructure and third-party applications may incur their own costs.

This is the *target blueprint*, not a list of completed functions.

## Development status and phase plan
The seven-phase execution plan is in [PHASE_INDEX](docs/PHASE_INDEX.md) with individual P0–P6 specifications and [kickoff report (2026-10-26)](docs/REPORT_2026-10-26.md). Nine architecture decisions are documented as proposals in [ADR_REGISTER](docs/ADR_REGISTER.md), alongside [THREAT_MODEL](docs/THREAT_MODEL.md) and [TEST_STRATEGY](docs/TEST_STRATEGY.md).

The [PR reconciliation report](docs/PR_RECONCILIATION_2026-11-13.md) accounts for every one of the 149 original open draft heads and their observed checks. The combined development tree includes the current API/auth/jobs/catalog, read-only UI and isolated component labs, along with the previously separate P0/P2/P3/P4/P5 review work. The [machine-readable ledger](docs/PR_RECONCILIATION_2026-11-13.json) preserves original commit and workflow evidence. These are development components and proposals; phase acceptance and a production release remain separate gates.

## Documentation — start here
| File | What it contains |
|---|---|
| [Current handoff](docs/RESUME_HANDOFF_2026-11-12.md) | Latest source, checks and next action |
| [PR reconciliation](docs/PR_RECONCILIATION_2026-11-13.md) | Complete draft inventory and preservation decisions |
| [Detailed phase plan](docs/PHASE_INDEX.md) | Execution sequence, dependencies and gates for P0–P6 |
| [Historical Phase 1 report](docs/PHASE1_REPORT_2026-10-26.md) | Dated 26 October implementation and CI evidence |
| [Kickoff report](docs/REPORT_2026-10-26.md) | Actual GitHub actions, evidence, blockers and next action |
| [Architecture decisions](docs/ADR_REGISTER.md) | Proposed ADR-001 to ADR-009, awaiting review |
| [Threat model](docs/THREAT_MODEL.md) | Attack surfaces, mitigations and tests mapped to phases |
| [Test strategy](docs/TEST_STRATEGY.md) | CI, OS matrix and release evidence plan |
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
**P0 review and P1/P2 development:** architecture/security decisions remain pending. Internal authentication, transactional metadata/jobs and isolated guest/component checks exist; public login, reviewed host adapters, signed installer and complete hosting acceptance remain unfinished. See [UPDATES.md](UPDATES.md) and the current integration PR for exact-head verification. There is no stable release.

## Installation
**Not available yet.** Please do not run unofficial install scripts claiming to be Tong Panel. A signed, versioned one-command installer for a tested OS matrix is a future requirement, not a current feature. Existing third-party shared hosting does not permit replacement of the provider's cPanel without root-level server administration.

## Source reuse and attribution
Tong Panel plans an original AGPL-3.0 codebase and compatible integrations to standard open-source server services. Upstream projects are researched for capability and design patterns, not automatically imported. Every dependency/code snippet needs source, licence, security and provenance review. cPanel and WHM are third-party trademarks and not affiliated with Tong Panel.

## Contributors
Read [AGENTS.md](AGENTS.md), [UPDATES.md](UPDATES.md) and [CONTRIBUTING.md](CONTRIBUTING.md) before starting or resuming work. Record true test status; do not treat design docs as working features.
