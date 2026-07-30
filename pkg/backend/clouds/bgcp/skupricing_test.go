package bgcp

import (
	"slices"
	"testing"
)

func TestParseComputeSku(t *testing.T) {
	tests := []struct {
		name          string
		description   string
		resourceGroup string
		preemptible   bool
		wantFamilies  []string
		wantResource  billingResource
		wantSpot      bool
		wantAdditive  bool
		wantReject    bool
	}{
		{
			name:          "plain family core",
			description:   "N2 Instance Core running in Americas",
			resourceGroup: "CPU",
			wantFamilies:  []string{"n2"},
			wantResource:  billingResourceCPU,
		},
		{
			name:          "amd modifier does not leak into the family",
			description:   "N2D AMD Instance Ram running in Americas",
			resourceGroup: "RAM",
			wantFamilies:  []string{"n2d"},
			wantResource:  billingResourceRAM,
		},
		{
			name:          "arm modifier",
			description:   "C4A Arm Instance Core running in Americas",
			resourceGroup: "CPU",
			wantFamilies:  []string{"c4a"},
			wantResource:  billingResourceCPU,
		},
		{
			name:          "predefined modifier",
			description:   "N1 Predefined Instance Core running in Americas",
			resourceGroup: "N1Standard",
			wantFamilies:  []string{"n1"},
			wantResource:  billingResourceCPU,
		},
		{
			name:          "compute optimized is c2",
			description:   "Compute optimized Core running in Americas",
			resourceGroup: "CPU",
			wantFamilies:  []string{"c2"},
			wantResource:  billingResourceCPU,
		},
		{
			name:          "compute optimized with instance word",
			description:   "Compute optimized Instance Ram running in Americas",
			resourceGroup: "RAM",
			wantFamilies:  []string{"c2"},
			wantResource:  billingResourceRAM,
		},
		{
			name:          "memory optimized covers m1 and m2",
			description:   "Memory-optimized Instance Core running in Americas",
			resourceGroup: "CPU",
			wantFamilies:  []string{"m1", "m2"},
			wantResource:  billingResourceCPU,
		},
		{
			name:          "m3 keeps its own family",
			description:   "M3 Memory-optimized Instance Ram running in Americas",
			resourceGroup: "RAM",
			wantFamilies:  []string{"m3"},
			wantResource:  billingResourceRAM,
		},
		{
			name:          "m2 upgrade premium stacks on the base rate",
			description:   "Memory Optimized Upgrade Premium for Memory-optimized Instance Core running in Americas",
			resourceGroup: "CPU",
			wantFamilies:  []string{"m2"},
			wantResource:  billingResourceCPU,
			wantAdditive:  true,
		},
		{
			name:          "m4 ultramem has its own family",
			description:   "M4Ultramem224 Instance Ram running in Americas",
			resourceGroup: "RAM",
			wantFamilies:  []string{"m4ultramem224"},
			wantResource:  billingResourceRAM,
		},
		{
			name:          "spot prefix is stripped",
			description:   "Spot Preemptible C3D Instance Core running in Americas",
			resourceGroup: "CPU",
			preemptible:   true,
			wantFamilies:  []string{"c3d"},
			wantResource:  billingResourceCPU,
			wantSpot:      true,
		},
		{
			name:          "spot description overrides an on-demand category",
			description:   "Spot Preemptible A4 Nvidia B200 (1 gpu slice) running in Americas",
			resourceGroup: "GPU",
			wantFamilies:  []string{"a4"},
			wantResource:  billingResourceGPU,
			wantSpot:      true,
		},
		{
			name:          "gpu sku",
			description:   "Nvidia H100 80GB Mega GPU running in Americas",
			resourceGroup: "GPU",
			wantFamilies:  []string{"a3plus"},
			wantResource:  billingResourceGPU,
		},
		{
			name:          "tpu sku",
			description:   "TpuV5e running in Americas",
			resourceGroup: "TPU",
			wantFamilies:  []string{"tpuv5e"},
			wantResource:  billingResourceTPU,
		},
		{
			name:          "tpu spot sku",
			description:   "TpuV6e attached to Spot Preemptible VMs running in Americas",
			resourceGroup: "TPU",
			preemptible:   true,
			wantFamilies:  []string{"tpuv6e"},
			wantResource:  billingResourceTPU,
			wantSpot:      true,
		},
		{
			name:          "g1 small",
			description:   "Small Instance with 1 VCPU running in Americas",
			resourceGroup: "G1Small",
			wantFamilies:  []string{"g1"},
			wantResource:  billingResourceCPU,
		},
		{
			name:          "sole tenancy is rejected",
			description:   "C3 Sole Tenancy Instance Core running in Americas",
			resourceGroup: "CPU",
			wantReject:    true,
		},
		{
			name:          "custom shapes are rejected",
			description:   "N2 Custom Instance Core running in Americas",
			resourceGroup: "CPU",
			wantReject:    true,
		},
		{
			name:          "commitments are rejected",
			description:   "Commitment v1: Compute optimized Cpu in Dallas for 1 Year",
			resourceGroup: "CPU",
			wantReject:    true,
		},
		{
			name:          "dws rates are rejected",
			description:   "DWS Defined Duration A3 Core running in Americas",
			resourceGroup: "CPU",
			wantReject:    true,
		},
		{
			name:          "calendar mode reservations are rejected",
			description:   "Reserved M4 Core in Americas",
			resourceGroup: "CPU",
			wantReject:    true,
		},
		{
			name:          "sole tenancy premium is rejected",
			description:   "Sole Tenancy Premium for C4 Sole Tenancy Instance Core running in Americas",
			resourceGroup: "CPU",
			wantReject:    true,
		},
		{
			name:          "unknown gpu is rejected",
			description:   "Nvidia Tesla T4 GPU running in Americas",
			resourceGroup: "GPU",
			wantReject:    true,
		},
		{
			name:          "local ssd is rejected",
			description:   "SSD backed Local Storage running in Americas",
			resourceGroup: "LocalSSD",
			wantReject:    true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseComputeSku(tc.description, tc.resourceGroup, tc.preemptible)
			if tc.wantReject {
				if ok {
					t.Fatalf("expected rejection, got families %v", got.families)
				}
				return
			}
			if !ok {
				t.Fatal("expected the sku to be parsed")
			}
			if !slices.Equal(got.families, tc.wantFamilies) {
				t.Errorf("families = %v, want %v", got.families, tc.wantFamilies)
			}
			if got.resource != tc.wantResource {
				t.Errorf("resource = %v, want %v", got.resource, tc.wantResource)
			}
			if got.spot != tc.wantSpot {
				t.Errorf("spot = %v, want %v", got.spot, tc.wantSpot)
			}
			if got.additive != tc.wantAdditive {
				t.Errorf("additive = %v, want %v", got.additive, tc.wantAdditive)
			}
		})
	}
}

