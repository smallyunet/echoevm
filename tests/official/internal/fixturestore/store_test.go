package fixturestore

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func makeArchive(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, contents := range entries {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(contents)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestFetchVerifiesAndInstallsFixtureRoot(t *testing.T) {
	archive := makeArchive(t, map[string]string{
		"fixtures/.meta/index.json":        `{"fixture_formats":["state_test"],"forks":["Osaka"]}`,
		"fixtures/state_tests/sample.json": `{"sample":{"network":"Osaka"}}`,
	})
	digest := sha256.Sum256(archive)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	dir := t.TempDir()
	manifest := Manifest{Repository: "ethereum/execution-specs", Release: "tests@v20.0.0", Asset: "fixtures.tar.gz", URL: server.URL, Size: int64(len(archive)), SHA256: hex.EncodeToString(digest[:]), LatestFork: "Osaka"}
	destination := filepath.Join(dir, "fixtures")
	if err := Fetch(context.Background(), server.Client(), manifest, filepath.Join(dir, "cache", "fixtures.tar.gz"), destination); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, ".meta", "index.json")); err != nil {
		t.Fatal(err)
	}
	installed, err := installedManifest(destination)
	if err != nil {
		t.Fatal(err)
	}
	if installed.Release != manifest.Release || installed.SHA256 != manifest.SHA256 {
		t.Fatalf("unexpected installed marker: %+v", installed)
	}
}

func TestExtractRejectsPathTraversal(t *testing.T) {
	archive := makeArchive(t, map[string]string{"../outside": "bad"})
	path := filepath.Join(t.TempDir(), "fixtures.tar.gz")
	if err := os.WriteFile(path, archive, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGZ(path, filepath.Join(t.TempDir(), "out")); err == nil {
		t.Fatal("expected path traversal to be rejected")
	}
}
