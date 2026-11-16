# Security review handoff: fd-relative restore-lab root opening

**Current checkpoint (17 November 2026):** PR #171 remains open/draft/unmerged at exact head db97e1d1a67a24a85ff61d59f0dccc8ca4033de0. Exact Go race/fuzz/vet workflow #43900009544 passed, including source-present OpenRootAt descriptor ownership and SQLite completion-crash tests. Overall exact-head CI still has one Debian P2 step-18 failure; this review is scoped component evidence, not P0-06 acceptance or production restore authorization. Local P2 diagnostics changes are uncommitted and have not run hosted. Keep issue #172 at 12 unchecked DoD items and all nine approval fields false.


## Historical review snapshot — published 2054 / local 3490235 (superseded 17 November 2026)

**Status:** Sol 6.1 Medium independently reviewed exact candidate 43ff309760a629cbaca3651c3d1118cb9b5de57b and found no concrete blocker for the explicitly opted-in, non-root disposable restore-lab/inventory scope; Development and QA also reviewed it without a code or descriptor-cleanup blocker. Those implementation/process-test changes are patch-equivalent to published PR #171 commits 70a1fd8 and 02258ae. Published snapshot at 2026-11-16 12:36 UTC: PR #171 remained open/draft/unmerged at 2054a425108abd4db8df6e84b29c421dafe17dd0, with 16/17 workflows and 24/25 checks successful; Debian step 18 failed twice, while Ubuntu passed once and its attempt-1 success was retained, not rerun. Step 17 passed on both attempt-1 guests and on Debian's attempt-2 rerun. New candidate 34902351b90c0bb7fe8f67d6681a15031f24e5d1 includes OpenRootAt fd-ownership cleanup coverage and the SQLite completion-crash Go test. At this 12:36 UTC snapshot, these candidate-added Go tests had not run locally (Go/WSL unavailable) or on exact-head hosted CI; the later exact b02 Go race/vet run passed, including OpenRootAt and SQLite coverage. The draft P0 crosswalk remains execution-unverified; P0-06 is unchecked and its 43 references resolve. Earlier 2054 hosted Go race/vet results apply only to code at 2054. Root cause of the P2 Debian failure is unknown and the umask hypothesis unconfirmed. This review grants no production restore authorization, quota-release authority, cross-process fence or owner/security approval.

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
- Integrated test-only commit 8412ccc adds nil/closed-root cases, accepts exact mode 0700, and rejects group/other and setuid/setgid/sticky bits after confirming those bits are observable. The Linux-tagged tests remain unexecuted locally because Go/WSL are unavailable; exact-head Go race/vet CI is still required.

The identity observations do **not** prove continuity against same-UID replacement before the first observation, ABA swaps or subsequent mutation. Ancestor replacement during earlier pathname validation and mount aliases remain outside this mechanism. The trusted/stable-parent and topology preconditions are required and documented.

## Evidence and remaining verification

- Exact branch-only Ubuntu run [#43770861328](https://github.com/parmtech/Tong-Panel/actions/runs/43770861328), job #132986235012, passed the restrictive-umask regression, affected Linux race suites and restore/inventory CLI process suites.
- Historical exact PR source b411 passed both Go race runs and P2 runner restore/inventory. P2 run `43832980490` passed Debian job 133182537181 and Ubuntu job 133182537635 through guest smoke, cleanup and Complete.
- Source candidate 61fdb4b8faf520ec277cb5aeef82d4ac3510fe88 is included in exact PR head 2054. Its hosted Go race/vet checks passed, including Linux-tagged tests. At the 2054 snapshot, the P2 Debian guest run remained unresolved as described above; this does not change the bounded OpenRootAt review scope. The later b02 P2 guest run passed and is recorded in the current checkpoint at the top of this file.
- OpenRootAt identity observations do not prove continuity against same-UID replacement before the first observation, ABA swaps or subsequent mutation. Ancestor replacement during earlier pathname validation and mount aliases remain outside this mechanism; stable-parent and topology preconditions remain required.
