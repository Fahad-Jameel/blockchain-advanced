package crypto

import (
	"crypto/sha256"
)

// Sign creates a simple signature for data using a private key
func Sign(data []byte, privateKey []byte) []byte {
	// Simplified signing for demo
	combined := append(data, privateKey...)
	hash := sha256.Sum256(combined)
	return hash[:]
}
