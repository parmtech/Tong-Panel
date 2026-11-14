# P3 deploy recovery model (non-normative)

This is a tiny, in-memory executable example for exploring a narrow subset of the crash/restart decision points in `docs/P3_ENTRY_AND_ACCEPTANCE_TEST_PLAN.md` (P3-04/P3-07). It does not inject termination at durable job, build/artifact, process-identity, audit, quota, or backup boundaries. It performs no filesystem, process, network, database, proxy, or host operations. It is not Tong Panel product code, a production contract, or evidence that any deployment/security property works.

The illustrative model allows a deployment to be called active only after the candidate artifact is verified, recorded, started, health-checked, and its route is observed. The route-effect method rechecks all promotion proofs before changing the simulated route; recovery with missing proof is uncertain. Repeated identical observations after promoted, not_promoted, and rolled_back retain those outcomes; contradictory observations after each move to uncertain. Tests serialize the in-memory record and reconstruct it before recovery to exercise four switch/rollback interruption boundaries. They also assert that a candidate route remains uncertain after a JSON round-trip when any promotion proof is missing, and that loss of proof during active recovery preserves the actual route while recording uncertainty. Rollback requests require a prior release to be available and verified, and a later loss of prior-release availability or verification is rejected before the route effect. This is only a JSON round-trip in one test process; it does not model durable storage, process termination, corruption, transaction ordering, or cross-process recovery. A second tiny model checks that repeating the same `(tenant, app, key)` with the same request digest returns the prior simulated job ID, while reusing that scope/key with a different digest conflicts. Same-process concurrent duplicates share one ID; concurrent payload conflicts have one winner. A test also checks that different payloads with the same key do not cross-contaminate the modeled tenant/app namespaces. It does not model multiple processes or durable transactions. That key scope is only a reviewable example, not an approved policy. The models are intentionally incomplete: they do not model tenant authorization, durable job lifecycle or execution, durable storage, quotas, audit persistence, or process supervision. Passing these model tests is not product or phase-acceptance evidence. Production design requires approved P1/P2 contracts and the reviewed decisions listed in the acceptance plan.

## Automated and local checks

This directory is a package in the repository's root Go module. The [offline model and fixture workflow](../../.github/workflows/offline-model-checks.yml) enters it explicitly, alongside the separate P4/P5 modules, and checks formatting, race-enabled tests and vet on Ubuntu 24.04 with Go 1.27.4. Module downloads are disabled and manifests must remain unchanged. Its separate fixture job runs the synthetic Node 24 loopback HTTP tests and the Python 3 in-process ASGI tests using the runner's Python interpreter, without installing third-party packages. These are model and fixture checks only; passing them does not satisfy P3 entry or acceptance gates.

From this directory, with Go 1.27.4 and a C compiler on `PATH`:

```sh
export CGO_ENABLED=1 GOTOOLCHAIN=local GOFLAGS=-mod=readonly
export GOWORK=off GOPROXY=off GOSUMDB=off
go test -race -count=1 -timeout=120s ./...
go vet ./...
```
