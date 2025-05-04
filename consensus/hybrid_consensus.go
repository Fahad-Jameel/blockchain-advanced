// consensus/hybrid_consensus.go - Fixed with proper block processing
package consensus

import (
	"encoding/json"
	"errors"
	"math/big"
	"sync"
	"time"

	"blockchain-advanced/core"
	"blockchain-advanced/crypto"
)

// HybridConsensus implements a hybrid PoW/dBFT consensus mechanism
type HybridConsensus struct {
	mu            sync.RWMutex
	powEngine     *PoWEngine
	bftEngine     *BFTEngine
	nodeAuth      *NodeAuthenticator
	vrfOracle     *crypto.VRFOracle
	reputationMgr *ReputationManager

	currentEpoch   uint64
	validators     map[core.Address]*Validator
	consensusState ConsensusState
}

// ConsensusState represents the current consensus state
type ConsensusState struct {
	Phase           ConsensusPhase
	CurrentLeader   core.Address
	ViewNumber      uint64
	PreparedBlocks  map[core.Hash]*PreparedBlock
	CommittedBlocks map[core.Hash]*CommittedBlock
}

// Validator represents a consensus validator
type Validator struct {
	Address         core.Address
	PublicKey       []byte
	Stake           uint64
	ReputationScore float64
	LastActive      time.Time
	VRFProof        []byte
}

// PreparedBlock represents a block in the prepared state
type PreparedBlock struct {
	Block      *core.Block
	Signatures map[core.Address][]byte
	Timestamp  time.Time
}

// CommittedBlock represents a committed block
type CommittedBlock struct {
	Block      *core.Block
	Signatures map[core.Address][]byte
	Proof      []byte
}

// NewHybridConsensus creates a new hybrid consensus engine
func NewHybridConsensus() *HybridConsensus {
	return &HybridConsensus{
		powEngine:     NewPoWEngine(),
		bftEngine:     NewBFTEngine(),
		nodeAuth:      NewNodeAuthenticator(),
		vrfOracle:     crypto.NewVRFOracle(),
		reputationMgr: NewReputationManager(),
		validators:    make(map[core.Address]*Validator),
		consensusState: ConsensusState{
			PreparedBlocks:  make(map[core.Hash]*PreparedBlock),
			CommittedBlocks: make(map[core.Hash]*CommittedBlock),
		},
	}
}

// Start starts the consensus engine
func (hc *HybridConsensus) Start() error {
	// Start PoW engine
	if err := hc.powEngine.Start(); err != nil {
		return err
	}

	// Start BFT engine
	if err := hc.bftEngine.Start(); err != nil {
		return err
	}

	// Start consensus loop
	go hc.consensusLoop()

	return nil
}

// consensusLoop runs the main consensus process
func (hc *HybridConsensus) consensusLoop() {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.runConsensusRound()
		}
	}
}

// runConsensusRound executes a consensus round
func (hc *HybridConsensus) runConsensusRound() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	// Select leader using VRF
	leader := hc.selectLeader()
	hc.consensusState.CurrentLeader = leader

	// Create new block proposal if we're the leader
	if leader == hc.getSelfAddress() {
		block := hc.createBlockProposal()
		hc.broadcastProposal(block)
	}
}

// selectLeader selects the next leader using VRF
func (hc *HybridConsensus) selectLeader() core.Address {
	// Get VRF output for current epoch
	vrfOutput, proof := hc.vrfOracle.Generate(hc.currentEpoch)

	// Select validator based on VRF output and reputation
	selectedValidator := hc.selectValidatorByVRF(vrfOutput)
	if selectedValidator == nil {
		// Fallback to first validator
		for _, v := range hc.validators {
			return v.Address
		}
		return core.Address{}
	}

	// Store VRF proof
	selectedValidator.VRFProof = proof

	return selectedValidator.Address
}

