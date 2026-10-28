# P2 — Secure Single-Server Hosting Alpha
**Status:** IN PROGRESS. Component prototypes exist on unmerged draft PRs, but P2 is not an integrated or installable alpha. **Depends:** P1 auth, reviewed typed jobs/host agent, installer lab and secret handling.

## Implementation slices
1. Supported distro package/installer, preflight, journal and recovery.
2. Web service adapters: Nginx and PHP-FPM per site; runtime/version selectors only for tested versions.
3. Domains/aliases/subdomains, DNS ownership/preflight, redirects, vhost config and health checks.
4. Databases MariaDB/MySQL: scoped user provisioning/rotation, export/import and isolation.
5. Original secure Finder/Explorer-style files UI/API, quotas and metadata/ACL.
6. ACME certificates: HTTP-01 and DNS-01 provider capability, renewal, status, HTTPS redirects and real verification.
7. Cron, sites logs, versioned server templates; graceful service failure handling.
8. WordPress/pure PHP deploy recipe with pinned versions, checksums and restore.
9. Account backup schedules, offsite encrypted targets, retention and complete restore drills.
10. Multi-tenant adversarial testing and understandable UX/onboarding.

## Must-pass end-to-end tests
- Two sites and two separate tenants cannot access each other's data via API, files, DB users or job logs.
- Unsupported OS, low disk and occupied ports fail **before** destructive mutations.
- Existing certificates detected; issuance/retry/renewal and HTTP->HTTPS state verified externally.
- Atomic config staged/syntax checked, service reload failure rolls back.
- Backup to offsite, destroy lab VM, restore on clean VM and verify data and settings.
- Upload of malicious archive/symlink cannot escape jailed user paths.
- Login/MFA/recovery doesn't allow bypass across users; audit logs redact secrets.
- Basic phone/tablet site/domain/file tasks complete with guidance.

## Alpha release gate
Signed experimental artifacts for published exact OS/architecture matrix. Every limitation visible. Not marketed as production ready; email/DNS authoritative host and Node/Python deferred to P3/P4.


## Evidence-based implementation checkpoint — 29 October 2026
Verified component prototypes on unmerged drafts include Nginx/PHP-FPM renderers, two-tenant clean-VM HTTP/TLS checks, SQLite site/domain/database metadata, isolated MariaDB grant tests, one-time domain proof, authenticated loopback site/domain APIs and site alias create/remove. They are not wired into an installer or production host adapters. The full P2 acceptance below remains outstanding wherever lifecycle integration, rollback, data recovery, certificate issue/renew, UI or owner/security approval is required.

**Additional evidence:** PR #69 adds verified-domain preview filtering and PR #70 exposes the tenant-scoped read-only preview API; their exact-head Go, daemon, archive and Ubuntu/Debian VM checks pass. The API remains disabled without explicit server-injected paths and does not apply host configuration.

