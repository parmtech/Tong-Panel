# Explicit local ACME end-to-end test

Run from this isolated module with official Go 1.27.4:

    go test -tags acme_e2e -count=1 -timeout=90s ./...

The build-tagged pebble_test.go exercises the existing HTTP-01 provider against a disposable local test CA. It adds no main-module/daemon dependency or feature and performs no public issuance, live-domain control, host configuration, system CA trust, filesystem key storage or service changes.

## Provenance

The official Go module github.com/letsencrypt/pebble/v2 v2.10.3 resolves to tag commit 5caaab384f267491d407b1ecaf2286cb69078fc4. Module checksum: h1:Zg/Rq9T8FvNMvfLvi3k+dXmrwgDnG9neRpeaJY2hwVM=; go.mod checksum: h1:t86OY665ATaOG3amRg83FSFMp7/L8MWjvlzBFmXANYc=. Go's default checksum database authenticated the download; no security controls were disabled.

[Official source](https://github.com/letsencrypt/pebble/tree/v2.10.3), [MPL-2.0 license](https://github.com/letsencrypt/pebble/blob/v2.10.3/LICENSE), and [upstream testing documentation](https://github.com/letsencrypt/pebble/blob/v2.10.3/README.md). Adding Pebble raised go-jose from v4.1.4 to v4.1.8. Exact dependency/test versions and checksums are recorded in go.mod/go.sum. Original license texts for Pebble (MPL-2.0), challtestsrv (MPL-2.0), miekg/dns (BSD-3-Clause), go-jose (Apache-2.0), lego (MIT) and backoff (MIT) are retained in third-party-notices/. No upstream implementation was copied. Full distribution/SBOM/transitive-license review remains open.

## Safety boundary and assertions

Each subtest starts ephemeral IPv4-loopback DNS, HTTP challenge and HTTPS ACME fixtures. The private TCP DNS responder returns loopback only for the selected reserved .test name and never forwards queries. The lego transport permits only the exact test ACME endpoint, has no proxy, and verifies TLS against the test server certificate. It never sets InsecureSkipVerify. Redirects outside that endpoint are rejected. No system resolver, hosts file or system CA trust is changed.

PEBBLE_VA_ALWAYS_VALID is explicitly false: Pebble performs actual HTTP-01 validation. Random sleeps and nonce rejection are disabled solely for deterministic timing. `TestPebbleHTTP01` runs these three cases, with no external email address/account:

| Subtest | Reserved domain | Assertions |
|---|---|---|
| `valid` | `valid.acme.test` | Real validation HTTP requests; exactly one successful `Present`/`CleanUp` pair; exact certificate SAN, chain to the in-memory Pebble root and matching private key; empty challenge state afterward |
| `invalid-challenge` | `invalid.acme.test` | Fixture forces a real HTTP 404; `Obtain` returns an error and no certificate; presentation and cleanup both occur; empty challenge state afterward |
| `canceled-attempt` | `canceled.acme.test` | Cancels the provider's attempt context immediately after the observed successful `Present`; the provider returns a real HTTP 404 during validation; `Obtain` returns an error and no certificate; cleanup occurs and token/key authorization/expiry are cleared; provider remains closed with no cancellation registration |

Each case constructs its provider with `NewForAttempt` and closes it at cleanup. The cancellation case leaves lego's order/transport running long enough to observe the failed validation; it does not establish whole-order cancellation. `Close` and cancellation fence subsequent provider admissions, while a response admitted before closure may still finish sending copied bytes. Unit regressions separately prove that blocked successful body writes and blocked 404 `WriteHeader` calls cannot block closure, and exercise concurrent presentation/serving/cancellation/cleanup. Future order authorization, key custody and restart behavior are specified in the [certificate operation contract](../../docs/P1_CERTIFICATE_OPERATION_CONTRACT.md), not implemented by this fixture.

Test CA/account/certificate keys stay in RAM and are never logged or written. Test listeners close after each subtest. Pebble's internal validation workers have no shutdown API and end with the short-lived go test process; do not embed this test setup in a persistent service.

## Historical Windows evidence — 2026-11-12

These results predate the attempt-lifecycle changes and cover only the original success and forced-404 cases.

- Initial explicit E2E pass: 10.37s (valid challenge 4.59s; invalid challenge 5.78s).
- Full tagged test suite: PASS, 10.927s; default unit suite: PASS, 1.082s.
- Tagged go vet, go mod verify and git diff --check: PASS.
- govulncheck -test -tags acme_e2e ./...: no vulnerabilities found.
- Race detection remains unverified because this Windows environment lacks cgo/compiler support.

Pebble is a test CA with deliberately different behavior. These checks do not establish public CA issuance, tenant/domain authorization, production security approval or phase completion. Next: account/key custody, interrupted-order/restart reconciliation, renewal/expiry and authorization/host integration review.

## Automated Linux coverage

[Isolated Go prototype checks](../../.github/workflows/prototype-go-checks.yml) runs the following from this module with pinned Go 1.27.4 and cgo enabled on Ubuntu 24.04:

```sh
go mod download
go mod verify
go test -race -count=1 -timeout=120s ./...
go vet ./...
go test -race -tags acme_e2e -count=1 -timeout=120s ./...
go vet -tags acme_e2e ./...
```

It installs `golang.org/x/vuln/cmd/govulncheck@v1.9.0` into a temporary tool directory and scans both the default and `acme_e2e` source/test graphs with `-test -scan symbol`. Module manifests must remain unchanged. A failed test, race report, analyzer error or known reachable vulnerability fails its job; no failure is treated as a skip. The 20-minute job limit includes dependency/tool downloads, while each test process has a 120-second limit.

This closes the automation gap caused by root-module Go commands excluding nested modules. Read the workflow run for the exact reviewed commit before asserting Linux/race success. The historical evidence above is unchanged, and the same reserved-domain, loopback, in-memory-key and no-public-CA boundaries apply.

### Historical local Linux baseline — before attempt lifecycle

The commands above were executed with the verified official Go 1.27.4 linux/amd64 toolchain, cgo enabled, `GOTOOLCHAIN=local`, and read-only module mode. The default race suite passed in 1.026s; the tagged race suite, including real local validation success and forced-404 cleanup, passed in 13.691s. Both vet variants and module verification passed, and both module manifests remained unchanged. No race report was produced.

`govulncheck v1.9.0` reported no vulnerabilities for either the default or tagged source/test graph using the database updated at `2026-11-11T02:51:11Z`. These are local Linux execution results; they do not substitute for the new hosted workflow result on its reviewed commit or cover production authorization, key custody, renewal or restart behavior.

The later [PR #167 hosted prototype workflow](https://github.com/parmtech/Tong-Panel/actions/runs/43224283911) succeeded on baseline commit `188851da58480f13b486b43b9d6e29ea3fd7c968`. Both that workflow and the local timings above precede the new provider lifecycle and third Pebble case.

### Attempt lifecycle continuation — local Linux evidence

On branch `feature/p1-acme-evidence-lifecycle`, based on the baseline above, the following checks ran with the verified official Go 1.27.4 linux/amd64 toolchain, cgo enabled, `GOTOOLCHAIN=local`, and `GOFLAGS='-mod=readonly -p=2'`:

| Command/check | Observed result |
|---|---|
| `go test -race -count=1 -timeout=120s ./...` | PASS, 1.043s, including lifecycle, synchronous cancellation fencing, blocked response I/O and concurrent use |
| `go vet ./...` | PASS |
| `go test -race -tags acme_e2e -count=1 -timeout=120s ./...` | PASS, 14.419s, including all three local Pebble scenarios |
| `go vet -tags acme_e2e ./...` | PASS |
| `govulncheck -test -scan symbol -show version ./...` | No vulnerabilities found |
| `govulncheck -test -tags acme_e2e -scan symbol -show version ./...` | No vulnerabilities found |
| `go mod verify`; module-manifest diff; scoped whitespace check | PASS; `go.mod` and `go.sum` unchanged |

Both scans used `govulncheck v1.9.0` and the database updated at `2026-11-11T02:51:11Z`. After the complete race/vet runs, a startup barrier was added only to the constructor-cancellation regression. The focused `go test -race -run '^TestCancellationDuringAttemptConstruction$' -count=1 -timeout=120s ./...` rerun passed in 1.032s. A final complete tagged race run after that test change passed in 6.201s, including the default tests and all three local Pebble scenarios. Production code did not change after the first complete runs; exact-commit hosted results remain separate evidence to record.

These checks establish the tested local provider lifecycle and local-CA outcomes. They do not approve production issuance, entire-order cancellation, certificate revocation, persistent key storage, renewal, restart reconciliation, host changes or phase completion.
