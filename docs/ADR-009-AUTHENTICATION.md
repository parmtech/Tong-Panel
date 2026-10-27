# ADR-009 — Authentication, sessions and recovery

**Status:** PROPOSED — not approved; do not use as production authorization.
**Date:** 2026-10-27
**Scope:** single-server P1 authentication foundation only. No login route or production credential store is approved by this draft.

## Context

The current API uses development bearer tokens and a JSON job journal. The repository has isolated in-memory and SQLite session primitives, a TOTP verification primitive, and a new Argon2id PHC verifier prototype. It has no account login route, MFA enrollment/recovery, replay protection, login throttling, CSRF/Origin middleware or production session wiring. ADR-003 and the threat model remain unapproved. Issue #6 requires a reviewed design before authentication integration.

Authentication KDF work can consume substantial CPU and memory. A corrupted stored verifier or unbounded login concurrency could exhaust a small VPS. User and tenant existence, account state, password validity and MFA state must not be disclosed by response differences.

## Proposed decision

### Password credentials

- Store only a canonical Argon2id v=19 PHC string. The current prototype uses a 16-byte random salt, 32-byte output and the RFC 9106 lower-memory recommendation: 64 MiB, 3 passes and 4 lanes. This is a proposed default, not yet benchmarked against supported VPS targets.
- New passwords must be 12–1024 bytes. Treat password input as bytes exactly as received; do not trim, silently truncate, normalize or log it. The prototype rejects shorter passwords for both creation and verification.
- Before calling the KDF, reject unsupported algorithms/versions, malformed encodings, salts outside 16–32 bytes, non-32-byte outputs, memory below 64 MiB or above 128 MiB, iterations below 3 or above 6, and parallelism outside 1–4. Keep the upper limits configurable only through a reviewed migration, not user input.
- Keep hash creation and verification outside SQLite write transactions. After benchmarking, enforce both per-account/source rate limits and a process-wide memory-budgeted concurrency bound. When the bound is full, fail quickly with a generic retry response; do not queue unbounded requests. Trust forwarded client IP only from explicitly configured reverse proxies.
- For nonexistent/disabled users, perform a bounded dummy verification at the same current profile where practical. Return the same status and response shape for unknown user, wrong password, disabled identity, wrong MFA and tenant denial. Do not expose which condition failed in logs or responses.
- Rehash after successful authentication when the stored profile differs from the approved current profile. Commit the replacement only after all required MFA checks pass; use a conditional identity update so concurrent password changes cannot be overwritten.

### MFA and recovery

- Require MFA for the initial owner/admin role before public remote access. Evaluate passkeys as a later option; the first implementation may use TOTP only after secret storage and replay behavior are reviewed.
- Generate at least 160 bits of TOTP secret with a cryptographic RNG. Encrypt it at rest with a key stored outside the SQLite database, redact provisioning URIs and secret values, and reveal the enrollment QR/secret only once over an authenticated TLS session.
- Enrollment remains pending until the user proves a current TOTP. On every accepted code, atomically record the accepted time-step and reject that step or any older step on later requests. Keep the existing ±1 time-step window only if every accepted step is replay-protected transactionally.
- Generate one-time recovery codes with at least 128 random bits; store only a keyed or slow hash, consume each code in the same transaction as session issuance, and display each code once. MFA reset requires a short-lived, single-use recovery/bootstrap flow, audit event and revocation of all prior sessions.

### Browser sessions and request checks

