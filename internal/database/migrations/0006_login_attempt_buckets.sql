-- P1 draft login throttling primitive: hashed account/source keys only.
-- No raw login name, IP address, password or cookie is stored here.
-- Requires stable 32-byte HMAC key provisioned outside SQLite by trusted caller.
CREATE TABLE login_attempt_buckets (
  bucket_sha256 BLOB PRIMARY KEY NOT NULL CHECK(length(bucket_sha256)=32),
  attempts INTEGER NOT NULL CHECK(attempts BETWEEN 1 AND 25),
  window_start_unix INTEGER NOT NULL CHECK(window_start_unix >= 0)
) STRICT;
CREATE INDEX login_attempt_expiry ON login_attempt_buckets(window_start_unix);
