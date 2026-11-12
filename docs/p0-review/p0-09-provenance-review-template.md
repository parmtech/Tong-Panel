> **Preservation draft:** historical working evidence; factual statements and links require review. No approval or phase completion is implied. Machine-local paths and private coordination details have been omitted.

# P0-09 qualified provenance, licence, and trademark review record

> **Blank review template.** This record is for a specifically identified source and distribution snapshot. Complete it only with evidence inspected by the named reviewers. Blank fields mean no disposition is recorded. “Reviewer to determine” is not a finding. This template is not legal advice, licence clearance, trademark clearance, security signoff, or distribution approval; it does not authorize publication, packaging, or release. “Publication” here refers to a reviewed product artifact/channel; this form does not change the repository's current public visibility or the existing AGPL-3.0 declaration for original Tong Panel code. Any change to repository visibility or that license declaration requires separate explicit owner authorization.

## 1. Review identity and authority

| Field | Record |
| --- | --- |
| Review record ID | _Reviewer to determine_ |
| Review date (UTC) | _Reviewer to determine_ |
| Project decision owner / maintainer | _Reviewer to determine_ |
| Decision owner's authority and scope | _Reviewer to determine_ |
| Qualified licence reviewer name and qualification | _Reviewer to determine_ |
| Licence reviewer's independence / conflicts | _Reviewer to determine_ |
| Qualified trademark reviewer name and qualification | _Reviewer to determine_ |
| Trademark reviewer's independence / conflicts | _Reviewer to determine_ |
| Counsel referral needed? (yes / no / reviewer to determine) | _Reviewer to determine_ |
| Counsel referral owner/status, if applicable | _Reviewer to determine_ |
| Review status (in progress / complete / deferred) | _Reviewer to determine_ |
| Maintainer decision (accept / reject / needs revision / deferred) | _Reviewer to determine_ |

Record qualifications and authority sufficient for this exact scope. If the reviewer cannot assess an item within their qualifications, mark it out of scope and identify the required qualified reviewer or counsel referral; do not infer clearance.

## 2. Frozen source and intended distribution scope

| Field | Record |
| --- | --- |
| Repository and canonical URL | _Reviewer to determine_ |
| Branch/tag and exact source commit SHA | _Reviewer to determine_ |
| Source tree/archive SHA-256 and retrieval date | _Reviewer to determine_ |
| Prior P0-09 record and exact revision/hash, if this review changes or supersedes an earlier scope | _Reviewer to determine_ |
| Scope delta since the prior record (added, removed, or changed source/components/artifacts) and affected evidence | _Reviewer to determine_ |
| Scope-change re-review triggers and required reviewer/maintainer follow-up (explicitly include any change to P0-07 Decision 0/2/3; ADR-001 original/fork/hybrid route, selected upstream source or assets; ADR-006 profile/service choices; ADR-008 artifact classes, channel, signing/bootstrap plan; or any inclusion mode, including runtime-fetched, vendored, generated/embedded payloads, or external services) | _Reviewer to determine_ |
| Dirty/untracked source state and included patches | _Reviewer to determine_ |
| Build recipe, toolchain, and exact build revision | _Reviewer to determine_ |
| Intended audience, delivery channel, and use | _Reviewer to determine_ |
| Exact distribution/package names and versions | _Reviewer to determine_ |
| Target OS/architecture/runtime profiles | _Reviewer to determine_ |
| Included services, installers, containers, images, docs, and examples | _Reviewer to determine_ |
| Excluded features, components, and assets | _Reviewer to determine_ |
| Scope limitations / unsupported conclusions | _Reviewer to determine_ |
| Evidence bundle location, revision, and integrity hash | _Reviewer to determine_ |


### Distribution-class worksheet — preparation only; owner choices blank

For every independently selectable artifact, service, package, or profile, record one separate row; do not use a combined selection for items that may have different scope. Choices are: `include` = intentionally part of this exact frozen distribution; `exclude` = explicitly not distributed in this scope; `out of scope` = not assessed in this review, with the boundary and responsible review path named; `unknown` = intended inclusion or distribution is unresolved and cannot be treated as absence. These choices apply only to the exact reviewed artifact/channel and do not change repository visibility or the project's existing AGPL-3.0 declaration. These rows are questions, not claims. No class is presumed included or excluded. Freeze the completed scope before qualified review; unresolved items remain explicit.

