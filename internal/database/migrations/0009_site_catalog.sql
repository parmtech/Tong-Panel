-- Declarative site/domain catalogue only; no host service or ownership proof is implied.
CREATE TABLE sites (
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    site_id TEXT NOT NULL,
    document_root TEXT NOT NULL CHECK (document_root = 'sites/' || site_id || '/public'),
    created_at INTEGER NOT NULL CHECK (created_at > 0),
    PRIMARY KEY (tenant_id, site_id)
) STRICT;

CREATE TABLE site_domains (
    domain TEXT NOT NULL PRIMARY KEY COLLATE NOCASE CHECK (domain COLLATE BINARY = lower(domain) COLLATE BINARY),
    tenant_id TEXT NOT NULL,
    site_id TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('primary', 'alias')),
    ownership_state TEXT NOT NULL DEFAULT 'pending'
        CHECK (ownership_state IN ('pending', 'verified')),
    FOREIGN KEY (tenant_id, site_id)
        REFERENCES sites(tenant_id, site_id) ON DELETE CASCADE
) STRICT;

CREATE UNIQUE INDEX site_domains_one_primary
    ON site_domains(tenant_id, site_id) WHERE kind = 'primary';

CREATE INDEX site_domains_site
    ON site_domains(tenant_id, site_id);
