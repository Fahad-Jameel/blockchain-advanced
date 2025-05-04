package merkle

import (
	"sync"
	"time"

	"blockchain-advanced/core"
)

// ShardTree manages the hierarchical structure of shards
type ShardTree struct {
	Root    *ShardNode
	NodeMap map[uint32]*ShardNode
	mu      sync.RWMutex
}

// ShardNode represents a node in the shard hierarchy
type ShardNode struct {
	ShardID    uint32
	Parent     *ShardNode
	Children   []*ShardNode
	Depth      int
	LoadFactor float64
}

// ShardRebalancer handles shard rebalancing operations
type ShardRebalancer struct {
	mu                sync.RWMutex
	loadThreshold     float64
	splitThreshold    float64
	mergeThreshold    float64
	rebalanceInterval time.Duration
}

// RebalanceOperation represents a rebalancing operation
type RebalanceOperation struct {
	Type          string // "split" or "merge"
	ShardID       uint32
	TargetShardID uint32 // For merge operations
	Timestamp     time.Time
}

// NewShardTree creates a new shard tree
func NewShardTree() *ShardTree {
	return &ShardTree{
		NodeMap: make(map[uint32]*ShardNode),
	}
}

// AddRootNode adds the root node to the tree
func (st *ShardTree) AddRootNode(shardID uint32) {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.Root = &ShardNode{
		ShardID:  shardID,
		Depth:    0,
		Children: []*ShardNode{},
	}
	st.NodeMap[shardID] = st.Root
}

// AddChildNodes adds child nodes to a parent
func (st *ShardTree) AddChildNodes(parentID, leftID, rightID uint32) {
	st.mu.Lock()
	defer st.mu.Unlock()

	parent := st.NodeMap[parentID]
	if parent == nil {
		return
	}

	leftChild := &ShardNode{
		ShardID:  leftID,
		Parent:   parent,
		Depth:    parent.Depth + 1,
		Children: []*ShardNode{},
	}

	rightChild := &ShardNode{
		ShardID:  rightID,
		Parent:   parent,
		Depth:    parent.Depth + 1,
		Children: []*ShardNode{},
	}

	parent.Children = []*ShardNode{leftChild, rightChild}
	st.NodeMap[leftID] = leftChild
	st.NodeMap[rightID] = rightChild
}

// ReplaceNodes replaces two nodes with a merged node
func (st *ShardTree) ReplaceNodes(id1, id2, mergedID uint32) {
	st.mu.Lock()
	defer st.mu.Unlock()

	node1 := st.NodeMap[id1]
	node2 := st.NodeMap[id2]

	if node1 == nil || node2 == nil || node1.Parent != node2.Parent {
		return
	}

	parent := node1.Parent
	mergedNode := &ShardNode{
		ShardID:  mergedID,
		Parent:   parent,
		Depth:    node1.Depth,
		Children: []*ShardNode{},
	}

	// Update parent's children
	newChildren := []*ShardNode{}
	for _, child := range parent.Children {
		if child.ShardID != id1 && child.ShardID != id2 {
			newChildren = append(newChildren, child)
		}
	}
	newChildren = append(newChildren, mergedNode)
	parent.Children = newChildren

	// Remove old nodes
	delete(st.NodeMap, id1)
	delete(st.NodeMap, id2)

	// Add merged node
	st.NodeMap[mergedID] = mergedNode
}

// NewShardRebalancer creates a new shard rebalancer
func NewShardRebalancer() *ShardRebalancer {
	return &ShardRebalancer{
		loadThreshold:     0.8,
		splitThreshold:    0.9,
		mergeThreshold:    0.2,
		rebalanceInterval: 5 * time.Minute,
	}
}

// AnalyzeShardLoads analyzes shard loads and determines rebalancing operations
func (sr *ShardRebalancer) AnalyzeShardLoads(shards map[uint32]*MerkleShard) []RebalanceOperation {
	operations := make([]RebalanceOperation, 0)
	now := time.Now()

	for shardID, shard := range shards {
		// Don't rebalance too frequently
		if now.Sub(shard.LastRebalanced) < sr.rebalanceInterval {
			continue
		}

		loadFactor := shard.LoadMetrics.LoadFactor

		if loadFactor > sr.splitThreshold {
			// Shard needs to be split
			operations = append(operations, RebalanceOperation{
				Type:      "split",
				ShardID:   shardID,
				Timestamp: now,
			})
		} else if loadFactor < sr.mergeThreshold {
			// Look for another shard to merge with
			targetShard := sr.findMergeCandidate(shards, shardID)
			if targetShard != 0 {
				operations = append(operations, RebalanceOperation{
					Type:          "merge",
					ShardID:       shardID,
					TargetShardID: targetShard,
					Timestamp:     now,
				})
			}
		}
	}

	return operations
}

// findMergeCandidate finds a suitable shard to merge with
func (sr *ShardRebalancer) findMergeCandidate(shards map[uint32]*MerkleShard, shardID uint32) uint32 {
	shard := shards[shardID]

	// Find sibling shards
	for id, candidate := range shards {
		if id == shardID {
			continue
		}

		// Check if they have the same parent
		if candidate.ParentShard == shard.ParentShard {
			// Check if combined load is acceptable
			combinedLoad := shard.LoadMetrics.LoadFactor + candidate.LoadMetrics.LoadFactor
			if combinedLoad < sr.loadThreshold {
				return id
			}
		}
	}

	return 0
}

