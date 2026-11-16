# Security review handoff: fd-relative restore-lab root opening

**Status:** Sol 6.1 Medium independently reviewed exact candidate `43ff309760a629cbaca3651c3d1118cb9b5de57b` and found no concrete blocker for the explicitly opted-in, non-root disposable restore-lab/inventory scope. Development and QA also reviewed it without a code or descriptor-cleanup blocker. The implementation and process-test changes are patch-equivalent to published PR #171 commits `70a1fd8` and `02258ae`. Published exact head `98788613f7dd0aee0648c0f32d89fe7c5c21aab1` passed 13/13 workflows and 21/21 checks, including Go race and P2 Ubuntu/Debian guest smoke, cleanup and Complete. Local integrated candidate `1f138335c0a3b8dea5320c5e49f7cead3489a6f2` contains additional unrelated diagnostics, UI and backup-test changes and awaits exact-head hosted CI. This review grants no production restore authorization, quota-release authority, cross-process fence or owner/security approval.

## Scope

The isolated candidate branch is `codex/qa-root-at-final-20261116` at `43ff309760a629cbaca3651c3d1118cb9b5de57b`, based on PR #171 source `dff0a230f3c7840df118efec63fa00011435e9a4`.

The source changes are `d98e0f35fab91709ac8536aae1d06ca5343e16d7` (pinned-root permission check and umask regression) and `2158373c92cb60556de906bccc229f6b9eb39ea5` (restore-to-inventory process coverage). The final candidate branch also contains documentation and an isolated branch-only workflow. The local forward-port keeps the source/tests and P2 lab documentation, replaces this handoff with this reviewed packet, and omits the one-off workflow.

Files and functions:

- `internal/files/root_at_linux.go`: `OpenRootAt` accepts one child component, performs no-follow metadata observations, opens with `openat2` using beneath/no-symlink/no-magic-link/no-cross-mount resolution, compares device and inode before/opened/after, and closes the descriptor on mismatch. The caller retains ownership of `parentFD`.
- `internal/files/root_at_linux.go`: `CheckOwnerPrivateDirectory0700` checks owner and exact mode with `fstat` on the pinned descriptor; it does not change the process-global umask.
- `internal/files/root_at_linux_test.go`: pinned-parent replacement, unsafe component, symlink, regular file, missing child and parent-FD ownership cases.
- `cmd/tongbackup/restore_site_lab_linux.go`: opens the new lab root relative to the pinned parent and refuses to read restore input if umask removed required permissions.
- `cmd/tongbackup/restore_site_lab_linux_test.go`: runs a subprocess with umask `0777`, verifies an unusable-root error before input read and retained mode-000 root, then repairs permissions and uses `os.ReadDir` to confirm the directory is empty. This test does not itself invoke the inventory CLI.
- `cmd/tongbackup/inspect_restore_lab_linux.go`: opens the selected root relative to its pinned parent and checks identity around the bounded inventory scan.
- `scripts/test_backup_restore_site_lab_cli.py`: checks count-only inventory after wrong-key denial and successful restore, and verifies inventory does not modify either root.
- `docs/P2_RESTORE_SITE_LAB.md` and `docs/P2_INSPECT_RESTORE_LAB.md`: document preconditions and same-UID, mount-alias and ancestor limitations.

The utilities remain non-root, explicitly opted-in disposable-lab tools. They add no production endpoint, authorization, quota settlement or cross-process filesystem fence.

## Sol 6.1 Medium review — 16 November 2026

No concrete blocker was found at exact candidate head `43ff309760a629cbaca3651c3d1118cb9b5de57b` for the bounded disposable-only claim.

- `OpenRootAt` rejects invalid basenames and uses `O_PATH|O_DIRECTORY|O_CLOEXEC|O_NOFOLLOW` with `RESOLVE_BENEATH|RESOLVE_NO_SYMLINKS|RESOLVE_NO_MAGICLINKS|RESOLVE_NO_XDEV`; it fails closed without a fallback. Before/opened/after device+inode comparisons detect some replacements.
- Descriptor ownership and post-open error cleanup are sound. `CheckOwnerPrivateDirectory0700` holds the root read lock while checking its pinned descriptor, excluding `Root.Close` during the check.
- Restore pins the parent, creates the child with `Mkdirat(...,0700)`, opens relative to that descriptor and validates exact pinned owner/mode before encrypted input reaches the restore routine. Restrictive umask fails truthfully, retains evidence and does not modify global umask.
- Inventory pins parent/root and rechecks the visible name’s device/inode and private owner/mode around the scan before emitting a result.
- Optional future coverage: nil/closed owner checks, exact-mode rejection with special bits, and injected identity-mismatch cleanup. These are not blockers for the current scope.

The identity observations do **not** prove continuity against same-UID replacement before the first observation, ABA swaps or subsequent mutation. Ancestor replacement during earlier pathname validation and mount aliases remain outside this mechanism. The trusted/stable-parent and topology preconditions are required and documented.

## Evidence and remaining verification

- Exact branch-only Ubuntu run [#43770861328](https://github.com/parmtech/Tong-Panel/actions/runs/43770861328), job #132986235012, passed the restrictive-umask regression, affected Linux race suites and restore/inventory CLI process suites.
- Exact PR head `98788613f7dd0aee0648c0f32d89fe7c5c21aab1` passed both Go race runs and P2 runner restore/inventory. P2 run [#43822023444](https://github.com/parmtech/Tong-Panel/actions/runs/43822023444) passed Debian job [#133147055374](https://github.com/parmtech/Tong-Panel/actions/runs/43822023444/job/133147055374) and Ubuntu job [#133147055116](https://github.com/parmtech/Tong-Panel/actions/runs/43822023444/job/133147055116) through guest smoke, cleanup and Complete.
- The local integrated candidate `1f138335c0a3b8dea5320c5e49f7cead3489a6f2` has not run at an exact PR head. Its new changes do not alter the reviewed OpenRootAt implementation; inspect candidate CI before recording additional exact-head evidence.
- OpenRootAt identity observations do not prove continuity against same-UID replacement before the first observation, ABA swaps or subsequent mutation. Ancestor replacement during earlier pathname validation and mount aliases remain outside this mechanism; stable-parent and topology preconditions remain required.

Before treating local candidate 1f1383 as verified, inspect its exact-head run and both guest workflows through cleanup and Complete. Do not expand this review into production restore authorization, durable quota release, cross-process fencing or release approval.
