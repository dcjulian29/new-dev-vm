// Command new-dev-vm is a CLI tool for provisioning Hyper-V development virtual machines.
package main

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

	"github.com/dcjulian29/new-dev-vm/internal/config"
	"github.com/dcjulian29/new-dev-vm/internal/elevation"
	"github.com/dcjulian29/new-dev-vm/internal/linux"
	"github.com/dcjulian29/new-dev-vm/internal/windows"
)

const usage = `new-dev-vm — Hyper-V development VM provisioner

Usage:
  new-dev-vm [option]

Options:
  --windows, -windows    Provision a Windows development VM (default)
  --ubuntu,  -ubuntu     Provision an Ubuntu development VM
  --debian,  -debian     Provision a Debian development VM
  --config,  -config     Print the active configuration from ~/.config/new-dev-vm.yml
  --help,    -help, -h   Show this help message

Configuration file: ~/.config/new-dev-vm.yml
  Copy new-dev-vm.example.yml to that path and edit before first use.
`

func main() {
	// Normalise args: strip leading "--" or "-" to compare uniformly.
	args := os.Args[1:]
	var mode string
	for _, a := range args {
		a = strings.TrimLeft(a, "-")
		switch strings.ToLower(a) {
		case "help", "h":
			fmt.Print(usage)
			os.Exit(0)
		case "config":
			mode = "config"
		case "windows":
			mode = "windows"
		case "ubuntu":
			mode = "ubuntu"
		case "debian":
			mode = "debian"
		default:
			fmt.Fprintf(os.Stderr, "Unknown option: %s\nRun 'new-dev-vm --help' for usage.\n", os.Args[len(os.Args)-len(args)])
			os.Exit(1)
		}
	}

	if mode == "" {
		mode = "windows" // default
	}

	// Load config (always needed except --help which already exited).
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// --config: print and exit.
	if mode == "config" {
		config.Print(cfg)
		os.Exit(0)
	}

	// All provisioning modes require elevation.
	if !elevation.IsElevated() {
		fmt.Println("Not running as Administrator — requesting elevation via UAC...")
		if err := elevation.RelaunchElevated(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to relaunch elevated: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Dispatch.
	switch mode {
	case "windows":
		if err := windows.ProvisionWindows(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Windows provisioning failed: %v\n", err)
			os.Exit(1)
		}
	case "ubuntu":
		if err := linux.ProvisionUbuntu(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Ubuntu provisioning failed: %v\n", err)
			os.Exit(1)
		}
	case "debian":
		if err := linux.ProvisionDebian(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Debian provisioning failed: %v\n", err)
			os.Exit(1)
		}
	}
}
