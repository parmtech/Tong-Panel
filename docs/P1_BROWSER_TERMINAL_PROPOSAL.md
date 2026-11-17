# V1 browser terminal — implementation and acceptance proposal

**Status: requested feature; NOT IMPLEMENTED; architecture/security decisions pending.** This document prepares a real interactive terminal, not a simulated command prompt. It adds no terminal UI, endpoint, WebSocket, dependency, shell, PTY, broker, database migration or executable job. V1 inclusion is a product request; delivery and safety acceptance are not established by writing this proposal.

## Product scope and decisions

The first proposed implementation is a terminal for one authorized tenant runtime, running as that runtime's isolated non-root Unix identity. A user chooses an existing server-authorized target, explicitly connects, sees the actual target/identity and session lifetime, uses a real PTY and can disconnect. This is arbitrary interactive execution within a reviewed workload boundary; it must not be described as read-only or a collection of allowlisted shell commands. A tenant filesystem path or changed working directory alone does not provide isolation.

The user has not yet selected tenant runtime access versus platform administrator/root console access. These are different security products. The recommended first scope below remains a proposal, not an accepted interpretation of that decision.

| Decision | Proposed starting point | Required accountable review |
|---|---|---|
| Target and privilege | Tenant runtime only; no root, panel-service identity, host-agent identity, Docker socket, other tenant or arbitrary remote SSH target | Product owner selects intended console; security owner accepts workload isolation boundary |
| Roles | New terminal permissions, default denied to every existing role until an explicit role/target matrix is accepted; do not infer terminal rights from sites:read, files:read or owner/admin labels | Owner/security approve each role, tenant boundary and administrative support access |
| Authentication | Actual TLS plus authoritative cookie session; no static development bearer tokens; recent separately verifiable step-up authentication before session creation | Authentication owner specifies assurance claim, maximum age, enrollment/recovery and revocation behavior; existing MFA primitives alone are not that claim |
| Deployment | Default disabled, disposable Linux lab first; stock plaintext loopback daemon stays incapable of opening terminals | Maintainer approves supported lab profile and later release gate |
| Execution mechanism | Separate non-root broker already confined to a verified runtime; an authenticated typed lifecycle protocol, not shell execution in HTTP handlers | ADR-002/004/007 review; any privileged identity transition is a separate narrow reviewed provisioner decision |
| Persistence | No detached/resumable shell in first scope; disconnect terminates its entire process scope | Owner accepts this user experience and security tradeoff |
| Audit/privacy | Lifecycle metadata only; no keystrokes, screen transcripts, command arguments, environment, tickets or secrets in logs by default | Security/privacy owner approves retention, access and any future optional recording |
| Supply chain | Evaluate a browser emulator, WebSocket library and Linux PTY library; select exact versions only after provenance, license, advisory and packaging review | Dependency/release owner records pins, checksums, notices and SBOM |

Acceptance must be recorded in the existing ADR process. [ARCHITECTURE.md](ARCHITECTURE.md) currently forbids HTTP endpoints directly executing shell strings. A terminal design needs an explicit amendment explaining that interactive bytes terminate inside an independently confined tenant workload, while the HTTP/API process and privileged agent never interpret them. The amendment must preserve the prohibition on arbitrary privileged shell operations. A platform/root console would need a separate threat model and is outside the proposed first implementation.

## Existing implementation boundaries

