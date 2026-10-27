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
| P1-03 Authentication | Static developer bearer, standalone SQLite sessions/CSRF and TLS cookie boundary; Argon2id primitive isolated in draft #18 | **No production credential bootstrap/login, integrated Argon2id, MFA enrollment/replay defense, bounded rate limiting, recovery, cookie issuance/logout, or verified TLS front door** |
| P1-04 Identity/RBAC | Tenant-role prototype; authenticated own-job read/cancel, current SQL role validation, no cross-tenant Owner wildcard | Full identity/team/reseller policies, permissioned global operator and end-to-end agent job identity binding |
| P1-05 Durable jobs | SQLite idempotency, lease fencing/retry/audit and queued cancellation; tests for race, expiry, and actor attribution | Real worker/host effect execution and monotonic fencing, safe rollback and crash/failure recovery, concurrency load tests |
| P1-06 Local agent | Non-root Unix socket proof-of-concept, peer UID, private socket directory, ping-only | **No privileged typed host actions** or external adversarial review; escalation/host mutation intentionally prohibited |
| P1-07 Metadata/migrations | SQLite STRICT metadata schema, 3 staged migrations, single writer and integrity tests | Schema migration backup/restore, corrupted database and mid-upgrade failure drills, encryption/secret rotation |
| P1-08 UI shell | Embedded CSS/JS static developer preview and TypeScript source | Functional auth/UI integration, responsive keyboard and screen-reader audits, notifications/search/jobs interaction E2E |
| P1-09 Diagnostics | Read-only OS/RAM/disk/port preflight via `tongctl doctor` | Safe redacted support bundle, log/diagnostic failure scenarios, candidate distro lab evaluation |
| P1-10 Packages/releases | **No signed experimental package, bootstrap installer or clean distro VM install evidence** | Immutable signed preview artifacts, SBOM/provenance, Ubuntu 24.04 + Debian 12 x86_64 clean VM install/upgrade/rollback/recovery |

**P0 prerequisites remain UNAPPROVED:** ADR-001 through ADR-008 are still draft per `docs/ADR_REGISTER.md` and the initial threat model is unreviewed. Review/signoff is a human responsibility; a coding assistant must not forge it. ADR-009 secure authentication plan also needs review.

## Dry-run matrix and how to reproduce it

The PR #25 source includes `scripts/smoke_p1.py` and `.github/workflows/p1-smoke.yml` which execute a compiled `tongd` binary **as a real local process**, not solely an httptest mock. The runner must demonstrate:

1. Fresh anonymous startup: `GET /healthz` and version; preview shell CSP; authenticated API denies missing configuration.
2. Opt-in developer token: unauthenticated denial, static bearer recognition, `validation.noop` job enqueue and idempotent retry, arbitrary command rejection, JSON-only cancellation disabled.
3. SQLite opt-in startup with fresh file: migrations reach version 3; **zero default users**; missing membership rejects token.
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

## Publish decision

**STOP: do NOT merge/release as an installable hosting panel or begin Phase 2 implementation against real hosts yet.** Review of the stacked PR chain must preserve dependencies; independent password and UI branches (#18/#19) are not automatically integrated by stacked PR #25.

**Required to advance P1 → P2:** approved P0/ADR security boundaries; tested and recoverable account bootstrap + Argon2id + MFA + login rate limits + sessions; typed authorized agent contract and malicious input/UID tests; crash/lease/rollback; API and UI integration with accessibility evidence; real dependency/security scans; signed preview package + tamper test; clean Ubuntu/Debian VM installation/start/restart/rollback and backup/restore evidence; signed reviewer/owner acceptance.

**Next actionable step after CI:** review the complete unmerged PR stack plus P0 ADR-009, scope the rate-limited Argon2id + replay-protected MFA + reviewed TLS login vertical slice, and run clean supported-distro installer/recovery evidence before exit signoff. Do not close Phase 1 from a green smoke test alone.

**Responsible project:** ParmTech. Status authored from available repository/CI evidence, not an independent pen-test or signed owner release decision.
