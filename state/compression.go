package state

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
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

func (sc *StateCompressor) CompressWorldState(state *WorldState) ([]byte, error) {
	// Serialize the state
	data, err := json.Marshal(state)
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
	// For now, return an error
	return nil, errors.New("decompression not implemented")
}
