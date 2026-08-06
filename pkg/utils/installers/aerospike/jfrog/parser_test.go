package jfrog

import "testing"

func TestParseFileName_RPM(t *testing.T) {
	cases := []struct {
		name string
		want NameParts
	}{
		{
			"aerospike-server-community-8.1.3.0-28.amzn2023.aarch64.rpm",
			NameParts{Edition: "community", Version: "8.1.3.0", Release: "28", OSName: "amazon", OSVersion: "2023", Arch: "aarch64", Format: "rpm"},
		},
		{
			"aerospike-server-community-8.1.3.0-28.amzn2023.x86_64.rpm",
			NameParts{Edition: "community", Version: "8.1.3.0", Release: "28", OSName: "amazon", OSVersion: "2023", Arch: "x86_64", Format: "rpm"},
		},
		{
			"aerospike-server-enterprise-8.1.3.0-28.el9.x86_64.rpm",
			NameParts{Edition: "enterprise", Version: "8.1.3.0", Release: "28", OSName: "centos", OSVersion: "9", Arch: "x86_64", Format: "rpm"},
		},
		{
			"aerospike-server-federal-8.1.3.0-28.amzn2.aarch64.rpm",
			NameParts{Edition: "federal", Version: "8.1.3.0", Release: "28", OSName: "amazon", OSVersion: "2", Arch: "aarch64", Format: "rpm"},
		},
		// resilient: double "aerospike-" prefix + "-TEST" tag
		{
			"aerospike-aerospike-server-enterprise-TEST-8.1.3.0-36.el9.x86_64.rpm",
			NameParts{Edition: "enterprise", Version: "8.1.3.0", Release: "36", OSName: "centos", OSVersion: "9", Arch: "x86_64", Format: "rpm"},
		},
		// dev builds: git describe folded into the rpm version field, where
		// its dashes become underscores because rpm forbids them there
		{
			"aerospike-server-enterprise-8.1.3.0_70_g282a6817d-1.el9.x86_64.rpm",
			NameParts{Edition: "enterprise", Version: "8.1.3.0", Release: "70_g282a6817d-1", OSName: "centos", OSVersion: "9", Arch: "x86_64", Format: "rpm"},
		},
		{
			"aerospike-server-federal-8.1.3.0_70_g282a6817d-1.amzn2023.x86_64.rpm",
			NameParts{Edition: "federal", Version: "8.1.3.0", Release: "70_g282a6817d-1", OSName: "amazon", OSVersion: "2023", Arch: "x86_64", Format: "rpm"},
		},
		// el10 is a newer distro tag than the ones the parser shipped with
		{
			"aerospike-server-community-8.1.3.0_70_g282a6817d-1.el10.aarch64.rpm",
			NameParts{Edition: "community", Version: "8.1.3.0", Release: "70_g282a6817d-1", OSName: "centos", OSVersion: "10", Arch: "aarch64", Format: "rpm"},
		},
	}
	for _, tc := range cases {
		got := ParseFileName(tc.name)
		if got == nil {
			t.Errorf("%s: parse returned nil", tc.name)
			continue
		}
		if *got != tc.want {
			t.Errorf("%s:\n  got  %+v\n  want %+v", tc.name, *got, tc.want)
		}
	}
}

