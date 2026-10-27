# P1 Internal Login Orchestration — Review Draft Only

**Status:** Experimental internal-only auth component. No browser login, public TLS listener or privileged host changes. Draft [PR #32](https://github.com/parmtech/Tong-Panel/pull/32), stacked on #31.

## Authentication order

1. The trusted caller supplies a **server-authorized tenant context** and an actual socket-trusted peer IP. Neither may be taken from arbitrary headers, query strings or user claims. There is **no HTTP handler** to invoke this module.
2. SQLite persistent `LoginLimiter.Admit` atomically records per-account and per-source attempts **before** any Argon2id work, with a fixed 15-minute window. Wrong password, nonexistent user, inactive membership and failed MFA consume attempts. The policy remains unapproved.
3. The store checks an active identity, tenant and membership. Missing/disabled identities undergo a precomputed dummy Argon2id verification using the same process-wide memory admission budget to reduce enumeration timing differences. The verifier returns the generic `ErrLoginDenied` for expected denials.
4. On a correct password, the service loads the **active encrypted TOTP seed**, decrypts it with the externally-provisioned identity-bound AES-GCM box, and validates the provided code against a **server-owned clock** with ±1 step tolerance.
5. Before writing, the service draws fresh independent 256-bit bearer+CSRF tokens and a random session/audit ID. SQLite `BEGIN IMMEDIATE` rechecks unchanged credential, enabled identity, active tenant, role membership and the same active MFA envelope. An identity-wide monotonic TOTP step is consumed, then hashed session+CSRF and human-attributed `auth.login` audit are written. All three actions **commit together**.
6. A failure in audit/session insert **rolls back OTP consumption**. Two concurrent logins using the same TOTP code cannot both mint a session. Stored session hashes can later be revoked with the existing SQLite session store; no plaintext token is persisted.

## Boundaries / important limitations

- The API and daemon do **not** initialize `InternalLoginService` or expose a login route. No public or browser deployment is authorized. A TLS handler must be separately threat-modeled, integrated and tested.
- The new internal method accepts a trusted peer IP argument, but **does not itself establish** trust in a socket, reverse proxy or tenant selection. This needs an authenticated transport and carefully configured proxy/trust boundary before use.
- MFA seed encryption/HMAC keys are injected in-process, with **no approved external key custody, rotation or recovery**. The constructor performs one Argon2id dummy-hash computation; it must run once per trusted process, never per request.
- Passwords and TOTP codes are supplied in memory, never included in audit entries, and must not be logged by future API handlers. Generic expected denial errors do not themselves guarantee indistinguishable response timing.
- This implements a short **30-minute absolute** session expiry only. The proposed 30-minute idle/12-hour absolute scheme, rotation, device management, logout endpoint and secure `__Host-` cookie issuance remain incomplete.
- No protected enrollment UI, recovery codes, MFA reset safeguards, successful-login counter policy, cross-process KDF capacity coordination or distributed failure telemetry. Failed login audits are not yet implemented. Account-targeted lockout denial-of-service policy is unapproved.
- ADR-009 and P0 architecture security signoff remain outstanding. Host agent privilege, signed Debian/Ubuntu package, SBOM, clean VM install+rollback and UI a11y/E2E gates are **not** satisfied by this change.

## Regression tests

`go test -race ./...` now checks valid login, session+CSRF lookup, OTP reuse denial, multi-goroutine single winner, wrong/unknown/disabled identities, persistent limiter exhaustion, and audit-failure rollback that preserves the OTP for a subsequent successful attempt.

**Success here proves only the internal prototype transactions.** No Phase 1 completion, user-facing publish readiness, production TLS login, or Phase 2 handoff is claimed.
