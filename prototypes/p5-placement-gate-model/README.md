# P5 placement gate model (prototype)

This offline, standard-library-only model explores two acceptance predicates from P5 scenarios 9, 29, and 30: a candidate node must have fresh/healthy inventory and an exact non-empty environment match, and a node inside a valid maintenance window is not eligible for new placement.

It does not schedule jobs, interpret timezones, resolve or authorize credentials, move data, contact nodes, or integrate with Tong Panel. It is not a product implementation or a security boundary. Maintenance behavior for queued/running jobs, schedule authority, environment-label ownership, and credential/data isolation still require architecture and security review. The half-open interval and exact string equality are candidate semantics for discussion only.

Run from this directory with Go 1.27.3 on `PATH`:

```powershell
go test ./...
go vet ./...
```
