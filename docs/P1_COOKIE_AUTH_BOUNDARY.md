# P1 — HTTPS Cookie Session Boundary (Developer Prototype)

**Draft:** [PR #24](https://github.com/parmtech/Tong-Panel/pull/24), stacked on [PR #23](https://github.com/parmtech/Tong-Panel/pull/23). No production login or host operations.

## What actually works in code

- The Go API `Server` can be configured with an existing `auth.SQLiteSessionStore`, and accepts a **pre-issued** `__Host-tong_session` cookie for protected routes.
- The cookie path requires the server's actual HTTPS connection (`r.TLS`). It **never** trusts `X-Forwarded-Proto` or other unsigned proxy headers.
- Mutating requests with a cookie (POST, DELETE, and any future non-safe HTTP method) must contain an exact HTTPS Origin matching the current request Host **and** the right per-session `X-CSRF-Token`. Missing, forged, cross-site or malformed Origin/CSRF inputs fail with 403; no CORS exceptions are made.
- SQLite session check revalidates current tenant, identity, membership, expiry and revocation. A role downgrade takes effect on the next request.
- A cookie cannot be combined with a developer bearer key; duplicate session cookies and bad/expired cookies fail closed. A failed cookie does not fall back to developer bearer authentication.
- Tenant RBAC now denies all cross-tenant job actions **including the tenant Owner role**. Global platform/root administration is a separate security subject requiring future design.

## Important boundaries

- **No login endpoint**, password credential exchange, enrollment, browser cookie issuance, refresh/logout handler, server TLS listener, trusted reverse proxy configuration or complete authentication lifecycle is added in this PR.
- The existing `tongd` process still binds `127.0.0.1:8090` using HTTP and dev-key auth. It does **not** configure the cookie session store and therefore cannot be used as a real browser login.
- Unit/integration tests inject a TLS request and an already-issued short-lived SQLite session, without creating login credentials or contacting a VPS.
- This is a useful foundation for future reviewed authentication, **not proof of production security**. Secure cookies, TLS termination, password+MFA flow, rate limits, session rotation, recovery, CSP, audit, worker identity, signing and server isolation require their own implementation and security reviews.
- Do not run on a public interface, or merge parent stacked drafts as a production release merely because unit tests pass.

## Proof required

GitHub Go CI must run gofmt, `go test -race ./...`, `go vet ./...` on this exact branch. Tests cover current role downgrade, cross-tenant Owner denial, TLS-spoofing via `X-Forwarded-Proto`, CSRF mismatches, malicious Origins, stale/revoked cookies and mixed authentication.

**Next integration review:** agree on ADR-009 auth design and permanent secure web origin/TLS topology; then integrate Argon2id verification, rate limiting, replay-proof MFA, session issuance/revocation and password recovery behind a reviewed login handler.
