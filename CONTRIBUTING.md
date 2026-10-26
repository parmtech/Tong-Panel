# Contributing to Tong Panel

Tong Panel is an AGPL-3.0-licensed open-source project from ParmTech. Its aim is safe and accessible Linux web hosting administration.

## Start here
Read AGENTS.md, UPDATES.md, docs/ROADMAP.md, docs/SECURITY.md and docs/UPSTREAM_RESEARCH.md. The project is currently in planning; new functional code must follow approved scope and architecture decisions.

## Development workflow
1. Link an issue/task, acceptance criteria and phase.
2. Work on a dedicated branch and narrow changes.
3. Explain security, server compatibility, licence sources and rollback plan.
4. Test code changes; never report tests that were not run.
5. Add a UPDATES.md journal entry and adjust docs.
6. Submit a PR for review; do not overwrite other agents' work.

## Pull request checklist
- [ ] Issue/owner/phase + user-visible acceptance
- [ ] Tests and results, or docs-only designation
- [ ] Role, tenant, file-system and privileged-operation security reviewed
- [ ] Fresh-host compatibility and possible operator lockout assessed
- [ ] Provenance/copyright and licence compatibility recorded
- [ ] Upgrade, rollback and data-preservation impact recorded
- [ ] UPDATES.md updated, no private credentials/data

## Responsible reporting
Do not post exploit details publicly. Use private GitHub vulnerability reporting if enabled; a maintainer should publish a verified security contact and response process before public release.

## Brand and contributions
Keep original UI/icons. Do not copy cPanel copyrighted assets or third-party commercial/pro edition source. Ensure you have the rights to contribute material under the project's AGPL-3.0 licence; declare any third-party components and their notices.
