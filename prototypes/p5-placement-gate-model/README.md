# P5 placement gate model (prototype)

This offline, standard-library-only model explores candidate predicates from P5 scenarios 9, 29, and 30: a candidate node must have fresh/healthy inventory, an exact non-empty environment match, exact requested labels, known sufficient abstract capacity, and no active maintenance window. Empty or whitespace-only IDs, environment bindings, and required-label keys or values are rejected; all other strings are compared exactly without normalization. Selection uses the lexicographically smallest eligible node ID so the result is stable across inventory ordering. Malformed constraints and blank or duplicate inventory node IDs fail closed.

It does not reserve capacity, provide fairness, schedule jobs, interpret timezones, resolve or authorize credentials, move data, contact nodes, or integrate with Tong Panel. Concurrent schedulers could over-allocate because selection is not a reservation. It is not a product implementation or security boundary. Maintenance behavior for queued/running jobs, schedule authority, inventory/label authority, environment-label ownership, resource accounting, and credential/data isolation still require architecture and security review. The tie-break rule, abstract capacity units, half-open maintenance interval, and exact string equality are candidate semantics for discussion only.

## Automated and local checks

Root-module `go test ./...` does not traverse this separate module. The [offline model and fixture workflow](../../.github/workflows/offline-model-checks.yml) enters it explicitly and checks formatting, race-enabled tests and vet on Ubuntu 24.04 with Go 1.27.4. Module downloads are disabled and manifests must remain unchanged. Passing these checks verifies candidate selection predicates only, not a scheduler, security boundary or P5 acceptance.

From this directory, with Go 1.27.4 and a C compiler on `PATH`:

```sh
export CGO_ENABLED=1 GOTOOLCHAIN=local GOFLAGS=-mod=readonly
export GOWORK=off GOPROXY=off GOSUMDB=off
go test -race -count=1 -timeout=120s ./...
go vet ./...
```
