-- Bounded requester-owned job history, ordered by its stable opaque ID.
CREATE INDEX jobs_requester_page ON jobs(tenant_id, requested_by, id);
