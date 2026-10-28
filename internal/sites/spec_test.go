package sites

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeDomain(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"Example.COM", "example.com"},
		{"www.Example.com.", "www.example.com"},
		{"xn--bcher-kva.example", "xn--bcher-kva.example"},
	} {
		got, err := NormalizeDomain(tc.input)
		if err != nil || got != tc.want {
			t.Fatalf("NormalizeDomain(%q)=(%q,%v), want %q", tc.input, got, err, tc.want)
		}
	}
}

func TestNormalizeDomainRejectsAmbiguousOrUnsafeNames(t *testing.T) {
	for _, input := range []string{
		"", "localhost", "example", "example..com", "-bad.example", "bad-.example",
		"*.example.com", "example.com:443", "user@example.com", "127.0.0.1",
		"2001:db8::1", "exa mple.com", "bücher.example", "example.com..",
		strings.Repeat("a", 64) + ".example",
	} {
		if got, err := NormalizeDomain(input); !errors.Is(err, ErrInvalidDomain) || got != "" {
			t.Errorf("NormalizeDomain(%q)=(%q,%v), want ErrInvalidDomain", input, got, err)
		}
	}
}

func TestValidateNormalizesSiteSpec(t *testing.T) {
	spec := SiteSpec{
		TenantID: "tenant_1", SiteID: "site-1", PrimaryDomain: "Example.COM.",
		Aliases: []string{"WWW.Example.com"}, DocumentRoot: "sites/site-1/public",
	}
	got, err := spec.Validate()
	if err != nil {
		t.Fatal(err)
	}
	if got.PrimaryDomain != "example.com" || !reflect.DeepEqual(got.Aliases, []string{"www.example.com"}) {
		t.Fatalf("normalized spec = %#v", got)
	}
	if spec.PrimaryDomain != "Example.COM." {
		t.Fatal("Validate mutated caller's spec")
	}
}

func TestValidateRejectsDuplicateAliasesAndUnsafeDocumentRoots(t *testing.T) {
	base := SiteSpec{TenantID: "tenant", SiteID: "site", PrimaryDomain: "example.com", DocumentRoot: "sites/site/public"}
	for _, tc := range []struct {
		name string
		edit func(*SiteSpec)
	}{
		{"duplicate normalized alias", func(s *SiteSpec) { s.Aliases = []string{"WWW.example.com", "www.EXAMPLE.com."} }},
		{"primary repeated as alias", func(s *SiteSpec) { s.Aliases = []string{"EXAMPLE.com"} }},
		{"path traversal", func(s *SiteSpec) { s.DocumentRoot = "sites/../etc" }},
		{"absolute root", func(s *SiteSpec) { s.DocumentRoot = "/srv/site/public" }},
		{"wrong site path", func(s *SiteSpec) { s.DocumentRoot = "sites/other/public" }},
		{"backslash path", func(s *SiteSpec) { s.DocumentRoot = "sites\\site\\public" }},
		{"invalid id", func(s *SiteSpec) { s.SiteID = "../victim" }},
		{"too many aliases", func(s *SiteSpec) { s.Aliases = make([]string, MaxAliases+1) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := base
			tc.edit(&spec)
			if _, err := spec.Validate(); !errors.Is(err, ErrInvalidSpec) {
				t.Fatalf("Validate error=%v, want ErrInvalidSpec", err)
			}
		})
	}
}
