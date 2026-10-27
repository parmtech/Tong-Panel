# P1 Authentication and SQLite Metadata — Development Checkpoint

**Date:** 2026-10-26, Australia/Sydney | **Status:** implemented foundations with unit tests; not integrated or production ready.

## Implemented in stacked feature branch
- `internal/auth/sessions.go`: in-memory cryptographically random 32-byte session and CSRF tokens, SHA-256 hashed storage, bounded expiration (<=24 hours), per-session and per-user revocation, an HTTPS-only host-bound `__Host-` cookie helper and a delete-cookie helper.
- `internal/auth/sessions_test.go`: expiry/revocation, CSRF mismatch, invalid role/duration, concurrent independent issuance and cookie security flags.
- `db/migrations/0001_initial.sql`: first SQLite STRICT schema for tenants, identities, role memberships, hashed sessions, jobs, job attempts and audit events; tenant-key constraints and tenant-scoped idempotency uniqueness.
- `db/test_schema.py`: migration and persistence round trip, disallowed tenant/identity relationships, idempotency uniqueness per tenant, invalid state/role and plaintext password-hash rejection.
- `.github/workflows/schema-checks.yml`: dependency-free schema test workflow.

## Validation
- Local `go test -race ./internal/auth` passed.
- Local `go test ./...` and `go vet ./...` passed on the developer-preview base.
- Local Python 3 `python -m unittest discover -s db -p 'test_*.py' -v` passed (4 tests).
- GitHub Actions verified: [Go checks #40310453646](https://github.com/parmtech/Tong-Panel/actions/runs/40310453646) passed (gofmt, go test -race, go vet); [SQLite schema checks #40310453620](https://github.com/parmtech/Tong-Panel/actions/runs/40310453620) passed (all 4 schema tests). PR remains draft/unmerged.

## Essential limitations and next steps
- There is **no login endpoint**, password verifier/Argon2id integration, TOTP enrollment/replay protection, session persistence, token rotation, CSRF/origin middleware or multi-factor enforcement. Session manager is an independent primitive **not wired to HTTP**; no cookies are being issued to users yet.
- SQL migration is tested against SQLite via Python's builtin driver but **not yet integrated with Go `database/sql`** or the existing JSON job journal. The Go API still uses the older developer token and local JSON journal, not SQL.
- No production secrets/key management; no database upgrade/rollback runner, job worker leases or authenticated backup. No root/installer changes made.
- `PRAGMA foreign_keys = ON` must be set on each SQLite connection; schema tests enforce this in setup but actual Go adapter must do the same.
- **Phase 1 remains IN PROGRESS.** Next task: review authentication threat model/ADR, add Argon2id hashing with vetted pinned dependency, transactional SQLite driver/migrations and session revocation persistence, then connect auth to API behind protected TLS and rate-limited endpoints; test cross-tenant and restart conditions.

## Workflow
Stacked from `feature/p1-core-foundation` in new review branch; merge its parent PR first to avoid duplicating changes. Do not deploy this branch to a live VPS.

## 2026-10-27 follow-on — Go SQLite and queue prototype

On `feature/p1-sqlite-transactional-queue`, stacked on PR #15:

- Moved the canonical schema to `internal/database/migrations/0001_initial.sql` and embed it in the Go binary. The Python schema tests now read that same source file.
- Added `internal/database.Open`: creates a private parent directory, configures `foreign_keys`, a bounded busy timeout, WAL and FULL synchronous mode through per-connection driver pragmas, applies the schema to a new database, rejects a newer `user_version`, and limits the Go connection pool to one connection for this single-process prototype.
- Added an opt-in `jobs.SQLiteStore` prototype. New job creation is a SQL transaction tied to an active tenant membership and enabled identity; idempotent retries return the same job, cross-tenant reads are scoped, and transitions are validated and committed with a state predicate.
- Added `auth.SQLiteSessionStore` as an alternative to the in-memory session primitive. It stores only SHA-256 session and CSRF token hashes, persists expiry and revocation, supports user-wide revocation, and re-reads current membership role and disabled identity state for each check. It is not connected to the login flow or API.
- Added tests for migration persistence, foreign-key enforcement, tenant isolation, idempotency, failed-insert rollback, concurrent retries and state transitions. The current queue accepts only `validation.noop`; no worker executes jobs.
- Added session tests for process-style database reopen, hash-only persistence, tenant membership enforcement, current role changes, disabled identity denial, expiry, CSRF mismatch, and individual/user-wide revocation.
- Follow-up review tightened session issuance/checks to deny suspended tenants, made schema startup fail closed on a future schema version, and blocked new queue rows for suspended tenants or disabled identities. Added regression coverage for those cases.
- Pinned the cgo-free `modernc.org/sqlite` driver at v1.38.2, the newest version checked that retains the repository's Go 1.23 compatibility baseline, to avoid a platform C compiler requirement. The upstream package is BSD-3-Clause and includes SQLite (public domain); transitive dependency licence inventory remains to be completed before release.
- The Go queue and SQLite session store are intentionally **not wired into the HTTP API or daemon**. Dev-token identity is not a production identity system, and schema/ADR-003/threat review remain unapproved. The existing JSON journal remains the running prototype's store, and the login path has no persistent session support yet.

### Auth/session direction remains proposed

Before any production login route, the design must specify a vetted Argon2id password verifier and parameter review, generic failure behavior and rate limits, MFA enrollment/recovery and replay handling, TLS-only `__Host-` cookies, server-side hashed session/CSRF token storage, expiry/rotation/revocation, Origin/CSRF checks, and tenant membership revalidation on authenticated requests and jobs. `SessionManager` remains memory-only; `SQLiteSessionStore` is an isolated persistence primitive, not a login flow. No Argon2id verifier, MFA flow, cookie issuance route, CSRF/Origin middleware or rate limiting has been implemented. ADR-003 and the threat model still require maintainer review.

### Validation on this branch

- Go 1.25.14 Windows/amd64: focused package tests for database/jobs, plus tests for API/auth/health/daemon/RBAC, passed; `go vet` for those packages passed; `go build ./cmd/tongd` passed.
- GitHub Actions on driver update commit `cc7608349e0f6f56ed3c85eec37d7869f0456b8c`: [Go checks #40423199616](https://github.com/parmtech/Tong-Panel/actions/runs/40423199616) passed, including gofmt, full `go test -race ./...`, and `go vet ./...`; [SQLite schema checks #40423199639](https://github.com/parmtech/Tong-Panel/actions/runs/40423199639) passed.
- GitHub Actions on session-store commit `6795148c39404f653f57d8936670f2d29595345f`: [Go checks #40423834545](https://github.com/parmtech/Tong-Panel/actions/runs/40423834545) passed, including gofmt, full `go test -race ./...`, and `go vet ./...`; [SQLite schema checks #40423834542](https://github.com/parmtech/Tong-Panel/actions/runs/40423834542) passed.
- GitHub Actions on tenant-suspension session fix `bb617eaed9dc2b6e8e10e1b8380d85987feb1408`: [Go checks #40424296948](https://github.com/parmtech/Tong-Panel/actions/runs/40424296948) and [SQLite schema checks #40424296939](https://github.com/parmtech/Tong-Panel/actions/runs/40424296939) passed.
- GitHub Actions on migration-version and queue-eligibility fixes `252c53127a9606835b9c43af298d5bb958519f6f`: [Go checks #40424600275](https://github.com/parmtech/Tong-Panel/actions/runs/40424600275), including full `go test -race ./...` and `go vet ./...`, and [SQLite schema checks #40424600243](https://github.com/parmtech/Tong-Panel/actions/runs/40424600243) passed.
- Local Go 1.25.14 Windows/amd64: `go test ./internal/database ./internal/jobs ./internal/auth` and `go vet ./internal/database ./internal/jobs ./internal/auth` passed after the latest follow-ups. The repository-wide Windows limitation remains as documented below.
- `go test ./...` could not build the existing Linux-only `internal/preflight` package on Windows (`syscall.Statfs_t`/`syscall.Statfs` unavailable).
- `go test -race ./...` could not run on the Windows toolchain because CGO is unavailable; the Linux Actions race run above passed.
- Python schema tests were not run locally because Python is unavailable; the updated schema workflow passed in GitHub Actions above.
- No API login wiring, Argon2id, MFA, lease/retry worker, audit trail for jobs, privileged operations, installer or VPS change was added.

