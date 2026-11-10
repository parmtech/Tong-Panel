# P0 P4 Provenance and Licence Readiness Review

**Status:** draft evidence inventory; not legal clearance, licence approval, or P0 acceptance.
**Owner:** P4 workstream, P0 assignment.
**Prepared:** 2026-11-10.
**P0 draft worktree:** `p0/p4-provenance-readiness-20261110`, based on `main` at `0aad68e81138a0f226e83e401f90bae4d0b6bddc`.
**Code inventory snapshot:** P1 branch `feature/p1-file-mutation-context` at `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23` (clean published commit); the P1 lead's P0 baseline uses the same main base. This review does not include P1's separately preserved uncommitted paths.
**Scope:** repository-declared dependencies, notices/licence evidence, CI supply-chain inputs, tracked source/assets indicators, generated UI indicators, and declared trademark constraints. Exact upstream versions and legal compatibility require qualified reviewer verification before distribution.

## Summary

The repository declares AGPL-3.0 for original Tong Panel code and separately licensed third-party components. The checked P1 source snapshot has pinned Go and npm dependency manifests, but there is no tracked third-party `NOTICE`/licence bundle or checked-in generated SBOM. A later P1 draft documents an unsigned developer-archive SBOM for two Go binaries; it explicitly excludes full source, UI/assets, OS packages and deployment inventory and performs no automated licence detection.

A tracked-file scan of the P1 snapshot found no `vendor`/`third_party` tree, standalone asset/icon/font/image directory, common image/font file extensions, `NOTICE`, `COPYRIGHT`, `COPYING`, SPDX file, or per-source SPDX/copyright headers. This is a bounded path/content search, not proof that all code is original or all inline assets are cleared. `cmd/tongd/ui/app.js` appears to be compiled output corresponding to `web/src/app.ts`; the browser workflow compiles the TypeScript and compares emitted `app.js` with the embedded file, while the separate type-check workflow uses `--noEmit`. A documented local regeneration command was not found in the inspected files.

The policy documents prohibit copying proprietary cPanel assets/design and characterize the project as original. These declarations are useful constraints, not provenance evidence or a legal determination. P0 item P0-09 therefore remains **unverified** pending a file-level inventory, contributor-rights evidence, dependency licence/NOTICE review, generated-output provenance, trademark review, and the owner's/security/licence review record.

## Repository and dependency evidence

Evidence below is scoped to the P1 source snapshot at `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23`, except where explicitly labeled as a main-branch planning declaration.

| Area | Observed evidence | Readiness gap |
|---|---|---|
| Project licence | `LICENSE` is GNU AGPL version 3. `README.md:4–6` says AGPL-3.0 applies to original Tong Panel code and third-party dependencies keep their own licences. | No independent copyright-holder/rights manifest for each original contribution was found in this bounded scan. Confirm contributor rights and any copyright notice expectations with the maintainer/legal reviewer. |
| Root Go module | `go.mod:1–20` declares Go 1.23.0; direct modules `golang.org/x/crypto v0.41.0`, `golang.org/x/sys v0.35.0`, and `modernc.org/sqlite v1.38.2`, plus indirect modules. `go.sum` has 51 lines. | No complete component-to-SPDX/licence/NOTICE review is recorded. `go.sum` content checksums pin module contents but are not licence findings or separate binary hashes. |
| Nested Go tool | `tools/openapi-check/go.mod:1–14` pins `github.com/getkin/kin-openapi v0.153.0` (MIT per `tools/openapi-check/README.md:3`) and indirect modules; its `go.sum` has 26 lines. The README says the root vulnerability scan excludes this module. | Confirm a separate vulnerability and licence scan for this tool and its transitive graph, or document why exclusion is acceptable. Its development-only purpose does not remove redistribution/licence obligations if distributed. |
| Browser test dependencies | `tools/ui-browser/package.json:5–9` pins `@axe-core/playwright 4.14.0` and `@playwright/test 1.65.0`. The lockfile records MPL-2.0 and Apache-2.0 package licence strings respectively and five package entries overall. | Record the full transitive npm licence inventory and how these dev/test dependencies are or are not included in distributions. Lockfile metadata is not legal approval. |
| CI tools/actions | Inspected workflows pin Go 1.27.3, `govulncheck v1.9.0`, `cyclonedx-gomod v1.12.0`, Node 24.x and TypeScript 5.9.3. Workflow action references are mixed: some immutable commit SHAs, some movable version tags such as `actions/checkout@v4` and `actions/setup-node@v4`. | Decide which release workflows require immutable action/tool digests and recorded provenance. This inventory does not establish current upstream integrity or tool security. |
| Notices and machine-readable inventory | The tracked P1 snapshot has `LICENSE`, but no `NOTICE`, `COPYRIGHT`, `COPYING`, `LICENSES/`, `.spdx`, or `.cdx.json` artifact in the scanned tree. | No repository-tracked complete third-party notice bundle or licence decision record was found. Reconcile with generated release contents before any distribution claim. |
| Binary SBOM | The P1-only draft `docs/P1_DEVELOPER_SBOM.md` describes CycloneDX 1.6 SBOMs for exact Linux amd64 CGO-disabled `tongd` and `tongctl` binaries in an unsigned preview archive, checked against binary build metadata. It explicitly excludes complete source, licence, embedded UI/asset, OS-package and deployment inventory; licence detection is disabled; it does not establish build provenance, reproducibility, signatures, or release approval. | Treat binary SBOM as a limited component inventory only. Decide on source/UI/npm/OS/deployment coverage, licence findings, provenance attestation, signatures and a reviewable distribution record. This file is on the P1 branch, not on main at the P0 draft base. |
| Upstream study | Main `docs/UPSTREAM_RESEARCH.md` is dated 2026-10-26 and explicitly says it is research, not copying permission; exact versions/licences must be revalidated. Its checklist asks for exact URL, version/tag, commit, SPDX ID, notices/copyright, integration/derivative analysis, brand/asset rights, security status, pins, SBOM, tests and removal path. | Refresh any upstream facts at the point of proposed integration. Feature research must not be mistaken for code or asset provenance. |

