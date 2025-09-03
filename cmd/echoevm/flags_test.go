package main

import (
	"flag"
	"os"
	"testing"
)

func TestParseFlags(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"cmd", "run", "-bin", "x.bin"}
	_, cfg, err := parseFlags()
	if err != nil {
		t.Fatalf("parseFlags failed: %v", err)
	}
	if cfg.Bin != "x.bin" {
		t.Fatalf("unexpected bin %s", cfg.Bin)
	}
}

func TestParseFlagsArtifact(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"cmd", "run", "-artifact", "x.json"}
	_, cfg, err := parseFlags()
	if err != nil {
		t.Fatalf("parseFlags failed: %v", err)
	}
	if cfg.Artifact != "x.json" {
		t.Fatalf("unexpected artifact %s", cfg.Artifact)
	}
}
