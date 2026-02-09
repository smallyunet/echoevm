package trie

import (
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// Node is the interface that all trie nodes must implement.
type Node interface {
	fstring(string) string
}

type (
	// FullNode is a branch node, containing 17 children.
	// 16 for each nibble (0-f), and 1 for the value at this node.
	FullNode struct {
		Children [17]Node // The children of the node
		flags    nodeFlag
	}

	// ShortNode is an extension or leaf node.
	// It has a key (encoded path) and a value (child node or value).
	ShortNode struct {
		Key   []byte
		Val   Node
		flags nodeFlag
	}

	// HashNode is a reference to a node that has not been loaded from the database yet.
	// It contains the hash of the referenced node.
	HashNode []byte

	// ValueNode is a leaf value.
	ValueNode []byte
)

// nodeFlag contains caching information for a node.
type nodeFlag struct {
	hash  hashNode // cached hash of the node (to avoid recomputing)
	dirty bool     // whether the node has changed
}

// hashNode is a cached hash of a node.
type hashNode struct {
	hash common.Hash
}

// fstring implements the Node interface.
func (n *FullNode) fstring(indent string) string {
	return fmt.Sprintf("%sFullNode", indent)
}

func (n *ShortNode) fstring(indent string) string {
	return fmt.Sprintf("%sShortNode{%x}", indent, n.Key)
}

func (n HashNode) fstring(indent string) string {
	return fmt.Sprintf("%sHashNode{%x}", indent, n)
}

func (n ValueNode) fstring(indent string) string {
	return fmt.Sprintf("%sValueNode{%x}", indent, n)
}

// EncodeRLP implements rlp.Encoder.
func (n *FullNode) EncodeRLP(w io.Writer) error {
	var children [17]interface{}
	for i, child := range n.Children {
		if child == nil {
			children[i] = []byte{}
		} else {
			children[i] = child
		}
	}
	return rlp.Encode(w, children)
}

func (n *ShortNode) EncodeRLP(w io.Writer) error {
	key := n.Key
	if _, ok := n.Val.(ValueNode); ok {
		if !hasTerm(key) {
			newKey := make([]byte, len(key)+1)
			copy(newKey, key)
			newKey[len(key)] = 16
			key = newKey
		}
	}
	return rlp.Encode(w, []interface{}{compactEncode(key), n.Val})
}

// Unwrap returns the value of the node if it's a ValueNode, or nil.
func (n ValueNode) Unwrap() []byte {
	return n
}

func (n HashNode) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []byte(n))
}

func (n ValueNode) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []byte(n))
}

// DecodeNode decodes a node from RLP data.
func DecodeNode(hash []byte, buf []byte) (Node, error) {
	if len(buf) == 0 {
		return nil, io.ErrUnexpectedEOF
	}

	// 1. Peek at the list
	kind, _, _, err := rlp.Split(buf)
	if err != nil {
		return nil, err
	}
	// A node must be a list
	if kind != rlp.List {
		return nil, fmt.Errorf("node is not a list")
	}

	// 2. Count elements to distinguish FullNode (17) vs ShortNode (2)
	// We can decode into []rlp.RawValue first
	var elems []rlp.RawValue
	if err := rlp.DecodeBytes(buf, &elems); err != nil {
		return nil, err
	}

	// 3. FullNode
	if len(elems) == 17 {
		n := &FullNode{flags: nodeFlag{dirty: true}}
		for i := 0; i < 16; i++ {
			n.Children[i], err = decodeChild(elems[i])
			if err != nil {
				return nil, err
			}
		}
		// 17th element is the value
		if len(elems[16]) > 0 {
			// It should be a string (bytes)
			var val []byte
			if err := rlp.DecodeBytes(elems[16], &val); err == nil && len(val) > 0 {
				n.Children[16] = ValueNode(val)
			}
		}
		return n, nil
	}

	// 4. ShortNode
	if len(elems) == 2 {
		n := &ShortNode{flags: nodeFlag{dirty: true}}
		// Decode key
		var key []byte
		if err := rlp.DecodeBytes(elems[0], &key); err != nil {
			return nil, err
		}
		n.Key = compactDecode(key)
		if len(n.Key) == 0 {
			// This might be OK if it's an extension node with empty key? No, extension must have key.
		}

		// Check if it is a leaf (has terminator)
		if hasTerm(n.Key) {
			// It is a leaf, Val is a value
			var val []byte
			if err := rlp.DecodeBytes(elems[1], &val); err != nil {
				return nil, err
			}
			n.Val = ValueNode(val)
			n.Key = n.Key[:len(n.Key)-1]
		} else {
			// It is an extension, Val is a node
			n.Val, err = decodeChild(elems[1])
			if err != nil {
				return nil, err
			}
		}
		return n, nil
	}

	return nil, fmt.Errorf("invalid node list size: %d", len(elems))
}

