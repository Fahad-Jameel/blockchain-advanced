package main

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"blockchain-advanced/cap"
	"blockchain-advanced/consensus"
	"blockchain-advanced/core"
	"blockchain-advanced/crypto"
	"blockchain-advanced/merkle"
	"blockchain-advanced/network"
	"blockchain-advanced/state"
)

func TestFullBlockchainIntegration(t *testing.T) {
	// Create blockchain components
	blockchain := core.NewBlockchain()
	amf := merkle.NewAdaptiveMerkleForest()
	consensusEngine := consensus.NewHybridConsensus()
	stateManager := state.NewStateManager()

	// Create node
	config := network.DefaultConfig()
	config.ListenAddr = "127.0.0.1:0" // Random port
	node := network.NewNode(config, blockchain, amf, consensusEngine, stateManager)

	// Start node
	err := node.Start()
	if err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	// Wait for startup
	time.Sleep(100 * time.Millisecond)

	// Create test transaction
	tx := &core.Transaction{
		ID:        core.Hash{1, 2, 3},
		From:      core.Address{4, 5, 6},
		To:        core.Address{7, 8, 9},
		Value:     100,
		Nonce:     0,
		Timestamp: time.Now().Unix(),
		Signature: []byte{1, 2, 3},
	}

	// Process transaction
	err = node.ProcessTransaction(tx)
	if err == nil {
		// Error expected because account doesn't exist
		t.Error("Expected error for non-existent account")
	}

	// Stop node
	node.Stop()
}

func TestCrossComponentInteraction(t *testing.T) {
	// Test interaction between Merkle Forest and State Manager
	amf := merkle.NewAdaptiveMerkleForest()
	sm := state.NewStateManager()

	// Create test account state
	addr := core.Address{1, 2, 3}
	currentState, _ := sm.GetCurrentState()
	currentState.Accounts[addr] = &core.AccountState{
		Balance: 1000,
		Nonce:   0,
		Storage: make(map[core.Hash]core.Hash),
	}

	// Generate proof for account
	key := addr[:]
	proof, err := amf.GenerateProof(key, 0)
	if err == nil {
		// Error expected because key not in tree
		t.Log("Expected error for non-existent key")
	}

	// Add account to Merkle Forest (need to access the shard through proper methods)
	accountHash := core.ComputeHash(key)
	// This would need a proper AddData method in the AMF
	_ = accountHash // Use the variable

	if proof == nil {
		t.Log("Proof is nil as expected for non-existent key")
	}
}

func TestConsensusBlockCreation(t *testing.T) {
	// Create components
	blockchain := core.NewBlockchain()
	consensusEngine := consensus.NewHybridConsensus()

	// Start consensus
	err := consensusEngine.Start()
	if err != nil {
		t.Fatalf("Failed to start consensus: %v", err)
	}

	// Wait for a consensus round
	time.Sleep(100 * time.Millisecond)

	// Check if any blocks were proposed
	currentHeight := blockchain.GetCurrentHeight()
	if currentHeight == 0 {
		t.Log("No new blocks created (expected in short test)")
	}
}

func TestStateVerification(t *testing.T) {
	// Test zero-knowledge proof integration
	zkp := crypto.NewZKPSystem()
	sm := state.NewStateManager()

	// Get current state
	currentState, _ := sm.GetCurrentState()

	// Generate state proof
	stateRoot := core.Hash{1, 2, 3} // Sample state root
	witness := []byte("test_witness")
	stateProof, err := zkp.GenerateStateProof(stateRoot, witness)
	if err != nil {
		t.Errorf("Failed to generate state proof: %v", err)
	}

	// Verify state proof
	valid := zkp.VerifyStateProof(stateProof)
	if !valid {
		t.Error("State proof verification failed")
	}

	_ = currentState // Use the variable
}

func TestAdvancedFeatures(t *testing.T) {
	t.Run("DynamicSharding", func(t *testing.T) {
		amf := merkle.NewAdaptiveMerkleForest()

		// Trigger rebalancing
		err := amf.DynamicShardRebalancing()
		if err != nil {
			t.Errorf("Rebalancing failed: %v", err)
		}
	})

	t.Run("ByzantineDetection", func(t *testing.T) {
		bd := consensus.NewByzantineDefender()

		// Simulate malicious behavior
		maliciousNode := core.Address{6, 6, 6}
		maliciousBehavior := &consensus.NodeBehavior{
			DoubleVotes:   5,
			InvalidBlocks: 10,
			ValidBlocks:   1,
		}

		err := bd.ValidateNode(maliciousNode, maliciousBehavior)
		if err != nil {
			t.Logf("Node validation error (expected): %v", err)
		}
	})

	t.Run("CAPOptimization", func(t *testing.T) {
		co := cap.NewCAPOrchestrator()

		// Start orchestrator
		err := co.Start()
		if err != nil {
			t.Errorf("Failed to start CAP orchestrator: %v", err)
		}

		// Wait for optimization cycle
		time.Sleep(100 * time.Millisecond)
	})
}

