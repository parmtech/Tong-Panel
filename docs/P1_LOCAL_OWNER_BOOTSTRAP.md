# P1 local first-owner bootstrap — **development-only / no remote login**

This document applies to the **draft** [PR #30](https://github.com/parmtech/Tong-Panel/pull/30), stacked on [PR #29](https://github.com/parmtech/Tong-Panel/pull/29). It does not authorize production installation, exposure of the API, or Phase 2 hosting. All P0/ADR security sign-offs remain pending.

## Implemented contract

- `auth.BootstrapOwner(ctx, db, username, tenantName, password)` only succeeds against a **completely empty, v5** SQLite metadata store. It rejects an existing tenant, user, membership, job, session, audit, MFA enrollment or replay record. It **never** resets existing owners or repairs an imported database.
- First Owner, Tenant, membership and human-attributed `owner.bootstrap` audit record are created within a single `BEGIN IMMEDIATE` transaction. If the audit write fails, all rows roll back. Concurrent bootstraps on separate connections cannot both succeed.
- The password is hashed with the existing bounded **Argon2id PHC** primitive; password and one-time token values are never written to stdout, the audit table, CLI flags or an environment variable. There is **no default username or password**.
- `tongctl bootstrap-owner` must be invoked locally with `TONG_ALLOW_LOCAL_OWNER_BOOTSTRAP=yes`, an explicit `--password-stdin` flag, and an anonymous **pipe**, not a terminal or regular file. A database path that is a symlink, group/world accessible, or not owned by the current operator is denied. New file is created `0600`; containing directory must be owner-only.
- Existing read-only `tongctl doctor` remains available; this new CLI operation is not an admin HTTP endpoint, Unix agent command or background worker.

## Reproduce in a **disposable**, locally controlled development lab only

From a trusted shell running as the account owning the prospective private database directory:

```bash
go build -o ./tongctl-dev ./cmd/tongctl
mkdir -p "$HOME/.tong-panel-dev"
chmod 700 "$HOME/.tong-panel-dev"
read -r -s -p 'Choose first owner password (12-1024 bytes): ' bootstrap_password
printf '\n'
printf '%s\n' "$bootstrap_password" |
  TONG_ALLOW_LOCAL_OWNER_BOOTSTRAP=yes ./tongctl-dev bootstrap-owner \
    --db "$HOME/.tong-panel-dev/panel.db" \
    --username firstowner --tenant "My development tenant" --password-stdin
unset bootstrap_password
```

The shell `read -s` command prevents terminal echo; the password is a **shell variable temporarily**, not an environment variable passed to the daemon or an argv argument. Never enable shell tracing (`set -x`), capture these commands into CI logs or put real passwords in scripts. You must protect the terminal itself and surrounding OS; the development example is not a substitute for audited password entry and operator key provisioning. The command refuses terminal `stdin` so it never unknowingly echoes a secret.

Re-running the command on the same database must fail. It does not bypass an existing panel, generate a recovery password, enroll MFA or issue a session.

## Automated tests and evidence

- Go `internal/auth/bootstrap_owner_test.go`: Argon2id, first owner role + audit, denial after import, two-handle concurrency and audit-failure rollback.
- Go `cmd/tongctl/bootstrap_test.go`: explicit operator opt-in, pipe-only secret, no argv password, secured database path and symlink rejection.
- `scripts/smoke_bootstrap_owner.py` invokes the **real compiled binary** using ephemeral SQLite, verifies permissions and audit, and checks repeat/disabled bootstrap fail-closed.
- `.github/workflows/p1-smoke.yml` and `p1-preview-package.yml` run the CLI dry-run on a compiled and archive-extracted binary alongside existing daemon/API/SQLite smoke.

## Still blocking actual Phase 1 exit

Bootstrap is **not** a secure login or MFA session lifecycle. No reviewed TLS origin, password-verification login endpoint, per-user/source durable throttling, encrypted TOTP master key provisioning, recovery codes, transactional MFA/session issuance, full device session rotation/logout or production browser shell is available. Privileged root-agent security, signed release/SBOM, clean Ubuntu/Debian VM install+upgrade+rollback and accessibility/end-to-end validation are also outstanding.

A live hosting installation and a Phase 2 release must remain blocked until these gates receive test evidence and human approval. See `docs/P1_EXIT_GATE_REPORT_2026-10-27.md`.
