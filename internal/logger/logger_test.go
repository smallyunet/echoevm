package logger

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewDefaultLogger ensures that the default configuration builds a logger without error.
func TestNewDefaultLogger(t *testing.T) {
	cfg := DefaultConfig()
	if _, err := New(cfg); err != nil {
		t.Fatalf("expected no error creating default logger, got %v", err)
	}
}

// TestFileOutputLogger verifies that when a file path is provided a file is created.
func TestFileOutputLogger(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.log")
	cfg := DefaultConfig()
	cfg.Output = filePath
	l, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create file output logger: %v", err)
	}
	// Write one line to ensure file gets touched through the writer
	l.Info().Msg("hello")
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected log file to exist at %s: %v", filePath, err)
	}
}
