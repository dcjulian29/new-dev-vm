# new-dev-vm

A CLI tool for provisioning Hyper-V development virtual machines.
Supports Windows, Ubuntu, and Debian guests.

[![Version](https://img.shields.io/github/v/release/dcjulian29/new-dev-vm)](https://github.com/dcjulian29/new-dev-vm/releases)
[![GitHub Issues](https://img.shields.io/github/issues-raw/dcjulian29/new-dev-vm.svg)](https://github.com/dcjulian29/new-dev-vm/issues)
[![Build](https://github.com/dcjulian29/new-dev-vm/actions/workflows/build.yml/badge.svg)](https://github.com/dcjulian29/new-dev-vm/actions/workflows/build.yml)

---

## Requirements

- Windows 10/11 or Windows Server 2019+ with Hyper-V enabled
- PowerShell 5.1+ (pre-installed on all supported Windows versions)
- Administrator privileges (UAC prompt fires automatically)

## Build

```powershell
go mod tidy
go mod verify
go vet
go build -o new-dev-vm.exe .
```

## Configuration

Copy `new-dev-vm.example.yml` to `%USERPROFILE%\.config\new-dev-vm.yml` and edit:

```powershell
Copy-Item "new-dev-vm.example.yml" "$env:USERPROFILE\.config\new-dev-vm.yml"
notepad "$env:USERPROFILE\.config\new-dev-vm.yml"
```

Key settings:

| Key | Purpose |
|-----|---------|
| `virtualSwitch` | Hyper-V switch name |
| `processorCount` | Number of processors to give the VM |
| `memoryBytes` | Memory size of the VM in bytes |
| `windowsBaseImagePath` | Directory containing base VHDX files |
| `windowsBaseImagePattern` | Pattern to used to find the Windows base VHDX |
| `windowsStartLayout` | Path to start menu layout XML |
| `windowsStartScript` | Path to startup PowerShell script to injected |
| `windowsInstallPackage` | Name of the installation package |
| `windowsSyncBasePath` | Directory that contains the Syncthing files |
| `windowsUnattendTemplate` | Path to unattend XML template |
| `windowsUser` | Name of the Windows user name for the VM |
| `linuxDisableSecureBoot` | `true` to disable Secure Boot on Linux VMs |
| `linuxDiskSizeBytes` | size of the VMDK disk in bytes |
| `ubuntuIsoPattern` | Pattern used to locate Ubuntu ISO |
| `ubuntuIsoSearchPath` | Directory to search for Ubuntu ISO |
| `debianIsoPattern` | Pattern used to locate Debian ISO |
| `debianIsoSearchPath` | Directory to search for Debian ISO |

## Usage

```
new-dev-vm [option]

Options:
  --windows,    Provision a Windows development VM (default)
  --ubuntu,     Provision an Ubuntu development VM
  --debian,     Provision a Debian development VM
  --config,     Print the active configuration
  --help, -h    Show this help message
```

Both `--flag` and `-flag` prefixes are accepted for every option.

## Examples

```powershell
# Windows VM (default — both forms work)
.\new-dev-vm.exe
.\new-dev-vm.exe --windows

# Ubuntu VM
.\new-dev-vm.exe --ubuntu

# Debian VM
.\new-dev-vm.exe --debian

# Verify config before running
.\new-dev-vm.exe --config

# Show help
.\new-dev-vm.exe --help
.\new-dev-vm.exe -h
```

## Project structure

```
new-dev-vm/
├── main.go                        # CLI entrypoint
├── go.mod / go.sum
├── new-dev-vm.example.yml
├── README.md
└── internal/
    ├── config/config.go           # YAML loader + --config printer
    ├── elevation/elevation.go     # Admin check + UAC re-launch
    ├── hyperv/
    │   ├── host.go                # Host checks, base image search
    │   ├── vhd.go                 # VHDX create / mount / dismount
    │   └── vm.go                  # VM create / configure / start
    ├── windows/windows.go         # Windows provisioning workflow
    ├── linux/linux.go             # Ubuntu & Debian provisioning workflow
    ├── disk/inject.go             # Unattend / script / layout injection
    ├── ps/ps.go                   # PowerShell subprocess helper
    └── util/util.go               # Shared helpers
```
