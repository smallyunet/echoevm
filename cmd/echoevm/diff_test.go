package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/smallyunet/echoevm/internal/differential"
)

func TestRunDiffTextAndJSON(t *testing.T) {
	for _, format := range []string{"text", "json"} {
		var out bytes.Buffer
		err := runDiff(context.Background(), &out, &diffFlags{code: "60026003015f5260205ff3", input: "0x", gas: 1_000_000, fork: "Cancun", format: format})
		if err != nil {
			t.Fatal(err)
		}
		if format == "text" && !strings.Contains(out.String(), "MATCH") {
			t.Fatalf("missing MATCH: %s", out.String())
		}
		if format == "json" {
			var result differential.ComparisonResult
			if err := json.Unmarshal(out.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if !result.Match {
				t.Fatalf("unexpected divergence: %+v", result.FirstDivergence)
			}
		}
	}
}

func TestRunDiffRejectsInvalidInput(t *testing.T) {
	for _, flags := range []*diffFlags{{format: "text"}, {code: "zz", input: "0x", gas: 1, fork: "Cancun", format: "text"}, {code: "00", input: "0x", gas: differential.MaxGasLimit + 1, fork: "Cancun", format: "text"}, {code: "00", input: "0x", gas: 1, fork: "Cancun", format: "yaml"}} {
		if err := runDiff(context.Background(), &bytes.Buffer{}, flags); err == nil {
			t.Fatalf("expected error for %+v", flags)
		}
	}
}
