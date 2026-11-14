package releasemodel

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

const sourceFixture = "0123456789abcdef0123456789abcdef01234567"

type fixture struct {
	expected Expected
	signed   SignedFixture
	manifest Manifest
	payloads map[string][]byte
	private  ed25519.PrivateKey
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	payloads := map[string][]byte{"tong-fixture.amd64.deb": []byte("inert synthetic bytes; not installable")}
	checksum := sha256.Sum256(payloads["tong-fixture.amd64.deb"])
	manifest := Manifest{
		Schema: 1, Product: modelProduct, Channel: modelChannel,
		Release: "0.3.0", SourceCommit: sourceFixture, OS: "linux", Arch: "amd64",
		Artifacts: []Artifact{{Name: "tong-fixture.amd64.deb", Bytes: int64(len(payloads["tong-fixture.amd64.deb"])), SHA256: hex.EncodeToString(checksum[:])}},
	}
	f := fixture{expected: Expected{KeyID: "reviewer-fixture", PublicKey: public, Release: manifest.Release, SourceCommit: manifest.SourceCommit, OS: manifest.OS, Arch: manifest.Arch}, manifest: manifest, payloads: payloads, private: private}
	f.sign(t)
	return f
}

func (f *fixture) sign(t *testing.T) {
	t.Helper()
	raw, err := json.Marshal(f.manifest)
	if err != nil {
		t.Fatal(err)
	}
	f.signed = SignedFixture{KeyID: f.expected.KeyID, Manifest: raw, Signature: ed25519.Sign(f.private, append([]byte(domainTag), raw...))}
}

func requireReject(t *testing.T, f fixture) {
	t.Helper()
	if err := Verify(f.expected, f.signed, f.payloads); !errors.Is(err, ErrRejected) {
		t.Fatalf("untrusted fixture was accepted: %v", err)
	}
}

func TestCanonicalPinnedFixtureSucceedsOnlyWithOutOfBandTrust(t *testing.T) {
	f := newFixture(t)
	if err := Verify(f.expected, f.signed, f.payloads); err != nil {
		t.Fatal(err)
	}
	f.expected.PublicKey = nil
	requireReject(t, f)
}

func TestRejectTamperedSignatureManifestAndKeyIdentity(t *testing.T) {
	for _, mode := range []string{"signature", "manifest", "different key", "self-asserted key ID", "wrong key length", "signature length", "unsigned"} {
		t.Run(mode, func(t *testing.T) {
			f := newFixture(t)
			switch mode {
			case "signature":
				f.signed.Signature[0] ^= 1
			case "manifest":
				f.signed.Manifest[3] ^= 1
			case "different key":
				k, _, _ := ed25519.GenerateKey(rand.Reader)
				f.expected.PublicKey = k
			case "self-asserted key ID":
				f.signed.KeyID = "other-key"
			case "wrong key length":
				f.expected.PublicKey = f.expected.PublicKey[:4]
			case "signature length":
				f.signed.Signature = f.signed.Signature[:1]
			case "unsigned":
				f.signed.Signature = nil
			}
			requireReject(t, f)
		})
	}
}

func TestRejectCorrectlySignedWrongTarget(t *testing.T) {
	for _, mode := range []string{"release", "source", "os", "arch", "schema", "product", "channel"} {
		t.Run(mode, func(t *testing.T) {
			f := newFixture(t)
			switch mode {
			case "release":
				f.manifest.Release = "0.2.0" // downgrade, still signed
			case "source":
				f.manifest.SourceCommit = strings.Repeat("a", 40)
			case "os":
				f.manifest.OS = "windows"
			case "arch":
				f.manifest.Arch = "arm64"
			case "schema":
				f.manifest.Schema = 2
			case "product":
				f.manifest.Product = "tong-panel-production"
			case "channel":
				f.manifest.Channel = "stable"
			}
			f.sign(t)
			requireReject(t, f)
		})
	}
}

func TestRejectMissingExtraModifiedPayloadsAndMetadata(t *testing.T) {
	for _, mode := range []string{"missing", "extra", "modified", "wrong size", "wrong digest", "upper digest", "zero size", "huge size", "duplicate", "unsafe slash", "unsafe dot", "empty name", "missing inventory"} {
		t.Run(mode, func(t *testing.T) {
			f := newFixture(t)
			switch mode {
			case "missing":
				delete(f.payloads, "tong-fixture.amd64.deb")
			case "extra":
				f.payloads["extra"] = []byte("not signed")
			case "modified":
				f.payloads["tong-fixture.amd64.deb"][0] ^= 1
			case "wrong size":
				f.manifest.Artifacts[0].Bytes++
			case "wrong digest":
				f.manifest.Artifacts[0].SHA256 = strings.Repeat("0", 64)
			case "upper digest":
				f.manifest.Artifacts[0].SHA256 = strings.Repeat("F", 64)
			case "zero size":
				f.manifest.Artifacts[0].Bytes = 0
			case "huge size":
				f.manifest.Artifacts[0].Bytes = maxArtifactBytes + 1
			case "duplicate":
				f.manifest.Artifacts = append(f.manifest.Artifacts, f.manifest.Artifacts[0])
				f.payloads["extra"] = []byte("extra")
			case "unsafe slash":
				f.manifest.Artifacts[0].Name = "../../root"
			case "unsafe dot":
				f.manifest.Artifacts[0].Name = "a..b"
			case "empty name":
				f.manifest.Artifacts[0].Name = ""
			case "missing inventory":
				f.manifest.Artifacts = nil
			}
			if mode != "missing" && mode != "extra" && mode != "modified" {
				f.sign(t)
			}
			requireReject(t, f)
		})
	}
}

func TestRejectAmbiguousEvenWhenSignedJSON(t *testing.T) {
	for _, raw := range []string{
		`{"schema":1,"schema":1}`,
		`{"schema":1,"product":"tong-panel-fixture","owner_approved":true}`,
		` {"schema":1}`,
		`{"schema":1} `,
		`null`,
		`[]`,
	} {
		t.Run(raw, func(t *testing.T) {
			f := newFixture(t)
			f.signed.Manifest = []byte(raw)
			f.signed.Signature = ed25519.Sign(f.private, append([]byte(domainTag), f.signed.Manifest...))
			requireReject(t, f)
		})
	}
}

func TestRejectManifestBoundsAndIndependentPinErrors(t *testing.T) {
	f := newFixture(t)
	f.signed.Manifest = bytes.Repeat([]byte{'a'}, maxManifestBytes+1)
	requireReject(t, f)
	for _, mode := range []string{"no ID", "bad ID", "bad expected version", "bad expected source", "unsupported expected platform"} {
		t.Run(mode, func(t *testing.T) {
			f := newFixture(t)
			switch mode {
			case "no ID":
				f.expected.KeyID = ""
			case "bad ID":
				f.expected.KeyID = "a/../../b"
				f.signed.KeyID = f.expected.KeyID
			case "bad expected version":
				f.expected.Release = "0.03.0"
			case "bad expected source":
				f.expected.SourceCommit = "draft-main"
			case "unsupported expected platform":
				f.expected.OS = "freebsd"
			}
			requireReject(t, f)
		})
	}
}

func TestSignatureIsDomainSeparated(t *testing.T) {
	f := newFixture(t)
	f.signed.Signature = ed25519.Sign(f.private, f.signed.Manifest) // legitimate key, wrong protocol
	requireReject(t, f)
}
