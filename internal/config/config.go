// Package config provides functions to handle loading and printing the configuration
package config

/*
Copyright © 2026 Julian Easterling

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all settings loaded from ~/.config/new-dev-vm.yml.
type Config struct {
	// Shared
	MemoryBytes    int64  `yaml:"memoryBytes"`
	ProcessorCount int    `yaml:"processorCount"`
	VirtualSwitch  string `yaml:"virtualSwitch"`

	// Windows VM
	WindowsBaseImagePath    string `yaml:"windowsBaseImagePath"`
	WindowsBaseImagePattern string `yaml:"windowsBaseImagePattern"`
	WindowsInstallPackage   string `yaml:"windowsInstallPackage"`
	WindowsStartLayout      string `yaml:"windowsStartLayout"`
	WindowsStartScript      string `yaml:"windowsStartScript"`
	WindowsSyncBasePath     string `yaml:"windowsSyncBasePath"`
	WindowsUnattendTemplate string `yaml:"windowsUnattendTemplate"`
	WindowsUser             string `yaml:"windowsUser"`

	// Linux (shared)
	LinuxDisableSecureBoot bool  `yaml:"linuxDisableSecureBoot"`
	LinuxDiskSizeBytes     int64 `yaml:"linuxDiskSizeBytes"`

	// Ubuntu
	UbuntuIsoPattern    string `yaml:"ubuntuIsoPattern"`
	UbuntuIsoSearchPath string `yaml:"ubuntuIsoSearchPath"`

	// Debian
	DebianIsoPattern    string `yaml:"debianIsoPattern"`
	DebianIsoSearchPath string `yaml:"debianIsoSearchPath"`
}

// Load reads and parses the YAML config file, applying defaults if needed.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, fmt.Errorf("resolving config path: %w", err)
	}

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Apply defaults where values are missing.
	if cfg.ProcessorCount == 0 {
		cfg.ProcessorCount = 4
	}

	if cfg.MemoryBytes == 0 {
		cfg.MemoryBytes = 8 * 1024 * 1024 * 1024 // 8 GB
	}

	if cfg.LinuxDiskSizeBytes == 0 {
		cfg.LinuxDiskSizeBytes = 40 * 1024 * 1024 * 1024 // 40 GB
	}

	return &cfg, nil
}

// Print outputs the active configuration in a human-readable format.
func Print(cfg *Config) {
	path, _ := configPath()

	fmt.Printf("\nActive configuration  (%s)\n\n", path)
	fmt.Println("── Shared ──────────────────────────────────────────────────")
	fmt.Printf("  memoryBytes          : %d (%.1f GB)\n", cfg.MemoryBytes,
		float64(cfg.MemoryBytes)/1e9)
	fmt.Printf("  processorCount       : %d\n", cfg.ProcessorCount)
	fmt.Printf("  virtualSwitch        : %s\n", cfg.VirtualSwitch)
	fmt.Println()
	fmt.Println("── Windows VM ──────────────────────────────────────────────")
	fmt.Printf("  baseImagePath        : %s\n", cfg.WindowsBaseImagePath)
	fmt.Printf("  baseImagePattern     : %s\n", cfg.WindowsBaseImagePattern)
	fmt.Printf("  installPackage       : %s\n", cfg.WindowsInstallPackage)
	fmt.Printf("  unattendTemplate     : %s\n", cfg.WindowsUnattendTemplate)
	fmt.Printf("  startLayout          : %s\n", cfg.WindowsStartLayout)
	fmt.Printf("  startScript          : %s\n", cfg.WindowsStartScript)

	fmt.Println()
	fmt.Println("── Linux (shared) ──────────────────────────────────────────")
	fmt.Printf("  disableSecureBoot    : %v\n", cfg.LinuxDisableSecureBoot)
	fmt.Printf("  diskSizeBytes        : %d (%.1f GB)\n", cfg.LinuxDiskSizeBytes, float64(cfg.LinuxDiskSizeBytes)/1e9)
	fmt.Println()
	fmt.Println("── Ubuntu ──────────────────────────────────────────────────")
	fmt.Printf("  isoPattern           : %s\n", cfg.UbuntuIsoPattern)
	fmt.Printf("  isoSearchPath        : %s\n", cfg.UbuntuIsoSearchPath)
	fmt.Println()
	fmt.Println("── Debian ──────────────────────────────────────────────────")
	fmt.Printf("  isoPattern           : %s\n", cfg.DebianIsoPattern)
	fmt.Printf("  isoSearchPath        : %s\n", cfg.DebianIsoSearchPath)
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "new-dev-vm.yml"), nil
}
