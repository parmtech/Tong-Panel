# Inspect retained stages in a disposable restore lab

`tongbackup inspect-restore-lab` is a non-root, read-only **disposable-lab diagnostic**, not a production reconciliation service. It requires `TONG_DISPOSABLE_SITE_RESTORE=1` and `--disposable-only`, and accepts an existing owner-controlled mode-0700 `tong-restore-lab-*` directory under an owner-controlled mode-0700 parent beneath `/tmp`:

```sh
TONG_DISPOSABLE_SITE_RESTORE=1 tongbackup inspect-restore-lab \
  --root /tmp/private-lab-parent/tong-restore-lab-001 --disposable-only
```

It reports only the number of private, correctly named retained archive-stage directories immediately under the pinned root. It never opens their payloads, reports their names or timestamps, deletes them, or modifies quota. It rejects malformed, symlinked, unowned or non-private archive-stage entries and fails rather than returning a partial count if more than 10,000 root entries are encountered. Directory metadata reads may update filesystem access times.

The count is a transient observation: active stages are included when observed, changes can race a scan, and a zero count is not evidence that no restore was published or that a reservation may be released. Cancellation is checked between operations but cannot interrupt a blocked filesystem call or mutex acquisition. It cannot determine whether a stage was published, whether an operation committed, or whether cleanup is safe; all require an approved durable reconciliation procedure. No automated cleanup, production restore, authorization or quota settlement is enabled.

The `/tmp` location and explicit opt-in do not prove disposability. The operator must use a disposable runner or guest and ensure all parent paths and mount topology remain trusted and stable. Canonical path checks do not detect bind-mount aliases or fence another process running as the same user; pathname reacquisition can race a same-user replacement. These are mandatory preconditions, not guarantees from the command. Owner/security and production gates remain open.
