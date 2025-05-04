package consensus

import (
	"blockchain-advanced/core"
	"testing"
	"time"
)

func TestNewHybridConsensus(t *testing.T) {
	hc := NewHybridConsensus()

	if hc == nil {
		t.Fatal("NewHybridConsensus returned nil")
	}

	if hc.powEngine == nil {
		t.Error("PoW engine not initialized")
	}

	if hc.bftEngine == nil {
		t.Error("BFT engine not initialized")
	}
}

func TestValidateBlock(t *testing.T) {
	hc := NewHybridConsensus()

	block := &core.Block{
		Header: core.BlockHeader{
			Version:   1,
			Height:    1,
			Timestamp: time.Now().Unix(),
		},
	}

	err := hc.ValidateBlock(block)
	if err != nil {
		t.Errorf("Block validation failed: %v", err)
	}
}

func TestVRFLeaderSelection(t *testing.T) {
	hc := NewHybridConsensus()

	// Add test validators
	validator1 := &Validator{
		Address:         core.Address{1},
		Stake:           1000,
		ReputationScore: 0.8,
	}

	validator2 := &Validator{
		Address:         core.Address{2},
		Stake:           2000,
		ReputationScore: 0.9,
	}

	hc.validators[validator1.Address] = validator1
	hc.validators[validator2.Address] = validator2

	// Select leader
	leader := hc.selectLeader()

	// Check leader is one of the validators
	if leader != validator1.Address && leader != validator2.Address {
		t.Error("Selected leader is not a validator")
	}
}

func TestProcessVote(t *testing.T) {
	hc := NewHybridConsensus()

	vote := &Vote{
		BlockHash:     core.Hash{1, 2, 3},
		ValidatorAddr: core.Address{4, 5, 6},
		IsValid:       true,
		Timestamp:     time.Now(),
	}

	err := hc.ProcessVote(vote)
	if err != nil {
		t.Errorf("Failed to process vote: %v", err)
	}
}

func TestByzantineDefender(t *testing.T) {
	bd := NewByzantineDefender()

	nodeAddr := core.Address{1, 2, 3}
	behavior := &NodeBehavior{
		ValidBlocks:            10,
		InvalidBlocks:          1,
		ValidTransactions:      100,
		ConsensusParticipation: 0.9,
	}

	err := bd.ValidateNode(nodeAddr, behavior)
	if err != nil {
		t.Errorf("Node validation failed: %v", err)
	}

	// Check reputation score
	if score, exists := bd.nodeScores[nodeAddr]; !exists || score <= 0 {
		t.Error("Reputation score not updated")
	}
}

func TestAttackDetection(t *testing.T) {
	ad := NewAttackDetector()

	// Test double vote detection
	behavior := &NodeBehavior{
		DoubleVotes: 2,
	}

	attacks := ad.DetectAttacks(behavior)
	if len(attacks) == 0 {
		t.Error("Failed to detect double vote attack")
	}

	// Test invalid blocks detection
	behavior2 := &NodeBehavior{
		ValidBlocks:   5,
		InvalidBlocks: 10,
	}

	attacks2 := ad.DetectAttacks(behavior2)
	if len(attacks2) == 0 {
		t.Error("Failed to detect invalid blocks attack")
	}
}
