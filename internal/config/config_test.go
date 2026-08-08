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

package config

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	defaultMemoryBytes = 8 * 1024 * 1024 * 1024
	defaultDiskBytes   = 40 * 1024 * 1024 * 1024
)

// writeConfig points the home directory at a temporary location holding the
// given configuration file content.
func writeConfig(t *testing.T, content string) {
	t.Helper()

	home := t.TempDir()

	if err := os.MkdirAll(filepath.Join(home, ".config"), 0o750); err != nil {
		t.Fatalf("creating .config directory: %v", err)
	}

	path := filepath.Join(home, ".config", "new-dev-vm.yml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
}

func TestLoadAppliesDefaults(t *testing.T) {
	writeConfig(t, "virtualSwitch: \"Default Switch\"\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}

	if cfg.ProcessorCount != 4 {
		t.Errorf("ProcessorCount = %d, want 4", cfg.ProcessorCount)
	}

	if cfg.MemoryBytes != defaultMemoryBytes {
		t.Errorf("MemoryBytes = %d, want %d", cfg.MemoryBytes, defaultMemoryBytes)
	}

	if cfg.LinuxDiskSizeBytes != defaultDiskBytes {
		t.Errorf("LinuxDiskSizeBytes = %d, want %d", cfg.LinuxDiskSizeBytes, defaultDiskBytes)
	}
}

func TestLoadKeepsConfiguredValues(t *testing.T) {
	writeConfig(t, strings.Join([]string{
		"processorCount: 8",
		"memoryBytes: 2147483648",
		"linuxDiskSizeBytes: 1073741824",
	}, "\n"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}

	if cfg.ProcessorCount != 8 {
		t.Errorf("ProcessorCount = %d, want 8", cfg.ProcessorCount)
	}

	if cfg.MemoryBytes != 2147483648 {
		t.Errorf("MemoryBytes = %d, want 2147483648", cfg.MemoryBytes)
	}

	if cfg.LinuxDiskSizeBytes != 1073741824 {
		t.Errorf("LinuxDiskSizeBytes = %d, want 1073741824", cfg.LinuxDiskSizeBytes)
	}
}

// An unset maximum means static memory, so it must mirror the startup memory.
func TestLoadDefaultsMaximumMemoryToMemory(t *testing.T) {
	writeConfig(t, "memoryBytes: 2147483648\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}

	if cfg.MaximumMemoryBytes != cfg.MemoryBytes {
		t.Errorf("MaximumMemoryBytes = %d, want %d", cfg.MaximumMemoryBytes, cfg.MemoryBytes)
	}
}

func TestLoadKeepsLargerMaximumMemory(t *testing.T) {
	writeConfig(t, "memoryBytes: 2147483648\nmaximumMemoryBytes: 4294967296\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}

	if cfg.MaximumMemoryBytes != 4294967296 {
		t.Errorf("MaximumMemoryBytes = %d, want 4294967296", cfg.MaximumMemoryBytes)
	}
}

func TestLoadRejectsMaximumMemoryBelowMemory(t *testing.T) {
	writeConfig(t, "memoryBytes: 4294967296\nmaximumMemoryBytes: 2147483648\n")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() returned no error for a maximum below the startup memory")
	}

	if !strings.Contains(err.Error(), "maximumMemoryBytes") {
		t.Errorf("error %q does not mention maximumMemoryBytes", err)
	}
}

// The maximum is compared against the default when memoryBytes is omitted.
func TestLoadRejectsMaximumMemoryBelowDefaultMemory(t *testing.T) {
	writeConfig(t, "maximumMemoryBytes: 2147483648\n")

	if _, err := Load(); err == nil {
		t.Fatal("Load() returned no error for a maximum below the default memory")
	}
}

func TestLoadReportsMissingFile(t *testing.T) {
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	if _, err := Load(); err == nil {
		t.Fatal("Load() returned no error for a missing config file")
	}
}

func TestLoadReportsInvalidYAML(t *testing.T) {
	writeConfig(t, "memoryBytes: [this is not a number\n")

	if _, err := Load(); err == nil {
		t.Fatal("Load() returned no error for malformed YAML")
	}
}

// Every documented key must map to its field, so a rename cannot pass silently.
func TestLoadMapsEveryKey(t *testing.T) {
	writeConfig(t, strings.Join([]string{
		"maximumMemoryBytes: 8589934592",
		"memoryBytes: 4294967296",
		"processorCount: 2",
		"virtualSwitch: \"Lab Switch\"",
		"windowsBaseImagePath: \"C:\\\\images\"",
		"windowsBaseImagePattern: \"Base-*.vhdx\"",
		"windowsInstallPackage: \"devpkg\"",
		"windowsStartScript: \"C:\\\\scripts\\\\start.ps1\"",
		"windowsSyncBasePath: \"C:\\\\sync\"",
		"windowsUnattendTemplate: \"C:\\\\templates\\\\unattend.xml\"",
		"windowsUser: \"developer\"",
		"linuxDisableSecureBoot: true",
		"linuxDiskSizeBytes: 53687091200",
		"ubuntuIsoPattern: \"ubuntu-*.iso\"",
		"ubuntuIsoSearchPath: \"C:\\\\iso\\\\ubuntu\"",
		"debianIsoPattern: \"debian-*.iso\"",
		"debianIsoSearchPath: \"C:\\\\iso\\\\debian\"",
	}, "\n"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"MaximumMemoryBytes", cfg.MaximumMemoryBytes, int64(8589934592)},
		{"MemoryBytes", cfg.MemoryBytes, int64(4294967296)},
		{"ProcessorCount", cfg.ProcessorCount, 2},
		{"VirtualSwitch", cfg.VirtualSwitch, "Lab Switch"},
		{"WindowsBaseImagePath", cfg.WindowsBaseImagePath, `C:\images`},
		{"WindowsBaseImagePattern", cfg.WindowsBaseImagePattern, "Base-*.vhdx"},
		{"WindowsInstallPackage", cfg.WindowsInstallPackage, "devpkg"},
		{"WindowsStartScript", cfg.WindowsStartScript, `C:\scripts\start.ps1`},
		{"WindowsSyncBasePath", cfg.WindowsSyncBasePath, `C:\sync`},
		{"WindowsUnattendTemplate", cfg.WindowsUnattendTemplate, `C:\templates\unattend.xml`},
		{"WindowsUser", cfg.WindowsUser, "developer"},
		{"LinuxDisableSecureBoot", cfg.LinuxDisableSecureBoot, true},
		{"LinuxDiskSizeBytes", cfg.LinuxDiskSizeBytes, int64(53687091200)},
		{"UbuntuIsoPattern", cfg.UbuntuIsoPattern, "ubuntu-*.iso"},
		{"UbuntuIsoSearchPath", cfg.UbuntuIsoSearchPath, `C:\iso\ubuntu`},
		{"DebianIsoPattern", cfg.DebianIsoPattern, "debian-*.iso"},
		{"DebianIsoSearchPath", cfg.DebianIsoSearchPath, `C:\iso\debian`},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestConfigPathUsesHomeDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	path, err := configPath()
	if err != nil {
		t.Fatalf("configPath() returned an unexpected error: %v", err)
	}

	want := filepath.Join(home, ".config", "new-dev-vm.yml")
	if path != want {
		t.Errorf("configPath() = %q, want %q", path, want)
	}
}

func TestPrintIncludesConfiguredValues(t *testing.T) {
	writeConfig(t, "virtualSwitch: \"Lab Switch\"\nwindowsUser: \"developer\"\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}

	output := captureOutput(t, func() { Print(cfg) })

	for _, want := range []string{"maximumMemoryBytes", "memoryBytes", "processorCount", "Lab Switch"} {
		if !strings.Contains(output, want) {
			t.Errorf("Print() output does not contain %q\n%s", want, output)
		}
	}

	// The setting was removed; printing it again would resurrect dead config.
	if strings.Contains(output, "startLayout") {
		t.Errorf("Print() output still mentions startLayout\n%s", output)
	}
}

// captureOutput redirects standard output while fn runs and returns what it wrote.
func captureOutput(t *testing.T, fn func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}

	original := os.Stdout
	os.Stdout = writer

	done := make(chan string)

	go func() {
		var builder strings.Builder
		_, _ = io.Copy(&builder, reader)
		done <- builder.String()
	}()

	fn()

	os.Stdout = original

	if err := writer.Close(); err != nil {
		t.Fatalf("closing pipe: %v", err)
	}

	output := <-done

	if err := reader.Close(); err != nil {
		t.Fatalf("closing reader: %v", err)
	}

	return output
}
