package state

import (
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

	// Serialize the state
	data, err := json.Marshal(state)
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

	var state WorldState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
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
