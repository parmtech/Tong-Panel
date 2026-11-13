# Disposable site-file restore command

`tongbackup restore-site-lab` is an **experimental disposable-lab utility**, not a production installer or an authorized tenant restore service. Run it only on a disposable Linux runner or guest as a non-root user. The command requires both `TONG_DISPOSABLE_SITE_RESTORE=1` and `--disposable-only`. These are operator acknowledgements: neither the flag nor a path under `/tmp` proves that the environment is disposable.

Provide a trusted, owner-controlled mode-0700 parent beneath `/tmp`; the command creates a **new** `tong-restore-lab-*` directory in that parent and refuses an existing name. All parent paths and mount topology must remain trusted and stable throughout. The wrapping key must be outside the lab root, owner-readable and private under the existing Linux key-file checks. Canonical path comparisons do not detect bind-mount or other aliases and cannot prevent path replacement: the operator must ensure the key, parent and source have no such aliases. The root is deliberately retained on success and failure for supervised inspection. The parent descriptor is pinned for root creation, but the root is subsequently opened by pathname; this does not fence a same-user actor or a replaced ancestor. Parent and mount stability are mandatory caller preconditions.

```sh
TONG_DISPOSABLE_SITE_RESTORE=1 tongbackup restore-site-lab \
  --tenant tenant-a --key-file /tmp/private-lab-parent/key \
  --root /tmp/private-lab-parent/tong-restore-lab-001 \
  --destination site --disposable-only < /tmp/private-lab-parent/site-a.tpbk
```

The input must be an encrypted tenant-bound gzip-tar in the existing envelope format. The command binds the specified tenant to the fresh root, authenticates the entire envelope before extraction, and uses the existing no-follow, bounded, no-overwrite extractor. Successful execution is silent; user-facing failure messages suppress underlying filesystem error text that may contain archive member names. Wrong tenant/key, tampering, truncation and unsafe archives fail; the newly created lab root is not automatically removed. The operator must inspect any retained private staging debris.

If rename reports an ambiguous outcome, the command reports **publication unknown** and retains the entire lab root and any evidence; it must not be retried or cleaned up blindly. If publication succeeded but sync failed, it reports **published but durability uncertain**. Even a not-published error does not prove cleanup completed. A failed root-creation system call is itself reported as outcome unknown, because it may have taken effect; inspect the parent before retrying. Process termination can leave a root or staging data without a returned outcome. These outcomes require supervised reconciliation. There is no quota accounting, authentication/authorization ceremony, publication fencing, database import, service configuration, external key custody, API route or production restore. All owner/security, installer and release gates remain open.
