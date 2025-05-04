package core

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Blockchain represents the main blockchain structure
type Blockchain struct {
	mu            sync.RWMutex
	blocks        map[Hash]*Block
	heightToHash  map[uint64]Hash
	currentHeight uint64
	genesis       *Block
	shards        map[uint32]*ShardChain
	orphanBlocks  map[Hash]*Block
}

// ShardChain represents a shard's blockchain
type ShardChain struct {
	ShardID       uint32
	Blocks        map[Hash]*Block
	HeightToHash  map[uint64]Hash
	CurrentHeight uint64
	ParentShard   uint32
	ChildShards   []uint32
}

// NewBlockchain creates a new blockchain instance
func NewBlockchain() *Blockchain {
	bc := &Blockchain{
		blocks:       make(map[Hash]*Block),
		heightToHash: make(map[uint64]Hash),
		shards:       make(map[uint32]*ShardChain),
		orphanBlocks: make(map[Hash]*Block),
	}

	// Create genesis block
	bc.createGenesisBlock()

	return bc
}

// createGenesisBlock creates the genesis block
func (bc *Blockchain) createGenesisBlock() {
	genesis := &Block{
		Header: BlockHeader{
			Version:         1,
			Height:          0,
			Timestamp:       time.Now().Unix(),
			PrevBlockHash:   Hash{},
			Difficulty:      1,
			ShardID:         0,
			CrossShardLinks: []CrossShardLink{},
		},
		Transactions: []Transaction{},
	}

	// Compute merkle root
	genesis.Header.MerkleRoot = ComputeHash([]byte("genesis"))

	// Add genesis block
	bc.genesis = genesis
	bc.addBlock(genesis)
}

// AddBlock adds a new block to the blockchain
func (bc *Blockchain) AddBlock(block *Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Validate block
	if err := bc.validateBlock(block); err != nil {
		return err
	}

	// Fix: Add proper height validation
	expectedHeight := bc.currentHeight + 1
	if block.Header.Height != expectedHeight {
		return fmt.Errorf("invalid block height: expected %d, got %d", expectedHeight, block.Header.Height)
	}

	// Add block
	return bc.addBlock(block)
}

// validateBlock validates a block
func (bc *Blockchain) validateBlock(block *Block) error {
	// Check if previous block exists
	if block.Header.Height > 0 {
		if _, exists := bc.blocks[block.Header.PrevBlockHash]; !exists {
			// Store as orphan block
			bc.orphanBlocks[block.Header.PrevBlockHash] = block
			return errors.New("previous block not found")
		}
	}

	// Validate block header
	if err := bc.validateBlockHeader(&block.Header); err != nil {
		return err
	}

	// Validate transactions
	for _, tx := range block.Transactions {
		if err := bc.validateTransaction(&tx); err != nil {
			return err
		}
	}

	return nil
}

// validateBlockHeader validates a block header
func (bc *Blockchain) validateBlockHeader(header *BlockHeader) error {
	// Check version
	if header.Version == 0 {
		return errors.New("invalid block version")
	}

	// Check timestamp
	if header.Timestamp > time.Now().Unix()+300 { // 5 minutes in future max
		return errors.New("block timestamp too far in future")
	}

	// Check difficulty
	if header.Difficulty == 0 {
		return errors.New("invalid difficulty")
	}

	return nil
}

// validateTransaction validates a transaction
func (bc *Blockchain) validateTransaction(tx *Transaction) error {
	// Check if transaction ID is valid
	if tx.ID == (Hash{}) {
		return errors.New("invalid transaction ID")
	}

	// Check if addresses are valid
	if tx.From == (Address{}) || tx.To == (Address{}) {
		return errors.New("invalid addresses")
	}

	// Check signature
	if len(tx.Signature) == 0 {
		return errors.New("missing signature")
	}

	return nil
}

// addBlock adds a validated block to the chain
func (bc *Blockchain) addBlock(block *Block) error {
	blockHash := block.Header.Hash()

	// Add to main chain
	bc.blocks[blockHash] = block
	bc.heightToHash[block.Header.Height] = blockHash

	// Update current height
	if block.Header.Height > bc.currentHeight {
		bc.currentHeight = block.Header.Height
	}

	// Check for orphan blocks that can now be added
	bc.processOrphanBlocks(blockHash)

	return nil
}

// processOrphanBlocks processes orphan blocks that can now be added
func (bc *Blockchain) processOrphanBlocks(parentHash Hash) {
	if orphan, exists := bc.orphanBlocks[parentHash]; exists {
		delete(bc.orphanBlocks, parentHash)
		if err := bc.addBlock(orphan); err == nil {
			// Recursively process any orphans of this block
			bc.processOrphanBlocks(orphan.Header.Hash())
		}
	}
}

// GetBlock retrieves a block by hash
func (bc *Blockchain) GetBlock(hash Hash) (*Block, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	block, exists := bc.blocks[hash]
	if !exists {
		return nil, errors.New("block not found")
	}

	return block, nil
}

// GetBlockByHeight retrieves a block by height
func (bc *Blockchain) GetBlockByHeight(height uint64) (*Block, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	hash, exists := bc.heightToHash[height]
	if !exists {
		return nil, errors.New("block not found")
	}

	return bc.blocks[hash], nil
}

// GetCurrentHeight returns the current blockchain height
func (bc *Blockchain) GetCurrentHeight() uint64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return bc.currentHeight
}