| Existing entrypoint | What is reusable | What it does not establish |
|---|---|---|
| [cmd/tongd/main.go](../cmd/tongd/main.go), [listen_address.go](../cmd/tongd/listen_address.go), [development_server.go](../cmd/tongd/development_server.go) | Explicit development configuration, numeric IPv4-loopback listener, bounded shutdown | TLS, public deployment, terminal process supervision or privileged actions |
| [internal/api/server.go](../internal/api/server.go), [session_cookie.go](../internal/api/session_cookie.go) | Protected HTTP dispatch; actual TLS, cookie ambiguity, membership, same-origin and mutation CSRF checks | Long-lived WebSocket authorization; normal GET authentication does not enforce the proposed terminal upgrade Origin policy, and bearer fallback must not authorize terminal routes |
| [internal/auth/sqlite_sessions.go](../internal/auth/sqlite_sessions.go), [session_read_proof.go](../internal/auth/session_read_proof.go), [sqlite_session_lifecycle.go](../internal/auth/sqlite_session_lifecycle.go) | Authoritative session hashes, current membership, idle/absolute expiry, revocation and rotation primitives | Terminal-specific step-up assurance, attach ticket, revocation notification or continuous authorization |
| [internal/rbac/policy.go](../internal/rbac/policy.go) | Deny-by-default action checks and tenant scoping | No terminal action exists; none is implicitly granted |
| [internal/sites/authorized_read.go](../internal/sites/authorized_read.go) | Example of same-statement metadata/current-membership/session checking | A site record or verified domain is not an executable-runtime identity or shell permission |
| [internal/jobs/noop_worker.go](../internal/jobs/noop_worker.go), [store.go](../internal/jobs/store.go), [sqlite.go](../internal/jobs/sqlite.go) | Existing metadata-only queue, explicit kind checks | Only validation.noop is accepted. Do not disguise PTY execution as that job kind or widen the no-op worker |
| [internal/jobs/sqlite_lease.go](../internal/jobs/sqlite_lease.go) | Reference for short-lived hashed capabilities and authoritative state transitions | A terminal cannot be restarted by replaying a failed queue job; session ownership/fencing needs its own reviewed lifecycle |
| [internal/agent/agent_linux.go](../internal/agent/agent_linux.go) | Strict framing and Linux peer-UID checks as protocol examples | The existing agent is non-privileged and ping-only; never add a shell verb to it |
| [cmd/tongd/ui.go](../cmd/tongd/ui.go), [web/src/app.ts](../web/src/app.ts), [cmd/tongd/ui/index.html](../cmd/tongd/ui/index.html) | Existing embedded same-origin UI, CSP and source/emitted-JavaScript convention | Browser isolation for terminal keystrokes; any compromised same-page script would be highly sensitive |
| [cmd/tongd/jobs_browser_fixture_test.go](../cmd/tongd/jobs_browser_fixture_test.go), [tools/ui-browser/jobs.spec.mjs](../tools/ui-browser/jobs.spec.mjs) | Disposable real TLS/session browser fixture, origin restriction, keyboard/accessibility assertions | PTY, hostile-terminal-output handling or OS containment acceptance |

No terminal operation should be added to [api/openapi.json](../api/openapi.json) as implemented until its actual handlers exist. Proposed file names below are planning entrypoints, not existing capabilities.

## Proposed end-to-end lifecycle

