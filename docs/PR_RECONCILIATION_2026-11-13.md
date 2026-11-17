# Tong Panel — complete draft PR reconciliation

**Date:** 13 November 2026 (Australia/Sydney).

**Scope:** all 149 open Tong-Panel draft PRs found when the owner requested “fix all PRs”.

**Review branch:** `integration/pr-reconciliation-20261113`.

## What was inspected and preserved

Implementation commit: [`4aea031`](https://github.com/parmtech/Tong-Panel/commit/4aea031878ef39ad82319b0db402cec50ceee3c9). Subsequent documentation and exact-head hosted results are recorded in the [integration draft](https://github.com/parmtech/Tong-Panel/pulls?q=is%3Apr+head%3Aintegration%2Fpr-reconciliation-20261113).

Every original PR received a current-head/base read, mergeability check, exact-head workflow inventory and submitted-review read. Comments were inspected when present. The complete [machine-readable ledger](PR_RECONCILIATION_2026-11-13.json) records each source SHA, source PR, latest workflow IDs/outcomes and observation time. It contains metadata and evidence links rather than copied workflow logs or credentials.

| Original-head audit result | PR count |
|---|---:|
| All observed latest workflows successful | 129 |
| A latest workflow failed | 5 |
| A latest workflow was cancelled | 1 |
| No exact-head workflows observed | 14 |
| Total | 149 |

The ledger covers 591 latest-per-workflow results: 585 success, five failure and one cancellation. GitHub reported all 149 original PRs mergeable against their respective reported bases. There were zero submitted reviews or inline review comments, and one historical author comment on #145. These facts do not supply owner/security approval. The no-workflow entries are not counted as passing tests; five of them contain offline model/fixture code.

The original PR170 head `1dd0ae3cead20a2d042c7a491a2d825f93fda12a` subsequently completed all eight triggered workflows, including both clean guests. That newer observation replaces its earlier pending entry in this ledger. It verifies that original source, not the combined branch.

All **149 original head commits are ancestors of the combined review history**, and the documentation-only default branch is included. No original branch or commit is deleted. Of the original heads, 131 were already ancestors of PR170; 18 needed reconciliation. Older checkpoints remain available at their original commits. The current journal and handoff retain the latest source of truth instead of replacing it with an October checkpoint.

## Reconciliation decisions

| Source drafts | Decision and preserved result |
|---|---|
| #18 | Preserve the current Argon2 implementation and later no-downgrade rehash protection. The original runtime/tests were already integrated; restore the missing ADR-009 register entry and dated supporting documentation. |
| #19 | Preserve the integrated accessible status UI and later jobs/sites/Files/ACME views. The original `panel-status.mjs` is identical; this historical merge has no first-parent tree delta. |
| #71 | Preserve the current portable read-only doctor, original preflight tests and later support diagnostics. This historical merge has no first-parent tree delta. |
| #76 | Add the separate P0 architecture technical review as advisory evidence, with its pending owner/security status intact. |
| #85 | Restore the missing absolute DNS lookup and real loopback resolver tests; correct the newer authorized proof path and its fixtures to use the same rooted name. |
| #88 | Preserve current WordPress runtime/isolation corrections. This historical merge has no first-parent tree delta; its original failed VM results remain recorded below. |
| #144/#164 | Preserve P2 traceability, cron/log/template policy options and integrated acceptance proposals alongside current implementation history. |
| #145/#162 | Preserve the latest P3 deployment recovery model, admission model and synthetic Node/Python fixtures. |
| #146 | Preserve the standalone P4 migration-review binding model and acceptance proposal. |
| #147/#163 | Preserve P5 outcome, placement and plugin-capability models with the trust review packet. |
| #158/#159/#160/#161/#165 | Preserve the published P0 provenance, scope, exit-evidence and architecture-readiness packets. Proposal/template status remains unchanged. |

Source histories are retained as parents, so the integrated review can replace the overlapping draft queue without discarding source work. The integration PR records subsequent validation and closure of superseded drafts. Closing a superseded draft is not a merge into `main`, acceptance of an ADR or a production release.

## Historical failed and cancelled workflows

| PR / original run | Observed failure | Reconciliation and remaining interpretation |
|---|---|---|
| [#30 / 40589216600](https://github.com/parmtech/Tong-Panel/actions/runs/40589216600) | Concurrent startup failed at `PingContext` with `SQLITE_BUSY`. | Preserve fix [`46fa86f`](https://github.com/parmtech/Tong-Panel/commit/46fa86f19820c21b1fa6d645cec694d77fa27561): bounded caller-cancellable startup retry only for SQLite busy. Current concurrent-open, cancellation and fatal-error regressions passed. |
| [#57 / 40722043238](https://github.com/parmtech/Tong-Panel/actions/runs/40722043238) and [#59 / 40725113404](https://github.com/parmtech/Tong-Panel/actions/runs/40725113404) | Debian SSH reset while cloud-init restarted sshd; each Ubuntu peer passed. | Preserve existing readiness fix [`098a1bb`](https://github.com/parmtech/Tong-Panel/commit/098a1bbf38108e3c0ea03e0ffac0536674b753c7), then restrict SSH retries to transport exit 255. A guest command error now returns immediately rather than being retried into apparent success. |
| [#88 / 42631436094](https://github.com/parmtech/Tong-Panel/actions/runs/42631436094) | Both guests failed the WordPress installer-text assertion; the returned page was not recorded. | Preserve later isolation-before-readiness change [`66bb018`](https://github.com/parmtech/Tong-Panel/commit/66bb018c7cd89797f75b7c820e31c07b0352e206) and current lab corrections. Do not infer the missing original response body. New combined-head VM checks are required. |
| [#96 / 42722907394](https://github.com/parmtech/Tong-Panel/actions/runs/42722907394) | Ubuntu failed the HTTPS tenant-marker assertion; Debian passed. | Preserve bounded diagnostics [`0345de6`](https://github.com/parmtech/Tong-Panel/commit/0345de6b7081192cd65156a42a0dfc3c30967277). The old log cannot establish the lower-level HTTPS cause. Isolation assertions remain strict, with no retry of a wrong-tenant result. |
| [#108 / 42740006289](https://github.com/parmtech/Tong-Panel/actions/runs/42740006289) | Ubuntu cancelled during package configuration after 40m16s; Debian passed. | Preserve cancellation as a non-pass. Timing fits the existing 40-minute job bound, but the cancellation actor/reason was not proved. Obtain fresh combined-head guest evidence rather than relabelling the historical run. |

Historical failure/cancellation records stay unchanged. The consolidated PR's successful checks, if completed, apply only to its exact new head.

## Concrete fixes and coverage added

The DNS integration restores the absolute `_tong-panel-challenge` query for both trusted and authorized verification. Existing API tests reproduced a valid proof returning `403 domain_proof_invalid` until the authorized path and three later fixture registrations were aligned. Domain authorization, evidence, audit and proof-consumption rules are preserved.

The Jobs read integration fixes a confirmed post-authentication gap: twelve baseline list/page/detail cases still returned HTTP 200 after session revocation, rotation, absolute expiry or idle expiry. The new session-aware reads check the current session, membership and owned jobs in one statement snapshot, including empty and missing results. Rows close before cancellation-aware expiry recording. Session denial returns 401, wrong-pool proof returns 503, and cancelled requests expose no data. Existing bearer/trusted reads and role grants remain unchanged. Twenty-one expanded withdrawal cases and additional scope/pool/contention cases pass.

The VM helper is sourced as definitions only. It allows at most twelve SSH attempts and eleven five-second pauses for transport failures, while a non-transport guest error propagates on its first attempt. An individual command still relies on the existing 40-minute workflow job timeout. The actual smoke/restore commands remain direct checked operations. A regression proves that guest exit 2 followed by a hypothetical success must remain exit 2, and companion cases cover immediate readiness, a recovered SSH restart and exhausted transport retries.

The new [offline model workflow](../.github/workflows/offline-model-checks.yml) explicitly checks all five Go models, including the four nested modules that root Go commands do not traverse. Each receives formatting, race tests and vet with Go 1.27.4, read-only manifests and module downloads disabled. A separate bounded job runs the synthetic Node loopback and Python in-process fixtures. These are model/fixture checks, not P3/P4/P5 end-to-end acceptance or permission to deploy workloads.

## Validation and review record

Executed with official Go 1.27.4 linux/amd64 and race detection:

| Check | Executed result |
|---|---|
| DNS baseline on the unreconciled authorized path | Existing valid-proof API regression failed, 403 `domain_proof_invalid` |
| Sites full race suite after DNS fix | Passed, 18.043s |
| API full race suite after DNS fix | Passed, 42.805s |
| SQLite startup/concurrency/cancellation/fatal-error regressions | Passed, 1.432s |
| Job-session baseline, twelve withdrawal cases | Failed as expected, all twelve returned 200, 3.240s |
| Expanded focused jobs/API race | Passed, 4.383s / 10.762s |
| Complete jobs race suite | Passed, 32.225s |
| Final combined API race suite | Passed, 51.085s |
| Affected jobs/sites/API vet | Passed |
| Five offline model race suites / vet / formatting | Passed; individual race runs 1.013–1.025s |
| Synthetic Node / Python fixtures | Passed, 2/2 each |
| SSH helper baseline: guest error then hypothetical success | New regression failed against old helper, which returned success after two calls |
| Four SSH helper regressions, shell syntax and workflow lint | Passed |

Model checks used an empty module cache with downloads disabled; Node 24.22.1 and Python 3.12.14 ran the standard-library fixtures. Independent source/merge, job-session and documentation reviews found no concrete blocker. This is development review, not the outstanding independent production security approval.

Executed local and exact-head hosted results are recorded in [UPDATES.md](../UPDATES.md), the [resume handoff](RESUME_HANDOFF_2026-11-12.md), and the integration PR. The original-head table below is an audit snapshot, not the new combined validation result. The local executor's Unix-socket restriction is not worked around by skipping or weakening tests; full root/Unix-socket and clean-guest results remain hosted checks.

P0 owner/security decisions, production authentication, certificate/terminal authority, persistent key custody, signed installer/recovery and release acceptance remain open. Existing untriaged SAST and provenance findings are not cleared by a reporting workflow's success. No real customer deployment or privileged host adapter is enabled by reconciliation.

## Complete original-head inventory

| PR | Original head | Original exact-head checks |
|---|---|---|
| [#13](https://github.com/parmtech/Tong-Panel/pull/13) | [`ef57fd0`](https://github.com/parmtech/Tong-Panel/commit/ef57fd0b94b69b2ad83df084012bad848bd10d68) | 1/1 success |
| [#14](https://github.com/parmtech/Tong-Panel/pull/14) | [`a246ac7`](https://github.com/parmtech/Tong-Panel/commit/a246ac7950d3e99ff5794333acbf584f3d8bd53d) | 1/1 success |
| [#15](https://github.com/parmtech/Tong-Panel/pull/15) | [`e65496d`](https://github.com/parmtech/Tong-Panel/commit/e65496dc817942addf443c431459df84624b3bfe) | 2/2 success |
| [#16](https://github.com/parmtech/Tong-Panel/pull/16) | [`4387fdd`](https://github.com/parmtech/Tong-Panel/commit/4387fddac5a8796fb765fd6d5b1070cae0b7014a) | 2/2 success |
| [#17](https://github.com/parmtech/Tong-Panel/pull/17) | [`308bda0`](https://github.com/parmtech/Tong-Panel/commit/308bda0283e4ee0e01f4726be7a787fc9289abfe) | 2/2 success |
| [#18](https://github.com/parmtech/Tong-Panel/pull/18) | [`dcc95b3`](https://github.com/parmtech/Tong-Panel/commit/dcc95b35cfb29c0cd9be6db73476741a25140092) | 1/1 success |
| [#19](https://github.com/parmtech/Tong-Panel/pull/19) | [`5f58143`](https://github.com/parmtech/Tong-Panel/commit/5f5814392c8e4845464cf71ef08d1ed873fef604) | 2/2 success |
| [#20](https://github.com/parmtech/Tong-Panel/pull/20) | [`88b46a3`](https://github.com/parmtech/Tong-Panel/commit/88b46a3415b5ed88634d69b3260864a8416fd92c) | 1/1 success |
| [#21](https://github.com/parmtech/Tong-Panel/pull/21) | [`821ed5f`](https://github.com/parmtech/Tong-Panel/commit/821ed5f519fa88c3bd82949f2d22acab675b48ce) | 2/2 success |
| [#22](https://github.com/parmtech/Tong-Panel/pull/22) | [`ac2b78f`](https://github.com/parmtech/Tong-Panel/commit/ac2b78f4eae3567eb8671211de317b43b05e788e) | 1/1 success |
| [#23](https://github.com/parmtech/Tong-Panel/pull/23) | [`9012698`](https://github.com/parmtech/Tong-Panel/commit/9012698d17ac4357ea053bc7d23ea15caffd2449) | 1/1 success |
| [#24](https://github.com/parmtech/Tong-Panel/pull/24) | [`a077ff4`](https://github.com/parmtech/Tong-Panel/commit/a077ff47633f291236a4c97708600869d1fbce65) | 1/1 success |
| [#25](https://github.com/parmtech/Tong-Panel/pull/25) | [`2864d49`](https://github.com/parmtech/Tong-Panel/commit/2864d49ff023b4122e81bb18495536bc321e8569) | 2/2 success |
| [#26](https://github.com/parmtech/Tong-Panel/pull/26) | [`7d215f8`](https://github.com/parmtech/Tong-Panel/commit/7d215f81dd7101a1d1117abdbc91e7e89a4760f8) | 4/4 success |
| [#27](https://github.com/parmtech/Tong-Panel/pull/27) | [`19d375e`](https://github.com/parmtech/Tong-Panel/commit/19d375e65ff0a20485607d2433a2a07452cd26ba) | 3/3 success |
| [#28](https://github.com/parmtech/Tong-Panel/pull/28) | [`c3b0f67`](https://github.com/parmtech/Tong-Panel/commit/c3b0f67927fa35c0cd874ccb1f19bd8b75ed26b3) | 4/4 success |
| [#29](https://github.com/parmtech/Tong-Panel/pull/29) | [`b39ebe5`](https://github.com/parmtech/Tong-Panel/commit/b39ebe5ff48a9945cb4b1786207aea19656941b2) | 4/4 success |
| [#30](https://github.com/parmtech/Tong-Panel/pull/30) | [`a5dca4e`](https://github.com/parmtech/Tong-Panel/commit/a5dca4e940d558fee394880c7d567e7dc588eeec) | 2/3 success; [failure](https://github.com/parmtech/Tong-Panel/actions/runs/40589216600) |
| [#31](https://github.com/parmtech/Tong-Panel/pull/31) | [`865794e`](https://github.com/parmtech/Tong-Panel/commit/865794e4872095f43580dece621c913c8579f38d) | 4/4 success |
| [#32](https://github.com/parmtech/Tong-Panel/pull/32) | [`40d73d9`](https://github.com/parmtech/Tong-Panel/commit/40d73d979f3165a65f110b7bc494d1e3d23e10fb) | 3/3 success |
| [#33](https://github.com/parmtech/Tong-Panel/pull/33) | [`2f5f0ce`](https://github.com/parmtech/Tong-Panel/commit/2f5f0ce7e0d52929df5fb9dcd341921adc8372ab) | 4/4 success |
| [#34](https://github.com/parmtech/Tong-Panel/pull/34) | [`533c2ef`](https://github.com/parmtech/Tong-Panel/commit/533c2efd223ca4de7e2d55170dd16ed4f1bff641) | 3/3 success |
| [#35](https://github.com/parmtech/Tong-Panel/pull/35) | [`3c99386`](https://github.com/parmtech/Tong-Panel/commit/3c99386f13c02690e2c0f26f9764be900b77ff65) | 3/3 success |
| [#36](https://github.com/parmtech/Tong-Panel/pull/36) | [`1ee83b1`](https://github.com/parmtech/Tong-Panel/commit/1ee83b177bba0beaf3ba33fec0577ae25724439c) | 3/3 success |
| [#37](https://github.com/parmtech/Tong-Panel/pull/37) | [`bde039e`](https://github.com/parmtech/Tong-Panel/commit/bde039eff7339f8916201752c18a892761cd18e5) | 1/1 success |
| [#38](https://github.com/parmtech/Tong-Panel/pull/38) | [`ffbcabb`](https://github.com/parmtech/Tong-Panel/commit/ffbcabb5cc071e1c98063ef9883a8773df245f9d) | 3/3 success |
| [#39](https://github.com/parmtech/Tong-Panel/pull/39) | [`287dfdf`](https://github.com/parmtech/Tong-Panel/commit/287dfdfa7794036693d8c482635b62626fe1f77e) | 3/3 success |
| [#40](https://github.com/parmtech/Tong-Panel/pull/40) | [`9dc8a0c`](https://github.com/parmtech/Tong-Panel/commit/9dc8a0ccc5f8354c2debe9db4d93d294e7b5e0e5) | 3/3 success |
| [#41](https://github.com/parmtech/Tong-Panel/pull/41) | [`f1b6a91`](https://github.com/parmtech/Tong-Panel/commit/f1b6a915e7dea9039add1a49e8ed1c29952e7780) | 3/3 success |
| [#42](https://github.com/parmtech/Tong-Panel/pull/42) | [`78fa11b`](https://github.com/parmtech/Tong-Panel/commit/78fa11b655051c23eed1ed8c68a1e7b604cc590d) | 3/3 success |
| [#43](https://github.com/parmtech/Tong-Panel/pull/43) | [`721aae9`](https://github.com/parmtech/Tong-Panel/commit/721aae92c1a260e3a9708c119a261a8190f742a4) | None observed |
| [#44](https://github.com/parmtech/Tong-Panel/pull/44) | [`90f6d6e`](https://github.com/parmtech/Tong-Panel/commit/90f6d6e1fdaae19100dfc9fc5fb9dfdcb021cef7) | 1/1 success |
| [#45](https://github.com/parmtech/Tong-Panel/pull/45) | [`f51780e`](https://github.com/parmtech/Tong-Panel/commit/f51780e6d6be9ea7f4fec9639920c016d0a43477) | 1/1 success |
| [#46](https://github.com/parmtech/Tong-Panel/pull/46) | [`a17419f`](https://github.com/parmtech/Tong-Panel/commit/a17419f5d32289e28e48d7b61361d8bd0776917a) | 4/4 success |
| [#47](https://github.com/parmtech/Tong-Panel/pull/47) | [`db8a2d9`](https://github.com/parmtech/Tong-Panel/commit/db8a2d97f46f61faed90e4246465a410a27f8c80) | 4/4 success |
| [#48](https://github.com/parmtech/Tong-Panel/pull/48) | [`7606127`](https://github.com/parmtech/Tong-Panel/commit/7606127f416b59f65bc84ecf1ca21e2bb54e3894) | 4/4 success |
| [#49](https://github.com/parmtech/Tong-Panel/pull/49) | [`a41b83d`](https://github.com/parmtech/Tong-Panel/commit/a41b83d9083a37796251509da3b867c3d714dbf9) | 4/4 success |
| [#50](https://github.com/parmtech/Tong-Panel/pull/50) | [`89a8da6`](https://github.com/parmtech/Tong-Panel/commit/89a8da6224e6f30e9f59bd63e9e7f3d5c8241fa7) | 3/3 success |
| [#51](https://github.com/parmtech/Tong-Panel/pull/51) | [`437795d`](https://github.com/parmtech/Tong-Panel/commit/437795d4533e9565b58763f7726516736179f3ee) | 3/3 success |
| [#52](https://github.com/parmtech/Tong-Panel/pull/52) | [`c5cbe42`](https://github.com/parmtech/Tong-Panel/commit/c5cbe427d2745a9fe247be451b2637ab5a2c91d3) | 4/4 success |
| [#53](https://github.com/parmtech/Tong-Panel/pull/53) | [`2fba56f`](https://github.com/parmtech/Tong-Panel/commit/2fba56f00df137e896892851ec4811b78abb95fd) | 5/5 success |
| [#54](https://github.com/parmtech/Tong-Panel/pull/54) | [`8b72449`](https://github.com/parmtech/Tong-Panel/commit/8b7244934753313461a3ca39ead26c48c43bd43b) | 4/4 success |
| [#55](https://github.com/parmtech/Tong-Panel/pull/55) | [`993cc56`](https://github.com/parmtech/Tong-Panel/commit/993cc56e119f6a1b2c357a5e8b07d3ac5c7e6975) | 4/4 success |
| [#56](https://github.com/parmtech/Tong-Panel/pull/56) | [`957a780`](https://github.com/parmtech/Tong-Panel/commit/957a780ecdfa5c3daf900498f1e8362358e75a56) | 4/4 success |
| [#57](https://github.com/parmtech/Tong-Panel/pull/57) | [`5d0f318`](https://github.com/parmtech/Tong-Panel/commit/5d0f3180232fccdc1e84f4c4b303d1989d12ebe6) | 3/4 success; [failure](https://github.com/parmtech/Tong-Panel/actions/runs/40722043238) |
| [#58](https://github.com/parmtech/Tong-Panel/pull/58) | [`0a520c9`](https://github.com/parmtech/Tong-Panel/commit/0a520c9f4e69bb6a4cb35107b24f9ca9bf607643) | 5/5 success |
| [#59](https://github.com/parmtech/Tong-Panel/pull/59) | [`b0aadff`](https://github.com/parmtech/Tong-Panel/commit/b0aadfff9d1128865d67b65042617b90ec295461) | 4/5 success; [failure](https://github.com/parmtech/Tong-Panel/actions/runs/40725113404) |
| [#60](https://github.com/parmtech/Tong-Panel/pull/60) | [`098a1bb`](https://github.com/parmtech/Tong-Panel/commit/098a1bbf38108e3c0ea03e0ffac0536674b753c7) | 4/4 success |
| [#61](https://github.com/parmtech/Tong-Panel/pull/61) | [`d763820`](https://github.com/parmtech/Tong-Panel/commit/d76382022bdb559d1a93915dbf51952aa42ee907) | 4/4 success |
| [#62](https://github.com/parmtech/Tong-Panel/pull/62) | [`8fa5352`](https://github.com/parmtech/Tong-Panel/commit/8fa5352144a15f2ff72cb82913b285e830cd9827) | 4/4 success |
| [#63](https://github.com/parmtech/Tong-Panel/pull/63) | [`7bb3719`](https://github.com/parmtech/Tong-Panel/commit/7bb3719485a5451d14e279ec7e42349c3cd5c723) | 5/5 success |
| [#64](https://github.com/parmtech/Tong-Panel/pull/64) | [`b16c930`](https://github.com/parmtech/Tong-Panel/commit/b16c9300ea681885a7af5fa397cfa34428f60688) | 1/1 success |
| [#65](https://github.com/parmtech/Tong-Panel/pull/65) | [`46d8ecb`](https://github.com/parmtech/Tong-Panel/commit/46d8ecb347819dba221a868cb8b2a11d723f1d81) | 4/4 success |
| [#66](https://github.com/parmtech/Tong-Panel/pull/66) | [`47f8533`](https://github.com/parmtech/Tong-Panel/commit/47f85333d05887c60a90c7dc2e948f1eaebbf1ff) | 4/4 success |
| [#67](https://github.com/parmtech/Tong-Panel/pull/67) | [`0fc48cd`](https://github.com/parmtech/Tong-Panel/commit/0fc48cdb824ab6b57b84020ff07ee39fe6676d0e) | 4/4 success |
| [#68](https://github.com/parmtech/Tong-Panel/pull/68) | [`80d7aa5`](https://github.com/parmtech/Tong-Panel/commit/80d7aa532d69737a4ff8a26b802c5ae21a66f8e0) | 4/4 success |
| [#69](https://github.com/parmtech/Tong-Panel/pull/69) | [`e2c6a87`](https://github.com/parmtech/Tong-Panel/commit/e2c6a8718d11fb47412667c368b29bb9c8085201) | 4/4 success |
| [#70](https://github.com/parmtech/Tong-Panel/pull/70) | [`0ac1c0e`](https://github.com/parmtech/Tong-Panel/commit/0ac1c0e5abad1af991ba1dc05e54faedb14ba055) | 4/4 success |
| [#71](https://github.com/parmtech/Tong-Panel/pull/71) | [`cccfec5`](https://github.com/parmtech/Tong-Panel/commit/cccfec5c7b2876487076233796865d1e65b9e903) | 4/4 success |
| [#72](https://github.com/parmtech/Tong-Panel/pull/72) | [`cf7ce80`](https://github.com/parmtech/Tong-Panel/commit/cf7ce80ac3a01aee0ddc467152cb427db638eda4) | 5/5 success |
| [#73](https://github.com/parmtech/Tong-Panel/pull/73) | [`b41b0bf`](https://github.com/parmtech/Tong-Panel/commit/b41b0bf09f9dac26e4c6920fe41ef875334a7028) | 5/5 success |
| [#74](https://github.com/parmtech/Tong-Panel/pull/74) | [`356a4b8`](https://github.com/parmtech/Tong-Panel/commit/356a4b8644b2535139b258e3790db85ba70a28a0) | 1/1 success |
| [#75](https://github.com/parmtech/Tong-Panel/pull/75) | [`55ae0b9`](https://github.com/parmtech/Tong-Panel/commit/55ae0b90e0db076df07585b6f7603ff6a4553157) | 4/4 success |
| [#76](https://github.com/parmtech/Tong-Panel/pull/76) | [`77da443`](https://github.com/parmtech/Tong-Panel/commit/77da44392f94c0f578149516ad536f33b4085aa6) | None observed |
| [#77](https://github.com/parmtech/Tong-Panel/pull/77) | [`b62e2f0`](https://github.com/parmtech/Tong-Panel/commit/b62e2f09f212f3a2ab3cecbf6b973682c29906c9) | 4/4 success |
| [#78](https://github.com/parmtech/Tong-Panel/pull/78) | [`06d958c`](https://github.com/parmtech/Tong-Panel/commit/06d958ca9834d0d6671aa286de3f4a9da0b65d6b) | 4/4 success |
| [#79](https://github.com/parmtech/Tong-Panel/pull/79) | [`f9889c9`](https://github.com/parmtech/Tong-Panel/commit/f9889c978855c41d4ad10e7ad73625e5729e5a61) | 1/1 success |
| [#80](https://github.com/parmtech/Tong-Panel/pull/80) | [`22578a1`](https://github.com/parmtech/Tong-Panel/commit/22578a13da3980c79c6198e089a9b66b252a5cc4) | 4/4 success |
| [#81](https://github.com/parmtech/Tong-Panel/pull/81) | [`d947e53`](https://github.com/parmtech/Tong-Panel/commit/d947e53c7af2e97b160df3c69d9392b0dc1bf106) | 3/3 success |
| [#82](https://github.com/parmtech/Tong-Panel/pull/82) | [`0574a97`](https://github.com/parmtech/Tong-Panel/commit/0574a97521a45afb7faca1d1caf32dbfbe77d54c) | 3/3 success |
| [#83](https://github.com/parmtech/Tong-Panel/pull/83) | [`31fac34`](https://github.com/parmtech/Tong-Panel/commit/31fac34d427139567e006ccbd7fa575b0fea48e3) | 3/3 success |
| [#84](https://github.com/parmtech/Tong-Panel/pull/84) | [`955a610`](https://github.com/parmtech/Tong-Panel/commit/955a61098ea4a40b95bbc4e521f1a4c96542168c) | 4/4 success |
| [#85](https://github.com/parmtech/Tong-Panel/pull/85) | [`5ff4244`](https://github.com/parmtech/Tong-Panel/commit/5ff42442a15b9f1d4f0bd069cef23524cceeba31) | 4/4 success |
| [#86](https://github.com/parmtech/Tong-Panel/pull/86) | [`1af977d`](https://github.com/parmtech/Tong-Panel/commit/1af977d8d9f809f0d0d997ee81f998edebc48979) | 1/1 success |
| [#87](https://github.com/parmtech/Tong-Panel/pull/87) | [`dbc624b`](https://github.com/parmtech/Tong-Panel/commit/dbc624bd247bcf636ea28ec7d354918aaa5cb1aa) | 4/4 success |
| [#88](https://github.com/parmtech/Tong-Panel/pull/88) | [`35158c6`](https://github.com/parmtech/Tong-Panel/commit/35158c6fb1e76967ad24b9e2d9663a5652d078e2) | 3/4 success; [failure](https://github.com/parmtech/Tong-Panel/actions/runs/42631436094) |
| [#89](https://github.com/parmtech/Tong-Panel/pull/89) | [`51e12ab`](https://github.com/parmtech/Tong-Panel/commit/51e12abb99d919e5e8cb10a69e868dc65c4ed9a0) | 1/1 success |
| [#90](https://github.com/parmtech/Tong-Panel/pull/90) | [`a4e8c73`](https://github.com/parmtech/Tong-Panel/commit/a4e8c73fa394ab727cbf20fc27b219ea46820d14) | 5/5 success |
| [#91](https://github.com/parmtech/Tong-Panel/pull/91) | [`01ae223`](https://github.com/parmtech/Tong-Panel/commit/01ae2237b8bdb4240fb79ca45842f9e6d4f549a1) | 4/4 success |
| [#92](https://github.com/parmtech/Tong-Panel/pull/92) | [`87b0372`](https://github.com/parmtech/Tong-Panel/commit/87b0372e83db2308faebbe3e904c11e59438d660) | 3/3 success |
| [#93](https://github.com/parmtech/Tong-Panel/pull/93) | [`1832452`](https://github.com/parmtech/Tong-Panel/commit/18324524bb08135232ac7f1c952f66d0b8d95985) | 1/1 success |
| [#94](https://github.com/parmtech/Tong-Panel/pull/94) | [`0bd79b9`](https://github.com/parmtech/Tong-Panel/commit/0bd79b96fec86befbd576cb3375960047f8c3120) | 1/1 success |
| [#95](https://github.com/parmtech/Tong-Panel/pull/95) | [`197ce18`](https://github.com/parmtech/Tong-Panel/commit/197ce185393e1b635125d3491e0c887a9b7da406) | 5/5 success |
| [#96](https://github.com/parmtech/Tong-Panel/pull/96) | [`a1c0751`](https://github.com/parmtech/Tong-Panel/commit/a1c075102a949b10b8c9a1456bb91c1ebd47c2ae) | 4/5 success; [failure](https://github.com/parmtech/Tong-Panel/actions/runs/42722907394) |
| [#97](https://github.com/parmtech/Tong-Panel/pull/97) | [`a3f3f7f`](https://github.com/parmtech/Tong-Panel/commit/a3f3f7f2c2612e494e6f922208e78b1c3479bb06) | 5/5 success |
| [#98](https://github.com/parmtech/Tong-Panel/pull/98) | [`0345de6`](https://github.com/parmtech/Tong-Panel/commit/0345de6b7081192cd65156a42a0dfc3c30967277) | 1/1 success |
| [#99](https://github.com/parmtech/Tong-Panel/pull/99) | [`c330f78`](https://github.com/parmtech/Tong-Panel/commit/c330f787ff865eaee5fbb496f75f2368a6f28b72) | 6/6 success |
| [#100](https://github.com/parmtech/Tong-Panel/pull/100) | [`483e709`](https://github.com/parmtech/Tong-Panel/commit/483e709fb049d3c43cb61ddee218550964ee3752) | 6/6 success |
| [#101](https://github.com/parmtech/Tong-Panel/pull/101) | [`8d89ff7`](https://github.com/parmtech/Tong-Panel/commit/8d89ff76fdbf610065f623a38b0cd833bd1cece2) | 6/6 success |
| [#102](https://github.com/parmtech/Tong-Panel/pull/102) | [`8481713`](https://github.com/parmtech/Tong-Panel/commit/84817135da81e67bee25dfc3a40e87394f46165b) | 5/5 success |
| [#103](https://github.com/parmtech/Tong-Panel/pull/103) | [`f481bea`](https://github.com/parmtech/Tong-Panel/commit/f481beafc3e5f8a282a3ca4b21cb88e1418eeed5) | 5/5 success |
| [#104](https://github.com/parmtech/Tong-Panel/pull/104) | [`b9d2594`](https://github.com/parmtech/Tong-Panel/commit/b9d2594e1f85d4baa34c3cf74730c3d41ad0e948) | 5/5 success |
| [#105](https://github.com/parmtech/Tong-Panel/pull/105) | [`e3b471d`](https://github.com/parmtech/Tong-Panel/commit/e3b471dcb28edb20796f21a9a8dc1ebc32fe056d) | 5/5 success |
| [#106](https://github.com/parmtech/Tong-Panel/pull/106) | [`9768f14`](https://github.com/parmtech/Tong-Panel/commit/9768f1453eed771f2cc662bdc67eb61488a7555e) | 5/5 success |
| [#107](https://github.com/parmtech/Tong-Panel/pull/107) | [`fbfa3ab`](https://github.com/parmtech/Tong-Panel/commit/fbfa3abd66b727e3cfac6004b6ee49d12f22eaaf) | 5/5 success |
| [#108](https://github.com/parmtech/Tong-Panel/pull/108) | [`d9f3912`](https://github.com/parmtech/Tong-Panel/commit/d9f39121129230db40b9dc52886ca524b357562c) | 7/8 success; [cancelled](https://github.com/parmtech/Tong-Panel/actions/runs/42740006289) |
| [#109](https://github.com/parmtech/Tong-Panel/pull/109) | [`8ac16f2`](https://github.com/parmtech/Tong-Panel/commit/8ac16f2db6dd9e94e9b2f45704f1c5f7f3980abb) | 9/9 success |
| [#110](https://github.com/parmtech/Tong-Panel/pull/110) | [`1171741`](https://github.com/parmtech/Tong-Panel/commit/1171741e91102192c8ca62c7689f7be5acb6f9fb) | 8/8 success |
| [#111](https://github.com/parmtech/Tong-Panel/pull/111) | [`8a76fcd`](https://github.com/parmtech/Tong-Panel/commit/8a76fcd329c863083e5fb38b0682af43d3f3ec07) | 7/7 success |
| [#112](https://github.com/parmtech/Tong-Panel/pull/112) | [`115f05e`](https://github.com/parmtech/Tong-Panel/commit/115f05ea598fe7637bab2383989e1f5eb3d4f618) | 7/7 success |
| [#113](https://github.com/parmtech/Tong-Panel/pull/113) | [`183ef94`](https://github.com/parmtech/Tong-Panel/commit/183ef949b230f918992d7cf6bb07383ed2023c96) | 7/7 success |
| [#114](https://github.com/parmtech/Tong-Panel/pull/114) | [`ea84cd1`](https://github.com/parmtech/Tong-Panel/commit/ea84cd12423c6b557894f2c442960ce16f9d3df1) | 7/7 success |
| [#115](https://github.com/parmtech/Tong-Panel/pull/115) | [`8c946df`](https://github.com/parmtech/Tong-Panel/commit/8c946df9cc54d3da0e90ee455faa707eeb5579f2) | 7/7 success |
| [#116](https://github.com/parmtech/Tong-Panel/pull/116) | [`6085c0e`](https://github.com/parmtech/Tong-Panel/commit/6085c0e48ef03f2f0b12332a1039db5eb8de5935) | 7/7 success |
| [#117](https://github.com/parmtech/Tong-Panel/pull/117) | [`b33f16c`](https://github.com/parmtech/Tong-Panel/commit/b33f16c2a1ad3cfab22a43765068e6d3a3d7cf1f) | 4/4 success |
| [#118](https://github.com/parmtech/Tong-Panel/pull/118) | [`2f23d56`](https://github.com/parmtech/Tong-Panel/commit/2f23d568a94ed81c0343aee9a1e2b62cf3896bdf) | 6/6 success |
| [#119](https://github.com/parmtech/Tong-Panel/pull/119) | [`b5bd122`](https://github.com/parmtech/Tong-Panel/commit/b5bd1221fdfd0ea04abb8b80fdcdcb3d6d19810b) | 6/6 success |
| [#120](https://github.com/parmtech/Tong-Panel/pull/120) | [`bb36f78`](https://github.com/parmtech/Tong-Panel/commit/bb36f78eece73870594534c7a789463f0f74f555) | 6/6 success |
| [#121](https://github.com/parmtech/Tong-Panel/pull/121) | [`d522844`](https://github.com/parmtech/Tong-Panel/commit/d522844b03edf8c0beee7fc22e5d1fe68c186b63) | 6/6 success |
| [#122](https://github.com/parmtech/Tong-Panel/pull/122) | [`67e2ffc`](https://github.com/parmtech/Tong-Panel/commit/67e2ffc6e5bd502e6f1a1a8b2d72fcee3db81fb6) | 6/6 success |
| [#123](https://github.com/parmtech/Tong-Panel/pull/123) | [`f4bc0b6`](https://github.com/parmtech/Tong-Panel/commit/f4bc0b6a6686aeb790cbb1c9a207d8890a0996f2) | 6/6 success |
| [#124](https://github.com/parmtech/Tong-Panel/pull/124) | [`f656615`](https://github.com/parmtech/Tong-Panel/commit/f656615cd651ea6a5ed16c4652e9c73f1934fee7) | 6/6 success |
| [#125](https://github.com/parmtech/Tong-Panel/pull/125) | [`4fe444c`](https://github.com/parmtech/Tong-Panel/commit/4fe444c8f352724c5ff1cf26c169861c5cf98eab) | 6/6 success |
| [#126](https://github.com/parmtech/Tong-Panel/pull/126) | [`12fa535`](https://github.com/parmtech/Tong-Panel/commit/12fa53578dba65051e2fc7f9cf0bb2b881d5f889) | 6/6 success |
| [#127](https://github.com/parmtech/Tong-Panel/pull/127) | [`4089e59`](https://github.com/parmtech/Tong-Panel/commit/4089e5954583cddc925fad0909cd419e607fcec4) | 6/6 success |
| [#128](https://github.com/parmtech/Tong-Panel/pull/128) | [`83c8b68`](https://github.com/parmtech/Tong-Panel/commit/83c8b684a17961591c8094e38b9a1603218d56d5) | 6/6 success |
| [#129](https://github.com/parmtech/Tong-Panel/pull/129) | [`51ca6e1`](https://github.com/parmtech/Tong-Panel/commit/51ca6e12de9de165bd20d9fb08e8a86a96db7b30) | 7/7 success |
| [#130](https://github.com/parmtech/Tong-Panel/pull/130) | [`94a3ee7`](https://github.com/parmtech/Tong-Panel/commit/94a3ee724a5cd2c22503e81aaa21a8079afb6ee9) | 7/7 success |
| [#131](https://github.com/parmtech/Tong-Panel/pull/131) | [`b06056e`](https://github.com/parmtech/Tong-Panel/commit/b06056e2a67a49aadd6f2060e31965a189113c44) | 7/7 success |
| [#132](https://github.com/parmtech/Tong-Panel/pull/132) | [`a2fc5b1`](https://github.com/parmtech/Tong-Panel/commit/a2fc5b1ed5b912e781f71d18436db3ac02aa421e) | 7/7 success |
| [#133](https://github.com/parmtech/Tong-Panel/pull/133) | [`fbde7e8`](https://github.com/parmtech/Tong-Panel/commit/fbde7e81e2f4b27f12a9f76ce9a0ad70b7b56380) | 7/7 success |
| [#134](https://github.com/parmtech/Tong-Panel/pull/134) | [`9c5c24a`](https://github.com/parmtech/Tong-Panel/commit/9c5c24a89dd788e6614ebc70d6b684a63bca869e) | 6/6 success |
| [#135](https://github.com/parmtech/Tong-Panel/pull/135) | [`1985aae`](https://github.com/parmtech/Tong-Panel/commit/1985aae610f0ff12819193c96bac540fb1919585) | 6/6 success |
| [#136](https://github.com/parmtech/Tong-Panel/pull/136) | [`2096686`](https://github.com/parmtech/Tong-Panel/commit/20966869b1cb459dffc8cc9bf122df01bbbe288c) | 6/6 success |
| [#137](https://github.com/parmtech/Tong-Panel/pull/137) | [`3b34343`](https://github.com/parmtech/Tong-Panel/commit/3b34343c208ff6ae0e3de8e2c162781ed44eae39) | 6/6 success |
| [#138](https://github.com/parmtech/Tong-Panel/pull/138) | [`feede09`](https://github.com/parmtech/Tong-Panel/commit/feede09a7d4fbe5a5f8a3fe975144c7aa8e7ef99) | 6/6 success |
| [#139](https://github.com/parmtech/Tong-Panel/pull/139) | [`f21b526`](https://github.com/parmtech/Tong-Panel/commit/f21b526bf40a8c79d25d65cb25aea75664e6cb12) | 6/6 success |
| [#140](https://github.com/parmtech/Tong-Panel/pull/140) | [`6710fa9`](https://github.com/parmtech/Tong-Panel/commit/6710fa909d8e48f51bb64aa509057cd8f7b6c7b3) | 6/6 success |
| [#141](https://github.com/parmtech/Tong-Panel/pull/141) | [`2accb20`](https://github.com/parmtech/Tong-Panel/commit/2accb20ff722757e87ec8bad89e6d9ca31873b4a) | 6/6 success |
| [#142](https://github.com/parmtech/Tong-Panel/pull/142) | [`a06765d`](https://github.com/parmtech/Tong-Panel/commit/a06765d82f842c2557d69181794e60b72ed11902) | 6/6 success |
| [#143](https://github.com/parmtech/Tong-Panel/pull/143) | [`4820170`](https://github.com/parmtech/Tong-Panel/commit/482017005f9872c1fe7f2969ebfb477ea7de30dd) | 7/7 success |
| [#144](https://github.com/parmtech/Tong-Panel/pull/144) | [`2cf9211`](https://github.com/parmtech/Tong-Panel/commit/2cf9211d592c7233dff01ab1e059aa260752af76) | None observed |
| [#145](https://github.com/parmtech/Tong-Panel/pull/145) | [`24304bb`](https://github.com/parmtech/Tong-Panel/commit/24304bbba3d6d8882a0fd1d70aa9d13e751100c5) | None observed |
| [#146](https://github.com/parmtech/Tong-Panel/pull/146) | [`1b15276`](https://github.com/parmtech/Tong-Panel/commit/1b15276ef98b519335ba2566de04b6d862cb8277) | None observed |
| [#147](https://github.com/parmtech/Tong-Panel/pull/147) | [`7cf24a6`](https://github.com/parmtech/Tong-Panel/commit/7cf24a675490a6a07333a873d0ebe12dd00bd375) | None observed |
| [#149](https://github.com/parmtech/Tong-Panel/pull/149) | [`4dab4c6`](https://github.com/parmtech/Tong-Panel/commit/4dab4c60cc86186935e9a7d987b9fe1ca97a1f23) | 7/7 success |
| [#158](https://github.com/parmtech/Tong-Panel/pull/158) | [`8b05989`](https://github.com/parmtech/Tong-Panel/commit/8b05989bf94ef596c33142bade980f97565b7b4f) | None observed |
| [#159](https://github.com/parmtech/Tong-Panel/pull/159) | [`17241e2`](https://github.com/parmtech/Tong-Panel/commit/17241e2305c379ddaeba2db56d40a43fe76ba9a0) | None observed |
| [#160](https://github.com/parmtech/Tong-Panel/pull/160) | [`4a71768`](https://github.com/parmtech/Tong-Panel/commit/4a71768256a38d6c052461b6cd8871298e6f3435) | None observed |
| [#161](https://github.com/parmtech/Tong-Panel/pull/161) | [`0422e27`](https://github.com/parmtech/Tong-Panel/commit/0422e27612defeee5c72c70950b9f4f8da2d97ee) | None observed |
| [#162](https://github.com/parmtech/Tong-Panel/pull/162) | [`4934cfa`](https://github.com/parmtech/Tong-Panel/commit/4934cfa8b0be7aaa775f77d6211fad3b6303d3a0) | None observed |
| [#163](https://github.com/parmtech/Tong-Panel/pull/163) | [`b8a2cab`](https://github.com/parmtech/Tong-Panel/commit/b8a2cab2dee5a1df299186b5d6ac4913ff898b4e) | None observed |
| [#164](https://github.com/parmtech/Tong-Panel/pull/164) | [`149d214`](https://github.com/parmtech/Tong-Panel/commit/149d214155390bf8f991b6503cd214b3fabf9198) | None observed |
| [#165](https://github.com/parmtech/Tong-Panel/pull/165) | [`439f2e4`](https://github.com/parmtech/Tong-Panel/commit/439f2e4abf9e1c9d0caa9aec63c0710451910378) | None observed |
| [#166](https://github.com/parmtech/Tong-Panel/pull/166) | [`001f801`](https://github.com/parmtech/Tong-Panel/commit/001f801c47a5cd251486c2939b783b72946257de) | 9/9 success |
| [#167](https://github.com/parmtech/Tong-Panel/pull/167) | [`188851d`](https://github.com/parmtech/Tong-Panel/commit/188851da58480f13b486b43b9d6e29ea3fd7c968) | 6/6 success |
| [#168](https://github.com/parmtech/Tong-Panel/pull/168) | [`d14e561`](https://github.com/parmtech/Tong-Panel/commit/d14e5611d7b3d0302d4ae76bdbb8f210827e1bb9) | 10/10 success |
| [#169](https://github.com/parmtech/Tong-Panel/pull/169) | [`1351db4`](https://github.com/parmtech/Tong-Panel/commit/1351db454a54408af8d2ffd928e0e1d1a6676ec8) | 7/7 success |
| [#170](https://github.com/parmtech/Tong-Panel/pull/170) | [`1dd0ae3`](https://github.com/parmtech/Tong-Panel/commit/1dd0ae3cead20a2d042c7a491a2d825f93fda12a) | 8/8 success |