- Use an independent random 256-bit bearer token and CSRF token. Store only SHA-256 hashes in SQLite. Issue session rows only after password, MFA, enabled identity, active tenant and membership checks succeed; insert the session and audit event transactionally.
- Set the session only as a TLS-only, host-only `__Host-` cookie with `Secure`, `HttpOnly`, `Path=/`, no `Domain`, and `SameSite=Strict` unless an approved UX flow requires a narrower alternative. Never downgrade cookie security for local HTTP; local development must use a separate explicit mode.
- Proposed lifetime: 30-minute idle timeout and 12-hour absolute lifetime, with reauthentication for password/MFA changes and sensitive account operations. The schema and session store need idle-expiry, last-seen throttling and rotation support before adopting this policy. Rotate after login, MFA completion, privilege changes and recovery; revoke on logout, password change, disablement, membership removal and tenant suspension.
- For cookie-authenticated state changes, require both the independent CSRF token in a header and an exact match against the configured origin. Reject missing/untrusted Origin on browser mutations; do not enable credentialed cross-origin access by default. TLS termination and trusted proxy headers must be explicitly configured.
- Resolve the current identity, tenant, membership and role from the server-side session on every request and again before each job transition/apply. Never trust a tenant ID supplied by the browser as authorization. Apply object ownership checks in addition to role checks.

### Bootstrap, audit and operational limits

- Do not ship default credentials. Create the first owner through an explicit local, single-use bootstrap flow with a generated secret, expiry, audit record and mandatory password/MFA setup. Recovery and bootstrap material must never be written to ordinary logs.
- Audit successful and denied authentication, MFA enrollment/recovery, password changes, session issue/revoke and role/membership changes without recording passwords, cookies, CSRF values, TOTP secrets, recovery codes or raw verifier contents.
- Keep the login service behind HTTPS and a configured public origin. Rate limiting, process memory budget, proxy trust, bootstrap handling and cookie origin are mandatory configuration gates before exposing any login endpoint.

## Alternatives considered

- **Argon2id PHC versus custom format:** use standard PHC text to retain parameter/version metadata and support future rehash. Do not invent a binary password format.
- **Stateless tokens versus SQLite sessions:** use server-side revocable sessions for immediate disablement, logout and membership changes. Stateless bearer/JWT sessions are not proposed for the single-node MVP.
- **TOTP replay window without state versus transactionally consumed steps:** require persistent step consumption; a code accepted once must not be accepted again during its validity window.
- **Rate limit by IP only:** insufficient for distributed credential stuffing; combine account and source controls while avoiding account-existence disclosure and untrusted proxy headers.

## Risks and unresolved review questions

- Validate the 64 MiB / 3 / 4 profile and maximum login concurrency on the smallest supported VM. The developer workstation benchmark is not representative evidence.
- Decide whether minimum password length is measured in bytes or Unicode code points and document international input behavior before a user-facing registration flow.
- Choose the at-rest encryption/key rotation design for TOTP secrets and recovery-code pepper; define backup/restore behavior without putting keys in SQLite backups.
- Define proxy/origin configuration and a safe bootstrap/recovery procedure when the panel is unreachable.
- Decide whether the rate limiter needs persistence across restart and how account lockout avoids denial-of-service against a named victim.
- Complete dependency advisory and transitive licence review for `golang.org/x/crypto` before release.
- The current Go 1.23-compatible `x/crypto v0.41.0` pin has known advisories in its SSH/agent/OpenPGP packages. The P1 package-set `govulncheck` scan found no reachable vulnerable symbols, but reported those module-level findings plus a Windows-only `x/sys/windows` package finding. Avoid importing the affected packages; review whether to raise the Go toolchain baseline and update the pin before adding SSH functionality or releasing.

## Acceptance before login integration

1. Owner/security reviewers record Accepted, Rejected or Needs revision here and in issue #1/#6.
2. Password profile and KDF concurrency budget are benchmarked on the declared smallest supported VM.
3. SQLite migration and tests cover MFA secret/key references, TOTP step replay, recovery-code consumption, session idle/absolute expiry, rotation and revocation across restart.
4. Tests prove generic authentication failures, rate-limit behavior, CSRF/Origin denial, and a role/tenant/object authorization matrix for requests and jobs.
5. A security review covers bootstrap, recovery, proxy trust, backups, secret redaction and dependency provenance.

## Current implementation status

The standalone Argon2id primitive and isolated SQLite session store are development prototypes only. No login service, HTTP route, MFA enrollment/recovery, CSRF/Origin middleware, rate limiter, browser cookie issuance or production daemon wiring exists. P1 remains in progress; no live server was changed.