1. An HTTPS-only authenticated target-list read returns server-selected opaque runtime IDs available under the user's current explicit terminal permission. No browser-supplied UID, shell path, environment, root directory, host address or namespace identifier is trusted. Recheck target identity/revision against the authoritative tenant runtime mapping before launch.
2. A proposed POST /api/v1/terminal/sessions requires the cookie, current membership/role, accepted step-up proof, exact configured HTTPS Origin and CSRF header. Strict JSON accepts only target_id, columns and rows. It creates short-lived metadata and a 256-bit single-use attach ticket; only its digest is stored. The response has no-store headers, an opaque session ID and the one-time ticket. This POST does not launch a process.
3. A proposed same-origin WSS /api/v1/terminal/sessions/{id}/stream upgrade separately requires the actual TLS connection, cookie, exact approved Origin and current terminal authorization. Reject absent/null/multiple/cross-origin Origin, mixed bearer credentials and forwarded-protocol/origin spoofing. Do not rely on CORS or SameSite alone. Production proxy TLS/origin trust must be explicitly designed before proxy deployment.
4. The first application frame must consume the attach ticket atomically within 30 seconds. Bind it to the authenticated browser session, identity, tenant, target revision, terminal ID and connection generation. Send it in the frame, never a URL, cookie replacement, WebSocket subprotocol, log or persistent browser storage. Expire unopened connections after five seconds. Invalid/replayed tickets must fail before PTY allocation.
5. After both authoritative checks and ticket consumption, the broker creates one process scope inside the approved runtime isolation profile. Server constants select the permitted installed shell/argv and a minimal allowlisted environment. No shell command string is built from HTTP fields. Interactive bytes flow only to that scope's PTY, not through the panel process or privileged agent. Initially permit only one attachment, with no reconnect/replay of buffered input.
6. Authorize each input/control frame against current session/target state, and recheck active idle/absolute expiry and membership on a bounded periodic schedule while streaming output. Proposed maximum revocation-to-close delay is five seconds. Close immediately on a failed check or unavailable authoritative store; no fail-open grace period. Terminal traffic cannot extend a revoked session or its absolute expiry.
7. A proposed CSRF-protected DELETE closes the session; browser disconnect, logout/revocation, target deletion, deadline, backpressure, malformed frame or broker/API shutdown must also stop input and terminate the process group/cgroup. Remove the PTY and join/reap its processes within a bounded deadline. Do not leave a child shell running after the UI reports Disconnected.
8. On service restart, reconcile all recorded active sessions to closed and terminate orphaned scopes before accepting new launches. Monotonic connection/session generations must prevent an old worker from accepting input after replacement. If cleanup cannot be proven, fail closed for that target and report a generic actionable failure without sensitive process details.

Proposed state transitions: prepared -> attaching -> active -> terminating -> closed; preparation/attach errors become expired or failed, without a process. Persist closure reason and observed exit status separately. An exit event from a shell is not proof that all descendants were reaped. A process supervisor's authoritative cleanup result is required.

## Proposed protocol and resource limits

These numbers are initial testable budgets, not measured support promises; record revisions during lab tests.

| Resource | Proposed bound and failure behavior |
|---|---|
| Unused attach ticket | 30-second lifetime, one consumption, five-second first-frame timeout |
| Active sessions | One per identity/target and two per tenant by default; host budget separately configured |
| Lifetime | Five minutes without user input; fifteen minutes absolute, capped by browser-session expiry; output alone does not reset idle |
| Input frame | Binary PTY data at most 16 KiB; no implicit shell parsing or input replay |
| Control frame | Strict versioned JSON at most 1 KiB; resize only with integer columns 20–240, rows 5–100; reject unknown/duplicate fields/types |
| Output queue | At most 256 KiB per attachment; bounded write deadline of five seconds, then terminate rather than buffer indefinitely |
| Browser scrollback | At most 2,000 lines in memory; clear on close/logout; no localStorage/sessionStorage/IndexedDB/transcript persistence |
| Heartbeat | 15-second ping and 30-second liveness ceiling; heartbeat does not extend authorization or idle |
| Cleanup | Stop writes immediately; graceful termination budget two seconds, forced scope kill/reap budget five seconds total |
| Process resources | Per-runtime memory/process/CPU/disk/network profile must be selected and tested before launch; no unbounded defaults |

Maintain one bounded writer and a separately bounded reader per connection; resize/control traffic cannot bypass input quotas. Terminal bytes are untrusted binary data: avoid JSON round-tripping or UTF-8 conversions that corrupt PTY streams. Never inject output into innerHTML, evaluate it, build links automatically or expose it to unrelated UI components.

## Browser and transport security

A production browser terminal exposes keystrokes and output to its JavaScript context. Prefer a dedicated terminal document with a stricter CSP, pinned self-hosted assets, no analytics/third-party scripts and no token-entry/storage flow. A same-origin iframe alone does not isolate it from compromised same-origin scripts; origin separation, if selected, needs a separate authentication/Origin design. Do not claim an iframe is a security boundary without that review.

