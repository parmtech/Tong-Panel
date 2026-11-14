# P5 remote outcome model (prototype)

This standalone, standard-library-only Go module explores the P5 acceptance requirement that partitions and offline nodes must not produce ghost-success or duplicate effects. It has no networking, credentials, persistent storage, host integration, or runtime dependency on Tong Panel.

The proposed classifier reports success only when an acceptance response and effect verification are both present. A lost response, unverified acceptance, or reported failure after dispatch remains `Unknown`. A dispatch requires currently valid authorization. Unknown outcomes may only be replayed in this model when authorization is still valid, the exact same idempotency key, target node, and immutable request (tenant, action, and payload) are retained, and durable worker-side deduplication is assumed. Revocation denies both new dispatches and replay.

These are candidate safety semantics for discussion, not an approved protocol or an implementation contract. In particular, this prototype does not establish what effect verification means, how durable deduplication survives worker restore, or how authorization/revocation interacts with an in-flight operation. Those decisions require P0/P1/P5 review and the upstream gates recorded in `docs/P5_PREREQUISITE_AND_ACCEPTANCE_SPEC.md`. Do not connect this module to a real node or production system.

The tests include table cases and exhaustive checks over all 8-bit response/outcome values and boolean evidence combinations. They prove invariants of this small model only; they do not establish network, persistence, identity, or host-effect behavior.

## Automated and local checks

Root-module `go test ./...` does not traverse this separate module. The [offline model and fixture workflow](../../.github/workflows/offline-model-checks.yml) enters it explicitly and checks formatting, race-enabled tests and vet on Ubuntu 24.04 with Go 1.27.4. Module downloads are disabled and manifests must remain unchanged. Passing these checks verifies this offline model only, not P5 entry or product acceptance.

From this directory, with Go 1.27.4 and a C compiler on `PATH`:

```sh
export CGO_ENABLED=1 GOTOOLCHAIN=local GOFLAGS=-mod=readonly
export GOWORK=off GOPROXY=off GOSUMDB=off
go test -race -count=1 -timeout=120s ./...
go vet ./...
```
