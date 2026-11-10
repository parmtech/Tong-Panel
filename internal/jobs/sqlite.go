package jobs

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// SQLiteStore persists no-op jobs and their bounded lease lifecycle in the shared
// metadata DB. Its opt-in development worker advances only validation.noop;
// neither the store nor that worker executes host operations.
type SQLiteStore struct {
	db          *sql.DB
	ctx         context.Context
	now         func() time.Time
	maxAttempts int
}

func NewSQLiteStore(db *sql.DB) (*SQLiteStore, error) {
	if db == nil {
		return nil, errors.New("missing SQLite database")
	}
	return &SQLiteStore{db: db, ctx: context.Background(), now: time.Now, maxAttempts: 3}, nil
}

// CreateFor records a job and its requesting identity in one transaction. An
// identical retry returns the original job; a key reused for different work or
// by another identity fails with ErrConflict.
func (s *SQLiteStore) CreateFor(tenant, requester, kind, key string) (Job, error) {
	return s.CreateForContext(s.ctx, tenant, requester, kind, key)
}

// CreateForContext records a job while honoring cancellation from the caller.
// HTTP handlers should pass the request context so abandoned requests stop
// waiting for a pooled connection or SQLite writer reservation.
func (s *SQLiteStore) CreateForContext(ctx context.Context, tenant, requester, kind, key string) (Job, error) {
	if tenant == "" || len(tenant) > 128 || requester == "" || len(requester) > 128 || kind != KindNoop || len(key) < 8 || len(key) > 128 {
		return Job{}, errors.New("invalid job")
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return Job{}, err
	}
	id := hex.EncodeToString(raw[:])
	var job Job
	err := withImmediate(ctx, s.db, func(q sqliteSession) error {
		// Serialize before authority and idempotency reads. Deferred read
		// transactions on separate handles can otherwise race while upgrading
		// to the writer and return SQLITE_BUSY instead of the original job.
		if err := authorizedRequester(ctx, q, tenant, requester, "jobs:create"); err != nil {
			return err
		}
		existing, err := scanJob(q.QueryRowContext(ctx, `SELECT id,tenant_id,kind,state,idempotency_key,created_at,updated_at,requested_by
			FROM jobs WHERE tenant_id=? AND idempotency_key=?`, tenant, key))
		if err == nil {
			if existing.Kind != kind || existing.RequestedBy != requester {
				return ErrConflict
			}
			job = existing
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		now := s.now().UTC()
		_, err = q.ExecContext(ctx, `INSERT INTO jobs(id,tenant_id,requested_by,kind,state,idempotency_key,created_at,updated_at,available_at)
			VALUES(?,?,?,?,'queued',?,?,?,?)`, id, tenant, requester, kind, key, dbTime(now), dbTime(now), dbTime(now))
		if err != nil {
			return fmt.Errorf("insert job: %w", err)
		}
		job = Job{ID: id, TenantID: tenant, Kind: kind, State: Queued, IdempotencyKey: key, CreatedAt: now, UpdatedAt: now, RequestedBy: requester}
		return appendAudit(ctx, q, job, "job.created", now)
	})
	if err != nil {
		return Job{}, err
	}
	return job, nil
}

func (s *SQLiteStore) Get(tenant, id string) (Job, error) {
	var j Job
	j, err := scanJob(s.db.QueryRowContext(s.ctx, `SELECT id,tenant_id,kind,state,idempotency_key,created_at,updated_at,requested_by
		FROM jobs WHERE tenant_id=? AND id=?`, tenant, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	return j, err
}

func (s *SQLiteStore) List(tenant string) ([]Job, error) {
	rows, err := s.db.QueryContext(s.ctx, `SELECT id,tenant_id,kind,state,idempotency_key,created_at,updated_at,requested_by
		FROM jobs WHERE tenant_id=? ORDER BY created_at,id`, tenant)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Job
	for rows.Next() {
		var j Job
		if _, err := scanJobInto(rows, &j); err != nil {
			return nil, err
		}
		list = append(list, j)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return list, nil
}

type rowScanner interface{ Scan(...any) error }

func scanJob(row rowScanner) (Job, error) {
	var j Job
	return scanJobInto(row, &j)
}

func scanJobInto(row rowScanner, j *Job) (Job, error) {
	var created, updated string
	err := row.Scan(&j.ID, &j.TenantID, &j.Kind, &j.State, &j.IdempotencyKey, &created, &updated, &j.RequestedBy)
	if err != nil {
		return Job{}, err
	}
	j.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return Job{}, fmt.Errorf("invalid job created_at: %w", err)
	}
	j.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		return Job{}, fmt.Errorf("invalid job updated_at: %w", err)
	}
	return *j, nil
}
