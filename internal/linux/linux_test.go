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

package linux

import (
	"strings"
	"testing"
)

func TestComputerNameFor(t *testing.T) {
	cases := []struct {
		name     string
		hostname string
		distro   string
		want     string
	}{
		{"ubuntu", "bender", "Ubuntu", "BENDERDEVLU"},
		{"debian", "bender", "Debian", "BENDERDEVLD"},
		{"already upper case", "BENDER", "Ubuntu", "BENDERDEVLU"},
		{"lower case distro", "bender", "ubuntu", "BENDERDEVLU"},
		{"long host name is not shortened", "averylonghostname", "Debian", "AVERYLONGHOSTNAMEDEVLD"},
		{"empty host name", "", "Ubuntu", "DEVLU"},
		{"empty distro", "bender", "", "BENDERDEVL"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := computerNameFor(c.hostname, c.distro); got != c.want {
				t.Errorf("computerNameFor(%q, %q) = %q, want %q",
					c.hostname, c.distro, got, c.want)
			}
		})
	}
}

// Each distribution must get its own VM name so they cannot overwrite one another.
func TestComputerNameForDistinguishesDistros(t *testing.T) {
	ubuntu := computerNameFor("bender", "Ubuntu")
	debian := computerNameFor("bender", "Debian")

	if ubuntu == debian {
		t.Errorf("Ubuntu and Debian share the VM name %q", ubuntu)
	}
}

// The name is the host name uppercased, the suffix, then the distro initial.
func TestComputerNameForMatchesExpectedShape(t *testing.T) {
	for _, distro := range []string{"Ubuntu", "Debian"} {
		got := computerNameFor("bender", distro)

		want := strings.ToUpper("bender") + nameSuffix + strings.ToUpper(distro[:1])
		if got != want {
			t.Errorf("computerNameFor(%q, %q) = %q, want %q", "bender", distro, got, want)
		}

		if !strings.HasPrefix(got, "BENDER") {
			t.Errorf("computerNameFor(%q, %q) = %q, want it to start with the host name",
				"bender", distro, got)
		}

		if !strings.Contains(got, nameSuffix) {
			t.Errorf("computerNameFor(%q, %q) = %q, want it to contain %q",
				"bender", distro, got, nameSuffix)
		}
	}
}

// The parameters each entry point passes decide which ISO settings are used.
func TestProvisionParamsUseTheirOwnSettings(t *testing.T) {
	cases := []struct {
		distro string
		want   string
	}{
		{"Ubuntu", "U"},
		{"Debian", "D"},
	}

	for _, c := range cases {
		t.Run(c.distro, func(t *testing.T) {
			name := computerNameFor("bender", c.distro)

			if !strings.HasSuffix(name, c.want) {
				t.Errorf("computerNameFor(%q, %q) = %q, want it to end with %q",
					"bender", c.distro, name, c.want)
			}
		})
	}
}