Evaluate an emulator such as xterm.js with clipboard-write, automatic hyperlinks/navigation, title changes, image/file-transfer and unreviewed custom escape handlers disabled. Explicitly test OSC 52 clipboard attempts, OSC 8 links, title spoofing, malformed/incomplete sequences, very long lines and output floods. Clipboard paste must be explicit, bounded and confirm multi-line input; never transmit clipboard contents on focus. Provide keyboard focus/escape behavior, connection/target/lifetime labels, screen-reader guidance and an explicit disconnect action. A terminal widget alone is not an authorization or sandbox boundary.

The [xterm.js security guide](https://xtermjs.org/docs/guides/security/) identifies privileged execution and same-page JavaScript access to terminal data as integration risks and recommends restricted execution privileges and trustworthy assets. The [WebSocket origin guidance](https://pkg.go.dev/github.com/gorilla/websocket#hdr-Origin_Considerations) explains that browser WebSocket connections need server-side origin checks; selecting a library does not replace an explicit deployment allowlist. These are research references, not dependency selections or security certification.

## Phased implementation and acceptance

| Stage | Proposed implementation entrypoints | Required evidence before advancing |
|---|---|---|
| T0 — accepted scope/threat model | This proposal, ADR register, SECURITY/THREAT_MODEL; no executable paths | Named decisions above accepted/revised; root/tenant scope explicit; terminal:connect/close role matrix; auth assurance and isolation owners; current P0 gates recorded honestly |
| T1 — authorization/lifecycle, no process | Proposed internal/terminal/session.go and tests; schema migration; explicit API handlers | Exhaustive roles/tenant/target ownership; ticket replay/race and clock expiry; CSRF/Origin/TLS negatives; current role/session changes between create/attach; no PTY allocated on any failure; fuzz framing/resize; audit contains no secrets |
| T2 — transport harness, no shell | Proposed terminal transport tests with a bounded in-memory byte source, not a shipped mock UI | Two-way binary correctness; fragmented frames; malformed control; output backpressure; disconnect/timeout; unknown versions; wrong Origin; no unbounded goroutines/memory; default-disabled daemon and static-token denial |
| T3 — actual PTY in disposable Linux guests | Proposed separate terminal broker + Linux PTY tests; Ubuntu/Debian lab profiles | Correct non-root UID/groups/runtime namespace; process-group/cgroup cleanup; revoked session while output streams; kill/restart API/broker/client; no orphan process; CPU/memory/process limits; cross-tenant files/process/secrets denied; attempted root/socket/mount/namespace escapes denied |
| T4 — real browser terminal | Proposed dedicated terminal page/TypeScript entrypoint and pinned reviewed emulator assets | Real TLS session and real confined PTY round trip; keyboard/resize/Unicode; explicit paste; clipboard/OSC/link/XSS/flood tests; no credential leakage; accessibility/mobile layout; actual revocation disconnect within bound; no reconnect input replay |
| T5 — review and V1 acceptance | OpenAPI/stream protocol docs, packaging/service policy, SBOM/notices and CI | Independent security review; threat remediation; supported-OS recovery and cleanup evidence; production auth/proxy/TLS decisions; owner acceptance of exact privileges and limitations; no V1/phase-complete claim beforehand |

T1/T2 are protocol/security harnesses, not user-facing simulations. T3/T4 deliberately introduce real execution only after their prerequisites; none is authorized or implemented by this document. Broadening validation.noop or adding a shell verb to the ping-only agent is never an intermediate shortcut.

## Handoff

Current work is documentation only. No terminal dependency/version, schema, permission or route is selected in code, and there is no working terminal to use. The next actionable task is owner/security review of T0: choose tenant terminal versus a separately specified administrator console, approve an execution isolation profile and the explicit role/authentication matrix, then implement T1 with failing security tests before adding any PTY. Existing terminal availability must continue to be described as not implemented.
