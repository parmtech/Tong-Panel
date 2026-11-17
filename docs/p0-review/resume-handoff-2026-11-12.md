# P0 resume handoff — 2026-11-12

**Status: draft evidence only. P0 is in progress and not accepted; P1 is incomplete and not production ready.** This handoff does not record any approval or authorize a phase transition.

## GitHub checkpoint

- This branch starts from P5 checklist draft PR [#159](https://github.com/parmtech/Tong-Panel/pull/159), head `17241e2305c379ddaeba2db56d40a43fe76ba9a0` when inspected. Its checklist is already on GitHub.
- P4 provenance draft [#158](https://github.com/parmtech/Tong-Panel/pull/158), P3 scope draft [#160](https://github.com/parmtech/Tong-Panel/pull/160), and P1 mutation draft [#149](https://github.com/parmtech/Tong-Panel/pull/149) were open at the audit. Review their current heads and base dependencies before acting.
- `main` was `0aad68e81138a0f226e83e401f90bae4d0b6bddc` in the inspected snapshot. Refresh remote state before resuming.

## Unfinished work and evidence

A separate local P0 checkout contains modified README and UPDATES files, a staged version of the P5 checklist, and nine untracked P0 reconciliation and review-template drafts. The checklist differs from PR #159 only by a trailing blank line. Redacted copies of eight local review templates are included in this branch; machine-local paths were omitted. Their historical assertions and cross-references need reviewer verification and they do not record approvals. A redacted reconciliation draft is also included; 39 lines with conversation metadata and machine paths were omitted. The original files remain unchanged locally.

Other local worktrees contain uncommitted MFA recovery, P3 model, P5 model and WordPress lab changes. Their bases, overlap with existing remote branches, tests, and public suitability still need individual review. They are not included in this branch. Do not reset or overwrite them.

The previous P0 task recorded P0-05–10 as open, including architecture decisions, independent security review, scope and owner assignments, qualified provenance/licence review, and design/code go/no-go. Verify current primary evidence and obtain the applicable decisions from authorized reviewers; preparation of forms does not constitute approval.

## Next steps

1. Compare each local draft and worktree against its exact current remote base and existing PR. Preserve unrelated changes.
2. Review contents for secrets, private information, attribution and factual accuracy. Prepare redacted, explicitly unfinished draft branches and handoffs; do not publish sensitive material.
3. Run checks appropriate to each change and record actual results. Obtain applicable architecture, security and provenance reviews before treating any gate as passed.
4. Reconcile and update this handoff with verified branch, commit, PR and test evidence as work is saved. Do not claim the complete local inventory is on GitHub until verified.

This handoff is documentation only. No product, security, integration or VM tests were run for it.

## Saved after the initial inventory

- P3 offline deployment recovery model and acceptance-plan changes: draft PR [#162](https://github.com/parmtech/Tong-Panel/pull/162), head `4934cfa8b0be7aaa775f77d6211fad3b6303d3a0`, based on PR #145. The model tests and vet passed; this does not establish product acceptance.
- P5 offline placement model, acceptance specification and trust-review packet: draft PR [#163](https://github.com/parmtech/Tong-Panel/pull/163), head `b8a2cab2dee5a1df299186b5d6ac4913ff898b4e`, based on PR #147. The model tests and vet passed; P5 entry remains blocked.

Original local worktrees were not changed by these preservation copies. Remaining P0, MFA and WordPress work still requires the review described above.

Redacted historical copies of the local README and UPDATES are included as `local-readme-snapshot-2026-11-12.md` and `local-updates-snapshot-2026-11-12.md`; private paths and conversation metadata were omitted. They are not canonical replacements and need factual review. The local P5 checklist is already represented by the base PR except for a trailing blank line.

## MFA recovery local comparison

Nine locally changed or untracked MFA code, migration, test, smoke and primitive-document files were compared by Git blob hash with `feature/p1-mfa-recovery-codes` at `a4e8c73fa394ab727cbf20fc27b219ea46820d14`; all nine matched. Draft PR [#90](https://github.com/parmtech/Tong-Panel/pull/90) contains that branch. The local UPDATES journal includes an additional historical checkpoint which was not found by a narrow phrase search in that branch. It records the primitive as unintegrated, with external pepper provisioning/rotation, recovery orchestration, security review and acceptance still open. Its historical local Go test/vet claims have not been rerun in this audit; verify current PR checks and review before integration. The local journal remains to reconcile.

## WordPress lab inventory

- The local full-install script was preserved on `audit/wp-full-local-checkpoint-20261112` at `1523590bb865a1e760d29403ff662efa9c580b07`, based on `7547a7d`. Bash syntax passed; no VM test ran. Compare against newer PR #89.
- The local runtime script and test were preserved on `audit/wp-runtime-local-checkpoint-20261112` at `d15fc53a70b8856252595acadf60346d05b7b441`, based on `dbc624b`. Bash syntax passed and the Linux test binary cross-compiled; tests were not executed. Compare against newer PR #88.
- In the local restore worktree, the script and three new restore files matched the remote `feature/lab-wordpress-fresh-restore` blobs at the comparison. Its workflow additionally tracks `UPDATES.md`; the two added script paths were also found in the remote workflow. Reconcile that remaining workflow difference and the detached config/fix worktrees before declaring the WordPress inventory complete.

- A local WordPress config test matches the current runtime, full-install and fresh-restore branch versions by Git blob hash. A separate detached VM-script repair was preserved on `audit/wp-fix-local-checkpoint-20261112` at `312c24911efbb495b4beb882e2e59d33cfb2312e`, based on `e6b880d`; its large deletion needs review and it is not a current implementation. Outer shell syntax passed, but no generated-guest or VM execution ran.
- The staged local P3 scope backlog file matches PR #160's remote file by blob hash. The redacted P0 draft and local README/UPDATES snapshots require factual review and reconciliation; omitted private details remain local. Do not clean the machine until all remaining work is preserved and independently verified.

## Further P2 and coordination checkpoint

- P2 planning changes from the separate P2 worktree are preserved in draft PR [#164](https://github.com/parmtech/Tong-Panel/pull/164), head `149d214155390bf8f991b6503cd214b3fabf9198`; whitespace check passed. They do not approve P2 entry.
- The clean coordination worktree head `aec646542074a05e7e90b070110cbe896f5f7fcc` was verified as an ancestor of remote `docs/phase-coordination` head `75d466c392c7d0de696e9cb24358c787ef9edb55`. Two other clean P0/P2 worktree heads were found as exact remote refs. Other folders and history still need inventory; this is not a complete backup certificate.

## Additional local preservation — 2026-11-12

- The historical MFA journal is preserved in `mfa-local-journal-2026-11-10.md`; its unintegrated status and later PR #90 must be read together.
- The historical restore worktree, including the distinct workflow trigger, is preserved on `audit/wp-restore-local-checkpoint-20261112` at `738a5aea9536af73f6bcc4b9ee0df10a51b5c867`. Three scripts passed shell syntax and the Linux sites test binary cross-compiled; tests were not executed. This branch does not replace PR #91.
- WordPress 7.1.4 source inventory CSVs and modernc libc verification manifest are preserved under `evidence/`. Downloaded third-party archives, extracted licence corpus, module caches and the unsigned PR #149 preview archive are not included. Their digests and source context are recorded in the P0 evidence; exact artifacts and applicable notices must be retrieved and reviewed separately. No licence decision or distribution clearance follows.
- The earlier examined local Tong-Panel worktree heads were found as remote branch tips or ancestors; a later hidden-worktree search found additional local heads and uncommitted work, described below. This is a bounded filesystem audit, not proof that no other project file exists elsewhere on the machine. Original P0 files contain omitted private paths/conversation metadata and are not byte-for-byte backed up in this public repository. Review privacy and evidence limits before resetting any machine or deleting history.

## Additional hidden worktree findings

- The local P1 architecture readiness commit `439f2e4abf9e1c9d0caa9aec63c0710451910378` is now preserved at its original commit in draft PR [#165](https://github.com/parmtech/Tong-Panel/pull/165). It is a proposal, not approval.
- The P1 portability/site-read/UI/SAST work from the second hidden worktree is preserved in draft PR [#166](https://github.com/parmtech/Tong-Panel/pull/166), head `75d6e9a9c0afbff5301a2a23df097a5a686099c0`. Windows Go tests and vet for six packages passed, Linux test binaries cross-compiled; hosted/Linux execution, browser and SAST checks remain pending.
- A further local coordination commit `920e62a` was not an ancestor of the inspected remote coordination branch. Its technical content is preserved as `p1-coordination-technical-extract-2026-11-10.md`; private coordination metadata and helper assignments were omitted. The original commit is not backed up byte-for-byte in this public repository.
