package baws

import "testing"

func TestNormalizeAerospikeVersion(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"7.1.0.0c", "7.1.0.0-community"},
		{"7.1.0.0f", "7.1.0.0-federal"},
		{"7.1.0.0", "7.1.0.0-enterprise"},
		{"6.4.0.1", "6.4.0.1-enterprise"},
	}
	for _, c := range cases {
		if got := normalizeAerospikeVersion(c.in); got != c.want {
			t.Errorf("normalizeAerospikeVersion(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseV7ImageName(t *testing.T) {
	cases := []struct {
		name                                 string
		wantOS, wantVer, wantAS, wantArch    string
	}{
		{"aerolab4-template-ubuntu_22.04_7.1.0.0_amd64", "ubuntu", "22.04", "7.1.0.0", "amd64"},
		{"aerolab4-template-debian_12_6.4.0.1_arm64", "debian", "12", "6.4.0.1", "arm64"},
		{"aerolab4-template-ubuntu_22.04", "ubuntu", "22.04", "", ""},
		{"aerolab4-template-", "", "", "", ""},
		{"unrelated-name", "unrelated-name", "", "", ""},
	}
	for _, c := range cases {
		os, ver, as, arch := parseV7ImageName(c.name)
		if os != c.wantOS || ver != c.wantVer || as != c.wantAS || arch != c.wantArch {
			t.Errorf("parseV7ImageName(%q) = (%q,%q,%q,%q), want (%q,%q,%q,%q)",
				c.name, os, ver, as, arch, c.wantOS, c.wantVer, c.wantAS, c.wantArch)
		}
	}
}

func TestIsCloudCIDR(t *testing.T) {
	cases := []struct {
		cidr string
		want bool
	}{
		{"10.128.0.0/19", true},
		{"10.160.0.0/19", true},
		{"10.128.0.0/16", false},  // wrong mask (must be /19)
		{"10.128.1.0/19", true},   // /19 masks the host bits: network becomes 10.128.0.0
		{"192.168.0.0/19", false}, // wrong first octet
		{"not-a-cidr", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isCloudCIDR(c.cidr); got != c.want {
			t.Errorf("isCloudCIDR(%q) = %v, want %v", c.cidr, got, c.want)
		}
	}
}

func TestCidrsOverlap(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"0.0.0.0/0", "10.0.0.0/8", false}, // default route ignored
		{"10.0.0.0/8", "0.0.0.0/0", false},
		{"10.0.0.0/16", "10.0.1.0/24", true},   // b inside a
		{"10.0.1.0/24", "10.0.0.0/16", true},   // a inside b
		{"10.0.0.0/16", "10.1.0.0/16", false},  // disjoint
		{"bad", "10.0.0.0/8", false},           // parse error
	}
	for _, c := range cases {
		if got := cidrsOverlap(c.a, c.b); got != c.want {
			t.Errorf("cidrsOverlap(%q,%q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestComputeCloudAccessURL(t *testing.T) {
	cases := []struct {
		clientType, pub, priv, want string
	}{
		{"", "1.2.3.4", "10.0.0.1", ""},
		{"unknown", "1.2.3.4", "10.0.0.1", ""},
		{"vscode", "1.2.3.4", "10.0.0.1", "http://1.2.3.4:8080"},
		{"vscode", "", "10.0.0.1", "http://10.0.0.1:8080"},
		{"ams", "1.2.3.4", "", "http://1.2.3.4:3000"},
		{"graph", "1.2.3.4", "", "http://1.2.3.4:9090"},
		{"vscode", "", "", ""},
	}
	for _, c := range cases {
		if got := computeCloudAccessURL(c.clientType, c.pub, c.priv); got != c.want {
			t.Errorf("computeCloudAccessURL(%q,%q,%q) = %q, want %q", c.clientType, c.pub, c.priv, got, c.want)
		}
	}
}

func TestToInt(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"42", 42},
		{"0", 0},
		{"", 0},
		{"notanumber", 0},
		{"-7", -7},
	}
	for _, c := range cases {
		if got := toInt(c.in); got != c.want {
			t.Errorf("toInt(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestIsAGIInstance(t *testing.T) {
	if isAGIInstance(map[string]string{}) {
		t.Error("empty tags should not be AGI")
	}
	if isAGIInstance(map[string]string{"random": "value"}) {
		t.Error("unrelated tags should not be AGI")
	}
	if !isAGIInstance(map[string]string{V7_TAG_AGI_INSTANCE: "true"}) {
		t.Error("V7_TAG_AGI_INSTANCE present should be AGI")
	}
	if !isAGIInstance(map[string]string{V7_TAG_AGI_LABEL: "mylabel"}) {
		t.Error("V7_TAG_AGI_LABEL present should be AGI")
	}
	if isAGIInstance(map[string]string{V7_TAG_AGI_INSTANCE: ""}) {
		t.Error("empty-value AGI tag should not count")
	}
}