func TestParseFileName_DEB(t *testing.T) {
	cases := []struct {
		name string
		want NameParts
	}{
		{
			"aerospike-server-community_8.1.3.0-28debian12_amd64.deb",
			NameParts{Edition: "community", Version: "8.1.3.0", Release: "28", OSName: "debian", OSVersion: "12", Arch: "x86_64", Format: "deb"},
		},
		{
			"aerospike-server-enterprise_8.1.3.0-28ubuntu24.04_arm64.deb",
			NameParts{Edition: "enterprise", Version: "8.1.3.0", Release: "28", OSName: "ubuntu", OSVersion: "24.04", Arch: "aarch64", Format: "deb"},
		},
		{
			"aerospike-server-enterprise_8.1.3.0-28ubuntu26.04_amd64.deb",
			NameParts{Edition: "enterprise", Version: "8.1.3.0", Release: "28", OSName: "ubuntu", OSVersion: "26.04", Arch: "x86_64", Format: "deb"},
		},
		// resilient: double "aerospike-" prefix + "-TEST" tag (the real-world case)
		{
			"aerospike-aerospike-server-enterprise-TEST_8.1.3.0-36ubuntu24.04_arm64.deb",
			NameParts{Edition: "enterprise", Version: "8.1.3.0", Release: "36", OSName: "ubuntu", OSVersion: "24.04", Arch: "aarch64", Format: "deb"},
		},
		// dev builds: git describe plus a packaging revision in the release
		// field, e.g. build 8.1.3.0-70-g282a6817d
		{
			"aerospike-server-enterprise_8.1.3.0-70-g282a6817d-1ubuntu24.04_amd64.deb",
			NameParts{Edition: "enterprise", Version: "8.1.3.0", Release: "70-g282a6817d-1", OSName: "ubuntu", OSVersion: "24.04", Arch: "x86_64", Format: "deb"},
		},
		{
			"aerospike-server-community_8.1.3.0-70-g282a6817d-1debian13_arm64.deb",
			NameParts{Edition: "community", Version: "8.1.3.0", Release: "70-g282a6817d-1", OSName: "debian", OSVersion: "13", Arch: "aarch64", Format: "deb"},
		},
		{
			"aerospike-server-federal_8.1.3.0-70-g282a6817d-1ubuntu26.04_amd64.deb",
			NameParts{Edition: "federal", Version: "8.1.3.0", Release: "70-g282a6817d-1", OSName: "ubuntu", OSVersion: "26.04", Arch: "x86_64", Format: "deb"},
		},
		// the 5.7-era pipeline gave debs rpm-style dot separators
		{
			"aerospike-server-enterprise-5.7.0.32.ubuntu20.04.x86_64.deb",
			NameParts{Edition: "enterprise", Version: "5.7.0.32", OSName: "ubuntu", OSVersion: "20.04", Arch: "x86_64", Format: "deb"},
		},
	}
	for _, tc := range cases {
		got := ParseFileName(tc.name)
		if got == nil {
			t.Errorf("%s: parse returned nil", tc.name)
			continue
		}
		if *got != tc.want {
			t.Errorf("%s:\n  got  %+v\n  want %+v", tc.name, *got, tc.want)
		}
	}
}

func TestParseFileName_Ignored(t *testing.T) {
	skip := []string{
		"aerospike-server-community-8.1.3.0-28.amzn2023.aarch64.rpm.asc",
		"aerospike-server-community_8.1.3.0-28debian12_amd64.deb.asc",
		"aerospike-server-enterprise_8.1.3.0-70-g282a6817d-1ubuntu24.04_amd64.deb.asc",
		"aerospike-server-enterprise-8.1.3.0_70_g282a6817d-1.el9.x86_64.rpm.asc",
		"aerospike-server-enterprise_8.0.0.8_ubuntu24.04_x86_64.tgz", // public-download style
		"aerospike-tools_11.2.2_ubuntu24.04_aarch64.tgz",
		"aerospike-client-c_7.5.0-7-debian12_amd64.deb",
		"random.txt",
		"",
	}
	for _, n := range skip {
		if got := ParseFileName(n); got != nil {
			t.Errorf("%s: expected nil, got %+v", n, *got)
		}
	}
}

