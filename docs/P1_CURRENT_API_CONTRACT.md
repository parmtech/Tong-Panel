# Current development API contract

`api/openapi.json` records the implemented control-plane handlers as OpenAPI
3.0.3: 17 path templates and 23 operations. It is a snapshot of the unmerged
development implementation, not an approved production interface, stable release,
security assessment or complete P1 acceptance. It adds no routes or behavior.

The documented server is the default `http://127.0.0.1:8090` developer
listener. `TONG_DEV_LISTEN_ADDR` permits a numeric IPv4 loopback address and port
1–65535; it does not permit hostnames, wildcards, nonloopback addresses or IPv6.
The listener remains plaintext and development only. See
`docs/P1_DEVELOPMENT_BIND.md` for configuration and limits. Embedded UI and
static assets are outside this control-plane contract. No OpenAPI-document or
Swagger-UI HTTP route is invented by checking in the artifact.

## Implemented surface

- Public GET process liveness and experimental version identification.
- Authenticated GET principal, and conditional POST browser-session logout.
- Job list/create/read and SQLite-only queued cancellation.
- Conditional tenant-relative Files list/read/new-file upload/delete.
- Bounded site catalog list, site/domain metadata creation/read, alias changes and DNS TXT proof lifecycle.
- Read-only Nginx text preview, with separately configured hosting paths.
- Site-bound database-name metadata creation/list/read.

There is no public password login, MFA/recovery endpoint, cookie/session issuance
or rotation, general worker executor, site update/delete, database provisioning
or deletion, TLS issuance, privileged agent action or host-configuration apply.
Site/database writes describe catalog rows, not externally completed operations.
The Nginx preview is text that has not been installed or applied.

## Backend and credential boundaries

The daemon can configure a development bearer key and either its JSON journal
or an opted-in SQLite job store. SQLite mode also wires the site catalog and DNS
TXT resolver and requires an already existing identity/membership. It does not
silently create an account. An independently opted-in no-op worker can advance
only `validation.noop` metadata; job acceptance alone is not external execution.

`api.Server` additionally has optional session, Files, quota, mutation-barrier and
hosting-path dependencies. The current daemon does **not** configure those
backends or a TLS listener. Their declared handlers remain fail-closed unless a
trusted embedding configures them; an OpenAPI entry does not enable them. Each
operation records this distinction in `x-tong-backend-gate`.

OpenAPI security alternatives cannot completely encode the handler's rules:

- A recognized session cookie takes precedence and is never downgraded to a
  bearer token when invalid. A nonempty Authorization header mixed with that
  cookie is unauthorized; duplicate named cookies are rejected.
- Browser-session authentication requires actual connection TLS. Forwarded
  protocol/host headers are not TLS evidence. The cookie name is
  `__Host-tong_session`; no route here issues it.
- Cookie-authenticated mutations require a valid `X-CSRF-Token` and exactly one
  HTTPS Origin matching the actual Host, including explicit port, with no
  path/user information/query/fragment. Origin matching is not inferred from an
  OpenAPI security declaration. Developer bearer mutations do not require CSRF.
- Logout requires a separately configured session backend, direct TLS and the
  browser cookie. A bearer token alone cannot log out. Success expires the cookie
  and returns an empty 204; reuse of a revoked cookie is unauthorized.

Authentication runs before method, permission and backend checks. With no auth
backend, protected requests return 503 `auth_unconfigured`; invalid credentials
normally return 401. An unsupported method can therefore fail authentication
before reaching 405. HEAD/OPTIONS/CORS methods are not implicitly added.

## Ownership, representations and limits

Tenant context is derived from authentication; a request cannot select another
tenant or user to gain authority. SQLite job reads are also requester-scoped,
including for owner/admin. Jobs add current mutation-role checks inside their
transactions. The general protected wrapper refreshes a SQL principal before
entering a handler; this does not establish atomic role revalidation for every
site or Files operation.

Jobs/sites reads permit the current five roles, while mutation permits owner,
admin, reseller and customer. Files **reads as well as writes exclude auditor**.
The operation's `x-tong-rbac-action` names the actual policy action.

Responses preserve current case-sensitive JSON spellings. Principal, site,
domain, proof and database structs use uppercase Go field names such as `UserID`,
`SiteID`, `OwnershipState` and `ExpiresAt`. Job/File responses and request objects
use their declared lowercase or snake_case tags. Site/database/proof times are
Unix seconds; job times are date-time strings. Empty journal job lists are `null`;
SQLite own-job and site-database lists return arrays. Job `requested_by` is omitted
by the JSON journal and present in SQLite mode.