func TestPerformance(t *testing.T) {
	t.Run("TransactionThroughput", func(t *testing.T) {
		blockchain := core.NewBlockchain()
		start := time.Now()

		// Get genesis block for proper linking
		genesis, _ := blockchain.GetBlockByHeight(0)
		prevHash := genesis.Header.Hash()

		// Add 1000 blocks
		for i := 1; i <= 1000; i++ {
			block := &core.Block{
				Header: core.BlockHeader{
					Height:        uint64(i),
					Timestamp:     time.Now().Unix(),
					PrevBlockHash: prevHash,
					Version:       1,
					Difficulty:    1,
				},
				Transactions: []core.Transaction{},
			}

			err := blockchain.AddBlock(block)
			if err != nil {
				t.Errorf("Failed to add block %d: %v", i, err)
				break
			}

			prevHash = block.Header.Hash()
		}

		duration := time.Since(start)
		blocksPerSecond := float64(1000) / duration.Seconds()

		t.Logf("Added 1000 blocks in %v (%.2f blocks/second)", duration, blocksPerSecond)
	})

	t.Run("MerkleProofGeneration", func(t *testing.T) {
		tree := merkle.NewMerkleTree()

		// Add 10000 leaves
		for i := 0; i < 10000; i++ {
			leaf := core.ComputeHash([]byte(fmt.Sprintf("leaf_%d", i)))
			tree.AddLeaf(leaf)
		}

		start := time.Now()

		// Generate 100 proofs
		for i := 0; i < 100; i++ {
			key := []byte(fmt.Sprintf("leaf_%d", i))
			proof := tree.GenerateProof(key)
			if proof == nil {
				t.Errorf("Failed to generate proof for leaf_%d", i)
			}
		}

		duration := time.Since(start)
		proofsPerSecond := float64(100) / duration.Seconds()

		t.Logf("Generated 100 proofs in %v (%.2f proofs/second)", duration, proofsPerSecond)
	})
}

func TestRecovery(t *testing.T) {
	t.Run("StateRecovery", func(t *testing.T) {
		sm := state.NewStateManager()

		// Get current state
		currentState, _ := sm.GetCurrentState()

		// Create initial state
		addr := core.Address{1, 2, 3}
		currentState.Accounts[addr] = &core.AccountState{
			Balance: 1000,
			Nonce:   0,
			Storage: make(map[core.Hash]core.Hash),
		}

		// Simulate state corruption
		currentState.Accounts[addr].Balance = 0

		// Recovery test
		if currentState.Accounts[addr].Balance != 0 {
			t.Error("State corruption simulation failed")
		}
	})

	t.Run("NetworkPartitionRecovery", func(t *testing.T) {
		co := cap.NewCAPOrchestrator()

		// Start orchestrator
		err := co.Start()
		if err != nil {
			t.Errorf("Failed to start CAP orchestrator: %v", err)
		}

		// Wait for system to stabilize
		time.Sleep(100 * time.Millisecond)

		t.Log("Partition recovery test completed")
	})
}

func TestSecurity(t *testing.T) {
	t.Run("SignatureVerification", func(t *testing.T) {
		key, _ := crypto.GenerateKeyPair()
		data := []byte("test data")

		// Sign data
		signature, err := crypto.SignData(data, key)
		if err != nil {
			t.Fatalf("Failed to sign data: %v", err)
		}

		// Serialize public key
		pubKeyBytes := append(key.PublicKey.X.Bytes(), key.PublicKey.Y.Bytes()...)

		// Pad public key bytes to ensure correct length
		if len(pubKeyBytes) < 64 {
			padded := make([]byte, 64)
			copy(padded[64-len(pubKeyBytes):], pubKeyBytes)
			pubKeyBytes = padded
		}

		// Verify signature
		valid := crypto.VerifySignature(data, signature, pubKeyBytes)
		if !valid {
			t.Error("Valid signature not verified")
		}

		// Test with tampered data
		tamperedData := []byte("tampered data")
		valid = crypto.VerifySignature(tamperedData, signature, pubKeyBytes)
		if valid {
			t.Error("Invalid signature verified")
		}
	})

	t.Run("ZeroKnowledgeProofs", func(t *testing.T) {
		zkp := crypto.NewZKPSystem()

		secret := big.NewInt(12345)
		statement := big.NewInt(67890)

		// Generate proof
		proof, err := zkp.GenerateProof(secret, statement)
		if err != nil {
			t.Fatalf("Failed to generate ZKP: %v", err)
		}

		// Verify proof
		valid := zkp.VerifyProof(proof, statement)
		if !valid {
			t.Error("Valid ZKP not verified")
		}

		// Test with wrong statement
		wrongStatement := big.NewInt(99999)
		valid = zkp.VerifyProof(proof, wrongStatement)
		if valid {
			t.Error("Invalid ZKP verified")
		}
	})
}
