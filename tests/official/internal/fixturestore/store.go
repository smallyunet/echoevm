package fixturestore

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const MarkerName = ".echoevm-fixtures.json"

type Manifest struct {
	Repository  string `json:"repository"`
	Release     string `json:"release"`
	PublishedAt string `json:"publishedAt"`
	Asset       string `json:"asset"`
	URL         string `json:"url"`
	Size        int64  `json:"size"`
	SHA256      string `json:"sha256"`
	LatestFork  string `json:"latestFork"`
	Includes    string `json:"includes"`
}

func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	if manifest.Repository == "" || manifest.Release == "" || manifest.Asset == "" || manifest.URL == "" || manifest.Size <= 0 || len(manifest.SHA256) != 64 || manifest.LatestFork == "" {
		return Manifest{}, errors.New("manifest is missing required release metadata")
	}
	if _, err := hex.DecodeString(manifest.SHA256); err != nil {
		return Manifest{}, fmt.Errorf("invalid manifest sha256: %w", err)
	}
	return manifest, nil
}

func Fetch(ctx context.Context, client *http.Client, manifest Manifest, archivePath, destination string) error {
	current, err := installedManifest(destination)
	if err == nil && current.Release == manifest.Release && current.SHA256 == manifest.SHA256 {
		if _, statErr := os.Stat(filepath.Join(destination, ".meta", "index.json")); statErr == nil {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		return err
	}
	valid, err := verifyArchive(archivePath, manifest)
	if err != nil {
		return err
	}
	if !valid {
		if err := download(ctx, client, manifest, archivePath); err != nil {
			return err
		}
	}
	return installArchive(archivePath, destination, manifest)
}

func installedManifest(destination string) (Manifest, error) {
	return LoadManifest(filepath.Join(destination, MarkerName))
}

func verifyArchive(path string, manifest Manifest) (bool, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return false, err
	}
	if stat.Size() != manifest.Size {
		return false, nil
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false, err
	}
	return hex.EncodeToString(hash.Sum(nil)) == strings.ToLower(manifest.SHA256), nil
}

func download(ctx context.Context, client *http.Client, manifest Manifest, archivePath string) error {
	tmp, err := os.CreateTemp(filepath.Dir(archivePath), ".fixtures-download-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifest.URL, nil)
	if err != nil {
		tmp.Close()
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		tmp.Close()
		return fmt.Errorf("download %s: %w", manifest.Release, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		tmp.Close()
		return fmt.Errorf("download %s: HTTP %s", manifest.Release, resp.Status)
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(tmp, hash), resp.Body)
	closeErr := tmp.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written != manifest.Size {
		return fmt.Errorf("fixture asset size mismatch: want %d, got %d", manifest.Size, written)
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != strings.ToLower(manifest.SHA256) {
		return fmt.Errorf("fixture asset sha256 mismatch: want %s, got %s", manifest.SHA256, got)
	}
	if err := os.Rename(tmpPath, archivePath); err != nil {
		return err
	}
	return nil
}

func installArchive(archivePath, destination string, manifest Manifest) error {
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(parent, ".fixtures-extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	if err := extractTarGZ(archivePath, staging); err != nil {
		return err
	}

	root := staging
	if _, err := os.Stat(filepath.Join(staging, "fixtures", ".meta", "index.json")); err == nil {
		root = filepath.Join(staging, "fixtures")
	} else if _, err := os.Stat(filepath.Join(staging, ".meta", "index.json")); err != nil {
		return errors.New("official fixture archive has no .meta/index.json at its root")
	}
	marker, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	marker = append(marker, '\n')
	if err := os.WriteFile(filepath.Join(root, MarkerName), marker, 0o644); err != nil {
		return err
	}
	if err := os.RemoveAll(destination); err != nil {
		return err
	}
	if err := os.Rename(root, destination); err != nil {
		return err
	}
	return nil
}

func extractTarGZ(archivePath, destination string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		clean := filepath.Clean(filepath.FromSlash(header.Name))
		if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("unsafe fixture archive path %q", header.Name)
		}
		target := filepath.Join(destination, clean)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			return fmt.Errorf("unsupported fixture archive entry %q (type %d)", header.Name, header.Typeflag)
		}
	}
}