func decodeChild(buf []byte) (Node, error) {
	if len(buf) == 0 {
		return nil, nil // Should not happen for RawValue usually? RawValue includes prefix.
	}
	kind, content, _, err := rlp.Split(buf)
	if err != nil {
		return nil, err
	}

	// Empty string (0x80) -> nil
	if kind == rlp.String && len(content) == 0 {
		return nil, nil
	}

	// HashNode (32 bytes string)
	if kind == rlp.String && len(content) == 32 {
		return HashNode(content), nil // content is the bytes
	}

	// If it's a list, it's an inline node
	if kind == rlp.List {
		return DecodeNode(nil, buf)
	}

	// If it's a string < 32 bytes, it might be a small ShortNode serialized as string???
	// No, MPT spec says nodes < 32 bytes are stored inline.
	// If it is inline, it IS a valid RLP list (for node).
	// If it is a string, it must be HashNode or nil/empty.
	// Wait, what if it's a ValueNode?
	// ValueNode only appears in ShortNode val or FullNode val (17th).
	// But `decodeChild` is called for Branch children (0-15) or ShortNode val.
	// ShortNode val MUST be a Node (Branch or Leaf/Extension).
	// If ShortNode val is just bytes, it's a Leaf value?
	// If `Key` has terminator, `Val` is value.
	// My `ShortNode` structure handles `Val` as `Node`.
	// If it is a Leaf, `Val` should be `ValueNode`.
	// But `ValueNode` encodes as bytes.
	// So if `kind == rlp.String` and NOT 32 bytes (and not empty), handles as ValueNode?
	// But Branch children (0-15) CANNOT be ValueNode.
	// Only ShortNode val can be ValueNode (if leaf).
	// How to distinguish?
	// `decodeChild` logic depends on context?
	// If called from `ShortNode` and key indicates leaf...
	// But `DecodeNode` for `ShortNode` calls `decodeChild(elems[1])`.
	// If `n.Key` has terminator, `elems[1]` is the value.
	// So we should check `hasTerm` before decoding val?

	// If `kind == rlp.String`, it is likely a HashNode (if length 32) OR it is the Value itself (if leaf).
	// MPT ambiguity: "The value is stored in the node itself if it is small enough (<32 bytes)".
	// If it is HashNode, it refers to a DB lookup.
	// If it is Value, it is the data.
	// In strict MPT, Branch children are always Nodes (inlined or referenced by hash).
	// ShortNode val is Node (Extension/Branch) OR Value (Leaf).
	// If ShortNode is a Leaf, `Val` is the value bytes.

	// So I need context!
	// I'll update `DecodeNode` to handle ShortNode leaf logic separately.
	return nil, fmt.Errorf("unknown node type: kind=%v len=%d", kind, len(content))
}

func (n *FullNode) copy() *FullNode {
	copy := *n
	return &copy
}

func (n *ShortNode) copy() *ShortNode {
	copy := *n
	return &copy
}
