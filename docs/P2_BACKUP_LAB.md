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

Restore to a new private SQL file, inspect/validate it, and import only into a
disposable synthetic test database. Never use a real, customer, or production
database as the target:

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

At published source SHA `5257aa3bd3b217a5a28316812618009c6244c01f`, [clean-VM
run #43922690907](https://github.com/parmtech/Tong-Panel/actions/runs/43922690907)
passed guest-smoke step 18 in the [Ubuntu job
#133493840911](https://github.com/parmtech/Tong-Panel/actions/runs/43922690907/job/133493840911)
and [Debian job
#133493841211](https://github.com/parmtech/Tong-Panel/actions/runs/43922690907/job/133493841211).
This verifies the synthetic guest backup/restore smoke only. It does not execute
the proposed scheduled/offsite target flow or satisfy the P2 offsite
backup/restore acceptance gate.

## Not yet provided

There is no production key custody or recovery policy, scheduled backup,
offsite provider integration, retention/deletion workflow, application-driven
database export, job/audit wiring, restore preview, user authorization, or
multi-account complete restore. ADR-007 and threat-model owner/security review
remain pending. Do not treat this lab or CLI as a production backup service.

## Provider-neutral P2 acceptance checks

The following are proposed checks for disposable, non-privileged fixtures; they
do not enable an offsite integration or select a provider or policy:

- Parameterize the maximum transfer size, transfer duration, attempt count,
  retry/backoff schedule, and simultaneous-transfer cap from the reviewed test
  contract. Use a deterministic clock to assert that the fixture stays within
  each selected bound and records a clear terminal result when a limit or
  verification fails. On interrupted or failed uploads, assert that a partial
  remote object is either cleaned up and its removal verified, or left
  explicitly identified for safe reconciliation; it must never be reported as
  a complete backup.
- Bind every synthetic backup to an immutable backup ID, generation marker,
  and expected content digest before transfer. After timeout, interruption, or
  process restart, resume/retry only against that same identity under the
  selected policy and verify the digest before reporting completion. Exercise
  an out-of-order case where a newer generation completes before an older
  attempt resumes; the stale attempt must not overwrite or delete the newer
  artifact, and each attempt must reach an explicit verified, failed, or
  recoverable reconciliation state.
- Use only synthetic test data, disposable test keys and opaque test key
  references. Prove the matching tenant/artifact/key-reference combination for
  each test tenant succeeds, and cross-tenant or mismatched-key/reference
  combinations fail without publishing plaintext or mutating restore state.
  Evidence may include the non-secret fixture reference, but never key material.
- Put the same synthetic generation marker in each tenant's database and
  selected files/settings. Under the quiescence or snapshot contract selected
  for the test, assert that the restored markers agree. Restore into a private,
  non-serving stage; validate the markers and artifact before making the test
  instance available. A mismatch, interrupted restore or inconclusive
  publication must remain non-serving with an explicit recoverable result.

These checks define test assertions, not policy values or production behavior.
Maximum transfer size and duration, attempt/backoff values, simultaneous
transfer cap, retry/restart policy, retention policy, provider guarantees, real
key custody/rotation/recovery, database/service coordination, authorization,
and production restore/rollback require owner, security and provider decisions.
Until those decisions and the existing P0/P1 approvals are recorded, use only
synthetic fixtures and do not connect a production target or service.

### Offline generation-fence model (synthetic, non-normative)

The local [`p2-backup-retry-model`](../prototypes/p2-backup-retry-model/) adds an in-memory Python-standard-library model for immutable synthetic backup ID, generation and expected digest binding. It checks that digest verification precedes publication, an older attempt cannot replace a newer current generation, and stale deletion requests cannot remove that newer artifact. Source commit `d885a256f96bac36b5d96ff76d8e094b802d75f0` rejects boolean attempt IDs that alias integer ID 1 and adds `test_boolean_attempt_id_cannot_alias_integer_attempt`; the local suite passed 7/7. d885a256 is included in PR head `7f602a1c16dd5d1bdcdd6dd5a49fcb3b43c35656` through 570. The prior 022 head passed with synthetic model check #133688439715; the later 7f exact-head CI also passed all 13 workflows and 23 checks, including model check #133717481690. These are synthetic-only results. This model remains non-normative and does not select production policy. The production generation type and persisted format remain undecided.

This prototype has no persistence or restart behavior, provider/storage integration, retry/backoff policy, transfer limits, retention/deletion policy, key custody, tenant authorization, API or filesystem/network operation. PR head 022 includes the original model and workflow plus d885a256's boolean-ID fix; exact-head CI passed 13/13 Actions runs and 23/23 checks, including model check #133688439715. These tests are not P2 acceptance and do not close issue #152.

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
`RestoreEncryptedGzipTarNewWithResult` and its `TenantRoots` counterpart return an `ArchivePublishResult`: `ArchiveNotPublished` means no rename was attempted, `ArchivePublished` means rename returned success, and `ArchivePublicationUnknown` means rename was attempted but returned an error. For published or unknown results, `LogicalBytes` is the validated candidate's regular-file content length, including zero-byte files; it excludes directories, metadata and physical allocation. It is not a disk-usage measurement or authorization to settle quota. A crash can occur before a result is returned, and even not-published operations may leave private stage debris if cleanup fails. Reservations require reconciliation under a separately reviewed recovery policy.

A pre-rename destination check rejects a known existing target without attempting rename; `RENAME_NOREPLACE` remains necessary for races. Any error from an attempted rename, including `EEXIST`, is wrapped with `ErrPublicationOutcomeUnknown` and treated as unknown: the stage descriptor might already refer to the published destination, so cleanup must not recursively delete it. If rename actually failed, a private stage can remain for supervised cleanup. Generic age-based cleanup retains all archive stages: age and an absent lock cannot establish whether publication occurred. No durable journal or approved reconciler yet distinguishes an abandoned stage from unresolved recovery evidence; retained stages may consume storage and require supervised reconciliation. Cleanup does not settle quota. Filesystem rename/cancellation races and the security gates described above remain open.
