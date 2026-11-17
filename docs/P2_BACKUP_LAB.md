# Experimental encrypted backup and restore lab

`tongbackup` is a Linux-only operator utility for exercising the experimental
backup envelope. It is not connected to Tong Panel's API, database lifecycle,
job queue, scheduler, or host agent. It does not make the project production
ready.

## Current synthetic component evidence (final root rerun 2026-11-17T17:54:08Z; GitHub read window 2026-11-17T18:46:49Z–18:46:58Z)

Five focused P2 suites passed 101/101: published schedule-to-run identity 11/11; retention reconciliation 19/19 within the exact 8-path retention/Go subinventory; separate 3-path backup-retry tenant scoping 43/43; separate 3-path scheduled-run-to-backup-identity handoff 12/12; and separate 3-path restore-generation uncertainty 16/16. Keep the work inventories separate; do not infer an aggregate path count. Reciprocal Go tests in the retention inventory remain unrun because Go/gofmt are unavailable.

PR #171 remains open/draft/unmerged at c1fbf710e2b530d3b9778fc2ac2f2a53b6ba94c2. Its exact-head CI passed 13/13 PR Actions and 26/26 checks at 2026-11-17T16:52:07Z; this result covers c1 only. PR/ref and body readback occurred in the 18:46:49Z–18:46:58Z window, while API `updated_at` remains 16:48:42Z for the PR and 16:49:16Z for #172. Issue #172 remains open at 0/12 with nine approvals false; #152 remains open/design-only (`updated_at=2026-11-16T19:00:59Z`).

The retention model uses a shared operation-ID namespace and rejects single/batch cross-API reuse. QA confirmed the handoff value guard when configured with canonical `retry_contract.Identity`: it requires that exact type and validates the frozen, hashable identity value before run creation. The constructor accepts a caller-supplied frozen dataclass class, so callers must inject the canonical retry type. The tenant check is fixture consistency, not authorization. Restore read-back validates the full bundle before equality and rejects boolean/float generation aliases. These remain synthetic fixtures: no offsite provider, production scheduler, actual deletion, authorization, or durable restore is implemented. Row 9 remains **UNMET**, and issue #152 remains design-only.

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

There is no production key custody or recovery policy, integrated scheduled
backup, offsite provider integration, production retention/deletion workflow,
application-driven database export, job/audit wiring, restore preview, user
authorization, or multi-account complete restore. ADR-007 and threat-model owner/security review
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

### Historical offline generation-fence and transfer snapshots (synthetic, non-normative; superseded by the current candidate status below)

**Historical generation-fence snapshot:** The local p2-backup-retry-model adds an in-memory Python-standard-library model for immutable synthetic backup ID, generation and expected digest binding. It checks that digest verification precedes publication, an older attempt cannot replace a newer current generation, and stale deletion requests cannot remove that newer artifact. Source commit d885a256 rejects boolean attempt IDs that alias integer ID 1 and adds test_boolean_attempt_id_cannot_alias_integer_attempt; its local suite passed 7/7. d885 is included through 570 in pushed PR head 2f. Historical model checks #133688439715 (022) and #133717481690 (7f) apply only to those heads. The 2f backup-retry model check #133748137953 passed. This model remains synthetic and non-normative; it does not select production policy. The production generation type and persisted format remain undecided.

**Historical pushed-head 2f snapshot:** The model present in pushed PR head 2f had no transfer simulator, modeled transfer retry/backoff, size/duration limits, or concurrency controls; it also had no persistence/restart behavior, provider/storage integration, retention/deletion policy, key custody, tenant authorization, or product API/filesystem/network operation. Exact-head 2f CI was reread at 01:46:45Z with 13/13 Actions and 23/23 checks successful; P2 run #43996007201 completed at 01:45:06Z, including Debian/Ubuntu guest jobs #133748137743/#133748138165. The 022 and 7f CI results remain historical. These tests are not P2 acceptance and do not close issue #152.

**Historical post-7f synthetic-transfer successor snapshot (GitHub read 2026-11-17T03:06:06Z):** The synthetic TransferSimulator in this lab is included in 7f; its exact-head model check failed because finish_transfer looked for identity on _TransferAttempt. Local fix 3ff reads the digest identity from the owning _TransferOperation; the focused Python suite passes 17/17. At that snapshot the fix was not in the remote PR and had no hosted result. P2 clean-VM run #44004387562 passed on Debian #133776021340 and Ubuntu #133776021100. The model remains in-memory and synthetic: it performs no real transfer, provider call, remote object removal, persistence or recovery policy; no acceptance or key-custody decision follows.

