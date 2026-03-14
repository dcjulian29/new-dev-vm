// Package disk provides functions to inject files to a mountable VHDX file
package disk

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

	"github.com/dcjulian29/go-toolbox/filesystem"
	"github.com/dcjulian29/new-dev-vm/internal/config"
	"github.com/dcjulian29/new-dev-vm/internal/hyperv"
	"github.com/dcjulian29/new-dev-vm/internal/util"
)

var driveLetter string

// InjectWindowsFiles mounts the VHDX, injects the unattend XML, startup
// script, start layout, then dismounts.  computerName and password are used
// to populate the unattend template.
func InjectWindowsFiles(vhdxPath, computerName, password string, cfg config.Config) error {

	path, err := hyperv.MountVHDX(vhdxPath)
	if err != nil {
		return fmt.Errorf("mounting VHDX for injection: %w", err)
	}

	defer func() {
		if err := hyperv.DismountVHDX(vhdxPath); err != nil {
			fmt.Printf("Warning: failed to dismount %s: %v\n", vhdxPath, err)
		}
	}()

	driveLetter = path + `:\`

	if err := unattendFile(computerName, password, &cfg); err != nil {
		return err
	}

	if err := startScript(&cfg); err != nil {
		return err
	}

	if err := startCommand(&cfg); err != nil {
		return err
	}

	if err := startLayout(&cfg); err != nil {
		return err
	}

	return syncConfig(computerName, &cfg)
}

func startCommand(cfg *config.Config) error {
	root := filepath.Join(driveLetter, "Windows", "Setup", "Scripts")
	path := filepath.Join(root, "SetupComplete.cmd")
	content := "%WINDIR%\\System32\\WindowsPowerShell\\v1.0\\powershell.exe "
	content = content + "-NoProfile -NonInteractive -ExecutionPolicy Bypass -NoLogo -Command "
	content = content + root + "\\" + cfg.WindowsStartScript

	fmt.Println("  [inject] start command →", path)

	return filesystem.EnsureFileExist(path, []byte(content))
}

func startLayout(cfg *config.Config) error {
	if cfg.WindowsStartLayout != "" && filesystem.FileExists(cfg.WindowsStartLayout) {
		dest := filepath.Join(driveLetter, "Users", "Default", "AppData", "Local",
			"Microsoft", "Windows", "Shell", filepath.Base(cfg.WindowsStartLayout))

		if err := filesystem.CopyFile(cfg.WindowsStartLayout, dest); err != nil {
			return fmt.Errorf("injecting start layout: %w", err)
		}

		fmt.Println("  [inject] start layout →", dest)
	}

	return nil
}

func startScript(cfg *config.Config) error {
	if cfg.WindowsStartScript != "" && filesystem.FileExists(cfg.WindowsStartScript) {
		raw, err := os.ReadFile(cfg.WindowsStartScript)
		if err != nil {
			return fmt.Errorf("reading start script: %w", err)
		}

		content := strings.ReplaceAll(string(raw), "{{INSTALLPACKAGE}}", cfg.WindowsInstallPackage)

		dest := filepath.Join(driveLetter, "Windows", "Setup", "Scripts", filepath.Base(cfg.WindowsStartScript))

		if err := filesystem.EnsureDirectoryExist(filepath.Dir(dest)); err != nil {
			return fmt.Errorf("creating Scripts dir: %w", err)
		}

		if err := os.WriteFile(dest, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing startup script: %w", err)
		}

		fmt.Println("  [inject] startup script →", dest)
	}

	return nil
}

func syncConfig(computerName string, cfg *config.Config) error {
	if cfg.WindowsSyncBasePath != "" && filesystem.FileExists(cfg.WindowsStartScript) {
		files := []string{
			"config.xml",
			"key.pem",
			"cert.pem",
		}

		for _, file := range files {
			src := filepath.Join(cfg.WindowsSyncBasePath, computerName, file)
			dst := filepath.Join(driveLetter, "Windows", "Setup", "Scripts", file)

			if filesystem.FileExists(src) {
				if err := filesystem.CopyFile(src, dst); err != nil {
					return fmt.Errorf("injecting sync file '%s': %w", file, err)
				}
			}
		}
	}

	return nil
}

func unattendFile(computerName, password string, cfg *config.Config) error {
	if cfg.WindowsUnattendTemplate != "" && filesystem.FileExists(cfg.WindowsUnattendTemplate) {
		raw, err := os.ReadFile(cfg.WindowsUnattendTemplate)
		if err != nil {
			return fmt.Errorf("reading unattend template: %w", err)
		}

		content := string(raw)
		content = strings.ReplaceAll(content, "{{COMPUTERNAME}}", util.XMLEscape(computerName))
		content = strings.ReplaceAll(content, "{{PASSWORD}}", util.XMLEscape(password))
		content = strings.ReplaceAll(content, "{{USER}}", util.XMLEscape(cfg.WindowsUser))

		dest := filepath.Join(driveLetter, "unattend.xml")

		if err := os.WriteFile(dest, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing unattend.xml: %w", err)
		}

		fmt.Println("  [inject] unattend.xml →", dest)
	}

	return nil
}
