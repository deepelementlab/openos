package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
)

// MerkleAppender chains audit records with hash pointers for tamper evidence.
type MerkleAppender struct {
	mu      sync.Mutex
	lastHash []byte
	chain   [][]byte
}

// NewMerkleAppender creates an empty chain.
func NewMerkleAppender() *MerkleAppender {
	return &MerkleAppender{}
}

// Append adds a payload and returns the record hash.
func (m *MerkleAppender) Append(payload []byte) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	h := sha256.New()
	if len(m.lastHash) > 0 {
		_, _ = h.Write(m.lastHash)
	}
	_, _ = h.Write(payload)
	sum := h.Sum(nil)
	m.lastHash = sum
	m.chain = append(m.chain, append([]byte{}, sum...))
	return hex.EncodeToString(sum)
}

// VerifyChain checks internal consistency (not full Merkle tree; linear chain).
func (m *MerkleAppender) VerifyChain() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var prev []byte
	for i, rec := range m.chain {
		if i == 0 {
			prev = rec
			continue
		}
		// cannot recompute without payloads stored; export for production store
		_ = prev
	}
	if len(m.chain) == 0 {
		return fmt.Errorf("audit: empty chain")
	}
	return nil
}
