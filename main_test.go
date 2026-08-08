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

package main

import (
	"errors"
	"strings"
	"testing"
)

func TestParseModeSelectsMode(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"no arguments defaults to an empty mode", nil, ""},
		{"empty arguments", []string{}, ""},
		{"windows", []string{"--windows"}, "windows"},
		{"ubuntu", []string{"--ubuntu"}, "ubuntu"},
		{"debian", []string{"--debian"}, "debian"},
		{"config", []string{"--config"}, "config"},
		{"single dash", []string{"-ubuntu"}, "ubuntu"},
		{"extra dashes", []string{"---ubuntu"}, "ubuntu"},
		{"no dashes", []string{"ubuntu"}, "ubuntu"},
		{"upper case", []string{"--UBUNTU"}, "ubuntu"},
		{"mixed case", []string{"--UbUnTu"}, "ubuntu"},
		{"repeated option is not a conflict", []string{"--config", "--config"}, "config"},
		{"repeated in different case", []string{"--ubuntu", "--UBUNTU"}, "ubuntu"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseMode(c.args)
			if err != nil {
				t.Fatalf("parseMode(%q) returned an unexpected error: %v", c.args, err)
			}

			if got != c.want {
				t.Errorf("parseMode(%q) = %q, want %q", c.args, got, c.want)
			}
		})
	}
}

func TestParseModeReportsHelp(t *testing.T) {
	cases := [][]string{
		{"--help"},
		{"-h"},
		{"--HELP"},
		{"--ubuntu", "--help"},
		{"--help", "--bogus"},
	}

	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			mode, err := parseMode(args)

			if !errors.Is(err, errHelpRequested) {
				t.Fatalf("parseMode(%q) error = %v, want errHelpRequested", args, err)
			}

			if mode != "" {
				t.Errorf("parseMode(%q) = %q, want an empty mode", args, mode)
			}
		})
	}
}

// The reported option must be the one that was not recognised.
func TestParseModeNamesTheUnknownOption(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"only argument", []string{"--bogus"}, "--bogus"},
		{"after a valid option", []string{"--ubuntu", "--bogus"}, "--bogus"},
		{"after two valid options", []string{"--windows", "--windows", "--nope"}, "--nope"},
		{"bare dash", []string{"-"}, "-"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := parseMode(c.args)
			if err == nil {
				t.Fatalf("parseMode(%q) returned no error", c.args)
			}

			if errors.Is(err, errHelpRequested) {
				t.Fatalf("parseMode(%q) reported help instead of an unknown option", c.args)
			}

			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("parseMode(%q) error = %q, want it to name %q", c.args, err, c.want)
			}
		})
	}
}

// Selecting two different modes is ambiguous and must not silently pick one.
func TestParseModeRejectsConflictingOptions(t *testing.T) {
	cases := []struct {
		name  string
		args  []string
		first string
		last  string
	}{
		{"windows then ubuntu", []string{"--windows", "--ubuntu"}, "windows", "--ubuntu"},
		{"ubuntu then debian", []string{"--ubuntu", "--debian"}, "ubuntu", "--debian"},
		{"mode then config", []string{"--ubuntu", "--config"}, "ubuntu", "--config"},
		{"config then mode", []string{"--config", "--windows"}, "config", "--windows"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mode, err := parseMode(c.args)
			if err == nil {
				t.Fatalf("parseMode(%q) = %q, want a conflict error", c.args, mode)
			}

			for _, want := range []string{c.first, c.last} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("parseMode(%q) error = %q, want it to name %q", c.args, err, want)
				}
			}
		})
	}
}

// Every mode the usage text advertises must be accepted.
func TestParseModeAcceptsEveryDocumentedOption(t *testing.T) {
	for _, mode := range []string{"windows", "ubuntu", "debian", "config"} {
		if !strings.Contains(usage, "--"+mode) {
			t.Errorf("usage text does not document --%s", mode)
		}

		got, err := parseMode([]string{"--" + mode})
		if err != nil {
			t.Errorf("parseMode(--%s) returned an unexpected error: %v", mode, err)
		}

		if got != mode {
			t.Errorf("parseMode(--%s) = %q, want %q", mode, got, mode)
		}
	}
}
