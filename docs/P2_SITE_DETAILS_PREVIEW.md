# Site catalog details preview

The Sites list can open read-only details for a selected site through the existing `GET /api/v1/sites/{siteID}` handler. Each request is authorized by the server for the current tenant and session. The panel displays the tenant-relative document root and recorded primary/alias ownership states as text. These are catalog metadata, not evidence of current DNS control, installed hosting or a certificate.

The client limits response bodies to 16 KiB, validates the selected site and returned fields, and ignores canceled or stale responses. Closing details, refreshing or paging the list, tab hiding and focus loss clear the panel. An explicit 401/403 clears the visible site list and generated Nginx preview as well. As with other read-only previews, the client cannot detect an unseen session change while the page stays focused; the server checks authorization on every request. No site, DNS or host mutation is performed.
