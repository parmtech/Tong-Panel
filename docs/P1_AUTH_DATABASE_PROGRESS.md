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
- Added `internal/database.Open`: creates a private parent directory, configures `foreign_keys`, a bounded busy timeout, WAL and FULL synchronous mode through per-connection driver pragmas, applies the schema, and limits the Go connection pool to one connection for this single-process prototype.
- Added an opt-in `jobs.SQLiteStore` prototype. Job creation is a SQL transaction tied to an existing tenant membership; idempotent retries return the same job, cross-tenant reads are scoped, and transitions are validated and committed with a state predicate.
- Added tests for migration persistence, foreign-key enforcement, tenant isolation, idempotency, failed-insert rollback, concurrent retries and state transitions. The current queue accepts only `validation.noop`; no worker executes jobs.
- Pinned the cgo-free `modernc.org/sqlite` driver at v1.34.5 to avoid a platform C compiler requirement. The upstream package is BSD-3-Clause and includes SQLite (public domain); transitive dependency licence inventory remains to be completed before release.
- The Go queue is intentionally **not wired into the HTTP API or daemon**. Dev-token identity is not a production identity system, and schema/ADR-003/threat review remain unapproved. The existing JSON journal remains the running prototype's store.

### Auth/session direction remains proposed

Before any production login route, the design must specify a vetted Argon2id password verifier and parameter review, generic failure behavior and rate limits, MFA enrollment/recovery and replay handling, TLS-only `__Host-` cookies, server-side hashed session/CSRF token storage, expiry/rotation/revocation, Origin/CSRF checks, and tenant membership revalidation on authenticated requests and jobs. The current `Sessions` manager is memory-only; none of these login flows or controls has been implemented. ADR-003 and the threat model still require maintainer review.

### Validation on this branch

- Go 1.25.14 Windows/amd64: focused package tests for database/jobs, plus tests for API/auth/health/daemon/RBAC, passed; `go vet` for those packages passed; `go build ./cmd/tongd` passed.
- `go test ./...` could not build the existing Linux-only `internal/preflight` package on Windows (`syscall.Statfs_t`/`syscall.Statfs` unavailable).
- `go test -race ./...` could not run on the Windows toolchain because CGO is unavailable. Linux CI must provide the full race result.
- Python schema tests were not run locally because Python is unavailable. The updated schema workflow is expected to run in GitHub Actions after push; no result is claimed here.
- No API login, persistent sessions, lease/retry worker, audit trail for jobs, privileged operations, installer or VPS change was added.
