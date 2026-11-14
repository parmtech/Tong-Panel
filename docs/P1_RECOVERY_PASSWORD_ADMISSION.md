# Internal password admission for MFA recovery

Status: development-only trusted-caller primitive. It is not a public recovery
flow and does not issue a browser session, reset MFA, or replace enrollment.

`auth.InternalRecoveryService.RecoverTrusted` combines the existing durable
account/source limiter, bounded Argon2 verification, and one-time recovery
store. The limiter runs before credential lookup and KDF. Missing, disabled,
suspended, invalid-role, or malformed-hash accounts use the service's dummy
PHC for bounded verification. Incorrect credentials do not inspect or consume
recovery codes and do not revoke sessions.

For a valid password, the trusted service captures the exact password PHC,
membership role, and active MFA enrollment envelope before running Argon2.
After the KDF, one SQLite `BEGIN IMMEDIATE` transaction verifies that all three
values and the active tenant/membership/identity remain unchanged. In that same
transaction, it charges the recovery-code attempt budget, consumes the code,
revokes every existing session for the identity, and writes the audit event.
Wrong-code denial attempts and their audit commit durably. Audit or storage
failure rolls back code consumption and session revocation. Argon2 never runs
while holding the SQLite writer lock.

`SQLiteRecoveryStore.ConsumeTrusted` remains available for existing internal
callers and preserves its standalone semantics. The transaction-aware helper
lets the password coordinator compose the recheck and recovery mutation without
nested transactions.

The source address and tenant are trusted inputs and must come from server-side
context. Do not pass client-controlled forwarding headers or expose this method
through a developer-token API. Public integration still requires approved
proxy/source policy, pepper custody, generic transport behavior, and the full
recovery/reset ceremony in ADR-009. This slice creates no authentication
session and grants no MFA-reset authority.
