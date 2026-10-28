# Tong Panel — Phase 1 Exit / Phase 2 Entry Gate Report
**Prepared:** 2026-10-27 (Australia/Sydney)  
**Scope:** Exact Phase 1 requirements in `docs/phases/P1-CORE-FOUNDATION.md`; source tree at draft [PR #25](https://github.com/parmtech/Tong-Panel/pull/25).  
**Verdict:** **NOT PHASE 1 COMPLETE; NOT PUBLISH/INSTALL READY; PHASE 2 ENTRY BLOCKED.**

## Evidence policy

A passing Go or SQLite unit test means only those test contracts ran. It is **not** approval of product security, a working hosting panel, an OS-certified installation, a signed release, successful recovery, or an accessibility review. All changes in this report are on **draft, unmerged feature branches**. No production VPS was modified.

## Work-package reality (no invented completion percentages)

| P1 package | Verified code now in stacked drafts | Critical work left before exit |
| --- | --- | --- |
| P1-01 Repo/workflows | Go formatting + race tests + vet CI; SQLite schema tests, plus non-destructive daemon smoke workflow proposed in PR #25 | Frontend CI/a11y, SAST and dependency/license/SBOM scanning, release provenance |
| P1-02 Control-plane API | Loopback HTTP version/health, limited authenticated read/no-op job API, bounded JSON/errors | OpenAPI v1 and compatibility tests, hardened transport deployment, comprehensive API contract and abuse suite |
| P1-03 Authentication | Developer bearer, revocable SQLite sessions/CSRF and HTTPS cookie boundary; Argon2id #27, TOTP replay #28, encrypted MFA #29, local Owner #30, limiter #31, internal atomic login #32; **#33 adds SQLite v7 sliding idle expiry, atomic token rotation and audited logout primitives** | **No reviewed public HTTPS login/logout handler, trusted peer/proxy binding, approved external secret key custody, protected MFA recovery, browser session endpoints, abuse policy or security signoff** |
| P1-04 Identity/RBAC | Tenant-role prototype; authenticated own-job read/cancel, current SQL role validation, no cross-tenant Owner wildcard | Full identity/team/reseller policies, permissioned global operator and end-to-end agent job identity binding |
| P1-05 Durable jobs | SQLite idempotency, lease fencing/retry/audit and queued cancellation; tests for race, expiry, and actor attribution | Real worker/host effect execution and monotonic fencing, safe rollback and crash/failure recovery, concurrency load tests |
| P1-06 Local agent | Non-root Unix socket proof-of-concept, peer UID, private socket directory, ping-only | **No privileged typed host actions** or external adversarial review; escalation/host mutation intentionally prohibited |
| P1-07 Metadata/migrations | SQLite STRICT metadata schema, **6 staged migrations**, including TOTP replay (#28), encrypted MFA state (#29) and HMAC-keyed login attempt counters (#31); serialized writes and integrity tests | Schema migration backup/restore, corrupted database and mid-upgrade failure drills, external encryption/throttling key custody/rotation |
| P1-08 UI shell | Embedded UI now includes **panel-process-only** status/refresh, atomic live announcement, explicit nav labels and Node+TypeScript checks in stacked draft #26; combined Go/Node CI pass | Full functional auth/jobs UI, real-browser responsive/keyboard and screen-reader/WCAG audits, notifications/search/jobs interaction E2E |
| P1-09 Diagnostics | Read-only OS/RAM/disk/port preflight via `tongctl doctor` | Safe redacted support bundle, log/diagnostic failure scenarios, candidate distro lab evaluation |
| P1-10 Packages/releases | **Checksum-tested, explicitly unsigned Linux amd64 developer archive** produced by #25/26 CI; archive extracted and real daemon smoke repeated | **No signed package or bootstrap installer**, SBOM/provenance, Ubuntu 24.04 + Debian 12 x86_64 clean VM install/upgrade/rollback/recovery |

**P0 prerequisites remain UNAPPROVED:** ADR-001 through ADR-008 are still draft per `docs/ADR_REGISTER.md` and the initial threat model is unreviewed. Review/signoff is a human responsibility; a coding assistant must not forge it. ADR-009 secure authentication plan also needs review.

## Dry-run matrix and how to reproduce it

The P1 draft-stack source includes `scripts/smoke_p1.py` and `.github/workflows/p1-smoke.yml` which execute a compiled `tongd` binary **as a real local process**, not solely an httptest mock. The runner must demonstrate:

1. Fresh anonymous startup: `GET /healthz` and version; preview shell CSP; authenticated API denies missing configuration.
2. Opt-in developer token: unauthenticated denial, static bearer recognition, `validation.noop` job enqueue and idempotent retry, arbitrary command rejection, JSON-only cancellation disabled.
3. SQLite opt-in startup with fresh file: migrations reach version 7; **zero default users**; missing membership rejects token.
4. Explicit disposable fixture: create local test-only tenant and user; no-op job and queued cancellation; exactly one user-attributed audit event.
5. Disable test user in SQLite: subsequent request denied.
6. Read-only doctor emits four checks plus explicit *not certified* warning.

**CI VERIFIED:** Initial [P1 smoke run #40506869294](https://github.com/parmtech/Tong-Panel/actions/runs/40506869294) successfully built the daemon and passed ALL real-process smoke cases, but failed a doctor-warning test that searched for the wrong phrase. The assertion was fixed without changing the diagnostic code. Final [P1 smoke run #40507195050](https://github.com/parmtech/Tong-Panel/actions/runs/40507195050) **PASSED ALL STEPS**: daemon build, anonymous API/UI, journal no-op, SQLite v3 migration/membership/audit/revocation, read-only doctor JSON. This is Ubuntu GitHub runner evidence only, not signed OS installer or production VM install certification.

Existing evidence on parent branches: [Go race CI #40479073313](https://github.com/parmtech/Tong-Panel/actions/runs/40479073313), [SQLite schema CI #40436483021](https://github.com/parmtech/Tong-Panel/actions/runs/40436483021). These are **unit/integration** checks, not install certification.

### Local replication from a reviewed PR branch

```bash
go test -race ./...
go vet ./...
go build -o /tmp/tongd-smoke ./cmd/tongd
python3 scripts/smoke_p1.py /tmp/tongd-smoke
go run ./cmd/tongctl doctor
python3 -m unittest discover -s db -p 'test_*.py' -v
```

Only run in a **disposable lab environment** with a known free TCP 8090. The smoke script refuses to interfere with a preexisting local service, uses temporary test files, stops child processes, does not require root and must not be mistaken for an installer. The candidate `doctor` output is not OS certification. No SSH, DNS, ACME, real emails or customer services are touched.

## Latest combined verification — PR #26

This is **combined branch evidence**, not just separate PR evidence:

- [Go Actions #40508400960](https://github.com/parmtech/Tong-Panel/actions/runs/40508400960): **PASS** (gofmt, `go test -race ./...`, `go vet ./...`) on combined source including the UI.
- [UI Actions #40508400761](https://github.com/parmtech/Tong-Panel/actions/runs/40508400761): **PASS** (TypeScript 5.9 check and Node behavior tests for panel status).
- [Real-process dry-run #40508400737](https://github.com/parmtech/Tong-Panel/actions/runs/40508400737): **PASS** (compiled daemon, API/SQLite revocation and audit, embedded panel-status module and doctor).
- [Unsigned archive build/smoke #40508400655](https://github.com/parmtech/Tong-Panel/actions/runs/40508400655): **PASS** (Linux amd64 archive checksum, extraction, real daemon process smoke, read-only doctor, GitHub Actions artifact).
- [Earlier SQLite migration schema #40436483021](https://github.com/parmtech/Tong-Panel/actions/runs/40436483021): **PASS** with independent migration tests. PR #26 inherits these schemas, and combined Go suite exercised SQLite at runtime.

**Archive is UNSIGNED and marked DO NOT INSTALL.** Checksums protect an already trusted downloaded file against accidental corruption; they **do not authenticate publisher identity**. This is neither a publish-ready production release nor Ubuntu/Debian clean-VM certification. Source files on [draft PR #26](https://github.com/parmtech/Tong-Panel/pull/26) include ported UI work from independently reviewed-by-CI draft #19; [Argon2id draft #18](https://github.com/parmtech/Tong-Panel/pull/18) is not integrated. P0 human security approval remains pending.

## Authentication foundation continuation — PRs #27 and #28

- [Draft PR #27](https://github.com/parmtech/Tong-Panel/pull/27) integrates the standalone Argon2id PHC password primitive from independent [draft #18](https://github.com/parmtech/Tong-Panel/pull/18) into the combined source, plus a bounded 1–4-slot in-process verifier admission gate. New negative/race tests cover verifier capacity, no unbounded waiting and rejecting unknown PHC parameters. [Go Actions #40514729111](https://github.com/parmtech/Tong-Panel/actions/runs/40514729111), [daemon smoke #40514729807](https://github.com/parmtech/Tong-Panel/actions/runs/40514729807) and [unsigned archive lab #40514729051](https://github.com/parmtech/Tong-Panel/actions/runs/40514729051) **PASS**.
- [Draft PR #28](https://github.com/parmtech/Tong-Panel/pull/28) adds migration v4 and `auth.TOTPReplayGuard`, preventing acceptance of the same/older 30-second TOTP step across a user's tenant memberships and concurrently opened DB handles. Neither TOTP secret storage nor an MFA login endpoint exists; server-owned time and protected secret lookup remain required.
- **PR #28 CI:** Go [#40515875960](https://github.com/parmtech/Tong-Panel/actions/runs/40515875960), SQLite [#40515875837](https://github.com/parmtech/Tong-Panel/actions/runs/40515875837), daemon smoke [#40515875930](https://github.com/parmtech/Tong-Panel/actions/runs/40515875930), unsigned archive [#40515876140](https://github.com/parmtech/Tong-Panel/actions/runs/40515876140). **All four complete SUCCESS** at combined auth/code head `46790dcded3d49b044091a81386737ef26d1848f` (gofmt, full Go race suite, vet, SQLite v4 schema, compiled daemon + doctor, unsigned archive). A preceding Go run failed only gofmt and the exact CI diff was corrected. Later commits changed documentation, not executable code.
- **Important:** the in-process KDF limit is NOT distributed per-account/source throttling, and a replay ledger is NOT enrolled or cryptographically protected MFA. The security and installation P1 gates below remain blocked.

## Encrypted MFA seed and trusted-only enrollment — PR #29 (28 October 2026)

- [Draft PR #29](https://github.com/parmtech/Tong-Panel/pull/29) adds an AES-256-GCM `MFASecretBox` with randomized nonce, authenticated user-identity/key-version binding and explicit **external 32-byte key requirement**. No default or SQLite-stored master key. SQLite migration **v5** adds encrypted pending/active TOTP enrollment (10-minute pending lifetime) with identity foreign key and state invariants.
- Internal-only trusted caller methods generate a cryptographically random 160-bit seed, encrypt it before storage, and require a current TOTP proof to activate; replay step is consumed in the **same SQLite transaction** as activation. Active verification remains separately replay-protected, but **not atomic with session issuance**. There is no login/remote enrollment route, key provisioning, MFA recovery or public cookie issuance.
- **Verified CI for executable code head `ff284aff7052782f1cc2c11b97d3a66416ac87bb`:** [Go race/vet/format #40566200801](https://github.com/parmtech/Tong-Panel/actions/runs/40566200801) PASS; [Python SQLite schema #40566200785](https://github.com/parmtech/Tong-Panel/actions/runs/40566200785) PASS; [actual daemon/doctor smoke #40566200862](https://github.com/parmtech/Tong-Panel/actions/runs/40566200862) PASS; [unsigned developer archive #40566200800](https://github.com/parmtech/Tong-Panel/actions/runs/40566200800) PASS. An earlier formatting failure was corrected before this run. Technical boundaries: [Encrypted MFA prototype notes](https://github.com/parmtech/Tong-Panel/blob/feature/p1-encrypted-mfa-enrollment/docs/P1_MFA_ENCRYPTED_ENROLLMENT.md).
- **Publication status unchanged:** no reviewed external master key management, no production enrollment re-authentication, password/MFA session transaction or recovery, no signed installer or distro VM acceptance. Phase 1 exit remains BLOCKED.

## First-owner local bootstrap continuation — PR #30 (28 October 2026)

- [Draft PR #30](https://github.com/parmtech/Tong-Panel/pull/30) implements a **local-only, opt-in** first-owner bootstrap. `tongctl bootstrap-owner` requires an anonymous pipe for the password, explicit `TONG_ALLOW_LOCAL_OWNER_BOOTSTRAP=yes`, and a private owned 0600 SQLite file/0700 directory. It refuses symlinks, password-bearing CLI arguments and open directory/file modes. There is no default account or credential.
- `auth.BootstrapOwner` hashes Argon2id before SQLite writer lock and transactionally creates first tenant, identity, owner membership and human-attributed audit; any existing DB rows, concurrent second bootstrap and failed audit write deny/roll back creation.
- New Go unit tests and ephemeral real-CLI dry-run test `scripts/smoke_bootstrap_owner.py` are included in both daemon and extracted unsigned-archive CI workflows.
- **CI runs for branch:** [Go #40589039935](https://github.com/parmtech/Tong-Panel/actions/runs/40589039935), [real daemon+CLI dry-run #40589040002](https://github.com/parmtech/Tong-Panel/actions/runs/40589040002), [unsigned extracted archive+CLI smoke #40589039964](https://github.com/parmtech/Tong-Panel/actions/runs/40589039964). Check final workflow conclusion before claiming all green.
- **Still blocked:** first Owner cannot yet log in remotely with password+MFA; there is no approved key custody, secure recovery/reset, durable login throttling, atomic full auth+session or signed OS installer. Phase 1 exit/Phase 2 entry gates are unchanged.

## Durable login admission continuation — PR #31 (28 October 2026)

- [Draft PR #31](https://github.com/parmtech/Tong-Panel/pull/31) adds SQLite migration **v6** `login_attempt_buckets` with HMAC-SHA256 privacy-keyed account/source counters. Internal-only `SQLiteLoginLimiter.Admit` checks **5 attempts per account/tenant and 25 per trusted peer IP over 15 minutes**, within one `BEGIN IMMEDIATE` transaction. Multiple processes share SQLite counters and cannot consume the same last allowance simultaneously. Backward clock resets are denied, and global storage caps prevent unlimited growth. No account/IP/password is stored directly.
- Unit tests cover attempts distributed across source addresses, multiple accounts on one peer, two SQLite connections contending, expiry, invalid IPs and clock regression. Python schema v6 guards counters and bucket-key sizes. Compiled CLI/bootstrap and daemon/unsigned-archive smoke assert schema version 6.
- **Required CI evidence:** [Go #40590273596](https://github.com/parmtech/Tong-Panel/actions/runs/40590273596), [SQLite schema #40590273502](https://github.com/parmtech/Tong-Panel/actions/runs/40590273502), [real CLI+daemon smoke #40590273330](https://github.com/parmtech/Tong-Panel/actions/runs/40590273330), [unsigned archive smoke #40590273329](https://github.com/parmtech/Tong-Panel/actions/runs/40590273329). **All four workflows completed SUCCESS** for code commit `6e5f4522cdd5c9153d6eef11587a122845cb9073`: Go gofmt/race/vet, SQLite v6 constraints, compiled CLI/bootstrap+daemon smoke, and unsigned archive build/checksum/extracted binary smoke. Later commits updated documentation only.
- This is NOT an active login limiter until safely integrated with an approved login service. Its external HMAC key stability/rotation and hard account lockout DoS tradeoffs remain unresolved in [throttling limits](https://github.com/parmtech/Tong-Panel/blob/feature/p1-durable-auth-throttle/docs/P1_LOGIN_THROTTLE.md). The public network listener and actual credential issue path remain disabled.

## Internal authentication transaction — draft PR #32 (28 October 2026)

- [Draft PR #32](https://github.com/parmtech/Tong-Panel/pull/32) builds **internal-only** `auth.InternalLoginService` on top of the existing P1 prototypes, without adding any HTTP login endpoint. It invokes the SQLite account/source limiter **before** running bounded Argon2id, performs dummy KDF verification on missing/disabled account where feasible, and validates enrolled encrypted TOTP with server time.
- Under SQLite `BEGIN IMMEDIATE`, it rechecks unchanged password PHC, tenant, identity enablement, membership/role and active MFA envelope, then **atomically consumes an MFA step, inserts a hashed bearer+CSRF session and writes attributed `auth.login` audit**. A trigger-induced audit failure rolls back the session and restores MFA step eligibility.
- New auth regression tests exercise valid session/CSRF/revocation, wrong/unknown users, cross-tenant and disabled account, same-step replay, parallel OTP winner, persisted rate-limit denial, and fail-closed audit transaction. No real server account or VPS contacted.
- **CI VERIFIED PASS** for authentication code commit `c2cfadc0d6e85be13b5dfab7083a77405505caf6`: [Go gofmt/full `go test -race ./...`/`go vet ./...` #40591534457](https://github.com/parmtech/Tong-Panel/actions/runs/40591534457), [actual daemon/SQLite process dry-run #40591534497](https://github.com/parmtech/Tong-Panel/actions/runs/40591534497), and [unsigned developer archive build/extract/smoke #40591534190](https://github.com/parmtech/Tong-Panel/actions/runs/40591534190). All three workflows finished **SUCCESS**. This is not a public login integration or a security assessment.
- No public TLS listener, secure browser cookie delivery, trusted reverse proxy/client-IP derivation, encrypted master-key custody/rotation, recovery/login UI, device idle-timeout, external pen-test or signed installer. P1 completion and Phase 2 handoff remain **BLOCKED**.

## Internal session lifecycle hardening — draft PR #33 (28 October 2026)

- [Draft PR #33](https://github.com/parmtech/Tong-Panel/pull/33) adds SQLite **v7** `idle_deadline_unix`. Existing pre-v7 sessions are revoked during migration because their last activity cannot be trusted. New SQLite sessions and internal login sessions persist server-side idle deadlines.
- `SQLiteSessionStore.Check` requires a valid active tenant/identity/current role, absolute expiry and **30-minute sliding idle expiration**; it refreshes deadlines monotonically and permanently revokes sessions observed past an idle/absolute limit to prevent resurrection after clock rollback.
- Internal-only `Rotate` atomically revokes old bearer/CSRF, issues new random hashed bearer+CSRF with the **same original absolute expiry**, and audits user-attributed rotation. `Logout` revokes and audits in one transaction; audit failure rolls back revocation. No public HTTP/cookie issue/clear route enabled.
- New tests cover idle/absolute deadlines, concurrent reads, one-winner rotation, preserved expiry across repeated rotations, and audit-failure rollback. `scripts/smoke_p1.py`, bootstrap smoke and DB unit tests were updated for schema v7.
- **CI VERIFIED ALL PASS** at executable commit `6ae402888141e76aa83f38af5f96805598da5966`: [Go formatting/full `go test -race ./...`/`go vet ./...` #40598738018](https://github.com/parmtech/Tong-Panel/actions/runs/40598738018), [SQLite schema v7 #40598737975](https://github.com/parmtech/Tong-Panel/actions/runs/40598737975), [compiled daemon/SQLite smoke #40598738139](https://github.com/parmtech/Tong-Panel/actions/runs/40598738139), [unsigned developer archive/extraction/process smoke #40598737988](https://github.com/parmtech/Tong-Panel/actions/runs/40598737988). Initial gofmt differences were corrected; later commits are documentation-only. No production installation was certified.
- The full Phase 1 **release exit is still BLOCKED** by P0/ADR review, public secure login + origin binding, external key custody/recovery, idle-load testing, privileged agent trust, signed installer/SBOM, clean Ubuntu/Debian VM install/rollback/recovery, and UI accessibility/E2E.

## Publish decision

## Cookie Origin ambiguity hardening — stacked draft PR #34 (28 October 2026)

- The follow-up slice on [draft PR #34](https://github.com/parmtech/Tong-Panel/pull/34) makes cookie-authenticated mutations reject absent or duplicate `Origin` header values. This avoids authorizing the first value while an upstream component interprets another value.
- Focused tests deny two conflicting Origin headers and exercise 24 concurrent authenticated session reads. The existing cookie boundary still requires actual TLS, exact same-origin validation, CSRF, and a pre-issued session. No public login, cookie issue/rotate/logout HTTP route or proxy trust was added.
- Local Go formatting/tests/race/vet could not run because Go is unavailable in the current Windows workspace. Code head `6c983a22de1e5b1ea0de133ea28c6d1daadc490c` passed [Go formatting, full `go test -race ./...`, and vet #40599907640](https://github.com/parmtech/Tong-Panel/actions/runs/40599907640), [daemon smoke #40599907840](https://github.com/parmtech/Tong-Panel/actions/runs/40599907840), and [unsigned archive lab #40599907588](https://github.com/parmtech/Tong-Panel/actions/runs/40599907588). These are hosted CI prototype checks, not release certification or security approval.
- This does not satisfy public HTTPS login, ADR-009 approval, origin configuration, external key custody/recovery, security review or any remaining installer/agent/UI release gate. P0 and Phase 1 exit remain blocked; no production VPS was changed.

## P2 isolated file primitive checkpoint — draft PR #36 (28 October 2026)

- Draft PR [#36](https://github.com/parmtech/Tong-Panel/pull/36), stacked on #35, extends the unprivileged Linux file prototype: create-only writes hold a stage lock, and bounded stale cleanup now recognizes both write and archive stages. The tenant wrapper continues to select only the configured pinned root.
- Focused negative/race tests cover stale write-stage removal, active-lock skip and later cleanup, symlink/outside-root preservation, unrelated names, and tenant-root scope. Exact executable code commit `d7b1ae492e75355622b24b779b206640f99eba56` passed [Go checks #40609720874](https://github.com/parmtech/Tong-Panel/actions/runs/40609720874), [daemon smoke #40609720701](https://github.com/parmtech/Tong-Panel/actions/runs/40609720701), and [unsigned package lab #40609720693](https://github.com/parmtech/Tong-Panel/actions/runs/40609720693). A later PR-head commit changes documentation only.
- This is not a Files API/UI, user/object authorization, quota system, startup recovery integration, installer, site/domain/database/SSL lifecycle, or two-tenant end-to-end result. P0 ADR/threat review and P1 authentication, trusted-agent, installer, and recovery gates remain open. No production host or VPS was modified.
- **Phase 2 status remains NOT COMPLETE and gated.** This isolated prototype does not permit merging/releasing or implementing host mutations against real servers.
## P2 disposable clean-VM smoke lane — draft PR #37 (28 October 2026)

- Draft PR [#37](https://github.com/parmtech/Tong-Panel/pull/37), stacked on #36, adds a disposable QEMU/TCG run for candidate Ubuntu 24.04 and Debian 12 cloud images. It builds the explicitly unsigned developer preview, runs it as an unprivileged guest user, verifies the archive checksum/OS identity, and exercises the existing daemon/SQLite smoke and read-only doctor check.
- Both matrix jobs passed [clean VM run #40610418570](https://github.com/parmtech/Tong-Panel/actions/runs/40610418570) for workflow/code commit `de431b6ccbe6f5b418a11f94eb483f24a2b4e205`.
- This is clean-guest runtime evidence only. No installer, service install/restart, rollback, restore, supported OS matrix certification, or P2 end-to-end hosting lifecycle is tested. P0/P1 gates remain open; no live host was changed.
- **Phase 2 status remains NOT COMPLETE and gated.** Do not treat this infrastructure or the unsigned preview as permission to install or deploy.
## P2 Files clean-guest test matrix — draft PR #44

- Draft PR [#44](https://github.com/parmtech/Tong-Panel/pull/44), stacked on the current P2 draft chain, builds the Linux `internal/files` test binary and runs it inside fresh unprivileged Ubuntu 24.04 and Debian 12 QEMU guests. Changes to `internal/files/**` trigger this workflow.
- Both guest jobs passed [clean VM workflow #40614167120](https://github.com/parmtech/Tong-Panel/actions/runs/40614167120), including the filesystem suite, daemon/SQLite smoke and read-only doctor check.
- This checks the primitives against candidate guest kernels only. It does not install the panel, certify either OS, test service lifecycle/rollback/restore, or satisfy P2 hosting E2E. No production host changed; P0/P1 review remains open.

## P2 root-close-safe streamed file upload — draft PR #42

- Draft PR [#42](https://github.com/parmtech/Tong-Panel/pull/42), stacked on #41, duplicates the pinned root descriptor before consuming the caller's bounded stream. The root lifecycle lock is released before I/O; close prevents new operations but need not wait for a blocked upload, and the in-flight operation can finish using its pinned descriptor copy.
- A gated-reader regression test verifies prompt close, safe completion/publication after close and rejection of post-close reads.
- Code commit `faa220eed997f68b225dc105cc4386227d742dfe` passed [Go formatting/full race/vet #40613452838](https://github.com/parmtech/Tong-Panel/actions/runs/40613452838), [daemon smoke #40613452810](https://github.com/parmtech/Tong-Panel/actions/runs/40613452810), and [unsigned package lab #40613452879](https://github.com/parmtech/Tong-Panel/actions/runs/40613452879).
- Caller cancellation/deadline and disk/concurrency quotas remain outside this primitive. No API/auth/ACL or host service changes; P0/P1 remain open and P2 incomplete.

## P2 bounded streaming create-only file upload — draft PR #41

- Draft PR [#41](https://github.com/parmtech/Tong-Panel/pull/41), stacked on draft #40, refactors `WriteNewFrom` to stream request bytes directly into a private staging file, using constant application buffer memory and at most the caller cap+1 reads (cap ≤16 MiB). The file is published only after EOF within the cap through atomic no-replace rename; overflow or source error discards the stage.
- Negative/race coverage includes over-limit and reader-error non-publication, stage cleanup, invalid/closed-root rejection, concurrent one-winner creation, and root close synchronization during a blocked stream.
- Executable commit `df450a054fe8df50333e8e820f4145601e8d75f8` passed [Go format/full race/vet #40612826353](https://github.com/parmtech/Tong-Panel/actions/runs/40612826353), [daemon smoke #40612826486](https://github.com/parmtech/Tong-Panel/actions/runs/40612826486), and [unsigned package lab #40612826389](https://github.com/parmtech/Tong-Panel/actions/runs/40612826389).
- No global concurrency limit, tenant-wide disk quota, deadline, HTTP/auth/ACL, installer or host lifecycle is added. P0/P1 review remains open and P2 remains incomplete. No production host changed.

## P2 internal two-tenant Files lifecycle regression — draft PR #40

- Draft PR [#40](https://github.com/parmtech/Tong-Panel/pull/40), stacked on draft #39, adds integration-style coverage across two distinct temporary pinned tenant roots.
- The test exercises bounded and create-only writes, buffered and streamed archive extraction, per-tenant read/list, tenant-scoped removal, unknown-tenant denial and cross-root traversal denial.
- Test commit `28bde9fee671c0a8438337ec5017c5d564b778d7` passed [Go formatting/full `go test -race ./...`/vet #40612116106](https://github.com/parmtech/Tong-Panel/actions/runs/40612116106), [daemon smoke #40612116078](https://github.com/parmtech/Tong-Panel/actions/runs/40612116078), and [unsigned package smoke #40612116013](https://github.com/parmtech/Tong-Panel/actions/runs/40612116013).
- This is internal selector integration evidence, not authenticated HTTP, user/object authorization, clean install/recovery, or the roadmap's complete two-tenant acceptance gate. P0/P1 remain open and P2 remains incomplete. No production host changed.

## P2 bounded streamed create-only file writes — draft PR #39

- Draft PR [#39](https://github.com/parmtech/Tong-Panel/pull/39), stacked on draft #38, adds bounded reader ingestion for create-only file writes through the pinned root and tenant-root selector.
- Executable code commit `aca299ed6090de6c327f78700be28086ee785502` passed [Go formatting/full `go test -race ./...`/vet #40611776793](https://github.com/parmtech/Tong-Panel/actions/runs/40611776793), [daemon smoke #40611776807](https://github.com/parmtech/Tong-Panel/actions/runs/40611776807), and [unsigned package lab #40611776733](https://github.com/parmtech/Tong-Panel/actions/runs/40611776733).
- Focused tests cover exact cap, bounded cap+1 consumption, rejection without target publication, invalid path/limit/nil input, closed root, unknown tenant before reader consumption, tenant separation and concurrent single-winner publication.
- This holds a bounded input buffer (caller cap at most 16 MiB); it is not a tenant-wide quota, request timeout, replacement write, API/auth/ACL integration, installer or P2 acceptance gate. No live host changed; P0/P1 remain open and P2 remains incomplete.

## P2 bounded archive stream primitive — draft PR #38

- Draft PR [#38](https://github.com/parmtech/Tong-Panel/pull/38), stacked on the clean-VM smoke work, adds a bounded `io.Reader` entry point for gzip/tar extraction. Tests cover exact/over-limit reads, no target on oversize, successful extraction, and tenant denial before stream consumption.
- Executable code commit `fd418e1769efef5e5b21a7e7e3ac722fe31aba51` passed [Go formatting/full `go test -race ./...`/vet #40611138073](https://github.com/parmtech/Tong-Panel/actions/runs/40611138073), [daemon smoke #40611138018](https://github.com/parmtech/Tong-Panel/actions/runs/40611138018), and [unsigned package lab #40611138068](https://github.com/parmtech/Tong-Panel/actions/runs/40611138068). Local Go is unavailable; subsequent PR-head changes are documentation-only.
- This does not add an HTTP upload route, user/object authorization, quotas, request timeouts, scanning, installation, or service lifecycle. It is not a P2 acceptance result.
**STOP: do NOT merge/release as an installable hosting panel or begin Phase 2 implementation against real hosts yet.** Review of the stacked PR chain must preserve dependencies; independent password and UI branches (#18/#19) are not automatically integrated by stacked PR #25.

**Required to advance P1 → P2:** approved P0/ADR security boundaries; tested and recoverable account bootstrap + Argon2id + MFA + login rate limits + sessions; typed authorized agent contract and malicious input/UID tests; crash/lease/rollback; API and UI integration with accessibility evidence; real dependency/security scans; signed preview package + tamper test; clean Ubuntu/Debian VM installation/start/restart/rollback and backup/restore evidence; signed reviewer/owner acceptance.

**Next actionable step after CI:** review the complete unmerged PR stack plus P0 ADR-009, scope the rate-limited Argon2id + replay-protected MFA + reviewed TLS login vertical slice, and run clean supported-distro installer/recovery evidence before exit signoff. Do not close Phase 1 from a green smoke test alone.

**Responsible project:** ParmTech. Status authored from available repository/CI evidence, not an independent pen-test or signed owner release decision.
## P2 clean-VM image pinning — draft PR #45 (28 October 2026)

- Draft PR [#45](https://github.com/parmtech/Tong-Panel/pull/45) is stacked on #44. It pins Ubuntu 24.04 to official release `20261017` with SHA-256 and Debian 12 to official build `20261026-2643` with SHA-512. The downloaded image is verified before QEMU overlay creation or guest boot.
- Focused Bash checks cover both algorithms and reject malformed/wrong digests, altered images and missing files. Local checksum tests, shell syntax checks and `git diff --check` passed.
- Both fresh guest jobs passed on exact code commit `c9b4767eaa6722af22f31e742c50600ea2b31440`: [clean-VM matrix #40615254194](https://github.com/parmtech/Tong-Panel/actions/runs/40615254194). The matrix also ran the internal Files suite, daemon/SQLite smoke and read-only doctor.
- **Gate status:** this is repeatable unsigned developer-preview execution and image-integrity evidence only. It does not validate an installer, service lifecycle, rollback, restore, distro certification, authenticated API/ACL or the P2 end-to-end hosting lifecycle. ADR-001–009/threat-model owner-security review remains pending; Phase 1 and P2 remain incomplete. No production host changed.
## P2 inclusive expanded archive limit — draft PR #46 (28 October 2026)

- Draft PR [#46](https://github.com/parmtech/Tong-Panel/pull/46) is stacked on #45. It fixes an inclusive-boundary error in gzip/tar extraction by allowing the decompressed stream to end exactly at `ArchiveLimits.ExpandedBytes`, while reading one byte beyond the limit to reject overflow.
- The focused regression test measures the valid archive size; exact-limit extraction succeeds, while a limit one byte smaller returns `ErrArchiveLimit` without publishing the target or leaving a stage.
- Local Go is unavailable. Exact code commit `07d18da765610c4ccc185f4b96358bf95a8cee13` passed [Go formatting/full race/vet #40616565133](https://github.com/parmtech/Tong-Panel/actions/runs/40616565133), [daemon smoke #40616565143](https://github.com/parmtech/Tong-Panel/actions/runs/40616565143), [unsigned package lab #40616565239](https://github.com/parmtech/Tong-Panel/actions/runs/40616565239), and both clean guests in [VM matrix #40616565111](https://github.com/parmtech/Tong-Panel/actions/runs/40616565111).
- **Gate status:** this is an internal file-extraction correction only. It does not satisfy authenticated Files API/UI, quotas, installer/service lifecycle, backup restore or the full two-tenant/end-to-end P2 exit gate. ADR/threat-model owner-security review remains pending; P2 incomplete. No production host changed.


## P2 gzip framing regression coverage — draft PR #47 (28 October 2026)

- Draft [PR #47](https://github.com/parmtech/Tong-Panel/pull/47), stacked on #46, rejects malformed gzip framing through regression tests for corrupted trailer checksum, a second concatenated gzip member and a trailing byte. Each case verifies failure does not publish the target and removes private staging state.
- Exact code commit `12ee26f28b038e64a9930be4dc77ef55a85339ae` passed [Go formatting/full race/vet #40617596259](https://github.com/parmtech/Tong-Panel/actions/runs/40617596259), [daemon dry-run #40617596170](https://github.com/parmtech/Tong-Panel/actions/runs/40617596170), [unsigned archive #40617596192](https://github.com/parmtech/Tong-Panel/actions/runs/40617596192), and both Ubuntu 24.04/Debian 12 clean guests in [VM matrix #40617596202](https://github.com/parmtech/Tong-Panel/actions/runs/40617596202).
- **Gate status:** internal archive regression coverage only. ADR-001–009 and threat-model owner/security review remain pending. Phase 1 remains unapproved and Phase 2 incomplete; authenticated Files API/UI, quotas, installer and rollback, hosting lifecycle and backup/restore evidence remain outstanding. No production host changed.


## P2 concurrent archive publication race — draft PR #48 (28 October 2026)

- Draft [PR #48](https://github.com/parmtech/Tong-Panel/pull/48), stacked on #47/#46, launches 16 concurrent extractions of two different valid trees against one target. The test requires exactly one successful atomic publication, every loser to receive `ErrAlreadyExists`, the resulting tree to match one complete archive, and all private stages to be cleaned.
- Exact code commit `beaca356887c645520c144cc9fb0fa6cdfd8eeb1` passed [Go formatting/full race/vet #40618389233](https://github.com/parmtech/Tong-Panel/actions/runs/40618389233), [daemon dry-run #40618389360](https://github.com/parmtech/Tong-Panel/actions/runs/40618389360), [unsigned archive #40618389336](https://github.com/parmtech/Tong-Panel/actions/runs/40618389336), and Ubuntu 24.04/Debian 12 guests in [VM matrix #40618389193](https://github.com/parmtech/Tong-Panel/actions/runs/40618389193).
- **Gate status:** internal race coverage only. ADR-001–009 and threat-model owner/security review remain pending. Authenticated Files API/UI, quotas, installer/rollback, site/domain/database/SSL lifecycle and backup/restore evidence remain outstanding; P2 is incomplete and no production host changed.


## P2 concurrent two-tenant archive isolation — draft PR #49 (28 October 2026)

- Draft [PR #49](https://github.com/parmtech/Tong-Panel/pull/49), stacked on #48/#47/#46, tests 16 simultaneous create-only extractions per tenant to the same relative site path under two distinct pinned roots. It requires one winner in each tenant, `ErrAlreadyExists` for every same-tenant loser, tenant-specific content on readback, and no leaked stages.
- Exact code commit `141e0aab6119c346f2ff761c4bb1ecc2da62beae` passed [Go formatting/full race/vet #40618958111](https://github.com/parmtech/Tong-Panel/actions/runs/40618958111), [daemon dry-run #40618958138](https://github.com/parmtech/Tong-Panel/actions/runs/40618958138), [unsigned archive #40618957985](https://github.com/parmtech/Tong-Panel/actions/runs/40618957985), and Ubuntu 24.04/Debian 12 in [VM matrix #40618958036](https://github.com/parmtech/Tong-Panel/actions/runs/40618958036).
- **Gate status:** internal pinned-root concurrency evidence only. ADR-001–009 and threat-model owner/security review remain pending. Authenticated Files API/UI, quotas, supported signed installer and recovery, site/domain/database/SSL lifecycle, offsite backups and clean-host restore remain unproven. Phase 2 is incomplete; no production host changed.


## P2 fail-closed pre-mutation gate — draft PR #50 (28 October 2026)

- Draft [PR #50](https://github.com/parmtech/Tong-Panel/pull/50), stacked on #49, adds pure `EvaluateMutationGate` logic for a future installer/host mutator. Its caller must supply a nonzero minimum disk threshold and required TCP ports. It blocks unsupported OS, malformed OS facts, low or unknown disk, occupied or unknown required ports, and incomplete/invalid requirements.
- The gate is read-only and not yet connected to an installer; `Allowed` means only that the supplied facts satisfy the supplied policy.
- Corrected code head `55544dcb3a6505a4967fa67ad44046e6111d1b0b` passed [Go formatting/full race/vet #40619972006](https://github.com/parmtech/Tong-Panel/actions/runs/40619972006), [daemon #40619972004](https://github.com/parmtech/Tong-Panel/actions/runs/40619972004), and [unsigned archive #40619972005](https://github.com/parmtech/Tong-Panel/actions/runs/40619972005). Initial gofmt alignment failed and was corrected; all checks listed here are from the corrected head.
- **Gate status:** no actual installer, supported OS policy approval, rollback, restore, signed package or P2 lifecycle acceptance. ADR-001–009 and threat-model owner/security review remain pending. P2 incomplete; no production host changed.

## P2 authenticated tenant-scoped Files read API — draft PR #51 (28 October 2026)

- Draft [PR #51](https://github.com/parmtech/Tong-Panel/pull/51), stacked on #50, adds opt-in list/read handlers behind existing protected API middleware. Tenant scope comes from the verified principal; Owner/Admin/Reseller/Customer are permitted while Auditor and unknown roles are denied. Requests reject duplicate/unknown query keys, traversal and symlink escapes, and enforce 1,000-entry and 16 MiB response caps.
- Focused negative coverage includes role denial, malformed query parameters, tenant separation, path traversal and bounds. The API remains unconfigured in the daemon by default and adds no public login or mutation route.
- Corrected code head `01be544dca165c9554e5f429bc3029c0015ca009` passed hosted Go formatting/full race/vet [#40621248079](https://github.com/parmtech/Tong-Panel/actions/runs/40621248079), daemon smoke [#40621248030](https://github.com/parmtech/Tong-Panel/actions/runs/40621248030), and unsigned package lab [#40621248033](https://github.com/parmtech/Tong-Panel/actions/runs/40621248033). Local Go is unavailable.
- **Gate status:** authenticated API prototype only. No Files UI, write/delete/upload route, object ACL, quota enforcement, daemon lifecycle, public login, installer, backup/restore or P2 end-to-end evidence. ADR-001–009 and threat-model owner/security review remain pending. Phase 1 and P2 are incomplete; no production host changed.


## P2 persistent tenant Files quota reservations — draft PR #53 (29 October 2026)

- Draft [PR #53](https://github.com/parmtech/Tong-Panel/pull/53), stacked on #52, adds migration v8 with tenant-bound quota state and durable per-upload reservations. Reserve, commit and release update used/reserved bytes transactionally; quota rows reference existing tenants, enforce the configured limit without signed-integer overflow, and refuse reconfiguration while reservations remain.
- Focused tests exercise unconfigured/over-quota denial, actual-byte commit, release, tenant separation, zero-byte reservations, pending-reservation survival across database reopen, and concurrent reservations through two independent SQLite handles. The migration version change also updates local bootstrap and developer smoke expectations.
- Exact final code head `f9a67c534bc3276b3f04037dbc94f67b1043392b` passed [Go formatting/full race/vet #40703478052](https://github.com/parmtech/Tong-Panel/actions/runs/40703478052), [SQLite schema checks #40703478117](https://github.com/parmtech/Tong-Panel/actions/runs/40703478117), [daemon/bootstrap smoke #40703478331](https://github.com/parmtech/Tong-Panel/actions/runs/40703478331), [unsigned package lab #40703477994](https://github.com/parmtech/Tong-Panel/actions/runs/40703477994), and [Ubuntu 24.04/Debian 12 clean-VM smoke #40703477997](https://github.com/parmtech/Tong-Panel/actions/runs/40703477997). Local Go/gofmt are unavailable.
- **Gate status:** this is a persistent accounting primitive, not a complete quota lifecycle. It has no trusted filesystem scanner, startup barrier or recovery of crash-stranded reservations. There is no delete accounting, Files UI/object ACL, daemon quota wiring, installer/service lifecycle or backup/restore proof. ADR-001–009 and threat-model owner/security review remain pending. Phase 1 and P2 remain incomplete; no production host changed.

## P2 quota-gated authenticated Files upload API — draft PR #52 (28 October 2026)

- Draft [PR #52](https://github.com/parmtech/Tong-Panel/pull/52), stacked on #51, adds create-only upload behind protected middleware. Same-tenant `files:write` RBAC is required; Auditor/unknown roles are denied. It accepts only `application/octet-stream`, a single strict relative path and a maximum 16 MiB payload. It reserves quota before consuming bytes and uses the existing staged atomic no-replace write.
- Negative/race coverage includes quota denial before body reads, quota commit/release, cross-tenant isolation, Auditor denial, CSRF/auth, declared oversize, duplicate destination, concurrent same-target writes, and accounting when publication durability or quota commit is uncertain.
- Corrected code head `c63709af69c29d01e0ed683ac1d8ea04cc9d9a6e` passed Go format/full race/vet [#40622572789](https://github.com/parmtech/Tong-Panel/actions/runs/40622572789), daemon smoke [#40622572813](https://github.com/parmtech/Tong-Panel/actions/runs/40622572813), unsigned package lab [#40622572868](https://github.com/parmtech/Tong-Panel/actions/runs/40622572868), and Ubuntu 24.04/Debian 12 disposable VM smoke [#40622572815](https://github.com/parmtech/Tong-Panel/actions/runs/40622572815). Local Go is unavailable.
- **Gate status:** this API requires an injected quota contract but has no persistent quota ledger, usage reconciliation or delete accounting. No daemon wiring, Files UI/object ACL, installer, service lifecycle, offsite backup or clean-host restore evidence. ADR-001–009 and threat-model owner/security review remain pending. Phase 1 and P2 are incomplete; no production host changed.


## P2 authenticated tenant Files deletion accounting — draft PR #54 (29 October 2026)

- Draft [PR #54](https://github.com/parmtech/Tong-Panel/pull/54), stacked on #53, adds DELETE to the protected Files content route. It requires `files:write`, strict relative-path validation and existing CSRF/origin checks.
- The pinned-root delete atomically moves the actual directory entry into a private locked stage before reading its type and size, then unlinks and syncs directories. This avoids relying on a pre-unlink path stat for quota accounting under symlink replacement races. Symlinks count as zero. SQLite accounting rejects negative/underflow decrements and is updated only after durable unlink.
- Focused coverage includes role/CSRF/path negatives, tenant isolation, duplicate deletion, regular-file versus symlink byte counts, underflow and existing symlink-swap race regression. Exact code head `4ff8529f347dd2f2dac81523852d6a6790016cf1` passed [Go formatting/full race/vet #40708184553](https://github.com/parmtech/Tong-Panel/actions/runs/40708184553), [daemon smoke #40708184530](https://github.com/parmtech/Tong-Panel/actions/runs/40708184530), [unsigned package lab #40708184387](https://github.com/parmtech/Tong-Panel/actions/runs/40708184387), and [Ubuntu 24.04/Debian 12 clean VM #40708184391](https://github.com/parmtech/Tong-Panel/actions/runs/40708184391).
- **Gate status:** a process crash after unlink but before quota decrement can conservatively overcount. Trusted filesystem usage reconciliation and startup recovery are still required before daemon wiring. No Files UI/object ACL, installer/service lifecycle, backup/restore or owner/security approval. P2 remains incomplete; no production host changed.

## P2 measured tenant Files quota setup — draft PR #55 (29 October 2026)

- Draft [PR #55](https://github.com/parmtech/Tong-Panel/pull/55), stacked on #54, adds a bounded Linux scan of a server-selected pinned tenant root and configures initially unconfigured quota from measured logical regular-file bytes.
- The scanner serializes with writes, deletes, extraction and stage cleanup through the same `Root`; uses descriptor-relative no-follow/no-cross-mount traversal; skips symlink payloads; and rejects detected path changes, special files, entry/depth limits and integer overflow. It skips only valid private root stage directories.
- Focused tests cover nested files, symlink and private-stage behavior, special-file and scan limits, synchronization with an in-flight streamed write, tenant isolation and pending-reservation denial during reconfiguration. Exact code head `2a56d93d250b4ad352831da97db6a7a8b0a8c134` passed [Go formatting/full race/vet #40711216528](https://github.com/parmtech/Tong-Panel/actions/runs/40711216528), [daemon smoke #40711216435](https://github.com/parmtech/Tong-Panel/actions/runs/40711216435), [unsigned package lab #40711216395](https://github.com/parmtech/Tong-Panel/actions/runs/40711216395), and [Ubuntu 24.04/Debian 12 clean VM #40711216498](https://github.com/parmtech/Tong-Panel/actions/runs/40711216498).
- **Gate status:** callers must quiesce external writers; this is not a cross-process startup barrier and does not recover crash-stranded reservations. No daemon quota wiring, Files UI/object ACL, installer/service lifecycle, backup/restore or owner/security approval. P2 remains incomplete; no production host changed.

## P2 cross-process Files mutation barrier — draft PR #56 (29 October 2026)

- The isolated branch `feature/p2-quota-maintenance-barrier` adds a Linux per-tenant advisory shared/exclusive barrier in a service-owned lock directory. Authenticated upload/delete routes hold a shared lease across quota reservation or removal, filesystem mutation and accounting finalization; exclusive maintenance waits for those API mutations.
- `ReconcileTenantQuotaFromRoots` requires a `TenantFileQuiescer`, then obtains the exclusive API barrier, bounded-scans the pinned root and atomically refreshes usage while deleting crash-stranded reservations. Focused tests cover subprocess lock exclusion, mutation drain, tenant independence, cancellation, unsafe lock-directory permissions, measured recovery, and quiescer/scan/over-limit fail-closed behavior.
- Pre-recovery code head `1364c68ab32faa0f90c1f01efe65d50cf100525f` passed [Go formatting/full race/vet #40714421755](https://github.com/parmtech/Tong-Panel/actions/runs/40714421755), [daemon smoke #40714421877](https://github.com/parmtech/Tong-Panel/actions/runs/40714421877), and [unsigned package #40714421724](https://github.com/parmtech/Tong-Panel/actions/runs/40714421724). Two earlier Go runs failed on trailing blank-line formatting only; the corrected head passed. Clean-VM head `680c210cd97cdeeca4b9794be205fe5d986032d1` passed Debian but failed Ubuntu because test temporary lock directories were rejected by the permission check; the harness now explicitly creates a 0700 lock directory and reports owner/mode details. Recovery and harness fixes await exact-head checks.
- **Limits/gate status:** the lock only coordinates participating Tong Panel API processes. No host-level quiescer adapter or daemon/startup wiring exists; without one, recovery is not enabled. It does not itself quiesce PHP/WordPress, shell or other external filesystem writers. No signed installer, backup/restore, ADR/threat-model owner-security approval or P2 completion is claimed. No production host changed.
