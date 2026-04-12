// Package supplychain handles image/package signing and verification hooks.
package supplychain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Signer produces detached signatures (stub: integrate cosign/notation).
type Signer struct {
	KeyID string
}

// NewSigner creates a signer.
func NewSigner(keyID string) *Signer {
	return &Signer{KeyID: keyID}
}

// SignPayload returns a hex-encoded SHA256 of payload as a stub signature artifact.
func (s *Signer) SignPayload(ctx context.Context, payload []byte) (string, error) {
	_ = ctx
	if len(payload) == 0 {
		return "", fmt.Errorf("supplychain: empty payload")
	}
	h := sha256.Sum256(payload)
	return hex.EncodeToString(h[:]), nil
}

// VerifyPayload checks stub signature matches recomputed hash.
func VerifyPayload(ctx context.Context, payload []byte, signatureHex string) error {
	_ = ctx
	h := sha256.Sum256(payload)
	sig := hex.EncodeToString(h[:])
	if sig != signatureHex {
		return fmt.Errorf("supplychain: signature mismatch")
	}
	return nil
}
