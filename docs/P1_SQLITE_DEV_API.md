# Tong Panel P1 — Optional SQLite Developer API (PR #23)

**Status:** Draft development prototype; not a hosting release, not production authentication and not an installer.

## What is now wired
- The daemon still binds `127.0.0.1:8090` only and normally uses the local JSON journal for its **single-developer preview**.
- Set **both** `TONG_DEV_TOKEN` (random, at least 32 characters) and `TONG_DEV_SQLITE_DB` (a local file path) to explicitly select the SQLite job backend. SQLite initialization errors are fatal, not silently downgraded to JSON.
- SQLite mode validates the caller's token-bound tenant/identity **against already-present active tenant, identity and membership rows on every protected HTTP request**. The live database membership role replaces any outdated role recorded in the developer token. It never creates a default owner, user, password or membership.
- `GET /api/v1/jobs` and `GET /api/v1/jobs/{id}` return **only the current principal's own jobs**, never all jobs in a shared tenant. Invalid/deactivated principals see no SQL job details.
- `POST /api/v1/jobs` creates only `validation.noop` under the identity from the authenticated developer token, using transactional idempotency.
- `DELETE /api/v1/jobs/{id}` in SQLite mode only cancels the authenticated requester's **queued** job with an atomic audit; it refuses leased, running or completed work (409). Legacy JSON journal has no requester identity and explicitly returns 501 for deletion.
- No scheduled execution, operating-system mutation, password login, cookie session, MFA or root agent commands are enabled.

## Manual disposable local development setup ONLY

Use a disposable developer environment, never an Internet-facing server. In a checked-out branch with Go available:

```bash
mkdir -p .tong-dev-data && chmod 700 .tong-dev-data
export TONG_DEV_TOKEN="$(openssl rand -hex 32)"
export TONG_DEV_SQLITE_DB="$PWD/.tong-dev-data/panel.db"
go run ./cmd/tongd
```

The Go startup creates and migrates the SQLite file. It does **not** seed any identity or password. In a separate terminal, after the database has been initialized, explicitly create the development-only mapping with an installed `sqlite3` CLI (replace the path):

```bash
sqlite3 "$PWD/.tong-dev-data/panel.db" <<'SQL'
INSERT INTO tenants(id,name) VALUES('dev-tenant','Local development');
INSERT INTO identities(id,username) VALUES('local-dev','localdev');
INSERT INTO memberships(tenant_id,identity_id,role)
VALUES('dev-tenant','local-dev','owner');
SQL
```

Only then will dev-token queue creation succeed. Missing membership, disabled identity or suspended tenant fails closed. Do **not** place this SQLite file in an untrusted/shared directory. These identities are not production users.

## Integration tests / required evidence
`internal/api/sqlite_test.go` tests token denial, own-job reads, cross-user denial even within the same tenant, cross-tenant denial, duplicate idempotency protection, active membership checks, user/tenant revocation and audited cancellation, including refusal to cancel an already leased job. Check PR #23's GitHub Go Actions for definitive pass/fail status.

## Serious limitations
- `TONG_DEV_TOKEN` is a **static developer bearer key**, not a secure real-user session. No password challenge, rotation, TOTP login, origin-based CSRF protection, reverse proxy/TLS deployment or production threat review. Never expose the preview port to an untrusted network.
- No database-backed identity bootstrap/API workflow, sessions integrated into HTTP or approved operator roles. SQL creation is a manual developer fixture, not an installer procedure.
- JSON preview mode does not have per-requester ownership, and should not be used with multiple accounts.
- The database store remains a **no-op** job prototype with no actual worker. There is no host-side monotonic fencing or privileged agent authorization; enabling real Linux work is prohibited until those security gates are passed.
- P1 and P2–P6 acceptance gates, signed installer and disposable-VM distro evidence remain incomplete. Parent stacked PRs #14–#22 need code/security review first.
