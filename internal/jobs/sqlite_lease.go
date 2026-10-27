package jobs

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNoJobs    = errors.New("no eligible jobs")
	ErrLeaseLost = errors.New("job lease is no longer valid")
)

const (
	maxLeaseDuration = 5 * time.Minute
	maxRetryDelay    = time.Hour
)

// Claim is a short-lived worker capability. Token is returned once and only its
// SHA-256 hash is persisted. It must be kept secret by a future worker.
type Claim struct {
	Job       Job
	WorkerID  string
	Token     string
	ExpiresAt time.Time
}

type sqliteSession interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *SQLiteStore) ClaimNext(workerID string, duration time.Duration) (Claim, error) {
	if s == nil || !safeWorkerID(workerID) || duration < time.Second || duration > maxLeaseDuration || s.maxAttempts < 1 || s.maxAttempts > 10 {
		return Claim{}, errors.New("invalid lease request")
	}
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return Claim{}, err
	}
	token := hex.EncodeToString(raw[:])
	tokenHash := sha256.Sum256(raw[:])
	now := s.now().UTC()
	expires := now.Add(duration)
	var claim Claim
	noJobs := false
	err := withImmediate(s.ctx, s.db, func(q sqliteSession) error {
		var j Job
		var state, created, updated string
		var attempts int
		err := q.QueryRowContext(s.ctx, `SELECT j.id,j.tenant_id,j.kind,j.state,j.idempotency_key,j.created_at,j.updated_at,j.requested_by,j.attempt_count
			FROM jobs j
			JOIN memberships m ON m.tenant_id=j.tenant_id AND m.identity_id=j.requested_by
			JOIN tenants t ON t.id=j.tenant_id AND t.status='active'
			JOIN identities i ON i.id=j.requested_by AND i.disabled=0
			WHERE (j.state='queued' AND COALESCE(julianday(j.available_at),julianday(j.created_at))<=julianday(?))
			   OR (j.state IN ('leased','running') AND COALESCE(julianday(j.lease_until),0)<=julianday(?))
			-- Claim eligible jobs before expired leases that already exhausted their retry budget.
			-- Otherwise the oldest exhausted lease can starve newer runnable jobs.
			ORDER BY CASE WHEN j.attempt_count >= ? THEN 1 ELSE 0 END, j.created_at,j.id LIMIT 1`, dbTime(now), dbTime(now), s.maxAttempts).
			Scan(&j.ID, &j.TenantID, &j.Kind, &state, &j.IdempotencyKey, &created, &updated, &j.RequestedBy, &attempts)
		if errors.Is(err, sql.ErrNoRows) {
			noJobs = true
			return nil
		}
		if err != nil {
			return err
		}
		j.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return fmt.Errorf("invalid job created_at: %w", err)
		}
		j.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
		if err != nil {
			return fmt.Errorf("invalid job updated_at: %w", err)
		}
		if state == Leased || state == Running {
			_, err = q.ExecContext(s.ctx, `UPDATE job_attempts SET status='failed',error_code='lease_expired',finished_at=?
				WHERE id=(SELECT id FROM job_attempts WHERE job_id=? AND status='started' ORDER BY started_at DESC LIMIT 1)`, dbTime(now), j.ID)
			if err != nil {
				return err
			}
			if err := appendNonUserAudit(s.ctx, q, j, "job.lease_expired", "system", "lease-reconciler", now); err != nil {
				return err
			}
		}
		if attempts >= s.maxAttempts {
			_, err = q.ExecContext(s.ctx, `UPDATE jobs SET state='failed',lease_until=NULL,lease_owner=NULL,lease_token_sha256=NULL,updated_at=?
				WHERE id=? AND state=?`, dbTime(now), j.ID, state)
			if err != nil {
				return err
			}
			j.State = Failed
			if err := appendNonUserAudit(s.ctx, q, j, "job.failed", "system", "lease-reconciler", now); err != nil {
				return err
			}
			noJobs = true
			return nil
		}
		attemptID, err := randomID()
		if err != nil {
			return err
		}
		result, err := q.ExecContext(s.ctx, `UPDATE jobs SET state='leased',lease_until=?,lease_owner=?,lease_token_sha256=?,
			attempt_count=attempt_count+1,available_at=NULL,updated_at=? WHERE id=? AND state=?`,
			dbTime(expires), workerID, tokenHash[:], dbTime(now), j.ID, state)
		if err != nil {
			return err
		}
		changed, err := result.RowsAffected()
		if err != nil || changed != 1 {
			return ErrLeaseLost
		}
		_, err = q.ExecContext(s.ctx, `INSERT INTO job_attempts(id,job_id,started_at,status) VALUES(?,?,?,'started')`, attemptID, j.ID, dbTime(now))
		if err != nil {
			return err
		}
		j.State = Leased
		j.UpdatedAt = now
		if err := appendNonUserAudit(s.ctx, q, j, "job.leased", "worker", workerID, now); err != nil {
			return err
		}
		claim = Claim{Job: j, WorkerID: workerID, Token: token, ExpiresAt: expires}
		return nil
	})
	if err != nil {
		return Claim{}, err
	}
	if noJobs {
		return Claim{}, ErrNoJobs
	}
	return claim, nil
}

