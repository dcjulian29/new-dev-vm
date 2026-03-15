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

	"github.com/dcjulian29/go-toolbox/execute"
)

// CreateDifferencingVHDX creates a differencing VHDX.
func CreateDifferencingVHDX(referencePath, vhdxPath string) error {
	script := fmt.Sprintf(
		`New-VHD -ParentPath "%s" -Path "%s" -Differencing -ErrorAction Stop`,
		execute.EscapeForPowershell(referencePath),
		execute.EscapeForPowershell(vhdxPath),
	)

	if err := execute.RunPowershell(script); err != nil {
		return fmt.Errorf("creating differencing VHDX: %w", err)
	}

	return nil
}

// CreateFixedVHDX creates a new fixed-size VHDX with the given size.
func CreateFixedVHDX(vhdxPath string, sizeBytes int64) error {
	script := fmt.Sprintf(
		`New-VHD -Path "%s" -SizeBytes %d -Fixed -ErrorAction Stop`,
		execute.EscapeForPowershell(vhdxPath), sizeBytes,
	)

	if err := execute.RunPowershell(script); err != nil {
		return fmt.Errorf("creating fixed VHDX: %w", err)
	}

	return nil
}

// CreateDynamicVHDX creates a new dynamic VHDX with the given size.
func CreateDynamicVHDX(vhdxPath string, sizeBytes int64) error {
	script := fmt.Sprintf(
		`New-VHD -Path "%s" -SizeBytes %d -Dynamic -ErrorAction Stop`,
		execute.EscapeForPowershell(vhdxPath), sizeBytes,
	)

	if err := execute.RunPowershell(script); err != nil {
		return fmt.Errorf("creating dynamic VHDX: %w", err)
	}

	return nil
}

// MountVHDX mounts the VHDX and returns the drive letter assigned by Windows.
func MountVHDX(vhdxPath string) (string, error) {
	script := fmt.Sprintf(
		`$v = Mount-VHD -Path "%s" -PassThru -ErrorAction Stop; `+
			`($v | Get-Disk | Get-Partition | Get-Volume).DriveLetter`,
		execute.EscapeForPowershell(vhdxPath),
	)

	letter, err := execute.RunPowershellCapture(script)
	if err != nil {
		return "", fmt.Errorf("mounting VHDX %s: %w", filepath.Base(vhdxPath), err)
	}

	if letter == "" {
		return "", fmt.Errorf("no drive letter assigned after mounting %s", filepath.Base(vhdxPath))
	}

	return letter, nil
}

// DismountVHDX unmounts the VHDX at the given path.
func DismountVHDX(vhdxPath string) error {
	script := fmt.Sprintf(
		`Dismount-VHD -Path "%s" -ErrorAction SilentlyContinue`,
		execute.EscapeForPowershell(vhdxPath),
	)

	return execute.RunPowershell(script)
}
