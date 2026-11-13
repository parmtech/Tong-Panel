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

## Verify before restore

To authenticate one encrypted artifact for its tenant without publishing or printing plaintext, run:

```sh
tongbackup verify --tenant tenant-a --key-file /run/private/backup.key < /srv/private-backups/tenant-a-2026-10-29.tpbk
```

The command reads the complete envelope, verifies the tenant-bound wrapping key, every chunk, final tag and end of stream, and exits successfully without producing an output file or success message. It uses the existing 64 GiB plaintext bound and Linux-only private key-file checks. A wrong key or tenant, tampering, truncation, trailing bytes, input failure or limit violation fails; no partial result is published. It requires exactly one `--tenant` and `--key-file`, and rejects `--output` and `--old-key-file`.

Verification authenticates the bytes consumed in that invocation. It does not inspect SQL semantics, prove that keys are recoverable, perform a restore, establish service health, or guarantee that a changed/replaced artifact will later restore successfully. Keep restore input controlled and still perform a supervised restore drill. The release and security blockers above remain unchanged.

## Rooted encrypted archive restore primitive

On Linux, `files.Root.RestoreEncryptedGzipTarNew` composes the existing tenant-bound envelope authentication with the existing pinned-root gzip-tar extractor. It authenticates the complete encrypted stream before it creates an extraction stage, holds at most the configured compressed archive limit (32 MiB maximum) in a private in-memory buffer, then extracts with the existing size, entry, no-link, beneath-root and no-overwrite checks. The buffer is cleared on return where practical; Go may retain other plaintext copies in memory. This is a low-level unprivileged building block, not an API route, SQL import, WordPress restoration workflow or authorization system.

A trusted caller must bind the current authorized tenant, its pinned root, backup source and quota reservation. The helper does not establish that the tenant identifier corresponds to the supplied root, recheck authorization at publication, or prevent a source from changing outside the supplied reader. Cancellation is checked during reads and just before the atomic no-replace rename, but cannot interrupt a blocked reader and can race publication after the final check. The extractor may return `ErrPublishedDurabilityUnconfirmed` after a successful rename if directory sync/open fails: the target may exist, so callers must reconcile and must not blindly release a quota reservation or retry as if nothing was published. Failed staging cleanup can leave private stage debris. These constraints and the existing owner/security/production gates remain open.

`TenantRoots.RestoreEncryptedGzipTarNewForTenant` narrows one integration hazard: it uses the same supplied tenant ID to select the registered pinned root and authenticate the backup envelope. Unknown tenants are rejected before the encrypted source is read. The ID must still be derived from a currently authorized server-side principal; this method does not authenticate or authorize the caller, coordinate quota or fence a revoked user at publication. No production route is enabled by this wrapper.

## Rooted site-file export primitive

On Linux, `TenantRoots.ExportEncryptedGzipTarForTenant` selects a registered tenant root and uses that same trusted tenant ID as the backup envelope binding. It reads a selected nonempty relative directory through pinned descriptors, produces a bounded gzip-tar in memory and encrypts it into a caller-supplied writer. It does not select or publish the output file: an error can leave partial ciphertext, so a trusted caller must use private staging and publish only after success. The destination must be independent of the source tree. The compressed plaintext buffer is cleared on return where practical; Go may retain other in-memory copies.

The exporter rejects symlinks, multiply-linked files, special files, mount crossings and internal staging names. File, total, entry, depth, expanded and compressed limits apply; modes and timestamps are not preserved. The source must be externally quiesced, including directory renames, for the entire source scan. In-process root mutations are excluded during the scan and metadata changes are checked, but these checks do not create a filesystem snapshot or prove consistency against concurrent external writes. Root locks are released before ciphertext is written; an arbitrary blocked source, destination or lock acquisition cannot be interrupted reliably by cancellation.

An exported archive can be fed to `RestoreEncryptedGzipTarNewForTenant` with the same tenant/key, subject to its publication and quota caveats. This backs up selected files only. It does not export a database, coordinate application quiescence, choose keys, verify current authorization, reserve quota, schedule jobs, write a production artifact or prove recoverability. No API route or production host operation is enabled; owner/security and restore gates remain open.
