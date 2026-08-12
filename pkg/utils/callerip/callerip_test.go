package callerip

import (
	"reflect"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "bare ipv4", value: "1.2.3.4", want: "1.2.3.4/32"},
		{name: "padded", value: "  1.2.3.4  ", want: "1.2.3.4/32"},
		{name: "already a /32", value: "1.2.3.4/32", want: "1.2.3.4/32"},
		{name: "host bits masked off", value: "10.1.2.3/8", want: "10.0.0.0/8"},
		{name: "any", value: "0.0.0.0/0", want: "0.0.0.0/0"},
		{name: "empty", value: "", wantErr: true},
		{name: "garbage", value: "not-an-ip", wantErr: true},
		{name: "bad mask", value: "1.2.3.4/33", wantErr: true},
		{name: "ipv6", value: "2001:db8::1", wantErr: true},
		{name: "ipv6 cidr", value: "2001:db8::/32", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Normalize(test.value)
			if test.wantErr {
				if err == nil {
					t.Fatalf("Normalize(%q) = %q, want error", test.value, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Normalize(%q) returned unexpected error: %s", test.value, err)
			}
			if got != test.want {
				t.Errorf("Normalize(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestParseList(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    []string
		wantErr bool
	}{
		{name: "empty", value: "", want: []string{}},
		{name: "single", value: "1.2.3.4", want: []string{"1.2.3.4/32"}},
		{name: "multiple", value: "1.2.3.4, 10.0.0.0/8", want: []string{"1.2.3.4/32", "10.0.0.0/8"}},
		{name: "deduplicated", value: "1.2.3.4,1.2.3.4/32", want: []string{"1.2.3.4/32"}},
		{name: "discover keyword dropped", value: DiscoverKeyword, want: []string{}},
		{name: "discover keyword among values", value: DiscoverKeyword + ",1.2.3.4", want: []string{"1.2.3.4/32"}},
		{name: "invalid entry", value: "1.2.3.4,nope", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseList(test.value)
			if test.wantErr {
				if err == nil {
					t.Fatalf("ParseList(%q) = %v, want error", test.value, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseList(%q) returned unexpected error: %s", test.value, err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("ParseList(%q) = %v, want %v", test.value, got, test.want)
			}
		})
	}
}

func TestResolvePrefersOverride(t *testing.T) {
	Reset()
	defer Reset()
	if err := SetOverride("203.0.113.7"); err != nil {
		t.Fatalf("SetOverride returned unexpected error: %s", err)
	}
	got, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve returned unexpected error: %s", err)
	}
	want := []string{"203.0.113.7/32"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Resolve() = %v, want %v", got, want)
	}
}

func TestResolvePrefersEnvOverDiscovery(t *testing.T) {
	Reset()
	defer Reset()
	t.Setenv(EnvOverride, "198.51.100.0/24,203.0.113.7")
	got, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve returned unexpected error: %s", err)
	}
	want := []string{"198.51.100.0/24", "203.0.113.7/32"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Resolve() = %v, want %v", got, want)
	}
}

func TestResolveUsesCacheWithoutNetwork(t *testing.T) {
	Reset()
	defer Reset()
	cached = []string{"192.0.2.1/32"}
	got, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve returned unexpected error: %s", err)
	}
	want := []string{"192.0.2.1/32"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Resolve() = %v, want %v", got, want)
	}
}

func TestResolveDoesNotExposeInternalSlice(t *testing.T) {
	Reset()
	defer Reset()
	cached = []string{"192.0.2.1/32"}
	got, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve returned unexpected error: %s", err)
	}
	got[0] = "0.0.0.0/0"
	again, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve returned unexpected error: %s", err)
	}
	if again[0] != "192.0.2.1/32" {
		t.Errorf("mutating the returned slice changed the cache: got %q", again[0])
	}
}
