-- One-time DNS TXT proof tokens are stored only as SHA-256 digests.
ALTER TABLE site_domains ADD COLUMN verification_token_sha256 TEXT
    CHECK (verification_token_sha256 IS NULL OR length(verification_token_sha256) = 64);
ALTER TABLE site_domains ADD COLUMN verification_expires_at INTEGER
    CHECK (verification_expires_at IS NULL OR verification_expires_at > 0);

CREATE TRIGGER site_domains_verification_state_insert
BEFORE INSERT ON site_domains
WHEN NOT (
    (NEW.ownership_state = 'pending' AND NEW.verification_token_sha256 IS NULL AND NEW.verification_expires_at IS NULL)
    OR (NEW.ownership_state = 'pending' AND NEW.verification_token_sha256 IS NOT NULL AND NEW.verification_expires_at IS NOT NULL)
    OR (NEW.ownership_state = 'verified' AND NEW.verification_token_sha256 IS NULL AND NEW.verification_expires_at IS NULL)
)
BEGIN
    SELECT RAISE(ABORT, 'invalid domain verification state');
END;

CREATE TRIGGER site_domains_verification_state_update
BEFORE UPDATE OF ownership_state, verification_token_sha256, verification_expires_at ON site_domains
WHEN NOT (
    (NEW.ownership_state = 'pending' AND NEW.verification_token_sha256 IS NULL AND NEW.verification_expires_at IS NULL)
    OR (NEW.ownership_state = 'pending' AND NEW.verification_token_sha256 IS NOT NULL AND NEW.verification_expires_at IS NOT NULL)
    OR (NEW.ownership_state = 'verified' AND NEW.verification_token_sha256 IS NULL AND NEW.verification_expires_at IS NULL)
)
BEGIN
    SELECT RAISE(ABORT, 'invalid domain verification state');
END;
