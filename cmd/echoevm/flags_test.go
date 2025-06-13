package main

import (
	"flag"
	"os"
	"testing"
)

func TestParseFlags(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"cmd", "-bin", "x.bin"}
	cfg := parseFlags()
	if cfg.Bin != "x.bin" {
		t.Fatalf("unexpected bin %s", cfg.Bin)
	}
}
