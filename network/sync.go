package network

import (
	"sync"
	"time"
)

// SyncManager handles blockchain synchronization
type SyncManager struct {
	node     *Node
	syncing  bool
	mu       sync.Mutex
	stopChan chan struct{}
}

func NewSyncManager() *SyncManager {
	return &SyncManager{
		stopChan: make(chan struct{}),
	}
}

func (sm *SyncManager) Start(node *Node) {
	sm.node = node
	go sm.syncLoop()
}

func (sm *SyncManager) Stop() {
	close(sm.stopChan)
}

func (sm *SyncManager) syncLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.synchronize()
		case <-sm.stopChan:
			return
		}
	}
}

func (sm *SyncManager) synchronize() {
	sm.mu.Lock()
	if sm.syncing {
		sm.mu.Unlock()
		return
	}
	sm.syncing = true
	sm.mu.Unlock()

	defer func() {
		sm.mu.Lock()
		sm.syncing = false
		sm.mu.Unlock()
	}()

	// Check each peer's height
	for _, peer := range sm.node.peers {
		// Request peer's current height
		sm.requestBlockchainState(peer)
	}
}

func (sm *SyncManager) requestBlockchainState(peer *Peer) {
	// Send get blocks message
	request := GetBlocksMessage{
		FromHeight: sm.node.blockchain.GetCurrentHeight() + 1,
		ToHeight:   sm.node.blockchain.GetCurrentHeight() + 100,
	}

	sm.node.sendMessage(peer, "getblocks", request)
}
