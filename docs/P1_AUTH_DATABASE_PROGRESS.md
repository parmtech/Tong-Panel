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
