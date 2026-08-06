package jfrog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// File is a single artifact resolved from a JFrog build.
type File struct {
	Repo        string
	Path        string
	Name        string
	Size        int64
	SHA1        string
	Created     time.Time
	DownloadURL string
	Parts       *NameParts // nil for signatures, source archives, etc.
}

// Files is a list of File with picker helpers.
type Files []File

// MatchCriteria describes the install target we want a package for.
//
// OSName  : "amazon" | "centos" | "debian" | "ubuntu" (post-translation)
// OSVersion: e.g. "2023", "9", "12", "24.04"
// Arch    : "x86_64" | "aarch64" (after debArch normalisation)
// Edition : "community" | "enterprise" | "federal"
type MatchCriteria struct {
	OSName    string
	OSVersion string
	Arch      string
	Edition   string
}

// Match returns the single File matching the criteria. The choice of
// package format (rpm vs deb) is implied by OSName: amazon/centos use
// rpm, debian/ubuntu use deb.
func (fs Files) Match(c MatchCriteria) (*File, error) {
	wantFormat := formatForOS(c.OSName)
	if wantFormat == "" {
		return nil, fmt.Errorf("jfrog: unsupported OS %q (only amazon/centos/debian/ubuntu have JFrog packages)", c.OSName)
	}

	for i := range fs {
		f := &fs[i]
		if f.Parts == nil {
			continue
		}
		if f.Parts.Format != wantFormat {
			continue
		}
		if f.Parts.Edition != c.Edition {
			continue
		}
		if f.Parts.OSName != c.OSName {
			continue
		}
		if f.Parts.OSVersion != c.OSVersion {
			continue
		}
		if f.Parts.Arch != c.Arch {
			continue
		}
		return f, nil
	}
	return nil, fs.noMatchError(c, wantFormat)
}

// diagSampleSize caps how many artifact names a failed match reports. A build
// carries a few hundred artifacts; a sample is enough to tell a genuinely
// missing package apart from a naming scheme the parser does not know.
const diagSampleSize = 10

// noMatchError explains a failed Match. The artifact list is fetched, filtered
// and discarded within a single call, so this error is the only place an
// operator gets to see what the build actually carried — without it, an
// upstream naming change is indistinguishable from a missing package.
func (fs Files) noMatchError(c MatchCriteria, wantFormat string) error {
	var candidates, unrecognised []string
	for i := range fs {
		f := &fs[i]
		if f.Parts == nil {
			// signatures and checksums are expected not to parse; only
			// package-looking names say anything about the naming scheme
			if isPackageName(f.Name) {
				unrecognised = append(unrecognised, f.Name)
			}
			continue
		}
		if f.Parts.Edition == c.Edition && f.Parts.Format == wantFormat {
			candidates = append(candidates, f.Parts.OSName+f.Parts.OSVersion+"/"+f.Parts.Arch)
		}
	}
	if len(candidates) > 0 {
		return fmt.Errorf(
			"jfrog: no %s %s package matches %s%s/%s; this build has %s %s packages only for: %s",
			c.Edition, wantFormat, c.OSName, c.OSVersion, c.Arch,
			c.Edition, wantFormat, strings.Join(uniq(candidates), ", "))
	}
	if len(unrecognised) == 0 {
		for i := range fs {
			unrecognised = append(unrecognised, fs[i].Name)
		}
	}
	return fmt.Errorf(
		"jfrog: no %s %s package found for %s%s/%s; none of the %d artifacts on this build "+
			"parsed as an aerospike server package. Either the build publishes a naming scheme "+
			"aerolab does not recognise yet, or your JFrog credentials cannot read the repository "+
			"holding the package (AQL silently omits unreadable artifacts). Artifacts seen: %s",
		c.Edition, wantFormat, c.OSName, c.OSVersion, c.Arch, len(fs),
		sample(unrecognised, diagSampleSize))
}

// isPackageName reports whether a name looks like an installable package
// rather than a signature, checksum or source archive.
func isPackageName(name string) bool {
	return strings.HasSuffix(name, ".deb") || strings.HasSuffix(name, ".rpm")
}

// uniq removes duplicates while preserving order. Builds publish the same
// artifact into several repositories, so raw lists repeat themselves.
func uniq(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// sample renders at most max entries, noting how many were elided.
func sample(in []string, max int) string {
	in = uniq(in)
	if len(in) == 0 {
		return "(none)"
	}
	if len(in) <= max {
		return strings.Join(in, ", ")
	}
	return fmt.Sprintf("%s, ... (+%d more)", strings.Join(in[:max], ", "), len(in)-max)
}

// MatchTools returns the "aerospike-tools_*.tgz" artifact matching the OS
// and architecture in c. Edition and package format are ignored — the tools
// bundle is edition-agnostic and always shipped as a .tgz. Returns nil when
// the build has no matching tools package, so callers can fall back to a
// server-only install (and warn the operator).
func (fs Files) MatchTools(c MatchCriteria) *File {
	for i := range fs {
		f := &fs[i]
		tp := ParseToolsFileName(f.Name)
		if tp == nil {
			continue
		}
		if tp.OSName != c.OSName || tp.OSVersion != c.OSVersion || tp.Arch != c.Arch {
			continue
		}
		return f
	}
	return nil
}

// LatestToolsFile searches the whole JFrog instance (not just the current
// build) for the most recently created "aerospike-tools_*.tgz" matching the
// given OS and architecture. It is the second-preference fallback used when
// the resolved build has no matching tools artifact of its own. Returns
// nil (no error) when nothing matches.
func (c *Config) LatestToolsFile(ctx context.Context, osName, osVersion, arch string) (*File, error) {
	if c == nil {
		return nil, fmt.Errorf("jfrog: nil config")
	}
	tag := osTag(osName, osVersion)
	if tag == "" {
		return nil, fmt.Errorf("jfrog: unsupported OS %q for tools lookup", osName)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout())
		defer cancel()
	}

	glob := fmt.Sprintf("aerospike-tools_*_%s_%s.tgz", tag, arch)
	query := fmt.Sprintf(
		`items.find({"name":{"$match":"%s"}}).include("repo","path","name","size","actual_sha1","created").sort({"$desc":["created"]}).limit(50)`,
		jsonEscape(glob),
	)
	raw, err := c.AQL(ctx, query)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results []struct {
			Repo    string    `json:"repo"`
			Path    string    `json:"path"`
			Name    string    `json:"name"`
			Size    int64     `json:"size"`
			SHA1    string    `json:"actual_sha1"`
			Created time.Time `json:"created"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("jfrog: parse tools AQL response: %w", err)
	}
	// Results are created-desc; return the first that actually parses and
	// matches (the glob is a coarse filter, ParseToolsFileName is exact).
	for _, r := range resp.Results {
		tp := ParseToolsFileName(r.Name)
		if tp == nil || tp.OSName != osName || tp.OSVersion != osVersion || tp.Arch != arch {
			continue
		}
		return &File{
			Repo:        r.Repo,
			Path:        r.Path,
			Name:        r.Name,
			Size:        r.Size,
			SHA1:        r.SHA1,
			Created:     r.Created,
			DownloadURL: c.ArtifactoryURL("/" + r.Repo + "/" + r.Path + "/" + r.Name),
			Parts:       ParseFileName(r.Name),
		}, nil
	}
	return nil, nil
}

// formatForOS returns the package format JFrog publishes for a given OS.
func formatForOS(osName string) string {
	switch osName {
	case "amazon", "centos", "rocky":
		return "rpm"
	case "debian", "ubuntu":
		return "deb"
	}
	return ""
}