func (s *SQLiteStore) Start(claim Claim) (Job, error) {
	var job Job
	err := s.withValidClaim(claim, func(q sqliteSession, now time.Time, hash [32]byte) error {
		var err error
		job, err = loadClaimedJob(s.ctx, q, claim, now, hash)
		if err != nil {
			return err
		}
		result, err := q.ExecContext(s.ctx, `UPDATE jobs SET state='running',updated_at=? WHERE id=? AND state='leased'`, dbTime(now), job.ID)
		if err != nil {
			return err
		}
		if !oneRow(result) {
			return ErrLeaseLost
		}
		job.State, job.UpdatedAt = Running, now
		return appendNonUserAudit(s.ctx, q, job, "job.started", "worker", claim.WorkerID, now)
	})
	return job, err
}

func (s *SQLiteStore) Renew(claim Claim, duration time.Duration) (Claim, error) {
	if duration < time.Second || duration > maxLeaseDuration {
		return Claim{}, errors.New("invalid lease duration")
	}
	var renewed Claim
	err := s.withValidClaim(claim, func(q sqliteSession, now time.Time, hash [32]byte) error {
		job, err := loadClaimedJob(s.ctx, q, claim, now, hash)
		if err != nil {
			return err
		}
		expires := now.Add(duration)
		result, err := q.ExecContext(s.ctx, `UPDATE jobs SET lease_until=?,updated_at=? WHERE id=? AND state IN ('leased','running')`, dbTime(expires), dbTime(now), job.ID)
		if err != nil {
			return err
		}
		if !oneRow(result) {
			return ErrLeaseLost
		}
		job.UpdatedAt = now
		renewed = Claim{Job: job, WorkerID: claim.WorkerID, Token: claim.Token, ExpiresAt: expires}
		return appendNonUserAudit(s.ctx, q, job, "job.lease_renewed", "worker", claim.WorkerID, now)
	})
	return renewed, err
}

func (s *SQLiteStore) Complete(claim Claim) (Job, error) {
	var job Job
	err := s.withValidClaim(claim, func(q sqliteSession, now time.Time, hash [32]byte) error {
		var err error
		job, err = loadClaimedJob(s.ctx, q, claim, now, hash)
		if err != nil {
			return err
		}
		if job.State != Running {
			return ErrLeaseLost
		}
		result, err := q.ExecContext(s.ctx, `UPDATE jobs SET state='completed',lease_until=NULL,lease_owner=NULL,lease_token_sha256=NULL,updated_at=? WHERE id=? AND state='running'`, dbTime(now), job.ID)
		if err != nil {
			return err
		}
		if !oneRow(result) {
			return ErrLeaseLost
		}
		if err := finishAttempt(s.ctx, q, job.ID, "completed", "", now); err != nil {
			return err
		}
		job.State, job.UpdatedAt = Done, now
		return appendNonUserAudit(s.ctx, q, job, "job.completed", "worker", claim.WorkerID, now)
	})
	return job, err
}

