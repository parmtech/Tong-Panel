// Package capabilitymodel explores exact-scope plugin authorization.
// It is an offline design prototype and is not used by Tong Panel runtime code.
package capabilitymodel

// Grant is a trusted policy snapshot supplied by a hypothetical policy engine.
// Plugin code must never be allowed to construct or modify this value.
type Grant struct {
	Enabled    bool
	PluginID   string
	InstallID  string
	TenantID   string
	ResourceID string
	Capability string
}

// Request is the identity and target resolved by a hypothetical trusted host
// adapter, not caller-supplied strings passed through from an untrusted plugin.
type Request struct {
	PluginID   string
	InstallID  string
	TenantID   string
	ResourceID string
	Capability string
}

// Allows grants access only when every binding matches and is concrete.
// Empty fields and the wildcard sentinel are denied; this prototype has no
// global or wildcard capability mode.
func Allows(grant Grant, request Request) bool {
	if !grant.Enabled {
		return false
	}
	if !concrete(grant.PluginID) || !concrete(grant.InstallID) ||
		!concrete(grant.TenantID) || !concrete(grant.ResourceID) || !concrete(grant.Capability) {
		return false
	}
	if !concrete(request.PluginID) || !concrete(request.InstallID) ||
		!concrete(request.TenantID) || !concrete(request.ResourceID) || !concrete(request.Capability) {
		return false
	}
	return grant.PluginID == request.PluginID &&
		grant.InstallID == request.InstallID &&
		grant.TenantID == request.TenantID &&
		grant.ResourceID == request.ResourceID &&
		grant.Capability == request.Capability
}

func concrete(value string) bool {
	return value != "" && value != "*"
}
