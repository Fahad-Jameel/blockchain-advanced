package network

import (
	"net"
	"sync"
)

// P2PNetwork manages peer-to-peer networking
type P2PNetwork struct {
	config   *NodeConfig
	listener net.Listener
	peers    map[string]*Peer
	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}
}

func NewP2PNetwork(config *NodeConfig) *P2PNetwork {
	return &P2PNetwork{
		config:   config,
		peers:    make(map[string]*Peer),
		stopChan: make(chan struct{}),
	}
}

func (p2p *P2PNetwork) Start(listener net.Listener) error {
	p2p.mu.Lock()
	p2p.listener = listener
	p2p.running = true
	p2p.mu.Unlock()

	return nil
}

func (p2p *P2PNetwork) Stop() {
	p2p.mu.Lock()
	p2p.running = false
	p2p.mu.Unlock()

	if p2p.listener != nil {
		p2p.listener.Close()
	}

	close(p2p.stopChan)
}

func (p2p *P2PNetwork) AddPeer(peer *Peer) {
	p2p.mu.Lock()
	defer p2p.mu.Unlock()

	p2p.peers[peer.ID] = peer
}

func (p2p *P2PNetwork) RemovePeer(id string) {
	p2p.mu.Lock()
	defer p2p.mu.Unlock()

	delete(p2p.peers, id)
}

func (p2p *P2PNetwork) GetPeer(id string) (*Peer, bool) {
	p2p.mu.RLock()
	defer p2p.mu.RUnlock()

	peer, exists := p2p.peers[id]
	return peer, exists
}

func (p2p *P2PNetwork) GetPeers() []*Peer {
	p2p.mu.RLock()
	defer p2p.mu.RUnlock()

	peers := make([]*Peer, 0, len(p2p.peers))
	for _, peer := range p2p.peers {
		peers = append(peers, peer)
	}

	return peers
}
