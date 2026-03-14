// Package linux provides functions to provision Linux VMs
package linux

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
	"strings"
	"time"

	"github.com/dcjulian29/go-toolbox/filesystem"
	"github.com/dcjulian29/new-dev-vm/internal/config"
	"github.com/dcjulian29/new-dev-vm/internal/hyperv"
)

type linuxVMParams struct {
	distro        string // "Ubuntu" or "Debian"
	isoPattern    string
	isoSearchPath string
}

// ProvisionUbuntu provisions an Ubuntu development VM.
func ProvisionUbuntu(cfg *config.Config) error {
	return provisionLinux(cfg, linuxVMParams{
		distro:        "Ubuntu",
		isoPattern:    cfg.UbuntuIsoPattern,
		isoSearchPath: cfg.UbuntuIsoSearchPath,
	})
}

// ProvisionDebian provisions a Debian development VM.
func ProvisionDebian(cfg *config.Config) error {
	return provisionLinux(cfg, linuxVMParams{
		distro:        "Debian",
		isoPattern:    cfg.DebianIsoPattern,
		isoSearchPath: cfg.DebianIsoSearchPath,
	})
}

func provisionLinux(cfg *config.Config, params linuxVMParams) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	computerName := strings.ToUpper(hostname) + "DEVL" + string(params.distro[0])

	fmt.Printf("\n[%s] Provisioning VM: %s\n", params.distro, computerName)

	fmt.Println("[1/9] Checking Hyper-V...")
	if err := hyperv.CheckHyperVEnabled(); err != nil {
		return err
	}

	fmt.Printf("[2/9] Locating '%s' ISO...\n", params.distro)
	isoPath, err := filesystem.SearchForFile(params.isoSearchPath, params.isoPattern)
	if err != nil {
		return fmt.Errorf("finding %s ISO: %w", params.distro, err)
	}

	fmt.Printf("       ISO: %s\n", isoPath)

	fmt.Println("[3/9] Creating VHDX...")
	vhdxPath := filepath.Join(computerName + ".vhdx")
	if err := hyperv.CreateDynamicVHDX(vhdxPath, cfg.LinuxDiskSizeBytes); err != nil {
		return fmt.Errorf("creating VHDX: %w", err)
	}

	fmt.Printf("       VHDX: %s\n", vhdxPath)

	fmt.Println("[4/9] Creating virtual machine (Generation 2)...")
	vmCfg := hyperv.VMConfig{
		Name:           computerName,
		VHDXPath:       vhdxPath,
		VirtualSwitch:  cfg.VirtualSwitch,
		MemoryBytes:    cfg.MemoryBytes,
		ProcessorCount: cfg.ProcessorCount,
		Generation:     2,
		SecureBoot:     !cfg.LinuxDisableSecureBoot,
	}

	if err := hyperv.CreateVM(vmCfg); err != nil {
		return err
	}

	fmt.Println("[5/9] Configuring VM...")
	if err := hyperv.SetProcessorCount(computerName, cfg.ProcessorCount); err != nil {
		return err
	}

	if cfg.LinuxDisableSecureBoot {
		fmt.Println("       Disabling Secure Boot...")
		if err := hyperv.DisableSecureBoot(computerName); err != nil {
			return err
		}
	} else {
		fmt.Println("       Setting Secure Boot template to MicrosoftUEFICertificateAuthority...")
		if err := hyperv.SetSecureBootTemplate(computerName, "MicrosoftUEFICertificateAuthority"); err != nil {
			return err
		}
	}

	if err := hyperv.DisableAutomaticCheckpoints(computerName); err != nil {
		return err
	}

	if err := hyperv.EnableCheckpoints(computerName); err != nil {
		return err
	}

	fmt.Printf("[6/9] Attaching %s ISO...\n", params.distro)
	if err := hyperv.AttachDVD(computerName, isoPath); err != nil {
		return fmt.Errorf("attaching ISO: %w", err)
	}

	fmt.Println("[7/9] Setting boot order (DVD first)...")
	if err := hyperv.SetBootOrderDVDFirst(computerName); err != nil {
		return fmt.Errorf("setting boot order: %w", err)
	}

	fmt.Println("[8/9] Starting VM...")
	if err := hyperv.StartVM(computerName); err != nil {
		return err
	}

	fmt.Println("[9/9] Opening console...")
	time.Sleep(2 * time.Second)
	if err := hyperv.OpenConsole(computerName); err != nil {
		fmt.Printf("Warning: could not open console: %v\n", err)
	}

	fmt.Printf("\n✓ %s VM %q provisioned successfully.\n", params.distro, computerName)

	fmt.Printf("  Complete the OS installation in the Hyper-V console window.\n")
	fmt.Printf("  The ISO will remain attached — eject it after installation.\n")

	return nil
}