func TestBillingFamilyFor(t *testing.T) {
	tests := []struct {
		machineType string
		want        machineBillingFamily
	}{
		{"n2-standard-8", machineBillingFamily{cpuRAM: "n2", gpu: "n2"}},
		{"c2-standard-4", machineBillingFamily{cpuRAM: "c2", gpu: "c2"}},
		{"c2d-highcpu-2", machineBillingFamily{cpuRAM: "c2d", gpu: "c2d"}},
		{"c4a-standard-4", machineBillingFamily{cpuRAM: "c4a", gpu: "c4a"}},
		{"m1-ultramem-40", machineBillingFamily{cpuRAM: "m1", gpu: "m1"}},
		{"m2-ultramem-208", machineBillingFamily{cpuRAM: "m2", gpu: "m2"}},
		{"m4-ultramem-112", machineBillingFamily{cpuRAM: "m4", gpu: "m4"}},
		{"m4-ultramem-224", machineBillingFamily{cpuRAM: "m4ultramem224"}},
		{"a2-highgpu-1g", machineBillingFamily{cpuRAM: "a2", gpu: "a2"}},
		{"a2-ultragpu-4g", machineBillingFamily{cpuRAM: "a2", gpu: "a2ultra"}},
		{"a3-highgpu-8g", machineBillingFamily{cpuRAM: "a3", gpu: "a3"}},
		{"a3-megagpu-8g", machineBillingFamily{cpuRAM: "a3plus", gpu: "a3plus"}},
		{"a3-ultragpu-8g", machineBillingFamily{cpuRAM: "a3ultra", gpu: "a3ultra"}},
		{"a4-highgpu-8g", machineBillingFamily{cpuRAM: "a4", gpu: "a4"}},
		{"x4-megamem-960-metal", machineBillingFamily{cpuRAM: "x4", gpu: "x4"}},
		{"ct5l-hightpu-8t", machineBillingFamily{tpu: "tpuv5e"}},
		{"ct5lp-hightpu-4t", machineBillingFamily{tpu: "tpuv5e"}},
		{"ct5p-hightpu-4t", machineBillingFamily{tpu: "tpuv5p"}},
		{"ct6e-standard-8t", machineBillingFamily{tpu: "tpuv6e"}},
		{"tpu7x-standard-4t", machineBillingFamily{tpu: "tpu7x"}},
	}
	for _, tc := range tests {
		t.Run(tc.machineType, func(t *testing.T) {
			if got := billingFamilyFor(tc.machineType); got != tc.want {
				t.Errorf("billingFamilyFor(%q) = %+v, want %+v", tc.machineType, got, tc.want)
			}
		})
	}
}

func TestTpuChipCount(t *testing.T) {
	tests := map[string]int{
		"ct5lp-hightpu-4t":  4,
		"ct5l-hightpu-1t":   1,
		"ct6e-standard-8t":  8,
		"tpu7x-standard-4t": 4,
		"n2-standard-8":     0,
	}
	for name, want := range tests {
		if got := tpuChipCount(name); got != want {
			t.Errorf("tpuChipCount(%q) = %d, want %d", name, got, want)
		}
	}
}

func TestArmFamilyDetection(t *testing.T) {
	arm := []string{"t2a-standard-4", "c4a-standard-8", "n4a-standard-2", "a4x-highgpu-4g"}
	x86 := []string{"t2d-standard-4", "n2-standard-8", "c4-standard-8", "x4-megamem-960-metal"}
	for _, name := range arm {
		if !armFamilies[machineFamilyToken(name)] {
			t.Errorf("%s should be arm64", name)
		}
	}
	for _, name := range x86 {
		if armFamilies[machineFamilyToken(name)] {
			t.Errorf("%s should be x86_64", name)
		}
	}
}
