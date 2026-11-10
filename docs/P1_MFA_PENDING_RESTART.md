# Expired pending MFA enrollment restart

The trusted-only `MFAEnrollmentStore.BeginTrusted` can replace an expired pending
enrollment with a newly generated encrypted TOTP seed and ten-minute deadline.
An active enrollment or pending enrollment before its deadline remains unchanged.
At the exact stored deadline the old pending enrollment is expired. A missing,
disabled or unauthorized identity or inactive tenant cannot start or restart.

This is one conditional SQLite upsert, preserving a single winner when callers
race. It never returns the prior seed and never replaces an active factor. The
caller must already hold approved current password/recent-authentication or
bootstrap proof; this method itself does not establish that authority.

Confirmation binds both replay consumption and activation to the encrypted
envelope and expiry that were read before code verification. A proof checked
against an older pending seed cannot activate a replacement enrollment or consume
the new enrollment's replay step. State and membership checks still run inside
the transaction, and activation failure rolls back replay consumption.

Tests cover deadline boundaries, persistence, active/live protection, concurrent
restart, authorization changes, stale-snapshot interleaving and rollback. The
stale-snapshot test uses the existing AEAD interface in test code to replace a
pending enrollment between decryption and confirmation; no production test hook
or sleep-based timing race is added.

No HTTP route, account reset/recovery authority, master-key custody or rotation
procedure, default key, session issuance or production approval is added. Active
factor replacement requires a separately reviewed recovery/re-enrollment design.
See `UPDATES.md` for actual execution evidence and remaining P1 gates.
