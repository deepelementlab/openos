package registry

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSignVerifyRoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	man := []byte(`{"metadata":{"name":"x"}}`)
	layers := []string{"deadbeef", "cafe"}
	sig, err := SignPackage(man, layers, priv)
	require.NoError(t, err)
	require.NoError(t, VerifyPackage(man, layers, sig, pub))
}

func TestVerifyPackage_WrongKey(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	wrongPub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	man := []byte(`{}`)
	sig, err := SignPackage(man, []string{"a"}, priv)
	require.NoError(t, err)
	err = VerifyPackage(man, []string{"a"}, sig, wrongPub)
	require.Error(t, err)
}

func TestVerifyPackage_TamperedManifest(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	man := []byte(`{"k":1}`)
	sig, err := SignPackage(man, []string{"a"}, priv)
	require.NoError(t, err)
	man2 := []byte(`{"k":2}`)
	err = VerifyPackage(man2, []string{"a"}, sig, pub)
	require.Error(t, err)
}

func TestVerifyPackageDir_MissingSignature(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "layers.json"), []byte(`{"layer_digests":[]}`), 0o644))
	err := VerifyPackageDir(dir)
	require.Error(t, err)
}

func TestVerifyPackageDir(t *testing.T) {
	dir := t.TempDir()
	man := []byte(`{"metadata":{"name":"x","version":"1"}}`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), man, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "layers.json"), []byte(`{"layer_digests":["a"]}`), 0o644))
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	require.NoError(t, WriteSignature(dir, man, []string{"a"}, priv))
	require.NoError(t, VerifyPackageDir(dir))
	_ = pub
}
