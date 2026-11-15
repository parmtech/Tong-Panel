# Security review handoff: fd-relative restore-lab root opening

**Status:** Independent Sol 6.1 Medium review is pending. Do not describe the
change as specialist-reviewed or integrate it into PR #171 until that review
and the updated Linux run are complete. This document is a review packet, not
an approval.

## Scope

The candidate is based on PR #171 source `dff0a230f3c7840df118efec63fa00011435e9a4`.
The isolated QA branch `codex/qa-root-at-runtime-20261116` contains the
candidate at `c79f589abb60ecfbb87201dd6fa1aaebd9e05975`; its branch-only Linux
workflow passed, but the current worktree has an additional umask edge under
review and requires a fresh exact-candidate run.

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
   restrictive process umask such as `0777`? The current follow-up checks the
   opened root's owner/mode using its pinned descriptor before reading the
   encrypted input. A subprocess regression uses umask `0777`, expects a
   truthful static error, verifies the empty mode-000 root is retained, then
   repairs it as its owner and confirms it is inspectable.
5. Does unsupported `openat2` fail closed consistently with existing Linux
   support assumptions?
6. Are the tests sufficient to support the stated disposable-only claim,
   without claiming inode fencing against a same-UID actor?

Return a concrete blocker/no-blocker assessment, exact code locations, and any
required regression tests. Do not expand the review into production restore
authorization or quota approval; those gates remain separate and blocked.

## Evidence before the umask follow-up

- PR #171 exact source `dff0a23` had all 13 workflows successful, including
  both clean VM guests. This verifies only that committed source.
- Branch-only run [43769672249](https://github.com/parmtech/Tong-Panel/actions/runs/43769672249)
  passed on Ubuntu at `c79f589` before the umask follow-up; it ran Linux race tests for
  `./internal/files` and `./cmd/tongbackup` plus both non-root restore and
  inventory CLI process tests. It is runner-only evidence, not clean-guest or
  PR-head verification.
- Umask-follow-up local checks: Go 1.27.3 `go test ./cmd/tongbackup` passed;
  `./internal/files` has no Windows build files and therefore did not execute
  on Windows. Linux amd64 test binaries cross-compiled and Linux-targeted
  `go vet ./cmd/tongbackup ./internal/files` passed. `git diff --check` passed.
- The subprocess regression did not run locally because this checkout is on
  Windows; an exact-candidate Linux workflow is still required.
- A separate independent code review found no concrete security blocker and
  documented the same-UID/mount-alias caveats; it identified the umask edge
  addressed by the uncommitted follow-up. The Sol review agent's latest
  attempt ended on a model usage limit, so no Sol review is claimed.

