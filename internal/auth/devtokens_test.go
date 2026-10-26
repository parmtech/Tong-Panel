package auth

import (
	"github.com/parmtech/tong-panel/internal/rbac"
	"testing"
)

const testKey = "a-very-long-secret-key-for-development-only-123456"

func TestStoreFailClosed(t *testing.T) {
	if _, err := NewDevStore(map[string]rbac.Principal{"abc": {UserID: "u", TenantID: "t", Role: rbac.Owner}}); err == nil {
		t.Fatal("weak secret accepted")
	}
	s, err := NewDevStore(map[string]rbac.Principal{testKey: {UserID: "u", TenantID: "t", Role: rbac.Admin}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Check("Bearer " + testKey); !ok {
		t.Fatal("good key denied")
	}
	for _, h := range []string{"", testKey, "Basic " + testKey, "Bearer " + testKey + "x", "Bearer short"} {
		if _, ok := s.Check(h); ok {
			t.Fatalf("bad key accepted %q", h)
		}
	}
}
