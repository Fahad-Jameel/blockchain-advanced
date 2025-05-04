package state

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"time"
)

// StateCompressor handles state compression
type StateCompressor struct {
	compressionLevel int
}

func NewStateCompressor() *StateCompressor {
	return &StateCompressor{
		compressionLevel: gzip.BestCompression,
	}
}

// SerializableAccountState is a JSON-serializable version of AccountState
type SerializableAccountState struct {
	Balance  uint64            `json:"balance"`
	Nonce    uint64            `json:"nonce"`
	CodeHash string            `json:"codeHash"`
	Storage  map[string]string `json:"storage"`
}

// SerializableContractState is a JSON-serializable version of ContractState
type SerializableContractState struct {
	CodeHash string            `json:"codeHash"`
	Storage  map[string]string `json:"storage"`
	Balance  uint64            `json:"balance"`
	Nonce    uint64            `json:"nonce"`
}

func (sc *StateCompressor) CompressWorldState(state *WorldState) ([]byte, error) {
	// Create a fully serializable version of the state
	serializableState := struct {
		StateRoot       string                                `json:"stateRoot"`
		Accounts        map[string]*SerializableAccountState  `json:"accounts"`
		Contracts       map[string]*SerializableContractState `json:"contracts"`
		Shards          map[uint32]interface{}                `json:"shards"`
		AccumulatorRoot string                                `json:"accumulatorRoot"`
		Timestamp       time.Time                             `json:"timestamp"`
	}{
		StateRoot:       state.StateRoot.String(),
		Accounts:        make(map[string]*SerializableAccountState),
		Contracts:       make(map[string]*SerializableContractState),
		Shards:          make(map[uint32]interface{}),
		AccumulatorRoot: state.AccumulatorRoot.String(),
		Timestamp:       state.Timestamp,
	}

	// Convert Accounts to serializable format
	for addr, account := range state.Accounts {
		serializableAccount := &SerializableAccountState{
			Balance:  account.Balance,
			Nonce:    account.Nonce,
			CodeHash: account.CodeHash.String(),
			Storage:  make(map[string]string),
		}

		// Convert storage map
		for k, v := range account.Storage {
			serializableAccount.Storage[k.String()] = v.String()
		}

		serializableState.Accounts[addr.String()] = serializableAccount
	}

	// Convert Contracts
	for addr, contract := range state.Contracts {
		serializableContract := &SerializableContractState{
			CodeHash: contract.CodeHash.String(),
			Balance:  contract.Balance,
			Nonce:    contract.Nonce,
			Storage:  make(map[string]string),
		}

		// Convert storage map
		for k, v := range contract.Storage {
			serializableContract.Storage[k.String()] = v.String()
		}

		serializableState.Contracts[addr.String()] = serializableContract
	}

	// Convert Shards
	for id, shard := range state.Shards {
		serializableShard := map[string]interface{}{
			"shardID":      shard.ShardID,
			"stateRoot":    shard.StateRoot.String(),
			"lastModified": shard.LastModified,
			"accounts":     make(map[string]interface{}),
		}

		for addr, account := range shard.Accounts {
			serializableAccount := map[string]interface{}{
				"balance":  account.Balance,
				"nonce":    account.Nonce,
				"codeHash": account.CodeHash.String(),
				"storage":  make(map[string]string),
			}

			for k, v := range account.Storage {
				serializableAccount["storage"].(map[string]string)[k.String()] = v.String()
			}

			serializableShard["accounts"].(map[string]interface{})[addr.String()] = serializableAccount
		}

		serializableState.Shards[id] = serializableShard
	}

	// Serialize the state
	data, err := json.Marshal(serializableState)
	if err != nil {
		return nil, err
	}

	// Compress the data
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, sc.compressionLevel)
	if err != nil {
		return nil, err
	}

	_, err = writer.Write(data)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (sc *StateCompressor) DecompressWorldState(snapshot *StateSnapshot) (*WorldState, error) {
	// In a real implementation, would retrieve compressed data from storage
	return nil, errors.New("decompression not implemented")
}
