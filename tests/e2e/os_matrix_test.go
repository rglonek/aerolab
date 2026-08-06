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
	parent := newDockerCLI(t)
	asVer := getenvDefault("AEROLAB_E2E_ASVER", "8.*")

	// The Docker tier of runostest (excludes amazon/arm which are AWS-only).
	//
	// minAsVer is the first Aerospike server release that ships packages for
	// that distro; droppedAsVer is the first release that stopped shipping them.
	// Entries outside the range are skipped unless AEROLAB_E2E_ASVER names a
	// concrete version that provably falls inside it, so a wildcard selector
	// cannot fail the matrix on a distro the selected server has no build for.
	matrix := []struct{ distro, version, minAsVer, droppedAsVer string }{
		{"ubuntu", "26.04", "8.1.3", ""},
		{"ubuntu", "24.04", "", ""},
		{"ubuntu", "22.04", "", ""},
		{"ubuntu", "20.04", "", "8.0"},
		{"centos", "10", "", ""},
		{"centos", "9", "", ""},
		{"centos", "8", "", ""},
		{"rocky", "10", "", ""},
		{"rocky", "9", "", ""},
		{"rocky", "8", "", ""},
		{"debian", "13", "", ""},
		{"debian", "12", "", ""},
		{"debian", "11", "", "8.0"},
	}

	for _, m := range matrix {
		m := m
		t.Run(m.distro+"-"+m.version, func(t *testing.T) {
			// Rebind the runner: a Fatalf against the parent's *testing.T would
			// abort the whole matrix instead of just this entry.
			c := parent.withT(t)
			if m.minAsVer != "" && !asVerAtLeast(asVer, m.minAsVer) {
				t.Skipf("%s %s needs aerospike server %s or later; set AEROLAB_E2E_ASVER to run it (current: %s)", m.distro, m.version, m.minAsVer, asVer)
			}
			if m.droppedAsVer != "" && !asVerBelow(asVer, m.droppedAsVer) {
				t.Skipf("%s %s has no aerospike server build from %s onwards; set AEROLAB_E2E_ASVER to an earlier version to run it (current: %s)", m.distro, m.version, m.droppedAsVer, asVer)
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
// resolves to min or newer.
func asVerAtLeast(selector string, min string) bool {
	cmp, known := compareAsVer(selector, min)
	return known && cmp >= 0
}

// asVerBelow reports whether the Aerospike version selector definitely resolves
// to something older than bound.
func asVerBelow(selector string, bound string) bool {
	cmp, known := compareAsVer(selector, bound)
	return known && cmp < 0
}

// compareAsVer compares an Aerospike version selector against a concrete
// version component by component, reporting -1, 0 or 1 like strings.Compare.
//
// The second return value is false when the selector is too vague to decide: a
// wildcard resolves to anything within its range, so once one is reached the
// ordering is only known if an earlier component already settled it.
func compareAsVer(selector string, other string) (int, bool) {
	sel := strings.Split(selector, ".")
	want := strings.Split(other, ".")
	for i := range want {
		if i >= len(sel) {
			// The selector is shorter and has matched so far, so it spans both
			// sides of other: "8" covers 8.0 as well as 8.1.
			return 0, false
		}
		s, err := strconv.Atoi(sel[i])
		if err != nil {
			return 0, false
		}
		w, err := strconv.Atoi(want[i])
		if err != nil {
			return 0, false
		}
		if s != w {
			if s > w {
				return 1, true
			}
			return -1, true
		}
	}
	return 0, true
}
