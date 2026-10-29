# Experimental encrypted backup and restore lab

`tongbackup` is a Linux-only operator utility for exercising the experimental
backup envelope. It is not connected to Tong Panel's API, database lifecycle,
job queue, scheduler, or host agent. It does not make the project production
ready.

## Key and file handling

- Supply a raw 32-byte wrapping key from an external key source. The CLI does
  not create, store, recover, or rotate keys for you.
- The key file must be a regular Linux file with no group or other permission
  bits (normally mode `0600` or `0400`). Do not place it beside a copied or
  offsite backup artifact.
- Choose an output directory that tenants and public web roots cannot access.
  Temporary output is mode `0600`; a complete result is published atomically
  and existing paths are never replaced.
- `encrypt` and `decrypt` read the data stream from standard input. Decrypt
  writes to a private staging file and publishes it only after the full backup
  authenticates. Failed operations may not be imported or published.

## Commands

Encrypt a SQL dump without writing plaintext to a disk file:

```sh
mariadb-dump --defaults-extra-file=/run/private/tenant.cnf tenant_a |
  tongbackup encrypt --tenant tenant-a --key-file /run/private/backup.key \
    --output /srv/private-backups/tenant-a-2026-10-29.tpbk
```

Restore to a new private SQL file, then inspect/validate before importing:

```sh
tongbackup decrypt --tenant tenant-a --key-file /run/private/backup.key \
  --output /run/private/restore.sql < /srv/private-backups/tenant-a-2026-10-29.tpbk
mariadb --defaults-extra-file=/run/private/tenant.cnf tenant_a < /run/private/restore.sql
```

Rotate by re-encrypting with a new externally managed key. The destination must
be a new path:

```sh
tongbackup rotate --tenant tenant-a --old-key-file /run/private/old.key \
  --key-file /run/private/new.key --output /srv/private-backups/tenant-a-rotated.tpbk \
  < /srv/private-backups/tenant-a-2026-10-29.tpbk
```

The disposable CI drill copies only the encrypted artifact and a temporary
test key to the runner, discards the source VM overlay, boots a fresh VM from
the pristine distro image, and restores/verifies the MariaDB marker there. The
runner staging is temporary CI storage, not an offsite backup provider or a
retention service.

## Not yet provided

There is no production key custody or recovery policy, scheduled backup,
offsite provider integration, retention/deletion workflow, application-driven
database export, job/audit wiring, restore preview, user authorization, or
multi-account complete restore. ADR-007 and threat-model owner/security review
remain pending. Do not treat this lab or CLI as a production backup service.
