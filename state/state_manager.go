package state

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"blockchain-advanced/core"
	"blockchain-advanced/crypto"
)

// StateManager manages blockchain state with advanced features
type StateManager struct {
	mu           sync.RWMutex
	currentState *WorldState
	stateArchive *StateArchive
	compressor   *StateCompressor
	pruner       *StatePruner

	accumulators map[core.Hash]*crypto.Accumulator
	stateHistory map[uint64]*StateSnapshot
	stateDB      StateDatabase
}

// WorldState represents the current global state
type WorldState struct {
	StateRoot       core.Hash
	Accounts        map[core.Address]*core.AccountState
	Contracts       map[core.Address]*ContractState
	Shards          map[uint32]*core.ShardState
	AccumulatorRoot core.Hash
	Timestamp       time.Time
}

// ContractState represents a smart contract's state
type ContractState struct {
	CodeHash core.Hash
	Storage  map[core.Hash]core.Hash
	Balance  uint64
	Nonce    uint64
}

// StateSnapshot represents a state at a specific height
type StateSnapshot struct {
	Height     uint64
	StateRoot  core.Hash
	Timestamp  time.Time
	Compressed bool
	Archived   bool
}

// StateDatabase interface for state storage
type StateDatabase interface {
	Get(key []byte) ([]byte, error)
	Put(key []byte, value []byte) error
	Delete(key []byte) error
	Has(key []byte) (bool, error)
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{
		currentState: &WorldState{
			Accounts:  make(map[core.Address]*core.AccountState),
			Contracts: make(map[core.Address]*ContractState),
			Shards:    make(map[uint32]*core.ShardState),
			Timestamp: time.Now(),
		},
		stateArchive: NewStateArchive(),
		compressor:   NewStateCompressor(),
		pruner:       NewStatePruner(),
		accumulators: make(map[core.Hash]*crypto.Accumulator),
		stateHistory: make(map[uint64]*StateSnapshot),
		stateDB:      NewMemoryStateDB(),
	}
}

// GetCurrentState returns the current world state
func (sm *StateManager) GetCurrentState() (*WorldState, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.currentState, nil
}

// UpdateState updates the world state with new transactions
func (sm *StateManager) UpdateState(block *core.Block) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Create state copy for atomic updates
	newState := sm.copyState(sm.currentState)

	// Apply transactions
	for _, tx := range block.Transactions {
		if err := sm.applyTransaction(newState, &tx); err != nil {
			return fmt.Errorf("failed to apply transaction %s: %v", tx.ID, err)
		}
	}

	// Update accumulator
	accumulator := sm.updateAccumulator(newState)
	newState.AccumulatorRoot = core.Hash(accumulator.Root())

	// Calculate new state root
	newState.StateRoot = sm.calculateStateRoot(newState)
	newState.Timestamp = time.Now()

	// Create state snapshot
	snapshot := &StateSnapshot{
		Height:    block.Header.Height,
		StateRoot: newState.StateRoot,
		Timestamp: time.Now(),
	}

	// Compress old state if needed
	if sm.shouldCompress(block.Header.Height) {
		if err := sm.compressState(sm.currentState); err != nil {
			return err
		}
		snapshot.Compressed = true
	}

	// Archive old state if needed
	if sm.shouldArchive(block.Header.Height) {
		if err := sm.archiveState(sm.currentState); err != nil {
			return err
		}
		snapshot.Archived = true
	}

	// Save state to database
	if err := sm.saveState(newState); err != nil {
		return err
	}

	// Update current state
	sm.currentState = newState
	sm.stateHistory[block.Header.Height] = snapshot

	return nil
}

// applyTransaction applies a transaction to the state
func (sm *StateManager) applyTransaction(state *WorldState, tx *core.Transaction) error {
	// Get sender account
	sender, exists := state.Accounts[tx.From]
	if !exists {
		return errors.New("sender account not found")
	}

	// Check nonce
	if tx.Nonce != sender.Nonce {
		return errors.New("invalid nonce")
	}

	// Check balance
	if sender.Balance < tx.Value {
		return errors.New("insufficient balance")
	}

	// Deduct from sender
	sender.Balance -= tx.Value
	sender.Nonce++

	// Add to receiver
	receiver, exists := state.Accounts[tx.To]
	if !exists {
		receiver = &core.AccountState{
			Balance: 0,
			Nonce:   0,
			Storage: make(map[core.Hash]core.Hash),
		}
		state.Accounts[tx.To] = receiver
	}
	receiver.Balance += tx.Value

	return nil
}

// updateAccumulator updates the cryptographic accumulator
func (sm *StateManager) updateAccumulator(state *WorldState) *crypto.Accumulator {
	accumulator := crypto.NewAccumulator()

	// Add all account states to accumulator
	for addr, account := range state.Accounts {
		data := append(addr[:], sm.serializeAccountState(account)...)
		accumulator.Add(data)
	}

	// Add contract states
	for addr, contract := range state.Contracts {
		data := append(addr[:], sm.serializeContractState(contract)...)
		accumulator.Add(data)
	}

	return accumulator
}

// calculateStateRoot calculates the state root hash
func (sm *StateManager) calculateStateRoot(state *WorldState) core.Hash {
	var data []byte

	// Add account hashes
	for addr, account := range state.Accounts {
		data = append(data, addr[:]...)
		accountHash := sm.hashAccountState(account)
		data = append(data, accountHash[:]...)
	}

	// Add contract hashes
	for addr, contract := range state.Contracts {
		data = append(data, addr[:]...)
		contractHash := sm.hashContractState(contract)
		data = append(data, contractHash[:]...)
	}

	// Add shard hashes
	for id, shard := range state.Shards {
		idBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(idBytes, id)
		data = append(data, idBytes...)
		data = append(data, shard.StateRoot[:]...)
	}

	return core.ComputeHash(data)
}

