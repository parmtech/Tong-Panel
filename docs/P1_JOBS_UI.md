# Read-only Jobs screen

The development UI can load job list and detail metadata through the existing
same-origin GET endpoints. It offers manual refresh and keyboard-accessible
detail selection, with explicit loading, empty, access and service-error states.
Responses are bounded and validated; rendered values use text nodes. Access
withdrawal clears previously displayed data. No token input, browser storage,
job mutation or new authentication endpoint is added.

The screen displays jobs available to the current session. SQLite job reads are
requester scoped, including for owners and auditors. The legacy development
journal is tenant scoped and omits requester metadata; the screen does not claim
that its rows are always created by the viewer or fabricate a client-side filter.
The screen requests pages of up to 50 jobs and provides Previous and Next
controls. Each response remains subject to the 256 KiB safety bound. Pages use
ascending job ID order, not creation order; jobs inserted at or before the
current cursor become visible after refresh. Pages are not a snapshot and do not
report a total. The legacy API request without pagination parameters continues
to return the entire list for compatibility and remains unbounded.

The stock development daemon does not configure browser sessions or TLS. The
screen explains the missing session/authentication state until a separately
reviewed embedding configures those capabilities. A completed `validation.noop`
is queue metadata and does not establish that hosting was provisioned.

`tools/ui-browser` contains locked, development-only Playwright and axe tools.
The test fixture is a Go test binary serving real handlers with synthetic SQLite
identities and cookies on loopback TLS; it is not part of the daemon binary and
does not expose a session-issuance endpoint. Browser tests cover actual requester
isolation, auditor and empty views, denied/unconfigured states, stale-data
clearing, keyboard focus, responsive layout and scoped automated accessibility
checks. A 1,500-job synthetic history exercises actual paginated handlers and
forward/backward navigation. Browser tests also inject synthetic responses to exercise non-JSON 401
handling, overlapping refreshes and literal rendering of HTML-looking metadata.
The axe checks cover the Jobs section in light, dark and mobile views; they do
not establish complete panel accessibility or a production
login, deployment or release. Test cookies and screenshots contain only synthetic
fixtures and are not uploaded by the workflow.
The workflow also recompiles the TypeScript into a temporary directory and
compares it with the embedded JavaScript so the two cannot silently diverge.

See `UPDATES.md` for exact executed test evidence and remaining P1 gates.
