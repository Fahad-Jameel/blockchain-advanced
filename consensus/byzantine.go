package consensus

import (
	"errors"
	"math"
	"sync"
	"time"

	"blockchain-advanced/core"
)

// ByzantineDefender implements Byzantine fault tolerance mechanisms
type ByzantineDefender struct {
	mu               sync.RWMutex
	reputationSystem *ReputationSystem
	attackDetector   *AttackDetector
	defenseStrategy  *DefenseStrategy

	nodeScores      map[core.Address]float64
	blacklist       map[core.Address]time.Time
	suspiciousNodes map[core.Address]int
}

// ReputationSystem manages node reputation scores
type ReputationSystem struct {
	scores        map[core.Address]*NodeReputation
	historyWindow time.Duration
	decayFactor   float64
}

// NodeReputation represents a node's reputation
type NodeReputation struct {
	Address        core.Address
	Score          float64
	HistoricalData []ReputationEvent
	LastUpdated    time.Time
	ViolationCount int
}

// ReputationEvent represents an event affecting reputation
type ReputationEvent struct {
	EventType   string
	Impact      float64
	Timestamp   time.Time
	Description string
}

// AttackDetector identifies potential Byzantine attacks
type AttackDetector struct {
	patterns        map[string]*AttackPattern
	detectionRules  []DetectionRule
	alertThresholds map[string]float64
}

// AttackPattern represents a known attack pattern
type AttackPattern struct {
	Name       string
	Indicators []string
	Severity   float64
	Frequency  int
	LastSeen   time.Time
}

// NewByzantineDefender creates a new Byzantine defender
func NewByzantineDefender() *ByzantineDefender {
	return &ByzantineDefender{
		reputationSystem: NewReputationSystem(),
		attackDetector:   NewAttackDetector(),
		defenseStrategy:  NewDefenseStrategy(),
		nodeScores:       make(map[core.Address]float64),
		blacklist:        make(map[core.Address]time.Time),
		suspiciousNodes:  make(map[core.Address]int),
	}
}

// ValidateNode validates a node's behavior and updates reputation
func (bd *ByzantineDefender) ValidateNode(nodeAddr core.Address, behavior *NodeBehavior) error {
	bd.mu.Lock()
	defer bd.mu.Unlock()

	// Check if node is blacklisted
	if blacklistTime, exists := bd.blacklist[nodeAddr]; exists {
		if time.Since(blacklistTime) < 24*time.Hour {
			return errors.New("node is blacklisted")
		}
		delete(bd.blacklist, nodeAddr)
	}

	// Analyze behavior for attacks
	attacks := bd.attackDetector.DetectAttacks(behavior)

	// Update reputation based on behavior
	reputationDelta := bd.calculateReputationDelta(behavior, attacks)
	bd.updateNodeReputation(nodeAddr, reputationDelta)

	// Apply defensive measures if needed
	if len(attacks) > 0 {
		bd.applyDefensiveMeasures(nodeAddr, attacks)
	}

	return nil
}

// calculateReputationDelta calculates reputation change based on behavior
func (bd *ByzantineDefender) calculateReputationDelta(behavior *NodeBehavior, attacks []Attack) float64 {
	delta := 0.0

	// Positive contributions
	if behavior.ValidBlocks > 0 {
		delta += float64(behavior.ValidBlocks) * 0.1
	}
	if behavior.ValidTransactions > 0 {
		delta += float64(behavior.ValidTransactions) * 0.01
	}
	if behavior.ConsensusParticipation > 0 {
		delta += behavior.ConsensusParticipation * 0.2
	}

	// Negative impacts from attacks
	for _, attack := range attacks {
		delta -= attack.Severity * 0.5
	}

	// Penalties for suspicious behavior
	if behavior.InvalidBlocks > 0 {
		delta -= float64(behavior.InvalidBlocks) * 0.3
	}
	if behavior.DoubleVotes > 0 {
		delta -= float64(behavior.DoubleVotes) * 1.0
	}

	return delta
}

// updateNodeReputation updates a node's reputation score
func (bd *ByzantineDefender) updateNodeReputation(nodeAddr core.Address, delta float64) {
	reputation := bd.reputationSystem.GetReputation(nodeAddr)

	// Apply delta with bounds
	newScore := reputation.Score + delta
	newScore = math.Max(0.0, math.Min(100.0, newScore))

	// Update reputation
	reputation.Score = newScore
	reputation.LastUpdated = time.Now()

	// Add event to history
	event := ReputationEvent{
		EventType:   "score_update",
		Impact:      delta,
		Timestamp:   time.Now(),
		Description: "Reputation score updated",
	}
	reputation.HistoricalData = append(reputation.HistoricalData, event)

	// Update node scores
	bd.nodeScores[nodeAddr] = newScore
}

// applyDefensiveMeasures applies defensive measures against detected attacks
func (bd *ByzantineDefender) applyDefensiveMeasures(nodeAddr core.Address, attacks []Attack) {
	for _, attack := range attacks {
		switch attack.Type {
		case "double_spend":
			bd.handleDoubleSpend(nodeAddr)
		case "sybil_attack":
			bd.handleSybilAttack(nodeAddr)
		case "eclipse_attack":
			bd.handleEclipseAttack(nodeAddr)
		case "long_range_attack":
			bd.handleLongRangeAttack(nodeAddr)
		default:
			bd.handleGenericAttack(nodeAddr, attack)
		}
	}
}

