package core

import (
	"testing"
	"time"
)

func TestNewBlockchain(t *testing.T) {
	bc := NewBlockchain()

	if bc == nil {
		t.Fatal("NewBlockchain returned nil")
	}

	if bc.genesis == nil {
		t.Error("Genesis block not created")
	}

	if bc.currentHeight != 0 {
		t.Errorf("Expected height 0, got %d", bc.currentHeight)
	}
}

func TestAddBlock(t *testing.T) {
	bc := NewBlockchain()
	genesis, _ := bc.GetBlockByHeight(0)

	// Create valid block
	block := &Block{
		Header: BlockHeader{
			Version:       1,
			Height:        1,
			Timestamp:     time.Now().Unix(),
			PrevBlockHash: genesis.Header.Hash(),
			Difficulty:    1,
		},
		Transactions: []Transaction{},
	}

	err := bc.AddBlock(block)
	if err != nil {
		t.Errorf("Failed to add valid block: %v", err)
	}

	// Create block with invalid height
	invalidBlock := &Block{
		Header: BlockHeader{
			Version:       1,
			Height:        3, // Skip height 2
			Timestamp:     time.Now().Unix(),
			PrevBlockHash: block.Header.Hash(),
			Difficulty:    1,
		},
	}

	err = bc.AddBlock(invalidBlock)
	if err == nil {
		t.Error("Expected error for invalid block, got nil")
	}
}

func TestGetBlock(t *testing.T) {
	bc := NewBlockchain()
	genesis, _ := bc.GetBlockByHeight(0)

	// Get existing block
	block, err := bc.GetBlock(genesis.Header.Hash())
	if err != nil {
		t.Errorf("Failed to get existing block: %v", err)
	}

	if block == nil {
		t.Error("Expected block, got nil")
	}

	// Get non-existent block
	nonExistentHash := Hash{}
	_, err = bc.GetBlock(nonExistentHash)
	if err == nil {
		t.Error("Expected error for non-existent block, got nil")
	}
}

func TestBlockValidation(t *testing.T) {
	bc := NewBlockchain()

	// Invalid block version
	block := &Block{
		Header: BlockHeader{
			Version: 0, // Invalid version
		},
	}

	err := bc.validateBlock(block)
	if err == nil {
		t.Error("Expected error for invalid version, got nil")
	}

	// Block timestamp too far in future
	futureBlock := &Block{
		Header: BlockHeader{
			Version:   1,
			Timestamp: time.Now().Unix() + 3600, // 1 hour in future
		},
	}

	err = bc.validateBlock(futureBlock)
	if err == nil {
		t.Error("Expected error for future timestamp, got nil")
	}
}

func TestOrphanBlocks(t *testing.T) {
	bc := NewBlockchain()

	// Create orphan block (missing parent)
	orphanBlock := &Block{
		Header: BlockHeader{
			Version:       1,
			Height:        2,
			PrevBlockHash: Hash{1, 2, 3}, // Non-existent parent
			Difficulty:    1,
		},
	}

	err := bc.AddBlock(orphanBlock)
	if err == nil {
		t.Error("Expected error for orphan block, got nil")
	}

	// Check if stored as orphan
	if len(bc.orphanBlocks) != 1 {
		t.Errorf("Expected 1 orphan block, got %d", len(bc.orphanBlocks))
	}
}
