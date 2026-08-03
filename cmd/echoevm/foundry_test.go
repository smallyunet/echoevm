package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadFoundrySettingsMergesProfileAndRemappingsFile(t *testing.T) {
	t.Setenv("FOUNDRY_PROFILE", "ci")
	directory := t.TempDir()
	config := `[profile.default]
optimizer = true
optimizer_runs = 100
via_ir = true
remappings = ["@openzeppelin/=lib/openzeppelin/"]

[profile.ci]
optimizer_runs = 500
remappings = ["@ci/=lib/ci/"]
`
	if err := os.WriteFile(filepath.Join(directory, "foundry.toml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "remappings.txt"), []byte("# generated\n@openzeppelin/=lib/openzeppelin/\n@extra/=lib/extra/\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	settings, err := loadFoundrySettings(directory)
	if err != nil {
		t.Fatal(err)
	}
	if !settings.Optimize || !settings.ViaIR || settings.OptimizerRuns != 500 {
		t.Fatalf("unexpected Foundry settings: %+v", settings)
	}
	want := []string{"@openzeppelin/=lib/openzeppelin/", "@ci/=lib/ci/", "@extra/=lib/extra/"}
	if !reflect.DeepEqual(settings.Remappings, want) {
		t.Fatalf("remappings = %#v, want %#v", settings.Remappings, want)
	}
}

func TestLoadFoundrySettingsRejectsMalformedRemapping(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "remappings.txt"), []byte("not-a-remapping\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := loadFoundrySettings(directory)
	if err == nil || !strings.Contains(err.Error(), "invalid Foundry remapping") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveSolidityCompilerSettingsLetsExplicitFlagsExtendFoundry(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "foundry.toml"), []byte("[profile.default]\noptimizer_runs = 100\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	settings, err := resolveSolidityCompilerSettings(directory, &solidityRunFlags{
		optimize: true, optimizerRuns: 200, viaIR: true, remappings: []string{"@a/=lib/a/"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !settings.Optimize || !settings.ViaIR || settings.OptimizerRuns != 200 || len(settings.Remappings) != 1 {
		t.Fatalf("unexpected merged settings: %+v", settings)
	}
}
