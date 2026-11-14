// Package releasemodel tests a deliberately inert candidate signature-and-
// checksum contract. Nothing in this package installs, executes, signs a real
// release or designates a trusted production key.
package releasemodel

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

const (
	domainTag        = "TONG-PANEL-OFFLINE-RELEASE-MODEL-v1\x00"
	maxManifestBytes = 64 * 1024
	maxArtifactBytes = 128 * 1024 * 1024
	maxArtifactCount = 16
	modelProduct     = "tong-panel-fixture"
	modelChannel     = "inert-test-only"
)

var (
	ErrRejected    = errors.New("candidate release fixture rejected")
	namePattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	keyIDPattern   = regexp.MustCompile(`^[a-z][a-z0-9.-]{0,63}$`)
	shaPattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	commitPattern  = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// Artifact is a name and the digest/length of one fixture, not a filesystem path.
type Artifact struct {
	Name   string `json:"name"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// Manifest fields are explicit so a canonical JSON byte sequence is testable.
// The schema is not a production API or compatible release format.
type Manifest struct {
	Schema       int        `json:"schema"`
	Product      string     `json:"product"`
	Channel      string     `json:"channel"`
	Release      string     `json:"release"`
	SourceCommit string     `json:"source_commit"`
	OS           string     `json:"os"`
	Arch         string     `json:"arch"`
	Artifacts    []Artifact `json:"artifacts"`
}

// SignedFixture carries raw canonical manifest bytes signed with an ephemeral
// test key. It cannot supply its own trusted public key.
type SignedFixture struct {
	KeyID     string
	Manifest  []byte
	Signature []byte
}

// Expected must be supplied independently by the caller, including the
// explicitly pinned public key, target and exact desired source/version.
// No key is loaded from the signed artifact or any repository file.
type Expected struct {
	KeyID        string
	PublicKey    ed25519.PublicKey
	Release      string
	SourceCommit string
	OS           string
	Arch         string
}

func rejected(reason string) error { return fmt.Errorf("%w: %s", ErrRejected, reason) }

// Verify checks the exact source-pinned target, domain-separated Ed25519
// signature, canonical JSON, and every supplied in-memory payload. A positive
// result is ONLY an offline test invariant, not a production-release decision.
func Verify(expected Expected, signed SignedFixture, payloads map[string][]byte) error {
	if len(expected.PublicKey) != ed25519.PublicKeySize || !keyIDPattern.MatchString(expected.KeyID) ||
		expected.KeyID != signed.KeyID || !versionPattern.MatchString(expected.Release) ||
		!commitPattern.MatchString(expected.SourceCommit) || expected.OS != "linux" || expected.Arch != "amd64" {
		return rejected("missing or mismatched independent trust/target pin")
	}
	if len(signed.Manifest) == 0 || len(signed.Manifest) > maxManifestBytes ||
		len(signed.Signature) != ed25519.SignatureSize ||
		!ed25519.Verify(expected.PublicKey, append([]byte(domainTag), signed.Manifest...), signed.Signature) {
		return rejected("manifest bound/signature invalid")
	}
	var manifest Manifest
	if err := json.Unmarshal(signed.Manifest, &manifest); err != nil {
		return rejected("invalid signed JSON")
	}
	canonical, err := json.Marshal(manifest)
	if err != nil || !bytes.Equal(canonical, signed.Manifest) {
		return rejected("noncanonical or ambiguous manifest")
	}
	if manifest.Schema != 1 || manifest.Product != modelProduct || manifest.Channel != modelChannel ||
		manifest.Release != expected.Release || manifest.SourceCommit != expected.SourceCommit ||
		manifest.OS != expected.OS || manifest.Arch != expected.Arch {
		return rejected("unsupported or unpinned release target")
	}
	if len(manifest.Artifacts) < 1 || len(manifest.Artifacts) > maxArtifactCount ||
		len(payloads) != len(manifest.Artifacts) {
		return rejected("artifact inventory count mismatch")
	}
	seen := make(map[string]bool, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		if !namePattern.MatchString(artifact.Name) || artifact.Name == "." ||
			bytes.Contains([]byte(artifact.Name), []byte("..")) || seen[artifact.Name] {
			return rejected("duplicate or unsafe artifact name")
		}
		seen[artifact.Name] = true
		if artifact.Bytes <= 0 || artifact.Bytes > maxArtifactBytes || !shaPattern.MatchString(artifact.SHA256) {
			return rejected("artifact size/digest policy invalid")
		}
		value, ok := payloads[artifact.Name]
		if !ok || int64(len(value)) != artifact.Bytes {
			return rejected("missing or wrong-length artifact")
		}
		checksum := sha256.Sum256(value)
		if hex.EncodeToString(checksum[:]) != artifact.SHA256 {
			return rejected("artifact checksum mismatch")
		}
	}
	return nil
}
