package state

import (
	"blockchain-advanced/core"
	"testing"
	"time"
)

func TestNewStateManager(t *testing.T) {
	sm := NewStateManager()

	if sm == nil {
		t.Fatal("NewStateManager returned nil")
	}

	if sm.currentState == nil {
		t.Error("Current state not initialized")
	}
}

func TestStateUpdate(t *testing.T) {
	sm := NewStateManager()

	// Create test account
	addr := core.Address{1, 2, 3}
	sm.currentState.Accounts[addr] = &core.AccountState{
		Balance: 1000,
		Nonce:   0,
	}

	// Create test transaction
	tx := core.Transaction{
		From:  addr,
		To:    core.Address{4, 5, 6},
		Value: 100,
		Nonce: 0,
	}

	// Create block with transaction
	block := &core.Block{
		Header: core.BlockHeader{
			Height: 1,
		},
		Transactions: []core.Transaction{tx},
	}

	// Update state
	err := sm.UpdateState(block)
	if err != nil {
		t.Errorf("Failed to update state: %v", err)
	}

	// Check balances
	sender := sm.currentState.Accounts[tx.From]
	if sender.Balance != 900 {
		t.Errorf("Expected sender balance 900, got %d", sender.Balance)
	}

	receiver := sm.currentState.Accounts[tx.To]
	if receiver.Balance != 100 {
		t.Errorf("Expected receiver balance 100, got %d", receiver.Balance)
	}
}

func TestStateCompression(t *testing.T) {
	compressor := NewStateCompressor()

	// Create test state
	state := &WorldState{
		Accounts: map[core.Address]*core.AccountState{
			core.Address{1}: {Balance: 100},
		},
		Timestamp: time.Now(),
	}

	// Compress state
	compressed, err := compressor.CompressWorldState(state)
	if err != nil {
		t.Errorf("Failed to compress state: %v", err)
	}

	if len(compressed) == 0 {
		t.Error("Compressed data is empty")
	}
}

func TestStateArchival(t *testing.T) {
	archive := NewStateArchive()

	// Create test state
	state := &WorldState{
		Timestamp: time.Now(),
	}

	// Archive state
	err := archive.Archive(state)
	if err != nil {
		t.Errorf("Failed to archive state: %v", err)
	}

	// Retrieve state
	retrieved, err := archive.Retrieve(uint64(state.Timestamp.Unix()))
	if err != nil {
		t.Errorf("Failed to retrieve state: %v", err)
	}

	if retrieved == nil {
		t.Error("Retrieved state is nil")
	}
}

func TestMemoryStateDB(t *testing.T) {
	db := NewMemoryStateDB()

	// Test Put
	key := []byte("test_key")
	value := []byte("test_value")

	err := db.Put(key, value)
	if err != nil {
		t.Errorf("Failed to put value: %v", err)
	}

	// Test Get
	retrieved, err := db.Get(key)
	if err != nil {
		t.Errorf("Failed to get value: %v", err)
	}

	if string(retrieved) != string(value) {
		t.Errorf("Expected %s, got %s", string(value), string(retrieved))
	}

	// Test Has
	exists, err := db.Has(key)
	if err != nil {
		t.Errorf("Failed to check existence: %v", err)
	}

	if !exists {
		t.Error("Key should exist")
	}

	// Test Delete
	err = db.Delete(key)
	if err != nil {
		t.Errorf("Failed to delete key: %v", err)
	}

	exists, err = db.Has(key)
	if err != nil {
		t.Errorf("Failed to check existence: %v", err)
	}

	if exists {
		t.Error("Key should not exist after deletion")
	}
}

func TestStatePruning(t *testing.T) {
	pruner := NewStatePruner()

	// Create test state history
	stateHistory := map[uint64]*StateSnapshot{
		1: {Height: 1, Timestamp: time.Now().Add(-40 * 24 * time.Hour)}, // Old
		2: {Height: 2, Timestamp: time.Now().Add(-20 * 24 * time.Hour)}, // Recent
		3: {Height: 3, Timestamp: time.Now()},                           // Current
	}

	// Prune states before height 3
	pruned, err := pruner.Prune(stateHistory, 3)
	if err != nil {
		t.Errorf("Failed to prune: %v", err)
	}

	// Check if old state was pruned
	foundOld := false
	for _, height := range pruned {
		if height == 1 {
			foundOld = true
			break
		}
	}

	if !foundOld {
		t.Error("Old state should have been pruned")
	}
}
