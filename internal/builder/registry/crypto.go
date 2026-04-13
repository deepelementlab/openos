package registry

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// PackageSignatureFile is the on-disk signature sidecar next to manifest.json.
const PackageSignatureFile = "signature.json"

// AAPSignature wraps an Ed25519 signature over the canonical package digest.
type AAPSignature struct {
	Algorithm string `json:"algorithm"`
	Signature string `json:"signature"`  // base64
	PublicKey string `json:"public_key"` // base64 (optional, for distribution)
}

// PackageDigest computes the canonical SHA-256 over manifest bytes and layer digests.
func PackageDigest(manifest []byte, layerDigests []string) [32]byte {
	h := sha256.New()
	_, _ = h.Write(manifest)
	for _, d := range layerDigests {
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(d))
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// SignPackage returns a detached Ed25519 signature for the package digest.
func SignPackage(manifest []byte, layerDigests []string, privateKey ed25519.PrivateKey) ([]byte, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("crypto: invalid ed25519 private key size")
	}
	d := PackageDigest(manifest, layerDigests)
	return ed25519.Sign(privateKey, d[:]), nil
}

// VerifyPackage checks the signature against the canonical digest and public key.
func VerifyPackage(manifest []byte, layerDigests []string, signature []byte, publicKey ed25519.PublicKey) error {
	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("crypto: invalid ed25519 public key size")
	}
	d := PackageDigest(manifest, layerDigests)
	if !ed25519.Verify(publicKey, d[:], signature) {
		return fmt.Errorf("crypto: signature verification failed")
	}
	return nil
}

// WriteSignature writes signature.json into an AAP directory.
func WriteSignature(dir string, manifest []byte, layerDigests []string, privateKey ed25519.PrivateKey) error {
	sig, err := SignPackage(manifest, layerDigests, privateKey)
	if err != nil {
		return err
	}
	pub := privateKey.Public().(ed25519.PublicKey)
	wrap := AAPSignature{
		Algorithm: "ed25519",
		Signature: base64.StdEncoding.EncodeToString(sig),
		PublicKey: base64.StdEncoding.EncodeToString(pub),
	}
	b, err := json.MarshalIndent(wrap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, PackageSignatureFile), b, 0o644)
}

// VerifyPackageDir loads manifest.json, layers.json, and signature.json and verifies.
func VerifyPackageDir(dir string) error {
	man, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return err
	}
	layersPath := filepath.Join(dir, "layers.json")
	lb, err := os.ReadFile(layersPath)
	if err != nil {
		return err
	}
	var meta struct {
		LayerDigests []string `json:"layer_digests"`
	}
	if err := json.Unmarshal(lb, &meta); err != nil {
		return err
	}
	sb, err := os.ReadFile(filepath.Join(dir, PackageSignatureFile))
	if err != nil {
		return fmt.Errorf("crypto: %w", err)
	}
	var wrap AAPSignature
	if err := json.Unmarshal(sb, &wrap); err != nil {
		return err
	}
	if wrap.Algorithm != "ed25519" {
		return fmt.Errorf("crypto: unsupported algorithm %q", wrap.Algorithm)
	}
	sig, err := base64.StdEncoding.DecodeString(wrap.Signature)
	if err != nil {
		return err
	}
	pubBytes, err := base64.StdEncoding.DecodeString(wrap.PublicKey)
	if err != nil {
		return err
	}
	return VerifyPackage(man, meta.LayerDigests, sig, ed25519.PublicKey(pubBytes))
}
