package bdocker

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/registry"
)

func TestImageNaming(t *testing.T) {
	cases := []struct {
		distro, version, arch, want string
	}{
		{"ubuntu", "22.04", "amd64", "amd64/ubuntu:22.04"},
		{"ubuntu", "22.04", "arm64", "arm64v8/ubuntu:22.04"},
		{"ubuntu", "26.04", "amd64", "amd64/ubuntu:26.04"},
		{"ubuntu", "26.04", "arm64", "arm64v8/ubuntu:26.04"},
		{"debian", "12", "amd64", "amd64/debian:12"},
		{"ubuntu", "22.04", "", "ubuntu:22.04"}, // fallthrough to default
		{"rocky", "9", "amd64", "amd64/rockylinux:9"},
		{"rocky", "9", "arm64", "arm64v8/rockylinux:9"},
		{"rocky", "9", "", "rockylinux:9"},
		// The arch-prefixed rockylinux repos stop at 9; 10+ is multi-arch only.
		{"rocky", "10", "amd64", "rockylinux/rockylinux:10"},
		{"rocky", "10", "arm64", "rockylinux/rockylinux:10"},
		{"rocky", "10", "", "rockylinux/rockylinux:10"},
		{"amazon", "2023", "amd64", "amd64/amazonlinux:2023"},
		{"amazon", "2023", "arm64", "arm64v8/amazonlinux:2023"},
		{"centos", "6", "amd64", "quay.io/centos/centos:6"},
		{"centos", "7", "arm64", "quay.io/centos/centos:7"},
		{"centos", "9", "amd64", "quay.io/centos/amd64:stream9"},
		{"centos", "9", "arm64", "quay.io/centos/arm64v8:stream9"},
		{"centos", "9", "", "quay.io/centos/centos:stream9"},
		{"centos", "10", "amd64", "quay.io/centos/amd64:stream10"},
		{"centos", "10", "arm64", "quay.io/centos/arm64v8:stream10"},
		{"alpine", "3.19", "amd64", "alpine:3.19"}, // unknown distro -> default
	}
	for _, c := range cases {
		if got := ImageNaming(c.distro, c.version, c.arch); got != c.want {
			t.Errorf("ImageNaming(%q,%q,%q) = %q, want %q", c.distro, c.version, c.arch, got, c.want)
		}
	}
}

func TestComputeAccessURL(t *testing.T) {
	ports := []container.PortSummary{
		{PrivatePort: 8080, PublicPort: 49153},
		{PrivatePort: 3000, PublicPort: 0}, // not published
	}
	cases := []struct {
		clientType string
		ports      []container.PortSummary
		want       string
	}{
		{"", ports, ""},
		{"unknown", ports, ""},
		{"vscode", ports, "http://localhost:49153"},
		{"ams", ports, ""},   // mapped port has PublicPort 0
		{"graph", ports, ""}, // no matching private port
		{"vscode", nil, ""},
	}
	for _, c := range cases {
		if got := computeAccessURL(c.clientType, c.ports); got != c.want {
			t.Errorf("computeAccessURL(%q, %v) = %q, want %q", c.clientType, c.ports, got, c.want)
		}
	}
}

func TestEncodeAuthToBase64(t *testing.T) {
	auth := registry.AuthConfig{Username: "user", Password: "pass", ServerAddress: "ghcr.io"}
	got, err := encodeAuthToBase64(auth)
	if err != nil {
		t.Fatalf("encodeAuthToBase64: %v", err)
	}
	decoded, err := base64.URLEncoding.DecodeString(got)
	if err != nil {
		t.Fatalf("result is not valid base64url: %v", err)
	}
	var back registry.AuthConfig
	if err := json.Unmarshal(decoded, &back); err != nil {
		t.Fatalf("decoded payload is not valid JSON auth: %v", err)
	}
	if back.Username != auth.Username || back.Password != auth.Password || back.ServerAddress != auth.ServerAddress {
		t.Errorf("round-trip mismatch: got %+v, want %+v", back, auth)
	}
}
