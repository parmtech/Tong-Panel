# P1 / ADR-008 — offline signed-release verification model

**Status: test-only candidate protocol. Not Tong Panel product code, a signed release, trusted production key, certified installer or release acceptance.** This separate Go module has no third-party dependencies, filesystem/network/host actions, executable installer, real package signing or account credentials.

The `Verify` function takes three **independent** inputs: (1) externally pinned expected key ID + Ed25519 public key, exact release, source commit and platform, (2) the signed raw canonical manifest and signature, and (3) an in-memory exact artifact-bytes map. This makes self-asserted trusted keys fail; the signed payload cannot change the out-of-band policy. It uses domain-separated Ed25519 bytes (`TONG-PANEL-OFFLINE-RELEASE-MODEL-v1`), checks canonical JSON (duplicate/unknown fields or alternate encodings rejected), pins a deliberately inert `tong-panel-fixture` and `inert-test-only` channel, requires Linux amd64 and exact source/version, and verifies all artifact lengths and SHA-256 hashes with no extras. Manifest and artifact counts/size are bounded. This deliberately cannot accept a real Tong Panel release.

Tests generate ephemeral signing keys at runtime and create **synthetic bytes** rather than real `.deb` artifacts. Positive fixtures prove only that the tiny verifier accepts one internally consistent test input. Negative cases include modified/unsigned/cross-domain signatures, wrong key ID or key, correctly signed downgrade/OS/architecture/source/product/channel mismatch, modified/missing/extra/duplicated or unsafe artifact names/content/digests, malformed/duplicate-field/unknown-field JSON, overlarge inputs and missing external trust. There are no test keys or secrets stored in Git.

This intentionally does **not** define or guarantee a production signer, independent key distribution/rotation/revocation, anti-rollback floor policy, source/build reproducibility, signature format across releases, OS package authenticity, repository/TUF metadata, secure artifact retrieval, secure storage, installation privilege/recovery, threat-review completion, or certificate-authority private key custody. A signature alone never grants approval to execute or install a package.

## Independent offline checks

This is a nested Go module and is **not covered** by root `go test ./...`. The [offline-model workflow](../../.github/workflows/offline-model-checks.yml) enters it explicitly with Go 1.27.4 and runs formatting, race-enabled unit tests and `go vet` with no network/module downloads.

```sh
cd prototypes/p1-release-manifest-model
export GOTOOLCHAIN=local GOWORK=off GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly CGO_ENABLED=1
go test -race -count=1 ./...
go vet ./...
```

Before a **real** signed installer can exist, ADR-008 and ADR-007 must be explicitly accepted by actual owner/security reviewers. A separately audited production design must pin authenticated trusted keys and release versions, address key compromise and rollback, attach artifact proofs to the exact signed/reproducible release, and perform staged, recoverable clean-VM acceptance on an approved OS matrix. See [security acceptance](../../docs/P1_SECURITY_ACCEPTANCE_2026-11-13.md), [developer package structural QA](../../docs/P1_DEV_PACKAGE_STRUCTURE.md), [master tracker #172](https://github.com/parmtech/Tong-Panel/issues/172) and [integration PR #171](https://github.com/parmtech/Tong-Panel/pull/171).

## Interrupted upgrade and rollback recommendation experiment

The `recovery.go` / `recovery_test.go` addition **does not perform an upgrade or rollback**. It supplies a read-only, deterministic recovery-advice function over a simulated journal and simulated external observations. It is deliberately coupled to `Verify`, so **recovery re-verifies the current candidate** with an independently pinned public key and exact manifest/artifact bytes before considering any outcome.

The journal carries one immutable attempt ID, previous/candidate source commits, exact candidate version and signed-manifest digest, plus a recorded phase (`prepared`, `switch_requested`, `rollback_requested`, `committed`, `rolled_back`). JSON round-trips simulate restart boundaries **inside one test process**. Noncanonical, duplicate-field, malformed, oversized and unexpected-field journals fail closed. The simulated external observation supplies the active source, health state, a claim that prior effects have been fenced, and a claim that the previous version is retained. A production caller would have to authenticate/verify those claims independently; this module does not.

The planner returns only an advisory state, such as `manual_review`, `keep_previous`, `candidate_ready_for_reviewed_commit`, `request_reviewed_rollback`, `rollback_consistent` or `record_consistent`. It **never writes a journal**, installs a package, stops/starts a service, switches a symlink, deletes a prior release, or declares release approval. A healthy observed candidate with a retained previous release and fenced effects is *eligible for subsequent independent commit review*, not a completed installation. If health is unknown, effects unfenced, the prior copy lost, an active release unrecognized, the journal corrupted or the verified candidate changes, the model withholds any automatic action. Repeating the same reconciliation is side-effect-free.

The test suite covers eighteen phase/effect/health/retention combinations, JSON-restart uncertainty across all five phases, fresh signature/key/source/artifact verification, malformed journal/observation denial, and deterministic repeatability. These are **model tests, not actual kill-process, power-loss, fsync, atomic rename, package-manager, signature-key-rotation, guest reinstall, host service isolation or end-to-end restore tests**. The existing P3 deploy-recovery model covers separate app-deployment semantics; this P1 extension is specifically about binding hypothetical installer recovery to fresh **signed-artifact identity and trust**.
