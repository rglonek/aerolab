package bgcp

import (
	"regexp"
	"strconv"
	"strings"
)

// This file maps Compute Engine machine types onto the Cloud Billing SKUs that
// price them. GCP does not expose a machine-type -> SKU link, so the only
// available join key is the SKU description, e.g. "C4A Arm Instance Core
// running in Americas". The parsing below turns those descriptions into a
// billing family name, and machineBillingFamily turns a machine type name into
// the same key.
//
// Matching has to be on the whole family token rather than a string prefix:
// "c4" is a prefix of "c4a", "c4d" and "c4n", all of which have their own
// SKUs and their own (different) prices.

// billingResource is the unit a SKU rate is charged in.
type billingResource int

const (
	billingResourceCPU billingResource = iota
	billingResourceRAM
	billingResourceGPU
	billingResourceTPU
)

// machineBillingFamily is the set of billing family keys for one machine type.
// vCPU/RAM, accelerators and TPU chips can each be billed under a different
// name for the same machine type, and a machine type may legitimately have
// only some of them (a TPU VM bills per chip only, an x4 has no public SKU).
type machineBillingFamily struct {
	cpuRAM string
	gpu    string
	tpu    string
}

// familyOverrides covers machine types whose billing family is not simply the
// leading token of their name. Matched by name prefix, longest first.
var familyOverrides = []struct {
	prefix string
	family machineBillingFamily
}{
	// A2 ultra bills vCPU/RAM as plain A2 but the GPU under its own SKU.
	{"a2-ultragpu-", machineBillingFamily{cpuRAM: "a2", gpu: "a2ultra"}},
	// A3 Mega/Ultra are separate billing families from plain A3.
	{"a3-megagpu-", machineBillingFamily{cpuRAM: "a3plus", gpu: "a3plus"}},
	{"a3-ultragpu-", machineBillingFamily{cpuRAM: "a3ultra", gpu: "a3ultra"}},
	// M4 ultramem 224 is priced above the rest of the M4 family.
	{"m4-ultramem-224", machineBillingFamily{cpuRAM: "m4ultramem224"}},
	// TPU VMs are billed per chip; the vCPUs and RAM are included.
	{"ct5l-", machineBillingFamily{tpu: "tpuv5e"}},
	{"ct5lp-", machineBillingFamily{tpu: "tpuv5e"}},
	{"ct5p-", machineBillingFamily{tpu: "tpuv5p"}},
	{"ct6e-", machineBillingFamily{tpu: "tpuv6e"}},
	{"tpu7x-", machineBillingFamily{tpu: "tpu7x"}},
}

// armFamilies lists the machine families that run on 64-bit Arm cores.
var armFamilies = map[string]bool{
	"t2a": true, // Ampere Altra
	"c4a": true, // Google Axion
	"n4a": true, // Google Axion
	"a4x": true, // NVIDIA Grace
}

// machineFamilyToken is the leading token of a machine type name, which for
// most families is also the billing family key.
func machineFamilyToken(name string) string {
	name = strings.ToLower(name)
	if i := strings.Index(name, "-"); i >= 0 {
		return name[:i]
	}
	return name
}

// billingFamilyFor returns the billing family keys used to price a machine type.
func billingFamilyFor(name string) machineBillingFamily {
	name = strings.ToLower(name)
	for _, o := range familyOverrides {
		if strings.HasPrefix(name, o.prefix) {
			return o.family
		}
	}
	token := machineFamilyToken(name)
	return machineBillingFamily{cpuRAM: token, gpu: token}
}

var tpuChipsRe = regexp.MustCompile(`-(\d+)t$`)

