package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type foundryFile struct {
	Profiles map[string]foundryProfile `toml:"profile"`
}

type foundryProfile struct {
	Optimizer     *bool    `toml:"optimizer"`
	OptimizerRuns *uint64  `toml:"optimizer_runs"`
	ViaIR         *bool    `toml:"via_ir"`
	Remappings    []string `toml:"remappings"`
}

type solidityCompilerSettings struct {
	Optimize      bool
	OptimizerRuns uint64
	ViaIR         bool
	Remappings    []string
}

func resolveSolidityCompilerSettings(basePath string, flags *solidityRunFlags) (solidityCompilerSettings, error) {
	settings, err := loadFoundrySettings(basePath)
	if err != nil {
		return solidityCompilerSettings{}, err
	}
	if flags.optimize {
		settings.Optimize = true
	}
	if flags.optimizerRuns > 0 {
		settings.OptimizerRuns = flags.optimizerRuns
	}
	if flags.viaIR {
		settings.ViaIR = true
	}
	settings.Remappings = uniqueStrings(append(settings.Remappings, flags.remappings...))
	return settings, nil
}

func loadFoundrySettings(basePath string) (solidityCompilerSettings, error) {
	var settings solidityCompilerSettings
	configPath := filepath.Join(basePath, "foundry.toml")
	var config foundryFile
	if _, err := toml.DecodeFile(configPath, &config); err != nil && !errors.Is(err, os.ErrNotExist) {
		return settings, fmt.Errorf("parse Foundry config %s: %w", configPath, err)
	}

	applyFoundryProfile(&settings, config.Profiles["default"])
	profileName := strings.TrimSpace(os.Getenv("FOUNDRY_PROFILE"))
	if profileName != "" && profileName != "default" {
		profile, ok := config.Profiles[profileName]
		if !ok {
			return settings, fmt.Errorf("Foundry profile %q is not defined in %s", profileName, configPath)
		}
		applyFoundryProfile(&settings, profile)
	}

	fileRemappings, err := readFoundryRemappings(filepath.Join(basePath, "remappings.txt"))
	if err != nil {
		return settings, err
	}
	settings.Remappings = uniqueStrings(append(settings.Remappings, fileRemappings...))
	return settings, nil
}

func applyFoundryProfile(settings *solidityCompilerSettings, profile foundryProfile) {
	if profile.Optimizer != nil {
		settings.Optimize = *profile.Optimizer
	}
	if profile.OptimizerRuns != nil {
		settings.OptimizerRuns = *profile.OptimizerRuns
	}
	if profile.ViaIR != nil {
		settings.ViaIR = *profile.ViaIR
	}
	if profile.Remappings != nil {
		settings.Remappings = append(settings.Remappings, profile.Remappings...)
	}
}

func readFoundryRemappings(path string) ([]string, error) {
	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Foundry remappings %s: %w", path, err)
	}
	var remappings []string
	for lineNumber, line := range strings.Split(string(contents), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, "=") {
			return nil, fmt.Errorf("invalid Foundry remapping at %s:%d: %q", path, lineNumber+1, line)
		}
		remappings = append(remappings, line)
	}
	return remappings, nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