func TestParseToolsFileName(t *testing.T) {
	cases := []struct {
		name string
		want ToolsParts
	}{
		{
			"aerospike-tools_11.2.2_ubuntu24.04_aarch64.tgz",
			ToolsParts{Version: "11.2.2", OSName: "ubuntu", OSVersion: "24.04", Arch: "aarch64"},
		},
		{
			"aerospike-tools_11.2.2_ubuntu24.04_x86_64.tgz",
			ToolsParts{Version: "11.2.2", OSName: "ubuntu", OSVersion: "24.04", Arch: "x86_64"},
		},
		{
			"aerospike-tools_11.2.2_amzn2023_x86_64.tgz",
			ToolsParts{Version: "11.2.2", OSName: "amazon", OSVersion: "2023", Arch: "x86_64"},
		},
		{
			"aerospike-tools_11.2.2_el9_aarch64.tgz",
			ToolsParts{Version: "11.2.2", OSName: "centos", OSVersion: "9", Arch: "aarch64"},
		},
		{
			"aerospike-tools_11.2.2_debian12_x86_64.tgz",
			ToolsParts{Version: "11.2.2", OSName: "debian", OSVersion: "12", Arch: "x86_64"},
		},
		// resilient: double "aerospike-" prefix
		{
			"aerospike-aerospike-tools_11.2.2_ubuntu24.04_aarch64.tgz",
			ToolsParts{Version: "11.2.2", OSName: "ubuntu", OSVersion: "24.04", Arch: "aarch64"},
		},
	}
	for _, tc := range cases {
		got := ParseToolsFileName(tc.name)
		if got == nil {
			t.Errorf("%s: parse returned nil", tc.name)
			continue
		}
		if *got != tc.want {
			t.Errorf("%s:\n  got  %+v\n  want %+v", tc.name, *got, tc.want)
		}
	}

	// non-tools names must not parse as tools
	for _, n := range []string{
		"aerospike-server-enterprise_8.0.0.8_ubuntu24.04_x86_64.tgz",
		"aerospike-tools_11.2.2_ubuntu24.04_aarch64.tgz.asc",
		"aerospike-server-community-8.1.3.0-28.amzn2023.aarch64.rpm",
		"random.txt",
		"",
	} {
		if got := ParseToolsFileName(n); got != nil {
			t.Errorf("%s: expected nil, got %+v", n, *got)
		}
	}
}

func TestMatchTools(t *testing.T) {
	fs := Files{
		{Name: "aerospike-server-enterprise_8.1.3.0-28ubuntu24.04_arm64.deb", Parts: ParseFileName("aerospike-server-enterprise_8.1.3.0-28ubuntu24.04_arm64.deb")},
		{Name: "aerospike-tools_11.2.2_ubuntu24.04_x86_64.tgz"},
		{Name: "aerospike-tools_11.2.2_ubuntu24.04_aarch64.tgz"},
	}
	got := fs.MatchTools(MatchCriteria{OSName: "ubuntu", OSVersion: "24.04", Arch: "aarch64"})
	if got == nil || got.Name != "aerospike-tools_11.2.2_ubuntu24.04_aarch64.tgz" {
		t.Fatalf("MatchTools aarch64: got %v", got)
	}
	if got := fs.MatchTools(MatchCriteria{OSName: "ubuntu", OSVersion: "24.04", Arch: "x86_64"}); got == nil || got.Name != "aerospike-tools_11.2.2_ubuntu24.04_x86_64.tgz" {
		t.Fatalf("MatchTools x86_64: got %v", got)
	}
	if got := fs.MatchTools(MatchCriteria{OSName: "debian", OSVersion: "12", Arch: "x86_64"}); got != nil {
		t.Fatalf("MatchTools miss: expected nil, got %v", got)
	}
}

func TestEditionFromInput(t *testing.T) {
	cases := []struct {
		in          string
		def         string
		wantEdition string
		wantVersion string
	}{
		{"8.1.3.0-28-g302194ebc", "enterprise", "enterprise", "8.1.3.0-28-g302194ebc"},
		// git SHA ending in 'c' must NOT be treated as community shorthand
		{"8.1.3.0-28-g302194ebc", "", "enterprise", "8.1.3.0-28-g302194ebc"},
		// explicit separators
		{"8.1.3.0-28-g302194ebc:c", "enterprise", "community", "8.1.3.0-28-g302194ebc"},
		{"8.1.3.0-28-g302194ebc:community", "enterprise", "community", "8.1.3.0-28-g302194ebc"},
		{"8.1.3.0-28-g302194ebc:f", "enterprise", "federal", "8.1.3.0-28-g302194ebc"},
		{"8.1.3.0-28-g302194ebc:e", "community", "enterprise", "8.1.3.0-28-g302194ebc"},
		// default override
		{"8.1.3.0-28-g302194ebc", "community", "community", "8.1.3.0-28-g302194ebc"},
		// unknown suffix is left in place
		{"foo:bar", "enterprise", "enterprise", "foo:bar"},
	}
	for _, tc := range cases {
		gotEd, gotV := EditionFromInput(tc.in, tc.def)
		if gotEd != tc.wantEdition || gotV != tc.wantVersion {
			t.Errorf("EditionFromInput(%q, %q) = (%q, %q); want (%q, %q)",
				tc.in, tc.def, gotEd, gotV, tc.wantEdition, tc.wantVersion)
		}
	}
}
