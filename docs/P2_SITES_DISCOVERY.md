# Read-only site discovery

The development panel can request bounded site catalog summaries for the
authenticated tenant and copy a selected ID into the Nginx preview form.
The list includes the primary domain ownership state; it does not show aliases
or establish that a site is provisioned, a service is running, or configuration
has been applied.

The API uses ascending binary site-ID order and a limit of at most 50. The
cursor is an ordering boundary, not an authorization token. The principal's
tenant and `sites:read` permission determine access. Pages are not a snapshot:
records inserted at or before the current boundary appear after refresh.
The browser validates and bounds responses and renders values as text.
The site-list query checks the active tenant, enabled identity, current
membership and permitted role in the same SQLite statement that selects the
page. This check uses the catalog database for every HTTP caller, including
developer tokens. Session validation remains a separate operation: session
revocation racing after authentication is not covered by this query. Other
site endpoints have not been changed by this read-specific guard.

The stock development daemon does not configure public browser login or TLS;
browser tests use synthetic cookie sessions and a loopback TLS fixture. Site
creation, host changes and production deployment are outside this feature.