// Fail retries a failed attempt until maxAttempts is reached. errorCode accepts
// only a small identifier alphabet so sensitive failure details cannot be stored
// in the audit/attempt tables by mistake.
func (s *SQLiteStore) Fail(claim Claim, errorCode string, retryable bool, retryDelay time.Duration) (Job, error) {
	if !safeErrorCode(errorCode) || retryDelay < 0 || retryDelay > maxRetryDelay || s == nil || s.maxAttempts < 1 || s.maxAttempts > 10 {
		return Job{}, errors.New("invalid job failure")
	}
	var job Job
	err := s.withValidClaim(claim, func(q sqliteSession, now time.Time, hash [32]byte) error {
		var attemptCount int
		var err error
		job, attemptCount, err = loadClaimedJobWithAttempt(s.ctx, q, claim, now, hash)
		if err != nil {
			return err
		}
		if job.State != Running {
			return ErrLeaseLost
		}
		next := Failed
		var available any
		action := "job.failed"
		if retryable && attemptCount < s.maxAttempts {
			next = Queued
			available = dbTime(now.Add(retryDelay))
			action = "job.retry_scheduled"
		}
		result, err := q.ExecContext(s.ctx, `UPDATE jobs SET state=?,lease_until=NULL,lease_owner=NULL,lease_token_sha256=NULL,
			available_at=?,updated_at=? WHERE id=? AND state='running'`, next, available, dbTime(now), job.ID)
		if err != nil {
			return err
		}
		if !oneRow(result) {
			return ErrLeaseLost
		}
		if err := finishAttempt(s.ctx, q, job.ID, "failed", errorCode, now); err != nil {
			return err
		}
		job.State, job.UpdatedAt = next, now
		if err := appendNonUserAudit(s.ctx, q, job, action, "worker", claim.WorkerID, now); err != nil {
			return err
		}
		return nil
	})
	return job, err
}

func (s *SQLiteStore) withValidClaim(claim Claim, fn func(sqliteSession, time.Time, [32]byte) error) error {
	if s == nil || !safeWorkerID(claim.WorkerID) || claim.Job.ID == "" || claim.Job.TenantID == "" {
		return ErrLeaseLost
	}
	raw, err := hex.DecodeString(claim.Token)
	if err != nil || len(raw) != 32 {
		return ErrLeaseLost
	}
	hash := sha256.Sum256(raw)
	return withImmediate(s.ctx, s.db, func(q sqliteSession) error {
		return fn(q, s.now().UTC(), hash)
	})
}

func loadClaimedJob(ctx context.Context, q sqliteSession, claim Claim, now time.Time, hash [32]byte) (Job, error) {
	job, _, err := loadClaimedJobWithAttempt(ctx, q, claim, now, hash)
	return job, err
}

func loadClaimedJobWithAttempt(ctx context.Context, q sqliteSession, claim Claim, now time.Time, hash [32]byte) (Job, int, error) {
	var job Job
	var created, updated string
	var attempts int
	err := q.QueryRowContext(ctx, `SELECT j.id,j.tenant_id,j.kind,j.state,j.idempotency_key,j.created_at,j.updated_at,j.requested_by,j.attempt_count
		FROM jobs j
		JOIN memberships m ON m.tenant_id=j.tenant_id AND m.identity_id=j.requested_by
		JOIN tenants t ON t.id=j.tenant_id AND t.status='active'
		JOIN identities i ON i.id=j.requested_by AND i.disabled=0
		WHERE j.id=? AND j.tenant_id=? AND j.lease_owner=? AND j.lease_token_sha256=?
		  AND j.state IN ('leased','running') AND julianday(j.lease_until)>julianday(?)`,
		claim.Job.ID, claim.Job.TenantID, claim.WorkerID, hash[:], dbTime(now)).
		Scan(&job.ID, &job.TenantID, &job.Kind, &job.State, &job.IdempotencyKey, &created, &updated, &job.RequestedBy, &attempts)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Job{}, 0, ErrLeaseLost
		}
		return Job{}, 0, err
	}
	job.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return Job{}, 0, fmt.Errorf("invalid job created_at: %w", err)
	}
	job.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	return job, attempts, err
}

