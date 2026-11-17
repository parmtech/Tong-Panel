# P4 migration review-binding prototype

This standalone Go module is a **partial, offline, non-normative binding model** for part of the candidate P4 migration dry-run review contract; it is not an implementation of the candidate report or execution flow. It compares a later request against a reviewed migration ID, modeled plan, tenant, destination, principal, action set, one illustrative inventory revision, policy revision, and a separately supplied current-authorization result. `PlanSHA256` and `ActionSetSHA256` are local prototype hashes, **not** the candidate manifest `plan_fingerprint` and not interchangeable with it. The candidate fingerprint also covers source format/parser identity, inventory classifications, required capabilities, secret actions, unsupported items, warnings, and resource estimates, which this prototype does not model. Hashing canonicalizes action and dependency arrays by stable reference; dependencies define required ordering, and array order does not define execution order. The single opaque inventory revision is illustrative only: its scope, issuer, and consistency semantics are not specified or approved here. Per-action `ExpectedRevision` is also only a modeled hash field; the candidate contract does not define its requiredness or create/absence sentinel. Fixture values such as `fixture-revision-4` have no normative meaning.

It does not parse archives, access a filesystem or network, authenticate or authorize a principal, sign a review receipt, execute actions, or prove production safety. `AuthorizationResult` only models a hypothetical allow/deny result bound in this model to tenant, destination, principal, action-set hash, and policy revision. The prototype compares the supplied values but does not produce or verify the decision, model capability-ceiling calculation/enforcement, or prove that the external policy check was authentic, current at effect time, or race-free. Its in-memory `Review` value is not a security credential. The candidate data types and validation rules are not an approved Tong Panel API or migration format; they require P0/P1/P2/P3 contract and security review before product use.

## Automated and local checks

This is a nested Go module, so root-module `go test ./...` does not traverse it. The [offline model and fixture workflow](../../.github/workflows/offline-model-checks.yml) enters this module explicitly and checks formatting, race-enabled tests and vet on Ubuntu 24.04 with Go 1.27.4. Module downloads are disabled and manifests must remain unchanged.

From this directory, with Go 1.27.4 and a C compiler on `PATH`:

```sh
export CGO_ENABLED=1 GOTOOLCHAIN=local GOFLAGS=-mod=readonly
export GOWORK=off GOPROXY=off GOSUMDB=off
go test -race -count=1 -timeout=120s ./...
go vet ./...
```

The synthetic identifiers and digests in tests are inert fixtures. Do not treat passing these model tests as P4 acceptance evidence.
