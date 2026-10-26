# P3 — Modern Application Platform Beta
**Status:** PLANNED. **Depends:** P2 secure site routing, auth/quotas, backup and deploy jobs.

## Modules
1. Versioned runtimes for supported Node.js LTS and Python releases (pin per app; benchmark/compatibility test).
2. Git-based deploy using restricted deploy keys or app OAuth; webhook signatures verified; never leak repo tokens.
3. Native process manager under dedicated site UID with systemd sandbox and tightly-scoped resources.
4. Isolated build runners with CPU/RAM/disk/time/network limits, artifact integrity and abort/retry semantics.
5. Python Django/Flask/FastAPI WSGI/ASGI and virtualenv; app/log/perf checks.
6. Node Express/Next.js/other supported profiles with build/start scripts, reverse proxy, reserved ports.
7. Secrets management/rotation, per-app environment variables, masking and audit.
8. Deployment history, health checks, rollback to pinned artifacts, clear status/logs.
9. Optional rootless-container execution with no Docker socket exposure, image digest checks, restricted network/volumes.
10. Curated signed app marketplace: CMS, collaboration, ecommerce, monitoring and developer recipes, with licences and resource requirements.

## Acceptance scenarios
- Node app from Git + custom domain/SSL and verified route; upgrade/rollback in lab.
- Python ASGI app and dependency version isolation without host package pollution.
- Two hostile sample apps cannot read each other's secrets, process spaces, backups or file trees.
- Builds hitting resource limits fail gracefully and leave host healthy.
- Unverified public registry tags or untrusted app definitions cannot install by default.
- App recipe removal does not delete user-created data without explicit backup/export choice.

## Exit
Curated beta for tested application types only. Marketplace and broad framework support are compatibility lists backed by tests, not promises.