## Distribution package path observations

These are development/test packaging paths observed at the P1 source snapshot. They are not approved releases and this review did not build or inspect generated artifacts.

| Path | Assembled content observed in source | Inventory limitation |
|---|---|---|
| `.github/workflows/p1-preview-package.yml:54–78` unsigned developer tarball | Linux amd64 CGO-disabled `tongd` and `tongctl`, copied `LICENSE`, `DO_NOT_INSTALL.txt`, two binary-bound CycloneDX SBOMs under `sbom/`, tarball and separate SHA256SUMS. The workflow uploads the tarball and checksum as a seven-day CI artifact. UI is embedded in `tongd`; separate npm/UI files are not copied. | SBOMs cover binary Go module metadata only, without full licence/source/UI/OS/deployment inventory or provenance/signature. |
| `.github/workflows/p2-clean-vm-smoke.yml:63–79` disposable-VM smoke tarball | `tongd`, `tongctl`, `tongbackup`, three compiled test binaries, `LICENSE`, smoke scripts, `DO_NOT_INSTALL.txt`, tarball and SHA256SUMS; `scripts/clean_vm_smoke.sh` transfers these into a disposable guest. | No SBOM is assembled in this path, and no `upload-artifact` step was found in the inspected workflow. It is a test path, not a release. |
| `scripts/build_deb.sh:35–43` and `scripts/test_deb_package.sh:25–45` Debian test fixtures | `.deb` contains `tongd`, `tongctl`, systemd service, `DO_NOT_INSTALL.txt`, `LICENSE` installed as Debian `copyright`, maintainer scripts and package metadata; fixture harness builds v1/v2 packages and a checksum list. | Builder does not add an SBOM, Go/npm manifests, or separate third-party notice bundle. The fixtures are test inputs, not a supported package. |

The release-scope checklist below should treat each path separately. Which test-only artifacts need SBOM/notice handling remains a maintainer/release-policy decision; the existence of a checksum or a binary SBOM does not establish a signed release or complete provenance.

## Source, asset and generated-file observations

At P1 source commit `4dab4c60cc86186935e9a7d987b9fe1ca97a1f23`:

- A `git ls-tree` scan found no tracked `vendor` or `third_party` directory, common standalone `assets`, `icons`, `fonts` or `images` directory, common image/font files (`png`, `jpg/jpeg`, `webp`, `svg`, `ico`, `woff/woff2`, `ttf`, `otf`), or source-file `SPDX-License-Identifier`/copyright headers under `cmd`, `internal`, `db`, `web`, `scripts`, and `tools`.
- UI static files are embedded by `cmd/tongd/ui.go:10–11` via `//go:embed ui/*`. The inspected UI uses Unicode emoji tile symbols in `app.js`; no third-party icon/font image files were found by the scan.
- `web/tsconfig.json:8–11` sets `outDir` to `../cmd/tongd/ui`, and `cmd/tongd/ui/app.js` has TypeScript-stripping characteristics and matching UI imports with `web/src/app.ts`. `.github/workflows/ui-checks.yml:25–26` runs a type check with `--noEmit`; separately, `.github/workflows/ui-browser-checks.yml:34–37` compiles using TypeScript 5.9.3 to a temporary directory and compares emitted `app.js` with the embedded file using `cmp`. This CI check supports the generated-output relationship. A generated-file header or documented local update/regeneration command was not found in the inspected files.
- No file-level source-origin manifest was found. The tracked-tree scan cannot establish absence of copied snippets in inline source, historic commits, generated files, or material not matched by these extensions and names. It also does not inspect P1's uncommitted paths.

The P1 source branch's `README.md:41–42` describes cPanel/WHM as third-party marks and says the project is unaffiliated; `README.md:10` and `docs/UX_UI_SPEC.md:5` call for original design. `CONTRIBUTING.md:29` requires original UI/icons and prohibits cPanel assets and commercial/proprietary source while requiring contributor rights and dependency notices. `docs/UPSTREAM_RESEARCH.md:37–40,47` says permission/licence compatibility does not automatically clear icons, screenshots, brands, designs, plugins or exceptions. These are policy declarations. This audit did not perform a legal trademark search or inspect every visible string, screenshot, commit, or binary.

