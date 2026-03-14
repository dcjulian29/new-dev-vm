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

	"github.com/dcjulian29/new-dev-vm/internal/ps"
)

// CreateDifferencingVHDX creates a differencing VHDX.
func CreateDifferencingVHDX(reference, name string) error {
	directory, err := GetVMStoragePath()
	if err != nil {
		return err
	}

	script := fmt.Sprintf(
		`New-VHD -ParentPath "%s" -Path "%s\%s" -Differencing -ErrorAction Stop`,
		ps.Escape(reference), ps.Escape(directory), ps.Escape(name),
	)
	if err := ps.RunPowershell(script); err != nil {
		return fmt.Errorf("creating differencing VHDX: %w", err)
	}

	return nil
}

// CreateFixedVHDX creates a new fixed-size VHDX with the given size.
func CreateFixedVHDX(name string, sizeBytes int64) error {
	directory, err := GetVMStoragePath()
	if err != nil {
		return err
	}

	script := fmt.Sprintf(
		`New-VHD -Path "%s\%s" -SizeBytes %d -Fixed -ErrorAction Stop`,
		ps.Escape(directory), ps.Escape(name), sizeBytes,
	)
	if err := ps.RunPowershell(script); err != nil {
		return fmt.Errorf("creating fixed VHDX: %w", err)
	}

	return nil
}

// CreateDynamicVHDX creates a new dynamic VHDX with the given size.
func CreateDynamicVHDX(name string, sizeBytes int64) error {
	directory, err := GetVMStoragePath()
	if err != nil {
		return err
	}

	script := fmt.Sprintf(
		`New-VHD -Path "%s\%s" -SizeBytes %d -Dynamic -ErrorAction Stop`,
		ps.Escape(directory), ps.Escape(name), sizeBytes,
	)
	if err := ps.RunPowershell(script); err != nil {
		return fmt.Errorf("creating dynamic VHDX: %w", err)
	}
	return nil
}

// MountVHDX mounts the VHDX and returns the drive letter assigned by Windows.
func MountVHDX(name string) (string, error) {
	directory, err := GetVMStoragePath()
	if err != nil {
		return "", err
	}

	script := fmt.Sprintf(
		`$v = Mount-VHD -Path "%s\%s" -PassThru -ErrorAction Stop; `+
			`($v | Get-Disk | Get-Partition | Get-Volume).DriveLetter`,
		ps.Escape(directory), ps.Escape(name),
	)
	letter, err := ps.RunPowershellOutput(script)
	if err != nil {
		return "", fmt.Errorf("mounting VHDX %s: %w", filepath.Base(name), err)
	}

	if letter == "" {
		return "", fmt.Errorf("no drive letter assigned after mounting %s", filepath.Base(name))
	}

	return letter, nil
}

// DismountVHDX unmounts the VHDX at the given path.
func DismountVHDX(name string) error {
	directory, err := GetVMStoragePath()
	if err != nil {
		return err
	}

	script := fmt.Sprintf(
		`Dismount-VHD -Path "%s\%s" -ErrorAction SilentlyContinue`,
		ps.Escape(directory), ps.Escape(name),
	)
	return ps.RunPowershell(script)
}
