package compliance

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestNoGoEthereumDependency(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	module, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	forbidden := []byte("github.com/ethereum/" + "go-ethereum")
	if bytes.Contains(module, forbidden) {
		t.Fatal("go.mod contains a go-ethereum dependency")
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || path == filepath.Join(root, "tests", "official", "fixtures") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(data, append([]byte{'"'}, forbidden...)) {
			t.Errorf("%s imports go-ethereum", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
