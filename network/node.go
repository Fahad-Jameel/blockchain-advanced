package network

import (
	"crypto/rand"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"time"

	"blockchain-advanced/cap"
	"blockchain-advanced/consensus"
	"blockchain-advanced/core"
	"blockchain-advanced/merkle"
	"blockchain-advanced/state"
)

// Node represents a blockchain network node
type Node struct {
	mu              sync.RWMutex
	config          *NodeConfig
	blockchain      *core.Blockchain
	merkleForest    *merkle.AdaptiveMerkleForest
	consensus       *consensus.HybridConsensus
	stateManager    *state.StateManager
	capOrchestrator *cap.CAPOrchestrator

	p2pNetwork  *P2PNetwork
	syncManager *SyncManager

	peers    map[string]*Peer
	mempool  *Mempool
	running  bool
	stopChan chan struct{}
}

// NodeConfig holds node configuration
type NodeConfig struct {
	ListenAddr      string
	MaxPeers        int
	NetworkID       uint32
	NodeID          string
	BootstrapPeers  []string
	ConsensusConfig *consensus.ConsensusConfig
}

// Peer represents a connected peer
type Peer struct {
	ID              string
	Address         string
	Connection      net.Conn
	Version         uint32
	LastSeen        time.Time
	ReputationScore float64
}

// Mempool manages pending transactions
type Mempool struct {
	mu           sync.RWMutex
	transactions map[core.Hash]*core.Transaction
	maxSize      int
}

// NewMempool creates a new mempool
func NewMempool(maxSize int) *Mempool {
	return &Mempool{
		transactions: make(map[core.Hash]*core.Transaction),
		maxSize:      maxSize,
	}
}

// AddTransaction adds a transaction to the mempool
func (m *Mempool) AddTransaction(tx *core.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.transactions) >= m.maxSize {
		return errors.New("mempool full")
	}

	m.transactions[tx.ID] = tx
	return nil
}

// GetTransactions returns a batch of transactions from the mempool
func (m *Mempool) GetTransactions(limit int) []*core.Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	txs := make([]*core.Transaction, 0, limit)
	count := 0

	for _, tx := range m.transactions {
		if count >= limit {
			break
		}
		txs = append(txs, tx)
		count++
	}

	return txs
}

// RemoveTransaction removes a transaction from the mempool
func (m *Mempool) RemoveTransaction(txID core.Hash) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.transactions, txID)
}

// NewNode creates a new blockchain node
func NewNode(config *NodeConfig, blockchain *core.Blockchain,
	merkleForest *merkle.AdaptiveMerkleForest, consensus *consensus.HybridConsensus,
	stateManager *state.StateManager) *Node {

	return &Node{
		config:          config,
		blockchain:      blockchain,
		merkleForest:    merkleForest,
		consensus:       consensus,
		stateManager:    stateManager,
		capOrchestrator: cap.NewCAPOrchestrator(),
		p2pNetwork:      NewP2PNetwork(config),
		syncManager:     NewSyncManager(),
		peers:           make(map[string]*Peer),
		mempool:         NewMempool(10000),
		stopChan:        make(chan struct{}),
	}
}

// Start starts the node
func (n *Node) Start() error {
	n.mu.Lock()
	if n.running {
		n.mu.Unlock()
		return errors.New("node already running")
	}
	n.running = true
	n.mu.Unlock()

	// Start listening for connections
	listener, err := net.Listen("tcp", n.config.ListenAddr)
	if err != nil {
		return err
	}

	// Start P2P network
	if err := n.p2pNetwork.Start(listener); err != nil {
		return err
	}

	// Start consensus engine
	if err := n.consensus.Start(); err != nil {
		return err
	}

	// Start CAP orchestrator
	if err := n.capOrchestrator.Start(); err != nil {
		return err
	}

	// Start sync manager
	go n.syncManager.Start(n)

	// Start accepting connections
	go n.acceptConnections(listener)

	// Start main loop
	go n.mainLoop()

	// Connect to bootstrap peers
	n.connectToBootstrapPeers()

	return nil
}

// Stop stops the node
func (n *Node) Stop() {
	n.mu.Lock()
	if !n.running {
		n.mu.Unlock()
		return
	}
	n.running = false
	n.mu.Unlock()

	close(n.stopChan)

	// Stop components
	n.p2pNetwork.Stop()
	n.syncManager.Stop()

	// Close peer connections
	n.mu.Lock()
	for _, peer := range n.peers {
		peer.Connection.Close()
	}
	n.mu.Unlock()
}

// acceptConnections accepts incoming peer connections
func (n *Node) acceptConnections(listener net.Listener) {
	for {
		select {
		case <-n.stopChan:
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				continue
			}

			go n.handleNewConnection(conn)
		}
	}
}

