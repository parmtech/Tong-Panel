# Disposable HTTPS isolation probe

The guest HTTPS check now uses a status-aware, bounded probe rather than storing
arbitrary curl output and testing for a substring. It requests only the two fixed
tenant fixture hostnames, resolves them to loopback and requires HTTP 200 with the
exact fixture marker. It makes one attempt per host and never retries bad content
until it happens to pass. A wrong tenant marker remains a failing isolation check.

The fixture uses self-signed certificates, so certificate verification is disabled
only for this explicit loopback test. Proxy and curl configuration are disabled;
there are no user-selected URLs, credentials or redirect following. Connect,
request and subprocess timeouts bound the probe. Response staging is private and
removed after checking. The helper bounds its response read and requests curl's
file-size limit; it is not a general-purpose downloader.

Failure diagnostics include only the fixed hostname, sanitized HTTP status,
transport result, bounded byte count and a static classification. Raw response
bodies, stderr, credentials, private service logs and arbitrary input are not
printed. Tests inject success, error status, malformed status, timeout, wrong
content and oversized responses and check that failures remain failures.

PR96's Ubuntu job 129521096773 failed the old combined marker assertion about
61 seconds after HTTPS configuration passed. HTTP isolation and full WordPress
installation had already passed. The original probe captured no HTTP status or
response diagnostics and curl could return success for an HTTP error page. The
timing suggests an upstream timeout, but does not prove its cause. This change
adds evidence and bounds failure; it does not declare the service issue fixed,
weaken isolation checks or certify production TLS/hosting support.
