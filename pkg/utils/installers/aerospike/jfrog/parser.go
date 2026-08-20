package jfrog

import (
	"regexp"
	"strings"
)

// NameParts is the parsed form of a JFrog artifact filename. Only the
// "aerospike-server-{edition}" RPM/DEB packages used by the cluster-create
// flow are parsed; everything else (asc signatures, source tarballs, etc.)
// returns nil.
type NameParts struct {
	Edition   string // community | enterprise | federal
	Version   string // 8.1.3.0
	Release   string // 28 | 70-g282a6817d-1 (git describe on dev builds) | ""
	OSName    string // amazon | centos | debian | ubuntu
	OSVersion string // 2023 | 9 | 12 | 24.04
	Arch      string // x86_64 | aarch64    (normalised to aerospike package conventions)
	Format    string // rpm | deb
}

// Resilient matching notes:
//   - An optional prefix ending in "-" is allowed before "aerospike-server-"
//     so double-prefixed CI names like "aerospike-aerospike-server-..." match.
//   - An optional tag (e.g. "-TEST") may sit between the edition and the
//     version separator, so names like
//     "aerospike-aerospike-server-enterprise-TEST_8.1.3.0-36ubuntu24.04_arm64.deb"
//     match. For rpm the tag segments must start with a letter so they can't
//     eat the numeric version that follows the "-" separator.
//   - Only the upstream version is pinned to a numeric shape. Everything
//     between it and the osTag is treated as an opaque release field, because
//     dev builds put a git describe there ("70-g282a6817d-1") and rpm rewrites
//     its dashes as underscores ("8.1.3.0_70_g282a6817d-1"). The osTag, arch
//     and extension are the only reliable anchors at the tail of the name.

// osTagPat matches the distro tag JFrog embeds in artifact names:
// "amzn2023", "el9", "el10", "debian13", "ubuntu24.04".
const osTagPat = `((?:amzn|el|debian|ubuntu)[0-9]+(?:\.[0-9]+)?)`

// archPat accepts both the dpkg and the rpm architecture vocabularies; JFrog
// uses dpkg labels on server debs but rpm labels on some other debs, and
// debArch folds them into one.
const archPat = `(amd64|arm64|x86_64|aarch64)`

// versionPat matches the upstream version ("8.1.3.0"). It must end on a digit
// so a dot-separated name does not leave the separator glued to the version.
const versionPat = `([0-9](?:[0-9.]*[0-9])?)`

// rpm: [prefix-]aerospike-server-{edition}[-{tag}]-{version}[{release}].{osTag}.{arch}.rpm
//
//	aerospike-server-community-8.1.3.0-28.amzn2023.aarch64.rpm
//	aerospike-aerospike-server-enterprise-TEST-8.1.3.0-36.ubuntu24.04.aarch64.rpm
//	aerospike-server-enterprise-8.1.3.0_70_g282a6817d-1.el9.x86_64.rpm
var rpmRE = regexp.MustCompile(
	`^(?:.*?-)?aerospike-server-(community|enterprise|federal)` +
		`(?:-[A-Za-z][A-Za-z0-9]*)*-` +
		versionPat + `(.*)\.` +
		osTagPat + `\.` +
		archPat + `\.rpm$`,
)

// deb: [prefix-]aerospike-server-{edition}[-{tag}]_{version}[{release}]{osTag}_{arch}.deb
//
//	aerospike-server-community_8.1.3.0-28debian12_amd64.deb
//	aerospike-server-enterprise_8.1.3.0-28ubuntu24.04_arm64.deb
//	aerospike-aerospike-server-enterprise-TEST_8.1.3.0-36ubuntu24.04_arm64.deb
//	aerospike-server-enterprise_8.1.3.0-70-g282a6817d-1ubuntu24.04_amd64.deb
//	aerospike-server-enterprise-5.7.0.32.ubuntu20.04.x86_64.deb
//
// Note: there is no underscore between {release} and {osTag}, so the release
// field is bounded by "no underscores" rather than by a separator. The 5.7-era
// pipeline used rpm-style dot separators for debs, hence the flexible ones.
var debRE = regexp.MustCompile(
	`^(?:.*?-)?aerospike-server-(community|enterprise|federal)` +
		`(?:-[A-Za-z0-9]+)*[-_]` +
		versionPat + `([^_]*)` +
		osTagPat +
		`[._]` + archPat + `\.deb$`,
)

// ToolsParts is the parsed form of an "aerospike-tools_*.tgz" artifact.
// The tools bundle (asinfo, asadm, aql, …) is edition-agnostic, so unlike
// NameParts it carries no edition/release/format.
type ToolsParts struct {
	Version   string // 11.2.2
	OSName    string // amazon | centos | debian | ubuntu
	OSVersion string // 2023 | 9 | 12 | 24.04
	Arch      string // x86_64 | aarch64
}

