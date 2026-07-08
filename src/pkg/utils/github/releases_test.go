package github

import (
	"testing"
	"time"
)

func sampleReleases() Releases {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return Releases{
		{TagName: "v1.0.0", Prerelease: false, PublishedAt: base},
		{TagName: "v1.1.0-beta", Prerelease: true, PublishedAt: base.Add(48 * time.Hour)},
		{TagName: "v1.1.0", Prerelease: false, PublishedAt: base.Add(72 * time.Hour)},
		{TagName: "other-2.0.0", Prerelease: false, PublishedAt: base.Add(24 * time.Hour)},
	}
}

func TestReleasesWithTag(t *testing.T) {
	r := sampleReleases()
	got := r.WithTag("v1.1.0")
	if got == nil {
		t.Fatal("WithTag returned nil for existing tag")
	}
	if got.TagName != "v1.1.0" {
		t.Fatalf("WithTag returned %q, want v1.1.0", got.TagName)
	}
	if r.WithTag("does-not-exist") != nil {
		t.Fatal("WithTag should return nil for missing tag")
	}
	var empty Releases
	if empty.WithTag("x") != nil {
		t.Fatal("WithTag on empty should return nil")
	}
}

func TestReleasesWithTagPrefix(t *testing.T) {
	r := sampleReleases()
	got := r.WithTagPrefix("v1.")
	if len(got) != 3 {
		t.Fatalf("WithTagPrefix returned %d releases, want 3", len(got))
	}
	for _, rel := range got {
		if rel.TagName[:3] != "v1." {
			t.Fatalf("unexpected tag %q in v1. prefix result", rel.TagName)
		}
	}
	if len(r.WithTagPrefix("nope-")) != 0 {
		t.Fatal("WithTagPrefix should return empty for no matches")
	}
}

func TestReleasesWithPrerelease(t *testing.T) {
	r := sampleReleases()
	pre := r.WithPrerelease(true)
	if len(pre) != 1 || pre[0].TagName != "v1.1.0-beta" {
		t.Fatalf("WithPrerelease(true) = %+v, want single beta", pre)
	}
	stable := r.WithPrerelease(false)
	if len(stable) != 3 {
		t.Fatalf("WithPrerelease(false) returned %d, want 3", len(stable))
	}
}

func TestReleasesLatest(t *testing.T) {
	r := sampleReleases()
	latest := r.Latest()
	if latest == nil {
		t.Fatal("Latest returned nil")
	}
	if latest.TagName != "v1.1.0" {
		t.Fatalf("Latest = %q, want v1.1.0 (most recent PublishedAt)", latest.TagName)
	}
	var empty Releases
	if empty.Latest() != nil {
		t.Fatal("Latest on empty should return nil")
	}
}

func sampleAssets() Assets {
	return Assets{
		{Name: "aerolab-linux-amd64.zip", ContentType: "application/zip"},
		{Name: "aerolab-linux-arm64.zip", ContentType: "application/zip"},
		{Name: "aerolab-macos.pkg", ContentType: "application/octet-stream"},
		{Name: "checksums.txt", ContentType: "text/plain"},
	}
}

func TestAssetsSelectors(t *testing.T) {
	a := sampleAssets()

	if got := a.WithName("aerolab-macos.pkg"); got == nil || got.Name != "aerolab-macos.pkg" {
		t.Fatalf("WithName mismatch: %+v", got)
	}
	if a.WithName("missing") != nil {
		t.Fatal("WithName should return nil for missing asset")
	}
	if got := a.WithNamePrefix("aerolab-linux"); got == nil || len(*got) != 2 {
		t.Fatalf("WithNamePrefix returned %v, want 2", got)
	}
	if got := a.WithNameSuffix(".zip"); got == nil || len(*got) != 2 {
		t.Fatalf("WithNameSuffix returned %v, want 2", got)
	}
	if got := a.WithNameContains("arm64"); got == nil || len(*got) != 1 {
		t.Fatalf("WithNameContains returned %v, want 1", got)
	}
	if got := a.WithNamePattern(`aerolab-linux-(amd|arm)64\.zip`); got == nil || len(*got) != 2 {
		t.Fatalf("WithNamePattern returned %v, want 2", got)
	}
	if got := a.WithContentType("text/plain"); got == nil || len(*got) != 1 {
		t.Fatalf("WithContentType returned %v, want 1", got)
	}
}

func TestAssetsListAndFirst(t *testing.T) {
	a := sampleAssets()
	if len(a.List()) != 4 {
		t.Fatalf("List returned %d, want 4", len(a.List()))
	}
	if f := a.First(); f == nil || f.Name != "aerolab-linux-amd64.zip" {
		t.Fatalf("First = %+v, want first asset", f)
	}
	var empty Assets
	if empty.List() != nil {
		t.Fatal("List on empty should return nil")
	}
	if empty.First() != nil {
		t.Fatal("First on empty should return nil")
	}
}
