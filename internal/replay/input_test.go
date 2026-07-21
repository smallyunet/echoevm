package replay

import "testing"

const testHash = "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestParseTransactionReference(t *testing.T) {
	tests := []struct {
		input string
		chain uint64
	}{
		{testHash, 0},
		{" https://etherscan.io/tx/" + testHash + "?utm_source=test ", 1},
		{"https://sepolia.etherscan.io/tx/" + testHash, 11155111},
	}
	for _, test := range tests {
		ref, err := ParseTransactionReference(test.input)
		if err != nil {
			t.Fatalf("ParseTransactionReference(%q): %v", test.input, err)
		}
		if ref.Hash.Hex() != testHash || ref.ChainID != test.chain {
			t.Fatalf("reference = %s/%d, want %s/%d", ref.Hash, ref.ChainID, testHash, test.chain)
		}
	}
}

func TestParseTransactionReferenceRejectsUnsafeOrMalformedInput(t *testing.T) {
	for _, input := range []string{"", "0x1234", "https://example.com/tx/" + testHash, "https://etherscan.io/address/" + testHash} {
		if _, err := ParseTransactionReference(input); err == nil {
			t.Fatalf("expected %q to be rejected", input)
		}
	}
}
