# Tong Panel — Live Project Status and Work Journal
**Updated:** 2026-10-28 (Sydney)
**Project:** ParmTech Tong Panel (https://github.com/parmtech/Tong-Panel)  
**Maintained by:** any authorized contributor/agent who commits changes.  
**Rule:** This file is a **manual committed source of truth**, not an automatic background process. Every meaningful work session should update checkpoint and log.

## VERIFIED P1 CHECKPOINT — 2026-10-28, trusted browser boundary
- **Phase:** P0/ADR review remains pending; P1 development continues on isolated drafts. Phase 1 is not complete or production-ready.
- **Current stacked slice:** new branch `feature/p1-cookie-origin-duplicate-guard`, based on draft PR #33 (`feature/p1-session-idle-rotation`). It rejects ambiguous duplicate `Origin` headers for cookie-authenticated mutations and adds a concurrent authenticated-read test. No public login, cookie issuance, logout route, proxy trust, or server deployment is enabled.
- **Local validation:** Go is not installed in this Windows workspace. Corrected code head `6c983a22de1e5b1ea0de133ea28c6d1daadc490c` passed hosted Go/race/vet, daemon smoke, and archive workflows; see run links below.
- **P0/auth decision:** `docs/ADR-009-AUTHENTICATION.md` is explicitly proposed and unapproved; `docs/ADR_REGISTER.md` marks architecture decisions as draft and requires owner/security review. Do not add a public login handler until that review is recorded.
- **Release blockers:** owner/security approval, external key custody and recovery, public HTTPS login/origin topology, trusted-agent review, signed installer/SBOM, clean Ubuntu/Debian VM install/rollback/recovery, and UI accessibility/E2E remain open. No installer currently exists; no production VPS was changed.
- **Verified code head:** `6c983a22de1e5b1ea0de133ea28c6d1daadc490c` passed Go checks [#40599907640](https://github.com/parmtech/Tong-Panel/actions/runs/40599907640) (gofmt, full `go test -race ./...`, vet), non-destructive daemon smoke [#40599907840](https://github.com/parmtech/Tong-Panel/actions/runs/40599907840), and unsigned developer archive lab [#40599907588](https://github.com/parmtech/Tong-Panel/actions/runs/40599907588). These validate this code head; they do not certify a release or security approval.
- **Authentication boundary:** owner/security review must record ADR-009/P0 decisions before a public login route; keep public login and production deployment disabled. Isolated non-privileged prototypes can continue.

## CURRENT P2 SLICE — pinned clean-VM images and checksum rejection tests
- **Branch/PR:** `feature/p2-pin-vm-images`, draft [PR #45](https://github.com/parmtech/Tong-Panel/pull/45), stacked on #44/#43/#42/#41 and the Files foundation #35. Code commit: `c9b4767eaa6722af22f31e742c50600ea2b31440`.
- **Scope:** replace mutable Ubuntu `noble/current` and Debian `bookworm/latest` guest URLs with dated official image paths. Verify pinned SHA-256/SHA-512 before QEMU overlay creation or guest boot. Negative tests reject malformed/wrong digests, altered images and missing files; both supported algorithms have valid-fixture tests.
- **Validation:** local Cygwin Bash checksum tests, `bash -n` and `git diff --check` passed. Both Ubuntu 24.04 and Debian 12 guests passed [GitHub matrix #40615254194](https://github.com/parmtech/Tong-Panel/actions/runs/40615254194), including checker negative cases, Files suite, daemon/SQLite smoke and doctor.
- **Limits/status:** reproducibility and image-integrity coverage only; this still runs an unsigned developer preview, not an installer or certified OS matrix, and does not validate services, rollback, restore, authenticated API/ACL or full P2 hosting E2E. ADR-001–009 and threat-model owner/security review remain pending; P0/P1 are unapproved, P2 incomplete, and no real host changed.
- **Next:** continue a safe non-privileged P2 slice after the matrix verifies. Do not expose public login or implement privileged host mutations before architecture/security review.

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
- **Branch:** `feature/p2-safe-tenant-file-read`, stacked on draft PR #34. PR #34's cookie-origin change remains a draft; its latest exact-head Go/race/vet, daemon smoke, and archive workflows passed (see prior entry).
- **Scope:** Linux-only `internal/files.Root` uses a pinned directory descriptor and `openat2` constraints for bounded reads, sorted directory listing, atomic create-only writes, no-follow removal, and bounded atomic gzip/tar extraction. All paths reject traversal and symlink traversal; archive extraction rejects links and special entries; writes stage under a random mode-0700 directory and publish with `renameat2(RENAME_NOREPLACE)` as mode 0600. Fixed server-owned tenant roots now scope reads, listings, create-only writes, removal and archive extraction; constructor rejects closed roots and tenant mappings that resolve to the same device/inode. The mapping remains a selector, not authorization. Archive extraction defaults to 32 MiB compressed, 80 MiB expanded, 16 MiB per file, 64 MiB total file bytes, and 2,048 entries/nodes. Explicit orphan cleanup is tenant-root scoped and requires one-hour age, owner/private-mode checks and an exclusive lock; it is not wired to startup. Reserved internal stage names are blocked from generic operations and hidden from root listings. No HTTP or privileged operation is wired.
- **Validation:** code head `a2922cac5945f6ff8f47302ad8099754ba6bba5a` passed hosted Go checks [#40608916965](https://github.com/parmtech/Tong-Panel/actions/runs/40608916965) (gofmt, full `go test -race ./...`, vet), daemon smoke [#40608916946](https://github.com/parmtech/Tong-Panel/actions/runs/40608916946), and unsigned package lab [#40608917120](https://github.com/parmtech/Tong-Panel/actions/runs/40608917120). Local Go executable is unavailable.
- **P2 status:** PLANNED/IN PROGRESS only. No UI/API, ACL/quotas integration, two-tenant E2E, site/domain/database/SSL lifecycle, installer, or restore drill exists. Replacement writes, automatic startup invocation of orphan cleanup, and HTTP upload/download/delete/authorization are not implemented. Directory listing is bounded to 1,000 entries and may be incomplete during concurrent mutations. The tenant selector now applies to listing and create-only write operations as well as reads. P1 approvals and host-agent/installer prerequisites remain open.
- **Next action:** continue isolated archive/file-manager negative/race coverage and document crash/orphan recovery requirements; do not wire file operations into HTTP until P1 identity and API trust boundaries receive maintainer review.

### 2026-10-28 — P2 clean-VM image pinning and integrity checks
- **Branch/PR:** `feature/p2-pin-vm-images`, draft [PR #45](https://github.com/parmtech/Tong-Panel/pull/45), stacked on draft #44. Code commit `c9b4767eaa6722af22f31e742c50600ea2b31440`.
- **Change:** pin Ubuntu 24.04 to official release `20261017`/SHA-256 and Debian 12 to official build `20261026-2643`/SHA-512. Verify each download before creating a QEMU overlay. Add checksum tests for both algorithms and rejection of malformed/wrong digests, tampering and missing files.
- **Tests:** local Cygwin Bash verifier suite, `bash -n` for all three scripts and `git diff --check` passed. Hosted clean-VM workflow [#40615254194](https://github.com/parmtech/Tong-Panel/actions/runs/40615254194) passed both Ubuntu 24.04 and Debian 12 guest jobs, including checksum rejection tests and the Files/daemon/doctor smoke.
- **Security/limits:** no public login, privileged host operation, live server, installer or release was enabled. P0/ADR/security owner review remains pending. This is image integrity and guest test evidence, not OS certification or P2 completion.
- **Next step:** finish hosted Ubuntu/Debian guest verification; proceed to another approved, non-privileged P2 slice only.

### 2026-10-28 — P2 Files tests in clean Ubuntu/Debian guests
- **Branch/PR:** `feature/p2-files-on-clean-vm`, draft [PR #44](https://github.com/parmtech/Tong-Panel/pull/44), stacked on #43/#42.
- **Change:** build the Linux `internal/files` Go test binary and include it inside the checksum-verified unsigned preview bundle. The QEMU script runs it as unprivileged user `ci` inside fresh Ubuntu 24.04 and Debian 12 guest kernels. Added `internal/files/**` to workflow path triggers.
- **Validation:** both guest jobs succeeded in [clean VM workflow #40614167120](https://github.com/parmtech/Tong-Panel/actions/runs/40614167120). The workflow also retained daemon/SQLite smoke and read-only doctor checks.
- **Limits/next:** this validates the file primitives on clean candidate guest kernels only. It is not an installer, OS certification, service/rollback/restore test or hosting E2E. P0/P1 remain open; no live host changed.

### 2026-10-28 — P0/P1 status reconciliation for owner review
- **Branch/PR:** `docs/phase-status-2026-10-28`, draft [PR #43](https://github.com/parmtech/Tong-Panel/pull/43), stacked on draft #42.
- **Change:** update `docs/ROADMAP.md` to reflect that P0/P1 drafts exist but owner review remains pending, and correct the stale “ADR work not started” next action. Update ADR-009’s implementation status to distinguish internal draft prototypes #27–#34 from production HTTPS login; no proposed architecture choice is approved.
- **Evidence:** rechecked draft PR states #27–#42 and open issues [#1](https://github.com/parmtech/Tong-Panel/issues/1) and [#6](https://github.com/parmtech/Tong-Panel/issues/6); both issues remain open without review comments. Documentation only; no code/CI behavior or live host changed.
- **Next:** owner/security reviewers must record ADR/threat-model decisions before production authentication, privileged installer/service integration, or full P2 acceptance can proceed.

### 2026-10-28 — P2 root-close-safe streamed file upload
- **Branch/PR:** `feature/p2-stream-close-safe`, draft [PR #42](https://github.com/parmtech/Tong-Panel/pull/42), stacked on #41.
- **Change:** duplicate the pinned root descriptor while protected by the lifecycle lock, then release the lock before reading caller input. Staging continues through the retained descriptor; `Close` can return without waiting on a blocked stream, while the already-started operation can complete safely.
- **Tests:** a gated reader blocks upload input while Close returns; after releasing the reader the file is atomically published, and new operations fail with `ErrClosed`.
- **Validation:** executable commit `faa220eed997f68b225dc105cc4386227d742dfe` passed [Go format/full race/vet #40613452838](https://github.com/parmtech/Tong-Panel/actions/runs/40613452838), [daemon smoke #40613452810](https://github.com/parmtech/Tong-Panel/actions/runs/40613452810), and [unsigned archive lab #40613452879](https://github.com/parmtech/Tong-Panel/actions/runs/40613452879).
- **Limits/next:** caller must cancel/close a stalled stream; this does not add deadline, global admission, tenant quota, API/auth/ACL or real host operations. P0/P1 remain open.

### 2026-10-28 — P2 stream file uploads directly to private staging
- **Branch/PR:** `feature/p2-stream-file-to-stage`, draft [PR #41](https://github.com/parmtech/Tong-Panel/pull/41), stacked on #40.
- **Change:** refactor `WriteNewFrom` to copy bounded input directly into its locked private staging file instead of holding the whole upload buffer in memory. It consumes cap+1 at most, discards the stage on limit overflow or reader error, and publishes only a complete file using atomic no-replace rename. `WriteNew` delegates through the same path.
- **Tests:** overflow/no target or leaked stage, reader error/no partial target, exact limit, invalid input and closed root before consumption, concurrent one-winner creation, and a gated reader proving `Close` waits until publication completes.
- **Validation:** code commit `df450a054fe8df50333e8e820f4145601e8d75f8` passed [Go formatting/full race/vet #40612826353](https://github.com/parmtech/Tong-Panel/actions/runs/40612826353), [daemon smoke #40612826486](https://github.com/parmtech/Tong-Panel/actions/runs/40612826486), and [unsigned package lab #40612826389](https://github.com/parmtech/Tong-Panel/actions/runs/40612826389).
- **Limits/next:** no aggregate disk quota, global concurrency admission, request deadline, API/auth/ACL, install or host lifecycle. P0/P1 remain open; continue safe primitives without wiring public routes.

### 2026-10-28 — P2 internal two-tenant Files lifecycle regression
- **Branch/PR:** `feature/p2-tenant-files-lifecycle-test`, draft [PR #40](https://github.com/parmtech/Tong-Panel/pull/40), stacked on #39.
- **Change/tests:** one integration-style test runs both fixed tenant roots through bounded/create-only writes, buffered and streamed archive extraction, tenant read/list, removal, unknown-tenant denial, and cross-root traversal denial. It confirms distinct data remains isolated and tenant A deletion does not change tenant B's file.
- **Validation:** test commit `28bde9fee671c0a8438337ec5017c5d564b778d7` passed gofmt/full race suite/vet [#40612116106](https://github.com/parmtech/Tong-Panel/actions/runs/40612116106), daemon smoke [#40612116078](https://github.com/parmtech/Tong-Panel/actions/runs/40612116078), and unsigned package smoke [#40612116013](https://github.com/parmtech/Tong-Panel/actions/runs/40612116013).
- **Limits/next:** internal tenant-root integration only; no auth/API/ACL/real install. It does not satisfy the roadmap's end-to-end hosting gate. Continue safe internal file tests; review ADR/P0 gates before API/host work.

### 2026-10-28 — P2 bounded streamed create-only file writes
- **Branch/PR:** `feature/p2-bounded-file-writer`, draft [PR #39](https://github.com/parmtech/Tong-Panel/pull/39), stacked on #38.
- **Change:** add `Root.WriteNewFrom` and `TenantRoots.WriteNewFromForTenant`. Input is capped at max+1 bytes (caller cap up to 16 MiB); oversize input is rejected before publication, and successful writes reuse the pinned-root create-only atomic no-replace path. This is a primitive, not an HTTP endpoint or authorization layer.
- **Tests:** exact-cap write; only cap+1 consumption on overflow; oversize target absence; invalid path/limit/nil reader; closed-root denial before reading; unknown tenant denial before reading and tenant-root isolation; concurrent same-target single-winner.
- **Validation:** executable code commit `aca299ed6090de6c327f78700be28086ee785502` passed Go format/race/vet [#40611776793](https://github.com/parmtech/Tong-Panel/actions/runs/40611776793), daemon smoke [#40611776807](https://github.com/parmtech/Tong-Panel/actions/runs/40611776807), and unsigned package smoke [#40611776733](https://github.com/parmtech/Tong-Panel/actions/runs/40611776733). Local Go/gofmt unavailable.
- **Limits/next:** bounded per-request memory only, not aggregate tenant quota. No HTTP/ACL/auth, replacement semantics, timeout, startup cleanup, installer or real host changes. P0/P1 approval gates remain open; continue safe isolated P2 slices.

### 2026-10-28 — P2 bounded archive request-stream ingestion
- **Branch/PR:** `feature/p2-bounded-archive-reader`, draft [PR #38](https://github.com/parmtech/Tong-Panel/pull/38), stacked on #37.
- **Change:** add a bounded `io.Reader` helper and root/tenant archive extraction methods. The request stream is limited to compressed cap plus one byte before erroring, and the tenant selector is resolved before reading. No HTTP route or identity is wired.
- **Tests:** exact/over-limit read bounds, invalid limits and nil reader, no publication on oversize, valid stream extraction, and unknown-tenant non-consumption.
- **Validation:** executable code commit `fd418e1769efef5e5b21a7e7e3ac722fe31aba51` passed Go formatting/full `go test -race ./...`/vet [#40611138073](https://github.com/parmtech/Tong-Panel/actions/runs/40611138073), daemon smoke [#40611138018](https://github.com/parmtech/Tong-Panel/actions/runs/40611138018), and unsigned package lab [#40611138068](https://github.com/parmtech/Tong-Panel/actions/runs/40611138068). Local Go/gofmt are unavailable.
- **Limits/next:** no HTTP, auth/ACL/quota, timeout, scanner, installer or service lifecycle integration. P0/P1 decisions remain unapproved; verify Actions before extending the slice.

### 2026-10-28 — P2 tenant-scoped archive-stage cleanup
- **Branch/PR:** `feature/p2-safe-tenant-file-read`, draft [PR #35](https://github.com/parmtech/Tong-Panel/pull/35), stacked on #34.
- **Change:** add `TenantRoots.CleanupStaleArchiveStagesForTenant`, which resolves cleanup only to the mapped pinned root. Added tests proving tenant A cleanup removes its stale archive stage without touching tenant B's stage, while unknown tenant identifiers fail closed.
- **Validation:** code head `a2922cac5945f6ff8f47302ad8099754ba6bba5a` passed Go formatting/full race tests/vet [#40608916965](https://github.com/parmtech/Tong-Panel/actions/runs/40608916965), daemon smoke [#40608916946](https://github.com/parmtech/Tong-Panel/actions/runs/40608916946), and unsigned package lab [#40608917120](https://github.com/parmtech/Tong-Panel/actions/runs/40608917120).
- **Limits:** mapping cleanup to a tenant root does not replace user/object authorization. Startup recovery integration, API/UI, ACL, quotas and the end-to-end two-tenant gate remain outstanding.
- **Next:** connect cleanup to a reviewed lifecycle only after the P1 identity/trust boundary approval.
### 2026-10-28 — P2 create-only write-stage orphan cleanup
- **Branch/PR:** `feature/p2-write-stage-cleanup`, stacked on draft [PR #35](https://github.com/parmtech/Tong-Panel/pull/35); draft [PR #36](https://github.com/parmtech/Tong-Panel/pull/36) is open.
- **Change:** `WriteNew` now holds an exclusive lock on its private stage so cleanup cannot remove an active write. Unified `CleanupStaleStages` handles both write and archive stages, with the existing archive-cleanup entrypoints retained as compatibility wrappers. It matches only exact lowercase-hex generated names and keeps the existing age, owner, mode, inode recheck, lock, no-follow removal, and root-sync protections. The tenant wrapper and regression test prove a write-stage orphan in one tenant root cannot trigger cleanup in another.
- **Tests added:** stale write-stage removal; active write-stage lock skip then cleanup after unlock; symlink stage/outside directory and unrelated-name preservation; tenant-scope denial/isolation.
- **Validation:** local Go/gofmt unavailable on this Windows host. Code commit `d7b1ae492e75355622b24b779b206640f99eba56` passed Go formatting/full race tests/vet [#40609720874](https://github.com/parmtech/Tong-Panel/actions/runs/40609720874), daemon smoke [#40609720701](https://github.com/parmtech/Tong-Panel/actions/runs/40609720701), and unsigned package lab [#40609720693](https://github.com/parmtech/Tong-Panel/actions/runs/40609720693). A docs-only line-ending/status correction followed in `ae91dce1ac361f115bc2d8d8965ede9cc2896ec6`; no code changed after the verified code commit.
- **Limits/next:** startup cleanup remains unwired; no API/ACL/UI, quotas, installer, hosting service lifecycle, backup/restore drill or P2 two-tenant E2E. Next unblocked step is to finish exact-head CI and continue the P2 Files/API and lifecycle gates only within reviewed trust boundaries.

### 2026-10-28 — P2 locked archive-stage orphan cleanup
- **Branch/PR:** `feature/p2-safe-tenant-file-read`, draft [PR #35](https://github.com/parmtech/Tong-Panel/pull/35), stacked on #34.
- **Change:** archive extraction now holds an exclusive flock on a reserved private staging directory. `CleanupStaleArchiveStages` explicitly removes exact-name stages only when they are at least one hour old, owned by the effective UID, private-mode directories and not actively locked. It refuses symlink traversal and fsyncs the pinned root after removals. Generic read/write/list/remove/extract paths reject reserved root stage names; root listing hides them. This cleaner is not yet wired to a daemon startup/recovery path.
- **Validation:** final code head `6a30f9402a7a93b3ff8ec8e854fe081268b3ba4b` passed Go formatting/full race tests/vet [#40608642107](https://github.com/parmtech/Tong-Panel/actions/runs/40608642107), daemon smoke [#40608642147](https://github.com/parmtech/Tong-Panel/actions/runs/40608642147), and unsigned package lab [#40608642235](https://github.com/parmtech/Tong-Panel/actions/runs/40608642235). An initial build failure from the root-list filter's missing import was corrected before the passing head.
- **Limits:** explicit cleanup does not prove a complete crash-recovery lifecycle or cross-process deployment policy; startup integration, restore drills, API/UI, auth/ACL, site/service lifecycle and P2 end-to-end tests remain undone.
- **Next:** validate cleanup recovery through disposable integration scenarios and only wire it into an approved startup lifecycle after P1 trust boundaries are reviewed.

### 2026-10-28 — P2 bounded atomic archive extraction
- **Branch/PR:** `feature/p2-safe-tenant-file-read`, draft [PR #35](https://github.com/parmtech/Tong-Panel/pull/35), stacked on #34.
- **Change:** add `Root.ExtractGzipTarNew` and a tenant-root wrapper. It stages a new tree under a private random directory, accepts only regular files/directories, applies fixed restrictive modes, enforces compressed/expanded/per-file/total/entry limits (tests exercise each bound and invalid caller limits), and publishes by atomic no-replace rename. Traversal, absolute/duplicate paths, symlink/hardlink, device/FIFO and other special entries fail closed. Added failure cleanup, outside-target symlink-swap, tenant-scope and concurrent single-winner tests. Private staging directories can remain after process crash; sync errors after publication may leave durability uncertain.
- **Validation:** code head `40a4f0ceb19b96c6452519ee302a907ea2f791bd` passed Go formatting/full race tests/vet [#40607535968](https://github.com/parmtech/Tong-Panel/actions/runs/40607535968), daemon smoke [#40607535958](https://github.com/parmtech/Tong-Panel/actions/runs/40607535958), and unsigned package lab [#40607535969](https://github.com/parmtech/Tong-Panel/actions/runs/40607535969). An initial formatting failure was corrected before the passing head.
- **Limits:** extraction is not an HTTP upload, antivirus scan, ACL, user/object ownership check, UI or full two-tenant acceptance test. Caller must cap request body before allocating archive bytes. P0/ADR and P1 auth/agent/installer gates remain open; no real host was changed.
- **Next:** review bounded extraction recovery semantics, then continue only isolated primitives until the identity/API review gate is recorded.

### 2026-10-28 — P2 reject aliased tenant roots
- **Branch/PR:** `feature/p2-safe-tenant-file-read`, draft [PR #35](https://github.com/parmtech/Tong-Panel/pull/35), stacked on #34.
- **Change:** `NewTenantRoots` now checks each pinned root is open and rejects duplicate device/inode identities. This prevents an accidental mapping from assigning the same directory to multiple tenants, including when the same directory is opened through separate descriptors. Tests cover both alias forms and closed-root rejection.
- **Validation:** final code head `69f78aa62f40a297aaa00ac21dd1d5b3553c0745` passed Go formatting/full race tests/vet [#40605933592](https://github.com/parmtech/Tong-Panel/actions/runs/40605933592), daemon smoke [#40605933546](https://github.com/parmtech/Tong-Panel/actions/runs/40605933546), and unsigned package lab [#40605933477](https://github.com/parmtech/Tong-Panel/actions/runs/40605933477). An initial gofmt failure was corrected before the passing head.
- **Limits:** inode uniqueness does not prove authorization or prevent later filesystem reconfiguration; API ACLs and two-tenant E2E remain outstanding.
- **Next:** continue the Files API/object-ownership contract after P0/P1 trust boundary review, then connect it with end-to-end isolation evidence.

### 2026-10-28 — P2 rooted no-follow removal primitive
- **Branch/PR:** `feature/p2-safe-tenant-file-read`, draft [PR #35](https://github.com/parmtech/Tong-Panel/pull/35), stacked on #34.
- **Change:** add `Root.RemoveFile` and tenant-root selection. It only removes regular-file or symlink entries, does not follow symlinks, refuses directories/special files, and syncs the parent directory after unlink. A sync error can happen after the entry is already removed, so durability may be uncertain. Tests cover path escape, directory/FIFO rejection, symlink target preservation under concurrent swap, and tenant-root separation.
- **Validation:** code head `7781a2506adf2710e7c233d1a89b2ef45a6f51d0` passed Go formatting/full race tests/vet [#40605411933](https://github.com/parmtech/Tong-Panel/actions/runs/40605411933), daemon smoke [#40605412063](https://github.com/parmtech/Tong-Panel/actions/runs/40605412063), and unsigned package lab [#40605411997](https://github.com/parmtech/Tong-Panel/actions/runs/40605411997).
- **Limits:** this is a path operation, not user/object authorization or a complete delete API; same-tenant ACL policy, audit wiring, UI, quotas, API and recovery remain missing. No host mutation or deployment.
- **Next:** review path-operation concurrency semantics and continue with API-level authorization only after P0/P1 decisions are accepted.

### 2026-10-28 — P2 tenant-scoped file operations
- **Branch/PR:** `feature/p2-safe-tenant-file-read`, draft [PR #35](https://github.com/parmtech/Tong-Panel/pull/35), stacked on #34.
- **Change:** extend the copied server-owned tenant-root selector with `ListDirForTenant` and `WriteNewForTenant`. Tests prove a tenant can list/create only within its pinned root, while empty/unknown IDs and traversal are denied. A gofmt check first found a trailing blank after the new test; that formatting-only issue was corrected.
- **Validation:** final code head `f8c376d288a57693a77d317cf284145e8a1fbe38` passed Go formatting/full race suite/vet [#40604871616](https://github.com/parmtech/Tong-Panel/actions/runs/40604871616), daemon smoke [#40604871521](https://github.com/parmtech/Tong-Panel/actions/runs/40604871521), and unsigned package lab [#40604871598](https://github.com/parmtech/Tong-Panel/actions/runs/40604871598).
- **Limits:** tenant key selection is not proof of identity, authorization, ACL enforcement or end-to-end isolation. P0/ADR and P1 identity review remain unapproved; no API/UI or privileged path is connected.
- **Next:** proceed with reviewed Files API authorization contract and disposable installer/OS validation prerequisites; do not claim P2 acceptance from these isolated primitives.

### 2026-10-28 — P2 bounded directory listing prototype
- **Branch/PR:** `feature/p2-safe-tenant-file-read`, draft [PR #35](https://github.com/parmtech/Tong-Panel/pull/35), stacked on #34.
- **Change:** add Linux `Root.ListDir`, which opens beneath the pinned root with `openat2`, returns sorted bounded metadata (maximum 1,000 entries), classifies symlinks without following them, and fails closed on symlinked directory traversal. Concurrent removals may produce an incomplete snapshot. Tests cover path traversal, entry bounds, type/size classification, symlink non-dereference, and concurrent outside-directory symlink swaps.
- **Validation:** code head `71c614c0bbbc4ad93a4eee7bd72fddccf7120cb1` passed Go formatting/full race tests/vet [#40603829603](https://github.com/parmtech/Tong-Panel/actions/runs/40603829603), daemon smoke [#40603829544](https://github.com/parmtech/Tong-Panel/actions/runs/40603829544), and unsigned package lab [#40603829661](https://github.com/parmtech/Tong-Panel/actions/runs/40603829661).
- **Limits/blockers:** listing is not a UI/API, authorization or stable transaction. P2 still lacks file update/delete, quotas, archive handling, tenant ACL integration, site/service lifecycle, install/rollback and restore evidence. ADR-009/P0 and P1 prerequisites remain unresolved; no production change or readiness claim.
- **Next:** continue toward the P2 file-management acceptance contract only after authorization boundaries are reviewed; in parallel, establish disposable installer/OS validation once a real installer entrypoint exists.

### 2026-10-28 — P2 create-only file write prototype
- **Branch/PR:** `feature/p2-safe-tenant-file-read`, draft [PR #35](https://github.com/parmtech/Tong-Panel/pull/35), stacked on #34.
- **Change:** add `Root.WriteNew` with a 16 MiB limit, mode 0600, an unpredictable private staging directory at the pinned root, and atomic no-replace publication. Add negative coverage for overwrite/traversal/oversize/symlink parents, concurrent one-winner creation, staging cleanup, and symlink-swap attempts against an outside secret. This remains a primitive without HTTP or authorization integration.
- **Validation:** initial hosted gofmt failed on a spacing alignment; corrected in commit `0659f0e993df4a8c35530231b651518824dd930c`. Exact-head Go/race/vet [#40603084285](https://github.com/parmtech/Tong-Panel/actions/runs/40603084285), daemon smoke [#40603084123](https://github.com/parmtech/Tong-Panel/actions/runs/40603084123), and unsigned package lab [#40603084109](https://github.com/parmtech/Tong-Panel/actions/runs/40603084109) all passed.
- **Limits/blockers:** process crash may leave staging directories; durability may be uncertain if destination-directory fsync fails after publication. No recovery cleaner, quotas, API/UI, tenant ACL, installer or restore drill is delivered. ADR-009/P0 and P1 prerequisites remain unresolved; no production readiness/security approval claim.
- **Next:** add isolated orphan-stage inspection/cleanup rules with strict age and ownership checks, or move to disposable installer lab only after reviewing its unprivileged boundary.

### 2026-10-28 — P2 safe file-read foundation
- **Branch/PR:** `feature/p2-safe-tenant-file-read`, draft [PR #35](https://github.com/parmtech/Tong-Panel/pull/35), stacked on #34.
- **Change:** add a Linux-only unprivileged `openat2`-based descriptor-rooted regular-file reader with a 16 MiB cap, no symlink/mount fallback, and a fixed tenant-to-root selector. Add traversal/symlink/FIFO/size/close, unknown-tenant, and concurrent symlink-swap tests.
- **Validation:** code commit `6100e7092f219d8a2bf99a4db5c3489f9a973c8c` passed hosted Go/race/vet [#40601259497](https://github.com/parmtech/Tong-Panel/actions/runs/40601259497), daemon smoke [#40601259446](https://github.com/parmtech/Tong-Panel/actions/runs/40601259446), and unsigned archive lab [#40601259524](https://github.com/parmtech/Tong-Panel/actions/runs/40601259524). An initial gofmt alignment issue and a race-test fixture write race were corrected before this passing run.
- **Limitations:** does not implement files API/UI, tenant identity/ACL, writes, archive extraction or integration with host agent. P2 alpha remains gated by P1, installer, service adapters, two-tenant isolation, and full recovery evidence.
- **Next:** draft Files API contract with tenant identity supplied only by server-side authorization; keep this primitive isolated until P1 trust boundaries receive review.

### 2026-10-28 — Cookie Origin ambiguity guard
- **Branch:** `feature/p1-cookie-origin-duplicate-guard`, based on #33. A local author identity and Git HTTPS credentials are unavailable, so the working tree patch could not be committed or pushed from this checkout; the same changes are being published through the connected GitHub repository integration.
- **Code/tests:** `internal/api/session_cookie.go` now rejects requests with zero or multiple Origin header values; `internal/api/session_cookie_test.go` adds duplicate hostile Origin denial and 24 simultaneous authenticated-session reads.
- **Validation:** local Go toolchain unavailable. At code commit `6c983a22de1e5b1ea0de133ea28c6d1daadc490c`, Go checks [#40599907640](https://github.com/parmtech/Tong-Panel/actions/runs/40599907640) passed gofmt, full `go test -race ./...`, and vet; daemon smoke [#40599907840](https://github.com/parmtech/Tong-Panel/actions/runs/40599907840) and unsigned archive lab [#40599907588](https://github.com/parmtech/Tong-Panel/actions/runs/40599907588) also passed. An initial Go run detected extra trailing blank lines from API file updates; these were corrected before the passing code-head run.
- **Security boundary:** no login or cookie issuance route added. ADR-009 and P0 remain unapproved; no security approval or production readiness claim.
- **Next:** owner/security review of ADR-009 and P0 remains necessary before any public authentication route; otherwise continue with a separate unblocked installer lab slice.

## LATEST VERIFIED WORK — 2026-10-27, Phase 1 (SQLite queue slice partial)
- **PR:** [Draft #14 — P1 Core Foundation](https://github.com/parmtech/Tong-Panel/pull/14), branch `feature/p1-core-foundation` (not merged).
- **Changed files:** `internal/preflight/doctor.go`, `internal/preflight/doctor_test.go`, new `docs/READ_ONLY_DOCTOR.md` on PR branch.
- **Working functionality:** `tongctl doctor` now reads candidate OS, available RAM, root filesystem free space, and TCP LISTEN 80/443 from /proc, returning explicit pass/warn/block JSON. Does not modify server or bind ports.
- **Local tests:** `go test -race ./...`, `go vet ./...`, `go build ./...` passed; container Debian 13 report correctly blocked unsupported initial distro candidate.
- **GitHub Actions verified:** [Go checks run #40262321339](https://github.com/parmtech/Tong-Panel/actions/runs/40262321339) **success**, gofmt + go test -race + go vet.
- **Not done:** production install, VM distro certification, kernel/cloud firewall verification, installer rollback, host agent security audit, secure auth, SQL data store and work queues. **Phase 1 remains IN PROGRESS.**
- **Next one action:** implement reviewed production auth/session architecture (#6) or continue safe installer diagnostics (#11), then update this file with tests and commits. Read AGENTS.md and open PR #14 before starting.

## LATEST SESSION — 2026-10-27, P1 SQLite queue slice
- **Branch/PR:** `feature/p1-sqlite-transactional-queue`, draft [PR #16](https://github.com/parmtech/Tong-Panel/pull/16), stacked on draft PR #15 (`feature/p1-auth-sessions-sqlite`), then #14. SQLite/session implementation commit `6795148c39404f653f57d8936670f2d29595345f`.
- **Scope:** issue #7, a Go SQLite migration runner and non-executing transactional queue prototype; no changes to the live VPS.
- **Code:** schema moved to `internal/database/migrations/0001_initial.sql` and embedded in a Go opener; added a pinned cgo-free SQLite driver, transactional job store, and independent SQLite-backed session registry that stores only session/CSRF hashes and rechecks live membership. Neither is wired into the API/daemon.
- **Tests actually run:** Go 1.25.14 Windows/amd64 focused tests passed for `internal/database`, `internal/jobs`, `internal/api`, `internal/auth`, `internal/health`, `cmd/tongd` and `internal/rbac`; `go vet` passed for those packages; `go build ./cmd/tongd` passed. This includes the new session persistence, expiry, revocation, tenant-membership and disabled-identity tests.
- **Remote checks on driver update commit:** GitHub Actions [Go run #40423199616](https://github.com/parmtech/Tong-Panel/actions/runs/40423199616) passed (gofmt, `go test -race ./...`, `go vet ./...`); [SQLite schema run #40423199639](https://github.com/parmtech/Tong-Panel/actions/runs/40423199639) passed.
- **Not passed/not run locally:** full `go test ./...` is blocked on existing Linux-only `internal/preflight` syscall usage on Windows; `go test -race ./...` cannot run because the Windows environment lacks CGO; Python schema tests were not run because Python is unavailable. GitHub Actions on the session-store commit passed: Go checks [#40423834545](https://github.com/parmtech/Tong-Panel/actions/runs/40423834545) (`gofmt`, full `go test -race ./...`, `go vet ./...`) and SQLite schema checks [#40423834542](https://github.com/parmtech/Tong-Panel/actions/runs/40423834542).
- **Auth limitations:** the SQLite session store is not production login and is not wired to the API. No Argon2id verifier, TOTP enrollment/replay protection, CSRF/Origin middleware, rate limiting or browser session route exists. The original in-memory manager remains in the codebase.
- **Follow-up review correction:** read-only review identified that tenant suspension was not checked by the new session primitive. Issuance and every session/CSRF lookup now require an active tenant. On follow-up commit `bb617eaed9dc2b6e8e10e1b8380d85987feb1408`, Go checks [#40424296948](https://github.com/parmtech/Tong-Panel/actions/runs/40424296948) and SQLite schema checks [#40424296939](https://github.com/parmtech/Tong-Panel/actions/runs/40424296939) passed.
- **Additional follow-up review correction:** migration startup now refuses a database with a newer `user_version` instead of reapplying v1; regression test verifies version 2 stays unchanged. New queue rows now require an active tenant and enabled member identity; requester-mismatch idempotency conflicts remain intact. On code commit `252c53127a9606835b9c43af298d5bb958519f6f`, Go checks [#40424600275](https://github.com/parmtech/Tong-Panel/actions/runs/40424600275) and SQLite schema checks [#40424600243](https://github.com/parmtech/Tong-Panel/actions/runs/40424600243) passed. Focused local tests and vet passed for auth/database/jobs/API/health/daemon/RBAC; daemon build passed.
- **Queue limitations:** only `validation.noop` is accepted; the SQLite store has a lease/retry/audit lifecycle prototype but no executing worker or host operation. It is not wired into `tongd` pending actor identity and ADR/threat review. A future real worker also needs an ordered host-side fence checked before apply; database fencing alone cannot stop an already-running external side effect.
- **Dependency/security:** `modernc.org/sqlite` v1.38.2 is pinned to retain Go 1.23 compatibility; upstream package metadata identifies BSD-3-Clause and SQLite public domain. Full transitive licence inventory and ADR-003 review remain open.
- **Phase:** P1 remains IN PROGRESS. No Phase 1 acceptance or production-readiness claim.
- **Next one action:** request maintainer review of ADR-003, auth/session design, dependency licences and the #14/#15 parent chain before any API wiring or merge.

## LATEST SESSION — 2026-10-27, P1 job lease and recovery slice
- **Branch/PR:** `feature/p1-transactional-job-leases`, draft [PR #17](https://github.com/parmtech/Tong-Panel/pull/17), stacked on draft PR #16 (itself stacked on #15/#14); no live VPS changes.
- **Code:** added ordered migration v2 for availability scheduling and lease owner/token-hash fields; migration runner now serializes and applies each numbered migration atomically. Replaced SQLite's unrestricted manual state transition with token-fenced claim/start/renew/complete/fail operations, max-three-attempt retry policy, lease expiry recovery, SHA-256-only lease token storage and same-transaction attempt/audit records. Only `validation.noop` is eligible; this is not wired into the API/daemon and has no execution behavior.
- **Tests actually run:** local Go 1.25.14 Windows/amd64 `go test` and `go vet` passed for database/jobs/auth/API/health/daemon/RBAC packages. Coverage includes v1-to-v2 upgrade, newer-schema refusal, transaction serialization across two DB handles, unique concurrent claims, expiry/reclaim after reopen, stale token rejection, retry timing/limits, tenant suspension/disabled identity, and audit creation.
- **Review:** read-only review found no current token/tenant/state bypass in the no-op prototype. Before any external operation, the worker/agent needs a monotonic host-side fence checked at apply time; DB token fencing cannot stop an already-paused process from performing a real side effect after lease expiry.
- **Remote CI:** Go checks [#40425964059](https://github.com/parmtech/Tong-Panel/actions/runs/40425964059) passed (gofmt, full `go test -race ./...`, `go vet ./...`); SQLite schema checks [#40425964024](https://github.com/parmtech/Tong-Panel/actions/runs/40425964024) passed on code commit `b132a9e169aa2674adc82ae483595b5e04b0a889`.
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

**Phase:** P0 architecture/security review pending; P1 core development IN PROGRESS (draft code).
**Phase state:** Seven phase plans, draft architecture ADR register, threat model, test strategy and report published; approval and deeper threat review still pending.
**Implementation state:** P1 developer prototype remains in [draft PR #14](https://github.com/parmtech/Tong-Panel/pull/14); auth/session foundations are in [draft PR #15](https://github.com/parmtech/Tong-Panel/pull/15); SQLite/session prototypes are in [draft PR #16](https://github.com/parmtech/Tong-Panel/pull/16); the lease/recovery slice is in [draft PR #17](https://github.com/parmtech/Tong-Panel/pull/17). These drafts are unmerged and not production-ready. **P1 NOT COMPLETE.** No hosting app or installer exists.
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
