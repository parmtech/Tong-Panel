-- Retain server-authored evidence of successful domain proof completion.
-- Existing verified rows deliberately remain unknown (all NULL); migration
-- time is not evidence of domain control, and no proof token is retained.
ALTER TABLE site_domains ADD COLUMN ownership_observed_at INTEGER
    CHECK (ownership_observed_at IS NULL OR ownership_observed_at > 0);
ALTER TABLE site_domains ADD COLUMN ownership_verified_at INTEGER
    CHECK (ownership_verified_at IS NULL OR ownership_verified_at > 0);
ALTER TABLE site_domains ADD COLUMN ownership_verification_id TEXT
    CHECK (ownership_verification_id IS NULL OR
        (length(ownership_verification_id) = 32 AND ownership_verification_id NOT GLOB '*[^0-9a-f]*'));

CREATE UNIQUE INDEX site_domains_verification_evidence_id
    ON site_domains(ownership_verification_id) WHERE ownership_verification_id IS NOT NULL;

CREATE TRIGGER site_domains_evidence_state_insert
BEFORE INSERT ON site_domains
WHEN NOT (
    (NEW.ownership_observed_at IS NULL AND NEW.ownership_verified_at IS NULL AND NEW.ownership_verification_id IS NULL)
    OR (NEW.ownership_state = 'verified' AND NEW.ownership_observed_at IS NOT NULL
        AND NEW.ownership_verified_at IS NOT NULL AND NEW.ownership_verification_id IS NOT NULL
        AND NEW.ownership_observed_at <= NEW.ownership_verified_at)
)
BEGIN
    SELECT RAISE(ABORT, 'invalid domain verification evidence');
END;

CREATE TRIGGER site_domains_evidence_state_update
BEFORE UPDATE OF ownership_state, ownership_observed_at, ownership_verified_at, ownership_verification_id ON site_domains
WHEN NOT (
    (NEW.ownership_observed_at IS NULL AND NEW.ownership_verified_at IS NULL AND NEW.ownership_verification_id IS NULL)
    OR (NEW.ownership_state = 'verified' AND NEW.ownership_observed_at IS NOT NULL
        AND NEW.ownership_verified_at IS NOT NULL AND NEW.ownership_verification_id IS NOT NULL
        AND NEW.ownership_observed_at <= NEW.ownership_verified_at)
)
BEGIN
    SELECT RAISE(ABORT, 'invalid domain verification evidence');
END;
