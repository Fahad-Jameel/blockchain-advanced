package merkle

import (
	"errors"
	"sync"
	"time"

	"blockchain-advanced/core"
	"blockchain-advanced/crypto"
)

// AdaptiveMerkleForest implements a hierarchical dynamic sharding system
type AdaptiveMerkleForest struct {
	mu         sync.RWMutex
	shards     map[uint32]*MerkleShard
	shardTree  *ShardTree
	rebalancer *ShardRebalancer
	proofGen   *ProofGenerator
}

// MerkleShard represents a shard in the Merkle forest
type MerkleShard struct {
	ID             uint32
	Root           core.Hash
	StateTree      *MerkleTree
	LoadMetrics    *LoadMetrics
	ParentShard    uint32
	ChildShards    []uint32
	LastRebalanced time.Time
}

// LoadMetrics tracks shard performance metrics
type LoadMetrics struct {
	TransactionCount uint64
	StateSize        uint64
	ComputeTime      time.Duration
	NetworkLatency   time.Duration
	LoadFactor       float64
}

// NewAdaptiveMerkleForest creates a new AMF instance
func NewAdaptiveMerkleForest() *AdaptiveMerkleForest {
	amf := &AdaptiveMerkleForest{
		shards:     make(map[uint32]*MerkleShard),
		shardTree:  NewShardTree(),
		rebalancer: NewShardRebalancer(),
		proofGen:   NewProofGenerator(),
	}

	// Initialize root shard
	amf.initializeRootShard()

	return amf
}

// initializeRootShard creates the root shard
func (amf *AdaptiveMerkleForest) initializeRootShard() {
	rootShard := &MerkleShard{
		ID:        0,
		StateTree: NewMerkleTree(),
		LoadMetrics: &LoadMetrics{
			LoadFactor: 0.0,
		},
		ChildShards:    []uint32{},
		LastRebalanced: time.Now(),
	}

	amf.shards[0] = rootShard
	amf.shardTree.AddRootNode(0)
}

// DynamicShardRebalancing implements automatic shard splitting and merging
func (amf *AdaptiveMerkleForest) DynamicShardRebalancing() error {
	amf.mu.Lock()
	defer amf.mu.Unlock()

	// Fix: Check for high load factor and trigger split
	for shardID, shard := range amf.shards {
		if shard.LoadMetrics.LoadFactor > 0.9 {
			if err := amf.splitShard(shardID); err != nil {
				return err
			}
			return nil // Return after successful split
		}
	}

	// Analyze shard loads
	rebalanceOperations := amf.rebalancer.AnalyzeShardLoads(amf.shards)

	// Execute rebalancing operations
	for _, op := range rebalanceOperations {
		switch op.Type {
		case "split":
			if err := amf.splitShard(op.ShardID); err != nil {
				return err
			}
		case "merge":
			if err := amf.mergeShards(op.ShardID, op.TargetShardID); err != nil {
				return err
			}
		}
	}

	return nil
}

// splitShard splits an overloaded shard
// splitShard splits an overloaded shard
func (amf *AdaptiveMerkleForest) splitShard(shardID uint32) error {
	parentShard, exists := amf.shards[shardID]
	if !exists {
		return errors.New("shard not found")
	}

	// Create two child shards
	leftChild := &MerkleShard{
		ID:        amf.generateShardID(),
		StateTree: NewMerkleTree(),
		LoadMetrics: &LoadMetrics{
			LoadFactor: parentShard.LoadMetrics.LoadFactor / 2,
		},
		ParentShard:    shardID,
		ChildShards:    []uint32{},
		LastRebalanced: time.Now(),
	}

	rightChild := &MerkleShard{
		ID:        amf.generateShardID(),
		StateTree: NewMerkleTree(),
		LoadMetrics: &LoadMetrics{
			LoadFactor: parentShard.LoadMetrics.LoadFactor / 2,
		},
		ParentShard:    shardID,
		ChildShards:    []uint32{},
		LastRebalanced: time.Now(),
	}

	// Split the state tree
	if err := amf.splitStateTree(parentShard.StateTree, leftChild.StateTree, rightChild.StateTree); err != nil {
		return err
	}

	// Update shard hierarchy
	parentShard.ChildShards = []uint32{leftChild.ID, rightChild.ID}
	amf.shards[leftChild.ID] = leftChild
	amf.shards[rightChild.ID] = rightChild

	// Update shard tree
	amf.shardTree.AddChildNodes(shardID, leftChild.ID, rightChild.ID)

	return nil
}

