package main

import (
	"fmt"
	"math/big"
	"testing"

	"blockchain-advanced/core"
	"blockchain-advanced/crypto"
	"blockchain-advanced/merkle"
	"blockchain-advanced/state"
)

func BenchmarkBlockValidation(b *testing.B) {
	blockchain := core.NewBlockchain()
	genesis, _ := blockchain.GetBlockByHeight(0)

	block := &core.Block{
		Header: core.BlockHeader{
			Height:        1,
			PrevBlockHash: genesis.Header.Hash(),
			Version:       1,
			Difficulty:    1,
		},
		Transactions: []core.Transaction{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		blockchain.AddBlock(block)
	}
}

func BenchmarkMerkleProofGeneration(b *testing.B) {
	tree := merkle.NewMerkleTree()

	// Add 1000 leaves
	for i := 0; i < 1000; i++ {
		leaf := core.ComputeHash([]byte(fmt.Sprintf("leaf_%d", i)))
		tree.AddLeaf(leaf)
	}

	key := []byte("leaf_500")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.GenerateProof(key)
	}
}

func BenchmarkZKPGeneration(b *testing.B) {
	zkp := crypto.NewZKPSystem()
	secret := big.NewInt(12345)
	statement := big.NewInt(67890)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		zkp.GenerateProof(secret, statement)
	}
}

func BenchmarkVRFGeneration(b *testing.B) {
	vrf := crypto.NewVRFOracle()
	seed := uint64(12345)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vrf.Generate(seed)
	}
}

func BenchmarkStateUpdate(b *testing.B) {
	sm := state.NewStateManager()

	// Get current state
	currentState, _ := sm.GetCurrentState()

	// Create test account
	addr := core.Address{1, 2, 3}
	currentState.Accounts[addr] = &core.AccountState{
		Balance: 1000000,
		Nonce:   0,
		Storage: make(map[core.Hash]core.Hash),
	}

	block := &core.Block{
		Header: core.BlockHeader{
			Height: 1,
		},
		Transactions: []core.Transaction{
			{
				From:  addr,
				To:    core.Address{4, 5, 6},
				Value: 100,
				Nonce: 0,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm.UpdateState(block)
		// Update nonce for next iteration
		currentState.Accounts[addr].Nonce = uint64(i + 1)
		block.Transactions[0].Nonce = uint64(i + 1)
	}
}

func BenchmarkHashComputation(b *testing.B) {
	data := []byte("benchmark test data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		core.ComputeHash(data)
	}
}

func BenchmarkBlockHeaderHash(b *testing.B) {
	header := core.BlockHeader{
		Version:         1,
		Height:          1000,
		Timestamp:       1234567890,
		PrevBlockHash:   core.Hash{1, 2, 3},
		MerkleRoot:      core.Hash{4, 5, 6},
		StateRoot:       core.Hash{7, 8, 9},
		AccumulatorRoot: core.Hash{10, 11, 12},
		Nonce:           12345,
		Difficulty:      100,
		ShardID:         1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		header.Hash()
	}
}