**Files UI checkpoint (draft PR #73):** a read-only accessible Explorer can navigate folders and download regular files through the tenant-scoped API. It uses same-origin credentials, safe text rendering, path/entry validation, no link following, a 16 MiB download cap and bounded requests. Local Node 20/20 and TypeScript 5.9 checks passed. Final PR head `b41b0bf09f9dac26e4c6920fe41ef875334a7028` passed Go/race/vet [#40767149470](https://github.com/parmtech/Tong-Panel/actions/runs/40767149470), UI [#40767149375](https://github.com/parmtech/Tong-Panel/actions/runs/40767149375), daemon [#40767149338](https://github.com/parmtech/Tong-Panel/actions/runs/40767149338), archive [#40767149427](https://github.com/parmtech/Tong-Panel/actions/runs/40767149427), and Ubuntu/Debian guests [#40767149428](https://github.com/parmtech/Tong-Panel/actions/runs/40767149428). Writes remain unavailable without an approved browser CSRF/session path. This does not change the phase status: installer/recovery, app-driven database/ACME lifecycle, backup restore, integrated two-tenant acceptance and P0 owner/security decisions remain open.


**Disposable package lab checkpoint (draft PR #74):** this clearly do-not-install unsigned Debian developer package passed its lifecycle guest drill on Ubuntu 24.04 [job 123106800802](https://github.com/parmtech/Tong-Panel/actions/runs/40770709002/job/123106800802) and Debian 12 [job 123106800515](https://github.com/parmtech/Tong-Panel/actions/runs/40770709002/job/123106800515). It tests install without automatic start, fail-closed auth, non-root loopback service, upgrade, downgrade, and data-preserving removal. It is not a supported installer or recovery guarantee. P2 stays IN PROGRESS; owner/security review, authentication/authorization integration, service adapters, DB lifecycle, ACME issuance/renewal, writable Files lifecycle, offsite backup/restore, end-to-end adversarial tests, and owner acceptance remain open.


**Site database metadata checkpoints (draft PRs #75/#77):** tenant-scoped list/create and per-record read endpoints derive ownership from the authenticated principal and generate database/user names server-side. No MariaDB database/user is created and no credential is stored or returned. Metadata API code head #75 `b574d2b8e13d83e709a748988810d32b18ec70ce` passed Go/race/vet [#40773296478](https://github.com/parmtech/Tong-Panel/actions/runs/40773296478), unsigned archive [#40773296582](https://github.com/parmtech/Tong-Panel/actions/runs/40773296582), daemon [#40773296715](https://github.com/parmtech/Tong-Panel/actions/runs/40773296715), and Ubuntu 24.04/Debian 12 guests [job 123115914599](https://github.com/parmtech/Tong-Panel/actions/runs/40773296680/job/123115914599), [job 123115914897](https://github.com/parmtech/Tong-Panel/actions/runs/40773296680/job/123115914897). Read API code head #77 `c990e4ed6b0a41377306cf4181ede635afa2c2ce` passed Go/race/vet [#40774254765](https://github.com/parmtech/Tong-Panel/actions/runs/40774254765), archive [#40774254706](https://github.com/parmtech/Tong-Panel/actions/runs/40774254706), daemon [#40774254763](https://github.com/parmtech/Tong-Panel/actions/runs/40774254763), and Ubuntu/Debian guests [job 123119250721](https://github.com/parmtech/Tong-Panel/actions/runs/40774254715/job/123119250721), [job 123119250373](https://github.com/parmtech/Tong-Panel/actions/runs/40774254715/job/123119250373). Privileged adapter review, provisioning/removal, credential custody/rotation, backup/restore and end-to-end acceptance remain outstanding; P2 stays IN PROGRESS.


**Disposable MariaDB export/restore checkpoint (draft PR #79, 29 October 2026):** the existing temporary tenant-grant check now dumps and restores tenant A's data after a controlled mutation, while checking tenant A cannot read, export, or import into tenant B's database. The exact code head `f63bac3b46de8775ae94032525b367e284b1ba74` passed [Ubuntu 24.04](https://github.com/parmtech/Tong-Panel/actions/runs/40781823676/job/123145135988) and [Debian 12](https://github.com/parmtech/Tong-Panel/actions/runs/40781823676/job/123145135759). This is database engine/grant evidence only; it is not app-managed provisioning, encrypted offsite backup, retention, key custody, or fresh-host restore. P2 remains IN PROGRESS.

- Initial guest runs reached the new dump/restore assertions but exposed a package-smoke `curl | grep -q` SIGPIPE failure (curl exit 23). The smoke now captures the local page before matching; the corrected exact code head passed both clean-VM jobs: [Ubuntu 24.04](https://github.com/parmtech/Tong-Panel/actions/runs/40781823676/job/123145135988) and [Debian 12](https://github.com/parmtech/Tong-Panel/actions/runs/40781823676/job/123145135759).


## P2 Nginx config rollback lab — draft [PR #80](https://github.com/parmtech/Tong-Panel/pull/80) (29 October 2026)

- Adds a disposable Ubuntu/Debian guest check around a separately launched temporary Nginx instance. Invalid staged syntax must leave the current config serving; a simulated reload failure must restore the previous config; a valid config must pass `nginx -t`, atomically replace the test include and reload successfully.
- The test never writes `/etc/nginx` or touches the guest's system Nginx service. Exact PR head `2cc5efdee5c03d252760d2e64c3d35ab3dba756b` passed P2 clean-VM run [#40783997959](https://github.com/parmtech/Tong-Panel/actions/runs/40783997959): [Ubuntu 24.04 job 123152504585](https://github.com/parmtech/Tong-Panel/actions/runs/40783997959/job/123152504585) and [Debian 12 job 123152504984](https://github.com/parmtech/Tong-Panel/actions/runs/40783997959/job/123152504984). The Debian output confirms the raced logout denial test and Nginx rollback smoke passed. This is an isolated config lifecycle prototype only; the daemon still does not apply host configuration. P2 remains IN PROGRESS.

- Initial PR #80 Debian CI exposed a race in the existing logout test's status expectation: the loser can receive 401 if revocation precedes authentication, or 403 if it occurs between authentication and the CSRF check. The contract is one 204 winner, one denied loser, and exactly one audit event; the test now asserts that contract.
