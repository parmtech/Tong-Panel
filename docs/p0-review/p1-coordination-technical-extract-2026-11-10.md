# P1 coordination technical extract — historical draft

This redacted extract is from local commit `920e62a`; coordination identifiers, helper assignments and read receipts are omitted. It is a historical proposal, not an owner approval. The original local commit was not found as an ancestor of the inspected remote coordination branch. Verify every reference and reconcile with current phase owners before reliance.

## P1 exit-report ownership

P1 owns `docs/P1_EXIT_GATE_REPORT_2026-10-27.md`; P1 did not author or pre-approve the P2-authored report changes. I independently compared the exact PR base/head diffs:

- #57: base `957a780ecdfa5c3daf900498f1e8362358e75a56`, head `5d0f3180232fccdc1e84f4c4b303d1989d12ebe6`; adds a seven-line, dated P2 declarative site/domain validation history section, exact earlier workflow head and run evidence, and explicit limits.
- #63: base PR #62 head `8fa5352144a15f2ff72cb82913b285e830cd9827`, head `7bb3719485a5451d14e279ec7e42349c3cd5c723`; changes staged migrations 6→11 and ADR list 001–008→001–009, and adds the historical database metadata checkpoint with scope/gate limits. P1 already accepted this limited report delta in event `p1-20261110T105458Z-4068f207-3568-40ea-967e-60efa382c169`.
- #81: base `22578a13da3980c79c6198e089a9b66b252a5cc4`, head `d947e53c7af2e97b160df3c69d9392b0dc1bf106`; adds a dated encrypted-backup-core note explicitly saying hosted verification is pending and key custody/recovery/offsite/restore plus ADR/threat gates remain open.

P1 accepts these exact #57 and #81 additions as dated, attributable P2 history; the #63 limited acceptance stands. This is not approval of the entire P2 PRs, journal, P1 exit, P2 entry/exit, security decisions or merge. No extra report edits are authorized without another P1 review. Keep #57/#63/#81 draft. PR #57's current guest failure recorded by P2 applies to a later docs-follow-up head and does not rewrite the dated earlier result in its P1 report note.

`UPDATES.md` remains a separate shared-file blocker. Nothing in this report-file review authorizes P2's #57/#63/#81 or #82/#83 journal changes, commit/merge, or a new P1 journal edit. The coordinator has not published a serialized writer/base decision.

## PR #77 source ownership conflict and safe next step

I fetched PR #77 exact head `b62e2f09f212f3a2ab3cecbf6b973682c29906c9`, based on P2 PR #75 head `55ae0b90e0db076df07585b6f7603ff6a4553157`, and reviewed the exact diff. It adds the `/sites/{site}/databases/{database}` GET route/handler in `internal/api/sites_api.go` plus tests in `internal/api/sites_api_test.go`. The P1 branch already contains the route; P1's uncommitted site-read work now changes this handler from `SiteCatalog.GetDatabase(...)` to `GetDatabaseFor(ctx, tenant, site, database, requester, proof)` and similarly replaces direct site/database reads with current-membership/session rechecks. P1's tests cover the same database-detail GET as one of four protected endpoints. This is a confirmed file and behavior overlap, not only adjacent documentation.

P1 pauses further changes to `internal/api/sites_api.go` while the overlap resolution is pending. P1's current site API change remains local/uncommitted on `fix/p1-preflight-windows-build` at `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23`; the local auth/security update is necessary for the accepted conditional P1 read contract. No code from P2's branch was copied or changed.

Proposed safe ownership: P1 retains the current-membership/session security boundary and the overlapping site-read handler. Keep PR #77 draft and do not merge, rebase, retarget or rewrite it until P2 confirms. Once P1's implementation is verified and the shared-journal procedure permits a handoff, P2 may rebase/consume the reviewed P1 read methods and retain its independent route assertions in `internal/api/sites_api_test.go`, after confirming there is no duplicate handler or weaker direct-read path. Please record whether P2 agrees to this sequence or propose another non-destructive, file-exact resolution; neither phase edits the overlapping API until the agreement is in the hub. Independent P1 UI/bootstrap/preflight/testing and independent P2 evidence work continue.

## Current verification and helper ownership

P1's exact focused Go tests passed for `cmd/tongctl`, `internal/preflight`, `internal/sites`, `internal/api`, and `internal/jobs` after correcting the SQLite-reserved probe name. Linux amd64 test binaries for `cmd/tongctl` and preflight compiled; `go vet` passed for the first four packages. The complete Go browser suite ran against a temporary Windows fixture and passed all 20 tests. These are local component results; hosted Linux workflow, production, accessibility/manual and phase gates remain distinct/open.
