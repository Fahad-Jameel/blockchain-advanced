package crypto

import (
	"math/big"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	key, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	if key == nil {
		t.Error("Generated key is nil")
	}
}

func TestSignAndVerify(t *testing.T) {
	key, _ := GenerateKeyPair()
	data := []byte("test data")

	signature, err := SignData(data, key)
	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}

	// Serialize public key
	pubKeyBytes := append(key.PublicKey.X.Bytes(), key.PublicKey.Y.Bytes()...)

	// Verify signature
	valid := VerifySignature(data, signature, pubKeyBytes)
	if !valid {
		t.Error("Signature verification failed")
	}

	// Test with wrong data
	wrongData := []byte("wrong data")
	valid = VerifySignature(wrongData, signature, pubKeyBytes)
	if valid {
		t.Error("Signature verified with wrong data")
	}
}

func TestAccumulator(t *testing.T) {
	acc := NewAccumulator()

	// Add elements
	element1 := []byte("element1")
	element2 := []byte("element2")

	acc.Add(element1)
	acc.Add(element2)

	// Verify membership
	witness := acc.witnesses[string(element1)].Witness
	if !acc.Verify(element1, witness) {
		t.Error("Failed to verify element membership")
	}
}

func TestVRF(t *testing.T) {
	vrf := NewVRFOracle()

	seed := uint64(12345)
	output, proof := vrf.Generate(seed)

	if len(output) == 0 {
		t.Error("VRF output is empty")
	}

	if len(proof) == 0 {
		t.Error("VRF proof is empty")
	}
}

func TestZKP(t *testing.T) {
	zkp := NewZKPSystem()

	secret := big.NewInt(42)
	statement := big.NewInt(100)

	proof, err := zkp.GenerateProof(secret, statement)
	if err != nil {
		t.Fatalf("Failed to generate ZKP: %v", err)
	}

	valid := zkp.VerifyProof(proof, statement)
	if !valid {
		t.Error("ZKP verification failed")
	}
}

func TestAMQFilter(t *testing.T) {
	filter := NewAMQFilter(0.01, 4)

	// Add elements
	element1 := []byte("element1")
	element2 := []byte("element2")

	filter.Add(element1)
	filter.Add(element2)

	// Check membership
	if !filter.Contains(element1) {
		t.Error("Filter doesn't contain element1")
	}

	if !filter.Contains(element2) {
		t.Error("Filter doesn't contain element2")
	}

	// Test non-existent element
	nonExistent := []byte("non_existent")
	if filter.Contains(nonExistent) {
		t.Error("False positive for non-existent element")
	}
}

func TestHomomorphicCommitment(t *testing.T) {
	data := []byte("test data")
	commitment := CreateHomomorphicCommitment(data)

	if len(commitment) == 0 {
		t.Error("Commitment is empty")
	}

	// Test deterministic property (same data should produce different commitments due to random blinding)
	commitment2 := CreateHomomorphicCommitment(data)
	if string(commitment) == string(commitment2) {
		t.Error("Commitments should be different due to random blinding")
	}
}
