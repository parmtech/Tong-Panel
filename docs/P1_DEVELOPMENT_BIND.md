# Development listener configuration

The experimental `tongd` daemon defaults to `127.0.0.1:8090`. An optional
`TONG_DEV_LISTEN_ADDR` selects a numeric IPv4 loopback address and port 1–65535:

```sh
TONG_DEV_LISTEN_ADDR=127.0.0.1:18090 ./tongd
```

Empty means the default. The value must be an address and port, without URL
scheme, whitespace or path. Hostnames (including `localhost`), wildcard and
nonloopback addresses, IPv6, mapped IPv6 and port zero are rejected. The daemon
uses `tcp4`, preserving its existing IPv4 scope. It does not resolve DNS or
silently fall back to another port. An occupied address stops startup.

Configuration validation precedes opening SQLite or the JSON job journal.
Actual listener binding still precedes starting the optional no-op worker.
Tests may separately use an internal ephemeral listener; that does not make
operator port zero valid.

This remains a plaintext development listener, without remote/public binding,
TLS, reverse-proxy trust or public session/login support. Local processes and
host port forwarding remain outside address-parser guarantees. Optional bearer
authentication is configured separately; changing the port grants no authority.
Existing health/version, request limits and server timeouts remain in effect.

The default OpenAPI server URL stays `http://127.0.0.1:8090`; clients targeting a
configured development port must select that URL explicitly. This increment
does not establish production deployment or complete Phase 1 acceptance.
