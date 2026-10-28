package sites

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
)

type txtLookupFunc func(context.Context, string) ([]string, error)

func (f txtLookupFunc) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return f(ctx, name)
}

func newVerificationCatalog(t *testing.T) *Catalog {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec("INSERT INTO tenants(id,name) VALUES('tenant-a','Tenant A')"); err != nil {
		t.Fatal(err)
	}
	catalog, err := NewCatalog(db)
	if err != nil {
		t.Fatal(err)
	}
	catalog.now = func() time.Time { return time.Unix(2000, 0) }
	spec := SiteSpec{TenantID: "tenant-a", SiteID: "site-a", PrimaryDomain: "example.com", DocumentRoot: "sites/site-a/public"}
	if err := catalog.Create(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	return catalog
}

func TestDomainVerificationStoresDigestAndRequiresExactDNSProof(t *testing.T) {
	catalog := newVerificationCatalog(t)
	instruction, err := catalog.BeginDomainVerification(context.Background(), "tenant-a", "site-a", "EXAMPLE.COM.")
	if err != nil {
		t.Fatal(err)
	}
	if instruction.Name != "_tong-panel-challenge.example.com" || instruction.Value == "" || instruction.ExpiresAt != 5600 {
		t.Fatalf("unexpected verification instruction: %#v", instruction)
	}
	var digest string
	var expires int64
	if err := catalog.db.QueryRow("SELECT verification_token_sha256,verification_expires_at FROM site_domains WHERE domain='example.com'").Scan(&digest, &expires); err != nil {
		t.Fatal(err)
	}
	if digest == instruction.Value || expires != instruction.ExpiresAt {
		t.Fatal("raw token was stored or expiry was not persisted")
	}
	lookups := 0
	lookup := txtLookupFunc(func(_ context.Context, name string) ([]string, error) {
		lookups++
		if name != instruction.Name {
			t.Fatalf("lookup name=%q", name)
		}
		return []string{"unrelated", instruction.Value}, nil
	})
	if err := catalog.CompleteDomainVerification(context.Background(), "tenant-a", "site-a", "example.com", instruction.Value, lookup); err != nil {
		t.Fatal(err)
	}
	if lookups != 1 {
		t.Fatalf("TXT lookup count=%d", lookups)
	}
	if err := catalog.CompleteDomainVerification(context.Background(), "tenant-a", "site-a", "example.com", instruction.Value, lookup); !errors.Is(err, ErrDomainVerificationState) {
		t.Fatalf("replayed proof error=%v, want consumed-state error", err)
	}
	if lookups != 1 {
		t.Fatalf("replayed proof performed another TXT lookup: count=%d", lookups)
	}
	record, err := catalog.Get(context.Background(), "tenant-a", "site-a")
	if err != nil {
		t.Fatal(err)
	}
	if record.Domains[0].OwnershipState != "verified" {
		t.Fatalf("ownership state=%q", record.Domains[0].OwnershipState)
	}
	if err := catalog.db.QueryRow("SELECT verification_token_sha256 IS NULL AND verification_expires_at IS NULL FROM site_domains WHERE domain='example.com'").Scan(&expires); err != nil || expires != 1 {
		t.Fatalf("verified domain retained proof token: value=%d err=%v", expires, err)
	}
}

func TestDomainVerificationRejectsWrongOwnerProofWrongTenantAndExpiry(t *testing.T) {
	catalog := newVerificationCatalog(t)
	instruction, err := catalog.BeginDomainVerification(context.Background(), "tenant-a", "site-a", "example.com")
	if err != nil {
		t.Fatal(err)
	}
	called := false
	lookup := txtLookupFunc(func(context.Context, string) ([]string, error) {
		called = true
		return []string{instruction.Value}, nil
	})
	if err := catalog.CompleteDomainVerification(context.Background(), "tenant-a", "site-a", "example.com", "wrong-token", lookup); !errors.Is(err, ErrDomainProofMismatch) {
		t.Fatalf("wrong token error=%v", err)
	}
	if called {
		t.Fatal("DNS was queried before token digest validation")
	}
	if _, err := catalog.BeginDomainVerification(context.Background(), "other-tenant", "site-a", "example.com"); !errors.Is(err, ErrSiteNotFound) {
		t.Fatalf("cross-tenant token request error=%v", err)
	}
	catalog.now = func() time.Time { return time.Unix(instruction.ExpiresAt, 0) }
	if err := catalog.CompleteDomainVerification(context.Background(), "tenant-a", "site-a", "example.com", instruction.Value, lookup); !errors.Is(err, ErrDomainVerificationExpired) {
		t.Fatalf("expired proof error=%v", err)
	}
	if called {
		t.Fatal("DNS was queried for an expired token")
	}
}

func TestDomainVerificationRequiresDNSRecordAndReissuedToken(t *testing.T) {
	catalog := newVerificationCatalog(t)
	first, err := catalog.BeginDomainVerification(context.Background(), "tenant-a", "site-a", "example.com")
	if err != nil {
		t.Fatal(err)
	}
	second, err := catalog.BeginDomainVerification(context.Background(), "tenant-a", "site-a", "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if first.Value == second.Value {
		t.Fatal("reissue returned the same random token")
	}
	noRecord := txtLookupFunc(func(context.Context, string) ([]string, error) { return []string{"other"}, nil })
	if err := catalog.CompleteDomainVerification(context.Background(), "tenant-a", "site-a", "example.com", first.Value, noRecord); !errors.Is(err, ErrDomainProofMismatch) {
		t.Fatalf("replaced token error=%v", err)
	}
	if err := catalog.CompleteDomainVerification(context.Background(), "tenant-a", "site-a", "example.com", second.Value, noRecord); !errors.Is(err, ErrDomainProofMismatch) {
		t.Fatalf("missing DNS record error=%v", err)
	}
}

func TestDomainVerificationConcurrentReplayHasSingleWinner(t *testing.T) {
	catalog := newVerificationCatalog(t)
	instruction, err := catalog.BeginDomainVerification(context.Background(), "tenant-a", "site-a", "example.com")
	if err != nil {
		t.Fatal(err)
	}
	ready := make(chan struct{}, 2)
	release := make(chan struct{})
	lookup := txtLookupFunc(func(context.Context, string) ([]string, error) {
		ready <- struct{}{}
		<-release
		return []string{instruction.Value}, nil
	})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- catalog.CompleteDomainVerification(context.Background(), "tenant-a", "site-a", "example.com", instruction.Value, lookup)
		}()
	}
	<-ready
	<-ready
	close(release)
	wg.Wait()
	close(results)
	success, used := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrDomainVerificationExpired) {
			used++
		} else {
			t.Errorf("unexpected replay result: %v", err)
		}
	}
	if success != 1 || used != 1 {
		t.Fatalf("successes=%d replay denials=%d, want one each", success, used)
	}
}
