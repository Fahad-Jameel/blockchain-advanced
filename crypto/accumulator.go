package crypto

import (
	"crypto/sha256"
	"math/big"
)

// Accumulator implements a cryptographic accumulator
type Accumulator struct {
	value     *big.Int
	modulus   *big.Int
	witnesses map[string]*AccumulatorWitness
}

// AccumulatorWitness represents a witness for an element
type AccumulatorWitness struct {
	Element []byte
	Witness *big.Int
}

// NewAccumulator creates a new accumulator
func NewAccumulator() *Accumulator {
	// Use a safe prime for the modulus
	modulus, _ := big.NewInt(0).SetString("FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF", 16)

	return &Accumulator{
		value:     big.NewInt(2), // Start with generator
		modulus:   modulus,
		witnesses: make(map[string]*AccumulatorWitness),
	}
}

// Add adds an element to the accumulator
func (acc *Accumulator) Add(element []byte) {
	// Hash element to get a prime
	prime := hashToPrime(element)

	// Update accumulator value: value = value^prime mod modulus
	acc.value.Exp(acc.value, prime, acc.modulus)

	// Store witness
	witness := &AccumulatorWitness{
		Element: element,
		Witness: new(big.Int).Set(acc.value),
	}
	acc.witnesses[string(element)] = witness
}

// Verify verifies an element's membership
func (acc *Accumulator) Verify(element []byte, witness *big.Int) bool {
	prime := hashToPrime(element)

	// Verify: witness^prime = accumulator_value mod modulus
	result := new(big.Int).Exp(witness, prime, acc.modulus)
	return result.Cmp(acc.value) == 0
}

// Root returns the current accumulator value
func (acc *Accumulator) Root() []byte {
	return acc.value.Bytes()
}

// hashToPrime hashes data to a prime number
func hashToPrime(data []byte) *big.Int {
	hash := sha256.Sum256(data)
	num := new(big.Int).SetBytes(hash[:])

	// Find next prime
	for !num.ProbablyPrime(20) {
		num.Add(num, big.NewInt(1))
	}

	return num
}
