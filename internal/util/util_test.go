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

package util

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// promptWith runs PromptPassword against input supplied on standard input and
// returns the password, whatever was printed, and the resulting error. A file
// stands in for the console, which also exercises the non-console path.
func promptWith(t *testing.T, input string) (string, string, error) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "stdin")
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatalf("writing stdin file: %v", err)
	}

	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		t.Fatalf("opening stdin file: %v", err)
	}

	defer file.Close()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}

	originalIn, originalOut := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = file, writer

	defer func() {
		os.Stdin, os.Stdout = originalIn, originalOut
	}()

	done := make(chan string)

	go func() {
		var builder strings.Builder
		_, _ = io.Copy(&builder, reader)
		done <- builder.String()
	}()

	password, promptErr := PromptPassword("Enter password: ")

	if err := writer.Close(); err != nil {
		t.Fatalf("closing pipe: %v", err)
	}

	output := <-done

	if err := reader.Close(); err != nil {
		t.Fatalf("closing reader: %v", err)
	}

	return password, output, promptErr
}

func TestPromptPasswordReadsLine(t *testing.T) {
	password, _, err := promptWith(t, "secret\n")
	if err != nil {
		t.Fatalf("PromptPassword() returned an unexpected error: %v", err)
	}

	if password != "secret" {
		t.Errorf("PromptPassword() = %q, want %q", password, "secret")
	}
}

// Windows console input arrives with a carriage return that must not survive.
func TestPromptPasswordTrimsCarriageReturn(t *testing.T) {
	password, _, err := promptWith(t, "secret\r\n")
	if err != nil {
		t.Fatalf("PromptPassword() returned an unexpected error: %v", err)
	}

	if password != "secret" {
		t.Errorf("PromptPassword() = %q, want %q", password, "secret")
	}
}

func TestPromptPasswordKeepsInteriorCharacters(t *testing.T) {
	const want = "p@ss w0rd!#$%^&*()"

	password, _, err := promptWith(t, want+"\r\n")
	if err != nil {
		t.Fatalf("PromptPassword() returned an unexpected error: %v", err)
	}

	if password != want {
		t.Errorf("PromptPassword() = %q, want %q", password, want)
	}
}

// Only line endings are trimmed, so a deliberate trailing space is preserved.
func TestPromptPasswordKeepsTrailingSpace(t *testing.T) {
	password, _, err := promptWith(t, "secret \r\n")
	if err != nil {
		t.Fatalf("PromptPassword() returned an unexpected error: %v", err)
	}

	if password != "secret " {
		t.Errorf("PromptPassword() = %q, want %q", password, "secret ")
	}
}

func TestPromptPasswordAcceptsEmptyInput(t *testing.T) {
	password, _, err := promptWith(t, "\r\n")
	if err != nil {
		t.Fatalf("PromptPassword() returned an unexpected error: %v", err)
	}

	if password != "" {
		t.Errorf("PromptPassword() = %q, want an empty string", password)
	}
}

// Input that ends before a newline is reported rather than silently accepted.
func TestPromptPasswordReportsUnterminatedInput(t *testing.T) {
	password, _, err := promptWith(t, "secret")
	if err == nil {
		t.Fatal("PromptPassword() returned no error for input without a line ending")
	}

	if password != "" {
		t.Errorf("PromptPassword() = %q, want an empty string on error", password)
	}
}

func TestPromptPasswordWritesPrompt(t *testing.T) {
	_, output, err := promptWith(t, "secret\r\n")
	if err != nil {
		t.Fatalf("PromptPassword() returned an unexpected error: %v", err)
	}

	if !strings.Contains(output, "Enter password: ") {
		t.Errorf("prompt was not printed, got %q", output)
	}

	// The password must never be echoed back to the terminal.
	if strings.Contains(output, "secret") {
		t.Errorf("output echoed the password, got %q", output)
	}
}