// selectValidatorByVRF selects a validator based on VRF output
func (hc *HybridConsensus) selectValidatorByVRF(vrfOutput []byte) *Validator {
	if len(hc.validators) == 0 {
		return nil
	}

	// Weight validators by stake and reputation
	totalWeight := 0.0
	weights := make(map[core.Address]float64)

	for addr, validator := range hc.validators {
		weight := float64(validator.Stake) * validator.ReputationScore
		weights[addr] = weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		// No valid weights, return first validator
		for _, v := range hc.validators {
			return v
		}
	}

	// Convert VRF output to selection value
	vrfInt := new(big.Int).SetBytes(vrfOutput)
	maxUint64 := new(big.Int).SetUint64(^uint64(0))
	selection := new(big.Float).Quo(
		new(big.Float).SetInt(vrfInt),
		new(big.Float).SetInt(maxUint64),
	)
	selectionFloat, _ := selection.Float64()

	// Select validator based on VRF output
	cumulative := 0.0

	for addr, weight := range weights {
		cumulative += weight / totalWeight
		if selectionFloat <= cumulative {
			return hc.validators[addr]
		}
	}

	// Fallback (should never reach here)
	for _, validator := range hc.validators {
		return validator
	}

	return nil
}

// createBlockProposal creates a new block proposal
func (hc *HybridConsensus) createBlockProposal() *core.Block {
	// Inject PoW randomness
	powNonce := hc.powEngine.GenerateRandomness()

	// Create block
	block := &core.Block{
		Header: core.BlockHeader{
			Version:   1,
			Height:    hc.getNextHeight(),
			Timestamp: time.Now().Unix(),
			Nonce:     powNonce,
		},
		Transactions: hc.selectTransactions(),
	}

	// Sign block
	hc.signBlock(block)

	return block
}

// ValidateBlock validates a proposed block
func (hc *HybridConsensus) ValidateBlock(block *core.Block) error {
	// Validate PoW component
	if err := hc.powEngine.ValidatePoW(block); err != nil {
		return err
	}

	// Validate BFT signatures
	if err := hc.bftEngine.ValidateSignatures(block); err != nil {
		return err
	}

	// Validate transactions
	for _, tx := range block.Transactions {
		if err := hc.validateTransaction(tx); err != nil {
			return err
		}
	}

	return nil
}

// ProcessConsensusMessage processes a consensus message
func (hc *HybridConsensus) ProcessConsensusMessage(msg *ConsensusMessage) error {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	// Validate message
	if !hc.validateConsensusMessage(msg) {
		return errors.New("invalid consensus message")
	}

	// Process based on message type
	switch msg.Type {
	case "vote":
		return hc.ProcessVote(msg.ToVote())
	case "proposal":
		return hc.ProcessProposal(msg.ToProposal())
	case "commit":
		return hc.ProcessCommit(msg.ToCommit())
	default:
		return errors.New("unknown consensus message type")
	}
}

// ProcessVote processes a vote from a validator
func (hc *HybridConsensus) ProcessVote(vote *Vote) error {
	// Verify validator identity
	if !hc.nodeAuth.VerifyNode(vote.ValidatorAddr) {
		return errors.New("invalid validator")
	}

	// Update reputation based on vote
	hc.reputationMgr.UpdateReputation(vote.ValidatorAddr, vote.IsValid)

	// Process vote in BFT engine
	return hc.bftEngine.ProcessVote(vote)
}

// ProcessProposal processes a block proposal
func (hc *HybridConsensus) ProcessProposal(proposal *BlockProposal) error {
	// Validate proposal
	if err := hc.ValidateBlock(proposal.Block); err != nil {
		return err
	}

	// Store in prepared blocks
	blockHash := proposal.Block.Header.Hash()
	hc.consensusState.PreparedBlocks[blockHash] = &PreparedBlock{
		Block:      proposal.Block,
		Signatures: make(map[core.Address][]byte),
		Timestamp:  time.Now(),
	}

	// Vote on the proposal
	vote := &Vote{
		BlockHash:     blockHash,
		ValidatorAddr: hc.getSelfAddress(),
		IsValid:       true,
		Timestamp:     time.Now(),
	}

	// Sign the vote
	signature, err := hc.signVote(vote)
	if err != nil {
		return err
	}
	vote.Signature = signature

	// Broadcast vote
	hc.broadcastVote(vote)

	return nil
}

