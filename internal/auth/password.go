package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	passwordHashMemoryKiB = 64 * 1024
	passwordHashTime      = 3
	passwordHashThreads   = 4
	passwordHashLength    = 32
	passwordSaltLength    = 16
	maxPasswordBytes      = 1024
	maxPasswordPHCBytes   = 512
)

var (
	ErrPasswordTooShort = errors.New("password must be at least 12 bytes")
	ErrPasswordTooLong  = errors.New("password exceeds maximum length")
)

type argon2Params struct {
	memory  uint32
	time    uint32
	threads uint8
	salt    []byte
	hash    []byte
}

// HashPassword creates an Argon2id PHC string using the RFC 9106 lower-memory
// recommendation. This primitive is not wired to login or account creation.
// Callers must enforce rate limits before exposing password hashing to requests.
func HashPassword(password []byte) (string, error) {
	if len(password) < 12 {
		return "", ErrPasswordTooShort
	}
	if len(password) > maxPasswordBytes {
		return "", ErrPasswordTooLong
	}
	salt := make([]byte, passwordSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey(password, salt, passwordHashTime, passwordHashMemoryKiB, passwordHashThreads, passwordHashLength)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		passwordHashMemoryKiB, passwordHashTime, passwordHashThreads,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}

// VerifyPassword rejects malformed or over-budget PHC strings before running
// Argon2. Password bytes are consumed as-is; callers must not trim or log them.
func VerifyPassword(password []byte, encoded string) bool {
	if len(password) < 12 || len(password) > maxPasswordBytes || len(encoded) > maxPasswordPHCBytes {
		return false
	}
	params, ok := parseArgon2PHC(encoded)
	if !ok {
		return false
	}
	actual := argon2.IDKey(password, params.salt, params.time, params.memory, params.threads, uint32(len(params.hash)))
	return subtle.ConstantTimeCompare(actual, params.hash) == 1
}

// PasswordHashNeedsRehash reports whether a supported hash uses parameters
// other than the current creation profile. Malformed or unsupported strings
// return true so callers never treat them as an acceptable stored credential.
func PasswordHashNeedsRehash(encoded string) bool {
	params, ok := parseArgon2PHC(encoded)
	return !ok || params.memory != passwordHashMemoryKiB || params.time != passwordHashTime || params.threads != passwordHashThreads
}

func parseArgon2PHC(encoded string) (argon2Params, bool) {
	var zero argon2Params
	if len(encoded) == 0 || len(encoded) > maxPasswordPHCBytes {
		return zero, false
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		return zero, false
	}
	var memory, iterations uint64
	var parallelism uint64
	seen := map[string]bool{}
	for _, item := range strings.Split(parts[3], ",") {
		key, value, found := strings.Cut(item, "=")
		if !found || seen[key] {
			return zero, false
		}
		seen[key] = true
		parsed, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return zero, false
		}
		switch key {
		case "m":
			memory = parsed
		case "t":
			iterations = parsed
		case "p":
			parallelism = parsed
		default:
			return zero, false
		}
	}
	// Enforce lower bounds and strict CPU/memory ceilings before KDF invocation.
	// The supported range allows bounded future tuning without accepting weak or
	// attacker-amplified values from a corrupted/database-sourced PHC string.
	if len(seen) != 3 || memory < 64*1024 || memory > 128*1024 || iterations < 3 || iterations > 6 || parallelism < 1 || parallelism > 4 || memory < 8*parallelism {
		return zero, false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 16 || len(salt) > 32 {
		return zero, false
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(hash) != passwordHashLength {
		return zero, false
	}
	return argon2Params{memory: uint32(memory), time: uint32(iterations), threads: uint8(parallelism), salt: salt, hash: hash}, true
}
