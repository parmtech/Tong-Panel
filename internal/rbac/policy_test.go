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
		if (role != Owner) && p.Can("jobs:read", "tenant-b") {
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