`GET /api/v1/jobs` supports opt-in paging when `limit` or `after` occurs in the
query. A paged request accepts only those two keys, once each. `limit` is a
decimal integer from 1 to 100 and defaults to 50; a supplied `after` must be a
nonempty 32-character lowercase hexadecimal job ID. Invalid, duplicate, unknown
or malformed page parameters return 400 `invalid_page`. Pages return
`{"jobs":[],"next_after":""}` with at most the requested number of jobs, sorted
by ascending ID. A nonempty `next_after` is the final returned ID and means more
visible jobs remain. It is a position, not authority to read another requester
or tenant. Each page rechecks the caller's current access.

Pages are live reads rather than a frozen history: inserts before the cursor
appear only after refreshing from the beginning. No-query and unrelated-query-only
legacy GETs still return the original unbounded list, including journal `null`
when empty. POST query behavior is unchanged. The response's disjoint `oneOf`
schemas distinguish the legacy list from a bounded page; paging does not enable
job mutations, cookie/session issuance or production authentication.

`GET /api/v1/sites` returns a bounded tenant catalog page of site ID, primary
domain and ownership state. The default and maximum `limit` are 50; `after` is
an exclusive site ID boundary. Only these nonempty parameters are accepted,
once each; invalid queries return 400 `invalid_page`. `next_after` is empty on
the final page, otherwise the last returned site ID. The boundary does not
authorize another tenant. ID ordering is live, not a snapshot, so refresh to
observe inserts before the cursor. Catalog entries do not imply provisioned
hosting. Site summary fields use snake_case. Site-list reads recheck the
active identity, tenant and membership in the catalog database in the same
statement as the page. Cookie reads also recheck session revocation and expiry
at the catalog snapshot; the catalog and session store must share the same
database pool. An already-started snapshot can finish after a later revocation.
Membership-only bearer denial returns 403; an invalid cookie session returns 401.

Only job creation requires `Idempotency-Key`: use 8–128 ASCII bytes. Identical
retries return the original job with 202; tenant-scoped reuse for different work
conflicts, and SQLite also binds the requester. Current mutation permission is
still required for an idempotent retry. Site/domain/database metadata creation
does not implement this header; duplicate claims return their documented conflict.

Site creation rechecks current write membership, and for cookie callers the
session, under the same SQLite writer lock as the site, domain claims and
`site.created` audit event. This is declarative metadata, not host provisioning.
Other site mutation endpoints have not yet received this transaction guard.

JSON writes require the exact Content-Type string `application/json`; binary
uploads require exact `application/octet-stream`. Parameters such as `charset`
are rejected. Request byte caps are recorded in `x-max-request-bytes`: jobs 2048,
site creation 16384, alias/proof-completion/database creation 4096, and file upload
16777216. File reads have the same 16 MiB response cap. Unknown-length uploads
reserve the full cap before reading. Upload creates a new file without replacing
an existing path. A zero-byte file is allowed with the required media type.

Canonical request names are documented, but Go's JSON decoder also recognizes
case-insensitive matches and processes duplicate members with its normal
last-value behavior. Unknown fields and additional top-level JSON values are
rejected. The contract does not claim duplicate-member rejection. Begin-proof,
logout and delete operations do not require parsed JSON request bodies.

Files queries accept a single `path`, and directory listing additionally accepts
a single `limit` from 1 to 1000 (default 200). Unknown/duplicate query keys fail.
The limit is an entry ceiling, not a paginated/truncated result. Site collection
POST rejects nonempty queries with 404; site resources reject them with 400. Other
core/job handlers generally ignore query strings.

## Errors and evidence limits

JSON failures have the shared `{error:{code,message}}` schema. `message` is the
standard HTTP status text; raw internal errors are not returned. Response-level
`x-error-codes` records the applicable codes in addition to the shared schema.
JSON responses declare no-store/nosniff headers, and API handler responses include
a generated request ID. This ID is not the separately generated audit request ID.
Native ServeMux unknown-path and canonical-redirect responses can be text or
redirects and are outside the declared operations.

A 503 `file_durability_unconfirmed` or `quota_unavailable` can follow visible
filesystem publication/deletion. It is not a guarantee that nothing changed,
and the new-file endpoint must not be retried as an overwrite operation.

Machine-readable parsing and actual-handler response samples are evidence about
this contract snapshot. They do not replace deployment/TLS, race/abuse, tenant
isolation, Files durability or external security validation. Optional backend
fixtures must be identified as such; they do not prove the stock daemon exposes
that capability. Keep the API, security policy and contract synchronized as those
separately reviewed capabilities change.
