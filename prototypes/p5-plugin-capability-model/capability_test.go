package capabilitymodel

import "testing"

func TestAllowsRequiresExactEnabledGrant(t *testing.T) {
	grant := Grant{
		Enabled:    true,
		PluginID:   "plugin.example",
		InstallID:  "install-7",
		TenantID:   "tenant-a",
		ResourceID: "site-3",
		Capability: "site.read",
	}
	request := Request{
		PluginID:   "plugin.example",
		InstallID:  "install-7",
		TenantID:   "tenant-a",
		ResourceID: "site-3",
		Capability: "site.read",
	}
	tests := []struct {
		name    string
		grant   Grant
		request Request
		want    bool
	}{
		{name: "exact enabled scope", grant: grant, request: request, want: true},
		{name: "disabled grant", grant: setEnabled(grant, false), request: request},
		{name: "wrong plugin", grant: grant, request: setRequestPlugin(request, "plugin.other")},
		{name: "wrong installation", grant: grant, request: setRequestInstall(request, "install-8")},
		{name: "wrong tenant", grant: grant, request: setRequestTenant(request, "tenant-b")},
		{name: "wrong resource", grant: grant, request: setRequestResource(request, "site-4")},
		{name: "wrong capability", grant: grant, request: setRequestCapability(request, "site.write")},
		{name: "empty grant tenant", grant: setGrantTenant(grant, ""), request: request},
		{name: "empty request resource", grant: grant, request: setRequestResource(request, "")},
		{name: "wildcard grant tenant", grant: setGrantTenant(grant, "*"), request: request},
		{name: "wildcard request capability", grant: grant, request: setRequestCapability(request, "*")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Allows(tt.grant, tt.request); got != tt.want {
				t.Fatalf("Allows(%+v, %+v) = %t, want %t", tt.grant, tt.request, got, tt.want)
			}
		})
	}
}

func TestDisableRevokesEveryRequestForGrant(t *testing.T) {
	grant := Grant{Enabled: true, PluginID: "p", InstallID: "i", TenantID: "t", ResourceID: "r", Capability: "c"}
	request := Request{PluginID: "p", InstallID: "i", TenantID: "t", ResourceID: "r", Capability: "c"}
	if !Allows(grant, request) {
		t.Fatal("control request must be allowed before disable")
	}
	grant.Enabled = false
	for n := 0; n < 100; n++ {
		if Allows(grant, request) {
			t.Fatalf("request %d remained authorized after disable", n)
		}
	}
}

func setEnabled(grant Grant, enabled bool) Grant      { grant.Enabled = enabled; return grant }
func setGrantTenant(grant Grant, tenant string) Grant { grant.TenantID = tenant; return grant }
func setRequestPlugin(request Request, plugin string) Request {
	request.PluginID = plugin
	return request
}
func setRequestInstall(request Request, install string) Request {
	request.InstallID = install
	return request
}
func setRequestTenant(request Request, tenant string) Request {
	request.TenantID = tenant
	return request
}
func setRequestResource(request Request, resource string) Request {
	request.ResourceID = resource
	return request
}
func setRequestCapability(request Request, capability string) Request {
	request.Capability = capability
	return request
}