// handleDoubleSpend handles double spend attempts
func (bd *ByzantineDefender) handleDoubleSpend(nodeAddr core.Address) {
	// Increase suspicion level
	bd.suspiciousNodes[nodeAddr]++

	// Blacklist if threshold exceeded
	if bd.suspiciousNodes[nodeAddr] > 3 {
		bd.blacklist[nodeAddr] = time.Now()
	}

	// Notify network
	bd.broadcastAlert("double_spend", nodeAddr)
}

// GetConsensusThreshold returns adaptive consensus threshold
func (bd *ByzantineDefender) GetConsensusThreshold() float64 {
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	// Calculate network health
	healthScore := bd.calculateNetworkHealth()

	// Adjust threshold based on health
	baseThreshold := 0.67 // 2/3 for BFT

	if healthScore < 0.5 {
		// Increase threshold if network health is poor
		return math.Min(0.9, baseThreshold+0.1)
	} else if healthScore > 0.8 {
		// Can decrease threshold if network is very healthy
		return math.Max(0.6, baseThreshold-0.05)
	}

	return baseThreshold
}

// calculateNetworkHealth calculates overall network health score
func (bd *ByzantineDefender) calculateNetworkHealth() float64 {
	totalScore := 0.0
	nodeCount := 0

	for _, score := range bd.nodeScores {
		totalScore += score
		nodeCount++
	}

	if nodeCount == 0 {
		return 1.0 // Assume healthy if no data
	}

	return totalScore / (float64(nodeCount) * 100.0)
}

// Additional defensive handlers

func (bd *ByzantineDefender) handleSybilAttack(nodeAddr core.Address) {
	// Check for similar nodes
	similarNodes := bd.findSimilarNodes(nodeAddr)
	if len(similarNodes) > 5 {
		// Potential Sybil attack
		for _, addr := range similarNodes {
			bd.blacklist[addr] = time.Now()
		}
	}
}

func (bd *ByzantineDefender) handleEclipseAttack(nodeAddr core.Address) {
	// Limit connections from single source
	bd.broadcastAlert("eclipse_attack", nodeAddr)
}

func (bd *ByzantineDefender) handleLongRangeAttack(nodeAddr core.Address) {
	// Implement checkpointing
	bd.broadcastAlert("long_range_attack", nodeAddr)
}

func (bd *ByzantineDefender) handleGenericAttack(nodeAddr core.Address, attack Attack) {
	// Generic attack handling
	bd.suspiciousNodes[nodeAddr]++
	if attack.Severity > 0.8 {
		bd.blacklist[nodeAddr] = time.Now()
	}
}

func (bd *ByzantineDefender) broadcastAlert(attackType string, nodeAddr core.Address) {
	// In real implementation, would broadcast to network
}

func (bd *ByzantineDefender) findSimilarNodes(nodeAddr core.Address) []core.Address {
	// In real implementation, would check for similar behavior patterns
	return []core.Address{}
}

// Supporting type implementations

func NewReputationSystem() *ReputationSystem {
	return &ReputationSystem{
		scores:        make(map[core.Address]*NodeReputation),
		historyWindow: 24 * time.Hour,
		decayFactor:   0.95,
	}
}

func (rs *ReputationSystem) GetReputation(addr core.Address) *NodeReputation {
	rep, exists := rs.scores[addr]
	if !exists {
		rep = &NodeReputation{
			Address:        addr,
			Score:          50.0, // Start with neutral score
			HistoricalData: make([]ReputationEvent, 0),
			LastUpdated:    time.Now(),
		}
		rs.scores[addr] = rep
	}
	return rep
}

func NewAttackDetector() *AttackDetector {
	return &AttackDetector{
		patterns:        make(map[string]*AttackPattern),
		detectionRules:  make([]DetectionRule, 0),
		alertThresholds: make(map[string]float64),
	}
}

func (ad *AttackDetector) DetectAttacks(behavior *NodeBehavior) []Attack {
	attacks := make([]Attack, 0)

	// Check for double voting
	if behavior.DoubleVotes > 0 {
		attacks = append(attacks, Attack{
			Type:        "double_vote",
			Severity:    float64(behavior.DoubleVotes),
			Timestamp:   time.Now(),
			Description: "Multiple votes detected",
		})
	}

	// Check for invalid blocks
	if behavior.InvalidBlocks > behavior.ValidBlocks {
		attacks = append(attacks, Attack{
			Type:        "invalid_blocks",
			Severity:    float64(behavior.InvalidBlocks) / float64(behavior.ValidBlocks+1),
			Timestamp:   time.Now(),
			Description: "High ratio of invalid blocks",
		})
	}

	return attacks
}

func NewReputationManager() *ReputationManager {
	return &ReputationManager{
		reputations: make(map[core.Address]float64),
	}
}

type ReputationManager struct {
	reputations map[core.Address]float64
	mu          sync.RWMutex
}

func (rm *ReputationManager) UpdateReputation(addr core.Address, isValid bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	current, exists := rm.reputations[addr]
	if !exists {
		current = 50.0 // Start neutral
	}

	if isValid {
		current = math.Min(100.0, current+1.0)
	} else {
		current = math.Max(0.0, current-5.0)
	}

	rm.reputations[addr] = current
}

func NewDefenseStrategy() *DefenseStrategy {
	return &DefenseStrategy{}
}

type DefenseStrategy struct{}

// Supporting types

type NodeBehavior struct {
	ValidBlocks            int
	InvalidBlocks          int
	ValidTransactions      int
	InvalidTransactions    int
	ConsensusParticipation float64
	DoubleVotes            int
	NetworkLatency         time.Duration
}

type Attack struct {
	Type        string
	Severity    float64
	Timestamp   time.Time
	Description string
}

type DetectionRule struct {
	Name      string
	Condition func(*NodeBehavior) bool
	Severity  float64
}