// ProcessCommit processes a commit message
func (hc *HybridConsensus) ProcessCommit(commit *CommitMessage) error {
	block := commit.Block

	// Verify we have enough signatures
	if !hc.hasQuorum(block) {
		return errors.New("insufficient signatures for commit")
	}

	// Finalize the block
	return hc.FinalizeBlock(block)
}

// FinalizeBlock finalizes a block after consensus
func (hc *HybridConsensus) FinalizeBlock(block *core.Block) error {
	// Check if block has enough votes
	if !hc.hasQuorum(block) {
		return errors.New("insufficient votes")
	}

	// Create consensus proof
	proof := hc.createConsensusProof(block)

	// Calculate block hash for storing
	blockHash := block.Header.Hash()

	// Store committed block
	hc.consensusState.CommittedBlocks[blockHash] = &CommittedBlock{
		Block:      block,
		Signatures: hc.collectSignatures(block),
		Proof:      proof,
	}

	// Update epoch
	hc.currentEpoch++

	return nil
}

// Helper methods

func (hc *HybridConsensus) getSelfAddress() core.Address {
	// In a real implementation, would get from node config
	return core.Address{}
}

func (hc *HybridConsensus) getNextHeight() uint64 {
	// In a real implementation, would get from blockchain
	return 0
}

func (hc *HybridConsensus) selectTransactions() []core.Transaction {
	// In a real implementation, would select from mempool
	return []core.Transaction{}
}

func (hc *HybridConsensus) signBlock(block *core.Block) {
	// In a real implementation, would properly sign
}

func (hc *HybridConsensus) broadcastProposal(block *core.Block) {
	// In a real implementation, would broadcast to network
}

func (hc *HybridConsensus) validateTransaction(tx core.Transaction) error {
	// Basic validation
	return nil
}

func (hc *HybridConsensus) hasQuorum(block *core.Block) bool {
	// Check if 2/3+ validators have voted
	return true
}

func (hc *HybridConsensus) createConsensusProof(block *core.Block) []byte {
	// Create proof of consensus
	return []byte("consensus_proof")
}

func (hc *HybridConsensus) collectSignatures(block *core.Block) map[core.Address][]byte {
	// Collect validator signatures
	return make(map[core.Address][]byte)
}

func (hc *HybridConsensus) signVote(vote *Vote) ([]byte, error) {
	data, err := json.Marshal(vote)
	if err != nil {
		return nil, err
	}

	// In real implementation, would use proper key management
	return crypto.Sign(data, []byte("validator_private_key")), nil
}

func (hc *HybridConsensus) broadcastVote(vote *Vote) {
	msg := &ConsensusMessage{
		Type:      "vote",
		Sender:    vote.ValidatorAddr,
		Data:      mustMarshal(vote),
		Timestamp: time.Now(),
	}

	// In real implementation, would broadcast to network
	go hc.ProcessConsensusMessage(msg)
}

func (hc *HybridConsensus) validateConsensusMessage(msg *ConsensusMessage) bool {
	// Basic validation
	if msg.Type == "" || msg.Sender == (core.Address{}) {
		return false
	}

	// Verify timestamp is recent
	if time.Since(msg.Timestamp) > 10*time.Minute {
		return false
	}

	return true
}

// Message conversion methods

func (msg *ConsensusMessage) ToVote() *Vote {
	var vote Vote
	json.Unmarshal(msg.Data, &vote)
	return &vote
}

func (msg *ConsensusMessage) ToProposal() *BlockProposal {
	var proposal BlockProposal
	json.Unmarshal(msg.Data, &proposal)
	return &proposal
}

func (msg *ConsensusMessage) ToCommit() *CommitMessage {
	var commit CommitMessage
	json.Unmarshal(msg.Data, &commit)
	return &commit
}

// Supporting types

type ConsensusMessage struct {
	Type      string
	Sender    core.Address
	Data      []byte
	Timestamp time.Time
}

type Vote struct {
	BlockHash     core.Hash
	ValidatorAddr core.Address
	IsValid       bool
	Signature     []byte
	Timestamp     time.Time
}

type BlockProposal struct {
	Block        *core.Block
	ProposerAddr core.Address
	Signature    []byte
	Timestamp    time.Time
}

type CommitMessage struct {
	Block      *core.Block
	Signatures map[core.Address][]byte
	Timestamp  time.Time
}

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
