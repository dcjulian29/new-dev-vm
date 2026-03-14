// Package windows provides functions to provision Windows VMs
package windows

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
	"strings"
	"time"

	"github.com/dcjulian29/new-dev-vm/internal/config"
	"github.com/dcjulian29/new-dev-vm/internal/disk"
	"github.com/dcjulian29/new-dev-vm/internal/hyperv"
	"github.com/dcjulian29/new-dev-vm/internal/util"
)

// ProvisionWindows creates a Windows development VM
func ProvisionWindows(cfg *config.Config) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	computerName := strings.ToUpper(hostname) + "DEV"
	prompt := fmt.Sprintf("Enter password for '%s' on %s: ", cfg.WindowsUser, computerName)
	password, err := util.PromptPassword(prompt)
	if err != nil {
		return fmt.Errorf("prompting for password: %w", err)
	}

	fmt.Printf("\n[Windows] Provisioning VM: %s\n", computerName)

	fmt.Println("[1/9] Checking Hyper-V...")
	if err := hyperv.CheckHyperVEnabled(); err != nil {
		return err
	}

	fmt.Println("[2/9] Locating base image...")
	baseImage, err := hyperv.FindLatestBaseImage(cfg.WindowsBaseImagePath, cfg.WindowsBaseImagePattern)
	if err != nil {
		return err
	}

	fmt.Printf("       Base image: %s\n", baseImage)

	fmt.Println("[3/9] Creating differencing VHDX...")
	vhdxPath := computerName + ".vhdx"
	if err := hyperv.CreateDifferencingVHDX(baseImage, vhdxPath); err != nil {
		return err
	}

	fmt.Println("[4/9] Injecting files into VHDX...")
	if err := disk.InjectWindowsFiles(
		vhdxPath,
		computerName,
		password,
		*cfg,
	); err != nil {
		return fmt.Errorf("file injection failed: %w", err)
	}

	fmt.Println("[5/9] Creating virtual machine...")
	vmCfg := hyperv.VMConfig{
		Name:           computerName,
		VHDXPath:       vhdxPath,
		VirtualSwitch:  cfg.VirtualSwitch,
		MemoryBytes:    cfg.MemoryBytes,
		ProcessorCount: cfg.ProcessorCount,
		Generation:     2,
		SecureBoot:     true,
	}

	if err := hyperv.CreateVM(vmCfg); err != nil {
		return err
	}

	fmt.Println("[6/9] Configuring VM...")
	if err := hyperv.SetProcessorCount(computerName, cfg.ProcessorCount); err != nil {
		return err
	}

	if err := hyperv.SetSecureBootTemplate(computerName, "MicrosoftWindows"); err != nil {
		return err
	}

	if err := hyperv.DisableAutomaticCheckpoints(computerName); err != nil {
		return err
	}

	if err := hyperv.EnableCheckpoints(computerName); err != nil {
		return err
	}

	fmt.Println("[7/9] Configuring dynamic memory...")
	minMem := cfg.MemoryBytes / 4
	maxMem := cfg.MemoryBytes
	if err := hyperv.SetDynamicMemory(computerName, cfg.MemoryBytes, minMem, maxMem); err != nil {
		return err
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

	fmt.Printf("\n✓ Windows VM %q provisioned successfully.\n", computerName)

	return nil
}
