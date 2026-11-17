package auth

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"strconv"
	"time"
)

// TOTPCode implements RFC 6238 SHA1/6 digits for a pre-enrolled secret.
// This standalone primitive is not MFA enrollment, account storage or a login system.
func TOTPCode(secret []byte, at time.Time) (string, error) {
	if len(secret) < 20 {
		return "", fmt.Errorf("secret too short")
	}
	counter := uint64(at.Unix() / 30)
	var message [8]byte
	binary.BigEndian.PutUint64(message[:], counter)
	mac := hmac.New(sha1.New, secret)
	_, _ = mac.Write(message[:])
	sum := mac.Sum(nil)
	off := int(sum[len(sum)-1] & 0xf)
	n := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", n%1_000_000), nil
}

// VerifyTOTP checks a small clock skew window in constant-time for code matching.
func VerifyTOTP(secret []byte, code string, at time.Time) bool {
	if len(code) != 6 {
		return false
	}
	if _, err := strconv.Atoi(code); err != nil {
		return false
	}
	good := 0
	for _, step := range []int{-1, 0, 1} {
		expected, err := TOTPCode(secret, at.Add(time.Duration(step)*30*time.Second))
		if err != nil {
			return false
		}
		good |= subtle.ConstantTimeCompare([]byte(expected), []byte(code))
	}
	return good == 1
}
