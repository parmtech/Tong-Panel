# P1 Session Lifecycle Prototype — SQLite v7

**Scope:** [Draft PR #33](https://github.com/parmtech/Tong-Panel/pull/33), stacked on internal-only atomic login #32. All code remains experimental, unmerged, and non-public.

## Session semantics

- SQLite v7 adds `idle_deadline_unix`. Existing pre-v7 sessions are **revoked during migration** because their idle activity history cannot be proved. Users of a future supported release will need to reauthenticate after that upgrade.
- Sessions have an **absolute expiry** stored in `expires_at`, and a sliding **30-minute idle window**, never later than the absolute expiry. Both must be valid. Each valid lookup refreshes the server-owned idle deadline monotonically and rechecks current enabled identity, active tenant, role and revocation at the SQL update boundary.
- A session observed past its idle or absolute deadline is explicitly revoked, preventing reactivation after clock rollback. Concurrent valid reads are allowed and cannot shorten each other's expiry. Database writes are currently made on each authenticated read; this should be performance-tested before public deployment.
- `Rotate(ctx, oldToken)` uses a single SQLite writer transaction to revoke the previous token and its CSRF secret, issue new independent 256-bit random bearer+CSRF hashes, preserve the original absolute expiry and write an attributed `auth.session_rotated` audit. Repeating/parallel rotation of the same bearer must not issue another valid session.
- `Logout(ctx, token)` revokes a current token and writes `auth.logout` in the same transaction. If audit insertion fails, the revocation rolls back. The caller is responsible for sending `auth.ExpiredSessionCookie()` in a separately reviewed HTTPS handler.
- No token or CSRF plaintext is written to the database, audit, or normal log output.

## Expected test evidence

GitHub Go tests cover active/idle/absolute time boundaries; 30-minute sliding refresh; concurrent requests; clock rollback; rotation preserving absolute lifetime; simultaneous rotation single-winner; revoked predecessor; and logout audit rollback. Existing daemon, SQLite schema and extracted developer-only unsigned archive CI are also required to pass.

## Security limitations

- This is **not a public session lifecycle**. There is no browser login, logout or rotate endpoint in `tongd`, no trusted external TLS origin, no CSRF-protected public cookie set/clear flow, no secure device/session dashboard, no idle throttling benchmarks or user recovery.
- KDF encryption/HMAC key custody and MFA recovery remain unapproved. Proposed ADR-009 and the P0 threat model require formal owner/security review.
- Critical P1 gates still include privilege-boundary audit, host fencing and crash recovery, signed installer/SBOM/provenance, clean Ubuntu/Debian VM install/upgrade/rollback, dependency security scanning and accessible browser E2E. **Phase 1 is not complete, Phase 2 entry blocked.**

To reproduce the narrow behavior on a disposable checked-out branch: `go test -race ./...`, `go vet ./...`, `python3 -m unittest discover -s db -p 'test_*.py' -v` and `python3 scripts/smoke_p1.py /path/to/tongd`. These are not independent penetration tests or verified operating-system installers.
