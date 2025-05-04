package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"blockchain-advanced/consensus"
	"blockchain-advanced/core"
	"blockchain-advanced/merkle"
	"blockchain-advanced/network"
	"blockchain-advanced/state"
)

func main() {
	// Initialize logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Create new blockchain node
	node, err := createNode()
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}

	// Start the node
	if err := node.Start(); err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}

	fmt.Println("Blockchain node started successfully")

	// Wait for interrupt signal
	waitForShutdown(node)
}

func createNode() (*network.Node, error) {
	// Initialize components
	config := network.DefaultConfig()

	// Create blockchain
	blockchain := core.NewBlockchain()

	// Create Adaptive Merkle Forest
	amf := merkle.NewAdaptiveMerkleForest()

	// Create consensus engine
	consensusEngine := consensus.NewHybridConsensus()

	// Create state manager
	stateManager := state.NewStateManager()

	// Create network node
	node := network.NewNode(config, blockchain, amf, consensusEngine, stateManager)

	return node, nil
}

func waitForShutdown(node *network.Node) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
	node.Stop()
}
