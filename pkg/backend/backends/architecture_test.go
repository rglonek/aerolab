package backends

import "testing"

func TestArchitectureFromStringAliases(t *testing.T) {
	cases := []struct {
		in   string
		want Architecture
	}{
		{"amd64", ArchitectureX8664},
		{"x86_64", ArchitectureX8664},
		{"x86-64", ArchitectureX8664},
		{"AMD64", ArchitectureX8664},
		{"arm64", ArchitectureARM64},
		{"aarch64", ArchitectureARM64},
		{"ARM64", ArchitectureARM64},
		{"AARCH64", ArchitectureARM64},
		{"native", ArchitectureNative},
		{"default", ArchitectureNative},
	}
	for _, tc := range cases {
		var got Architecture
		if err := got.FromString(tc.in); err != nil {
			t.Fatalf("FromString(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("FromString(%q)=%v, want %v", tc.in, got, tc.want)
		}
	}

	var a Architecture
	if err := a.FromString("ppc64le"); err == nil {
		t.Fatal("expected error for unknown architecture")
	}
}

func TestArchitectureIsARM(t *testing.T) {
	if ArchitectureX8664.IsARM() {
		t.Fatal("x86_64 should not be ARM")
	}
	if !ArchitectureARM64.IsARM() {
		t.Fatal("arm64 should be ARM")
	}
}