// tools: [prefix-]aerospike-tools_{version}[{release}]_{osTag}_{arch}.tgz
//
//	aerospike-tools_11.2.2_ubuntu24.04_aarch64.tgz
//	aerospike-tools_11.2.2_ubuntu24.04_x86_64.tgz
//	aerospike-tools_11.2.2_amzn2023_x86_64.tgz
//	aerospike-tools_11.2.2-70-g282a6817d_ubuntu24.04_x86_64.tgz
//
// The optional prefix mirrors the server parser so double-prefixed CI
// names (e.g. "aerospike-aerospike-tools_...") still match, and the release
// field is tolerated for the same git-describe reason.
var toolsRE = regexp.MustCompile(
	`^(?:.*?-)?aerospike-tools_` +
		`([0-9][0-9.]*)[^_]*_` +
		osTagPat + `_` +
		archPat + `\.tgz$`,
)

// ParseToolsFileName returns the parsed ToolsParts, or nil if the name is
// not an Aerospike tools .tgz.
func ParseToolsFileName(name string) *ToolsParts {
	m := toolsRE.FindStringSubmatch(name)
	if m == nil {
		return nil
	}
	os, ver := splitOSTag(m[2])
	return &ToolsParts{
		Version:   m[1],
		OSName:    os,
		OSVersion: ver,
		Arch:      debArch(m[3]),
	}
}

// ParseFileName returns the parsed NameParts, or nil if the name does not
// match an Aerospike server RPM or DEB.
func ParseFileName(name string) *NameParts {
	if m := rpmRE.FindStringSubmatch(name); m != nil {
		os, ver := splitOSTag(m[4])
		return &NameParts{
			Edition:   m[1],
			Version:   m[2],
			Release:   trimRelease(m[3]),
			OSName:    os,
			OSVersion: ver,
			Arch:      debArch(m[5]),
			Format:    "rpm",
		}
	}
	if m := debRE.FindStringSubmatch(name); m != nil {
		os, ver := splitOSTag(m[4])
		return &NameParts{
			Edition:   m[1],
			Version:   m[2],
			Release:   trimRelease(m[3]),
			OSName:    os,
			OSVersion: ver,
			Arch:      debArch(m[5]),
			Format:    "deb",
		}
	}
	return nil
}

// trimRelease strips the separator that joins the release field to the
// version or the osTag around it, leaving just the release itself ("28",
// "70-g282a6817d-1", "70_g282a6817d-1", or "" when there is none).
func trimRelease(raw string) string {
	return strings.Trim(raw, "-_.~+")
}

// splitOSTag turns a JFrog osTag like "amzn2023" / "debian12" / "ubuntu24.04"
// into the (OSName, OSVersion) pair the rest of aerolab expects.
func splitOSTag(tag string) (osName, osVersion string) {
	switch {
	case strings.HasPrefix(tag, "amzn"):
		return "amazon", strings.TrimPrefix(tag, "amzn")
	case strings.HasPrefix(tag, "el"):
		return "centos", strings.TrimPrefix(tag, "el")
	case strings.HasPrefix(tag, "debian"):
		return "debian", strings.TrimPrefix(tag, "debian")
	case strings.HasPrefix(tag, "ubuntu"):
		return "ubuntu", strings.TrimPrefix(tag, "ubuntu")
	}
	return "", tag
}

// osTag is the inverse of splitOSTag: it renders an (osName, osVersion)
// pair back into the JFrog filename tag ("ubuntu24.04", "amzn2023",
// "el9", "debian12"). Returns "" for OS names JFrog does not publish.
func osTag(osName, osVersion string) string {
	switch osName {
	case "amazon":
		return "amzn" + osVersion
	case "centos", "rocky":
		return "el" + osVersion
	case "debian":
		return "debian" + osVersion
	case "ubuntu":
		return "ubuntu" + osVersion
	}
	return ""
}

// debArch maps Debian's package arch labels to the rpm/aerolab labels so
// the matcher only ever has to think in one vocabulary.
func debArch(in string) string {
	switch strings.ToLower(in) {
	case "amd64", "x86_64":
		return "x86_64"
	case "arm64", "aarch64":
		return "aarch64"
	}
	return in
}

// EditionFromInput extracts the desired edition from a -v string.
//
// The public-download path uses a single trailing 'c' / 'f' to switch
// edition. JFrog build numbers can end with a git SHA whose last hex char
// could legitimately be 'c' or 'f', so we require an explicit ":c", ":f"
// or ":e" separator in JFrog mode and never strip a plain trailing char.
// If neither separator nor env var is present, the caller's `defaultEdition`
// is returned.
func EditionFromInput(version, defaultEdition string) (edition, cleanVersion string) {
	if i := strings.LastIndex(version, ":"); i >= 0 {
		switch version[i+1:] {
		case "c", "community":
			return "community", version[:i]
		case "f", "federal":
			return "federal", version[:i]
		case "e", "enterprise":
			return "enterprise", version[:i]
		}
	}
	if defaultEdition == "" {
		defaultEdition = "enterprise"
	}
	return defaultEdition, version
}
