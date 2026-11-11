# P0-06 formal threat model proposal ready for review

- Event ID: parallel-threat-20261111T035848Z-2ed4f2ec-9205-4dcb-aafe-3e75bb3ff53a
- UTC time: 2026-11-11T03:58:48Z
- Writer/phase: Parallel threat reproducibility / P0 supplemental
- Kind: status
- Recipients: coordinator, maintainer/security reviewer

## Read receipt and source basis

Re-read current coordination protocol and roster on docs/phase-coordination: protocol blob 64b66594fce7532dd059c5967466d626112c55af; roster blob d60ecc69b4ba1a0a992ab67a1f9942a38b7bc1d8. Read own prior events:
- adoption parallel-threat-20261111T035006Z-a5b71c43-0a19-49ba-b2b1-68bb914133ca;
- pointer/source correction parallel-threat-20261111T035156Z-6dce84b0-49ba-45fa-b5f8-c03d98cf9ccd;
- case-contract claim parallel-threat-20261111T035400Z-9714dd2c-183d-49aa-994c-2ca42b38c827.

Canonical P0 plan source docs/phases/P0-PLANNING.md at 0aad68e81138a0f226e83e401f90bae4d0b6bddc, blob 55fe6830ff0345b374bb08dd06f4ce95f4921bd0, lists P0-06 as open and requires formal assets/STRIDE/mitigations/test cases, plus maintainer approval of privileged agent boundary. Related ADR, architecture, security, threat and test strategy sources were read at the same baseline; the local model records exact blob SHAs.

## Deliverables

- Local proposed formal threat model: proposed docs/p0-review/parallel-threat/proposed-formal-threat-model.md; SHA-256 3643DFF1D27EEEA64B1AA79DDC4278CCF6B9C16624CDA40DD68B2B12F62ECD7B.
- Local source-pinned reproducibility matrix: threat-repro-evidence/threat-test-reproducibility.md; SHA-256 B5B1B1C3F1BA97D9F91CD8CCE1160C39ACD9B64C5D97EBBC0CD7E30289A76BD4.
- The model consolidates existing T01–T15 under STRIDE, enumerates assets/boundaries, states proposed controls/test cases and exclusions, and links exact visible test names to executable commits. It corrects two misleading draft pointers (cabef25… is SQLite queue contention, not agent decoder fuzz; 6c983a… is formatting-only, with cookie/origin tests in a different executable commit).
- Proposed case IDs cover T02/T03/T04/T15 with deterministic setup, action, expected denial/no-effect oracle, positive control, evidence and cleanup. These remain proposed and unexecuted.

## Maintainer decision requested; no approval inferred

Please relay/record a maintainer decision for ADR-007 / issue #1 on the local privileged boundary before accepting any privileged verb: dedicated unprivileged API UID and allowed agent socket peer UID/GID; socket owner/group/mode and parent directory policy; Linux peer-credential mechanism and fail-closed behavior on supported lab profiles; binding of principal/tenant/resource/job/expiry/nonce through durable queue to agent; and first allowed typed operation (if any) plus independent reviewer. The draft recommends no privileged verb until this is decided and security-reviewed. This is a request, not an approval.

## Tests, gates and blocker

No tests were run. The workspace has no checkout/build environment; open PR patches and CI metadata establish source/test names and run summaries only. Linux guest and smoke scripts are reserved for intended disposable CI VMs and were not invoked here. No production/privileged action or code edit occurred.

P0-06 and P0 overall remain open. This stream does not approve the formal model or self-advance phases. Available GitHub file fetches do not enumerate the branch's event directories, so only the known protocol/roster and own prior event IDs above were read; no sibling acknowledgment is implied. Coordinator should provide any relevant event IDs/questions or use a listing-capable read path for the next checkpoint.

## Next action

Coordinator/maintainer reviews the proposal, answers the ADR-007 boundary question, and records accepted/revise/reject outcome in canonical P0/ADR sources. Until then, keep the model proposed and preserve local outputs.
