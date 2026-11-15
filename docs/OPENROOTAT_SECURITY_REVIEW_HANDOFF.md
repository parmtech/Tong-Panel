# Security review handoff: fd-relative restore-lab root opening

**Status:** Independent Sol 6.1 Medium review is pending. Do not describe the
change as specialist-reviewed or integrate it into PR #171 until that review
and the updated Linux run are complete. This document is a review packet, not
an approval.

## Scope

The candidate is based on PR #171 source `dff0a230f3c7840df118efec63fa00011435e9a4`.
The candidate code is local commit
`d98e0f35fab91709ac8536aae1d06ca5343e16d7`; follow-up commit
`2158373c92cb60556de906bccc229f6b9eb39ea5` adds restore-to-inventory process
coverage. Both are based on PR #171 source
`dff0a230f3c7840df118efec63fa00011435e9a4`. The isolated branch-only run at
`43ff309760a629cbaca3651c3d1118cb9b5de57b` passed these candidate changes on
Ubuntu; the PR remains unchanged and still needs specialist review and exact
PR-head CI.

Files in the candidate:

- `internal/files/root_at_linux.go`: `OpenRootAt` accepts one child component,
  performs no-follow metadata observations, opens with `openat2` using
  beneath/no-symlink/no-magic-link/no-cross-mount resolution, compares device
  and inode before/opened/after, and leaves parent-FD ownership with caller.
  `CheckOwnerPrivateDirectory0700` validates owner and mode with `fstat` on
  the pinned descriptor; it does not change the process-global umask.
- `internal/files/root_at_linux_test.go`: path-replaced parent, unsafe
  component, symlink, regular file, missing child and parent-FD ownership.
- `cmd/tongbackup/restore_site_lab_linux.go`: opens the just-created lab root
  relative to the pinned parent descriptor and refuses to read input if umask
  removed required permissions.
- `cmd/tongbackup/restore_site_lab_linux_test.go`: runs the helper in a
  subprocess with umask `0777`, requires a specific unusable-root error before
  input read, verifies the retained mode-000 root, repairs it, and confirms
  the root is empty and inspectable.
- `scripts/test_backup_restore_site_lab_cli.py`: checks the count-only
  inventory after wrong-key denial and successful restore, and verifies the
  inventory did not modify either root.
- `cmd/tongbackup/inspect_restore_lab_linux.go`: opens the selected lab root
  relative to its pinned parent and checks identity around the inventory scan.
- `docs/P2_RESTORE_SITE_LAB.md` and `docs/P2_INSPECT_RESTORE_LAB.md`: disclose
  same-UID, mount-alias, ancestor and first-observation limitations.

The commands remain non-root, explicitly opted-in disposable-lab tools. This
does not add a production endpoint, authorization, quota settlement or a
cross-process filesystem fence.

## Review questions

1. Are the basename validation, `openat2` resolution flags, descriptor flags,
   and before/opened/after identity comparisons correct for this scope?
2. Is the remaining same-UID replacement window before the first `fstatat`
   clearly bounded by the documented trusted-parent precondition? Does either
   caller accidentally imply stronger fencing?
3. Are descriptor ownership, error cleanup, and interaction with `Root.Close`
   safe when the caller owns and retains `parentFD`?
4. Does restore-root creation remain safely private and inspectable under a
   restrictive process umask such as `0777`? The implementation checks the
   opened root's owner/mode using its pinned descriptor before reading the
   encrypted input. A subprocess regression uses umask `0777`, expects a
   truthful static error, verifies the empty mode-000 root is retained, then
   repairs it as its owner and confirms it is inspectable. This case passed in
   the isolated Linux run below.
5. Does unsupported `openat2` fail closed consistently with existing Linux
   support assumptions?
6. Are the tests sufficient to support the stated disposable-only claim,
   without claiming inode fencing against a same-UID actor?

Return a concrete blocker/no-blocker assessment, exact code locations, and any
required regression tests. Do not expand the review into production restore
authorization or quota approval; those gates remain separate and blocked.

## Evidence

- PR #171 exact source `dff0a23` had all 13 workflows successful, including
  both clean VM guests. This verifies only that committed source.
- Branch-only run [43769672249](https://github.com/parmtech/Tong-Panel/actions/runs/43769672249)
  passed on Ubuntu at `c79f589`, before the umask fix.
- Branch-only run [43770584554](https://github.com/parmtech/Tong-Panel/actions/runs/43770584554)
  passed at `beee3dc`, based on `d98e0f3`. The restrictive-umask subprocess
  regression, affected Linux race suites, and non-root restore/inventory CLI
  tests all passed.
- Final combined run [43770861328](https://github.com/parmtech/Tong-Panel/actions/runs/43770861328)
  passed at `43ff309`, with candidate code through `2158373`. Its job shows
  successful explicit umask, race-suite, and process-suite steps, including
  restore-to-inventory checks. Both runs are Ubuntu runner-only evidence, not
  clean-guest or PR-head verification.
- Go 1.27.3 `go test ./cmd/tongbackup` passed;
  `./internal/files` has no Windows build files and therefore did not execute
  on Windows. Linux amd64 test binaries cross-compiled and Linux-targeted
  `go vet ./cmd/tongbackup ./internal/files` passed. `git diff --check` passed.
- The subprocess regression did not run locally because this checkout is on
  Windows; hosted Ubuntu supplied the runtime execution.
- A separate independent code review found no concrete security blocker and
  documented the same-UID/mount-alias caveats; it identified the umask edge
  fixed by `d98e0f3`. The Sol review agent's latest attempt ended on a model
  usage limit, so no Sol review is claimed.
