package auth

import (
	"testing"
	"time"
)

func TestTOTPKnownVector(t *testing.T) {
	secret := []byte("12345678901234567890")
	code, err := TOTPCode(secret, time.Unix(59, 0))
	if err != nil {
		t.Fatal(err)
	}
	// RFC 6238 8-digit SHA1 vector is 94287082, 6-digit truncation is 287082.
	if code != "287082" {
		t.Fatalf("TOTP code = %s", code)
	}
	if !VerifyTOTP(secret, code, time.Unix(59, 0)) {
		t.Fatal("valid code denied")
	}
	if VerifyTOTP(secret, "000000", time.Unix(59, 0)) {
		t.Fatal("bad code accepted")
	}
	if _, err = TOTPCode([]byte("weak"), time.Now()); err == nil {
		t.Fatal("weak secret accepted")
	}
}
