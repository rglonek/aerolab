//go:build integration_cloud

package e2e_test

import (
	"os"
	"testing"
)

// TestCloudInventoryMigrateDryRun reproduces the migrate smoke from
// test-migrate.sh: exercise `inventory migrate --dry-run` against the current
// cloud inventory. It is opt-in (AEROLAB_E2E_MIGRATE) on top of the cloud tier
// gate, and reads the SSH key path from AEROLAB_E2E_SSH_KEY_PATH instead of the
// hardcoded path the old script used.
func TestCloudInventoryMigrateDryRun(t *testing.T) {
	if os.Getenv("AEROLAB_E2E_MIGRATE") == "" {
		t.Skip("set AEROLAB_E2E_MIGRATE=1 to run the inventory migrate dry-run test")
	}
	c := newCloudCLI(t)

	args := []string{"inventory", "migrate", "--dry-run", "--verbose"}
	if key := os.Getenv("AEROLAB_E2E_SSH_KEY_PATH"); key != "" {
		args = append(args, "--ssh-key-path", key)
	}
	c.run(args...)
}
