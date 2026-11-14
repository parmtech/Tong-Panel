-- Declarative MariaDB database ownership metadata only; no DB account, secret, or remote service action.
CREATE TABLE site_databases (
    tenant_id TEXT NOT NULL,
    site_id TEXT NOT NULL,
    database_id TEXT NOT NULL
        CHECK (length(database_id) BETWEEN 1 AND 63
            AND database_id NOT GLOB '*[^A-Za-z0-9_-]*'),
    database_name TEXT NOT NULL UNIQUE
        CHECK (length(database_name) = 29
            AND substr(database_name, 1, 3) = 'tp_'
            AND database_name NOT GLOB '*[^a-z2-7_]*'),
    username TEXT NOT NULL UNIQUE
        CHECK (length(username) = 29
            AND substr(username, 1, 3) = 'tu_'
            AND username NOT GLOB '*[^a-z2-7_]*'),
    created_at INTEGER NOT NULL CHECK (created_at > 0),
    PRIMARY KEY (tenant_id, site_id, database_id),
    FOREIGN KEY (tenant_id, site_id)
        REFERENCES sites(tenant_id, site_id) ON DELETE CASCADE
) STRICT;

CREATE INDEX site_databases_site
    ON site_databases(tenant_id, site_id);
