package network

import (
	"blockchain-advanced/consensus"
	"blockchain-advanced/core"
	"blockchain-advanced/merkle"
	"blockchain-advanced/state"
	"net"
	"testing"
	"time"
)

func TestNewNode(t *testing.T) {
	config := DefaultConfig()
	blockchain := core.NewBlockchain()
	amf := merkle.NewAdaptiveMerkleForest()
	consensus := consensus.NewHybridConsensus()
	stateManager := state.NewStateManager()

	node := NewNode(config, blockchain, amf, consensus, stateManager)

	if node == nil {
		t.Fatal("NewNode returned nil")
	}

	if node.config.NodeID == "" {
		t.Error("Node ID not generated")
	}
}

func TestMempool(t *testing.T) {
	mempool := NewMempool(10)

	// Add transaction
	tx := &core.Transaction{
		ID:    core.Hash{1, 2, 3},
		Value: 100,
	}

	err := mempool.AddTransaction(tx)
	if err != nil {
		t.Errorf("Failed to add transaction: %v", err)
	}

	// Get transactions
	txs := mempool.GetTransactions(5)
	if len(txs) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(txs))
	}

	// Remove transaction
	mempool.RemoveTransaction(tx.ID)
	txs = mempool.GetTransactions(5)
	if len(txs) != 0 {
		t.Errorf("Expected 0 transactions, got %d", len(txs))
	}
}

func TestP2PNetwork(t *testing.T) {
	config := DefaultConfig()
	p2pNet := NewP2PNetwork(config)

	// Create test listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	err = p2pNet.Start(listener)
	if err != nil {
		t.Errorf("Failed to start P2P network: %v", err)
	}

	p2pNet.Stop()
}

func TestHandshake(t *testing.T) {
	// Create server node
	serverConfig := DefaultConfig()
	serverConfig.ListenAddr = "127.0.0.1:0"
	serverNode := createTestNode(serverConfig)

	// Create client node
	clientConfig := DefaultConfig()
	clientConfig.ListenAddr = "127.0.0.1:0"
	clientNode := createTestNode(clientConfig)

	// Start nodes
	go serverNode.Start()
	go clientNode.Start()

	// Wait for startup
	time.Sleep(100 * time.Millisecond)

	// Clean up
	serverNode.Stop()
	clientNode.Stop()
}

func createTestNode(config *NodeConfig) *Node {
	blockchain := core.NewBlockchain()
	amf := merkle.NewAdaptiveMerkleForest()
	consensus := consensus.NewHybridConsensus()
	stateManager := state.NewStateManager()

	return NewNode(config, blockchain, amf, consensus, stateManager)
}

func TestSyncManager(t *testing.T) {
	syncMgr := NewSyncManager()

	// Create test node
	config := DefaultConfig()
	node := createTestNode(config)

	// Start sync manager
	syncMgr.Start(node)

	// Wait for sync cycle
	time.Sleep(100 * time.Millisecond)

	syncMgr.Stop()
}
