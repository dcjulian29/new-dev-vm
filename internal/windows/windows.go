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

// Package windows provides functions to provision Windows VMs
package windows

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

const (
	// nameSuffix is appended to the host name to name the VM.
	nameSuffix = "DEV"
	// maxHostnameLength leaves room for nameSuffix within the 15 character
	// limit Windows imposes on a computer name.
	maxHostnameLength = 15 - len(nameSuffix)
)

// ProvisionWindows creates a Windows development VM
func ProvisionWindows(cfg *config.Config) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	computerName, shortened := computerNameFor(hostname)
	if shortened {
		fmt.Printf("Hostname %q is too long; using %q as the computer name.\n",
			strings.ToUpper(hostname), computerName)
	}

	prompt := fmt.Sprintf("Enter password for '%s' on %s: ", cfg.WindowsUser, computerName)
	password, err := util.PromptPassword(prompt)
	if err != nil {
		return fmt.Errorf("prompting for password: %w", err)
	}

	fmt.Printf("\n[Windows] Provisioning VM: %s\n", computerName)

	stepOut("Checking Hyper-V...")
	if err := hyperv.EnsureEnabled(); err != nil {
		return err
	}

	stepOut("Locating base image...")
	baseImage, err := hypervhost.FindLatestBaseDisk(cfg.WindowsBaseImagePath, cfg.WindowsBaseImagePattern)
	if err != nil {
		return err
	}

	fmt.Printf("  Base image: %s\n", baseImage)

	stepOut("Creating differencing VHDX...")

	directory, err := hypervhost.VMStoragePath()
	if err != nil {
		return err
	}

	vhdxPath := filepath.Join(directory, computerName+".vhdx")

	if filesystem.FileExist(vhdxPath) {
		// A leftover disk without a VM is stale and can simply be removed.
		if hypervmachine.Exist(computerName) {
			state, err := hypervmachine.State(computerName)
			if err != nil {
				return err
			}

			switch state {
			case "Running":
				return fmt.Errorf("cannot create '%s' because '%s' is running", vhdxPath, computerName)
			case "Saved":
				return fmt.Errorf("cannot create '%s' because '%s' is saved", vhdxPath, computerName)
			}
		}

		if err := filesystem.RemoveFile(vhdxPath); err != nil {
			return err
		}
	}

	if err := hypervdisk.CreateDifferencing(baseImage, vhdxPath); err != nil {
		return err
	}

	fmt.Printf("  VHDX: %s\n", vhdxPath)

	stepOut("Injecting files into VHDX...")

	if err := injectFiles(vhdxPath, computerName, password, cfg); err != nil {
		return err
	}

	stepOut("Creating virtual machine...")

	if hypervmachine.Exist(computerName) {
		if err := hypervmachine.Remove(computerName); err != nil {
			return err
		}
	}

	vmCfg := hypervmachine.Config{
		Name:               computerName,
		VHDXPath:           vhdxPath,
		VirtualSwitch:      cfg.VirtualSwitch,
		MemoryBytes:        cfg.MemoryBytes,
		MaximumMemoryBytes: cfg.MaximumMemoryBytes,
		ProcessorCount:     cfg.ProcessorCount,
		Generation:         2,
		SecureBoot:         true,
	}

	if err := hypervmachine.Create(&vmCfg); err != nil {
		return err
	}

	stepOut("Configuring VM...")
	if err := hypervmachine.SetSecureBootTemplate(computerName, "MicrosoftWindows"); err != nil {
		return err
	}

	if err := hypervmachine.DisableAutomaticCheckpoints(computerName); err != nil {
		return err
	}

	if err := hypervmachine.EnableStandardCheckpoints(computerName); err != nil {
		return err
	}

	stepOut("Starting VM...")
	if err := hypervmachine.Start(computerName); err != nil {
		return err
	}

	stepOut("Opening console...")
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

// computerNameFor builds the VM computer name from the host name, shortening
// the host name when needed to stay within the Windows computer name limit.
// The second return value reports whether the host name had to be shortened.
func computerNameFor(hostname string) (string, bool) {
	name := strings.ToUpper(hostname)
	if len(name) <= maxHostnameLength {
		return name + nameSuffix, false
	}

	return name[:maxHostnameLength] + nameSuffix, true
}

// injectFiles mounts the VHDX, injects the setup files, and always dismounts
// the disk before returning so a failed injection cannot leave it attached.
func injectFiles(vhdxPath, computerName, password string, cfg *config.Config) (err error) {
	drive, err := hypervdisk.Mount(vhdxPath)
	if err != nil {
		return fmt.Errorf("failed vhdx mount: %w", err)
	}

	defer func() {
		if dismountErr := hypervdisk.Dismount(vhdxPath); dismountErr != nil && err == nil {
			err = fmt.Errorf("failed vhdx dismount: %w", dismountErr)
		}
	}()

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

	return nil
}

// syncConfig injects the Syncthing files for the VM. Syncthing is optional and
// is skipped when no base path is configured, but a configured base path must
// hold every file for the VM.
func syncConfig(drive, computerName string, cfg *config.Config) error {
	if cfg.WindowsSyncBasePath == "" {
		return nil
	}

	files := []string{
		"config.xml",
		"key.pem",
		"cert.pem",
	}

	source := filepath.Join(cfg.WindowsSyncBasePath, computerName)

	for _, file := range files {
		src := filepath.Join(source, file)
		dst := filepath.Join(drive, "Windows", "Setup", "Scripts", file)

		if !filesystem.FileExist(src) {
			return fmt.Errorf("sync file '%s' not found in '%s'", file, source)
		}

		if err := filesystem.CopyFile(src, dst); err != nil {
			return fmt.Errorf("injecting sync file '%s': %w", file, err)
		}
	}

	fmt.Printf("  Syncthing files injected from %s\n", source)

	return nil
}
