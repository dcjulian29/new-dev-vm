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
	"strings"

	"github.com/dcjulian29/go-toolbox/execute"
)

// VMConfig holds the parameters used when creating a Hyper-V VM.
type VMConfig struct {
	Name               string
	VHDXPath           string
	VirtualSwitch      string
	MaximumMemoryBytes int64
	MemoryBytes        int64
	ProcessorCount     int
	Generation         int  // 1 or 2
	SecureBoot         bool // only relevant for Generation 2
}

// CreateVM creates a new Hyper-V virtual machine with the provided config.
func CreateVM(cfg VMConfig) error {
	if cfg.Generation == 0 {
		cfg.Generation = 2
	}

	script := fmt.Sprintf(
		`New-VM -Generation %d -Name "%s" -MemoryStartupBytes %d `+
			`-SwitchName "%s"-VHDPath "%s" -ErrorAction Stop`,
		cfg.Generation,
		execute.EscapeForPowershell(cfg.Name),
		cfg.MemoryBytes,
		execute.EscapeForPowershell(cfg.VirtualSwitch),
		execute.EscapeForPowershell(cfg.VHDXPath),
	)

	if err := execute.RunPowershell(script); err != nil {
		return fmt.Errorf("creating VM %q: %w", cfg.Name, err)
	}

	return nil
}

// SetProcessorCount sets the number of virtual processors on an existing VM.
func SetProcessorCount(name string, count int) error {
	script := fmt.Sprintf(
		`Set-VMProcessor -VMName "%s" -Count %d -ErrorAction Stop`,
		execute.EscapeForPowershell(name), count,
	)

	return execute.RunPowershell(script)
}

// SetDynamicMemory enables dynamic memory with the given startup/min/max values.
func SetDynamicMemory(name string, startBytes, minBytes, maxBytes int64) error {
	script := fmt.Sprintf(
		`Set-VMMemory -VMName "%s" -DynamicMemoryEnabled $true `+
			`-StartupBytes %d -MinimumBytes %d -MaximumBytes %d -ErrorAction Stop`,
		execute.EscapeForPowershell(name), startBytes, minBytes, maxBytes,
	)

	return execute.RunPowershell(script)
}

// DisableSecureBoot turns off Secure Boot for a Generation 2 VM.
func DisableSecureBoot(name string) error {
	script := fmt.Sprintf(
		`Set-VMFirmware -VMName "%s" -EnableSecureBoot Off -ErrorAction Stop`,
		execute.EscapeForPowershell(name),
	)

	if err := execute.RunPowershell(script); err != nil {
		return fmt.Errorf("disabling Secure Boot for VM %q: %w", name, err)
	}

	return nil
}

// SetSecureBootTemplate sets the Secure Boot template (e.g. "MicrosoftUEFICertificateAuthority"
// for Linux, "MicrosoftWindows" for Windows).
func SetSecureBootTemplate(name, template string) error {
	script := fmt.Sprintf(
		`Set-VMFirmware -VMName "%s" -SecureBootTemplate "%s" -ErrorAction Stop`,
		execute.EscapeForPowershell(name),
		execute.EscapeForPowershell(template),
	)

	return execute.RunPowershell(script)
}

// AttachDVD attaches an ISO to the VM's DVD drive.
func AttachDVD(name, isoPath string) error {
	script := fmt.Sprintf(
		`Add-VMDvdDrive -VMName "%s" -Path "%s" -ErrorAction Stop`,
		execute.EscapeForPowershell(name),
		execute.EscapeForPowershell(isoPath),
	)

	return execute.RunPowershell(script)
}

// SetBootOrderDVDFirst sets the firmware boot order so that the DVD drive is first
func SetBootOrderDVDFirst(name string) error {
	script := fmt.Sprintf(
		`$vm = Get-VM -Name "%s"; `+
			`$dvd = Get-VMDvdDrive -VMName "%s"; `+
			`$hd  = Get-VMHardDiskDrive -VMName "%s"; `+
			`Set-VMFirmware -VM $vm -BootOrder $dvd,$hd -ErrorAction Stop`,
		execute.EscapeForPowershell(name),
		execute.EscapeForPowershell(name),
		execute.EscapeForPowershell(name),
	)

	return execute.RunPowershell(script)
}

// EnableCheckpoints enables standard (production) checkpoints.
func EnableCheckpoints(name string) error {
	script := fmt.Sprintf(
		`Set-VM -Name "%s" -CheckpointType Standard -ErrorAction Stop`,
		execute.EscapeForPowershell(name),
	)

	return execute.RunPowershell(script)
}

// DisableAutomaticCheckpoints turns off automatic checkpoints.
func DisableAutomaticCheckpoints(name string) error {
	script := fmt.Sprintf(
		`Set-VM -Name "%s" -AutomaticCheckpointsEnabled $false -ErrorAction Stop`,
		execute.EscapeForPowershell(name),
	)

	return execute.RunPowershell(script)
}

// StartVM starts the named virtual machine.
func StartVM(name string) error {
	script := fmt.Sprintf(`Start-VM -Name "%s" -ErrorAction Stop`, execute.EscapeForPowershell(name))

	if err := execute.RunPowershell(script); err != nil {
		return fmt.Errorf("starting VM %q: %w", name, err)
	}

	return nil
}

// OpenConsole opens the Hyper-V Virtual Machine Connection console.
func OpenConsole(name string) error {
	script := fmt.Sprintf(`vmconnect.exe localhost "%s"`, execute.EscapeForPowershell(name))

	return execute.RunPowershell(script)
}

// RemoveVM forcefully removes the VM and all of its associated files.
func RemoveVM(name string) error {
	script := fmt.Sprintf(
		`Stop-VM -Name "%s" -Force -TurnOff -ErrorAction SilentlyContinue; `+
			`Remove-VM -Name "%s" -Force -ErrorAction Stop`,
		execute.EscapeForPowershell(name),
		execute.EscapeForPowershell(name),
	)

	return execute.RunPowershell(script)
}

// VMExists returns true when a VM with that name is registered in Hyper-V.
func VMExists(name string) bool {
	script := fmt.Sprintf(
		`(Get-VM -VMName '%s' -ErrorAction SilentlyContinue) -ne $null`,
		execute.EscapeForPowershell(name),
	)

	exist, err := execute.RunPowershellCapture(script)

	return err == nil && strings.EqualFold(exist, "True")
}

// VMState returns the current state string ("Running", "Off", "Saved", ...).
func VMState(name string) (string, error) {
	script := fmt.Sprintf(`(Get-VM -VMName '%s').State`, execute.EscapeForPowershell(name))

	return execute.RunPowershellCapture(script)
}
