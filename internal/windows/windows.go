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
	"path/filepath"
	"strings"
	"time"

	"github.com/dcjulian29/go-toolbox/filesystem"
	"github.com/dcjulian29/go-toolbox/hyperv"
	"github.com/dcjulian29/go-toolbox/hypervdisk"
	"github.com/dcjulian29/go-toolbox/hypervhost"
	"github.com/dcjulian29/go-toolbox/hypervmachine"
	"github.com/dcjulian29/go-toolbox/textformat"
	"github.com/dcjulian29/new-dev-vm/internal/config"
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

	stepOut("[1/9] Checking Hyper-V...")
	if err := hyperv.Enabled(); err != nil {
		return err
	}

	stepOut("[2/9] Locating base image...")
	baseImage, err := hypervhost.FindLatestBaseDisk(cfg.WindowsBaseImagePath, cfg.WindowsBaseImagePattern)
	if err != nil {
		return err
	}

	fmt.Printf("       Base image: %s\n", baseImage)

	stepOut("[3/9] Creating differencing VHDX...")

	directory, err := hypervhost.VMStoragePath()
	if err != nil {
		return err
	}

	vhdxPath := filepath.Join(directory, computerName+".vhdx")

	if filesystem.FileExists(vhdxPath) {
		state, err := hypervmachine.State(computerName)
		if err != nil {
			return err
		}
		switch state {
		case "Running":
			return fmt.Errorf("cannot create '%s' because '%s' is running", vhdxPath, computerName)
		case "Saved":
			return fmt.Errorf("cannot create '%s' because '%s' is saved", vhdxPath, computerName)
		default:
			if err := filesystem.RemoveFile(vhdxPath); err != nil {
				return err
			}
		}
	}

	if err := hypervdisk.CreateDifferencing(baseImage, vhdxPath); err != nil {
		return err
	}

	fmt.Printf("      VHDX: %s\n", vhdxPath)

	stepOut("[4/9] Injecting files into VHDX...")

	drive, err := hypervdisk.Mount(vhdxPath)
	if err != nil {
		return fmt.Errorf("failed vhdx mount: %w", err)
	}

	injectCfg := hypervdisk.InjectConfig{
		ComputerName:     computerName,
		InstallPackage:   cfg.WindowsInstallPackage,
		MountedDrive:     drive,
		StartScript:      cfg.WindowsStartScript,
		UnattendTemplate: cfg.WindowsUnattendTemplate,
		UserName:         cfg.WindowsUser,
		UserPassword:     password,
	}

	if err := hypervdisk.InjectStartCommand(&injectCfg); err != nil {
		return fmt.Errorf("start command injection failed: %w", err)
	}

	if err := hypervdisk.InjectStartScript(&injectCfg); err != nil {
		return fmt.Errorf("start script injection failed: %w", err)
	}

	if err := hypervdisk.InjectUnattendFile(&injectCfg); err != nil {
		return fmt.Errorf("unattend file injection failed: %w", err)
	}

	if err := syncConfig(drive, computerName, cfg); err != nil {
		return fmt.Errorf("sync config injection failed: %w", err)
	}

	if err := hypervdisk.Dismount(vhdxPath); err != nil {
		return fmt.Errorf("failed vhdx dismount: %w", err)
	}

	stepOut("[5/9] Creating virtual machine...")

	if hypervmachine.Exists(computerName) {
		if err := hypervmachine.Remove(computerName); err != nil {
			return err
		}
	}

	vmCfg := hypervmachine.Config{
		Name:           computerName,
		VHDXPath:       vhdxPath,
		VirtualSwitch:  cfg.VirtualSwitch,
		MemoryBytes:    cfg.MemoryBytes,
		ProcessorCount: cfg.ProcessorCount,
		Generation:     2,
		SecureBoot:     true,
	}

	if err := hypervmachine.Create(vmCfg); err != nil {
		return err
	}

	stepOut("[6/9] Configuring VM...")
	if err := hypervmachine.SetProcessorCount(computerName, cfg.ProcessorCount); err != nil {
		return err
	}

	if err := hypervmachine.SetSecureBootTemplate(computerName, "MicrosoftWindows"); err != nil {
		return err
	}

	if err := hypervmachine.DisableAutomaticCheckpoints(computerName); err != nil {
		return err
	}

	if err := hypervmachine.EnableCheckpoints(computerName); err != nil {
		return err
	}

	stepOut("[7/9] Configuring dynamic memory...")
	minMem := cfg.MemoryBytes / 4
	maxMem := cfg.MemoryBytes
	if err := hypervmachine.SetDynamicMemory(computerName, cfg.MemoryBytes, minMem, maxMem); err != nil {
		return err
	}

	stepOut("[8/9] Starting VM...")
	if err := hypervmachine.Start(computerName); err != nil {
		return err
	}

	stepOut("[9/9] Opening console...")
	time.Sleep(2 * time.Second)
	if err := hyperv.OpenConsole(computerName); err != nil {
		fmt.Printf("Warning: could not open console: %v\n", err)
	}

	fmt.Printf("\n✓ Windows VM %q provisioned successfully.\n", computerName)

	return nil
}

func stepOut(text string) {
	fmt.Println(textformat.Yellow(text))
}

func syncConfig(drive, computerName string, cfg *config.Config) error {
	if cfg.WindowsSyncBasePath != "" && filesystem.FileExists(cfg.WindowsStartScript) {
		files := []string{
			"config.xml",
			"key.pem",
			"cert.pem",
		}

		for _, file := range files {
			src := filepath.Join(cfg.WindowsSyncBasePath, computerName, file)
			dst := filepath.Join(drive, "Windows", "Setup", "Scripts", file)

			if filesystem.FileExists(src) {
				if err := filesystem.CopyFile(src, dst); err != nil {
					return fmt.Errorf("injecting sync file '%s': %w", file, err)
				}
			}
		}
	}

	return nil
}
