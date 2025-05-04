package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"math"
	"math/big"
)

// GenerateKeyPair generates a new ECDSA key pair
func GenerateKeyPair() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// SignData signs data with a private key
func SignData(data []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return nil, err
	}

	signature := append(r.Bytes(), s.Bytes()...)
	return signature, nil
}

// VerifySignature verifies a signature
func VerifySignature(data, signature, publicKey []byte) bool {
	if len(signature) != 64 {
		return false
	}

	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])

	pubKey := new(ecdsa.PublicKey)
	pubKey.Curve = elliptic.P256()
	pubKey.X = new(big.Int).SetBytes(publicKey[:32])
	pubKey.Y = new(big.Int).SetBytes(publicKey[32:])

	hash := sha256.Sum256(data)
	return ecdsa.Verify(pubKey, hash[:], r, s)
}

// CreateHomomorphicCommitment creates a homomorphic commitment
func CreateHomomorphicCommitment(data []byte) []byte {
	// Simplified Pedersen commitment
	g := big.NewInt(2)
	h := big.NewInt(3)
	p, _ := big.NewInt(0).SetString("FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1", 16)

	// Convert data to big.Int
	value := new(big.Int).SetBytes(data)

	// Generate random blinding factor
	r, _ := rand.Int(rand.Reader, p)

	// Compute commitment: C = g^value * h^r mod p
	gv := new(big.Int).Exp(g, value, p)
	hr := new(big.Int).Exp(h, r, p)
	commitment := new(big.Int).Mul(gv, hr)
	commitment.Mod(commitment, p)

	return commitment.Bytes()
}

// AMQFilter implements an Approximate Membership Query filter
type AMQFilter struct {
	bitArray      []bool
	numHashes     int
	size          int
	hashFunctions []func([]byte) uint64
}

// NewAMQFilter creates a new AMQ filter
func NewAMQFilter(falsePositiveRate float64, numHashes int) *AMQFilter {
	size := calculateOptimalSize(1000, falsePositiveRate)

	filter := &AMQFilter{
		bitArray:  make([]bool, size),
		numHashes: numHashes,
		size:      size,
	}

	// Create hash functions
	filter.hashFunctions = make([]func([]byte) uint64, numHashes)
	for i := 0; i < numHashes; i++ {
		seed := uint64(i)
		filter.hashFunctions[i] = func(data []byte) uint64 {
			h := sha256.Sum256(append(data, byte(seed)))
			return binary.BigEndian.Uint64(h[:8])
		}
	}

	return filter
}

// Add adds an element to the filter
func (f *AMQFilter) Add(data []byte) {
	for _, hashFunc := range f.hashFunctions {
		index := hashFunc(data) % uint64(f.size)
		f.bitArray[index] = true
	}
}

// Contains checks if an element might be in the filter
func (f *AMQFilter) Contains(data []byte) bool {
	for _, hashFunc := range f.hashFunctions {
		index := hashFunc(data) % uint64(f.size)
		if !f.bitArray[index] {
			return false
		}
	}
	return true
}

// calculateOptimalSize calculates the optimal filter size
func calculateOptimalSize(n int, p float64) int {
	m := -float64(n) * math.Log(p) / (math.Log(2) * math.Log(2))
	return int(math.Ceil(m))
}
