package recovery

import (
	"errors"
	"fmt"
	"sync"
)

var ErrInvalidAdmission = errors.New("invalid admission request")
var ErrIdempotencyConflict = errors.New("idempotency key reused with a different request")

// AdmissionRequest is a value-only input for an idempotency thought model.
// This scope/key policy is illustrative and must be reviewed before adoption.
type AdmissionRequest struct {
	TenantID      string
	AppID         string
	Key           string
	PayloadDigest string
}

type admissionKey struct {
	tenant string
	app    string
	key    string
}

type admissionRecord struct {
	payloadDigest string
	jobID         string
}

// AdmissionModel simulates same-request deduplication in memory. It serializes
// callers within this process only; it does not model persistence, multiple
// processes, authorization, or job execution.
type AdmissionModel struct {
	mu    sync.Mutex
	byKey map[admissionKey]admissionRecord
	next  uint64
}

func NewAdmissionModel() *AdmissionModel {
	return &AdmissionModel{byKey: make(map[admissionKey]admissionRecord)}
}

func (m *AdmissionModel) Admit(request AdmissionRequest) (string, error) {
	if request.TenantID == "" || request.AppID == "" || request.Key == "" || request.PayloadDigest == "" {
		return "", ErrInvalidAdmission
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	key := admissionKey{tenant: request.TenantID, app: request.AppID, key: request.Key}
	if prior, ok := m.byKey[key]; ok {
		if prior.payloadDigest != request.PayloadDigest {
			return "", ErrIdempotencyConflict
		}
		return prior.jobID, nil
	}

	m.next++
	jobID := fmt.Sprintf("model-job-%06d", m.next)
	m.byKey[key] = admissionRecord{payloadDigest: request.PayloadDigest, jobID: jobID}
	return jobID, nil
}

func (m *AdmissionModel) JobCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.byKey)
}
