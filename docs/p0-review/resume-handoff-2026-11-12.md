# P0 resume handoff — 2026-11-12

**Status: draft evidence only. P0 is in progress and not accepted; P1 is incomplete and not production ready.** This handoff does not record any approval or authorize a phase transition.

## GitHub checkpoint

- This branch starts from P5 checklist draft PR [#159](https://github.com/parmtech/Tong-Panel/pull/159), head `17241e2305c379ddaeba2db56d40a43fe76ba9a0` when inspected. Its checklist is already on GitHub.
- P4 provenance draft [#158](https://github.com/parmtech/Tong-Panel/pull/158), P3 scope draft [#160](https://github.com/parmtech/Tong-Panel/pull/160), and P1 mutation draft [#149](https://github.com/parmtech/Tong-Panel/pull/149) were open at the audit. Review their current heads and base dependencies before acting.
- `main` was `0aad68e81138a0f226e83e401f90bae4d0b6bddc` in the inspected snapshot. Refresh remote state before resuming.

## Unfinished work and evidence

A separate local P0 checkout contains modified README and UPDATES files, a staged version of the P5 checklist, and nine untracked P0 reconciliation and review-template drafts. The checklist differs from PR #159 only by a trailing blank line. The nine local drafts are not yet included in this branch. A redacted copy was prepared locally, but further privacy and factual review is required before public push. The original files remain local. These are incomplete working documents, not approvals.

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

The local README and UPDATES changes have not been copied to this branch and still require comparison and privacy review. The local P5 checklist is already represented by the base PR except for a trailing blank line.

## MFA recovery local comparison

Nine locally changed or untracked MFA code, migration, test, smoke and primitive-document files were compared by Git blob hash with `feature/p1-mfa-recovery-codes` at `a4e8c73fa394ab727cbf20fc27b219ea46820d14`; all nine matched. Draft PR [#90](https://github.com/parmtech/Tong-Panel/pull/90) contains that branch. The local UPDATES journal includes an additional historical checkpoint which was not found by a narrow phrase search in that branch. It records the primitive as unintegrated, with external pepper provisioning/rotation, recovery orchestration, security review and acceptance still open. Its historical local Go test/vet claims have not been rerun in this audit; verify current PR checks and review before integration. The local journal remains to reconcile.