func finishAttempt(ctx context.Context, q sqliteSession, jobID, status, errorCode string, now time.Time) error {
	result, err := q.ExecContext(ctx, `UPDATE job_attempts SET status=?,error_code=NULLIF(?,'') ,finished_at=?
		WHERE id=(SELECT id FROM job_attempts WHERE job_id=? AND status='started' ORDER BY started_at DESC LIMIT 1) AND status='started'`,
		status, errorCode, dbTime(now), jobID)
	if err != nil {
		return err
	}
	if !oneRow(result) {
		return errors.New("active job attempt not found")
	}
	return nil
}

func activeRequester(ctx context.Context, q sqliteSession, tenant, requester string) error {
	var found int
	err := q.QueryRowContext(ctx, `SELECT 1 FROM memberships m
		JOIN tenants t ON t.id=m.tenant_id AND t.status='active'
		JOIN identities i ON i.id=m.identity_id AND i.disabled=0
		WHERE m.tenant_id=? AND m.identity_id=?`, tenant, requester).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("requester is not active in tenant")
	}
	return err
}

func appendAudit(ctx context.Context, q sqliteSession, job Job, action string, now time.Time) error {
	id, err := randomID()
	if err != nil {
		return err
	}
	_, err = q.ExecContext(ctx, `INSERT INTO audit_events(id,tenant_id,actor_identity_id,action,subject_id,request_id,created_at)
		VALUES(?,?,?,?,?,?,?)`, id, job.TenantID, job.RequestedBy, action, job.ID, id, dbTime(now))
	return err
}

// safeWorkerID constrains worker references to log-safe stable service IDs.
// This is not authentication; a future registered worker identity is required.
func safeWorkerID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, c := range value {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' {
			continue
		}
		return false
	}
	return true
}

// appendNonUserAudit records the actor that actually applied a queue
// transition. In particular, the job requester MUST NOT be attributed with
// lease/start/renew/complete/fail operations performed by a worker or scheduler.
// Only the actor ID (not a lease token) may appear in persisted audit rows.
func appendNonUserAudit(ctx context.Context, q sqliteSession, job Job, action, kind, actorRef string, now time.Time) error {
	if (kind != "worker" && kind != "system") || !safeWorkerID(actorRef) {
		return errors.New("invalid non-user audit actor")
	}
	id, err := randomID()
	if err != nil {
		return err
	}
	_, err = q.ExecContext(ctx,
		`INSERT INTO audit_events(id,tenant_id,actor_identity_id,actor_kind,actor_ref,action,subject_id,request_id,created_at)
		 VALUES(?,?,NULL,?,?,?,?,?,?)`,
		id, job.TenantID, kind, actorRef, action, job.ID, id, dbTime(now))
	return err
}

func randomID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func dbTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000000000Z07:00")
}

func oneRow(result sql.Result) bool {
	changed, err := result.RowsAffected()
	return err == nil && changed == 1
}

func safeErrorCode(value string) bool {
	if value == "" || len(value) > 64 || strings.TrimSpace(value) != value {
		return false
	}
	for _, c := range value {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '.' || c == '_' || c == '-' {
			continue
		}
		return false
	}
	return true
}

func withImmediate(ctx context.Context, db *sql.DB, fn func(sqliteSession) error) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	defer func() { _, _ = conn.ExecContext(context.Background(), "ROLLBACK") }()
	if err := fn(conn); err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, "COMMIT")
	return err
}
