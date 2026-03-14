package hyperv

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
	"path/filepath"

	"github.com/dcjulian29/go-toolbox/filesystem"
	"github.com/dcjulian29/new-dev-vm/internal/ps"
)

// GetVMStoragePath returns the configured VM storage path from the Hyper-V host.
func GetVMStoragePath() (string, error) {
	script := `(Get-VMHost).VirtualHardDiskPath`
	directory, err := ps.RunPowershellOutput(script)
	if err != nil {
		return "", fmt.Errorf("retrieving default hard disk path: %w", err)
	}

	return directory, nil
}

// FindLatestBaseImage searches directoryPath (and one level of subdirectories)
// for a VHDX whose name matches pattern and returns the full path of the
// alphabetically last match (which is usually the newest dated image).
func FindLatestBaseImage(directoryPath, pattern string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(directoryPath, pattern))
	if err != nil {
		return "", err
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no base image matching '%q' found in '%s'", pattern, directoryPath)
	}

	latest := matches[0]

	for _, m := range matches[1:] {
		if m > latest {
			latest = m
		}
	}

	if !filesystem.FileExists(latest) {
		return "", fmt.Errorf("base image %q not found", latest)
	}

	return latest, nil
}

// CheckHyperVEnabled returns an error if the Hyper-V role is not available.
func CheckHyperVEnabled() error {
	out, err := ps.RunPowershellOutput(
		`(Get-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V).State`,
	)
	if err != nil {
		return fmt.Errorf("could not query Hyper-V feature state: %w", err)
	}

	if out != "Enabled" {
		return fmt.Errorf("the Hyper-V feature is not enabled on this host (state: %s)", out)
	}

	return nil
}
