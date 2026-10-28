// Package rbac provides deny-by-default authorization policies.
package rbac

// Role is a server-authorized tenant role. No client input may upgrade this value.
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

// Can never authorizes cross-tenant actions, including for tenant owners.
// Global administration requires a separate explicit platform-scoped principal.
func (p Principal) Can(action, resourceTenant string) bool {
	if p.UserID == "" || p.TenantID == "" || resourceTenant == "" || p.TenantID != resourceTenant {
		return false
	}
	switch action {
	case "jobs:read":
		return (p.Role == Owner || p.Role == Admin || p.Role == Reseller || p.Role == Customer || p.Role == Auditor)
	case "jobs:create", "jobs:cancel":
		return (p.Role == Owner || p.Role == Admin || p.Role == Reseller || p.Role == Customer)
	case "files:read":
		return (p.Role == Owner || p.Role == Admin || p.Role == Reseller || p.Role == Customer)
	default:
		return false
	}
}

