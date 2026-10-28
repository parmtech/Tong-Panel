package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrFileQuotaUnconfigured = errors.New("tenant file quota is not configured")
	ErrFileQuotaReservation  = errors.New("file quota reservation is invalid or closed")
)

// SQLiteTenantFileQuota persists reservations so concurrent API processes that
// share the metadata database cannot over-admit file writes. This is a
// single-node primitive: it does not discover existing files. A trusted caller
// must reconcile the tenant's on-disk usage before configuring its quota.
type SQLiteTenantFileQuota struct {
	db  *sql.DB
	now func() time.Time
}

func NewSQLiteTenantFileQuota(db *sql.DB) (*SQLiteTenantFileQuota, error) {
	if db == nil {
		return nil, errors.New("missing SQLite database")
	}
	return &SQLiteTenantFileQuota{db: db, now: time.Now}, nil
}

// ConfigureTenantQuota records a trusted reconciliation result and quota limit.
// Callers must pause writes and measure all files for the tenant before calling
// this method. It refuses to overwrite active reservations.
func (q *SQLiteTenantFileQuota) ConfigureTenantQuota(tenantID string, limitBytes, reconciledUsage int64) error {
	if strings.TrimSpace(tenantID) == "" || tenantID != strings.TrimSpace(tenantID) || limitBytes < 0 || reconciledUsage < 0 || reconciledUsage > limitBytes {
		return ErrFileQuotaExceeded
	}
	tx, err := q.db.Begin()
	if err != nil {
		return fmt.Errorf("begin quota configuration: %w", err)
	}
	defer tx.Rollback()
	var reservations int
	err = tx.QueryRow(`SELECT count(*) FROM tenant_file_quota_reservations WHERE tenant_id = ?`, tenantID).Scan(&reservations)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("read quota state: %w", err)
	}
	if reservations != 0 {
		return ErrFileQuotaReservation
	}
	_, err = tx.Exec(`INSERT INTO tenant_file_quotas(tenant_id, limit_bytes, used_bytes, reserved_bytes, reconciled_at)
		VALUES(?, ?, ?, 0, ?)
		ON CONFLICT(tenant_id) DO UPDATE SET limit_bytes=excluded.limit_bytes,
		used_bytes=excluded.used_bytes, reserved_bytes=0, reconciled_at=excluded.reconciled_at`,
		tenantID, limitBytes, reconciledUsage, q.now().Unix())
	if err != nil {
		return fmt.Errorf("configure tenant file quota: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit quota configuration: %w", err)
	}
	return nil
}

func (q *SQLiteTenantFileQuota) Reserve(tenantID string, bytes int64) (FileQuotaReservation, error) {
	if strings.TrimSpace(tenantID) == "" || tenantID != strings.TrimSpace(tenantID) || bytes < 0 {
		return nil, ErrFileQuotaExceeded
	}
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, fmt.Errorf("generate quota reservation id: %w", err)
	}
	id := hex.EncodeToString(idBytes)
	tx, err := q.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin quota reservation: %w", err)
	}
	defer tx.Rollback()
	result, err := tx.Exec(`UPDATE tenant_file_quotas
		SET reserved_bytes = reserved_bytes + ?
		WHERE tenant_id = ? AND ? <= limit_bytes - used_bytes - reserved_bytes`, bytes, tenantID, bytes)
	if err != nil {
		return nil, fmt.Errorf("reserve tenant file quota: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read quota reservation result: %w", err)
	}
	if changed != 1 {
		var exists int
		err := tx.QueryRow(`SELECT 1 FROM tenant_file_quotas WHERE tenant_id = ?`, tenantID).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFileQuotaUnconfigured
		}
		if err != nil {
			return nil, fmt.Errorf("check tenant quota: %w", err)
		}
		return nil, ErrFileQuotaExceeded
	}
	_, err = tx.Exec(`INSERT INTO tenant_file_quota_reservations(id, tenant_id, reserved_bytes, created_at) VALUES(?, ?, ?, ?)`, id, tenantID, bytes, q.now().Unix())
	if err != nil {
		return nil, fmt.Errorf("persist quota reservation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit quota reservation: %w", err)
	}
	return &sqliteFileQuotaReservation{db: q.db, id: id, tenantID: tenantID, reserved: bytes}, nil
}

type sqliteFileQuotaReservation struct {
	db       *sql.DB
	id       string
	tenantID string
	reserved int64
}

func (r *sqliteFileQuotaReservation) Commit(actualBytes int64) error {
	if actualBytes < 0 || actualBytes > r.reserved {
		return ErrFileQuotaReservation
	}
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin quota commit: %w", err)
	}
	defer tx.Rollback()
	var reserved int64
	err = tx.QueryRow(`SELECT reserved_bytes FROM tenant_file_quota_reservations WHERE id = ? AND tenant_id = ?`, r.id, r.tenantID).Scan(&reserved)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrFileQuotaReservation
	}
	if err != nil {
		return fmt.Errorf("read quota reservation: %w", err)
	}
	if reserved != r.reserved {
		return ErrFileQuotaReservation
	}
	result, err := tx.Exec(`UPDATE tenant_file_quotas SET reserved_bytes=reserved_bytes-?, used_bytes=used_bytes+?
		WHERE tenant_id=? AND reserved_bytes>=? AND ? <= limit_bytes - used_bytes`, reserved, actualBytes, r.tenantID, reserved, actualBytes)
	if err != nil {
		return fmt.Errorf("account committed file bytes: %w", err)
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		return ErrFileQuotaReservation
	}
	if _, err := tx.Exec(`DELETE FROM tenant_file_quota_reservations WHERE id=?`, r.id); err != nil {
		return fmt.Errorf("close quota reservation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit file quota: %w", err)
	}
	return nil
}

func (r *sqliteFileQuotaReservation) Release() error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin quota release: %w", err)
	}
	defer tx.Rollback()
	var reserved int64
	err = tx.QueryRow(`SELECT reserved_bytes FROM tenant_file_quota_reservations WHERE id = ? AND tenant_id = ?`, r.id, r.tenantID).Scan(&reserved)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrFileQuotaReservation
	}
	if err != nil {
		return fmt.Errorf("read quota reservation: %w", err)
	}
	result, err := tx.Exec(`UPDATE tenant_file_quotas SET reserved_bytes=reserved_bytes-? WHERE tenant_id=? AND reserved_bytes>=?`, reserved, r.tenantID, reserved)
	if err != nil {
		return fmt.Errorf("release quota bytes: %w", err)
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		return ErrFileQuotaReservation
	}
	if _, err := tx.Exec(`DELETE FROM tenant_file_quota_reservations WHERE id=?`, r.id); err != nil {
		return fmt.Errorf("close quota reservation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit quota release: %w", err)
	}
	return nil
}
