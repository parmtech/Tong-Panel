// Package rbac provides deny-by-default authorization policies.
package rbac

// Role is a server-authorized role. No client input may upgrade this value.
type Role string

const (
	Owner    Role = "owner"
	Admin    Role = "admin"
	Reseller Role = "reseller"
	Customer Role = "customer"
	Auditor  Role = "auditor"
)

type Principal struct {
	UserID, TenantID string
	Role             Role
}

func (p Principal) Can(action, resourceTenant string) bool {
	if p.UserID == "" || p.TenantID == "" || resourceTenant == "" || p.TenantID != resourceTenant {
		return false
	}
	switch action {
	case "jobs:read":
		return (p.Role == Owner || p.Role == Admin || p.Role == Reseller || p.Role == Customer || p.Role == Auditor) && true
	case "jobs:create", "jobs:cancel":
		return (p.Role == Owner || p.Role == Admin || p.Role == Reseller || p.Role == Customer) && (p.TenantID == resourceTenant || p.Role == Owner)
	default:
		return false
	}
}
