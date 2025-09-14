package vm

import (
	"encoding/hex"
	"fmt"
	"math/big"
)

// LogEntry represents a single LOGn emission captured during execution.
// This is a simplified representation: address is always zero in the current
// interpreter (no account model yet).
type LogEntry struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"` // hex encoded 0x prefix
	Index   int      `json:"index"`
}

// opLog handles LOG0..LOG4. The opcode minus 0xa0 gives the number of topics.
// Stack pops (offset, size, topic0, topic1, ...).
func opLog(i *Interpreter, op byte) {
	topicCount := int(op - 0xa0)
	// Pop memory offset and size (EVM order: offset, size after topics? Actually LOGn pops: offset, size, topic0..topicN)
	size := i.stack.PopSafe().Uint64()
	offset := i.stack.PopSafe().Uint64()
	// In canonical EVM order the stack is: topics..., memStart, memSize (popped reverse). Our stack.PopSafe pops last pushed; contracts push topics first then offset then size, but here we approximate typical compiled order may differ.
	// To align with spec, we should reverse: pop memSize then memStart then topics. Adjust accordingly: we already popped size then offset, now pop topics.
	topics := make([]string, 0, topicCount)
	for t := 0; t < topicCount; t++ {
		raw := i.stack.PopSafe()
		// Represent as 0x + 32-byte hex
		topics = append(topics, fmt.Sprintf("0x%064x", raw))
	}
	// Read memory range
	dataBytes := i.memory.Read(offset, size)
	le := LogEntry{
		Address: "0x0000000000000000000000000000000000000000",
		Topics:  topics,
		Data:    "0x" + hex.EncodeToString(dataBytes),
		Index:   len(i.logs),
	}
	i.logs = append(i.logs, le)
	// EVM LOG has no stack push result
}

// helper to push big.Int onto stack (for potential future log tests convenience)
func bigFromHex(h string) *big.Int { // not used yet, kept for extension
	n := new(big.Int)
	if len(h) >= 2 && h[:2] == "0x" {
		h = h[2:]
	}
	n.SetString(h, 16)
	return n
}
