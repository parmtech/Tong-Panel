> **Historical preservation snapshot:** This is a redacted copy of local README.md; it is not the canonical file. Private machine paths and conversation metadata were omitted. Claims are historical and require verification; no approval is implied.

# Tong Panel
**A project to build a modern, friendly Linux web hosting control panel for self-managed servers.**

**Publisher:** [ParmTech](https://parmtech.au)
**Project:** [Tong Panel](https://github.com/parmtech/Tong-Panel)
**License:** [GNU AGPL-3.0](LICENSE) for original Tong Panel code
**Current status:** **Phase 0 architecture/security planning remains in progress; Phase 1 code is experimental, unmerged and not a functioning web-hosting control panel or installer.**

## What we are building
A capable but easy-to-use hosting dashboard intended for self-managed Linux VPS and dedicated servers. Candidate OS profiles and any support promise remain unselected and unvalidated; no distribution is certified as supported. Our design combines conventional hosting tools (web domains, PHP, mail, DNS, database, SSL, backups, resellers) with modern app deployment (Node.js, Python, Git-based apps, isolated containers), without copying proprietary cPanel designs or software.

- **Simple, familiar UX:** responsive tablet-like icon grid; global search; desktop-like file explorer with tree/list/grid, multi-select, drag/drop and preview; accessible light/dark modes; tours and contextual docs.
- **Websites and apps:** PHP, WordPress, Laravel, static sites, Node.js, Python and eventually more; environment variables, safe deployments, logs and rollbacks.
- **Hosting management:** domains and redirects, SSL/ACME, MariaDB/PostgreSQL, email, DNS, backups, file manager, cron, SSH/SFTP, users and quotas.
- **Server and developer operations:** service health, hardening, audit, migrations, API/CLI, app marketplace and future multi-server.
- **Free software:** first-party core functions remain open source. External servers, cloud services, optional infrastructure and third-party applications may incur their own costs.

This is the *target blueprint*, not a list of completed functions.

## Development status and phase plan
The seven-phase execution plan is in [PHASE_INDEX](docs/PHASE_INDEX.md) with individual P0–P6 specifications. [ADR_REGISTER](docs/ADR_REGISTER.md) contains draft ADR-001–008; a separate ADR-009 authentication proposal exists only on an unmerged Phase 1 branch. The [THREAT_MODEL](docs/THREAT_MODEL.md) and [TEST_STRATEGY](docs/TEST_STRATEGY.md) also remain proposals. P0 decisions and independent review are not recorded in the reviewed project sources.

[Private conversation/coordination metadata omitted.]

## Documentation — start here
| File | What it contains |
|---|---|
| [Detailed phase plan](docs/PHASE_INDEX.md) | Execution sequence, dependencies and gates for P0–P6 |
| [Phase 1 development report](docs/PHASE1_REPORT_2026-10-26.md) | Historical 2026-10-26 prototype report; see [Latest Progress](UPDATES.md) for newer status |
| [Kickoff report](docs/REPORT_2026-10-26.md) | Historical 2026-10-26 kickoff snapshot |
| [Architecture decisions](docs/ADR_REGISTER.md) | Draft ADR-001–008; outcomes and ADR-009 indexing remain pending review |
| [Threat model](docs/THREAT_MODEL.md) | Attack surfaces, mitigations and tests mapped to phases |
| [Test strategy](docs/TEST_STRATEGY.md) | CI, OS matrix and release evidence plan |
| [Master Blueprint](docs/BLUEPRINT.md) | Product vision, end-to-end feature definition, user journeys and differentiators |
| [Feature Matrix](docs/FEATURE_MATRIX.md) | Detailed functional modules, priorities and acceptance criteria |
| [Architecture](docs/ARCHITECTURE.md) | Proposed control plane, privileged agent, database/jobs, security boundaries |
| [Roadmap](docs/ROADMAP.md) | Milestones P0–P6 and exit conditions |
| [UX/UI Specification](docs/UX_UI_SPEC.md) | Launcher-style home, Explorer/Finder-like file manager, accessibility and tours |
| [Installer Specification](docs/INSTALLER_SPEC.md) | Proposed fresh-Linux setup, one-command goal, recovery and update strategy |
| [Security](docs/SECURITY.md) | Threat model, privilege separation, safe file handling, release gates |
| [Open-source Research](docs/UPSTREAM_RESEARCH.md) | HestiaCP, Virtualmin, ISPConfig, 1Panel, Coolify and third-party licence cautions |
| [Agent Handover](AGENTS.md) | How a developer/AI agent can resume work safely |
| [Latest Progress](UPDATES.md) | What is **actually** complete, blockers, history and exact next step |
| [Local P0 reconciliation draft](docs/p0-review/p0-phase0-reconciliation-2026-11-11.md) | Evidence crosswalk, open decisions and blank review templates; not published |
| [Contributing](CONTRIBUTING.md) | Pull request, testing and review expectations |

## Current milestone
**P0 planning; existing P1 code remains experimental:** architecture and threat decisions, P1 scope/support assumptions, qualified provenance review, and P1 design/code go/no-go remain unrecorded in the reviewed project sources. **Next:** route the P1 readiness-packet refresh to its owner once its publication/location is verified and an exact reachable path/ref is supplied. Proposal-only owner packets for P0-01/03/04/09 and PR #160 are in the [P0 reconciliation](docs/p0-review/p0-phase0-reconciliation-2026-11-11.md); authorized owners must review/apply any corrections in their files. Each decision still needs its own applicable evidence and required reviewers. No P0 exit, P0-to-P1 transition, or P1 promotion approval is recorded. No stable release, approved installer, or production-ready panel exists. See [UPDATES.md](UPDATES.md) for the current checkpoint.

## Installation
**Not available yet.** Please do not run unofficial install scripts claiming to be Tong Panel. A signed, versioned one-command installer for a tested OS matrix is a future requirement, not a current feature. Existing third-party shared hosting does not permit replacement of the provider's cPanel without root-level server administration.

## Source reuse and attribution
Tong Panel plans an original AGPL-3.0 codebase and compatible integrations to standard open-source server services. Upstream projects are researched for capability and design patterns, not automatically imported. Every dependency/code snippet needs source, licence, security and provenance review. cPanel and WHM are third-party trademarks and not affiliated with Tong Panel.

## Contributors
Read [AGENTS.md](AGENTS.md), [UPDATES.md](UPDATES.md) and [CONTRIBUTING.md](CONTRIBUTING.md) before starting or resuming work. Record true test status; do not treat design docs as working features.
