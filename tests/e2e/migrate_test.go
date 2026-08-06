//go:build integration_cloud

package e2e_test

import (
	"os"
	"testing"
)

// TestCloudInventoryMigrateDryRun reproduces the migrate smoke from
// test-migrate.sh: exercise `inventory migrate --dry-run` against the current
// AWS inventory. It is opt-in through AEROLAB_E2E_MIGRATE and reads the SSH key
// path from AEROLAB_E2E_SSH_KEY_PATH instead of the hardcoded path the old
// script used.
//
// This has nothing to do with Aerospike Cloud: it migrates a v7 aerolab
// inventory on the AWS backend, and it used to sit behind the Aerospike Cloud
// tier's gate as well, which is why it skipped even in runs that set
// AEROLAB_E2E_MIGRATE=1.
func TestCloudInventoryMigrateDryRun(t *testing.T) {
	if os.Getenv("AEROLAB_E2E_MIGRATE") == "" {
		t.Skip("set AEROLAB_E2E_MIGRATE=1 to run the inventory migrate dry-run test")
	}
	c := newAWSCLI(t)

	args := []string{"inventory", "migrate", "--dry-run", "--verbose"}
	if key := os.Getenv("AEROLAB_E2E_SSH_KEY_PATH"); key != "" {
		args = append(args, "--ssh-key-path", key)
	}
	c.run(args...)
}
