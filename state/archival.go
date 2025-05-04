package state

import (
	"blockchain-advanced/core"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// StateArchive handles state archival
type StateArchive struct {
	mu           sync.RWMutex
	archivedData map[uint64][]byte
}

func NewStateArchive() *StateArchive {
	return &StateArchive{
		archivedData: make(map[uint64][]byte),
	}
}

func (sa *StateArchive) Archive(state *WorldState) error {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	// Fix: Create a serializable version of the state
	serializableState := struct {
		StateRoot       string                        `json:"stateRoot"`
		Accounts        map[string]*core.AccountState `json:"accounts"`
		Contracts       map[string]*ContractState     `json:"contracts"`
		Shards          map[uint32]*core.ShardState   `json:"shards"`
		AccumulatorRoot string                        `json:"accumulatorRoot"`
		Timestamp       time.Time                     `json:"timestamp"`
	}{
		StateRoot:       state.StateRoot.String(),
		Accounts:        make(map[string]*core.AccountState),
		Contracts:       make(map[string]*ContractState),
		Shards:          state.Shards,
		AccumulatorRoot: state.AccumulatorRoot.String(),
		Timestamp:       state.Timestamp,
	}

	// Convert Address keys to string for JSON serialization
	for addr, account := range state.Accounts {
		serializableState.Accounts[addr.String()] = account
	}
	for addr, contract := range state.Contracts {
		serializableState.Contracts[addr.String()] = contract
	}

	// Serialize the state
	data, err := json.Marshal(serializableState)
	if err != nil {
		return err
	}

	// Store by timestamp
	sa.archivedData[uint64(state.Timestamp.Unix())] = data
	return nil
}

func (sa *StateArchive) Retrieve(height uint64) (*WorldState, error) {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	data, exists := sa.archivedData[height]
	if !exists {
		return nil, errors.New("archived state not found")
	}

	var serializableState struct {
		StateRoot       string                        `json:"stateRoot"`
		Accounts        map[string]*core.AccountState `json:"accounts"`
		Contracts       map[string]*ContractState     `json:"contracts"`
		Shards          map[uint32]*core.ShardState   `json:"shards"`
		AccumulatorRoot string                        `json:"accumulatorRoot"`
		Timestamp       time.Time                     `json:"timestamp"`
	}

	if err := json.Unmarshal(data, &serializableState); err != nil {
		return nil, err
	}

	// Convert back to proper WorldState
	state := &WorldState{
		Accounts:  make(map[core.Address]*core.AccountState),
		Contracts: make(map[core.Address]*ContractState),
		Shards:    serializableState.Shards,
		Timestamp: serializableState.Timestamp,
	}

	// Parse state root from hex string
	stateRootBytes, err := hex.DecodeString(serializableState.StateRoot)
	if err == nil && len(stateRootBytes) == 32 {
		copy(state.StateRoot[:], stateRootBytes)
	}

	// Parse accumulator root from hex string
	accumulatorRootBytes, err := hex.DecodeString(serializableState.AccumulatorRoot)
	if err == nil && len(accumulatorRootBytes) == 32 {
		copy(state.AccumulatorRoot[:], accumulatorRootBytes)
	}

	// Convert string keys back to Address
	for addrStr, account := range serializableState.Accounts {
		// Parse address from string (remove the 0x prefix if present)
		if len(addrStr) >= 2 && addrStr[:2] == "0x" {
			addrStr = addrStr[2:]
		}

		// Decode hex string to bytes
		addrBytes, err := hex.DecodeString(addrStr)
		if err != nil || len(addrBytes) != 20 {
			continue // Skip invalid addresses
		}

		var addr core.Address
		copy(addr[:], addrBytes)
		state.Accounts[addr] = account
	}

	// Convert string keys back to Address for contracts
	for addrStr, contract := range serializableState.Contracts {
		// Parse address from string (remove the 0x prefix if present)
		if len(addrStr) >= 2 && addrStr[:2] == "0x" {
			addrStr = addrStr[2:]
		}

		// Decode hex string to bytes
		addrBytes, err := hex.DecodeString(addrStr)
		if err != nil || len(addrBytes) != 20 {
			continue // Skip invalid addresses
		}

		var addr core.Address
		copy(addr[:], addrBytes)
		state.Contracts[addr] = contract
	}

	return state, nil
}

// StatePruner handles state pruning
type StatePruner struct {
	retentionPeriod time.Duration
}

func NewStatePruner() *StatePruner {
	return &StatePruner{
		retentionPeriod: 30 * 24 * time.Hour, // 30 days
	}
}

func (sp *StatePruner) Prune(stateHistory map[uint64]*StateSnapshot, beforeHeight uint64) ([]uint64, error) {
	pruned := make([]uint64, 0)
	now := time.Now()

	for height, snapshot := range stateHistory {
		if height < beforeHeight && !snapshot.Archived {
			// Check if old enough to prune
			if now.Sub(snapshot.Timestamp) > sp.retentionPeriod {
				pruned = append(pruned, height)
			}
		}
	}

	return pruned, nil
}
