package core

import (
	"bytes"
	"encoding/binary"
)

// Hash calculates the hash of a block header
func (bh *BlockHeader) Hash() Hash {
	var buf bytes.Buffer

	// Write all fields to buffer
	binary.Write(&buf, binary.BigEndian, bh.Version)
	binary.Write(&buf, binary.BigEndian, bh.Height)
	binary.Write(&buf, binary.BigEndian, bh.Timestamp)
	buf.Write(bh.PrevBlockHash[:])
	buf.Write(bh.MerkleRoot[:])
	buf.Write(bh.StateRoot[:])
	buf.Write(bh.AccumulatorRoot[:])
	binary.Write(&buf, binary.BigEndian, bh.Nonce)
	binary.Write(&buf, binary.BigEndian, bh.Difficulty)
	binary.Write(&buf, binary.BigEndian, bh.ShardID)

	// For cross-shard links
	for _, link := range bh.CrossShardLinks {
		binary.Write(&buf, binary.BigEndian, link.ShardID)
		buf.Write(link.BlockHash[:])
		buf.Write(link.StateProof)
	}

	return ComputeHash(buf.Bytes())
}
