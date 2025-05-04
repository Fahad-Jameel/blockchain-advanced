package merkle

import (
	"blockchain-advanced/core"
	"testing"
)

func TestNewAdaptiveMerkleForest(t *testing.T) {
	amf := NewAdaptiveMerkleForest()

	if amf == nil {
		t.Fatal("NewAdaptiveMerkleForest returned nil")
	}

	// Check root shard exists
	if _, exists := amf.shards[0]; !exists {
		t.Error("Root shard not initialized")
	}
}

func TestShardSplitting(t *testing.T) {
	amf := NewAdaptiveMerkleForest()

	// Set high load factor to trigger split
	amf.shards[0].LoadMetrics.LoadFactor = 0.95

	err := amf.DynamicShardRebalancing()
	if err != nil {
		t.Errorf("Failed to rebalance: %v", err)
	}

	// Check if shards were split
	if len(amf.shards) <= 1 {
		t.Error("Expected shard split, but no new shards created")
	}
}

func TestMerkleProofGeneration(t *testing.T) {
	amf := NewAdaptiveMerkleForest()

	// Add test data to root shard
	testKey := []byte("test_key")
	testValue := []byte("test_value")

	// Add leaf to tree
	leafHash := core.ComputeHash(append(testKey, testValue...))
	amf.shards[0].StateTree.AddLeaf(leafHash)

	// Generate proof
	proof, err := amf.GenerateProof(testKey, 0)
	if err != nil {
		t.Errorf("Failed to generate proof: %v", err)
	}

	// Verify proof
	if !amf.VerifyProof(proof, testKey, testValue) {
		t.Error("Proof verification failed")
	}
}

func TestCrossShardOperation(t *testing.T) {
	amf := NewAdaptiveMerkleForest()

	op := &CrossShardOp{
		Type:     "transfer",
		ShardIDs: []uint32{0},
		Data:     []byte("test_data"),
	}

	err := amf.CrossShardOperation(op)
	if err != nil {
		t.Errorf("Cross-shard operation failed: %v", err)
	}
}

func TestMerkleTree(t *testing.T) {
	tree := NewMerkleTree()

	// Add leaves
	leaf1 := core.ComputeHash([]byte("leaf1"))
	leaf2 := core.ComputeHash([]byte("leaf2"))
	leaf3 := core.ComputeHash([]byte("leaf3"))

	tree.AddLeaf(leaf1)
	tree.AddLeaf(leaf2)
	tree.AddLeaf(leaf3)

	// Check root is computed
	if tree.Root == (core.Hash{}) {
		t.Error("Merkle root not computed")
	}

	// Generate proof
	proof := tree.GenerateProof([]byte("leaf1"))
	if proof == nil {
		t.Error("Failed to generate proof")
	}
}

func TestProofCompression(t *testing.T) {
	pg := NewProofGenerator()

	// Create test proof
	proof := &MerkleProof{
		Path: []ProofNode{
			{Hash: core.Hash{1}, Position: Left},
			{Hash: core.Hash{2}, Position: Right},
			{Hash: core.Hash{3}, Position: Left},
		},
		Root:      core.Hash{4},
		ProofType: StandardProof,
	}

	// Compress proof
	compressed := pg.CompressProof(proof)
	if !compressed.Compressed {
		t.Error("Proof not marked as compressed")
	}

	// Decompress proof
	decompressed := pg.DecompressProof(compressed)
	if decompressed.Compressed {
		t.Error("Proof still marked as compressed after decompression")
	}
}