// handleNewConnection handles a new peer connection
func (n *Node) handleNewConnection(conn net.Conn) {
	peer := &Peer{
		Connection: conn,
		LastSeen:   time.Now(),
	}

	// Perform handshake
	if err := n.performHandshake(peer); err != nil {
		conn.Close()
		return
	}

	// Add peer
	n.mu.Lock()
	n.peers[peer.ID] = peer
	n.mu.Unlock()

	// Handle peer messages
	go n.handlePeer(peer)
}

// performHandshake performs the handshake with a peer
func (n *Node) performHandshake(peer *Peer) error {
	// Send version message
	versionMsg := &VersionMessage{
		Version:   1,
		NodeID:    n.config.NodeID,
		NetworkID: n.config.NetworkID,
		Height:    n.blockchain.GetCurrentHeight(),
	}

	if err := n.sendMessage(peer, "version", versionMsg); err != nil {
		return err
	}

	// Receive version message
	msg, err := n.receiveMessage(peer)
	if err != nil {
		return err
	}

	if msg.Type != "version" {
		return errors.New("expected version message")
	}

	var peerVersion VersionMessage
	if err := json.Unmarshal(msg.Payload, &peerVersion); err != nil {
		return err
	}

	// Verify network ID
	if peerVersion.NetworkID != n.config.NetworkID {
		return errors.New("network ID mismatch")
	}

	peer.ID = peerVersion.NodeID
	peer.Version = peerVersion.Version

	return nil
}

// mainLoop runs the main node loop
func (n *Node) mainLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			n.processRoutineTasks()
		case <-n.stopChan:
			return
		}
	}
}

// processRoutineTasks processes routine node tasks
func (n *Node) processRoutineTasks() {
	// Update peer connections
	n.updatePeerConnections()

	// Rebalance shards if needed
	if err := n.merkleForest.DynamicShardRebalancing(); err != nil {
		// Log error but continue
	}

	// Process pending transactions
	n.processPendingTransactions()

	// Update network metrics
	n.updateNetworkMetrics()
}

// handlePeer handles messages from a peer
func (n *Node) handlePeer(peer *Peer) {
	for {
		msg, err := n.receiveMessage(peer)
		if err != nil {
			n.disconnectPeer(peer)
			return
		}

		peer.LastSeen = time.Now()

		if err := n.handleMessage(peer, msg); err != nil {
			// Log error but continue
		}
	}
}

// handleMessage handles a message from a peer
func (n *Node) handleMessage(peer *Peer, msg *Message) error {
	switch msg.Type {
	case "block":
		var block core.Block
		if err := json.Unmarshal(msg.Payload, &block); err != nil {
			return err
		}
		return n.ProcessBlock(&block)

	case "transaction":
		var tx core.Transaction
		if err := json.Unmarshal(msg.Payload, &tx); err != nil {
			return err
		}
		return n.ProcessTransaction(&tx)

	case "consensus":
		var consensusMsg consensus.ConsensusMessage
		if err := json.Unmarshal(msg.Payload, &consensusMsg); err != nil {
			return err
		}
		return n.HandleConsensusMessage(&consensusMsg)

	case "getblocks":
		return n.handleGetBlocks(peer, msg)

	case "getdata":
		return n.handleGetData(peer, msg)

	default:
		return errors.New("unknown message type")
	}
}

// ProcessBlock processes a new block
func (n *Node) ProcessBlock(block *core.Block) error {
	// Validate block
	if err := n.consensus.ValidateBlock(block); err != nil {
		return err
	}

	// Add to blockchain
	if err := n.blockchain.AddBlock(block); err != nil {
		return err
	}

	// Update state
	if err := n.stateManager.UpdateState(block); err != nil {
		return err
	}

	// Remove processed transactions from mempool
	for _, tx := range block.Transactions {
		n.mempool.RemoveTransaction(tx.ID)
	}

	// Broadcast to peers
	n.broadcastBlock(block)

	return nil
}

// ProcessTransaction processes a new transaction
func (n *Node) ProcessTransaction(tx *core.Transaction) error {
	// Validate transaction
	if err := n.validateTransaction(tx); err != nil {
		return err
	}

	// Add to mempool
	if err := n.mempool.AddTransaction(tx); err != nil {
		return err
	}

	// Broadcast to peers
	n.broadcastTransaction(tx)

	return nil
}

// HandleConsensusMessage handles consensus messages
func (n *Node) HandleConsensusMessage(msg *consensus.ConsensusMessage) error {
	return n.consensus.ProcessConsensusMessage(msg)
}

// handleGetBlocks handles a request for blocks
func (n *Node) handleGetBlocks(peer *Peer, msg *Message) error {
	var request GetBlocksMessage
	if err := json.Unmarshal(msg.Payload, &request); err != nil {
		return err
	}

	blocks := make([]*core.Block, 0)
	for i := request.FromHeight; i <= request.ToHeight && i <= n.blockchain.GetCurrentHeight(); i++ {
		block, err := n.blockchain.GetBlockByHeight(i)
		if err != nil {
			continue
		}
		blocks = append(blocks, block)
	}

	return n.sendMessage(peer, "blocks", blocks)
}

