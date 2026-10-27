# P1 Durable Login Admission — Prototype (Not a Login Endpoint)

This is a **draft internal security primitive**, stacked on local owner bootstrap in [PR #30](https://github.com/parmtech/Tong-Panel/pull/30). Public login and Phase 1 completion remain **BLOCKED**.

## Guarantees inside this module

- `auth.SQLiteLoginLimiter.Admit(ctx,tenant,username,trustedPeerIP)` consumes one allowance in **both** account and source buckets, atomically under SQLite `BEGIN IMMEDIATE`, before any expensive password hash can be attempted.
- Default fixed policy: **5 attempts per tenant/account and 25 attempts per actual peer IP over a 15-minute window**. This is a proposed development policy, not approved to deploy to a customer host.
- Identifiers are HMAC-SHA256 keyed with an **independent, externally managed 32-byte high-entropy key**. Only fixed 32-byte digests, counters and window times are persisted in SQLite; not raw account names, IPs, passwords or session tokens.
- An existing SQLite database and stable limiter secret preserve limits across separate Go processes and restarts. SQLite transaction fencing denies parallel account/source bypass; source must parse as a real IP address. The eventual TLS/HTTP layer must derive it from a trusted socket/proxy, not arbitrary `X-Forwarded-For`.
- Clock regression fails closed. Old counters are indexed/pruned, with a global number-of-buckets cap that denies new admissions under overload rather than unbounded storage.

## Operational/security limitations

This component is **not wired to any HTTP login route**. Account-existence ambiguity is therefore currently preserved by not looking up accounts in the limiter; actual login handling still requires generic failure and dummy verification. Key custody, restart-stable key provisioning and safe limiter-key rotation are not implemented. Replacing the external HMAC key would reset access to old counters and must be prevented by reviewed deployment configuration. A hard per-account cap can be abused to lock out a named victim; policy and security review must resolve this tradeoff, possibly with progressively increasing delays and recovery.

Every admitted **attempt**, regardless of login success, consumes its two buckets. No untrusted user should call the in-memory service directly with a forged source. A real login must also enforce TLS-only origin, validated/sanitized peer IP, global memory-bound KDF, active tenant/member checks, encrypted MFA, replay prevention and **atomic session issue+audit**. No production credential pathway exists yet.

CI tests verify concurrent cross-DB enforcement, account and source quotas, clock rollback, expiry, malformed sources and SQLite schema guards. See [P1 Exit Gate Report](docs/P1_EXIT_GATE_REPORT_2026-10-27.md) for remaining release blockers.