## Gaps and questions for owner/security/licence reviewers

1. Who holds copyright in original project contributions, and what contributor-origin/rights attestation and copyright notice should accompany distributions?
2. Can maintainers provide a file-level provenance manifest for each nontrivial imported dependency, code snippet, generated file and asset: source URL, exact version/commit, component, SPDX identifier, copyright/NOTICE, integration method, and licence/trademark review status?
3. Does the CI compile-and-compare check for `cmd/tongd/ui/app.js` establish the required generated-output provenance, and what documented local command/tool version should maintainers use to regenerate the checked-in file?
4. Should the nested `tools/openapi-check` Go module receive its own vulnerability and licence scan, since the README explicitly excludes it from the root Go vulnerability scan?
5. What is the release-scope licence inventory for npm test packages, source/UI assets, OS packages, and deployment dependencies, given the binary-only SBOM's stated exclusions?
6. Which CI actions/tools must be pinned by immutable digest/checksum, and what reviewed build-provenance/signature evidence is required before distributing artifacts?
7. Can the project provide an explicit third-party notice bundle or generated licence report for every distributed dependency, and who reviews exceptions/compatibility questions?
8. How should a legal reviewer assess the README’s comparative product-positioning phrase “a future cPanel/WHM alternative” alongside the affiliation disclaimer? The phrase is not evidence of tested compatibility or affiliation. Are all branding, screenshots, icons, fonts and visual assets independently created or properly authorized? This draft does not conclude that any wording is cleared or infringing.

These are unresolved review questions, not a request to repeat routine project authorization. Project-work authorization does not answer technical licence choices or create legal/security approval.

## Minimum evidence checklist for closing the inventory gaps

These checks identify evidence needed for owner/security/licence review. They do not themselves grant clearance. The responsible role must be assigned; no named reviewer is assumed.

| Check | Evidence required | Reviewer role | Unresolved detail |
|---|---|---|---|
| Close P0-09 provenance review | Bounded file/source/asset inventory with scan method and limits; for each identified external/copied/embedded/generated item, exact origin/version/commit, rights/licence/attribution and trademark-review status. If none are identified, record the scoped none-found result and scan limits. Record the maintainer/reviewer disposition and unresolved exclusions. | P0 owner/maintainer and licence/trademark reviewer | Named reviewer, review method and any items needing legal advice |
| Freeze snapshot | Exact source SHA and retained hashes or copies of Go/npm manifests, lockfiles and relevant workflows | P1 maintainer / release engineer | Required retention format and location |
| Review root Go graph | Full direct/transitive inventory with version, verified SPDX identifier, source and notice reference; record exceptions | Dependency/SCA and licence reviewers | Compatibility and transitive notice obligations |
| Scan nested Go tool | Snapshot-bound vulnerability and licence reports for `tools/openapi-check`, or documented exclusions | Dependency/SCA reviewer | Whether this tool is distributed or source/CI-only |
| Review npm test tooling | Full five-package versions, SPDX identifiers and notice sources; record distribution inclusion | UI maintainer and licence reviewer | Distribution treatment and notices |
| Inventory CI inputs | Version/integrity record for Go, Node, TypeScript, scanners/generators and workflow actions | CI/release owner and security reviewer | Which inputs require immutable digest/checksum pins |
| Verify binary SBOMs | Each SBOM bound to exact binary hash, target and module inventory; retain verifier result | Release engineer and dependency/SCA reviewer | Required source/UI/OS/deployment coverage |
| Reconcile package contents | File-level archive/package inventory, including binaries, SBOMs, licences/notices and bundled dependencies; verify after extraction | Release engineer | Intended formats and any non-Go bundled components |
| Complete notices | Reviewed third-party notice/licence bundle or component-specific reason none is required, tied to package contents | Licence reviewer | Notice format, attribution placement and exceptions |
| Record build provenance | Source commit, workflow/tool versions, binary hashes, retained SBOMs and any required signature/attestation verification | Security/release reviewer | Provenance standard, signing identity and verification method |
| Record disposition | Exact snapshot, reports, reviewer decisions, exclusions, unresolved items and distribution scope | P0 decision owner and P1 maintainer | Named reviewers and distribution blockers |

## Verification state and next action

This is a documentation-only draft. The inventory used tracked-tree queries and read-only review of repository declarations/manifests at the exact commits listed above. No tests were run. No dependency was installed, updated, or copied. No legal review or owner/security sign-off was performed. P0-09 remains **NOT VERIFIED**.

Next: receive P1's verified P0 handoff, reconcile any changed source/asset inventory against its exact baseline (including separately preserved dirty files only after P1 supplies a safe read-only handoff), then publish this P4-owned evidence packet and unresolved questions through a P4 coordination event. Do not edit canonical ADRs, threat model, phase reports or `UPDATES.md` while cross-phase draft integration is held.
