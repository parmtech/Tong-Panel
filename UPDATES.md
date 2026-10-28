Warning: truncated output (original token count: 23341)
Total output lines: 499

# Tong Panel — Live Project Status and Work Journal
**Updated:** 2026-10-29 (Sydney)
**Project:** ParmTech Tong Panel (https://github.com/parmtech/Tong-Panel)  
**Maintained by:** any authorized contributor/agent who commits changes.  
**Rule:** This file is a **manual committed source of truth**, not an automatic background process. Every meaningful work session should update checkpoint and log.

## CURRENT CHECKPOINT — P2 web, database and HTTPS foundations (29 October 2026)
- **P0/P1 gates:** ADR-001–009 and threat-model owner/security decisions remain pending; the existing P0 engineering review is recommendations by a non-owner, not approval. Public production login, signed installer, supported install/rollback/recovery, UI acceptance and Phase 1 exit remain open.
- **PR #60:** draft Nginx vhost renderer. Exact code head `098a1bbf38108e3c0ea03e0ffac0536674b753c7` passed Go/race/vet [#40730244500](https://github.com/parmtech/Tong-Panel/actions/runs/40730244500/job/122964280023), daemon [#40730244475](https://github.com/parmtech/Tong-Panel/actions/runs/40730244475/job/122964279475), unsigned archive [#40730244578](https://github.com/parmtech/Tong-Panel/actions/runs/40730244578/job/122964279647), and Ubuntu/Debian guests [#40730244361](https://github.com/parmtech/Tong-Panel/actions/runs/40730244361), including real `nginx -t`.
- **PR #61:** draft PHP-FPM pool renderer. Exact code head `c6519adddd48a64a0616ff0414a6c60dd48deb51` passed Go/race/vet [#40733876763](https://github.com/parmtech/Tong-Panel/actions/runs/40733876763), daemon [#40733876869](https://github.com/parmtech/Tong-Panel/actions/runs/40733876869), archive [#40733876901](https://github.com/parmtech/Tong-Panel/actions/runs/40733876901), and Ubuntu 24.04/Debian 12 parser/Nginx checks [#40733876821](https://github.com/parmtech/Tong-Panel/actions/runs/40733876821).
- **PR #62:** draft two-tenant HTTP smoke. Exact executable head `3b970d5dc722399d08eef87b8e0e27717b1ae126` passed Go/race/vet [#40735348866](https://github.com/parmtech/Tong-Panel/actions/runs/40735348866), daemon [#40735348619](https://github.com/parmtech/Tong-Panel/actions/runs/40735348619), unsigned archive [#40735348622](https://github.com/parmtech/Tong-Panel/actions/runs/40735348622), and both Debian 12 and Ubuntu 24.04 guest tests [#40735348587](https://github.com/parmtech/Tong-Panel/actions/runs/40735348587). Each guest ran two real Nginx/PHP-FPM sites and verified private-runtime cross-tenant denial and hidden-file denial.
- **PR #63:** draft SQLite site-database metadata catalog, stacked on #62. Corrected code head `2b69846209b88998e059499755cc85cd4e924b78` passed Go/race/vet [#40737721111](https://github.com/parmtech/Tong-Panel/actions/runs/40737721111), SQLite schema [#40737721769](https://github.com/parmtech/Tong-Panel/actions/runs/40737721769), daemon [#40737721131](https://github.com/parmtech/Tong-Panel/actions/runs/40737721131), archive [#40737721452](https://github.com/parmtech/Tong-Panel/actions/runs/40737721452), and Ubuntu/Debian clean VM smoke [#40737721348](https://github.com/parmtech/Tong-Panel/actions/runs/40737721348). CI-exposed formatting, schema assertion and SQLite writer-race issues were corrected.
- **PR #64:** draft disposable two-tenant MariaDB isolation check, stacked on #63. Exact head `52e64adc70cf512cc0381c07c4b871a9d0ffd1b7` passed Debian 12 and Ubuntu 24.04 clean VMs [#40740998539](https://github.com/parmtech/Tong-Panel/actions/runs/40740998539). Each guest installed MariaDB, verified each temporary least-scope user's own DB reads and denied cross-tenant reads, then removed temporary DBs/users.
- **PR #65:** draft HTTPS vhost renderer, stacked on #64. Exact code head `cf6aeb93793a4359f69b1a4d07a869a226cb5a90` passed Go/race/vet [#40746211175](https://github.com/parmtech/Tong-Panel/actions/runs/40746211175), daemon [#40746211241](https://github.com/parmtech/Tong-Panel/actions/runs/40746211241), unsigned archive [#40746211152](https://github.com/parmtech/Tong-Panel/actions/runs/40746211152), and both Ubuntu 24.04/Debian 12 clean guests [#40746211249](https://github.com/parmtech/Tong-Panel/actions/runs/40746211249). The clean guest caught a missing explicit HTTP site root that broke ACME challenge serving on Debian; the renderer and regression test now set/check the verified root. The smoke uses temporary self-signed certs and does not issue/renew ACME certs.
- **P2 remains incomplete:** these drafts provide vhost, PHP-FPM and isolation foundations. The panel still lacks application-driven MariaDB provisioning and secret rotation/export, ACME issue/renew lifecycle, installer/preflight/rollback, cron/WordPress flows, encrypted offsite backup and fresh-VM restore, usable UI and full two-tenant end-to-end acceptance. No real host changed.

## VERIFIED P1 CHECKPOINT — 2026-10-28, trusted browser boundary
- **Phase:** P0/ADR review remains pending; P1 development continues on isolated drafts. Phase 1 is not complete or production-ready.
- **Current stacked slice:** new branch `feature/p1-cookie-origin-duplicate-guard`, based on draft PR #33 (`feature/p1-session-idle-rotation`). It rejects ambiguous duplicate `Origin` headers for cookie-authenticated mutations and adds a concurrent authenticated-read test. No public login, cookie issuance, logout route, proxy trust, or server deployment is enabled.
- **Local validation:** Go is not installed in this Windows workspace. Corrected code head `6c983a22de1e5b1ea0de133ea28c6d1daadc490c` passed hosted Go/race/vet, daemon smoke, and archive workflows; see run links below.
- **P0/auth decision:** `docs/ADR-009-AUTHENTICATION.md` is explicitly proposed and unapproved; `docs/ADR_REGISTER.md` marks architecture decisions as draft and requires owner/security review. Do not add a public login handler until that review is recorded.
- **Release blockers:** owner/security approval, external key custody and recovery, public HTTPS login/origin topology, trusted-agent review, signed installer/SBOM, clean Ubuntu/Debian VM install/rollback/recovery, and UI accessibility/E2E remain open. No installer currently exists; no production VPS was changed.
- **Verified code head:** `6c983a22de1e5b1ea0de133ea28c6d1daadc490c` passed Go checks [#40599907640](https://github.com/parmtech/Tong-Panel/actions/runs/40599907640) (gofmt, full `go test -race ./...`, vet), non-destructive daemon smoke [#40599907840](https://github.com/parmtech/Tong-Panel/actions/runs/40599907840), and unsigned developer archive lab [#40599907588](https://github.com/parmtech/Tong-Panel/actions/runs/40599907588). These validate this code head; they do not certify a release or security approval.
- **Authentication boundary:** owner/security review must record ADR-009/P0 decisions before a public login route; keep public login and production deployment disabled. Isolated non-privileged prototypes can continue.

## CURRENT P2 SLICE — cross-process Files mutation barrier
- **Branch/PR:** `feature/p2-quota-maintenance-barrier`, draft [PR #56](https://github.com/parmtech/Tong-Panel/pull/56), stacked on #55. Verified current code head `817c73d92b1a8b6e827e212750e4708167b3e9ae`: Go format/full race/vet [#40718460289](https://github.com/parmtech/Tong-Panel/actions/runs/40718460289), daemon [#40718460274](https://github.com/parmtech/Tong-Panel/actions/runs/40718460274), unsigned package [#40718460273](https://github.com/parmtech/Tong-Panel/actions/runs/40718460273), and Ubuntu 24.04/Debian 12 clean VMs [#40718460480](https://github.com/parmtech/Tong-Panel/actions/runs/40718460480) all pass.
- **Change/tests:** Linux service-owned per-tenant shared/exclusive file locks cover authenticated upload/delete through quota finalization; quota reconciliation requires injected external-writer quiescence, drains participating API writers, bounded-scans the pinned root and transactionally clears stale reservations. Locks pin the directory descriptor, use descriptor-relative lock opens, and reject unsafe ownership/modes. Tests cover subprocess exclusion, mutation drain, tenant isolation, cancelled waits, close lifecycle, unsafe directories and fail-closed recovery. Clean Ubuntu/Debian guests execute API and Files tests.
- **Limits/status:** recovery is not enabled in daemon/startup and there is no production external-writer quiescer. The API barrier does not stop PHP/WordPress, shell or other nonparticipating writers. No installer, signed release, restore drill, P2 completion, owner approval or production change is claimed.
- **Next:** proceed with an unprivileged site/domain lifecycle prototype; keep host mutation gated on owner/security decisions.

## P0 technical review prepared for owner decision
- ADR register, ADR-009, threat model, roadmap, feature matrix and security docs were reviewed. Issues #1 and #2 have no comments recording an owner/security decision as of 2026-10-29.
- Recommendations are recorded in [P0 engineering review](docs/P0_ENGINEERING_REVIEW_2026-10-29.md). This is a non-owner engineering assessment, not owner acceptance or an independent security audit. ADR-007/008 and ADR-009 remain blockers for privileged host actions and public login.
- No public login, host mutation, install, deployment or security approval claim was made.

## CURRENT P2 SLICE — one-time DNS domain proof (draft PR #59)
- **Branch/PR:** `feature/p2-domain-dns-verification`, draft [PR #59](https://github.com/parmtech/Tong-Panel/pull/59), stacked on #58. Exact executable head `87686d4e8956469727afd9f17194e861a53f503f` passed Go format/full race/vet [#40724519503](https://github.com/parmtech/Tong-Panel/actions/runs/40724519503), daemon [#40724519369](https://github.com/parmtech/Tong-Panel/actions/runs/40724519369), unsigned archive [#40724519405](https://github.com/parmtech/Tong-Panel/actions/runs/40724519405), SQLite schema [#40724519582](https://github.com/parmtech/Tong-Panel/actions/runs/40724519582), and Ubuntu 24.04/Debian 12 guests [#40724519430](https://github.com/parmtech/Tong-Panel/actions/runs/40724519430).
- **Change/tests:** schema v10 stores one-hour proof token digest/expiry; random 256-bit token is returned once and must appear as exact TXT at `_tong-panel-challenge.<domain>`. Success consumes token and moves pending claim to verified. Tests cover digest-at-rest, wrong token before DNS query, wrong/missing DNS, expiry, tenant denial, reissue, and concurrent one-winner replay. Daemon/bootstrap smoke now expect schema v10.
- **Limits:** no API route, domain deletion, DNSSEC/authoritative guarantee, ACME/certificate provisioning, site/DB lifecycle, host adapter, installer or restore. No real TXT lookup was performed by CI; tests inject lookup results. P0 review remains pending; P2 incomplete; no production host changed.
- **Next:** add certificate/site lifecycle only behind explicit ownership-verified preconditions and reviewed typed host-agent contract.

## PRIOR P2 SLICE — persistent site/domain catalog (draft PR #58)
- SQLite schema v9 stores sites and globally unique pending domain claims. Site/domain creation is transactional; tenant-scoped reads and concurrent two-connection collision are tested.
- Exact head `3d3b9205f13046e69d5f524c788c2482db56d3b7` passed Go/race/vet [#40722666219](https://github.com/parmtech/Tong-Panel/actions/runs/40722666219), daemon [#40722404529](https://github.com/parmtech/Tong-Panel/actions/runs/40722404529), archive [#40722404458](https://github.com/parmtech/Tong-Panel/actions/runs/40722404458), SQLite [#40722666354](https://github.com/parmtech/Tong-Panel/actions/runs/40722666354), and Ubuntu/Debian VMs [#40722666297](https://github.com/parmtech/Tong-Panel/actions/runs/40722666297).
- No domain proof or host lifecycle is implemented.

## PRIOR VERIFIED P2 SLICE — measured tenant Files quota setup
- **Branch/PR:** `feature/p2-quota-reconciliation`, draft [PR #55](https://github.com/parmtech/Tong-Panel/pull/55), stacked on #54; code head `2a56d93d250b4ad352831da97db6a7a8b0a8c134`.
- **Change/tests:** bounded Linux pinned-root scan counts nested regular-file logical bytes, never follows symlinks or crosses mounts, serializes against mutations on the same `Root`, and fails closed on entry/depth limits, special files, detected path changes and overflow. `ConfigureTenantQuotaFromRoots` records measured usage and still refuses pending reservations.
- **Validation:** code head passed Go formatting/full race/vet [#40711216528](https://github.com/parmtech/Tong-Panel/actions/runs/40711216528), daemon smoke [#40711216435](https://github.com/parmtech/Tong-Panel/actions/runs/40711216435), unsigned package [#40711216395](https://github.com/parmtech/Tong-Panel/actions/runs/40711216395), and Ubuntu 24.04/Debian 12 clean VM smoke [#40711216498](https://github.com/parmtech/Tong-Panel/actions/runs/40711216498). Documentation head `993cc56e119f6a1b2c357a5e8b07d3ac5c7e6975` also passed [Go #40712342446](https://github.com/parmtech/Tong-Panel/actions/runs/40712342446), [daemon #40712342583](https://github.com/parmtech/Tong-Panel/actions/runs/40712342583), [package #40712342394](https://github.com/parmtech/Tong-Panel/actions/runs/40712342394), and both clean guests [#40712342367](https://github.com/parmtech/Tong-Panel/actions/runs/40712342367). Local Go/gofmt unavailable.
- **Limits/status:** callers must quiesce writers outside the same `Root`; this is not a cross-process startup barrier and does not recover crash-stranded reservations. No daemon wiring, Files UI/object ACL, installer/service lifecycle, backup/restore or owner/security approval. P2 remains incomplete; no production host changed.
- **Next:** design and test a cross-process maintenance barrier that drains live uploads before scanned usage can safely clear stale reservations.

### 2026-10-29 — P2 cross-process file mutation barrier (in progress)
- **Branch/PR:** `feature/p2-quota-maintenance-barrier`, draft [PR #56](https://github.com/parmtech/Tong-Panel/pull/56), based on PR #55 head `993cc56e119f6a1b2c357a5e8b07d3ac5c7e6975`; full recovery head `3d09fe01001fbbcc8a19229af5a86e4c40a3c5e7` passed [Go #40717422190](https://github.com/parmtech/Tong-Panel/actions/runs/40717422190), [daemon #40717422469](https://github.com/parmtech/Tong-Panel/actions/runs/40717422469), [package #40717422494](https://github.com/parmtech/Tong-Panel/actions/runs/40717422494), and Ubuntu 24.04/Debian 12 clean guests [#40717422346](https://github.com/parmtech/Tong-Panel/actions/runs/40717422346).
- **Change:** add Linux per-tenant shared/exclusive advisory barrier for authenticated Files upload/delete through accounting; add `ReconcileTenantQuotaFromRoots`, which requires external writer quiescence, drains API writers, scans the pinned root and transactionally clears crash-stranded reservations.
- **Tests:** Local Go/gofmt unavailable. Two initial hosted Go checks failed solely because final blank lines were not normalized. Full recovery head `3d09fe01001fbbcc8a19229af5a86e4c40a3c5e7` passed Go format/full race/vet, daemon, package and both clean-VM API+Files suites. Ubuntu fixture lock-directory mode was explicitly set to 0700 after its first failure. Current descriptor-pinning hardening is pending CI.
- **Limits/security:** lock directory must be service-owned and not group/other writable. The API barrier coordinates participating Tong Panel processes only; a host-level quiescer adapter and daemon/startup wiring are still absent, and external writers must not be assumed quiesced. No owner/security approval or production host change. Phase 2 remains incomplete.
- **Next:** inspect and execute exact PR-head CI; continue toward safe startup recovery and the remaining P2 roadmap gates.

## PRIOR VERIFIED P2 SLICE — tenant Files deletion accounting
- **Branch/PR:** `feature/p2-files-delete-accounting`, draft [PR #54](https://github.com/parmtech/Tong-Panel/pull/54), stacked on #53; verified code head `4ff8529f347dd2f2dac81523852d6a6790016cf1`.
- **Change/tests:** authenticated DELETE requires `files:write`, strict path validation and existing CSRF/origin middleware. Removal atomically moves the actual entry into a locked private stage before measuring and unlinking it, so a concurrent symlink swap cannot corrupt returned byte accounting. SQLite used bytes decrement only after directory sync; symlinks count as zero and underflow fails closed.
- **Validation:** Go formatting/full race/vet [#40708184553](https://github.com/parmtech/Tong-Panel/actions/runs/40708184553), daemon smoke [#40708184530](https://github.com/parmtech/Tong-Panel/actions/runs/40708184530), unsigned package [#40708184387](https://github.com/parmtech/Tong-Panel/actions/runs/40708184387), and Ubuntu 24.04/Debian 12 clean VM smoke [#40708184391](https://github.com/parmtech/Tong-Panel/actions/runs/40708184391) passed on the exact code head. Earlier gofmt and seeded-session test failures were corrected before these green runs.
- **Limits/status:** crash after unlink but before quota update can conservatively overcount; trusted scan/startup reconciliation is still absent. No Files UI/object ACL or daemon wiring, installer/service lifecycle, backup/restore evidence or owner/security review. P2 remains incomplete; no production host changed.
- **Next:** add trusted tenant-root usage reconciliation and startup recovery for pending reservations before daemon quota wiring.

## PRIOR VERIFIED P2 SLICE — persistent tenant Files quota reservations
- **Branch/PR:** `feature/p2-sqlite-files-quota`, draft [PR #53](https://github.com/parmtech/Tong-Panel/pull/53), stacked on #52; final code head `f9a67c534bc3276b3f04037dbc94f67b1043392b`.
- **Change/tests:** schema v8 adds tenant-bound quota state and durable upload reservations. Reserve/commit/release counters are transactional, arithmetic is overflow-safe, and quota configuration refuses pending reservations. Tenant quota rows reference existing tenants. Tests cover unconfigured/over-quota denial, actual-byte commits, release, tenant isolation, zero-byte reservations, restart persistence, migrations, and concurrent admission through two independent SQLite handles.
- **Validation:** Go formatting/full race/vet [#40703478052](https://github.com/parmtech/Tong-Panel/actions/runs/40703478052), SQLite schema [#40703478117](https://github.com/parmtech/Tong-Panel/actions/runs/40703478117), daemon/bootstrap [#40703478331](https://github.com/parmtech/Tong-Panel/actions/runs/40703478331), unsigned package [#40703477994](https://github.com/parmtech/Tong-Panel/actions/runs/40703477994), and Ubuntu 24.04/Debian 12 clean VM smoke [#40703477997](https://github.com/parmtech/Tong-Panel/actions/runs/40703477997) passed on this exact code head. Local Go/gofmt unavailable.
- **Limits/status:** quota configuration accepts a trusted measured usage value but does not scan tenant roots. Reservations stranded by a process crash fail closed and have no automatic reconciliation yet. Files UI/object ACL, daemon wiring, installer/service lifecycle, offsite backup and clean-host restore remain open. ADR/threat-model review remains pending; P2 incomplete; no production host changed.
- **Next:** implement safe startup reconciliation and recover pending reservations before wiring the daemon.

## PRIOR VERIFIED P2 SLICE — quota-gated authenticated Files uploads
- **Branch/PR:** `feature/p2-files-write-boundary`, draft [PR #52](https://github.com/parmtech/Tong-Panel/pull/52), stacked on #51/#50. Corrected code head `c63709af69c29d01e0ed683ac1d8ea04cc9d9a6e`.
- **Change/tests:** POST create-only uploads require the verified principal's same-tenant `files:write` role and both an explicitly injected Files service and quota reservation service. Only `application/octet-stream` and one validated relative `path` are accepted; per-file input is capped at 16 MiB. Uploads use the existing bounded stream/stage/no-replace publication, reserve before consuming the body, commit actual bytes after publish, and release on failed writes. If directory durability or quota commit is uncertain after publication, reservation accounting is retained. Auditor/unknown roles are denied.
- **Validation:** Go formatting/full `go test -race ./...`/vet [#40622572789](https://github.com/parmtech/Tong-Panel/actions/runs/40622572789), daemon smoke [#40622572813](https://github.com/parmtech/Tong-Panel/actions/runs/40622572813), unsigned package lab [#40622572868](https://github.com/parmtech/Tong-Panel/actions/runs/40622572868), and Ubuntu 24.04/Debian 12 disposable VM smoke [#40622572815](https://github.com/parmtech/Tong-Panel/actions/runs/40622572815) passed on the code head. Local Go unavailable.
- **Limits/status:** quota is still only an injected contract; no durable cross-worker ledger/reconciliation or delete accounting exists. No Files UI, object ACL, daemon service wiring, installer, service lifecycle or restore is provided; endpoint remains disabled unless callers explicitly supply services. ADR/threat-model review remains pending. P2 incomplete; no production host changed.
- **Follow-up:** persistent SQLite reservation accounting is in draft PR #53; trusted filesystem reconciliation and removal accounting remain open.

## PRIOR VERIFIED P2 SLICE — authenticated tenant-scoped Files read API
- Draft [PR #51](https://github.com/parmtech/Tong-Panel/pull/51), stacked on #50, adds opt-in list/read handlers behind protected middleware. Tenant scope comes from the verified principal; Owner/Admin/Reseller/Customer are allowed and Auditor/unknown roles denied. It rejects duplicate/unknown query keys, traversal and symlink escapes and caps lists to 1,000 entries and reads to 16 MiB.
- Corrected code head `01be544dca165c9554e5f429bc3029c0015ca009` passed Go/race/vet [#40621248079](https://github.com/parmtech/Tong-Panel/actions/runs/40621248079), daemon [#40621248030](https://github.com/parmtech/Tong-Panel/actions/runs/40621248030), and unsigned package [#40621248033](https://github.com/parmtech/Tong-Panel/actions/runs/40621248033); docs head `437795d4533e9565b58763f7726516736179f3ee` also passed all three workflows. The daemon does not configure the Files service.
## PRIOR VERIFIED P2 SLICE — fail-closed pre-mutation gate
- **Branch/PR:** `feature/p2-mutation-preflight-gate`, draft [PR #50](https://github.com/parmtech/Tong-Panel/pull/50), stacked on #49/#48/#47. Code head `55544dcb3a6505a4967fa67ad44046e6111d1b0b` passed Go/race/vet [#40619972006](https://github.com/parmtech/Tong-Panel/actions/runs/40619972006), daemon [#40619972004](https://github.com/parmtech/Tong-Panel/actions/runs/40619972004), and unsigned package [#40619972005](https://github.com/parmtech/Tong-Panel/actions/runs/40619972005).
- The pure evaluator blocks unsupported OS, invalid or missing facts, low/unknown disk, occupied/unknown ports and incomplete policy. It remains read-only and is not connected to an installer; no rollback or recovery is proven. ADR/threat-model review remains pending and P2 is incomplete.
- **Next:** implement only explicitly reviewed non-privileged work until owner/security decisions unblock host mutation.
## PRIOR VERIFIED P2 SLICE — root-close-safe bounded file uploads
- **Branch/PR:** `feature/p2-stream-close-safe`, draft [PR #42](https://github.com/parmtech/Tong-Panel/pull/42), stacked on draft #41/#40/#39/#38/#37/#36/#35. Executable code commit: `faa220eed997f68b225dc105cc4386227d742dfe`.
- **Change:** `WriteNewFrom` duplicates the pinned root descriptor under the lifecycle lock, releases that lock before consuming a potentially slow reader, and uses the duplicate through bounded staging and publication. `Close` prevents new operations but no longer waits for a blocked stream; an already-started operation can safely finish on its descriptor copy.
- **Focused tests:** gated slow reader proves Close returns promptly, staged write completes after root close, and subsequent reads fail closed. Existing upload bound, error cleanup, same-target race, and tenant isolation tests remain.
- **Validation:** exact code commit `faa220eed997f68b225dc105cc4386227d742dfe` passed gofmt/full `go test -race ./...`/vet [#40613452838](https://github.com/parmtech/Tong-Panel/actions/runs/40613452838), daemon smoke [#40613452810](https://github.com/parmtech/Tong-Panel/actions/runs/40613452810), and unsigned package lab [#40613452879](https://github.com/parmtech/Tong-Panel/actions/runs/40613452879).
- **Limits/status:** an in-flight caller still owns/cancels its reader and private stage; there is no request deadline, global concurrency admission, tenant disk quota, HTTP/auth/ACL, installer or host lifecycle. P0/P1 remain unapproved; P2 incomplete and no live host changed.
- **Next:** continue only safe internal file work; API/UI, domain/site/database/SSL services and install/restore require reviewed P0/P1 boundaries.

## PRIOR VERIFIED P2 SLICE — bounded streaming create-only file upload
- **Branch/PR:** `feature/p2-stream-file-to-stage`, draft [PR #41](https://github.com/parmtech/Tong-Panel/pull/41), stacked on draft #40/#39/#38/#37/#36/#35. Executable code commit: `df450a054fe8df50333e8e820f4145601e8d75f8`.
- **Scope:** reworks `Root.WriteNewFrom` to stream directly into a private mode-0700 stage with constant application memory, consuming at most the configured cap+1 bytes (cap max 16 MiB). It publishes via existing mode-0600 atomic no-replace rename only after successful EOF within the cap. The tenant selector resolves before stream read; `WriteNew` shares the same staging path.
- **Focused negative/race tests:** exact limit, cap+1 overflow without publication/stage leak, invalid path/limit/nil reader, reader failure without partial publication, unknown tenant/closed root before consumption, concurrent same-destination single winner, and `Close` waits through stream publication.
- **Validation:** executable commit `df450a054fe8df50333e8e820f4145601e8d75f8` passed gofmt/full `go test -race ./...`/vet [#40612826353](https://github.com/parmtech/Tong-Panel/actions/runs/40612826353), daemon smoke [#40612826486](https://github.com/parmtech/Tong-Panel/actions/runs/40612826486), and unsigned package lab [#40612826389](https://github.com/parmtech/Tong-Panel/actions/runs/40612826389). No local Go toolchain is available.
- **Limits/status:** this avoids per-request upload buffering but adds no global concurrency limit, tenant disk quota, request deadline, HTTP/API/ACL/auth, replacement semantics, scanner, startup recovery or installation. P0/P1 gates remain open; P2 is incomplete and no real host changed.
- **Next:** complete remaining Files/hosting work only within reviewed identity, ACL and host-agent boundaries; full API/UI, installer and lifecycle acceptance remain.

## PRIOR VERIFIED P2 SLICE — internal two-tenant Files lifecycle regression
- **Branch/PR:** `feature/p2-tenant-files-lifecycle-test`, draft [PR #40](https://github.com/parmtech/Tong-Panel/pull/40), stacked on draft #39/#38/#37/#36/#35. Executable test commit: `28bde9fee671c0a8438337ec5017c5d564b778d7`.
- **Scope:** adds one integration-style regression across two distinct temporary pinned tenant roots. It exercises bounded/create-only writes, gzip/tar extraction (buffered and streamed), per-tenant read/list, scoped removal, unknown-tenant denial and cross-root traversal rejection.
- **Validation:** exact test commit passed Go formatting/full `go test -race ./...`/vet [#40612116106](https://github.com/parmtech/Tong-Panel/actions/runs/40612116106), daemon smoke [#40612116078](https://github.com/parmtech/Tong-Panel/actions/runs/40612116078), and unsigned package smoke [#40612116013](https://github.com/parmtech/Tong-Panel/actions/runs/40612116013).
- **Limits/status:** this is integration coverage of the internal tenant-root selector, not authenticated HTTP, user/object ACL, real install/recovery or the roadmap's complete two-tenant acceptance gate. P0/P1 remain open and P2 is incomplete. No real host changed.
- **Next:** continue only isolated Files primitive coverage until identity/ACL integration is approved; then full Files API/UI and hosting lifecycle acceptance still require implementation and review.

## PRIOR VERIFIED P2 SLICE — bounded streamed create-only file uploads

- **Branch/PR:** `feature/p2-bounded-file-writer`, draft [PR #39](https://github.com/parmtech/Tong-Panel/pull/39), stacked on draft #38/#37/#36/#35. Executable code commit: `aca299ed6090de6c327f78700be28086ee785502`.
- **Scope:** adds `Root.WriteNewFrom` and `TenantRoots.WriteNewFromForTenant`. Reads only up to caller-selected cap+1 (cap max 16 MiB), rejects oversize content before target publication, then uses existing mode-0600 atomic create-only no-replace publication. Unknown tenant IDs are rejected before stream consumption.
- **Focused tests:** exact cap success; cap+1 consumption and no partial target; invalid path/limit/nil reader; closed-root non-consumption; unknown-tenant non-consumption and tenant-root separation; concurrent same-target single winner.
- **Validation:** executable code commit `aca299ed6090de6c327f78700be28086ee785502` passed Go formatting/full `go test -race ./...`/vet [#40611776793](https://github.com/parmtech/Tong-Panel/actions/runs/40611776793), daemon smoke [#40611776807](https://github.com/parmtech/Tong-Panel/actions/runs/40611776807), and unsigned package lab [#40611776733](https://github.com/parmtech/Tong-Panel/actions/runs/40611776733). Local Go/gofmt remain unavailable; PR head follow-ups are documentation-only.
- **Limits/status:** the later PR #41 replaces this bounded in-memory stream buffer with direct staging-file streaming. Neither version adds an aggregate tenant quota, global concurrency admission, HTTP API, user/object ACL, auth integration, replacement writes, scanner, request timeout, startup recovery, installer or host mutation. P0/P1 remain open and P2 is incomplete.
- **Next:** continue isolated, non-privileged P2 work. API/auth and real installer/service lifecycle integration remain behind owner/security review and signed package/recovery gates.

## PRIOR VERIFIED P2 SLICE — bounded archive request-stream ingestion
- **Branch/PR:** `feature/p2-bounded-archive-reader`, draft [PR #38](https://github.com/parmtech/Tong-Panel/pull/38), stacked on draft #37/#36/#35. Executable code commit: `fd418e1769efef5e5b21a7e7e3ac722fe31aba51`.
- **Scope:** adds `ReadGzipTarBounded`, `Root.ExtractGzipTarNewFrom`, and a tenant-scoped stream wrapper. It reads at most the selected compressed-byte cap plus one before rejecting overflow; unknown tenants are denied before reading.
- **Focused tests:** exact cap/cap+1 consumption, invalid caps/nil reader, no target publication on oversize, successful bounded extraction, and unknown-tenant non-consumption.
- **Validation:** passed Go formatting/full `go test -race ./...`/vet [#40611138073](https://github.com/parmtech/Tong-Panel/actions/runs/40611138073), daemon smoke [#40611138018](https://github.com/parmtech/Tong-Panel/actions/runs/40611138018), and unsigned package lab [#40611138068](https://github.com/parmtech/Tong-Panel/actions/runs/40611138068). Follow-up PR-head commits are documentation-only.
- **Limits/status:** not an HTTP route, authorization/ACL, quota, timeout, scanner, startup recovery or production deployment. P0/P1 remain open; P2 is incomplete.

## PRIOR VERIFIED P2 QA SLICE — clean disposable VM smoke
- **Branch/PR:** `feature/p2-clean-vm-smoke`, draft [PR #37](https://github.com/parmtech/Tong-Panel/pull/37), stacked on draft #36/#35. Code/workflow commit `de431b6ccbe6f5b418a11f94eb483f24a2b4e205`; follow-up docs are on this branch.
- **Scope:** QEMU/TCG launches fresh Ubuntu 24.04 and Debian 12 cloud-image guests in GitHub Actions. It builds the unsigned preview, provisions an unprivileged test user, verifies OS/version and archive checksum, extracts the preview and runs the real daemon/SQLite smoke and read-only doctor check. The workflow uses repository read-only permission and no secrets, forwards SSH only to runner loopback, and removes guest artifacts after each job.
- **Validation:** both distro jobs passed [clean VM matrix #40610418570](https://github.com/parmtech/Tong-Panel/actions/runs/40610418570) on the exact workflow/code commit `de431b6ccbe6f5b418a11f94eb483f24a2b4e205`.
- **Limits:** this validates preview binary execution on clean candidate OS images only. It is not installer/service install, rollback, backup restore, supported OS certification, or security approval. The archive is unsigned and marked DO NOT INSTALL. P0/P1 remain open; P2 is incomplete.
- **Next:** implement only unprivileged, reviewable P2 pieces while P0/P1 human decisions remain pending; no privileged service or production host changes.

## PRIOR VERIFIED P2 SLICE — locked cleanup for write and archive staging
- **Branch:** `feature/p2-write-stage-cleanup`, stacked on draft PR #35 at `3c99386f13c02690e2c0f26f9764be900b77ff65`; draft PR #36 is [open](https://github.com/parmtech/Tong-Panel/pull/36). Drafts #33–#36 remain unmerged.
- **Scope:** extends the unprivileged Linux file prototype so orphan cleanup covers both create-only write stages and archive extraction stages. `WriteNew` holds a nonblocking exclusive flock through staging and publication; the bounded cleanup path only recognizes exact internal lowercase-hex names, validates effective-UID ownership/private permissions/one-hour age, skips active locks, removes without following symlinks, and syncs the pinned root. The tenant-scoped wrapper uses the selected pinned root only. Startup invocation remains intentionally unwired pending reviewed lifecycle/identity boundaries.
- **Validation:** the focused stale-stage negative/race tests and full Go formatting, race-test and vet workflow passed on code commit `d7b1ae492e75355622b24b779b206640f99eba56` [#40609720874](https://github.com/parmtech/Tong-Panel/actions/runs/40609720874). The non-destructive daemon smoke [#40609720701](https://github.com/parmtech/Tong-Panel/actions/runs/40609720701) and unsigned package lab [#40609720693](https://github.com/parmtech/Tong-Panel/actions/runs/40609720693) also passed. Follow-up commit `ae91dce1ac361f115bc2d8d8965ede9cc2896ec6` only corrected journal line endings/status; it contains no code changes. Local Go/gofmt remain unavailable.
- **P0/P1/P2 status:** ADR-001–009 and threat/security review remain unapproved. No public login, privileged host action, installer or live VPS change. This code is a primitive, not a Files API/UI or tenant authorization layer. P2 alpha remains incomplete.
- **Remaining P2 gates:** Files API/UI with reviewed identity and ACL/object ownership, quotas and upload-body bounds, cross-tenant E2E, site/domain/database/SSL lifecycle, supported-OS installer and rollback, service recovery, backup/restore drill, and accessibility/release evidence.
## PRIOR VERIFIED P2 FOUNDATION — isolated safe file-read primitive
- **Branch:** `feature/p2-safe-tenant-file-read`, stacked on…9341 tokens truncated…passed (gofmt, full `go test -race ./...`, `go vet ./...`); SQLite schema checks [#40425964024](https://github.com/parmtech/Tong-Panel/actions/runs/40425964024) passed on code commit `b132a9e169aa2674adc82ae483595b5e04b0a889`.
- **Limits:** local Python unavailable; full local race run blocked by Windows CGO; Linux CI passed. P0 ADR/security review remains unapproved; all PRs remain drafts. P1 and the cPanel alternative remain incomplete.
- **Next one action:** obtain recorded ADR/security review before wiring auth or queue stores into the daemon, then continue the authenticated single-host control-plane vertical slice.

### 2026-10-26 — P1 agent socket hardening (additional update)
- **Branch:** `feature/p1-core-foundation`, unmerged [draft PR #14](https://github.com/parmtech/Tong-Panel/pull/14).
- **Files:** `internal/agent/agent_linux.go` and `internal/agent/agent_linux_test.go`.
- **Changes:** removed process-global umask mutation, require an owned private non-symlink socket parent, retain socket permission 0600, bounded newline-delimited ping-only JSON requests with malformed/concatenated JSON rejection.
- **Local evidence:** `go test -race ./...` and `go vet ./...` passed; agent tests cover denied UIDs, disallowed ops, shared/symlink parents, malformed requests and valid ping.
- **Remote GitHub CI:** latest agent hardening Go Actions [run #40262754298](https://github.com/parmtech/Tong-Panel/actions/runs/40262754298) **PASSED** (gofmt, Go race tests, vet). Previous preflight commit Go Actions [run 40262321339](https://github.com/parmtech/Tong-Panel/actions/runs/40262321339) passed.
- **Safety:** agent still no privileged verb, not a root service and not approved for server mutations; phase P1 NOT COMPLETE.
- **Next:** review latest PR #14 CI, then security review of agent boundaries and development authentication limitations (#6/#8).

## LATEST SESSION — 2026-10-26 (Australia/Sydney), P1 authentication and metadata
- **Branch:** `feature/p1-auth-sessions-sqlite`, based on `feature/p1-core-foundation`; new [draft PR #15](https://github.com/parmtech/Tong-Panel/pull/15). Both PRs are unmerged.
- **Added Go:** `internal/auth/sessions.go` and tests: random 256-bit session & CSRF secrets, hashed in-memory token registry, expiry, revocation, secure HTTPS host-only cookie constructors; no login HTTP wiring.
- **Added SQLite planning code:** `db/migrations/0001_initial.sql` with tenant, user, membership, session, job, attempt, audit schema and constraints; `db/test_schema.py` and `.github/workflows/schema-checks.yml`.
- **Local tests performed:** `go test -race ./...`, `go vet ./...` **PASS** on Go 1.23.2 developer source checkout; `python3 -m unittest discover -s db -p 'test_*.py' -v` **4/4 PASS**.
- **GitHub Actions:** [SQLite schema run #40310453620](https://github.com/parmtech/Tong-Panel/actions/runs/40310453620) **PASS**. [Go checks run #40310453646](https://github.com/parmtech/Tong-Panel/actions/runs/40310453646) **PASS** (gofmt, go test -race, go vet). Both remote workflows passed.
- **Issue progress:** #6 auth/MFA and #7 durable jobs updated with partial evidence, left open. Parent #14, architecture review #1/#2 and database ADR remain open.
- **NOT IMPLEMENTED:** production login/Argon2id, persistent sessions, actual Go SQL adapter, transactional job queue and worker, complete host agent security, installer, distro/restore testing. P1 still **IN PROGRESS**.
- **NEXT ACTION:** Verify Go Actions for PR #15, review/merge PR #14 first, resolve authentication and SQLite architecture/security signoff, then integrate SQL migration runner and auth session persistence with tests. No live VPS touched.

## CURRENT CHECKPOINT — READ FIRST

**Phase:** P0 architecture/security review pending; P1 core development IN PROGRESS; P2 hosting alpha IN PROGRESS (draft prototypes only).
**Phase state:** ADR-001–009 and the threat model remain proposals/unreviewed. Neither Phase 1 nor Phase 2 is complete or production-ready.
**P2 implementation:** draft [PR #53](https://github.com/parmtech/Tong-Panel/pull/53), stacked on #52–#50, adds persistent tenant file quota reservations; it does not yet recover crash-stranded reservations, reconcile on-disk usage, account deletes or wire the Files API into the daemon. P2 hosting lifecycle/installer/backup-restore acceptance remains open.
**P1 implementation state:** authentication/session, local bootstrap and job/agent prototypes remain in unmerged drafts. No production login or privileged host mutator is approved. No hosting app or installer exists.
**Production/release state:** NOT RELEASED.  
**Primary branch:** main (verify live branch HEAD before work).  
**Owner:** ParmTech.

### Completed and verified on GitHub
- [x] Public GitHub repository created and checked.
- [x] Initial README exists; AGPL-3.0 LICENSE verified.
- [x] docs/BLUEPRINT.md: product, personas, features, acceptance, roadmap overview.
- [x] docs/FEATURE_MATRIX.md: priorities and proof requirements.
- [x] docs/ARCHITECTURE.md: proposed control plane / privileged local agent boundary.
- [x] docs/ROADMAP.md: P0–P6 milestones.
- [x] docs/SECURITY.md: preliminary threat model and security gates.
- [x] docs/UPSTREAM_RESEARCH.md: HestiaCP, Virtualmin, ISPConfig, 1Panel, Coolify, CloudPanel, File Browser and licensing/safety notes.
- [x] docs/UX_UI_SPEC.md: mobile icon grid, desktop-style File Explorer and guidance.
- [x] docs/INSTALLER_SPEC.md: signed fresh-host single-command installation plan.
- [x] AGENTS.md: any-agent resume and work protocol.
- [x] CONTRIBUTING.md: community work conventions.
- [x] README.md: documentation index and clear pre-release notice.
- [x] docs/PHASE_INDEX.md and docs/phases/P0..P6 separate phase instructions and gates.
- [x] docs/ADR_REGISTER.md, docs/THREAT_MODEL.md, docs/TEST_STRATEGY.md: draft decisions, threats and QA framework.
- [x] docs/REPORT_2026-10-26.md: actual phase kickoff report.
- [x] 8 P1 implementation issues (#5–#12) created with deliverables/acceptance.
- [x] Draft PR #13 contains loopback-only Go health API and tests; local Go 1.23.2 test/vet passed and GitHub Actions Go 1.27.3 gofmt/test-race/vet PASSED (branch only).

- [x] P1 developer prototype branch and [draft PR #14](https://github.com/parmtech/Tong-Panel/pull/14) pushed, with successful Go workflow run [40260348235](https://github.com/parmtech/Tong-Panel/actions/runs/40260348235).\n- [x] Detailed [Phase 1 report](docs/PHASE1_REPORT_2026-10-26.md) written.\n\n### Pending / unverified
- [ ] Architecture and licensing ADRs **reviewed and approved** (currently drafted only).
- [ ] Explicit P1 MVP scope and initial supported Linux matrix locked.
- [ ] Detailed adversarial threat model, signed-artifact/installer design reviewed.
- [x] P1 issues created (#5–#12).
- [ ] P1 issues assigned and ownership/reviews confirmed; draft PR #13 approved/merged; CI checks now VERIFIED passing, but PR remains draft and unmerged.
- [ ] Full implementation backend/frontend/agent, installer, CI run evidence, auth/security tests, app recipes.
- [ ] External security audit and install/restore compatibility evidence.
- [ ] Stable product release.

**Next ONE action for a new agent:** request ADR-003/auth-session design review and check the draft PR #14/#15 parent chain before any merge. No live VPS changes.

## GitHub work queue (OPEN — not assigned or completed)
- [#1 P0 architecture ADRs and upstream licences](https://github.com/parmtech/Tong-Panel/issues/1)
- [#2 P0 threat model and security validation](https://github.com/parmtech/Tong-Panel/issues/2)
- [#3 P0 finalize P1 MVP/distro scope](https://github.com/parmtech/Tong-Panel/issues/3)
- [#4 P0 UX flows and file-explorer design](https://github.com/parmtech/Tong-Panel/issues/4)

## Added implementation work queue
- [P1-01 API liveness #5](https://github.com/parmtech/Tong-Panel/issues/5) — prototype PR #13 draft; local tests and GitHub Actions Go checks pass; review pending.
- [P1-02 Auth/MFA #6](https://github.com/parmtech/Tong-Panel/issues/6) — planned.
- [P1-03 Durable jobs #7](https://github.com/parmtech/Tong-Panel/issues/7) — planned.
- [P1-04 Host agent #8](https://github.com/parmtech/Tong-Panel/issues/8) — planned, security gated.
- [P1-05 UI kit #9](https://github.com/parmtech/Tong-Panel/issues/9) — planned.
- [P1-06 Tenant model #10](https://github.com/parmtech/Tong-Panel/issues/10) — planned.
- [P1-07 Installer preflight #11](https://github.com/parmtech/Tong-Panel/issues/11) — planned, no live-host changes.
- [P1-08 CI/supply chain #12](https://github.com/parmtech/Tong-Panel/issues/12) — planned.

## P1 Core Dev Report (2026-10-26)
- Report: [docs/PHASE1_REPORT_2026-10-26.md](docs/PHASE1_REPORT_2026-10-26.md).
- Source: [feature/p1-core-foundation](https://github.com/parmtech/Tong-Panel/tree/feature/p1-core-foundation), [draft PR #14](https://github.com/parmtech/Tong-Panel/pull/14).
- Verified Go Actions [run 40260348235](https://github.com/parmtech/Tong-Panel/actions/runs/40260348235): **success**. Local `go test -race ./...`, `go vet ./...`, `go build ./...`, TypeScript tsc and HTTP smoke passed.
- P1 acceptance is NOT passed: production password/sessions/MFA, SQL metadata/queue, root-agent review, signing, install VM tests and accessibility need engineering and review.

## Status definitions
PLANNED = docs only. IN PROGRESS = active work, not proven. IMPLEMENTED = code exists. TESTED = test results recorded. VERIFIED = acceptance passed. RELEASED = signed, documented artifact available and validated.

## Work journal
### 2026-10-27 — P1 SQLite queue slice
- **Worker:** Codex, workstream C (storage/jobs) with read-only parallel review.
- **Task:** issue #7, incremental Go SQLite migration and transactional queue prototype.
- **Branch / commit / PR:** `feature/p1-sqlite-transactional-queue`, draft PR #16 based on PR #15; SQLite/session implementation commit `6795148c39404f653f57d8936670f2d29595345f`.
- **Changes:** one canonical embedded SQLite migration; database opener with per-connection foreign-key enforcement and single-writer settings; transactional job insert/transitions; tenant-scoped reads and idempotency; persistent hashed session/CSRF tokens with expiry, revocation and live membership revalidation; Windows-specific JSON journal test portability.
- **Tests:** on Go 1.25.14 Windows/amd64, focused tests passed for `internal/database` and `internal/jobs`, plus `cmd/tongd`, `internal/api`, `internal/auth`, `internal/health`, and `internal/rbac`; `go vet` passed for these packages; `go build ./cmd/tongd` passed. Full `go test ./...` and local race tests are blocked by Windows host/toolchain limitations recorded above. Python tests not run.
- **Security/licence:** queue accepts only no-op work and has no worker; session registry stores hashes only and is not wired to login/API. No host mutation. SQLite driver is pinned; transitive licence inventory and ADR-003/threat review remain open.
- **Next one step:** request ADR-003 and auth/session design review; keep the SQLite stores out of production API wiring until approval.

### 2026-10-26 — P0 documentation foundation
- **Worker:** ChatGPT, repo documentation session.
- **Repository:** https://github.com/parmtech/Tong-Panel
- **Branch:** main
- **Task/state:** initial blueprint and handoff foundation — docs created/updated.
- **Work:** created planning specifications, feature matrix, upstream/licence research, security and installer scope, UX file explorer specification, milestone roadmap, agent protocol and contributor guide; upgraded README to document links.
- **Evidence:** links to individual documents in README. GitHub create-file and update-file actions returned commit SHA(s); review git history for exact most recent commit.
- **Tests:** no code tests executed (documentation-only). No VPS was accessed, modified, or deployed to.
- **Known limitations:** design is not implementation, proposed technology stack is not approved, up-to-date licence review needed before source reuse, unsupported distros not guaranteed.
- **Security/third party:** original AGPL code preferred; upstream references are research until independent licence/security review.
- **Next handoff:** read AGENTS.md, draft ADRs, create scoped P1 GitHub issues, seek maintainers' design review before coding.

### 2026-10-26 — P0 backlog handoff update
- **State:** planning issues recorded, not executed or assigned.
- **Created GitHub Issues:** #1 architecture ADRs, #2 threat model, #3 P1 scope, #4 UX flows.
- **Tests:** no implementation tests; issue/workflow-only updates.
- **Handoff:** start issue #1; check current branch/PRs and AGENTS.md before working.

### 2026-10-26 — P0 phase breakdown and P1 prototype
- **Worker:** ChatGPT, workstreams A/C/G planning and experimental unprivileged API.
- **Phase/state:** P0 documents prepared but security/ADR signoff pending; P1-01 prototype committed only to draft branch.
- **Repo:** parmtech/Tong-Panel; default main; feature branch feature/p1-01-loopback-health; [Draft PR #13](https://github.com/parmtech/Tong-Panel/pull/13), related issue #5.
- **Files:** docs/phases/P0..P6, docs/PHASE_INDEX.md, ADR_REGISTER.md, THREAT_MODEL.md, TEST_STRATEGY.md, REPORT_2026-10-26.md; branch: go.mod, cmd/tongd/main.go, internal/health/handler.go and handler_test.go, .github/workflows/go-checks.yml.
- **Tests:** local Go 1.23.2 `go test ./...` passed; `go vet ./...` passed; locally formatted with gofmt. GitHub Actions Go checks run [#40256258155](https://github.com/parmtech/Tong-Panel/actions/runs/40256258155) completed successfully: setup, gofmt, go test -race, go vet all green. Docs-only files received GitHub commit SHAs and references.
- **Security:** no public-root API, no installer or live-server modifications; design controls are unreviewed.
- **Blockers:** architecture ADRs need review, CI/PR approval, real distro support tests, security model acceptance.
- **Next step:** handle architecture review #1 and human code review of draft PR #13; do not call hosting features implemented.

### 2026-10-26 — Expanded P1 prototype and report
- **Worker:** ChatGPT, scoped P1 implementation in disposable local environment.
- **Phase/state:** P1 IN PROGRESS, not completed; P0 architecture/security review pending.
- **Branch/PR:** `feature/p1-core-foundation` → [draft PR #14](https://github.com/parmtech/Tong-Panel/pull/14). Earlier [draft PR #13](https://github.com/parmtech/Tong-Panel/pull/13) is superseded by inclusion in new PR and must be coordinated before merge.
- **Changes:** Go local API + tests, temporary token gate and RBAC, JSON journal and tests, TOTP primitive and tests, Unix-socket ping/peer-check agent and tests, tongctl doctor, responsive TypeScript UI with HTML/CSS/static assets, developer preview.
- **Evidence:** [P1 report](docs/PHASE1_REPORT_2026-10-26.md) and [GitHub Go checks success](https://github.com/parmtech/Tong-Panel/actions/runs/40260348235).
- **Tests:** local Go race/vet/build; TypeScript compiler; HTTP UI/API smoke. Remote GitHub Go CI PASSED. Not tested on clean VPS, no security penetration tests or full UI E2E.
- **Security:** intentionally no root operations; never expose experimental dev bearer API publicly. No production credentials or installer.
- **Blockers:** ADR/threat signoff, real production auth/MFA/session, DB/migrations and durable SQL queue, audited root agent, signer/installer and VM testing, full a11y review.
- **Next step:** Review architecture #1/#2, then implement production identity/migrations in reviewed PR; keep P1 gate open.

## Agent session template (copy into new journal entry)
### YYYY-MM-DD HH:MM TZ — ID / workstream
- Task + owner:
- Phase/state:
- Branch / commit / issue / PR:
- Changed files:
- Tests (commands and exact results; not run if none):
- Security and third-party licence impact:
- Blockers / constraints:
- Next ONE step for handoff:


### 2026-10-28 — P2 gzip framing rejection tests
- Draft [PR #47](https://github.com/parmtech/Tong-Panel/pull/47), stacked on #46, adds negative coverage for corrupted checksum, concatenated gzip member and trailing bytes, with no-publication and stage-cleanup assertions.
- Code commit `12ee26f28b038e64a9930be4dc77ef55a85339ae` passed [Go/race/vet #40617596259](https://github.com/parmtech/Tong-Panel/actions/runs/40617596259), [daemon #40617596170](https://github.com/parmtech/Tong-Panel/actions/runs/40617596170), [package #40617596192](https://github.com/parmtech/Tong-Panel/actions/runs/40617596192), and [Ubuntu/Debian clean-VM matrix #40617596202](https://github.com/parmtech/Tong-Panel/actions/runs/40617596202).
- This is parser regression evidence, not P2 completion or security approval. Owner/security ADR and threat-model decisions remain pending; no live host changed.


### 2026-10-28 — P2 concurrent archive publication
- Draft [PR #48](https://github.com/parmtech/Tong-Panel/pull/48), stacked on #47/#46, adds a 16-way concurrent extraction test targeting the same new directory with different complete archives.
- It requires one winner, `ErrAlreadyExists` for every loser, a single intact winner tree and cleanup of all staging directories.
- Code commit `beaca356887c645520c144cc9fb0fa6cdfd8eeb1` passed [Go/race/vet #40618389233](https://github.com/parmtech/Tong-Panel/actions/runs/40618389233), [daemon #40618389360](https://github.com/parmtech/Tong-Panel/actions/runs/40618389360), [package #40618389336](https://github.com/parmtech/Tong-Panel/actions/runs/40618389336), and Ubuntu/Debian clean guests [#40618389193](https://github.com/parmtech/Tong-Panel/actions/runs/40618389193).
- The test validates existing internal create-only publication; P2 and owner/security review remain incomplete. No production host changed.


### 2026-10-28 — P2 concurrent two-tenant archive isolation
- Draft [PR #49](https://github.com/parmtech/Tong-Panel/pull/49), stacked on #48/#47/#46, adds 16 concurrent extraction attempts per tenant, targeting the same relative site path in two distinct pinned roots.
- It requires one successful publication per tenant, same-tenant losers to get `ErrAlreadyExists`, tenant-specific content on readback, and both stage areas clean.
- Code commit `141e0aab6119c346f2ff761c4bb1ecc2da62beae` passed [Go/race/vet #40618958111](https://github.com/parmtech/Tong-Panel/actions/runs/40618958111), [daemon #40618958138](https://github.com/parmtech/Tong-Panel/actions/runs/40618958138), [package #40618957985](https://github.com/parmtech/Tong-Panel/actions/runs/40618957985), and Ubuntu/Debian clean guests [#40618958036](https://github.com/parmtech/Tong-Panel/actions/runs/40618958036).
- This is internal Files primitive isolation evidence only. P0 owner/security review and P2 acceptance gates remain incomplete; no live host changed.


### 2026-10-28 — P2 fail-closed pre-mutation gate
- Draft [PR #50](https://github.com/parmtech/Tong-Panel/pull/50), stacked on #49/#48/#47, adds a pure evaluator for future installer callers with explicit disk and TCP-port requirements.
- Focused negatives block unsupported OS, invalid facts, low/unknown disk, occupied/unknown required ports and missing/invalid policy. No installer or host mutation consumes this function.
- Corrected code head `55544dcb3a6505a4967fa67ad44046e6111d1b0b` passed [Go/race/vet #40619972006](https://github.com/parmtech/Tong-Panel/actions/runs/40619972006), [daemon #40619972004](https://github.com/parmtech/Tong-Panel/actions/runs/40619972004), and [package #40619972005](https://github.com/parmtech/Tong-Panel/actions/runs/40619972005). Initial gofmt alignment was corrected.
- Thresholds require owner review. This is a fail-closed helper, not installer, distro certification or P2 exit proof. P2 remains incomplete.

### 2026-10-28 — P2 authenticated tenant-scoped Files read API
- **Branch/PR:** `feature/p2-files-read-api`, draft [PR #51](https://github.com/parmtech/Tong-Panel/pull/51), stacked on #50; corrected code head `01be544dca165c9554e5f429bc3029c0015ca009`.
- **Change:** opt-in list/read API uses authenticated principal tenant scope, role checks, strict query validation, path/symlink protection and bounded responses; no daemon wiring or public login.
- **Tests/evidence:** hosted Go formatting, full race suite and vet [#40621248079](https://github.com/parmtech/Tong-Panel/actions/runs/40621248079), daemon smoke [#40621248030](https://github.com/parmtech/Tong-Panel/actions/runs/40621248030), unsigned package [#40621248033](https://github.com/parmtech/Tong-Panel/actions/runs/40621248033), all passed on code head. Local Go unavailable.
- **Security/limits:** no write/delete/upload, object ACL, quota, UI, production login, installer or lifecycle. ADR-001–009/threat-model owner/security decision remains pending; no host changed; P2 incomplete.
- **Next one step:** review and implement the smallest quota-enforced Files write slice behind the same default-disabled service injection; keep upload bounds, tenant authority and failure cleanup explicit.

### 2026-10-28 — P2 quota-gated tenant Files upload API
- **Branch/PR:** `feature/p2-files-write-boundary`, draft [PR #52](https://github.com/parmtech/Tong-Panel/pull/52), stacked on #51/#50; corrected code head `c63709af69c29d01e0ed683ac1d8ea04cc9d9a6e`.
- **Change:** authenticated create-only POST upload, `files:write` tenant RBAC, strict path/content-type validation, 16 MiB cap, pre-read quota reservation, atomic publication and quota accounting for failed/uncertain operations. Files and quota dependencies remain opt-in/default-disabled.
- **Tests/evidence:** Go format/full race/vet [#40622572789](https://github.com/parmtech/Tong-Panel/actions/runs/40622572789), daemon [#40622572813](https://github.com/parmtech/Tong-Panel/actions/runs/40622572813), package [#40622572868](https://github.com/parmtech/Tong-Panel/actions/runs/40622572868), and clean Ubuntu 24.04/Debian 12 VM smoke [#40622572815](https://github.com/parmtech/Tong-Panel/actions/runs/40622572815) passed on the code head. Tests cover quota denial before body reads, cross-tenant isolation, invalid paths, Auditor denial, CSRF, oversize requests, conflicts, concurrency and uncertain publication/commit states. Local Go unavailable.
- **Security/limits:** no durable quota ledger, reconciliation, delete accounting, Files UI/object ACL, daemon wiring, installer, restore or P2 acceptance. ADR/threat-model owner-security decision remains pending; no host changed.
- **Next one step:** design and implement the persistent quota ledger/reservation migration, including reconciliation against existing pinned-root content before allowing configuration.

### 2026-10-29 — P2 persistent tenant Files quota reservations
- **Branch/PR:** `feature/p2-sqlite-files-quota`, draft [PR #53](https://github.com/parmtech/Tong-Panel/pull/53), stacked on #52; final verified code head `f9a67c534bc3276b3f04037dbc94f67b1043392b`.
- **Change:** added SQLite migration v8, transactional tenant quota counters/reservations, tenant foreign-key enforcement, exported schema version, and v8-compatible bootstrap/smoke expectations. No daemon quota service was enabled.
- **Tests/evidence:** focused tests cover quota denial/accounting, release, per-tenant isolation, independent-connection concurrent admission and restart persistence of pending reservations. Exact code head passed [Go format/race/vet #40703478052](https://github.com/parmtech/Tong-Panel/actions/runs/40703478052), [SQLite schema #40703478117](https://github.com/parmtech/Tong-Panel/actions/runs/40703478117), [daemon/bootstrap #40703478331](https://github.com/parmtech/Tong-Panel/actions/runs/40703478331), [unsigned package #40703477994](https://github.com/parmtech/Tong-Panel/actions/runs/40703477994), and [clean Ubuntu/Debian VM #40703477997](https://github.com/parmtech/Tong-Panel/actions/runs/40703477997).
- **Security/limits:** no automatic stale-reservation recovery, filesystem usage scan, delete accounting, Files UI/object ACL or daemon wiring. ADR-001–009 and threat-model owner/security review remain pending; no host changed; P2 incomplete.
- **Next one step:** build a startup barrier and trusted tenant-root reconciliation flow that safely recovers pending reservations before quota-enabled API wiring.

### 2026-10-29 — P2 tenant Files deletion accounting
- **Branch/PR:** `feature/p2-files-delete-accounting`, draft [PR #54](https://github.com/parmtech/Tong-Panel/pull/54), stacked on draft #53; code head `4ff8529f347dd2f2dac81523852d6a6790016cf1`.
- **Change:** authenticated tenant-scoped DELETE uses the existing write permission and CSRF/origin boundary. The pinned-root implementation atomically moves the entry into a private, locked staging directory before measuring its actual type/size and unlinking it. Quota usage decrements only after durable directory sync; symlink deletion is zero bytes.
- **Tests/evidence:** negatives cover invalid/traversal paths, CSRF, Auditor denial, cross-tenant isolation, duplicate delete, quota underflow and tenant separation. Existing symlink-swap race tests exercise the staged unlink path. Latest exact-head Go format/full race/vet [#40708184553](https://github.com/parmtech/Tong-Panel/actions/runs/40708184553), daemon smoke [#40708184530](https://github.com/parmtech/Tong-Panel/actions/runs/40708184530), unsigned package [#40708184387](https://github.com/parmtech/Tong-Panel/actions/runs/40708184387), and Ubuntu/Debian clean VM [#40708184391](https://github.com/parmtech/Tong-Panel/actions/runs/40708184391) passed. A first test run failed due a non-seeded auditor user; corrected run is green.
- **Security/limits:** a crash after unlink but before SQLite accounting can overcount safely; startup reconciliation and pending-reservation recovery remain open. No UI/object ACL, daemon quota wiring, installer, service lifecycle, backup/restore or owner/security approval. P2 incomplete; no host changed.
- **Next one step:** design a cross-process maintenance barrier to drain in-flight writes before startup recovery clears stale reservations.

### 2026-10-29 — P2 measured tenant Files quota setup
- **Branch/PR:** `feature/p2-quota-reconciliation`, draft [PR #55](https://github.com/parmtech/Tong-Panel/pull/55), stacked on #54; code head `2a56d93d250b4ad352831da97db6a7a8b0a8c134`.
- **Change:** add bounded Linux pinned-root regular-file usage measurement, tenant-root selection and measured quota setup. Scans serialize against in-process Root mutations, skip only valid private root staging directories, and fail closed on symlinks-to-path changes, special files, limits and overflow; symlinks themselves are not followed or counted.
- **Tests/evidence:** nested logical byte totals, symlink/staging behavior, depth/entry/special-file negatives, scan serialization with a blocked upload, tenant isolation and quota setup with pending-reservation denial. Exact code head passed [Go formatting/full race/vet #40711216528](https://github.com/parmtech/Tong-Panel/actions/runs/40711216528), [daemon smoke #40711216435](https://github.com/parmtech/Tong-Panel/actions/runs/40711216435), [unsigned package #40711216395](https://github.com/parmtech/Tong-Panel/actions/runs/40711216395), and [Ubuntu 24.04/Debian 12 clean VM #40711216498](https://github.com/parmtech/Tong-Panel/actions/runs/40711216498). Local Go unavailable.
- **Security/limits:** external tenant writers must be quiesced; same-process locking is not a cross-process startup barrier. Crash-stranded reservations still fail closed; no daemon wiring, UI/object ACL, installer, backup/restore or owner/security approval. P2 incomplete; no host changed.
- **Next one step:** design a cross-process maintenance barrier and live-upload drain before clearing pending reservations based on a scan.

### 2026-10-29 — P2 mutation barrier verified; P0 review documented
- **Branch/PR:** `feature/p2-quota-maintenance-barrier`, draft [PR #56](https://github.com/parmtech/Tong-Panel/pull/56), verified current code head `817c73d92b1a8b6e827e212750e4708167b3e9ae`.
- **Validation:** Go format/full race/vet [#40718460289](https://github.com/parmtech/Tong-Panel/actions/runs/40718460289), daemon [#40718460274](https://github.com/parmtech/Tong-Panel/actions/runs/40718460274), unsigned package [#40718460273](https://github.com/parmtech/Tong-Panel/actions/runs/40718460273), and Ubuntu 24.04/Debian 12 guest tests [#40718460480](https://github.com/parmtech/Tong-Panel/actions/runs/40718460480) all pass. The workflow runs API and Files suites in both guests.
- **Docs:** prepared `docs/P0_ENGINEERING_REVIEW_2026-10-29.md` with recommendations for maintainer decision and updated the P1/P2 exit report. No Go toolchain is installed locally; `git diff --check` passes.
- **Limits/security:** issues #1/#2 have no recorded decision comments. The review is not approval or an independent audit. Startup recovery lacks a real external-writer quiescer; P2 remains incomplete and no live host changed.
- **Next:** implement an unprivileged site/domain lifecycle model and tests; obtain owner/security decisions before privileged execution.

### 2026-10-29 — persistent site/domain catalog
- **Branch/PR:** `feature/p2-persistent-site-domain-catalog`, draft [PR #58](https://github.com/parmtech/Tong-Panel/pull/58), exact code head `3d3b9205f13046e69d5f524c788c2482db56d3b7`.
- **Validation:** Go format/race/vet, daemon, unsigned package, SQLite migration constraints, and Ubuntu 24.04/Debian 12 clean VM tests all passed at the stated head. One intermediate Go run exposed SQLite NOCASE comparison semantics; the migration now uses explicit BINARY comparison and the latest full suite is green.
- **Docs/security:** journal and P1/P2 exit report updated. Domain entries stay pending and cannot trigger certificate issuance. No owner/security approval or production host change.
- **Next:** design tenant domain proof and site lifecycle state contracts, retaining fail-closed no-host-effect behavior.

### 2026-10-29 — one-time DNS TXT domain proof
- **Branch/PR:** `feature/p2-domain-dns-verification`, draft [PR #59](https://github.com/parmtech/Tong-Panel/pull/59), exact executable head `87686d4e8956469727afd9f17194e861a53f503f`.
- **Validation:** Go format/race/vet [#40724519503](https://github.com/parmtech/Tong-Panel/actions/runs/40724519503), daemon [#40724519369](https://github.com/parmtech/Tong-Panel/actions/runs/40724519369), archive [#40724519405](https://github.com/parmtech/Tong-Panel/actions/runs/40724519405), schema [#40724519582](https://github.com/parmtech/Tong-Panel/actions/runs/40724519582), and Ubuntu/Debian clean guests [#40724519430](https://github.com/parmtech/Tong-Panel/actions/runs/40724519430) all passed. Intermediate format-only failure was corrected before this head.
- **Security/limits:** token is 256-bit random, stored only as SHA-256 digest, one-hour lifetime and single-use; no public endpoint or ACME integration. Test DNS resolver is injected; no live domain challenged. Owner/security review pending and P2 incomplete.
- **Next:** define reviewed typed host-agent contract and safe Nginx/PHP-FPM site lifecycle; retain no-effect boundary until approval.


### 2026-10-29 — P2 tenant-scoped site database catalog
- **Branch/PR:** `feature/p2-site-database-catalog`, draft [PR #63](https://github.com/parmtech/Tong-Panel/pull/63), stacked on draft #62. Corrected executable head `2b69846209b88998e059499755cc85cd4e924b78`.
- **Change:** SQLite migration v11 adds strictly constrained site database ownership metadata and deterministic bounded MariaDB identifiers; no password/credential fields are stored, and no database service is contacted. Creation is tenant/site scoped and atomic; reads require tenant, site and database IDs.
- **Tests/evidence:** negative coverage includes unsafe and missing owners, invalid IDs, duplicate requests and cross-tenant lookups. Independent SQLite connections race to create one record and exactly one succeeds. Exact code head passed Go format/full race/vet [#40737721111](https://github.com/parmtech/Tong-Panel/actions/runs/40737721111), schema [#40737721769](https://github.com/parmtech/Tong-Panel/actions/runs/40737721769), daemon [#40737721131](https://github.com/parmtech/Tong-Panel/actions/runs/40737721131), archive [#40737721452](https://github.com/parmtech/Tong-Panel/actions/runs/40737721452) and Ubuntu/Debian clean guests [#40737721348](https://github.com/parmtech/Tong-Panel/actions/runs/40737721348). First test runs caught and fixed formatting, stale migration expectation and a SQLite write-race issue before the green head.
- **Limits/next:** no actual MariaDB account/database provisioning, credential rotation/export/import or cross-tenant SQL connection test yet. P0 ADR/threat-model owner/security review is pending. P2 remains incomplete; no production host changed.


### 2026-10-29 — P2 disposable MariaDB two-tenant isolation
- **Branch/PR:** `feature/p2-mariadb-two-tenant-isolation`, draft [PR #64](https://github.com/parmtech/Tong-Panel/pull/64), stacked on #63; exact head `52e64adc70cf512cc0381c07c4b871a9d0ffd1b7`.
- **Change:** clean VM smoke now installs MariaDB inside temporary Debian 12 and Ubuntu 24.04 guests. It creates two random-password local database users with grants restricted to one database each, stores credentials only in mode-0600 temporary files, proves own-database reads and denies cross-tenant reads, then removes users/databases and stops MariaDB.
- **Validation:** both distro jobs passed in [#40740998539](https://github.com/parmtech/Tong-Panel/actions/runs/40740998539), including inherited full Files/API/site package tests, real Nginx/PHP-FPM two-tenant HTTP smoke and daemon smoke. Initial runs found a host-runner optional apt-source 403 and shell heredoc backtick substitution; fixes were verified by the passing exact head.
- **Limits/next:** this proves MariaDB grants can enforce tenant boundaries in the guest; it does not provision databases from Tong Panel, store/rotate credentials, export/import, or certify an installation. ADR/threat-model owner/security review remains pending; P2 incomplete; no real host changed.


### 2026-10-29 — P2 verified HTTPS vhost and HTTP-01 challenge path
- **Branch/PR:** `feature/p2-https-vhost-renderer`, draft [PR #65](https://github.com/parmtech/Tong-Panel/pull/65), stacked on #64. Exact tested code head `cf6aeb93793a4359f69b1a4d07a869a226cb5a90`.
- **Change:** added a pure renderer for verified-domain HTTPS vhosts, fixed tenant/site certificate paths, TLS 1.2/1.3, canonical 308 redirects and an HTTP-only ACME challenge path. It writes no configuration and triggers no host reload. The HTTP server now explicitly uses the validated tenant/site document root.
- **Tests/evidence:** negative Go coverage checks unverified/injected domains, certificate traversal/cross-tenant paths and unsafe roots. Go format/race/vet [#40746211175](https://github.com/parmtech/Tong-Panel/actions/runs/40746211175), daemon [#40746211241](https://github.com/parmtech/Tong-Panel/actions/runs/40746211241), archive [#40746211152](https://github.com/parmtech/Tong-Panel/actions/runs/40746211152), and Ubuntu 24.04/Debian 12 clean VMs [#40746211249](https://github.com/parmtech/Tong-Panel/actions/runs/40746211249) passed. Guests ran `nginx -t`, checked HTTPS content across both tenants, HTTP redirects, the challenge file over HTTP and hidden-file denial using ephemeral self-signed certificates.
- **Failure/fix record:** first guest run failed on Debian because the HTTP challenge server inherited no portable document root. Added explicit validated root and a unit assertion. A later Go check caught formatting in the new assertion; gofmt correction is included in the passing head.
- **Limits/next:** no ACME server challenge, certificate issuance/renewal, key custody/rotation, config write/reload/rollback, installer or full site lifecycle. ADR/threat-model owner-security review remains pending. P2 incomplete; no security approval or real-host change.
