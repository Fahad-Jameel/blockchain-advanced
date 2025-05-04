package core

import (
	"testing"
	"time"
)

func TestTransactionValidation(t *testing.T) {
	bc := NewBlockchain()

	// Valid transaction
	validTx := Transaction{
		ID:        Hash{1, 2, 3},
		From:      Address{4, 5, 6},
		To:        Address{7, 8, 9},
		Value:     100,
		Nonce:     1,
		Timestamp: time.Now().Unix(),
		Signature: []byte{1, 2, 3},
	}

	err := bc.validateTransaction(&validTx)
	if err != nil {
		t.Errorf("Valid transaction failed validation: %v", err)
	}

	// Invalid transaction (empty ID)
	invalidTx := Transaction{
		From:      Address{4, 5, 6},
		To:        Address{7, 8, 9},
		Value:     100,
		Signature: []byte{1, 2, 3},
	}

	err = bc.validateTransaction(&invalidTx)
	if err == nil {
		t.Error("Expected error for invalid transaction, got nil")
	}
}

func TestAddressString(t *testing.T) {
	addr := Address{0x12, 0x34, 0x56}
	expected := "123456000000000000000000000000000000000000"

	if addr.String() != expected {
		t.Errorf("Expected %s, got %s", expected, addr.String())
	}
}

func TestHashString(t *testing.T) {
	hash := Hash{0xab, 0xcd, 0xef}
	expected := "abcdef0000000000000000000000000000000000000000000000000000000000"

	if hash.String() != expected {
		t.Errorf("Expected %s, got %s", expected, hash.String())
	}
}