// MerkleTree represents a Merkle tree
type MerkleTree struct {
	Root   core.Hash
	Leaves []core.Hash
	Nodes  map[core.Hash]*MerkleNode
	mu     sync.RWMutex
}

// MerkleNode represents a node in the Merkle tree
type MerkleNode struct {
	Hash   core.Hash
	Left   *MerkleNode
	Right  *MerkleNode
	Parent *MerkleNode
	IsLeaf bool
	Data   []byte
}

// NewMerkleTree creates a new Merkle tree
func NewMerkleTree() *MerkleTree {
	return &MerkleTree{
		Nodes:  make(map[core.Hash]*MerkleNode),
		Leaves: []core.Hash{},
	}
}

// AddLeaf adds a leaf to the tree
func (mt *MerkleTree) AddLeaf(leaf core.Hash) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	mt.Leaves = append(mt.Leaves, leaf)

	leafNode := &MerkleNode{
		Hash:   leaf,
		IsLeaf: true,
	}
	mt.Nodes[leaf] = leafNode

	// Rebuild tree
	mt.buildTree()
}

// GetAllLeaves returns all leaves in the tree
func (mt *MerkleTree) GetAllLeaves() []core.Hash {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	return append([]core.Hash{}, mt.Leaves...)
}

// buildTree builds the Merkle tree from leaves
func (mt *MerkleTree) buildTree() {
	if len(mt.Leaves) == 0 {
		return
	}

	// Create leaf nodes
	currentLevel := make([]*MerkleNode, len(mt.Leaves))
	for i, leaf := range mt.Leaves {
		currentLevel[i] = mt.Nodes[leaf]
	}

	// Build tree bottom-up
	for len(currentLevel) > 1 {
		nextLevel := make([]*MerkleNode, 0)

		for i := 0; i < len(currentLevel); i += 2 {
			left := currentLevel[i]

			var right *MerkleNode
			if i+1 < len(currentLevel) {
				right = currentLevel[i+1]
			} else {
				// Duplicate last node if odd number
				right = left
			}

			// Create parent node
			combinedData := append(left.Hash[:], right.Hash[:]...)
			parentHash := core.ComputeHash(combinedData)

			parent := &MerkleNode{
				Hash:  parentHash,
				Left:  left,
				Right: right,
			}

			left.Parent = parent
			right.Parent = parent

			mt.Nodes[parentHash] = parent
			nextLevel = append(nextLevel, parent)
		}

		currentLevel = nextLevel
	}

	// Set root
	if len(currentLevel) > 0 {
		mt.Root = currentLevel[0].Hash
	}
}

// GenerateProof generates a Merkle proof for a key
// Add to MerkleTree struct methods:

// GenerateProof generates a Merkle proof for a key
func (mt *MerkleTree) GenerateProof(key []byte) *MerkleProof {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	// Hash the key to find the leaf
	leafHash := core.ComputeHash(key)

	// Check if leaf exists in tree
	var leafNode *MerkleNode
	for _, leaf := range mt.Leaves {
		if leaf == leafHash {
			leafNode = mt.Nodes[leaf]
			break
		}
	}

	if leafNode == nil {
		return nil
	}

	// Build proof path
	path := make([]ProofNode, 0)
	current := leafNode

	for current.Parent != nil {
		var sibling *MerkleNode
		var position Position

		if current.Parent.Left == current {
			sibling = current.Parent.Right
			position = Right
		} else {
			sibling = current.Parent.Left
			position = Left
		}

		if sibling != nil {
			path = append(path, ProofNode{
				Hash:     sibling.Hash,
				Position: position,
				IsLeaf:   sibling.IsLeaf,
			})
		}

		current = current.Parent
	}

	return &MerkleProof{
		Path:      path,
		Root:      mt.Root,
		ProofType: StandardProof,
	}
}

// ApplyOperation applies an operation to the state tree
func (mt *MerkleTree) ApplyOperation(op *CrossShardOp, commitment []byte) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	// Add operation data as new leaf
	operationHash := core.ComputeHash(append(op.Data, commitment...))
	mt.Leaves = append(mt.Leaves, operationHash)

	leafNode := &MerkleNode{
		Hash:   operationHash,
		IsLeaf: true,
		Data:   op.Data,
	}
	mt.Nodes[operationHash] = leafNode

	// Rebuild tree
	mt.buildTree()

	return nil
}

// Rollback rolls back the last operation
func (mt *MerkleTree) Rollback() {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if len(mt.Leaves) > 0 {
		// Remove last leaf
		lastLeaf := mt.Leaves[len(mt.Leaves)-1]
		mt.Leaves = mt.Leaves[:len(mt.Leaves)-1]
		delete(mt.Nodes, lastLeaf)

		// Rebuild tree
		mt.buildTree()
	}
}

// CrossShardOp represents a cross-shard operation
type CrossShardOp struct {
	Type     string
	ShardIDs []uint32
	Data     []byte
}
