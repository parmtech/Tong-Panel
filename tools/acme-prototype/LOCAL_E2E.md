# Explicit local ACME end-to-end test

Run from this isolated module with official Go 1.27.4:

    go test -tags acme_e2e -count=1 -timeout=90s ./...

The build-tagged pebble_test.go exercises the existing HTTP-01 provider against a disposable local test CA. It adds no main-module/daemon dependency or feature and performs no public issuance, live-domain control, host configuration, system CA trust, filesystem key storage or service changes.

## Provenance

The official Go module github.com/letsencrypt/pebble/v2 v2.10.3 resolves to tag commit 5caaab384f267491d407b1ecaf2286cb69078fc4. Module checksum: h1:Zg/Rq9T8FvNMvfLvi3k+dXmrwgDnG9neRpeaJY2hwVM=; go.mod checksum: h1:t86OY665ATaOG3amRg83FSFMp7/L8MWjvlzBFmXANYc=. Go's default checksum database authenticated the download; no security controls were disabled.

[Official source](https://github.com/letsencrypt/pebble/tree/v2.10.3), [MPL-2.0 license](https://github.com/letsencrypt/pebble/blob/v2.10.3/LICENSE), and [upstream testing documentation](https://github.com/letsencrypt/pebble/blob/v2.10.3/README.md). Adding Pebble raised go-jose from v4.1.4 to v4.1.8. Exact dependency/test versions and checksums are recorded in go.mod/go.sum. Original license texts for Pebble (MPL-2.0), challtestsrv (MPL-2.0), miekg/dns (BSD-3-Clause), go-jose (Apache-2.0), lego (MIT) and backoff (MIT) are retained in third-party-notices/. No upstream implementation was copied. Full distribution/SBOM/transitive-license review remains open.

## Safety boundary and assertions

Each subtest starts ephemeral IPv4-loopback DNS, HTTP challenge and HTTPS ACME fixtures. The private TCP DNS responder returns loopback only for the selected reserved .test name and never forwards queries. The lego transport permits only the exact test ACME endpoint, has no proxy, and verifies TLS against the test server certificate. It never sets InsecureSkipVerify. Redirects outside that endpoint are rejected. No system resolver, hosts file or system CA trust is changed.

PEBBLE_VA_ALWAYS_VALID is explicitly false: Pebble performs actual HTTP-01 validation. Random sleeps and nonce rejection are disabled solely for deterministic timing. Both the success and forced-404 failure assert actual validation HTTP requests, one Present/CleanUp pair and no remaining challenge token. Success verifies the exact SAN, chain to the generated in-memory Pebble root and matching private key. Failure returns no certificate. The names are valid.acme.test and invalid.acme.test, with no external email address/account.

Test CA/account/certificate keys stay in RAM and are never logged or written. Test listeners close after each subtest. Pebble's internal validation workers have no shutdown API and end with the short-lived go test process; do not embed this test setup in a persistent service.

## Executed evidence — 2026-11-12

- Initial explicit E2E pass: 10.37s (valid challenge 4.59s; invalid challenge 5.78s).
- Full tagged test suite: PASS, 10.927s; default unit suite: PASS, 1.082s.
- Tagged go vet, go mod verify and git diff --check: PASS.
- govulncheck -test -tags acme_e2e ./...: no vulnerabilities found.
- Race detection remains unverified because this Windows environment lacks cgo/compiler support.

Pebble is a test CA with deliberately different behavior. These checks do not establish public CA issuance, tenant/domain authorization, production security approval or phase completion. Next: account/key custody, interrupted-order/restart reconciliation, renewal/expiry and authorization/host integration review.
