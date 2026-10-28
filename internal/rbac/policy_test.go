package rbac

import "testing"

func TestDenyByDefault(t *testing.T) {
	for _, role := range []Role{Owner, Admin, Reseller, Customer, Auditor, "hacker"} {
		p := Principal{UserID: "u", TenantID: "tenant-a", Role: role}
		if p.Can("root:exec", "tenant-a") {
			t.Fatalf("role %q escaped allowlist", role)
		}
		if p.Can("jobs:create", "") {
			t.Fatal("empty tenant allowed")
		}
		if p.Can("jobs:read", "tenant-b") || p.Can("jobs:create", "tenant-b") || p.Can("jobs:cancel", "tenant-b") {
			t.Fatalf("role %q escaped tenant", role)
		}
	}
	if (Principal{}).Can("jobs:read", "tenant-a") {
		t.Fatal("anonymous allowed")
	}
	if (Principal{UserID: "a", TenantID: "t", Role: Auditor}).Can("jobs:create", "t") {
		t.Fatal("auditor mutated")
	}
}

func TestOwnerIsTenantBoundAndAuditorIsReadOnly(t *testing.T) {
	for _, role := range []Role{Owner, Admin, Reseller, Customer, Auditor} {
		p := Principal{UserID: "u", TenantID: "team-1", Role: role}
		if !p.Can("jobs:read", "team-1") {
			t.Fatalf("role %s cannot read own tenant", role)
		}
		for _, action := range []string{"jobs:read", "jobs:create", "jobs:cancel"} {
			if p.Can(action, "team-2") {
				t.Fatalf("role %s performed cross-tenant %s", role, action)
			}
		}
	}
	auditor := Principal{UserID: "audit", TenantID: "team-1", Role: Auditor}
	if auditor.Can("jobs:create", "team-1") || auditor.Can("jobs:cancel", "team-1") {
		t.Fatal("auditor gained mutation permission")
	}
}

func TestFilesReadIsTenantBoundAndAuditorDenied(t *testing.T) {
	for _, role := range []Role{Owner, Admin, Reseller, Customer} {
		p := Principal{UserID: "u", TenantID: "team-1", Role: role}
		if !p.Can("files:read", "team-1") {
			t.Errorf("role %s cannot read own tenant files", role)
		}
		if p.Can("files:read", "team-2") {
			t.Errorf("role %s crossed tenant boundary for files", role)
		}
	}
	if (Principal{UserID: "audit", TenantID: "team-1", Role: Auditor}).Can("files:read", "team-1") {
		t.Fatal("auditor unexpectedly received customer file access")
	}
	if (Principal{UserID: "u", TenantID: "team-1", Role: "unknown"}).Can("files:read", "team-1") {
		t.Fatal("unknown role unexpectedly received file access")
	}
}

func TestFilesWriteIsTenantBoundAndAuditorDenied(t *testing.T) {
	for _, role := range []Role{Owner, Admin, Reseller, Customer} {
		p := Principal{UserID: "u", TenantID: "team-1", Role: role}
		if !p.Can("files:write", "team-1") {
			t.Errorf("role %s cannot write own tenant files", role)
		}
		if p.Can("files:write", "team-2") {
			t.Errorf("role %s crossed tenant boundary for file writes", role)
		}
	}
	for _, role := range []Role{Auditor, "unknown"} {
		if (Principal{UserID: "u", TenantID: "team-1", Role: role}).Can("files:write", "team-1") {
			t.Errorf("role %q unexpectedly received file write access", role)
		}
	}
}


func TestSitePermissionsAreTenantBoundAndAuditorReadOnly(t *testing.T) {
	for _, role := range []Role{Owner, Admin, Reseller, Customer} {
		p := Principal{UserID: "u", TenantID: "team-1", Role: role}
		if !p.Can("sites:read", "team-1") || !p.Can("sites:write", "team-1") {
			t.Errorf("role %s cannot manage own-tenant site metadata", role)
		}
		if p.Can("sites:read", "team-2") || p.Can("sites:write", "team-2") {
			t.Errorf("role %s crossed tenant boundary for sites", role)
		}
	}
	auditor := Principal{UserID: "audit", TenantID: "team-1", Role: Auditor}
	if !auditor.Can("sites:read", "team-1") || auditor.Can("sites:write", "team-1") {
		t.Fatal("auditor site permission is not read-only")
	}
	if (Principal{UserID: "u", TenantID: "team-1", Role: "unknown"}).Can("sites:read", "team-1") {
		t.Fatal("unknown role unexpectedly received site access")
	}
}
