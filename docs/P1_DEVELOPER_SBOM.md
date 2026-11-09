# Binary SBOMs for the unsigned developer archive

The developer archive workflow produces one CycloneDX 1.6 JSON SBOM for each
Linux amd64, CGO-disabled `tongd` and `tongctl` binary. It installs the external
generator `github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.12.0`
outside the application module and uses its `bin` mode against the exact files
that are archived. This reads Go build metadata without executing either binary.
The generator's `-std` option includes the Go standard library component.
Its environment explicitly matches the binary build target so generated
package-URL qualifiers describe Linux amd64; build properties record CGO disabled.

The archive contains:

- `bin/tongd` with `sbom/tongd.cdx.json`
- `bin/tongctl` with `sbom/tongctl.cdx.json`
- the existing licence and explicit unsigned developer-preview warning

`scripts/verify_binary_sbom.py` compares each SBOM with its corresponding binary
before archiving and again after extraction. A missing or mismatched SBOM fails
the package job. The verifier's regression tests run before tool installation.
The existing archive checksum, extracted-binary process smoke, owner bootstrap
and read-only doctor checks continue to run. The SBOMs are distributed inside the
same checksummed developer archive rather than as unrelated loose files.

Verification binds the binary SHA-256 and expected main package, checks target
and toolchain properties, and compares the complete linked module name/version
inventory and module checksums against `go version -m -json`. Go module `h1`
checksums are compared with their hex representation in the generated SBOM;
these are module-content checksums, not hashes of separately distributed library
binaries. Module replacements are rejected by this developer contract. This
verifier enforces the stated inventory contract, not every CycloneDX schema rule.

This inventory is limited to information represented by Go binary build
metadata. It is not a complete source, licence, embedded UI/asset, operating
system package or deployment inventory. Automated licence detection is not
enabled, and the SBOMs do not assert third-party licence approval. The metadata
is produced from CI-built files; it cannot authenticate an independently
supplied or modified binary. These files and their archive remain unsigned and
do not establish build provenance, reproducibility, production support or a
release approval. No release version is invented for development builds.

The [`cyclonedx-gomod` v1.12.0 documentation](https://github.com/CycloneDX/cyclonedx-gomod/blob/v1.12.0/README.md#bin)
describes binary mode and its limitations. Signing, comprehensive dependency
licence/NOTICE review, full release inventory and the remaining P1 acceptance
gates remain separate work.
