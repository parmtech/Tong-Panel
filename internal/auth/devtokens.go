// Package auth provides a development-only bearer-token reader. It is NOT a production login system.
package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"github.com/parmtech/tong-panel/internal/rbac"
	"strings"
)

// Store never returns a principal without an exact timing-resistant token match.
type Store struct{ tokens map[[32]byte]rbac.Principal }

func NewDevStore(secrets map[string]rbac.Principal) (*Store, error) {
	s := &Store{tokens: make(map[[32]byte]rbac.Principal)}
	for secret, p := range secrets {
		if len(secret) < 32 || strings.TrimSpace(secret) != secret || p.UserID == "" || p.TenantID == "" {
			return nil, errors.New("invalid token or identity")
		}
		if !validRole(p.Role) {
			return nil, errors.New("invalid role")
		}
		h := sha256.Sum256([]byte(secret))
		s.tokens[h] = p
	}
	return s, nil
}
func validRole(r rbac.Role) bool {
	switch r {
	case rbac.Owner, rbac.Admin, rbac.Reseller, rbac.Customer, rbac.Auditor:
		return true
	}
	return false
}
func (s *Store) Check(authz string) (rbac.Principal, bool) {
	if s == nil || !strings.HasPrefix(authz, "Bearer ") {
		return rbac.Principal{}, false
	}
	token := strings.TrimPrefix(authz, "Bearer ")
	if len(token) < 32 || len(token) > 4096 {
		return rbac.Principal{}, false
	}
	hashed := sha256.Sum256([]byte(token))
	for stored, identity := range s.tokens {
		if subtle.ConstantTimeCompare(hashed[:], stored[:]) == 1 {
			return identity, true
		}
	}
	return rbac.Principal{}, false
}
