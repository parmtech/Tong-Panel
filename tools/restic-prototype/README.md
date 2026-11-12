# Isolated restic local restore drill

This original Tong Panel AGPL-3.0 script evaluates a separately managed restic executable with generated synthetic files only. Existing `internal/backups` code supplies streaming envelopes, rotation metadata and staged-file helpers; this drill adds evidence for an external snapshot/check/restore boundary, without integrating those packages or changing their encryption format.

The Python standard-library script creates its own temporary directory, local encrypted repository, synthetic tenant A and sibling tenant B, random process-only password, and a separate empty restore destination. It passes typed argument arrays with `shell=False`, hardcodes the local repository and permitted operations, bounds each subprocess to 90 seconds, and omits inherited cloud/storage credentials and RESTIC settings. No user-selected source, remote backend, live credential, daemon/API, service, retention/prune/delete command or real user data is involved. Passwords are not written to disk or command arguments, and raw restic output is not logged. TemporaryDirectory removes all generated fixtures, repository and restore data after the drill.

This is evidence of selected synthetic file behavior, not an authorization boundary: a production adapter still needs trusted tenant source/repository selection, process isolation, filesystem defenses, output limits, cancellation, crash recovery, secret custody and restore approval. The script captures output in memory; that is acceptable only for this tiny fixed fixture, not for unbounded production jobs.

## Verified pinned artifact

- Official release: [restic v0.19.1](https://github.com/restic/restic/releases/tag/v0.19.1), reported by its executable as `restic 0.19.1 compiled with go1.26.4 on windows/amd64`.
- Download: `https://github.com/restic/restic/releases/download/v0.19.1/restic_0.19.1_windows_amd64.zip`.
- Archive SHA256: `da948ad707ed690426473aaba2046cd61f8f90f6f0e7dab6be0d5796531de67d`, matched both the signed release SHA256SUMS and GitHub asset digest.
- Extracted executable SHA256: `b0dd1fd21eea5d8fe1325f55f7118213c21f36de8a261e04c0624a5ab9fd7830`. The script refuses any different executable before running it.
- GPG verified `SHA256SUMS.asc` against `SHA256SUMS`, returning `VALIDSIG CF8F18F2844575973F79D4E191A6868BD3F7A907`. This exactly matches the release signing fingerprint in the [official installation documentation](https://restic.readthedocs.io/en/stable/020_installation.html). Public key obtained from `https://restic.net/gpg-key-alex.asc`; verification used an isolated temporary GPG home. GPG's undefined ownertrust warning is expected for that empty keyring; identity was checked against the documented fingerprint rather than inferred from ownertrust.
- Upstream [v0.19.1 LICENSE](https://github.com/restic/restic/blob/v0.19.1/LICENSE): BSD-2-Clause; preserved as `restic-LICENSE.txt`. No upstream implementation was copied. No executable is distributed in this directory; full binary dependency/license/SBOM review remains open before distribution.

Download/verification artifacts and executable were kept outside git at `$env:TEMP\tong-restic-0.19.1-verified`. The executable was not installed globally. To reproduce, independently download and authenticate this exact official archive/signature/key, extract it outside the repository, then run:

```powershell
python tools/restic-prototype/drill.py --restic 'C:\path\to\verified\restic_0.19.1_windows_amd64.exe'
```

This exact executable pin intentionally supports only Windows amd64 for this local experiment. A Linux build needs its own authenticated artifact and updated platform-specific pin; it is not covered by these results.

## Executed evidence — 2026-11-12

`drill.py` passed in 6.19 seconds:

1. Local encrypted format-v2 repository initialization.
2. One snapshot tagged `tenant-a-fixture`, backing up tenant A only.
3. Wrong random password rejected with documented exit code 12.
4. Full repository `check --read-data` passed.
5. Restore to a separate empty temporary target with `restore --verify` passed.
6. Three restored files (text, binary and empty) exactly matched the original path/hash manifest; no file outside the selected tenant subtree and no sibling tenant marker was restored.

Python syntax compilation also passed. This is not proof of database crash consistency, hostile symlink protection, interruption recovery, remote backend integrity, Linux permissions, tenant authorization or fresh-machine service recovery.

## Security selection blocker

The official binary is authenticated but **is not cleared for integration or production use**. `govulncheck -mode binary` returned exit 3 and reported 40 symbol-level vulnerability findings across six dependency modules and its Go 1.26.4 standard library; one additional module finding was outside reported symbols. The full 2026-11-12 scan is retained in `binary-vulnerability-scan.txt`.

Examples include [GO-2026-6865](https://pkg.go.dev/vuln/GO-2026-6865) in its compression dependency and [GO-2026-4970](https://pkg.go.dev/vuln/GO-2026-4970) in Go filesystem handling. Binary screening lacks a source call graph and cannot establish that every reported symbol is reachable in this fixture or a selected restic command. The passing synthetic drill does not remediate or dismiss the findings. No untrusted archive, network backend or real data was processed during the drill.

Next: evaluate a newer authenticated official release, or a reproducible reviewed build with patched Go/dependencies, and rerun vulnerability/compatibility checks before connecting a typed backup job. After that, test interrupted backups, hostile repository restore inputs and local fault injection, then a disposable remote backend and fresh-machine restore. Do not infer a supported offsite backup service from this experiment.
