package core

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// Hash represents a SHA-256 hash
type Hash [32]byte

func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

// Address represents a wallet address
type Address [20]byte

func (a Address) String() string {
	return hex.EncodeToString(a[:])
}

// Transaction represents a blockchain transaction
type Transaction struct {
	ID        Hash
	From      Address
	To        Address
	Value     uint64
	Nonce     uint64
	Timestamp int64
	Signature []byte
}

// Block represents a block in the blockchain
type Block struct {
	Header       BlockHeader
	Transactions []Transaction
}

// BlockHeader contains the metadata of a block
type BlockHeader struct {
	Version         uint32
	Height          uint64
	Timestamp       int64
	PrevBlockHash   Hash
	MerkleRoot      Hash
	StateRoot       Hash
	AccumulatorRoot Hash
	Nonce           uint64
	Difficulty      uint64
	ShardID         uint32
	CrossShardLinks []CrossShardLink
}

// CrossShardLink represents a link to another shard
type CrossShardLink struct {
	ShardID    uint32
	BlockHash  Hash
	StateProof []byte
}

// ShardState represents the state of a shard
type ShardState struct {
	ShardID      uint32
	StateRoot    Hash
	Accounts     map[Address]AccountState
	LastModified time.Time
}

// AccountState represents an account's state
type AccountState struct {
	Balance  uint64
	Nonce    uint64
	CodeHash Hash
	Storage  map[Hash]Hash
}

// ComputeHash calculates the hash of data
func ComputeHash(data []byte) Hash {
	return Hash(sha256.Sum256(data))
}
