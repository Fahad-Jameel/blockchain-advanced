package crypto

import (
	"crypto/rand"
	"math/big"
	"time"

	"blockchain-advanced/core"
)

// ZKPSystem implements zero-knowledge proof functionality
type ZKPSystem struct {
	params *ZKPParams
}

// ZKPParams holds zero-knowledge proof parameters
type ZKPParams struct {
	Prime     *big.Int
	Generator *big.Int
	Order     *big.Int
}

// ZKProof represents a zero-knowledge proof
type ZKProof struct {
	Commitment *big.Int
	Challenge  *big.Int
	Response   *big.Int
}

// NewZKPSystem creates a new ZKP system
func NewZKPSystem() *ZKPSystem {
	// Initialize with secure parameters
	prime, _ := rand.Prime(rand.Reader, 2048)
	generator := big.NewInt(2)
	order := new(big.Int).Sub(prime, big.NewInt(1))

	return &ZKPSystem{
		params: &ZKPParams{
			Prime:     prime,
			Generator: generator,
			Order:     order,
		},
	}
}

// GenerateProof generates a zero-knowledge proof for state verification
func (zkp *ZKPSystem) GenerateProof(secret *big.Int, statement *big.Int) (*ZKProof, error) {
	// Generate random commitment
	r, err := rand.Int(rand.Reader, zkp.params.Order)
	if err != nil {
		return nil, err
	}

	// Compute commitment: g^r mod p
	commitment := new(big.Int).Exp(zkp.params.Generator, r, zkp.params.Prime)

	// Generate challenge (should be from verifier in interactive protocol)
	challenge, err := rand.Int(rand.Reader, zkp.params.Order)
	if err != nil {
		return nil, err
	}

	// Compute response: r + challenge * secret mod q
	response := new(big.Int).Mul(challenge, secret)
	response.Add(response, r)
	response.Mod(response, zkp.params.Order)

	return &ZKProof{
		Commitment: commitment,
		Challenge:  challenge,
		Response:   response,
	}, nil
}

// VerifyProof verifies a zero-knowledge proof
func (zkp *ZKPSystem) VerifyProof(proof *ZKProof, statement *big.Int) bool {
	// Verify: g^response = commitment * statement^challenge mod p
	left := new(big.Int).Exp(zkp.params.Generator, proof.Response, zkp.params.Prime)

	right := new(big.Int).Exp(statement, proof.Challenge, zkp.params.Prime)
	right.Mul(right, proof.Commitment)
	right.Mod(right, zkp.params.Prime)

	return left.Cmp(right) == 0
}

// GenerateStateProof generates a ZKP for state verification
func (zkp *ZKPSystem) GenerateStateProof(stateRoot core.Hash, witness []byte) (*StateProof, error) {
	// Convert state root to big.Int
	stateValue := new(big.Int).SetBytes(stateRoot[:])

	// Create witness commitment
	witnessValue := new(big.Int).SetBytes(witness)

	// Generate proof
	proof, err := zkp.GenerateProof(witnessValue, stateValue)
	if err != nil {
		return nil, err
	}

	return &StateProof{
		StateRoot: stateRoot,
		Proof:     proof,
		Timestamp: time.Now().Unix(),
	}, nil
}

// VerifyStateProof verifies a state proof
func (zkp *ZKPSystem) VerifyStateProof(stateProof *StateProof) bool {
	// Convert state root to big.Int
	stateValue := new(big.Int).SetBytes(stateProof.StateRoot[:])

	// Verify the proof
	return zkp.VerifyProof(stateProof.Proof, stateValue)
}

// StateProof represents a zero-knowledge proof for state
type StateProof struct {
	StateRoot core.Hash
	Proof     *ZKProof
	Timestamp int64
}