### Current published snapshot (c1fbf71; exact-head CI passed at 2026-11-17T16:52:07Z; body read window 18:46:49Z–18:46:58Z)

PR #171 remains open/draft/unmerged at c1fbf710e2b530d3b9778fc2ac2f2a53b6ba94c2. Exact-head CI passed all 13 pull_request Actions runs and 26 checks. The hosted offline-model job passed backup-retry #134184809724, restore-generation consistency #134184809534, and schedule-to-run #134184809837. The P2 clean-VM run #44124596264 passed Debian #134184804136 and Ubuntu #134184803775 through guest smoke, cleanup, and Complete. No raw VM or serial logs were read. These checks are synthetic/component evidence; issue #152 remains open/design-only and the model results do not meet integrated P2 acceptance.

### Historical synthetic checkpoint snapshot (published 3f8cc51; superseded by c1)

The reviewed checkpoint/reconstruction candidate is published at 3f8cc51. The local prototype suite passed 39/39, including clock-failure atomicity. Sol fixed the stale successful same-generation conflicting-digest and substituted-later-publisher findings while preserving exact-time ambiguity; Luna QA found no counterexample; root review added and independently QA-reviewed the narrow huge-integer overflow guard/regression. The prototype remains synthetic and in-memory: actual process restart, durable serialization/recovery, provider or product behavior, and P2 acceptance remain out of scope. Exact-head CI was observed at 2026-11-17T15:07:12Z: 13/13 pull_request Actions runs and 24/24 checks passed; no exact-SHA push runs, failures, or pending checks. Offline model run #44107182517 passed. P1 blocker inventory run #44107182674/check #134122595766 passed. P2 check #134122595446 passed. P2 clean-VM run #44107182469 passed Ubuntu job #134122594513 and Debian job #134122595784 through guest smoke and Complete. No raw VM or serial logs were read. The issue #152 API/body was reread at 2026-11-17T15:07:12Z: it remains open and design-only; returned updated_at is 2026-11-16T19:00:59Z. The separate issue #172 API/body read at that time remains open at 0/12, with all nine approvals false; its returned updated_at is 2026-11-17T12:42:30Z. This is a static snapshot; reread the relevant tracker before further action.

TransferSimulator.checkpoint() requires a non-empty synthetic epoch string and a previously observed finite clock anchor. SyntheticCheckpoint.reconstruct() requires the same epoch string and a supplied clock whose first reading exactly equals the saved anchor. It rebuilds only the fixture and preserves its identity/digest and size bindings, current artifacts, operation and attempt histories and counters, retry limits/backoff, and reconciliation states. Active or inconsistent state is rejected. Elapsed downtime has no modeled meaning; the checkpoint is not a disk/database/JSON or production persistence format. These synthetic assertions are not P2 acceptance and do not close issue #152. Theme persistence is unspecified and deferred; the P1 Jobs screen statement remains scoped to that screen.

### Local engineering follow-up after c1 (8-path scoped subinventory; no hosted CI)

The exact eight-path engineering subinventory adds the in-memory retention-delete reconciliation model, focused tests, offline-model CI wiring, two P2 notes, and reciprocal tenant-denial Go regressions. The retention model accepts only injected verified-absence evidence for a fixture intent and performs no deletion; its focused suite passed 19/19 at 2026-11-17T17:54:08Z. Go/gofmt are unavailable locally, so the reciprocal Go tests are unrun. A separate three-file backup-retry tenant-scoping extension passed 43/43 and independent review found no counterexample; tenant identity is fixture data, not authorization or remote storage isolation. The separate three-path scheduled handoff passed 12/12 at the same final rerun; callers must inject canonical `retry_contract.Identity` because the constructor accepts a caller-supplied frozen dataclass class. The type/value guard provides fixture consistency, not authorization. Separate restore uncertainty passed 16/16. These models remain synthetic and do not provide a provider, scheduling, persistence, real deletion, or integrated restore.

Other post-c1 scopes remain separate with hosted CI pending: original Go/P2/P1-gate workflow pinning; additional P1-smoke/schema/UI pinning; and the OpenAPI report decoder. The OpenAPI regressions are unrun because Go/gofmt are unavailable; API-contract PR CI is pending. P0 crosswalk passed 12/12 structurally without cited tests or mitigations; the clean-VM regression passed locally, with hosted verification pending and no diagnosis of historic guest failures. P1-09 supportbundle changes remain test-only/unrun; QA noted existing 4 KiB/redaction tests provide stronger coverage but do not prove implementation. These edits do not close issue #152, satisfy row 9, or close any P2 gate.

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
