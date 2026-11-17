# Internal one-time MFA recovery codes

Status: development prototype; ADR-009 remains proposed. No HTTP recovery,
password reset, MFA reset, authenticated session or production service is added.

`auth.SQLiteRecoveryStore` accepts an independent external 32-byte pepper and
stores only HMAC-SHA256 digests of eight independently random 256-bit codes.
Digests include a versioned domain and tenant/account scope. Plaintext codes
are returned only by issuance and must be displayed once without logging.

The trusted caller must prove recent password and active MFA before issuance
or replacement. Consumption requires the caller to have verified the password
and source admission separately. Code consumption alone grants no session or
MFA-reset authority. The API deliberately returns neither a principal nor a
session. Full recovery orchestration, session issuance, re-enrollment, HTTPS
delivery and owner-approved recovery policy remain future reviewed work.

Migration 12 binds each set to the current active MFA enrollment envelope.
Enabled identity, active tenant and current membership are rechecked inside
the SQLite writer transaction. Changing the enrollment invalidates old sets.
Replacement deletes old codes atomically; consumption deletes exactly one
digest, revokes all sessions for that identity across tenants, and records an
audit event in one transaction. Audit failure rolls everything back.

Five attempts per tenant/account set per 15-minute window are persisted across
restart; malformed codes also count. Clock rollback fails closed. This is an
account admission bound, not a complete source/abuse policy. Trusted issuance
is separately protected by its caller's recent authentication and must not be
exposed as a way to reset the attempt limit. Denied attempts use a system audit
actor rather than claiming that the targeted identity performed them.

Pepper provisioning, custody, rotation and recovery are deliberately external.
Losing or replacing it requires trusted reissuance. Do not include it in SQLite
backups. There is no key ring or automatic pepper migration in this primitive.

Tests cover reopen persistence, replay, independent-handle concurrent one-winner
consumption, identity-wide session revocation, account/tenant/MFA state changes,
membership removal, wrong pepper, replacement, durable attempt bounds, clock
rollback and transaction rollback when auditing fails. Local focused Go tests
and vet are recorded in UPDATES.md; Linux race/CI evidence is still required.

The SQLite opener also bounds retries for transient SQLITE_BUSY during the
driver's initial WAL pragma. Concurrent fresh database opens reproduced that
startup failure; the existing concurrent-migration suite validates the fix.
Only typed SQLITE_BUSY is retried, within a caller-cancellable five-second
startup window. Migration failures and other errors continue to fail closed.