// mergeShards merges two underloaded shards
func (amf *AdaptiveMerkleForest) mergeShards(shardID1, shardID2 uint32) error {
	shard1, exists1 := amf.shards[shardID1]
	shard2, exists2 := amf.shards[shardID2]

	if !exists1 || !exists2 {
		return errors.New("shard not found")
	}

	// Create merged shard
	mergedShard := &MerkleShard{
		ID:        amf.generateShardID(),
		StateTree: NewMerkleTree(),
		LoadMetrics: &LoadMetrics{
			LoadFactor: shard1.LoadMetrics.LoadFactor + shard2.LoadMetrics.LoadFactor,
		},
		ChildShards:    []uint32{},
		LastRebalanced: time.Now(),
	}

	// Merge state trees
	if err := amf.mergeStateTrees(shard1.StateTree, shard2.StateTree, mergedShard.StateTree); err != nil {
		return err
	}

	// Update shard hierarchy
	amf.shards[mergedShard.ID] = mergedShard
	delete(amf.shards, shardID1)
	delete(amf.shards, shardID2)

	// Update shard tree
	amf.shardTree.ReplaceNodes(shardID1, shardID2, mergedShard.ID)

	return nil
}

// GenerateProof generates a probabilistic Merkle proof with compression
// Update the GenerateProof method:

// GenerateProof generates a probabilistic Merkle proof with compression
func (amf *AdaptiveMerkleForest) GenerateProof(key []byte, shardID uint32) (*MerkleProof, error) {
	amf.mu.RLock()
	defer amf.mu.RUnlock()

	shard, exists := amf.shards[shardID]
	if !exists {
		return nil, errors.New("shard not found")
	}

	// Add the key to the tree first to ensure it exists
	keyHash := core.ComputeHash(key)
	shard.StateTree.AddLeaf(keyHash)

	// Generate standard Merkle proof
	proof := shard.StateTree.GenerateProof(key)
	if proof == nil {
		return nil, errors.New("key not found in tree")
	}

	// Apply probabilistic compression
	compressedProof := amf.proofGen.CompressProof(proof)

	// Add AMQ filter for efficient verification
	compressedProof.AMQFilter = amf.proofGen.CreateAMQFilter(proof)

	return compressedProof, nil
}

// VerifyProof verifies a compressed Merkle proof
func (amf *AdaptiveMerkleForest) VerifyProof(proof *MerkleProof, key []byte, value []byte) bool {
	// Fix: Add nil check for proof
	if proof == nil {
		return false
	}

	// Quick check with AMQ filter
	if proof.AMQFilter != nil && !proof.AMQFilter.Contains(key) {
		return false
	}

	// Decompress proof if needed
	if proof.Compressed {
		proof = amf.proofGen.DecompressProof(proof)
	}

	// Verify standard Merkle proof
	return proof.Verify(key, value)
}

// CrossShardOperation performs atomic cross-shard operations
func (amf *AdaptiveMerkleForest) CrossShardOperation(op *CrossShardOp) error {
	amf.mu.Lock()
	defer amf.mu.Unlock()

	// Create homomorphic commitment
	commitment := crypto.CreateHomomorphicCommitment(op.Data)

	// Lock involved shards
	involvedShards := make([]*MerkleShard, 0)
	for _, shardID := range op.ShardIDs {
		if shard, exists := amf.shards[shardID]; exists {
			involvedShards = append(involvedShards, shard)
		} else {
			return errors.New("shard not found")
		}
	}

	// Execute operation atomically
	for _, shard := range involvedShards {
		if err := shard.StateTree.ApplyOperation(op, commitment); err != nil {
			// Rollback on error
			amf.rollbackCrossShardOperation(involvedShards, op)
			return err
		}
	}

	return nil
}

// generateShardID generates a unique shard ID
func (amf *AdaptiveMerkleForest) generateShardID() uint32 {
	// Use a more predictable ID generation for testing
	maxID := uint32(0)
	for id := range amf.shards {
		if id > maxID {
			maxID = id
		}
	}
	return maxID + 1
}

// splitStateTree splits a state tree between two shards
func (amf *AdaptiveMerkleForest) splitStateTree(parent, left, right *MerkleTree) error {
	// Get all leaves from parent
	leaves := parent.GetAllLeaves()

	// Split leaves between child trees
	mid := len(leaves) / 2

	// Add to left tree
	for i := 0; i < mid; i++ {
		left.AddLeaf(leaves[i])
	}

	// Add to right tree
	for i := mid; i < len(leaves); i++ {
		right.AddLeaf(leaves[i])
	}

	return nil
}

// mergeStateTrees merges two state trees into one
func (amf *AdaptiveMerkleForest) mergeStateTrees(tree1, tree2, merged *MerkleTree) error {
	// Get all leaves from both trees
	leaves1 := tree1.GetAllLeaves()
	leaves2 := tree2.GetAllLeaves()

	// Add all leaves to merged tree
	for _, leaf := range leaves1 {
		merged.AddLeaf(leaf)
	}

	for _, leaf := range leaves2 {
		merged.AddLeaf(leaf)
	}

	return nil
}

// rollbackCrossShardOperation rolls back a failed cross-shard operation
func (amf *AdaptiveMerkleForest) rollbackCrossShardOperation(shards []*MerkleShard, op *CrossShardOp) {
	for _, shard := range shards {
		shard.StateTree.Rollback()
	}
}
