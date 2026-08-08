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

package windows

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dcjulian29/new-dev-vm/internal/config"
)

// maxComputerNameLength is the limit Windows imposes on a computer name.
const maxComputerNameLength = 15

func TestComputerNameFor(t *testing.T) {
	cases := []struct {
		name      string
		hostname  string
		want      string
		shortened bool
	}{
		{"short host name", "bender", "BENDERDEV", false},
		{"already upper case", "BENDER", "BENDERDEV", false},
		{"one below the limit", "abcdefghijk", "ABCDEFGHIJKDEV", false},
		{"exactly at the limit", "abcdefghijkl", "ABCDEFGHIJKLDEV", false},
		{"one past the limit", "abcdefghijklm", "ABCDEFGHIJKLDEV", true},
		{"far past the limit", "averylonghostnameindeed", "AVERYLONGHOSDEV", true},
		{"empty host name", "", "DEV", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, shortened := computerNameFor(c.hostname)

			if got != c.want {
				t.Errorf("computerNameFor(%q) = %q, want %q", c.hostname, got, c.want)
			}

			if shortened != c.shortened {
				t.Errorf("computerNameFor(%q) shortened = %v, want %v",
					c.hostname, shortened, c.shortened)
			}

			if len(got) > maxComputerNameLength {
				t.Errorf("computerNameFor(%q) = %q, length %d exceeds %d",
					c.hostname, got, len(got), maxComputerNameLength)
			}
		})
	}
}

// The suffix must still fit within the limit if it is ever changed.
func TestMaxHostnameLengthLeavesRoomForSuffix(t *testing.T) {
	if maxHostnameLength+len(nameSuffix) != maxComputerNameLength {
		t.Errorf("maxHostnameLength (%d) plus suffix %q does not total %d",
			maxHostnameLength, nameSuffix, maxComputerNameLength)
	}
}

// syncFiles are the files syncConfig injects for a VM.
var syncFiles = []string{"config.xml", "key.pem", "cert.pem"}

// newSyncSource creates a sync base path holding the named files for a VM.
func newSyncSource(t *testing.T, computerName string, files []string) string {
	t.Helper()

	base := t.TempDir()

	dir := filepath.Join(base, computerName)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("creating sync directory: %v", err)
	}

	for _, file := range files {
		path := filepath.Join(dir, file)
		if err := os.WriteFile(path, []byte("contents of "+file), 0o600); err != nil {
			t.Fatalf("writing %s: %v", file, err)
		}
	}

	return base
}

// newDrive creates a stand-in for the mounted VHDX with the Scripts directory
// that the start command injection would already have created.
func newDrive(t *testing.T) string {
	t.Helper()

	drive := t.TempDir()

	if err := os.MkdirAll(filepath.Join(drive, "Windows", "Setup", "Scripts"), 0o750); err != nil {
		t.Fatalf("creating Scripts directory: %v", err)
	}

	return drive
}

func TestSyncConfigCopiesEveryFile(t *testing.T) {
	const computerName = "BENDERDEV"

	base := newSyncSource(t, computerName, syncFiles)
	drive := newDrive(t)

	cfg := &config.Config{WindowsSyncBasePath: base}

	if err := syncConfig(drive, computerName, cfg); err != nil {
		t.Fatalf("syncConfig() returned an unexpected error: %v", err)
	}

	for _, file := range syncFiles {
		path := filepath.Join(drive, "Windows", "Setup", "Scripts", file)

		got, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			t.Fatalf("reading injected %s: %v", file, err)
		}

		if want := "contents of " + file; string(got) != want {
			t.Errorf("injected %s = %q, want %q", file, got, want)
		}
	}
}

// Syncthing is optional, so no configured base path means nothing to do.
func TestSyncConfigSkipsWhenNotConfigured(t *testing.T) {
	drive := newDrive(t)

	if err := syncConfig(drive, "BENDERDEV", &config.Config{}); err != nil {
		t.Fatalf("syncConfig() returned an unexpected error: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(drive, "Windows", "Setup", "Scripts"))
	if err != nil {
		t.Fatalf("reading Scripts directory: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("syncConfig() injected %d files without a configured base path", len(entries))
	}
}

// A configured base path means every file is required.
func TestSyncConfigFailsOnMissingFile(t *testing.T) {
	for _, missing := range syncFiles {
		t.Run(missing, func(t *testing.T) {
			const computerName = "BENDERDEV"

			var present []string

			for _, file := range syncFiles {
				if file != missing {
					present = append(present, file)
				}
			}

			base := newSyncSource(t, computerName, present)
			drive := newDrive(t)

			cfg := &config.Config{WindowsSyncBasePath: base}

			err := syncConfig(drive, computerName, cfg)
			if err == nil {
				t.Fatalf("syncConfig() returned no error with %s missing", missing)
			}

			if !strings.Contains(err.Error(), missing) {
				t.Errorf("error %q does not name the missing file %q", err, missing)
			}
		})
	}
}

func TestSyncConfigFailsWhenDirectoryMissing(t *testing.T) {
	base := newSyncSource(t, "BENDERDEV", syncFiles)
	drive := newDrive(t)

	cfg := &config.Config{WindowsSyncBasePath: base}

	// A different VM name means the per-VM directory does not exist.
	err := syncConfig(drive, "OTHERDEV", cfg)
	if err == nil {
		t.Fatal("syncConfig() returned no error for a missing VM directory")
	}

	if !strings.Contains(err.Error(), "OTHERDEV") {
		t.Errorf("error %q does not name the directory it looked in", err)
	}
}

// The source path is built from the computer name, so a shortened name must be
// reflected in the directory that is read.
func TestSyncConfigUsesShortenedComputerName(t *testing.T) {
	computerName, shortened := computerNameFor("averylonghostnameindeed")
	if !shortened {
		t.Fatalf("expected %q to be shortened", computerName)
	}

	base := newSyncSource(t, computerName, syncFiles)
	drive := newDrive(t)

	cfg := &config.Config{WindowsSyncBasePath: base}

	if err := syncConfig(drive, computerName, cfg); err != nil {
		t.Fatalf("syncConfig() returned an unexpected error: %v", err)
	}
}