| Artifact/component class — add one row per independent item | Owner selection (include / exclude / out of scope / unknown) | Intended artifact, channel, audience, profile | Exact frozen scope and evidence refs | Owner rationale / limits | Change trigger / re-review condition |
|---|---|---|---|---|---|
| Source tree/archive, including dirty/untracked paths | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (P0-09 P4-reported evidence annex, lines 77–83; §2 fields above) | _Owner to determine_ | _Owner to determine_ |
| Corresponding source or source-offer artifact/access path | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (P0-09 §3 “Source, generated code, and asset origin manifest”; §6 “Source offer, attribution, and notices”; exact source snapshot above) | _Owner to determine_ | _Owner to determine_ |
| Unsigned preview archive — one row per archive | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (P0-09 “Independent artifact evidence annex — PR #149 unsigned developer preview (no clearance)”) | _Owner to determine_ | _Owner to determine_ |
| Each binary contained in an unsigned preview archive — one row per binary and destination | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (P0-09 “Independent artifact evidence annex — PR #149 unsigned developer preview (no clearance)”) | _Owner to determine_ | _Owner to determine_ |
| Binary SBOM file(s), each distribution location | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (P0-09 independent artifact evidence annex and “SBOM-to-source-manifest crosswalk” subsection) | _Owner to determine_ | _Owner to determine_ |
| Debian package, per package name/version | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (`scripts/build_deb.sh` blob `f35310d899cba2d9f48d4f0d0f40dd29af7b78d8`; candidate-lab Debian fixture evidence: P0-07 PR #149 candidate-lab evidence row and P2 clean-VM workflow blob `65d270e349e5817aeab0721750d76e234168b9cd`, including `scripts/test_deb_package.sh`; separate unsigned preview comparison: P0-09 independent artifact evidence annex and `.github/workflows/p1-preview-package.yml` blob `8f4d53bda62f595f6d0e12a3d606e516d1b12970`) | _Owner to determine_ | _Owner to determine_ |
| Installer/bootstrap artifact — one row per artifact | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (reconciliation “Supplemental exact-artifact provenance checks”; P1 recipe blob `2ffccd06d17b3856f1af74e1af1a4d5aca8e1c03`; ADR-008; INSTALLER_SPEC) | _Owner to determine_ | _Owner to determine_ |
| Installer/bootstrap entrypoint or procedure — one row per entrypoint if separately distributed | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (ADR-008; INSTALLER_SPEC) | _Owner to determine_ | _Owner to determine_ |
| Signed release package/tag — one row per package | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (ADR-008; INSTALLER_SPEC; P0-09 independent artifact evidence annex identifies an unsigned developer preview, not a signed release) | _Owner to determine_ | _Owner to determine_ |
| Signature, attestation, or provenance file — link each to the exact package ID/digest | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (ADR-008; INSTALLER_SPEC) | _Owner to determine_ | _Owner to determine_ |
| License, NOTICE, attribution, and source-offer files/bundle, by exact destination | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (P0-09 P4-reported evidence annex; independent artifact evidence annex; official Go proxy archive evidence subsections; npm LICENSE/NOTICE archive evidence in the reconciliation note) | _Owner to determine_ | _Owner to determine_ |
| Clean-VM logs/checksums as provenance-review evidence | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (P0-07 PR #149 candidate-lab evidence row, run `42790473632`, jobs `129742295654` and `129742295031`; exact image inputs there) | _Owner to determine_ | _Owner to determine_ |
| Any VM-produced package/file offered as a product artifact | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (same run/job refs above; link to the same artifact ID/name/version/digest in the package rows if it is the same output; use a separate row only for a distinct artifact) | _Owner to determine_ | _Owner to determine_ |
| WordPress runtime-fetched archive — distinct source/archive row | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (P0-09 P4-reported evidence annex WordPress row; reconciliation “Supplemental exact-artifact provenance checks”; P1 recipe blob `2ffccd06d17b3856f1af74e1af1a4d5aca8e1c03`) | _Owner to determine_ | _Owner to determine_ |
| WordPress installed tenant payload — distinct deployed/runtime destination row | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (same runtime-fetch evidence above; exact destination remains for owner/reviewer determination) | _Owner to determine_ | _Owner to determine_ |
| OS package, runtime, or deployment dependency, one row per selected profile/component | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (P0-09 P4-reported evidence annex, line 78; P0-07 Decision 2; ADR-005; INSTALLER_SPEC) | _Owner to determine_ | _Owner to determine_ |
| UI source, generated/embedded build output, or asset, each as applicable | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (P0-09 P4-reported evidence annex; reconciliation “UI source dependency trace” and “UI tracked-file history follow-up — 2026-11-12 (source attribution only)”; preview archive inventory does not itself map UI source to embedded assets) | _Owner to determine_ | _Owner to determine_ |
| Developer/build/test tool or dependency, one row per toolchain/manifest as needed | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (P0-09 P4-reported evidence annex, line 78 (P4 source inventory refs 22–35); “Official Go proxy archives — root modules omitted from the preview SBOM”; “SBOM-to-source-manifest crosswalk”; “Nested OpenAPI-check tooling dependency archive evidence”; reconciliation “UI browser package source archive check” and its npm archive/SRI plus LICENSE/NOTICE evidence) | _Owner to determine_ | _Owner to determine_ |
| Service or provider integration, one row per service (including external DNS, managed DNS, mail) | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (P0-07 Decision 0 and ADR-006 service-scope worksheet; ADR-008; INSTALLER_SPEC; P0-09 P4-reported evidence annex, lines 77–81) | _Owner to determine_ | _Owner to determine_ |
| Container or VM image offered as a product artifact — one row per image | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (P0-09 §2 scope fields; P4-reported evidence annex, lines 77–81; P0-07 Decision 0 / ADR-006 as applicable) | _Owner to determine_ | _Owner to determine_ |
| Documentation, example, or install recipe — one row per independently selectable item | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (P0-09 §2 scope fields; P4-reported evidence annex, lines 77–81; P0-07 Decision 0 / ADR-006 as applicable) | _Owner to determine_ | _Owner to determine_ |
| App marketplace/catalog entry, signed or pinned manifest, plugin, or runtime-fetched tenant app payload — one row per independently selectable item, if applicable to frozen scope | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (BLUEPRINT lines 42–46 and 80; ADR-004 lines 26–30; INSTALLER_SPEC lines 23–25; scope applicability remains unselected) | _Owner to determine_ | _Owner to determine_ |
| CI guest image/input — distinguish test-only input from any image offered as a product artifact | _Owner to determine_ | _Owner to determine_ | _Owner to determine_ (P0-07 PR #149 candidate-lab evidence row; reconciliation “CI workflow input crosswalk”) | _Owner to determine_ | _Owner to determine_ |

For clean-VM evidence rows, choosing not to distribute logs does not remove their role as provenance evidence: preserve or cite the exact review logs/checksums where available. A product-distribution exclusion is not permission to omit evidence needed for the qualified review.
When two worksheet classes refer to the same package, archive/member, installer entrypoint, signature, recipe, or payload, reuse one item/artifact ID and one owner selection, then cross-reference the other row; do not count the same item twice. In particular, the generic app catalog/manifest/plugin/runtime-payload row covers only independently selectable items not already represented by a specific row such as the WordPress archive/payload rows. Keep a distinct row when the destination, channel, or scope genuinely differs.
Freeze the source and distribution scope before recording a disposition. A repository review alone does not establish what an assembled or distributed artifact contains.

## P4-reported evidence annex for reviewer verification — no clearance

Source packet: `docs/p0-review/p4-provenance-license-readiness.md` at PR #158 head `8b05989bf94ef596c33142bade980f97565b7b4f` (packet blob `20e1a2f3dccb154da9f5239e48ef896973d129bd`). Every item below is an observation or limitation reported by that P4 draft, not an independently accepted finding. Verify against the frozen scope and exact underlying material. Do not transfer these statements into a reviewer disposition without that verification.

| P4-reported evidence (exact packet references) | Scope/limitation stated by P4 | Reviewer verification or reconciliation to record |
|---|---|---|
| P4 based its code inventory on clean published P1 commit `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23`, on main base `0aad68e81138a0f226e83e401f90bae4d0b6bddc`; it excludes separately preserved P1 uncommitted paths. P4 scope names declared dependencies/notices, CI inputs, tracked source/assets, generated UI, and declared trademark constraints. (P4 lines 4–8, 20.) | This is a bounded source/declaration review, not all workspace content or a distribution-level review. | _Reviewer to determine_ |
| P4 reports pinned Go/npm dependency manifests, direct/indirect Go licence-file observations, npm development package versions/integrity, CI action/toolchain inputs, test/package path descriptions, no tracked complete third-party notice bundle or generated SBOM, and a later P1 draft with binary-only CycloneDX SBOMs for two Linux amd64 binaries. The described SBOM excludes full source, UI/assets, OS packages and deployment inventory and has no automated licence detection. (P4 lines 12–16, 22–35.) | These are declared manifests, bounded inventories and described workflows; P4 says it did not build or inspect generated artifacts. Packaging paths are proposed from source and are not approved releases. (P4 lines 63–73.) | _Reviewer to determine_ |
| P4 reports a runtime-fetched WordPress 7.1.4 archive URL and pinned SHA-256 in the P1 source; it says this tenant payload is outside the binary SBOM. The P4 upstream check says the archive itself was not downloaded/inspected and general WordPress licence information does not inventory archive notices. (P4 lines 31, 57–62, 83.) | The P4 statement is specific to that packet's review. The [reconciliation note's “Supplemental exact-artifact provenance checks” section](p0-phase0-reconciliation-2026-11-11.md#supplemental-exact-artifact-provenance-checks) separately records later local archive identity/hash and read-only path inventory evidence; verify that evidence independently and keep it distinct from legal/notice review. | _Reviewer to determine_ |
| For `modernc.org/libc v1.66.3`, P4 reports that the exact tag lists `LICENSE-3RD-PARTY.md`, but its contents were not retrievable. (P4 lines 27, 45.) | The [reconciliation evidence section](p0-phase0-reconciliation-2026-11-11.md#supplemental-exact-artifact-provenance-checks), official Go proxy inventory below, and exact-tag API/tree evidence separately record checks contradicting the claimed root path. The inventory found root `LICENSE`, `LICENSE-GO`, and nested licence/copyright files, but no root `LICENSE-3RD-PARTY.md` (lines 90–106). Reconcile the exact path/ref against those checks; do not assume the companion file exists or treat uninspected contents as reviewed. | _Reviewer to determine_ |
| P4 reports a bounded tracked-file/content scan with no vendor/third_party tree, common standalone asset/icon/font/image directories or searched extensions, no repository notice bundle, and no per-source SPDX/copyright headers. It notes embedded UI output, Unicode symbols, no located external UI asset loads in a targeted static search, a generated-file relationship, and no found local regeneration command. It reports product/brand policy declarations and a bounded cPanel/WHM/UI reference search. (P4 lines 14–16, 33–35, 75–88.) | P4 explicitly says these searches do not prove all code/assets are original or cleared, do not examine every inline/historical/generated/binary item, and are not a legal trademark search or determination. | _Reviewer to determine_ |
| Read-only comparison of the exact P4 packet at PR #158 head `8b05989bf94ef596c33142bade980f97565b7b4f` (blob `20e1a2f3dccb154da9f5239e48ef896973d129bd`) found that the P4 packet supplies no immutable notice-byte snapshots, hashes, or verified destination mappings for built artifacts. The packet lists upstream licence URLs and describes source package/test workflow paths, including LICENSE-copy steps, but reports no built-artifact inspection. (P4 lines 27, 31, 45, 57–73.) | These packet observations are bounded and do not establish the contents of any assembled distribution. Subsequent local WordPress source-archive candidate-file hashes below do not establish applicability, interpretation, or destination in a retained product artifact; `modernc.org/libc` archive/module identity/path checks likewise do not map notice bytes into built artifacts. Confirm frozen release scope, then map exact applicable notice bytes to destination paths in each in-scope artifact tied to its digest. | _Reviewer to verify and record; P4 supplied no notice-byte or destination evidence_ |
| P4 states P0-09 remains **NOT VERIFIED** pending file-level provenance, contributor-rights, dependency/notice, generated-output, trademark and reviewer records; it reports no legal review, security/owner signoff, or approval. (P4 lines 1–2, 16, 122–124.) | P4 publication and reported component checks are not a qualified licence/trademark disposition or clearance. | _Reviewer to determine_ |
| Independent in-memory source recheck on 2026-11-11 confirms the official WordPress 7.1.4 archive is 35,378,819 bytes with SHA-256 `881e53a4681ff2cd911651b11cccdede5ae477ddf0765af5d6a1a8b219a1d835`; its SHA-1 `b175879c4fb2d35300f9abe1153da9b543a35a2b` matches WordPress.org's published value. Read-only tar listing found root `wordpress/license.txt` plus the 12 bundled license-file paths enumerated in the [reconciliation note](p0-phase0-reconciliation-2026-11-11.md#supplemental-exact-artifact-provenance-checks). Sources: [WordPress 7.1.4 archive](https://wordpress.org/wordpress-7.1.4.tar.gz), [release index](https://wordpress.org/download/releases/), [official SHA-1](https://wordpress.org/wordpress-7.1.4.tar.gz.sha1). For `modernc.org/libc v1.66.3`, the public checksum database reports module and go.mod sums `h1:cfCbjTUcdsKyyZZfEUKfoHcP3S0Wkvz3jgSzByEWVCQ=` and `h1:XD9zO8kt59cANKvHPXpx7yS2ELPheAey0vjIuZOhOU8=`; independent in-memory listing of the Go proxy ZIP and GitLab v1.66.3 tag source archive found `LICENSE`, `LICENSE-GO`, `honnef.co/go/netdb/LICENSE`, and `testdata/.../crlibm/COPYING`, but no root `LICENSE-3RD-PARTY.md`. The GitLab root-file API returned 404 and the tag root tree lists `LICENSE` and `LICENSE-GO`. Sources: [Go checksum database](https://sum.golang.org/lookup/modernc.org/libc@v1.66.3), [Go proxy ZIP](https://proxy.golang.org/modernc.org/libc/@v/v1.66.3.zip), [GitLab v1.66.3 source archive](https://gitlab.com/cznic/libc/-/archive/v1.66.3/libc-v1.66.3.tar.gz), [GitLab tag tree](https://gitlab.com/api/v4/projects/cznic%2Flibc/repository/tree?ref=v1.66.3&per_page=100), [GitLab root-file API](https://gitlab.com/api/v4/projects/cznic%2Flibc/repository/files/LICENSE-3RD-PARTY.md?ref=v1.66.3). | This row records the earlier in-memory-only retrieval: bytes were not persisted at that time, and the later check of the then-specified scratch folder did not find them. That historical state is superseded by the retained-archive follow-up below. Notice contents, artifact applicability, and obligations were not reviewed. | _Reviewer to independently verify against frozen scope and record evidence/disposition_ |

| Retained WordPress archive follow-up, 2026-11-11 22:03 UTC; reviewer corpus added 2026-11-12 | A new direct HTTPS fetch from `downloads.wordpress.org` returned 200 without redirect; the retained 35,378,819-byte archive SHA-256 matches the P1 recipe blob `2ffccd06d17b3856f1af74e1af1a4d5aca8e1c03` pin, and its SHA-1 matches WordPress.org's published checksum. A no-extraction inspection verified 4,250 members, one `wordpress/` root, 468 directories and 3,782 regular files; no duplicate/unsafe paths or links/special entries. The CSV manifests and local retained archive are under `[local path omitted]` (machine-local path, not a repository link): archive SHA-256 `881e53a4681ff2cd911651b11cccdede5ae477ddf0765af5d6a1a8b219a1d835`; full member manifest SHA-256 `fd1512b4a03e9598efb1b49c629531b028efcb5a1e557006c4372e907c1bb06e`; candidate-file manifest SHA-256 `1560970273f639cfdb2842642014461434202440c5943ec4bf4d2fa2d9a178b3`. All 4,250 member rows match path/type/size/mode; each of 24 filename-keyword candidates also matches archive path/size/mode and has a SHA-256. Thirteen basenames are license-named (root `wordpress/license.txt` plus the 12 nested paths above); the other 11 are credits/copyright/notice-name candidates, not license-named files. A separate corpus at `[local path omitted]` contains only those 13 license-named members with their archive-relative paths preserved. Its `corpus-manifest.csv` records archive path, extracted relative path, size and SHA-256; manifest SHA-256 is `1179c1d073ffa91c440cc0cb52e558893ebf7dbfcaea0b24249435dbb9b56a51`; post-write verification confirmed all 13 paths, sizes and hashes match. | This supersedes the earlier in-memory-only retention status. The corpus is a copy of source-document bytes for later review only; it does not interpret contents, establish applicability, select a distribution, provide clearance, or record a reviewer disposition. The archive/member and filename-candidate inventories remain source evidence only, not a frozen distribution decision or mapping to assembled artifacts. The other 11 non-license-named entries were not included in the corpus. | Review applicable contents against a frozen scope; map source/runtime paths to each artifact digest; record qualified reviewer and separate maintainer dispositions. No clearance is implied. |

### Second P4 discrepancy check — exact path and distribution boundary (no clearance)

A separate read-only helper rechecked P4 PR #158 at head `8b05989bf94ef596c33142bade980f97565b7b4f` against the P1 source commit above. At GitLab tag commit `6b8211ebda96d570fde5b5f21a49fd39ca23994d`, the root `LICENSE-3RD-PARTY.md` API path returns 404; the exact-tag root `LICENSE` does exist (1,482 bytes; content SHA-256 `95ff867eb55a56935fa7492406cfa953fb7c13ca73f4c0a86ae05756b4605600`, blob `0c6500bc3615df5cf4cd5c43e0598af481e03534`). P1 pins `modernc.org/libc v1.66.3` in `go.mod` blob `9737ad41c6eb33929aa3dc98d75606d792e95181` and records module sums in `go.sum` blob `7c0e1fb7dbe96cad88ed60c9149cba13cb782f9e`; module sums do not hash a licence file. This is a citation/path discrepancy only; it does not establish that no other notice exists.

### Official Go proxy archive verification — modernc.org/libc v1.66.3 (path evidence only)

On 2026-11-11, a read-only helper fetched the official [Go proxy ZIP](https://proxy.golang.org/modernc.org/libc/@v/v1.66.3.zip) (HTTP 200; 18,238,212 bytes) in memory and inspected ZIP entry names/content without extracting or installing. Raw ZIP SHA-256: `6cd7f37dcff7d4249e8674b0bfdff88057fbd4e646a87b77121efd9d1536dc4f`; 4,159 entries. Recomputed module ZIP hash `h1:cfCbjTUcdsKyyZZfEUKfoHcP3S0Wkvz3jgSzByEWVCQ=` and go.mod hash `h1:XD9zO8kt59cANKvHPXpx7yS2ELPheAey0vjIuZOhOU8=` match the official [checksum database record](https://sum.golang.org/lookup/modernc.org/libc@v1.66.3) and the exact P1 `go.sum` blob `7c0e1fb7dbe96cad88ed60c9149cba13cb782f9e` for pinned P1 source commit `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23`.

| Exact module ZIP path | Bytes | Content SHA-256 |
|---|---:|---|
| `modernc.org/libc@v1.66.3/LICENSE` | 1,482 | `95ff867eb55a56935fa7492406cfa953fb7c13ca73f4c0a86ae05756b4605600` |
| `modernc.org/libc@v1.66.3/LICENSE-GO` | 1,479 | `2d36597f7117c38b006835ae7f537487207d8ec407aa9d9980794b2030cbc067` |
| `modernc.org/libc@v1.66.3/COPYRIGHT-MUSL` | 6,204 | `f9bc4423732350eb0b3f7ed7e91d530298476f8fec0c6c427a1c04ade22655af` |
| `modernc.org/libc@v1.66.3/honnef.co/go/netdb/LICENSE` | 1,058 | `657707d00e00d3077e4ccfac2d6a123ad908c3f503370798eb2cb52e68e135a6` |
| `modernc.org/libc@v1.66.3/testdata/nsz.repo.hu/libc-test/src/math/crlibm/COPYING` | 18,007 | `dcc100d4161cc0b7177545ab6e47216f84857cda3843847c792a25289852dcaa` |
| `modernc.org/libc@v1.66.3/AUTHORS` | 729 | `7e5e6de42f1f53dfb9c6c5c018529c60972697a33daafc9f462339bc7460baff` |
| `modernc.org/libc@v1.66.3/CONTRIBUTORS` | 834 | `d144b5bdbcba7c6026528c971ad4c010beedcebf1a6fba7a642459ecd5b563ee` |
| `modernc.org/libc@v1.66.3/testdata/nsz.repo.hu/libc-test/src/math/crlibm/AUTHORS` | 103 | `d04faa7fbed5d7e30d5f55c014c0f5e702245ca5ba62a39c7bc4df8724853a6c` |
| `modernc.org/libc@v1.66.3/testdata/nsz.repo.hu/libc-test/src/math/crlibm/COPYRIGHT` | 1,314 | `41b8f2e2706012eb534f89e72ff547308cb81dcb9acf077c40a9058a7c06088d` |

The exact module ZIP contains no root `NOTICE` and does not contain the P4-claimed root path `LICENSE-3RD-PARTY.md`. This resolves the pinned Go module ZIP's path inventory only; it does not interpret file contents, decide which nested/test files apply, or establish distribution scope, licence obligations, or clearance. The P4 citation discrepancy remains for the qualified reviewer and packet owner to reconcile.

### Official Go proxy archives — root modules omitted from the preview SBOM (path evidence only)

On 2026-11-11, a read-only helper verified the two root `go.sum` entries reported absent from the PR #149 binary SBOM. It fetched only the official Go proxy ZIPs and checksum database records, inspected archive entries in memory, and compared the computed Go module and go.mod hashes to the exact P1 `go.sum` blob `7c0e1fb7dbe96cad88ed60c9149cba13cb782f9e` at commit `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23`.

| Pinned module | Official proxy ZIP / bytes / raw ZIP SHA-256 | Computed module h1 / go.sum result | Exact matching path inventory within ZIP |
|---|---|---|---|
| `github.com/mattn/go-isatty@v0.0.20` | [ZIP](https://proxy.golang.org/github.com/mattn/go-isatty/@v/v0.0.20.zip); 8,936 bytes; `f2d5f89ca451577e17464b9bb596dc0d0ecececb5eaa63622c41b57cd0b7b8cc` | `h1:xfD0iDuEKnDkl03q4limB+vH+GxLEtL/jb4xVJSWWEY=`; matches checksum database and exact P1 `go.sum`; go.mod h1 `h1:W+V8PltTTMOvKvAeJH7IuucS94S2C6jfK/D7dTCTo3Y=` also matches | `github.com/mattn/go-isatty@v0.0.20/LICENSE` — 1,099 bytes; SHA-256 `08eab1118c80885fa1fa6a6dd7303f65a379fcb3733e063d20d1bbc2c76e6fa1`; no other matching legal/attribution filename in this ZIP |
| `github.com/ncruces/go-strftime@v0.1.9` | [ZIP](https://proxy.golang.org/github.com/ncruces/go-strftime/@v/v0.1.9.zip); 15,977 bytes; `3c46ee9c9db8fde8ce93c768a8701fa01f630bab0cfff338481cde768fe561ac` | `h1:bY0MQC28UADQmHmaF5dgpLmImcShSi2kHU9XLdhx/f4=`; matches checksum database and exact P1 `go.sum`; go.mod h1 `h1:Fwc5htZGVVkseilnfgOVb9mKy6w1naJmn9CehxcKcls=` also matches | `github.com/ncruces/go-strftime@v0.1.9/LICENSE` — 1,068 bytes; SHA-256 `38ae43959daf953a393a585b2988672cb65a5a541aca0d0be5e72595a0a16883`; no other matching legal/attribution filename in this ZIP |

These exact source-archive inventories do not interpret licence contents, establish whether either module is distributed, or resolve notice/source-offer obligations. Their absence from the preview SBOM does not establish exclusion from distribution. No files were written or modules installed.

At P1 source commit `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23`, `internal/sites/wordpress_recipe_linux.go` blob `2ffccd06d17b3856f1af74e1af1a4d5aca8e1c03` pins WordPress 7.1.4's URL and SHA-256 and extracts the runtime-fetched archive below a pinned tenant root at a caller-provided relative destination. The retained source archive and member manifest are recorded above; this identifies source-archive members only, not the selected tenant destination or inclusion in a frozen distribution scope. The preview workflow `.github/workflows/p1-preview-package.yml` blob `8f4d53bda62f595f6d0e12a3d606e516d1b12970` builds the two Linux amd64 binaries and packages `LICENSE`, `DO_NOT_INSTALL.txt`, and two binary-bound SBOMs at `tong-p1-dev/sbom/tongd.cdx.json` and `tong-p1-dev/sbom/tongctl.cdx.json`; it uploads the unsigned preview archive and `SHA256SUMS` as a CI artifact.

The separate Debian builder `scripts/build_deb.sh` blob `f35310d899cba2d9f48d4f0d0f40dd29af7b78d8` takes the binaries as inputs and installs them at `/usr/libexec/tong-panel-dev/tongd` and `/usr/bin/tongctl`, with the service at `/usr/lib/systemd/system/tong-panel-dev.service`, `DO_NOT_INSTALL.txt` at `/usr/share/doc/tong-panel-dev/DO_NOT_INSTALL.txt`, and the project `LICENSE` at `/usr/share/doc/tong-panel-dev/copyright`; it also creates Debian maintainer scripts and package metadata. This builder does not package an SBOM or a separate third-party notice bundle. The Debian test fixtures are built by `scripts/test_deb_package.sh` blob `37ad878e8c872ded93086aa4ea96d697ec792a3c`; the clean-VM workflow `.github/workflows/p2-clean-vm-smoke.yml` blob `65d270e349e5817aeab0721750d76e234168b9cd` transfers those fixtures and checksums into the disposable guest. P4 records these as test packaging paths, not approved releases, and reports no generated-artifact inspection (PR #158 head `8b05989bf94ef596c33142bade980f97565b7b4f`, packet blob `20e1a2f3dccb154da9f5239e48ef896973d129bd`, distribution-path table). These source observations do not select any product distribution channel or map notices to a built artifact digest. Reviewer follow-up remains to choose scope, map exact notice bytes and hashes to every selected destination/artifact digest, and record qualified reviewer and separate maintainer dispositions. No legal or distribution approval is inferred.

All items above remain evidence pointers requiring review. They do not populate the manifest, artifact/SBOM findings, per-item disposition, qualified reviewer decision, or P0-09 gate fields below.

**Current retained-archive state (updated 2026-11-12):** The preceding instruction to retain or refetch the WordPress archive is superseded by the direct download recorded in the retained-archive evidence row above. The archive and member/candidate manifests exist in the linked local P0-09 evidence folder, and the archive SHA-256 matches the P1 pin. The separate reviewer corpus at `[local path omitted]` contains only the 13 license-named source-archive members, with preserved archive-relative paths; its `corpus-manifest.csv` SHA-256 is `1179c1d073ffa91c440cc0cb52e558893ebf7dbfcaea0b24249435dbb9b56a51`. It is a source-document corpus only; no interpretation, applicability, distribution choice, clearance, or reviewer disposition is recorded. Frozen distribution scope, notice-content review, artifact-destination mapping, and qualified reviewer/maintainer dispositions remain open.

The earlier paragraph's retain/refetch instruction is superseded by the **Retained WordPress archive follow-up** above. The bytes and manifests now exist in the linked local evidence folder; reviewer verification, frozen distribution scope, notice-content review, artifact-destination mapping, and qualified/maintainer dispositions remain open.

## Independent artifact evidence annex — PR #149 unsigned developer preview (no clearance)

This annex records a separate read-only inspection of the GitHub Actions artifact; it is not part of P4's packet. Artifact ID `13751448173`, name `tong-p1-UNSIGNED-developer-preview-linux-amd64`, came from run `42790473328`, tied to PR #149 head `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23` on feature base `482017005f9872c1fe7f2969ebfb477ea7de30dd`. GitHub listed expiry `2026-11-17 09:25:56 UTC`. The ZIP was downloaded to scratch outside the repository and read in memory; no product binary was executed and no repository file was extracted or modified.

| Evidence item | Independently verified observation | Boundary / reviewer action |
|---|---|---|
| Outer artifact | ZIP size `16,921,562` bytes; SHA-256 `1c4338e44382c1271d27955271cce0e99debbd7479cc67e8494fc0f6b4a7c9bc`. It contains `tong-p1-UNSIGNED-dev-linux-amd64.tar.gz` and `SHA256SUMS`. | Digest identifies this downloaded artifact only; verify against retained artifact and frozen distribution scope before disposition. |
| Inner tarball | SHA-256 `9c23619ecce3fef3711c748a1d8a8b5d460177310fd1722eb5113383d47893a3`, matching the bundled `SHA256SUMS`. Read-only member listing: `DO_NOT_INSTALL.txt` (317 bytes), `LICENSE` (34,523), `sbom/tongd.cdx.json` (11,504), `sbom/tongctl.cdx.json` (11,514), `bin/tongd` (16,579,844), and `bin/tongctl` (12,905,095), all under `tong-p1-dev/`. | This is an unsigned developer preview, not a signed release or clean-VM guest artifact. `DO_NOT_INSTALL.txt` says not to install in production/expose to the internet and says it is not a supported distro package or signed release. |
| Packaged project license | `tong-p1-dev/LICENSE` SHA-256 `8486a10c4393cee1c25392769ddd3b2d6c242d6ec7928e1414efff7dfb2f07ef`; its bytes match `LICENSE` at the exact PR #149 source head. This maps one included license file to one exact inner tarball path, inner tarball digest, and outer artifact digest. | It is the GNU AGPL-3.0 text for the Tong Panel package. This does not establish that third-party obligations are fully covered or determine whether this preview is an in-scope distribution. |
| Separate notices and SBOM licence metadata | The inventory has no separate `NOTICE` file or `LICENSES` directory. Each CycloneDX SBOM lists 11 components and zero component entries with `licenses` metadata. | Inventory is limited to this artifact and these SBOMs; do not infer that no third-party notice obligations apply. Reviewer must reconcile each component and applicable notice/source offer to the frozen artifact and distribution scope. |

### SBOM-to-source-manifest crosswalk (read-only helper audit; no licence disposition)

The two archive SBOMs contain the same 11 components and hashes. The SHA-256 values below were reported to match the hexadecimal-decoded exact-version `h1:` checksums in root `go.sum` for the ten listed Go modules. All eleven SBOM component entries in each file have `licenses: null`; the `std go1.27.3` row has no component hash and is not a `go.mod`/`go.sum` dependency. This is checksum correspondence, not verification of licence terms or whether the component is legally in scope.

| Component and version | SBOM component SHA-256 | Root `go.sum` match | SBOM licence metadata |
|---|---|---|---|
| `github.com/dustin/go-humanize v1.0.1` | `1b392163b4f954d8449301f43d52608f3f9f5f5ae106b47ba514f79839297826` | Yes | Absent |
| `github.com/google/uuid v1.6.0` | `348bda24330eb231c0f27d630212d2833ac0cf2d4782bfa136b6f9edefbde05d` | Yes | Absent |
| `github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec` | `5b4f4854973de2272ae0d8d8ddc95becb93c3b5a89f01741105f33d226d4d2b1` | Yes | Absent |
| `golang.org/x/crypto v0.41.0` | `58a63159e74f1824d5565e7e587492ace053d0ef25c77dbecf1987c628e05e9e` | Yes | Absent |
| `golang.org/x/exp v0.0.0-20250620022241-b7579e27df2b` | `336ac333acf7161a338bd3bb356b3102483fcaa4bf95027a3e6932215dd83fea` | Yes | Absent |
| `golang.org/x/sys v0.35.0` | `bf3d4ddfb80fe5bb3cf6cec77bc5ee2185e9c98d3e425b0a9b38a909b52dcb12` | Yes | Absent |
| `modernc.org/libc v1.66.3` | `71f09b8d351c76c2b2c9965f11429fa0770fdd2d1692fcf78e04b30721165424` | Yes | Absent |
| `modernc.org/mathutil v1.7.1` | `18265519775a37c813a81d4c7ffbaca7563f852aa0236bc01863f88d9302c4e5` | Yes | Absent |
| `modernc.org/memory v1.11.0` | `a38402f1a310ce673008adeddd4c7f6479b014fcc4ea17f66392db911b3e85b2` | Yes | Absent |
| `modernc.org/sqlite v1.38.2` | `01c96eefeb608e0710552859aa29b8d416f0f42868d32ffb5b3629b57aa411e9` | Yes | Absent |
| `std go1.27.3` | Not reported | Not a root module row | Absent |

The helper also compared the exact PR #149 source manifests with P4's pinned PR #158 inventory. Root `go.sum` rows `github.com/mattn/go-isatty v0.0.20` and `github.com/ncruces/go-strftime v0.1.9` are not represented in either binary SBOM; they remain source-manifest provenance questions, not cleared exclusions. A separate `tools/openapi-check` module has seven pinned modules (`kin-openapi`, `jsonpointer`, `swag/jsonname`, `gorilla/mux`, `jsonschema/v6`, `go.yaml.in/yaml/v4`, and `x/text`) outside these binary SBOMs. The `tools/ui-browser` lockfile has five Playwright/axe packages outside this archive. P4 describes these as separate development-tool/browser-test inventories and states that the binary SBOMs exclude source, UI/assets, OS packages and deployment inventory and do not perform licence detection.

The helper reported that each binary's embedded hash matches its SBOM (`tongd`: `9295fe90913601e0ee2947a9373354788037b6bd17d494ec54bbed439fce1656`; `tongctl`: `b1dc7c1e35ef95424c69753010f1da72cbfe5aabdd7ca81666025e2133aee0c1`). SBOM file hashes reported were `0be1096956386099cfb8bc0f77569ff8c4b2c222d17a36eebcd693ac20fd5666` (`tongd.cdx.json`) and `fd9390928ed8fbbbf9c9ffc253f7a16eb15b92c9ca7be3698d5d9be0f393b00f` (`tongctl.cdx.json`). These reported correspondences were not independently rerun in this worktree. The reviewer must still define distribution scope; reconcile source, build-only, UI, artifact and OS dependencies; determine applicable licences/notices/source offers; and map exact notice bytes into every in-scope artifact. No qualified disposition or maintainer approval is recorded.


### Nested OpenAPI-check tooling dependency archive evidence — read-only, no licence disposition

At PR #149 source head `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23`, `tools/openapi-check/go.mod` blob `5881338d936010dcfef8fc982e9078c36694452d` and `tools/openapi-check/go.sum` blob `1364591f27b506d9a839334b423cbc09e1c029ff` pin the seven module versions below. Each module ZIP and `.mod` file was fetched in memory from the official Go module proxy. Recomputed Go `h1:` hashes for every ZIP and `go.mod` matched the exact nested `go.sum` rows and corresponding official checksum-database records at `https://sum.golang.org/lookup/<module>@<version>`. Module ZIP source: `https://proxy.golang.org/<module>/@v/<version>.zip`.

| Module and version | ZIP `h1:` | `go.mod` `h1:` |
|---|---|---|
| `github.com/getkin/kin-openapi v0.153.0` | `Qo4qWW+Pj8o90LUTE6MDyG5TJAtWnfvrcWOBcWg7eBw=` | `mEZ6Ip3gyqPDokUsJsttZhv3r46/NeH3tGwrFUMcoSE=` |
| `github.com/go-openapi/jsonpointer v0.22.5` | `8on/0Yp4uTb9f4XvTrM2+1CPrV05QPZXu+rvu2o9jcA=` | `gyUR3sCvGSWchA2sUBJGluYMbe1zazrYWIkWPjjMUY0=` |
| `github.com/go-openapi/swag/jsonname v0.25.5` | `8p150i44rv/Drip4vWI3kGi9+4W9TdI3US3uUYSFhSo=` | `jNqqikyiAK56uS7n8sLkdaNY/uq6+D2m2LANat09pKU=` |
| `github.com/gorilla/mux v1.8.0` | `i40aqfkR1h2SlN9hojwV5ZA91wcXFOvkdNIeFDP5koI=` | `DVbg23sWSpFRCP0SfiEN6jmj59UnW/n46BH5rLB71So=` |
| `github.com/santhosh-tekuri/jsonschema/v6 v6.0.3` | `1EYB5IzjZawrrnELUi78f9fPu57HuXjmddZPjrls/28=` | `JXeL+ps8p7/KNMjDQk3TCwPpBy0wYklyWTfbkIzdIFU=` |
| `go.yaml.in/yaml/v4 v4.0.0` | `/LG1pNO84Z4l603j/xMn69CcS5OYhMQ/f7l2LoQUGuw=` | `aZqd9kCMsGL7AuUv/m/PvWLdg5sjJsZ4oHDEnfPPfY0=` |
| `golang.org/x/text v0.14.0` | `ScX5w1eTa3QqT8oi6+ziP7dTV1S2+ALU0bI+0zXKWiQ=` | `18ZOQIKpY8NJVqYksKHtTdi31H5itFRjB5/qKTNYzSU=` |

The following exact root/member paths are relative to each module ZIP's `<module>@<version>/` prefix. Sizes are uncompressed bytes; SHA-256 values were computed from member bytes held in memory. Attribution and legal-file names are recorded as source evidence only.

| Module/version | Exact member path | Bytes | SHA-256 |
|---|---|---:|---|
| `github.com/getkin/kin-openapi v0.153.0` | `LICENSE` | 1,082 | `612a11e78c07e12765d9cf3866e3edc5f8212c541602721b14e465311afb3aa6` |
| `github.com/go-openapi/jsonpointer v0.22.5` | `CONTRIBUTORS.md` | 1,396 | `9a9dca09bb0295da5d84f28bb079e4eb58cc172a6b6cc24fc7c4beb384d70017` |
| `github.com/go-openapi/jsonpointer v0.22.5` | `LICENSE` | 11,357 | `c71d239df91726fc519c6eb72d318ec65820627232b2f796219e87dcf35d0ab4` |
| `github.com/go-openapi/jsonpointer v0.22.5` | `NOTICE` | 1,586 | `83beb6f0e9796c82a96ee42535db23d01d91c44bf95b8d4423eadc868fb07117` |
| `github.com/go-openapi/swag/jsonname v0.25.5` | `LICENSE` | 11,358 | `cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30` |
| `github.com/gorilla/mux v1.8.0` | `AUTHORS` | 276 | `b73d9792b0b99af35ed05442080a264acffe8221a7a0949033199bcfc30c9117` |
| `github.com/gorilla/mux v1.8.0` | `LICENSE` | 1,486 | `af984eea4080365beb209c62c42e3e197dd0d578a3ccd0a1e5dbbb1c23550211` |
| `github.com/santhosh-tekuri/jsonschema/v6 v6.0.3` | `LICENSE` | 10,141 | `c8858a5a76440bbca484e134cf7df46385d090dd18b2c58e650f939258802e5b` |
| `go.yaml.in/yaml/v4 v4.0.0` | `LICENSE` | 11,365 | `4a4e095045f46875715234a884741acd2b61b88959d6f47a243ae03ef95c4e76` |
| `go.yaml.in/yaml/v4 v4.0.0` | `NOTICE` | 702 | `257bb2d062ee0b9226e3b5f9c3c5d6aa3d38a450df27a9e060f061d4d4a4eb6f` |
| `golang.org/x/text v0.14.0` | `LICENSE` | 1,479 | `2d36597f7117c38b006835ae7f537487207d8ec407aa9d9980794b2030cbc067` |
| `golang.org/x/text v0.14.0` | `PATENTS` | 1,303 | `96f408bfae65bf137fc2525d3ecb030271c50c1e90799f87abf8846d8dd505cc` |

These are nested `tools/openapi-check` developer-tooling dependencies, outside the PR #149 preview binary SBOM; the exact distribution scope of this tooling is not established here. This section does not interpret licence terms, determine notice applicability or distribution obligations, or provide legal clearance. Reviewer and maintainer dispositions remain blank.

The exact download remains in `[local path omitted]` with a source-`LICENSE` comparison copy in the same scratch folder. This evidence supplements but does not revise the P4 packet's statement that P4 itself did not inspect generated artifacts. The preview's presence does not settle the owner's intended distribution scope. No complete component-level licence/NOTICE review, source-offer mapping, qualified licence/trademark disposition, or maintainer distribution decision is recorded; P0-09 remains NOT VERIFIED.

## 3. Source, generated code, and asset origin manifest

Record every shipped or embedded first-party, third-party, generated, copied, bundled, and inline asset. Include UI artwork, icons, fonts, screenshots, templates, documentation media, installer payloads, vendored files, generated bindings, and code derived from tools or schemas.

| Component / file or path | Origin / upstream URL | Author / contributor and rights evidence | Version / exact source revision | Local SHA-256 | Acquisition / inclusion mode (tracked / copied / vendored / generated / embedded / runtime-fetched / build-test-only / external service) | Generated or modified? (how/by what) | Shipped in which artifact(s), or runtime/external scope | Licence / terms evidence | Notice / attribution required | Exception, question, or evidence reference | Reviewer disposition |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |

| Manifest control | Record |
| --- | --- |
| How completeness was checked (tracked/untracked, generated, embedded, build output) | _Reviewer to determine_ |
| Contributor assignment / CLA / employer-rights evidence reviewed | _Reviewer to determine_ |
| Unattributed or uncertain-origin items | _Reviewer to determine_ |
| Manifest reviewer and exact snapshot reviewed | _Reviewer to determine_ |

## 4. Dependency, licence, notice, and exception graph

Include direct and transitive dependencies, optional/build-only dependencies where distributed or material, vendored code, embedded runtimes, operating-system packages, fonts, and generated output. Tie each node and edge to the frozen artifact and exact evidence. Identify combined or modified works and applicable source/offer/notice obligations for reviewer determination.

| Component ID | Parent / dependency edge | Name, version, source, hash | Direct / transitive / vendored / generated | SPDX identifier and evidence | Licence text / notice path and hash | Modified / combined? | Source offer / attribution obligation | Exception or unresolved question | Reviewer disposition |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |

| Graph / notice controls | Record |
| --- | --- |
| Dependency inventory commands/tools and versions | _Reviewer to determine_ |
| Lockfiles and their exact hashes | _Reviewer to determine_ |
| SPDX expression / graph export and hash | _Reviewer to determine_ |
| Exceptions, dual licences, and interpretation questions | _Reviewer to determine_ |
| Notice bundle location, completeness check, and hash | _Reviewer to determine_ |
| Source code offer / corresponding-source access, if applicable | _Reviewer to determine_ |
| Attribution placement and verification | _Reviewer to determine_ |
| Reviewer conclusion on each obligation and evidence | _Reviewer to determine_ |

## 5. Built artifact inventory and SBOM coverage

List exact built outputs, not only source archives. Preserve artifact hashes and an extracted inventory from each artifact. Record what the SBOM covers and omits, including application source, generated/UI assets, operating-system layers, deployment dependencies, and installer payloads as applicable.

| Artifact ID / filename | Version | Build source SHA / recipe revision | Platform/profile | Artifact SHA-256 | Extraction method / tool version | Extracted inventory path and hash | SBOM format, scope, and hash | Provenance/signature attestation and verification evidence | Notice/source offer present? | Reviewer disposition |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |

| Coverage / release controls | Record |
| --- | --- |
| SBOM generation tool/version and reproducible command | _Reviewer to determine_ |
| Components missing from or not represented by the SBOM | _Reviewer to determine_ |
| Source-to-artifact and SBOM-to-extracted-inventory reconciliation | _Reviewer to determine_ |
| Signature/provenance verification method and trust basis | _Reviewer to determine_ |
| Unavailable or uninspected archives, packages, layers, or assets | _Reviewer to determine_ |
| Distribution artifacts explicitly excluded from this review | _Reviewer to determine_ |

## 6. Source offer, attribution, and notices

| Item / obligation | Exact artifact location or URL | Evidence / hash / tested retrieval date | Required by reviewer? | Reviewer disposition and rationale |
| --- | --- | --- | --- | --- |
| Corresponding source or source offer | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| Third-party notices and licence texts | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| Attribution / copyright statements | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| Modification notices / combined-work notices | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| Source-access duration and availability owner | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |

For every notice or licence text relied on for a disposition, preserve the exact retrieved bytes in the review evidence bundle or record a durable immutable retrieval reference, calculate SHA-256 over those bytes, and identify the precise destination in each in-scope source archive and assembled distribution artifact (package/image name and version plus internal path or equivalent location). Tie each destination to that artifact's digest. If the bytes, hash, or destination cannot be verified, record it as unavailable or unknown in sections 8–10; a cited URL, package manifest, or source-code claim alone does not close that evidence gap.

## 7. Trademark, product-name, and UI references

| Name / mark / visual reference | Owner / source | Use location and context (code, UI, docs, domain, marketing) | Evidence / permission or policy reviewed | Proposed treatment / limitation | Reviewer disposition |
| --- | --- | --- | --- | --- | --- |
| _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |

Include product names, logos, comparative references, screenshots, imported visual assets, fonts, and third-party UI likenesses. Record limits on affiliation, endorsement, compatibility claims, and attribution for reviewer determination; do not infer permission from public availability.

## 8. Exclusions, open questions, and required follow-up

| Item / component / artifact | Why excluded or unresolved | Risk or obligation potentially affected | Smallest evidence or decision needed | Responsible reviewer/owner to determine | Due date / disposition |
| --- | --- | --- | --- | --- | --- |
| _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |

Explicitly list uninspected archives, inaccessible sources, missing origin/rights records, dependency exceptions, unknown package contents, SBOM gaps, trademark questions, and any counsel referral. An exclusion is not clearance.

## 9. Per-item findings and disposition

For each manifest row, dependency, notice obligation, built artifact, asset, mark, and exclusion, record a separate result and supporting evidence. Use only the disposition vocabulary selected by the qualified reviewer; do not treat “not reviewed,” “unknown,” or “out of scope” as acceptance. The maintainer's separate aggregate decision is recorded in §10; cross-reference its exact decision record to every item it covers, and list any item excluded, unresolved, or subject to a distinct disposition there. A reviewer disposition is not a maintainer decision.

| Item ID | Evidence revision/hash and exact reviewed location | Finding / obligation | Disposition (reviewer to determine) | Rationale | Required action / owner to determine | Residual risk / exclusion |
| --- | --- | --- | --- | --- | --- | --- |
| _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |

## 10. Maintainer decision and P0-09 gate

| Decision field | Record |
| --- | --- |
| Maintainer's explicit P0-09 disposition (satisfied / not satisfied / deferred / reviewer to determine) | _Reviewer to determine_ |
| Exact source and distribution snapshot covered | _Reviewer to determine_ |
| Qualified licence reviewer disposition and date | _Reviewer to determine_ |
| Qualified trademark reviewer disposition and date | _Reviewer to determine_ |
| Independent reviewer qualifications and independence confirmed by | _Reviewer to determine_ |
| Counsel needed / referral disposition | _Reviewer to determine_ |
| Residual exclusions and accepted limitations | _Reviewer to determine_ |
| Evidence gaps that keep P0-09 open | _Reviewer to determine_ |
| Decision owner, authority, and date | _Reviewer to determine_ |
| Exact reviewed record revision/hash | _Reviewer to determine_ |
| Item IDs covered by this decision and per-item exclusions/open dispositions (cross-reference §9) | _Reviewer to determine_ |

**P0-09 gate result: _Reviewer to determine._** The gate is not satisfied until the authorized maintainer records a decision for the exact scope, supported by qualified licence and trademark review, complete artifact and source/asset provenance evidence, applicable notices/source-offer evidence, and explicit unresolved exclusions. For notice evidence, this includes the exact notice bytes or immutable retrieval reference, their byte-level SHA-256, and a verified mapping to each in-scope package destination; unresolved bytes or destinations remain explicit open gaps, not assumed absence or clearance. This record alone does not satisfy the gate. A P0-09 disposition does not imply independent security approval, P0 completion, P1/P2 authorization, or permission to distribute.

## 11. Signoff record

| Role | Name / qualification / authority | Exact revision reviewed | Decision / disposition | Date (UTC) | Signature or durable record reference |
| --- | --- | --- | --- | --- | --- |
| Qualified licence reviewer | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| Qualified trademark reviewer | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |
| Project maintainer / decision owner | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ | _Reviewer to determine_ |

No signature, finding, or approval is recorded by creating or filling this template without the named reviewers' explicit dated dispositions.

## Supplemental source-archive content observations — no reviewer or maintainer disposition

This annex records bounded source/package-byte observations gathered after the preceding evidence sections. It supplements those evidence rows only. It does not complete or change any scope selection, reviewer disposition, exclusion, or maintainer decision in §§1–11; all such fields remain blank for the authorized owner/reviewers.

### WordPress 7.1.4 runtime archive

At the exact P1 recipe source blob `2ffccd06d17b3856f1af74e1af1a4d5aca8e1c03`, the pinned archive was 35,378,819 bytes; SHA-256 `881e53a4681ff2cd911651b11cccdede5ae477ddf0765af5d6a1a8b219a1d835` matches the recipe pin and SHA-1 `b175879c4fb2d35300f9abe1153da9b543a35a2b` matches the publisher's checksum. The 13 license-named file paths and hashes are in the reconciliation's **P0-09 WordPress 7.1.4 archive identity and notice-file inventory** section (lines 1345–1369); the literal content observations are in **P0-09 WordPress archive content facts** (lines 1371–1389).

Literal observations include the archive root `wordpress/license.txt` stating GPL v2 or later and containing GPL v2 text; GPL v2 text in the Akismet and plupload files; SIL Open Font License 1.1 text in the named font files; the getID3 file's listed GPL/LGPL/MPL options; the Crystal Project Icons file's “Released under LGPL” wording; an empty zero-byte SWFUpload `license.txt`; TinyMCE's LGPL v2.1 text; and the sodium_compat file's “ISC License” heading. The DM Sans and Source Serif Pro font files include reserved-font-name “Source” wording; Source Serif Pro names Adobe Systems and identifies “Source” as an Adobe trademark. These are statements present in those inspected bytes, not determinations about component scope or use.

### `modernc.org/libc` v1.66.3 Go module/tag archives

The exact Go proxy ZIP and GitLab tag archive have matching six non-empty notice-like files. Their complete paths, byte sizes, and SHA-256 values are in this form's **Official Go proxy archive verification — modernc.org/libc v1.66.3** table (§100–116); the reconciliation's **P0-09 modernc.org/libc v1.66.3 notice-content facts** section (lines 1391–1404) records literal content. The root `LICENSE` and `LICENSE-GO` have no SPDX identifier or titled licence text observed and begin with BSD-style terms, with attributions to The Libc Authors (2017) and The Go Authors (2009), respectively. `COPYRIGHT-MUSL` states “musl as a whole is licensed under the following standard MIT license” and names Rich Felker and other contributors. The nested netdb `LICENSE` contains MIT-style text and names Dominik Honnef (2012). The testdata `libc-test/COPYRIGHT` states a standard MIT license, names libc-test AUTHORS (2005–2013), and notes math-test numbers under BSD and GPL. The nested `crlibm/COPYING` begins “GNU GENERAL PUBLIC LICENSE, Version 2, June 1991” and includes the Free Software Foundation 1989/1991 copyright line.

The exact module ZIP and tag archive did not contain the P4-claimed root `LICENSE-3RD-PARTY.md`; this is a path-citation discrepancy, not a finding that other notices are absent. Module checksums identify module bytes, not individual notice files.

### Playwright 1.65.0 npm tarball nested `.LICENSE` files

At P1 source commit `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23`, the five `tools/ui-browser` package versions and registry integrity values are recorded in the reconciliation's **UI browser package source archive check** section (lines 1024–1034). The six nested `.LICENSE` paths, byte lengths, and SHA-256 values appear in **P0-09 Playwright npm tarball notice-file inventory correction** (lines 1406–1419); literal content summaries appear in **P0-09 npm bundled-notice content facts** (lines 1431–1444).

All six files are non-empty and none contains an `SPDX-License-Identifier` string. The three `playwright@1.65.0` files identify bundled Babel/Jridgewell, ESM-loader, and Babel/Jest bundle paths and include repeated MIT text and contributor attributions. The `playwright-core@1.65.0` `serverRegistry.js.LICENSE` names chokidar and readdirp and includes MIT text; `utilsBundle.js.LICENSE` lists bundled packages including Hono, the Model Context Protocol SDK, and readable-stream packages and contains multiple contributor/copyright statements; `webp_codec.LICENSE` identifies `webp_codec.wasm` and generated `webp_codec.js`, libwebp, Emscripten, Node, and musl, with license headings and attribution text. The packages' root `LICENSE`/`NOTICE` files are separately described in the cited reconciliation section; the nested bundle files are additional archive members.

### Evidence boundary

These observations identify source archive/package bytes and literal file text only. They do not determine whether any component or text applies to a particular Tong Panel artifact, what must be distributed, whether a notice set is complete or sufficient, licence compatibility, attribution sufficiency, trademark permission, or distribution rights. No source/distribution scope was frozen and no qualified licence/trademark review or separate maintainer disposition is recorded here. P0-09 remains **NOT VERIFIED**; all reviewer, owner, disposition, exclusion, and signoff fields above remain for authorized completion.
