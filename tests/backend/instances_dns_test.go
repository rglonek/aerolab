//go:build integration_cloud

package backend_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aerospike/aerolab/pkg/backend/backends"
	"github.com/aerospike/aerolab/pkg/backend/clouds/baws"
	"github.com/aerospike/aerolab/pkg/backend/clouds/bgcp"
	"github.com/stretchr/testify/require"
)

func Test20_InstancesDNS(t *testing.T) {
	t.Cleanup(cleanup)
	test := &testInstancesDNS{}
	t.Run("setup", testSetup)
	t.Run("inventory empty", testInventoryEmpty)
	t.Run("create instance", test.testCreateInstance)
	t.Run("test dns", test.testInstancesDNS)
	t.Run("cleanup dns", test.testCleanupDNS)
	t.Run("terminate instance", test.testInstancesTerminate)
	t.Run("cleanup dns", test.testCleanupDNS)
	t.Run("end inventory empty", testInventoryEmpty)
}

// dnsConfig describes the hosted zone the DNS test attaches records to. It is
// account-specific, so the test skips unless it is configured.
//
// AEROLAB_TEST_DNS_ZONE_ID is the Route53 hosted zone id on AWS and the managed
// zone name on GCP. AEROLAB_TEST_DNS_REGION is the Route53 region on AWS; GCP
// Cloud DNS is global.
func dnsConfig(t *testing.T) (zoneID, domain, region string) {
	t.Helper()
	zoneID = os.Getenv("AEROLAB_TEST_DNS_ZONE_ID")
	domain = os.Getenv("AEROLAB_TEST_DNS_DOMAIN")
	if zoneID == "" || domain == "" {
		t.Skip("set AEROLAB_TEST_DNS_ZONE_ID and AEROLAB_TEST_DNS_DOMAIN to run the DNS test")
	}
	region = os.Getenv("AEROLAB_TEST_DNS_REGION")
	if region == "" {
		region = "us-east-1"
		if cloud == "gcp" {
			region = "global"
		}
	}
	return zoneID, domain, region
}

func (d *testInstancesDNS) testCreateInstance(t *testing.T) {
	require.NoError(t, setup(false))
	if cloud == "docker" {
		t.Skip("docker does not support dns")
		return
	}
	zoneID, domain, region := dnsConfig(t)
	require.NoError(t, testBackend.RefreshChangedInventory())
	image := getBasicImage(t)
	placement := Options.TestRegions[0] + "a"
	if strings.Count(Options.TestRegions[0], "-") == 1 {
		placement = Options.TestRegions[0] + "-a"
	}
	params := map[backends.BackendType]any{
		backends.BackendTypeAWS: &baws.CreateInstanceParams{
			Image:            image,
			NetworkPlacement: Options.TestRegions[0] + "a",
			InstanceType:     "r6a.large",
			Disks:            []string{"type=gp2,size=20,count=1"},
			Firewalls:        []string{},
			CustomDNS: &backends.InstanceDNS{
				DomainID:   zoneID,
				DomainName: domain,
				Region:     region,
			},
		},
		backends.BackendTypeGCP: gcpParams(&bgcp.CreateInstanceParams{
			Image:            image,
			NetworkPlacement: placement,
			InstanceType:     "e2-standard-4",
			Disks:            []string{"type=pd-ssd,size=20,count=1"},
			Firewalls:        []string{},
			CustomDNS: &backends.InstanceDNS{
				DomainID:   zoneID,
				DomainName: domain,
				Region:     region,
			},
		}),
	}
	insts, err := testBackend.CreateInstances(&backends.CreateInstanceInput{
		ClusterName:           "test-cluster",
		Nodes:                 3,
		BackendType:           backendType,
		Owner:                 "test-owner",
		Description:           "test-description",
		BackendSpecificParams: params,
	}, 2*time.Minute)
	require.NoError(t, err)
	require.Equal(t, insts.Instances.Count(), 3)
	err = testBackend.RefreshChangedInventory()
	require.NoError(t, err)
	require.Equal(t, testBackend.GetInventory().Instances.WithNotState(backends.LifeCycleStateTerminated).Count(), 3)
}

func (d *testInstancesDNS) testInstancesDNS(t *testing.T) {
	require.NoError(t, setup(false))
	if cloud == "docker" {
		t.Skip("docker does not support dns")
		return
	}
	zoneID, domain, region := dnsConfig(t)
	require.NoError(t, testBackend.RefreshChangedInventory())
	inst := testBackend.GetInventory().Instances.WithNotState(backends.LifeCycleStateTerminated)
	require.Equal(t, inst.Count(), 3)
	for _, i := range inst.Describe() {
		require.Equal(t, i.CustomDNS.DomainID, zoneID)
		require.Equal(t, i.CustomDNS.DomainName, domain)
		require.Equal(t, i.CustomDNS.Region, region)
		// The record name is left empty at create time, so each backend fills
		// in its own default: the instance id on AWS, a hash of the instance
		// name prefixed with "i-" on GCP.
		if cloud == "gcp" {
			require.True(t, strings.HasPrefix(i.CustomDNS.Name, "i-"), "GCP DNS name %q should be an i-<hash> record", i.CustomDNS.Name)
		} else {
			require.Equal(t, i.CustomDNS.Name, i.InstanceID)
		}
	}
}

func (d *testInstancesDNS) testInstancesTerminate(t *testing.T) {
	require.NoError(t, setup(false))
	if cloud == "docker" {
		t.Skip("docker does not support dns")
		return
	}
	require.NoError(t, testBackend.RefreshChangedInventory())
	inst := testBackend.GetInventory().Instances.WithNotState(backends.LifeCycleStateTerminated)
	err := inst.Terminate(2 * time.Minute)
	require.NoError(t, err)
	err = testBackend.RefreshChangedInventory()
	require.NoError(t, err)
	require.Equal(t, testBackend.GetInventory().Instances.WithNotState(backends.LifeCycleStateTerminated).Count(), 0)
	require.NoError(t, testBackend.GetInventory().Firewalls.Delete(10*time.Minute))
}

func (d *testInstancesDNS) testCleanupDNS(t *testing.T) {
	require.NoError(t, setup(false))
	if cloud == "docker" {
		t.Skip("docker does not support dns")
		return
	}
	require.NoError(t, testBackend.RefreshChangedInventory())
	err := testBackend.CleanupDNS()
	require.NoError(t, err)
}
