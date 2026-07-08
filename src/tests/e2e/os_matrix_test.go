//go:build integration_docker

package e2e_test

import (
	"os"
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
	matrix := []struct{ distro, version string }{
		{"ubuntu", "24.04"},
		{"ubuntu", "22.04"},
		{"ubuntu", "20.04"},
		{"centos", "9"},
		{"centos", "8"},
		{"rocky", "9"},
		{"rocky", "8"},
		{"debian", "12"},
		{"debian", "11"},
	}

	for _, m := range matrix {
		m := m
		t.Run(m.distro+"-"+m.version, func(t *testing.T) {
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
