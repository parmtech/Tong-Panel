# P1: SQLite-backed TOTP replay barrier — experimental (draft)

This implementation adds **only replay consumption** to the existing standalone RFC 6238 TOTP primitive. It does not implement end-to-end MFA enrollment, secret encryption, account login, password recovery or cookie issuance.

## What is implemented

- Migration `0004_totp_replay_state.sql` adds a single `last_accepted_step` per **identity**, with foreign key to `identities` and nonnegative step check.
- `auth.NewTOTPReplayGuard(db)` exposes `VerifyAndConsume(ctx, tenantID, identityID, secret, sixDigitCode, at)`. It validates the code in a ±1 step window and chooses the highest matching step to avoid code collisions being replayed.
- One SQLite conditional UPSERT checks the identity is enabled, the tenant active, and that identity is an active member of that tenant, then advances the monotonic step **only if larger than the prior accepted step**. A replay or missing membership returns false. Two simultaneous consumers cannot both succeed.
- The source uses trusted server time **provided by the caller**, and intentionally neither receives nor stores secrets from HTTP. The caller must securely load a previously enrolled and encrypted TOTP secret. Insecure secret storage is not added here.
- Identity-level, not tenant-level, fencing ensures a code used in one tenant cannot be reused in another membership.
- Tests cover concurrent calls through separate SQLite connections, repeat rejection while valid within skew window, sequential step advancement, account suspension/disablement, missing membership and malformed codes; schema tests protect FK/step constraints.

## Safety and remaining work

- **No MFA login/registration endpoint**, TOTP enrollment proof, encrypted/rotatable secret store, rate limiting, recovery-code consume, session issue transaction or public TLS configuration. The existing `VerifyTOTP` without replay memory must not be used as the sole security check in a new login handler.
- Do not accept client-supplied clocks or untrusted identity/tenant claims as authority. A true login flow needs authenticated account context and server time, with a complete security review and generic failure behavior.
- Consuming a code before session issuance can burn the one-time code if session issuance fails. A production login design should atomically coordinate MFA consumption, session issuance, audit and recovery transaction semantics.
- Time-step and credential-verification policy is **proposed** in ADR-009, not accepted. Owner/security signoff still required.
- Adds SQLite schema version **4**, requiring a migration backup/restore plan and compatibility testing on disposable OS VMs; it is not appropriate for production customer data.
- All code remains a stacked draft, no host privileges, installer, public login, production DB write or VPS change. Phase 1 and Phase 2 gate remain incomplete.

## Tests

- `go test -race ./...` and `go vet ./...`
- `python3 -m unittest discover -s db -p 'test_*.py' -v`
- Actual daemon dry-run and unsigned archive tests (smoke expects SQLite `PRAGMA user_version=4`)
- Cross-process SQLite contention is tested with two distinct connections, not with real root workers.

See parent draft #27 and P1 exit gate report for broader release blockers.
