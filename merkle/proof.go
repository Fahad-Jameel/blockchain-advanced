package merkle

import (
	"bytes"

	"blockchain-advanced/core"
	"blockchain-advanced/crypto"
)

// MerkleProof represents a Merkle proof with advanced features
type MerkleProof struct {
	Path       []ProofNode
	Root       core.Hash
	AMQFilter  *crypto.AMQFilter
	Compressed bool
	ProofType  ProofType
}

// ProofNode represents a node in the proof path
type ProofNode struct {
	Hash     core.Hash
	Position Position
	IsLeaf   bool
}

// Position indicates whether the node is left or right
type Position int

const (
	Left Position = iota
	Right
)

// ProofType indicates the type of proof
type ProofType int

const (
	StandardProof ProofType = iota
	CompressedProof
	ProbabilisticProof
	AccumulatorProof
)

// ProofGenerator generates and manages proofs
type ProofGenerator struct {
	compressor   *ProofCompressor
	amqGenerator *AMQGenerator
}

// NewProofGenerator creates a new proof generator
func NewProofGenerator() *ProofGenerator {
	return &ProofGenerator{
		compressor:   NewProofCompressor(),
		amqGenerator: NewAMQGenerator(),
	}
}

// CompressProof applies probabilistic compression to a proof
func (pg *ProofGenerator) CompressProof(proof *MerkleProof) *MerkleProof {
	if proof == nil {
		return nil
	}

	compressed := &MerkleProof{
		Root:       proof.Root,
		Compressed: true,
		ProofType:  CompressedProof,
	}

	// Apply probabilistic sampling
	sampledPath := pg.compressor.SamplePath(proof.Path)
	compressed.Path = sampledPath

	// Generate AMQ filter
	compressed.AMQFilter = pg.amqGenerator.GenerateFilter(proof.Path)

	return compressed
}

// DecompressProof decompresses a compressed proof
func (pg *ProofGenerator) DecompressProof(proof *MerkleProof) *MerkleProof {
	if !proof.Compressed {
		return proof
	}

	decompressed := &MerkleProof{
		Root:       proof.Root,
		Compressed: false,
		ProofType:  StandardProof,
		Path:       proof.Path, // In real implementation, would reconstruct
	}

	return decompressed
}

// CreateAMQFilter creates an Approximate Membership Query filter
func (pg *ProofGenerator) CreateAMQFilter(proof *MerkleProof) *crypto.AMQFilter {
	return pg.amqGenerator.GenerateFilter(proof.Path)
}

// Verify verifies a Merkle proof
func (proof *MerkleProof) Verify(key []byte, value []byte) bool {
	// Quick check with AMQ filter if available
	if proof.AMQFilter != nil && !proof.AMQFilter.Contains(key) {
		return false
	}

	// Calculate leaf hash
	leafHash := core.ComputeHash(append(key, value...))
	currentHash := leafHash

	// Traverse proof path
	for _, node := range proof.Path {
		if node.Position == Left {
			currentHash = core.ComputeHash(append(node.Hash[:], currentHash[:]...))
		} else {
			currentHash = core.ComputeHash(append(currentHash[:], node.Hash[:]...))
		}
	}

	return bytes.Equal(currentHash[:], proof.Root[:])
}

// ProofCompressor handles proof compression
type ProofCompressor struct {
	samplingRate    float64
	minNodes        int
	compressionAlgo string
}

// NewProofCompressor creates a new proof compressor
func NewProofCompressor() *ProofCompressor {
	return &ProofCompressor{
		samplingRate:    0.7,
		minNodes:        3,
		compressionAlgo: "probabilistic",
	}
}

// SamplePath applies probabilistic sampling to proof path
func (pc *ProofCompressor) SamplePath(path []ProofNode) []ProofNode {
	if len(path) <= pc.minNodes {
		return path
	}

	sampled := make([]ProofNode, 0)

	// Always include first and last
	if len(path) > 0 {
		sampled = append(sampled, path[0])
	}

	// Probabilistically sample intermediate nodes
	for i := 1; i < len(path)-1; i++ {
		if pc.shouldSample(i, len(path)) {
			sampled = append(sampled, path[i])
		}
	}

	// Always include last
	if len(path) > 1 {
		sampled = append(sampled, path[len(path)-1])
	}

	return sampled
}

// shouldSample determines if a node should be sampled
func (pc *ProofCompressor) shouldSample(index, totalLength int) bool {
	// More important nodes have higher probability
	importance := 1.0 - float64(index)/float64(totalLength)
	return importance > (1.0 - pc.samplingRate)
}

// AMQGenerator generates Approximate Membership Query filters
type AMQGenerator struct {
	falsePositiveRate float64
	hashFunctions     int
}

// NewAMQGenerator creates a new AMQ generator
func NewAMQGenerator() *AMQGenerator {
	return &AMQGenerator{
		falsePositiveRate: 0.01,
		hashFunctions:     4,
	}
}

// GenerateFilter generates an AMQ filter from proof path
func (ag *AMQGenerator) GenerateFilter(path []ProofNode) *crypto.AMQFilter {
	filter := crypto.NewAMQFilter(ag.falsePositiveRate, ag.hashFunctions)

	for _, node := range path {
		filter.Add(node.Hash[:])
	}

	return filter
}
