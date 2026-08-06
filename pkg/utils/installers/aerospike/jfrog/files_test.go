package jfrog

import (
	"strings"
	"testing"
)

func namedFile(name string) File {
	return File{Repo: "database-deb-dev-local", Name: name, Parts: ParseFileName(name)}
}

// devBuildFiles is a trimmed-down copy of the artifact list JFrog returns for
// build "aerospike-server" number "8.1.3.0-70-g282a6817d-artifacts", with the
// names reproduced verbatim.
func devBuildFiles() Files {
	names := []string{
		"aerospike-server-enterprise_8.1.3.0-70-g282a6817d-1ubuntu24.04_amd64.deb",
		"aerospike-server-enterprise_8.1.3.0-70-g282a6817d-1ubuntu24.04_amd64.deb.asc",
		"aerospike-server-enterprise_8.1.3.0-70-g282a6817d-1ubuntu24.04_arm64.deb",
		"aerospike-server-enterprise_8.1.3.0-70-g282a6817d-1debian12_amd64.deb",
		"aerospike-server-community_8.1.3.0-70-g282a6817d-1ubuntu24.04_amd64.deb",
		"aerospike-server-enterprise-8.1.3.0_70_g282a6817d-1.el9.x86_64.rpm",
		"aerospike-server-enterprise-8.1.3.0_70_g282a6817d-1.amzn2023.aarch64.rpm",
		"aerospike-server-federal-8.1.3.0_70_g282a6817d-1.el10.x86_64.rpm",
	}
	fs := make(Files, 0, len(names))
	for _, n := range names {
		fs = append(fs, namedFile(n))
	}
	return fs
}

// A dev build whose packages carry a git describe in the version/release
// field used to match nothing at all, so `cluster create -v <dev build>`
// failed with "no enterprise deb package found for ubuntu/24.04/x86_64".
func TestMatch_GitDescribeDevBuild(t *testing.T) {
	fs := devBuildFiles()
	cases := []struct {
		crit MatchCriteria
		want string
	}{
		{
			MatchCriteria{Edition: "enterprise", OSName: "ubuntu", OSVersion: "24.04", Arch: "x86_64"},
			"aerospike-server-enterprise_8.1.3.0-70-g282a6817d-1ubuntu24.04_amd64.deb",
		},
		{
			MatchCriteria{Edition: "enterprise", OSName: "ubuntu", OSVersion: "24.04", Arch: "aarch64"},
			"aerospike-server-enterprise_8.1.3.0-70-g282a6817d-1ubuntu24.04_arm64.deb",
		},
		{
			MatchCriteria{Edition: "community", OSName: "ubuntu", OSVersion: "24.04", Arch: "x86_64"},
			"aerospike-server-community_8.1.3.0-70-g282a6817d-1ubuntu24.04_amd64.deb",
		},
		{
			MatchCriteria{Edition: "enterprise", OSName: "centos", OSVersion: "9", Arch: "x86_64"},
			"aerospike-server-enterprise-8.1.3.0_70_g282a6817d-1.el9.x86_64.rpm",
		},
		{
			MatchCriteria{Edition: "federal", OSName: "centos", OSVersion: "10", Arch: "x86_64"},
			"aerospike-server-federal-8.1.3.0_70_g282a6817d-1.el10.x86_64.rpm",
		},
	}
	for _, tc := range cases {
		got, err := fs.Match(tc.crit)
		if err != nil {
			t.Errorf("Match(%+v): %v", tc.crit, err)
			continue
		}
		if got.Name != tc.want {
			t.Errorf("Match(%+v):\n  got  %s\n  want %s", tc.crit, got.Name, tc.want)
		}
	}
}

// A signature must never be selected as the package to install.
func TestMatch_SkipsSignatures(t *testing.T) {
	got, err := fs99Signatures().Match(MatchCriteria{Edition: "enterprise", OSName: "ubuntu", OSVersion: "24.04", Arch: "x86_64"})
	if err == nil {
		t.Fatalf("expected no match when only signatures are present, got %s", got.Name)
	}
}

func fs99Signatures() Files {
	return Files{namedFile("aerospike-server-enterprise_8.1.3.0-70-g282a6817d-1ubuntu24.04_amd64.deb.asc")}
}

// When the build has the edition but not the requested target, the error must
// name the targets it does have.
func TestMatch_ErrorListsAvailableTargets(t *testing.T) {
	_, err := devBuildFiles().Match(MatchCriteria{Edition: "enterprise", OSName: "ubuntu", OSVersion: "22.04", Arch: "x86_64"})
	if err == nil {
		t.Fatal("expected an error for an OS version the build does not carry")
	}
	for _, want := range []string{"ubuntu24.04/x86_64", "debian12/x86_64"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should list %q; got: %v", want, err)
		}
	}
}

// When nothing parses at all, the error must show the artifact names so an
// unrecognised naming scheme can be told apart from a missing package.
func TestMatch_ErrorSamplesUnrecognisedNames(t *testing.T) {
	fs := Files{
		namedFile("aerospike-server-enterprise_8.1.3.0+weird+scheme.deb"),
		namedFile("aerospike-server-enterprise_8.1.3.0+weird+scheme.deb.asc"),
	}
	_, err := fs.Match(MatchCriteria{Edition: "enterprise", OSName: "ubuntu", OSVersion: "24.04", Arch: "x86_64"})
	if err == nil {
		t.Fatal("expected an error when no artifact parses")
	}
	if !strings.Contains(err.Error(), "aerospike-server-enterprise_8.1.3.0+weird+scheme.deb") {
		t.Errorf("error should sample the unrecognised names; got: %v", err)
	}
	if strings.Contains(err.Error(), ".deb.asc") {
		t.Errorf("error should not report signatures as unrecognised packages; got: %v", err)
	}
}

func TestMatch_UnsupportedOS(t *testing.T) {
	if _, err := devBuildFiles().Match(MatchCriteria{Edition: "enterprise", OSName: "windows", OSVersion: "11", Arch: "x86_64"}); err == nil {
		t.Fatal("expected an error for an OS JFrog does not publish packages for")
	}
}
