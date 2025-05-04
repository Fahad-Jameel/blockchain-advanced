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
	// Use smaller modulus for testing
	modulus := big.NewInt(257) // A prime number

	return &Accumulator{
		value:     big.NewInt(3), // Start with generator
		modulus:   modulus,
		witnesses: make(map[string]*AccumulatorWitness),
	}
}

// Add adds an element to the accumulator
func (acc *Accumulator) Add(element []byte) {
	// Hash element to get a prime
	prime := hashToPrime(element)

	// Store witness BEFORE updating accumulator
	witness := new(big.Int).Set(acc.value)

	// Update accumulator value: value = value^prime mod modulus
	acc.value.Exp(acc.value, prime, acc.modulus)

	// Store witness
	acc.witnesses[string(element)] = &AccumulatorWitness{
		Element: element,
		Witness: witness,
	}
}

// Verify verifies an element's membership
func (acc *Accumulator) Verify(element []byte, witness *big.Int) bool {
	if witness == nil || acc.value == nil {
		return false
	}

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
	// Use only part of hash to keep the number small
	num := new(big.Int).SetBytes(hash[:8])

	// Ensure reasonable size
	num.Mod(num, big.NewInt(100))
	num.Add(num, big.NewInt(2))

	// Find next prime
	for !num.ProbablyPrime(20) {
		num.Add(num, big.NewInt(1))
	}

	return num
}