// Helper methods

func (sm *StateManager) copyState(state *WorldState) *WorldState {
	newState := &WorldState{
		Accounts:  make(map[core.Address]*core.AccountState),
		Contracts: make(map[core.Address]*ContractState),
		Shards:    make(map[uint32]*core.ShardState),
		Timestamp: state.Timestamp,
	}

	// Deep copy accounts
	for addr, account := range state.Accounts {
		newAccount := &core.AccountState{
			Balance:  account.Balance,
			Nonce:    account.Nonce,
			CodeHash: account.CodeHash,
			Storage:  make(map[core.Hash]core.Hash),
		}
		for k, v := range account.Storage {
			newAccount.Storage[k] = v
		}
		newState.Accounts[addr] = newAccount
	}

	// Deep copy contracts
	for addr, contract := range state.Contracts {
		newContract := &ContractState{
			CodeHash: contract.CodeHash,
			Balance:  contract.Balance,
			Nonce:    contract.Nonce,
			Storage:  make(map[core.Hash]core.Hash),
		}
		for k, v := range contract.Storage {
			newContract.Storage[k] = v
		}
		newState.Contracts[addr] = newContract
	}

	// Deep copy shards
	for id, shard := range state.Shards {
		newShard := &core.ShardState{
			ShardID:      shard.ShardID,
			StateRoot:    shard.StateRoot,
			LastModified: shard.LastModified,
			Accounts:     make(map[core.Address]core.AccountState),
		}
		for addr, account := range shard.Accounts {
			newShard.Accounts[addr] = account
		}
		newState.Shards[id] = newShard
	}

	return newState
}

func (sm *StateManager) serializeAccountState(account *core.AccountState) []byte {
	data, _ := json.Marshal(account)
	return data
}

func (sm *StateManager) serializeContractState(contract *ContractState) []byte {
	data, _ := json.Marshal(contract)
	return data
}

func (sm *StateManager) hashAccountState(account *core.AccountState) core.Hash {
	data := sm.serializeAccountState(account)
	return core.ComputeHash(data)
}

func (sm *StateManager) hashContractState(contract *ContractState) core.Hash {
	data := sm.serializeContractState(contract)
	return core.ComputeHash(data)
}

func (sm *StateManager) saveState(state *WorldState) error {
	// Save state root
	rootKey := []byte("stateRoot")
	if err := sm.stateDB.Put(rootKey, state.StateRoot[:]); err != nil {
		return err
	}

	// Save accounts
	for addr, account := range state.Accounts {
		key := append([]byte("account:"), addr[:]...)
		data := sm.serializeAccountState(account)
		if err := sm.stateDB.Put(key, data); err != nil {
			return err
		}
	}

	// Save contracts
	for addr, contract := range state.Contracts {
		key := append([]byte("contract:"), addr[:]...)
		data := sm.serializeContractState(contract)
		if err := sm.stateDB.Put(key, data); err != nil {
			return err
		}
	}

	return nil
}

func (sm *StateManager) shouldCompress(height uint64) bool {
	return height%1000 == 0
}

func (sm *StateManager) shouldArchive(height uint64) bool {
	return height%10000 == 0
}

func (sm *StateManager) compressState(state *WorldState) error {
	compressed, err := sm.compressor.CompressWorldState(state)
	if err != nil {
		return err
	}

	key := []byte(fmt.Sprintf("compressed:%d", time.Now().Unix()))
	return sm.stateDB.Put(key, compressed)
}

func (sm *StateManager) archiveState(state *WorldState) error {
	return sm.stateArchive.Archive(state)
}

// GetState retrieves state at a specific height
func (sm *StateManager) GetState(height uint64) (*WorldState, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	snapshot, exists := sm.stateHistory[height]
	if !exists {
		return nil, errors.New("state not found")
	}

	// If archived, retrieve from archive
	if snapshot.Archived {
		return sm.stateArchive.Retrieve(height)
	}

	// If compressed, decompress
	if snapshot.Compressed {
		return sm.compressor.DecompressWorldState(snapshot)
	}

	// If current height, return current state
	if sm.currentState != nil && snapshot.StateRoot == sm.currentState.StateRoot {
		return sm.currentState, nil
	}

	return nil, errors.New("state not available")
}

// Supporting types

type StateProof struct {
	Address      core.Address
	AccountState *core.AccountState
	StateRoot    core.Hash
	Timestamp    time.Time
}

// MemoryStateDB is an in-memory state database
type MemoryStateDB struct {
	data map[string][]byte
	mu   sync.RWMutex
}

func NewMemoryStateDB() *MemoryStateDB {
	return &MemoryStateDB{
		data: make(map[string][]byte),
	}
}

func (db *MemoryStateDB) Get(key []byte) ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	value, exists := db.data[string(key)]
	if !exists {
		return nil, errors.New("key not found")
	}

	return value, nil
}

func (db *MemoryStateDB) Put(key []byte, value []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.data[string(key)] = value
	return nil
}

func (db *MemoryStateDB) Delete(key []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	delete(db.data, string(key))
	return nil
}

func (db *MemoryStateDB) Has(key []byte) (bool, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	_, exists := db.data[string(key)]
	return exists, nil
}
