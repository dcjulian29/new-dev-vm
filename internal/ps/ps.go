// Package ps provides functions to execute Powershell commands and scripts
package ps

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
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// RunPowershell executes a PowerShell command and streams stdout/stderr to the
// caller's terminal.  An error is returned if the exit code is non-zero.
func RunPowershell(command string) error {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive",
		"-ExecutionPolicy", "Bypass", "-Command", command)
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run()
}

// RunPowershellOutput executes a PowerShell command and returns the combined
// stdout as a trimmed string.
func RunPowershellOutput(command string) (string, error) {
	var out, errBuf bytes.Buffer

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive",
		"-ExecutionPolicy", "Bypass", "-Command", command)
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w\n%s", err, errBuf.String())
	}

	return strings.TrimSpace(out.String()), nil
}

// Escape wraps a string value for safe inclusion inside a PowerShell
// double-quoted string by escaping backtick, dollar, and double-quote chars.
func Escape(s string) string {
	s = strings.ReplaceAll(s, "`", "``")
	s = strings.ReplaceAll(s, "$", "`$")
	s = strings.ReplaceAll(s, `"`, "`\"")

	return s
}
