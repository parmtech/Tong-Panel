package auth

import "errors"

// ErrPasswordVerifierBusy indicates that no Argon2id memory budget remains.
// External authentication endpoints must translate this to an indistinguishable,
// rate-limited generic response and MUST NOT reveal account existence.
var ErrPasswordVerifierBusy = errors.New("password verification capacity reached")

// PasswordVerifyBudget bounds simultaneous Argon2id hashing and verification.
// A malicious client must not be able to force arbitrary concurrent 64–128 MiB
// derivations. Admission does not queue requests; overload fails immediately.
// This is NOT a per-account/IP login rate limiter or a complete login service.
type PasswordVerifyBudget struct {
	slots  chan struct{}
	verify func([]byte, string) bool
	hash   func([]byte) (string, error)
}

// NewPasswordVerifyBudget sets a fixed limit of 1–4 concurrent derivations.
// On small candidate VPS hosts production limits MUST be benchmarked and reviewed.
func NewPasswordVerifyBudget(limit int) (*PasswordVerifyBudget, error) {
	if limit < 1 || limit > 4 {
		return nil, ErrInvalidSession
	}
	return &PasswordVerifyBudget{
		slots:  make(chan struct{}, limit),
		verify: VerifyPassword,
		hash:   HashPassword,
	}, nil
}

// Hash shares verification's memory admission budget. It never queues KDF work
// and must still be preceded by the caller's account/source attempt gate.
func (b *PasswordVerifyBudget) Hash(password []byte) (string, error) {
	if b == nil || b.slots == nil || b.hash == nil {
		return "", ErrPasswordVerifierBusy
	}
	select {
	case b.slots <- struct{}{}:
		defer func() { <-b.slots }()
		return b.hash(password)
	default:
		return "", ErrPasswordVerifierBusy
	}
}

// Verify tries to reserve a memory slot before invoking Argon2id. It does not
// change or store password inputs or hash strings; slots release even on panic.
// Always use a separately approved source/account throttle before calling it.
func (b *PasswordVerifyBudget) Verify(password []byte, encoded string) (bool, error) {
	if b == nil || b.slots == nil || b.verify == nil {
		return false, ErrPasswordVerifierBusy
	}
	select {
	case b.slots <- struct{}{}:
		defer func() { <-b.slots }()
		return b.verify(password, encoded), nil
	default:
		return false, ErrPasswordVerifierBusy
	}
}