// tpuChipCount reads the chip count out of a TPU machine type name, e.g.
// ct5lp-hightpu-4t has four chips. TPU SKUs are priced per chip-hour.
func tpuChipCount(name string) int {
	m := tpuChipsRe.FindStringSubmatch(strings.ToLower(name))
	if m == nil {
		return 0
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return n
}

// gpuSkuFamilies maps an accelerator SKU description to the billing family of
// the machine types it is attached to. Accelerators that are only sold as an
// add-on to N1 (T4, P4, P100, V100) are deliberately absent: they are not part
// of any machine type's base price.
var gpuSkuFamilies = map[string]string{
	"Nvidia Tesla A100 GPU":        "a2",
	"Nvidia Tesla A100 80GB GPU":   "a2ultra",
	"Nvidia H100 80GB GPU":         "a3",
	"Nvidia H100 80GB Plus GPU":    "a3",
	"Nvidia H100 80GB Mega GPU":    "a3plus",
	"Nvidia H100 Mega 80GB GPU":    "a3plus",
	"H200 141GB GPU":               "a3ultra",
	"A4 Nvidia B200 (1 gpu slice)": "a4",
	"Nvidia L4 GPU":                "g2",
	"RTX 6000 96GB":                "g4",
}

// tpuSkuFamilies maps a TPU SKU description to a billing family.
var tpuSkuFamilies = map[string]string{
	"TpuV5e": "tpuv5e",
	"TpuV5p": "tpuv5p",
	"TpuV6e": "tpuv6e",
	"TPU7x":  "tpu7x",
}

// skuLabelFamilies maps the leftover label of a vCPU/RAM SKU description to
// billing families, for the descriptions that do not name their family the way
// the machine type does.
var skuLabelFamilies = map[string][]string{
	"compute optimized": {"c2"},
	"memory-optimized":  {"m1", "m2"},
}

// memoryOptimizedPremium is charged on top of the base memory-optimized rate
// for the M2 family only.
const memoryOptimizedPremium = "Memory Optimized Upgrade Premium for Memory-optimized Instance "

// skuRejectMarkers identify descriptions that are not a plain pay-as-you-go
// rate for a predefined machine type: commitments, sole-tenant nodes, custom
// shapes, Dynamic Workload Scheduler and calendar-mode reservations all have
// their own prices that would otherwise be mistaken for the standard rate.
var skuRejectMarkers = []string{
	"Sole Tenancy",
	"Commitment",
	"Committed Use",
	"Custom",
	"Extended",
	"DWS ",
	"Defined Duration",
	"Calendar Mode",
	"Capacity Optimized",
	"Reserved ",
}

var runningInSuffixRe = regexp.MustCompile(` running in .*$`)

// parsedSku is the outcome of interpreting one Cloud Billing SKU description.
type parsedSku struct {
	families []string
	resource billingResource
	spot     bool
	// additive rates stack on top of the family's base rate instead of
	// replacing it.
	additive bool
}

// parseComputeSku interprets a Compute Engine SKU into the billing families and
// resource it prices. resourceGroup is the SKU's category.resourceGroup and
// preemptible its category.usageType == "Preemptible".
func parseComputeSku(description string, resourceGroup string, preemptible bool) (parsedSku, bool) {
	out := parsedSku{spot: preemptible}
	desc := strings.TrimSpace(runningInSuffixRe.ReplaceAllString(description, ""))

	// A handful of SKUs describe themselves as spot while being categorised as
	// on-demand, so trust the description over the category.
	if rest, ok := strings.CutPrefix(desc, "Spot Preemptible "); ok {
		desc = rest
		out.spot = true
	}
	if rest, ok := strings.CutSuffix(desc, " attached to Spot Preemptible VMs"); ok {
		desc = rest
		out.spot = true
	}

	switch resourceGroup {
	case "GPU":
		family, ok := gpuSkuFamilies[desc]
		if !ok {
			return out, false
		}
		out.families = []string{family}
		out.resource = billingResourceGPU
		return out, true
	case "TPU":
		family, ok := tpuSkuFamilies[desc]
		if !ok {
			return out, false
		}
		out.families = []string{family}
		out.resource = billingResourceTPU
		return out, true
	case "G1Small", "F1Micro":
		out.families = []string{"g1"}
		if resourceGroup == "F1Micro" {
			out.families = []string{"f1"}
		}
		out.resource = billingResourceCPU
		return out, true
	}

	if rest, ok := strings.CutPrefix(desc, memoryOptimizedPremium); ok {
		resource, ok := billingResourceFromWord(rest)
		if !ok {
			return out, false
		}
		out.families = []string{"m2"}
		out.resource = resource
		out.additive = true
		return out, true
	}
	for _, marker := range skuRejectMarkers {
		if strings.Contains(desc, marker) {
			return out, false
		}
	}

	label, resource, ok := splitSkuLabel(desc)
	if !ok {
		return out, false
	}
	out.resource = resource
	if families, ok := skuLabelFamilies[strings.ToLower(label)]; ok {
		out.families = families
		return out, true
	}
	// Anything left should be a bare family name such as "N2D" or "M4Ultramem224".
	if !isFamilyToken(label) {
		return out, false
	}
	out.families = []string{strings.ToLower(label)}
	return out, true
}

// skuModifierSuffixes are vendor/shape words that sit between the family name
// and "Instance" in a SKU description, e.g. "N2D AMD Instance Core".
var skuModifierSuffixes = []string{" AMD", " Arm", " Predefined", " Memory-optimized"}

// splitSkuLabel peels the boilerplate off a vCPU/RAM SKU description, leaving
// the family label: "C4A Arm Instance Core" -> "C4A", per-core.
func splitSkuLabel(desc string) (label string, resource billingResource, ok bool) {
	fields := strings.Fields(desc)
	if len(fields) < 2 {
		return "", 0, false
	}
	resource, ok = billingResourceFromWord(fields[len(fields)-1])
	if !ok {
		return "", 0, false
	}
	label = strings.Join(fields[:len(fields)-1], " ")
	label = strings.TrimSuffix(label, " Instance")
	for _, mod := range skuModifierSuffixes {
		if trimmed, found := strings.CutSuffix(label, mod); found && trimmed != "" {
			label = trimmed
			break
		}
	}
	if label == "" {
		return "", 0, false
	}
	return label, resource, true
}

func billingResourceFromWord(word string) (billingResource, bool) {
	switch strings.ToLower(strings.TrimSpace(word)) {
	case "core":
		return billingResourceCPU, true
	case "ram":
		return billingResourceRAM, true
	}
	return 0, false
}

// isFamilyToken reports whether a label looks like a bare machine family name
// rather than a description we failed to fully parse.
func isFamilyToken(label string) bool {
	if label == "" {
		return false
	}
	for _, r := range label {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

// familyRates are the per-unit hourly rates published for one billing family
// in one region.
type familyRates struct {
	onDemand [4]float64
	spot     [4]float64
}

func (r *familyRates) set(resource billingResource, spot bool, value float64) {
	if spot {
		r.spot[resource] = value
		return
	}
	r.onDemand[resource] = value
}

func (r *familyRates) add(other *familyRates) {
	for i := range r.onDemand {
		r.onDemand[i] += other.onDemand[i]
		r.spot[i] += other.spot[i]
	}
}

// rateTable holds rates keyed by region and billing family. Rates are regional,
// so a machine type must only ever be priced with the rates of its own region.
type rateTable map[string]map[string]*familyRates

func (t rateTable) rates(region, family string) *familyRates {
	byFamily, ok := t[region]
	if !ok {
		byFamily = make(map[string]*familyRates)
		t[region] = byFamily
	}
	r, ok := byFamily[family]
	if !ok {
		r = &familyRates{}
		byFamily[family] = r
	}
	return r
}

func (t rateTable) lookup(region, family string) *familyRates {
	if family == "" {
		return nil
	}
	byFamily, ok := t[region]
	if !ok {
		return nil
	}
	return byFamily[family]
}
