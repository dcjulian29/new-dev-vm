// Package util provides various utility functions.
package util

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
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

// XMLEscape escapes the five predefined XML entities in s.
func XMLEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// PromptPassword prints prompt, disables console echo via
// windows.GetConsoleMode / windows.SetConsoleMode (no unsafe.Pointer needed),
// reads one line, then restores the original console mode.
//
// If stdin is not a real console (e.g. piped input) the echo-disable step is
// silently skipped and the line is read as plain text.
func PromptPassword(prompt string) (string, error) {
	fmt.Print(prompt)

	stdin := windows.Handle(os.Stdin.Fd())

	var oldMode uint32
	consoleErr := windows.GetConsoleMode(stdin, &oldMode)

	if consoleErr == nil {
		noEcho := oldMode &^ windows.ENABLE_ECHO_INPUT
		if err := windows.SetConsoleMode(stdin, noEcho); err != nil {
			return "", fmt.Errorf("SetConsoleMode (disable echo): %w", err)
		}

		defer windows.SetConsoleMode(stdin, oldMode) //nolint:errcheck
	}

	reader := bufio.NewReader(os.Stdin)
	pw, err := reader.ReadString('\n')
	fmt.Println() // move to next line after hidden input
	if err != nil {
		return "", fmt.Errorf("reading password: %w", err)
	}
	return strings.TrimRight(pw, "\r\n"), nil
}