// handleGetData handles a request for specific data
func (n *Node) handleGetData(peer *Peer, msg *Message) error {
	var request GetDataMessage
	if err := json.Unmarshal(msg.Payload, &request); err != nil {
		return err
	}

	switch request.Type {
	case "block":
		block, err := n.blockchain.GetBlock(request.Hash)
		if err != nil {
			return err
		}
		return n.sendMessage(peer, "block", block)

	case "transaction":
		// Get transaction from mempool or blockchain
		// Implementation left as an exercise
		return nil

	default:
		return errors.New("unknown data type")
	}
}

// connectToBootstrapPeers connects to bootstrap peers
func (n *Node) connectToBootstrapPeers() {
	for _, addr := range n.config.BootstrapPeers {
		go n.connectToPeer(addr)
	}
}

// connectToPeer connects to a specific peer
func (n *Node) connectToPeer(addr string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return
	}

	peer := &Peer{
		Address:    addr,
		Connection: conn,
		LastSeen:   time.Now(),
	}

	if err := n.performHandshake(peer); err != nil {
		conn.Close()
		return
	}

	n.mu.Lock()
	n.peers[peer.ID] = peer
	n.mu.Unlock()

	go n.handlePeer(peer)
}

// disconnectPeer disconnects from a peer
func (n *Node) disconnectPeer(peer *Peer) {
	n.mu.Lock()
	delete(n.peers, peer.ID)
	n.mu.Unlock()

	peer.Connection.Close()
}

// updatePeerConnections updates peer connections
func (n *Node) updatePeerConnections() {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Remove inactive peers
	for id, peer := range n.peers {
		if time.Since(peer.LastSeen) > 5*time.Minute {
			peer.Connection.Close()
			delete(n.peers, id)
		}
	}
}

// processPendingTransactions processes pending transactions
func (n *Node) processPendingTransactions() {
	// Get transactions from mempool
	txs := n.mempool.GetTransactions(100)
	if len(txs) == 0 {
		return
	}

	// Create new block
	block := &core.Block{
		Header: core.BlockHeader{
			Height:    n.blockchain.GetCurrentHeight() + 1,
			Timestamp: time.Now().Unix(),
		},
		Transactions: make([]core.Transaction, len(txs)),
	}

	// Copy transactions (dereference pointers)
	for i, tx := range txs {
		block.Transactions[i] = *tx
	}

	// Process block
	if err := n.ProcessBlock(block); err != nil {
		// Log error but continue
	}
}

// updateNetworkMetrics updates network metrics
func (n *Node) updateNetworkMetrics() {
	// Implementation for updating network metrics
	// This could include latency, bandwidth, peer count, etc.
}

// validateTransaction validates a transaction
func (n *Node) validateTransaction(tx *core.Transaction) error {
	// Get current state
	state, err := n.stateManager.GetCurrentState()
	if err != nil {
		return err
	}

	// Check sender account
	sender, exists := state.Accounts[tx.From]
	if !exists {
		return errors.New("sender account not found")
	}

	// Check balance
	if sender.Balance < tx.Value {
		return errors.New("insufficient balance")
	}

	// Check nonce
	if tx.Nonce != sender.Nonce {
		return errors.New("invalid nonce")
	}

	return nil
}

// broadcastBlock broadcasts a block to all peers
func (n *Node) broadcastBlock(block *core.Block) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, peer := range n.peers {
		go n.sendMessage(peer, "block", block)
	}
}

// broadcastTransaction broadcasts a transaction to all peers
func (n *Node) broadcastTransaction(tx *core.Transaction) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, peer := range n.peers {
		go n.sendMessage(peer, "transaction", tx)
	}
}

// sendMessage sends a message to a peer
func (n *Node) sendMessage(peer *Peer, msgType string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := &Message{
		Type:    msgType,
		Payload: data,
	}

	encoder := gob.NewEncoder(peer.Connection)
	return encoder.Encode(msg)
}

// receiveMessage receives a message from a peer
func (n *Node) receiveMessage(peer *Peer) (*Message, error) {
	var msg Message
	decoder := gob.NewDecoder(peer.Connection)

	if err := decoder.Decode(&msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

// Message types

type Message struct {
	Type    string
	Payload []byte
}

type VersionMessage struct {
	Version   uint32
	NodeID    string
	NetworkID uint32
	Height    uint64
}

type GetBlocksMessage struct {
	FromHeight uint64
	ToHeight   uint64
}

type GetDataMessage struct {
	Type string // "block" or "transaction"
	Hash core.Hash
}

// DefaultConfig returns default node configuration
func DefaultConfig() *NodeConfig {
	return &NodeConfig{
		ListenAddr: "0.0.0.0:8080",
		MaxPeers:   50,
		NetworkID:  1,
		NodeID:     generateNodeID(),
	}
}

// generateNodeID generates a unique node ID
func generateNodeID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
