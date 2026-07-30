//go:build integration_docker

package e2e_test

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestDockerOSMatrix reproduces test.sh's runostest for the Docker backend:
// deploy a single-node cluster on each supported distro/version and tear it
// down. It is heavy (one image build + start per entry) so it is opt-in via
// AEROLAB_E2E_OS_MATRIX and, like the lifecycle test, needs a features file for
// Enterprise images.
func TestDockerOSMatrix(t *testing.T) {
	if os.Getenv("AEROLAB_E2E_OS_MATRIX") == "" {
		t.Skip("set AEROLAB_E2E_OS_MATRIX=1 to run the multi-distro OS matrix test")
	}
	if os.Getenv("AEROLAB_FEATURES_FILE") == "" {
		t.Skip("set AEROLAB_FEATURES_FILE to run the OS matrix test (Enterprise images require a feature key)")
	}
	c := newDockerCLI(t)
	asVer := getenvDefault("AEROLAB_E2E_ASVER", "8.*")

	// The Docker tier of runostest (excludes amazon/arm which are AWS-only).
	// minAsVer, where set, is the first Aerospike server release that ships
	// packages for that distro. Those entries are skipped unless
	// AEROLAB_E2E_ASVER names a concrete version that satisfies it, so a
	// wildcard selector cannot fail the matrix on an OS with no build yet.
	matrix := []struct{ distro, version, minAsVer string }{
		{"ubuntu", "26.04", "8.1.3"},
		{"ubuntu", "24.04", ""},
		{"ubuntu", "22.04", ""},
		{"ubuntu", "20.04", ""},
		{"centos", "10", ""},
		{"centos", "9", ""},
		{"centos", "8", ""},
		{"rocky", "10", ""},
		{"rocky", "9", ""},
		{"rocky", "8", ""},
		{"debian", "13", ""},
		{"debian", "12", ""},
		{"debian", "11", ""},
	}

	for _, m := range matrix {
		m := m
		t.Run(m.distro+"-"+m.version, func(t *testing.T) {
			if m.minAsVer != "" && !asVerAtLeast(asVer, m.minAsVer) {
				t.Skipf("%s %s needs aerospike server %s or later; set AEROLAB_E2E_ASVER to run it (current: %s)", m.distro, m.version, m.minAsVer, asVer)
			}
			name := uniqueClusterName()
			t.Cleanup(func() {
				_, _ = c.runErr("cluster", "destroy", "-n", name, "--force")
			})
			c.run("cluster", "create", "-n", name, "-c", "1",
				"-d", m.distro, "-i", m.version, "-v", asVer)
			c.run("cluster", "destroy", "-n", name, "--force")
		})
	}
}

// asVerAtLeast reports whether the Aerospike version selector definitely
// resolves to min or newer. A selector containing a wildcard could resolve to
// anything in its range, so it is reported as not satisfying the minimum.
func asVerAtLeast(selector string, min string) bool {
	if strings.ContainsAny(selector, "*?") {
		return false
	}
	sel := strings.Split(selector, ".")
	want := strings.Split(min, ".")
	for i := range want {
		if i >= len(sel) {
			return false
		}
		s, err := strconv.Atoi(sel[i])
		if err != nil {
			return false
		}
		w, err := strconv.Atoi(want[i])
		if err != nil {
			return false
		}
		if s != w {
			return s > w
		}
	}
	return true
}
