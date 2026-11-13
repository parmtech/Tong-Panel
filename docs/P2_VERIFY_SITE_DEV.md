# Verify a site archive before a restore drill

`tongbackup verify-site` is a Linux-only development utility for checking a tenant-bound encrypted site-file backup before a separate restore drill. It uses the existing private external wrapping-key file checks and accepts only one tenant and one key-file flag:

```sh
tongbackup verify-site --tenant tenant-a --key-file /run/private/backup.key < /srv/private-backups/site-a.tpbk
```

It authenticates the complete envelope, then consumes the entire gzip-tar using the same bounded parser and path/type/collision/size rules as the file extractor. It rejects traversal, symlinks, hardlinks, special entries, duplicate or conflicting paths, truncated or trailing data, and archive limits. No root directory is opened, no stage or plaintext file is created, and success is silent. The bounded compressed plaintext is temporarily held in memory and cleared where practical; Go may retain copies. Cancellation is checked during source reads and before/after parsing, but a blocked reader or already-running parser cannot be forcibly interrupted.

This checks only the bytes consumed during that invocation. It does not authorize a later restore, bind a tenant to a filesystem root, reserve quota, prove current ownership, validate application or database semantics, establish disk permissions/capacity, or guarantee that a changed artifact will restore. Caller-controlled input identity, trusted keys, quiescence, owner/security approvals and a supervised recovery drill remain required. No API or production restore route is added.
