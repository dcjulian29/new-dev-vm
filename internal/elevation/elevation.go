// Package elevation provides functions to check if elevated and restart in an elevated state
package elevation

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
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// IsElevated returns true when the current process holds administrator privileges.
func IsElevated() bool {
	var token windows.Token

	if err := windows.OpenProcessToken(
		windows.CurrentProcess(),
		windows.TOKEN_QUERY,
		&token,
	); err != nil {
		return false
	}

	defer token.Close() //nolint

	return token.IsElevated()
}

// RelaunchElevated re-executes the current process with administrative privileges
// which triggers the Windows UAC prompt.
func RelaunchElevated() error {
	verb, err := syscall.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	exePtr, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}

	args := ""
	if len(os.Args) > 1 {
		for _, a := range os.Args[1:] {
			args += " " + a
		}
	}
	var argsPtr *uint16

	if args != "" {
		argsPtr, err = syscall.UTF16PtrFromString(args)
		if err != nil {
			return err
		}
	}

	shell32 := syscall.NewLazyDLL("shell32.dll")
	shellExec := shell32.NewProc("ShellExecuteW")

	r, _, _ := shellExec.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(exePtr)),
		uintptr(unsafe.Pointer(argsPtr)),
		0,
		1, // SW_SHOWNORMAL
	)

	if r <= 32 {
		return syscall.EINVAL
	}

	return nil
}
