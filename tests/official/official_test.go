package official

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/smallyunet/echoevm/tests/official/internal/fixturestore"
)

type fixtureIndex struct {
	FixtureFormats []string `json:"fixture_formats"`
	Forks          []string `json:"forks"`
	TestCount      int      `json:"test_count"`
	TestCases      []struct {
		JSONPath string `json:"json_path"`
	} `json:"test_cases"`
}

func TestPinnedOfficialManifest(t *testing.T) {
	manifest, err := fixturestore.LoadManifest("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Repository != "ethereum/execution-specs" || manifest.Release != "tests@v20.0.0" || manifest.LatestFork != "Osaka" {
		t.Fatalf("unexpected official fixture pin: %+v", manifest)
	}
}

func TestOfficialFixtureInventory(t *testing.T) {
	root := os.Getenv("ECHOEVM_OFFICIAL_FIXTURES")
	if root == "" {
		t.Skip("full official fixtures are opt-in; run make test-official-fixtures")
	}
	indexData, err := os.ReadFile(filepath.Join(root, ".meta", "index.json"))
	if err != nil {
		t.Fatalf("read official fixture index: %v", err)
	}
	var index fixtureIndex
	if err := json.Unmarshal(indexData, &index); err != nil {
		t.Fatalf("decode official fixture index: %v", err)
	}
	if len(index.FixtureFormats) == 0 || len(index.Forks) == 0 || index.TestCount == 0 {
		t.Fatalf("official fixture index is missing formats or forks: %+v", index)
	}
	if len(index.TestCases) != index.TestCount {
		t.Fatalf("official fixture index declares %d cases but lists %d", index.TestCount, len(index.TestCases))
	}
	indexedPaths := make(map[string]struct{})
	for _, testCase := range index.TestCases {
		if testCase.JSONPath == "" {
			t.Fatal("official fixture index contains an empty json_path")
		}
		indexedPaths[testCase.JSONPath] = struct{}{}
	}

	shard, shards := fixtureShard(t)
	files, auxiliaryFiles, cases := 0, 0, 0
	formats := make(map[string]int)
	observedPaths := make(map[string]struct{})
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if filepath.Ext(path) != ".json" || rel == "index.json" || strings.HasPrefix(rel, ".meta/") || rel == fixturestore.MarkerName {
			return nil
		}
		if fixtureShardFor(rel, shards) != shard {
			return nil
		}
		auxiliary := strings.Contains(rel, "/pre_alloc/")
		if _, indexed := indexedPaths[rel]; !indexed && !auxiliary {
			return fmt.Errorf("fixture file %s is absent from the official index", rel)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var contents map[string]json.RawMessage
		if err := json.Unmarshal(data, &contents); err != nil {
			return fmt.Errorf("decode %s: %w", rel, err)
		}
		fileCases := 0
		for name := range contents {
			if !strings.HasPrefix(name, "_") {
				fileCases++
			}
		}
		if fileCases == 0 {
			return fmt.Errorf("fixture file %s contains no test cases", rel)
		}
		if auxiliary {
			auxiliaryFiles++
			return nil
		}
		format := strings.SplitN(rel, "/", 2)[0]
		formats[format] += fileCases
		files++
		cases += fileCases
		observedPaths[rel] = struct{}{}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files == 0 || cases == 0 {
		t.Fatalf("official fixture shard %d/%d selected no fixture cases", shard, shards)
	}
	for path := range indexedPaths {
		if fixtureShardFor(path, shards) != shard {
			continue
		}
		if _, observed := observedPaths[path]; !observed {
			t.Fatalf("official index path %s is missing from fixture shard %d/%d", path, shard, shards)
		}
	}
	t.Logf("OFFICIAL FIXTURE INVENTORY release=tests@v20.0.0 shard=%d/%d fixtureFiles=%d auxiliaryFiles=%d jsonEntries=%d indexedCases=%d formats=%s indexForks=%s", shard, shards, files, auxiliaryFiles, cases, index.TestCount, formatCounts(formats), strings.Join(index.Forks, ","))
}

func fixtureShard(t *testing.T) (int, int) {
	t.Helper()
	value := os.Getenv("ECHOEVM_FIXTURE_SHARD")
	if value == "" {
		return 0, 1
	}
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		t.Fatalf("ECHOEVM_FIXTURE_SHARD must be index/count, got %q", value)
	}
	shard, err1 := strconv.Atoi(parts[0])
	shards, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || shards < 1 || shard < 0 || shard >= shards {
		t.Fatalf("invalid ECHOEVM_FIXTURE_SHARD %q", value)
	}
	return shard, shards
}

func fixtureShardFor(path string, shards int) int {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(path))
	return int(hash.Sum32() % uint32(shards))
}

func formatCounts(counts map[string]int) string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return strings.Join(parts, ",")
}
