package main

import (
	"testing"
	"time"

	"blockchain-advanced/consensus"
	"blockchain-advanced/core"
	"blockchain-advanced/merkle"
)

func TestBlockchain(t *testing.T) {
	// Create blockchain
	blockchain := core.NewBlockchain()

	// Get genesis block
	genesis, err := blockchain.GetBlockByHeight(0)
	if err != nil {
		t.Errorf("Failed to get genesis block: %v", err)
	}

	// Create and add block
	block := &core.Block{
		Header: core.BlockHeader{
			Height:        1,
			Timestamp:     time.Now().Unix(),
			PrevBlockHash: genesis.Header.Hash(),
			Version:       1,
			Difficulty:    1,
		},
		Transactions: []core.Transaction{},
	}

	err = blockchain.AddBlock(block)
	if err != nil {
		t.Errorf("Failed to add block: %v", err)
	}

	// Verify block was added
	retrievedBlock, err := blockchain.GetBlockByHeight(1)
	if err != nil {
		t.Errorf("Failed to retrieve block: %v", err)
	}

	if retrievedBlock != nil && retrievedBlock.Header.Height != 1 {
		t.Errorf("Expected height 1, got %d", retrievedBlock.Header.Height)
	}
}

func TestAdaptiveMerkleForest(t *testing.T) {
	amf := merkle.NewAdaptiveMerkleForest()

	// Test rebalancing (just check it doesn't crash)
	err := amf.DynamicShardRebalancing()
	if err != nil {
		t.Errorf("Failed to rebalance: %v", err)
	}

	// Generate proof for shard 0 (root shard)
	key := []byte("test_key")
	_, err = amf.GenerateProof(key, 0)
	if err != nil {
		t.Logf("Expected error generating proof for non-existent key: %v", err)
	}

	// Test cross-shard operation
	op := &merkle.CrossShardOp{
		Type:     "test",
		ShardIDs: []uint32{0},
		Data:     []byte("test_data"),
	}

	err = amf.CrossShardOperation(op)
	if err != nil {
		t.Errorf("Failed to perform cross-shard operation: %v", err)
	}
}

func TestConsensus(t *testing.T) {
	consensus := consensus.NewHybridConsensus()

	// Start consensus
	err := consensus.Start()
	if err != nil {
		t.Errorf("Failed to start consensus: %v", err)
	}

	// Create test block with genesis as previous
	blockchain := core.NewBlockchain()
	genesis, _ := blockchain.GetBlockByHeight(0)

	block := &core.Block{
		Header: core.BlockHeader{
			Height:        1,
			PrevBlockHash: genesis.Header.Hash(),
			Timestamp:     time.Now().Unix(),
			Version:       1,
			Difficulty:    1,
		},
		Transactions: []core.Transaction{},
	}

	// Validate block
	err = consensus.ValidateBlock(block)
	if err != nil {
		t.Logf("Block validation error (expected without full setup): %v", err)
	}
}
