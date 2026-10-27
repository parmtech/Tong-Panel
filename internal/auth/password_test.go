package auth

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/crypto/argon2"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	password := []byte("correct horse battery staple")
	encoded, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$v=19$m=65536,t=3,p=4$") {
		t.Fatalf("unexpected PHC profile: %q", encoded)
	}
	if !VerifyPassword(password, encoded) {
		t.Fatal("valid password rejected")
	}
	if VerifyPassword([]byte("incorrect password"), encoded) {
		t.Fatal("incorrect password accepted")
	}
	if VerifyPassword([]byte("short"), encoded) {
		t.Fatal("password below the configured minimum accepted")
	}
	if PasswordHashNeedsRehash(encoded) {
		t.Fatal("current profile requires rehash")
	}
}

func TestPasswordHashUsesIndependentSalts(t *testing.T) {
	first, err := HashPassword([]byte("long-enough-password"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := HashPassword([]byte("long-enough-password"))
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("same password received same encoded hash")
	}
}

func TestPasswordHashRejectsLengthBounds(t *testing.T) {
	if _, err := HashPassword([]byte("short")); err != ErrPasswordTooShort {
		t.Fatalf("short password error = %v", err)
	}
	if _, err := HashPassword([]byte(strings.Repeat("x", maxPasswordBytes+1))); err != ErrPasswordTooLong {
		t.Fatalf("oversized password error = %v", err)
	}
	valid := "$argon2id$v=19$m=65536,t=3,p=4$MDEyMzQ1Njc4OWFiY2RlZg$MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY"
	if VerifyPassword([]byte(strings.Repeat("x", maxPasswordBytes+1)), valid) {
		t.Fatal("oversized password verified")
	}
}

func TestPasswordVerifyRejectsLegacyHashBelowMinimum(t *testing.T) {
	password := []byte("short")
	salt := []byte("0123456789abcdef")
	hash := argon2.IDKey(password, salt, passwordHashTime, passwordHashMemoryKiB, passwordHashThreads, passwordHashLength)
	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		passwordHashMemoryKiB, passwordHashTime, passwordHashThreads,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash))
	if VerifyPassword(password, encoded) {
		t.Fatal("stored password below the configured minimum accepted")
	}
}

func TestPasswordPHCRejectsMalformedAndUnboundedParameters(t *testing.T) {
	validSalt := "MDEyMzQ1Njc4OWFiY2RlZg"
	validHash := "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY"
	for _, encoded := range []string{
		"",
		"$argon2i$v=19$m=65536,t=3,p=4$" + validSalt + "$" + validHash,
		"$argon2id$v=16$m=65536,t=3,p=4$" + validSalt + "$" + validHash,
		"$argon2id$v=19$m=4294967295,t=3,p=4$" + validSalt + "$" + validHash,
		"$argon2id$v=19$m=131073,t=3,p=4$" + validSalt + "$" + validHash,
		"$argon2id$v=19$m=65536,t=999999,p=4$" + validSalt + "$" + validHash,
		"$argon2id$v=19$m=65536,t=7,p=4$" + validSalt + "$" + validHash,
		"$argon2id$v=19$m=65536,t=3,p=255$" + validSalt + "$" + validHash,
		"$argon2id$v=19$m=+65536,t=3,p=4$" + validSalt + "$" + validHash,
		"$argon2id$v=19$m=65536,m=65536,t=3,p=4$" + validSalt + "$" + validHash,
		"$argon2id$v=19$m=65536,t=3,p=4,x=1$" + validSalt + "$" + validHash,
		"$argon2id$v=19$m=65536,t=3,p=4$short$" + validHash,
		"$argon2id$v=19$m=65536,t=3,p=4$" + validSalt + "$short",
		"$argon2id$v=19$m=65536,t=3,p=4$" + validSalt[:8] + "\n" + validSalt[8:] + "$" + validHash,
		"$argon2id$v=19$m=65536,t=3,p=4$" + validSalt + "$" + validHash[:8] + "\r\n" + validHash[8:],
	} {
		if VerifyPassword([]byte("irrelevant"), encoded) {
			t.Fatalf("accepted malformed PHC string %q", encoded)
		}
		if !PasswordHashNeedsRehash(encoded) {
			t.Fatalf("malformed PHC string does not require rehash: %q", encoded)
		}
	}
	if VerifyPassword([]byte("password"), strings.Repeat("x", maxPasswordPHCBytes+1)) {
		t.Fatal("oversized PHC string accepted")
	}
}

func TestPasswordPHCRehashBoundedAlternateProfile(t *testing.T) {
	encoded := "$argon2id$v=19$m=131072,t=4,p=2$MDEyMzQ1Njc4OWFiY2RlZg$MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY"
	if !PasswordHashNeedsRehash(encoded) {
		t.Fatal("alternate bounded profile should be rehashed")
	}
}

func BenchmarkHashPassword(b *testing.B) {
	password := []byte("correct horse battery staple")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := HashPassword(password); err != nil {
			b.Fatal(err)
		}
	}
}

func FuzzParseArgon2PHC(f *testing.F) {
	validSalt := "MDEyMzQ1Njc4OWFiY2RlZg"
	validHash := "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY"
	for _, seed := range []string{
		"",
		"$argon2id$v=19$m=65536,t=3,p=4$" + validSalt + "$" + validHash,
		"$argon2i$v=19$m=65536,t=3,p=4$" + validSalt + "$" + validHash,
		"$argon2id$v=19$m=+65536,t=3,p=4$" + validSalt + "$" + validHash,
		strings.Repeat("x", maxPasswordPHCBytes+1),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, encoded string) {
		_, _ = parseArgon2PHC(encoded)
	})
}

