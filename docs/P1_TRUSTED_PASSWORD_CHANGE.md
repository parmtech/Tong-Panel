# Internal trusted password change

`InternalLoginService.ChangePasswordTrusted` changes the password for an identity
that proves its current password and active encrypted TOTP factor. Tenant,
username and socket-derived peer IP must come from a reviewed trusted caller.
This method is not an administrative reset, public endpoint, developer-token
operation or authorization derived from client-controlled forwarding headers.

The sequence is:

1. Apply the existing durable account/source attempt gate and validate the new
   password's 12–1024-byte bounds. Password bytes are used unchanged.
2. Verify the current credential, using bounded dummy work for unavailable or
   malformed accounts. Check active MFA and generate the new salted verifier
   outside the SQLite writer transaction. Hashing and verification share the
   same non-queueing process-wide KDF slots. Check cancellation between KDF work;
   an Argon2 operation already running is not interruptible by this interface.
3. Recheck the code with server time after hashing. In one `BEGIN IMMEDIATE`
   transaction, recheck exact old PHC/enrollment/role and current enabled identity,
   active tenant and membership. Consume the identity-wide replay step, replace
   the exact old verifier, revoke all SQLite sessions for that identity and write
   a user-attributed `auth.password_changed` audit event.
4. Commit all effects together. A stale proof, replay, cancellation or audit
   failure must not commit a partial password/replay/session change.

Passwords are global to an identity, so revocation covers that identity's session
rows across tenants. Other identities' sessions are unchanged. Active MFA and
recovery material are preserved; no replacement session or reset token is issued.
Separately configured development bearer keys are not managed by this method.

Credential/capacity denials use a generic result. Operational failures must not
return raw SQL errors or private material. The attempt limiter may record a denied
request even when credential, replay and session state remain unchanged. Tests
exercise persistence, cross-tenant revocation, stale-proof races, overload,
cancellation and transaction rollback using synthetic fixtures.

This follows the internal reauthentication/revocation direction in draft ADR-009.
It does not approve that ADR or provide browser transport, CSRF/origin handling,
public password-change UI, account recovery, key custody, account administration
or supported-host KDF benchmarks. No new HTTP route or schema is added.

See `UPDATES.md` for actual code-head test results and remaining P1 gates.
