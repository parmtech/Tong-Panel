# Development API contract checker

This separate Go module pins MIT-licensed kin-openapi v0.153.0. It adds no
application runtime dependencies. Run from the repository root:

```sh
go -C tools/openapi-check test ./...
go -C tools/openapi-check run . ../../api/openapi.json
TONG_OPENAPI_REPORT=/private/new-report.json go test ./internal/api -run '^TestCurrentOpenAPIContractScenarios$' -count=1
go -C tools/openapi-check run . ../../api/openapi.json /private/new-report.json
```

Handler scenarios require Linux. Reports use synthetic fixtures, exclude
authorization, cookie and CSRF headers, and must remain private and ephemeral.
CI deletes its report and does not upload it. The checker refuses external
references, credential headers, undocumented statuses and missing successful
operation coverage. Validation diagnostics omit bodies and underlying errors.

Authentication validation is deliberately skipped in the schema checker because
the report omits credentials. The actual Go handler scenarios independently
assert authentication, TLS/CSRF and ownership behavior. This initial snapshot
does not establish a released compatibility baseline or production approval.
The root module vulnerability scan excludes this nested development-tool module.
