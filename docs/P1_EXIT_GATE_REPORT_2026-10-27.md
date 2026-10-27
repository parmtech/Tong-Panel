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
| P1-03 Authentication | Developer bearer, revocable SQLite sessions/CSRF and HTTPS-only cookie boundary; Argon2id budget #27, TOTP replay #28, encrypted enrollment #29, local one-shot Owner CLI #30; **#31 adds durable SQLite per-account/source admission gate** | **No reviewed production login wired to limiter or MFA, external key provisioning/rotation, remote authenticated enrollment/recovery, approved account lockout behavior, atomic session+MFA login transaction, cookie issuance/rotation/logout, or verified TLS front door** |
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
3. SQLite opt-in startup with fresh file: migrations reach version 6; **zero default users**; missing membership rejects token.
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
- **Required CI evidence:** [Go #40590273596](https://github.com/parmtech/Tong-Panel/actions/runs/40590273596), [SQLite schema #40590273502](https://github.com/parmtech/Tong-Panel/actions/runs/40590273502), [real CLI+daemon smoke #40590273330](https://github.com/parmtech/Tong-Panel/actions/runs/40590273330), [unsigned archive smoke #40590273329](https://github.com/parmtech/Tong-Panel/actions/runs/40590273329). Confirm final results before describing all as PASS.
- This is NOT an active login limiter until safely integrated with an approved login service. Its external HMAC key stability/rotation and hard account lockout DoS tradeoffs remain unresolved in [throttling limits](https://github.com/parmtech/Tong-Panel/blob/feature/p1-durable-auth-throttle/docs/P1_LOGIN_THROTTLE.md). The public network listener and actual credential issue path remain disabled.

## Publish decision

**STOP: do NOT merge/release as an installable hosting panel or begin Phase 2 implementation against real hosts yet.** Review of the stacked PR chain must preserve dependencies; independent password and UI branches (#18/#19) are not automatically integrated by stacked PR #25.

**Required to advance P1 → P2:** approved P0/ADR security boundaries; tested and recoverable account bootstrap + Argon2id + MFA + login rate limits + sessions; typed authorized agent contract and malicious input/UID tests; crash/lease/rollback; API and UI integration with accessibility evidence; real dependency/security scans; signed preview package + tamper test; clean Ubuntu/Debian VM installation/start/restart/rollback and backup/restore evidence; signed reviewer/owner acceptance.

**Next actionable step after CI:** review the complete unmerged PR stack plus P0 ADR-009, scope the rate-limited Argon2id + replay-protected MFA + reviewed TLS login vertical slice, and run clean supported-distro installer/recovery evidence before exit signoff. Do not close Phase 1 from a green smoke test alone.

**Responsible project:** ParmTech. Status authored from available repository/CI evidence, not an independent pen-test or signed owner release decision.
