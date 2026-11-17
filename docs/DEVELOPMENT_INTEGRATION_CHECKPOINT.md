# Core and WordPress restore development integration

This candidate combines two unmerged development histories, without merging or
deploying either to main:

- PR98 `0345de6b7081192cd65156a42a0dfc3c30967277`: schema12, internal recovery
  admission, current-role queue authorization, opt-in no-op worker, support
  export, vulnerability checks, binary SBOMs and HTTPS diagnostics.
- PR91 `01ae2237b8bdb4240fb79ca45842f9e6d4f549a1`: encrypted full WordPress-site
  snapshot and fresh-guest restore lab, including source-upload preservation.

Both parents are retained as merge ancestry. The imported restore test and two
helpers are unchanged from PR91. The shared VM workflow keeps the newer HTTPS
probe/test step and both restore-helper triggers. Its script stages both helpers,
quiesces the source before snapshot, transfers ciphertext and the ephemeral lab
wrapping key, runs the existing marker restore and the full-site target helper,
then removes private restore material. The journal preserves both histories.

## Validation scope

| Component | Integration contract | Evidence still required for this exact tree |
| --- | --- | --- |
| Core | Retain schema12, internal auth, role checks, worker opt-in and support export | Full Go/race/vet and built/extracted daemon smoke |
| Archive | Retain checksummed unsigned archive and binary-bound SBOM checks | Hosted artifact build/extraction and process checks |
| Restore | Retain tenant/key authentication, strict manifest/member validation, scoped SQL import, config/content restoration and cleanup | Actual Ubuntu/Debian fresh-guest execution |
| HTTPS | Require HTTP200 and exact isolated markers with bounded diagnostics, no retries | Both guest result lines; investigate failures from their actual status |

Local integration checks passed: three Bash syntax checks, actionlint, 19 Python
tests (the Linux-only socket test was skipped on Windows), Linux-amd64 site-test
cross-compilation and targeted vet. Govulncheck v1.9.0 on Linux source/test
packages reported zero reachable/imported-package findings and 21 module-only
advisories. These checks do not execute PHP/MariaDB or Linux test binaries locally.
Prior parent CI passes are separate evidence, not proof that this combined tree
passes. Hosted integration results must be recorded against its exact commit.

## Limits and remaining gates

The source snapshot assumes an idle disposable lab and uses ephemeral runner-key
handoff. It is not production key custody, offsite retention, control-plane backup
or an approved hosting restore API. The P1 archive inventories `tongd`/`tongctl`;
the VM-only `tongbackup` artifact is not silently covered by those SBOMs.

The Ubuntu HTTPS failure on PR96 remains unexplained until diagnostic evidence
establishes a cause; a new passing run alone does not establish that cause.
Owner/security ADR review, public authentication/recovery, privileged typed host
actions, comprehensive licence/release inventory, signing, supported installation
and the remaining UI/lifecycle/phase acceptance are still open. This candidate is
not a production release or a Phase1 completion claim. No production VPS changes.
