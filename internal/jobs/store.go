// Package jobs contains a small durable local development job journal. It is not a production multi-node queue.
package jobs

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const KindNoop = "validation.noop"
const (
	Queued  = "queued"
	Leased  = "leased"
	Running = "running"
	Done    = "completed"
	Failed  = "failed"
)

var ErrConflict = errors.New("idempotency key already used for different work")
var ErrNotFound = errors.New("job not found")

type Job struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	RequestedBy    string    `json:"requested_by,omitempty"`
	Kind           string    `json:"kind"`
	State          string    `json:"state"`
	IdempotencyKey string    `json:"idempotency_key"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
type Store struct {
	mu    sync.Mutex
	path  string
	items []Job
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("missing journal path")
	}
	s := &Store{path: path, items: []Job{}}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(b, &s.items); err != nil {
		return nil, fmt.Errorf("invalid job journal: %w", err)
	}
	for _, j := range s.items {
		if j.ID == "" || j.TenantID == "" || j.Kind != KindNoop {
			return nil, errors.New("invalid job record")
		}
	}
	return s, nil
}
func (s *Store) save(items []Job) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tong-jobs-")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err = tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err = tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(name, s.path); err != nil {
		return err
	}
	return syncDir(dir)
}
func (s *Store) Create(tenant, kind, key string) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tenant == "" || len(tenant) > 128 || kind != KindNoop || len(key) < 8 || len(key) > 128 {
		return Job{}, errors.New("invalid job")
	}
	for _, j := range s.items {
		if j.TenantID == tenant && j.IdempotencyKey == key {
			if j.Kind == kind {
				return j, nil
			}
			return Job{}, ErrConflict
		}
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return Job{}, err
	}
	now := time.Now().UTC()
	j := Job{ID: hex.EncodeToString(id[:]), TenantID: tenant, Kind: kind, State: Queued, IdempotencyKey: key, CreatedAt: now, UpdatedAt: now}
	next := append(append([]Job(nil), s.items...), j)
	if err := s.save(next); err != nil {
		return Job{}, err
	}
	s.items = next
	return j, nil
}
func (s *Store) Get(tenant, id string) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.items {
		if j.ID == id && j.TenantID == tenant {
			return j, nil
		}
	}
	return Job{}, ErrNotFound
}
func (s *Store) List(tenant string) []Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	var list []Job
	for _, j := range s.items {
		if j.TenantID == tenant {
			list = append(list, j)
		}
	}
	return list
}

// Transition is an explicit validated state update. No workers execute user code in P1.
func (s *Store) Transition(tenant, id, to string) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, j := range s.items {
		if j.TenantID != tenant || j.ID != id {
			continue
		}
		if !((j.State == Queued && to == Running) || (j.State == Running && (to == Done || to == Failed))) {
			return Job{}, errors.New("invalid job transition")
		}
		j.State = to
		j.UpdatedAt = time.Now().UTC()
		next := append([]Job(nil), s.items...)
		next[i] = j
		if err := s.save(next); err != nil {
			return Job{}, err
		}
		s.items = next
		return j, nil
	}
	return Job{}, ErrNotFound
}
