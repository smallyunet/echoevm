package trie

// keybytesToHex converts a key (byte slice) to a nibble slice.
// Each byte is split into two nibbles.
func keybytesToHex(str []byte) []byte {
	l := len(str) * 2
	var nibbles = make([]byte, l)
	for i, b := range str {
		nibbles[i*2] = b / 16
		nibbles[i*2+1] = b % 16
	}
	return nibbles
}

// hexToKeybytes converts a nibble slice back to a byte slice.
// This assumes the nibble slice has an even length.
func hexToKeybytes(hex []byte) []byte {
	if hasTerm(hex) {
		hex = hex[:len(hex)-1]
	}
	if len(hex)&1 != 0 {
		panic("can't convert odd-length hex string")
	}
	key := make([]byte, len(hex)/2)
	for i := 0; i < len(key); i++ {
		key[i] = hex[i*2]*16 + hex[i*2+1]
	}
	return key
}

// compactEncode encodes a hex slice (nibbles) into a compact byte slice.
// It adds a prefix to indicate whether the key length is even or odd,
// and whether the node is a leaf (terminated) or extension.
func compactEncode(hex []byte) []byte {
	term := 0
	if hasTerm(hex) {
		term = 1
		hex = hex[:len(hex)-1]
	}
	oddlen := len(hex) & 1
	flags := byte(2*term + oddlen)
	var firstByte byte = flags << 4
	if oddlen != 0 {
		firstByte |= hex[0]
		hex = hex[1:]
	}
	result := make([]byte, len(hex)/2+1)
	result[0] = firstByte
	for i := 0; i < len(hex)/2; i++ {
		result[i+1] = hex[2*i]*16 + hex[2*i+1]
	}
	return result
}

// hasTerm checks if a hex slice has the terminator flag.
func hasTerm(s []byte) bool {
	return len(s) > 0 && s[len(s)-1] == 16
}

// compactDecode decodes a compact encoded byte slice back to hex (nibbles).
func compactDecode(compact []byte) []byte {
	if len(compact) == 0 {
		return nil
	}
	base := keybytesToHex(compact)

	// base[0] is the high nibble of the first byte (flags)
	// flags: 00xx (even), 01xx (odd), 10xx (even term), 11xx (odd term)
	// The lowest bit of the flag indicates odd/even length.

	flag := base[0]
	isOdd := (flag & 1) != 0

	var keyNibbles []byte
	if isOdd {
		// Odd length: Skip the flag nibble
		keyNibbles = base[1:]
	} else {
		// Even length: Skip the flag nibble AND the padding nibble
		keyNibbles = base[2:]
	}

	isLeaf := (flag & 2) != 0
	if isLeaf {
		keyNibbles = append(keyNibbles, 16)
	}
	return keyNibbles
}
