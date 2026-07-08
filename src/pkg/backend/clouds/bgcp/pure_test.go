package bgcp

import (
	"reflect"
	"testing"
)

func TestStringValue(t *testing.T) {
	if got := stringValue(nil); got != "" {
		t.Errorf("stringValue(nil) = %q, want empty", got)
	}
	s := "hello"
	if got := stringValue(&s); got != "hello" {
		t.Errorf("stringValue(&%q) = %q", s, got)
	}
}

func TestGetValueFromURL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"projects/p/zones/us-central1-a", "us-central1-a"},
		{"https://www.googleapis.com/compute/v1/projects/p/global/images/img-1", "img-1"},
		{"single", "single"},
		{"", ""},
	}
	for _, c := range cases {
		if got := getValueFromURL(c.in); got != c.want {
			t.Errorf("getValueFromURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestZoneToRegion(t *testing.T) {
	cases := []struct{ in, want string }{
		{"us-central1-a", "us-central1"},
		{"europe-west1-b", "europe-west1"},
		{"asia-southeast1-c", "asia-southeast1"},
	}
	for _, c := range cases {
		if got := zoneToRegion(c.in); got != c.want {
			t.Errorf("zoneToRegion(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSanitize(t *testing.T) {
	cases := []struct{ in, want string }{
		{"My_Cluster.Name", "my-cluster-name"},
		{"Test123", "test123"},
		{"UPPER", "upper"},
		{"a__b", "a-b"},
		{"123abc", "a123abc"}, // must start with a lowercase letter
		{"", "a"},             // empty becomes a valid single-letter name
	}
	for _, c := range cases {
		if got := sanitize(c.in, false); got != c.want {
			t.Errorf("sanitize(%q,false) = %q, want %q", c.in, got, c.want)
		}
	}
	// withUUID must still start with a letter, contain no "--", and stay within 63 chars.
	u := sanitize("My Cluster", true)
	if len(u) == 0 || u[0] < 'a' || u[0] > 'z' {
		t.Errorf("sanitize withUUID = %q must start with a lowercase letter", u)
	}
	if len(u) > 63 {
		t.Errorf("sanitize withUUID = %q exceeds 63 chars", u)
	}
}

func TestLabelRoundTrip(t *testing.T) {
	m := map[string]string{"foo": "bar", "hello": "world", "num": "42"}
	labels := encodeToLabels(m)
	got, err := decodeFromLabels(labels)
	if err != nil {
		t.Fatalf("decodeFromLabels: %v", err)
	}
	if !reflect.DeepEqual(got, m) {
		t.Errorf("label round-trip = %v, want %v", got, m)
	}
	// Empty label set decodes to nil, no error.
	got, err = decodeFromLabels(map[string]string{})
	if err != nil || got != nil {
		t.Errorf("decodeFromLabels(empty) = (%v,%v), want (nil,nil)", got, err)
	}
}

func TestDescriptionFieldRoundTrip(t *testing.T) {
	m := map[string]string{"a": "1", "b": "2"}
	got, err := decodeFromDescriptionField(encodeToDescriptionField(m))
	if err != nil {
		t.Fatalf("decodeFromDescriptionField: %v", err)
	}
	if !reflect.DeepEqual(got, m) {
		t.Errorf("description round-trip = %v, want %v", got, m)
	}
	if _, err := decodeFromDescriptionField("not json"); err == nil {
		t.Error("expected error decoding invalid JSON description")
	}
}

func TestIntervalToCron(t *testing.T) {
	ok := []struct {
		in   int
		want string
	}{
		{5, "*/5 * * * *"},
		{30, "*/30 * * * *"},
		{60, "0 * * * *"},
		{120, "0 */2 * * *"},
		{1440, "0 1 * * *"},
	}
	for _, c := range ok {
		got, err := intervalToCron(c.in)
		if err != nil || got != c.want {
			t.Errorf("intervalToCron(%d) = (%q,%v), want (%q,nil)", c.in, got, err, c.want)
		}
	}
	for _, bad := range []int{90, 1500} {
		if _, err := intervalToCron(bad); err == nil {
			t.Errorf("intervalToCron(%d) expected error", bad)
		}
	}
}

func TestCronToInterval(t *testing.T) {
	ok := []struct {
		in   string
		want int
	}{
		{"*/5 * * * *", 5},
		{"0 */2 * * *", 120},
		{"0 1 * * *", 1440},
	}
	for _, c := range ok {
		got, err := cronToInterval(c.in)
		if err != nil || got != c.want {
			t.Errorf("cronToInterval(%q) = (%d,%v), want (%d,nil)", c.in, got, err, c.want)
		}
	}
	for _, bad := range []string{"0 * * * *", "bad", "1 2 3"} {
		if _, err := cronToInterval(bad); err == nil {
			t.Errorf("cronToInterval(%q) expected error", bad)
		}
	}
}

func TestUnsanitizeHelpers(t *testing.T) {
	if got := unsanitizeVersion("8-0-0-5"); got != "8.0.0.5" {
		t.Errorf("unsanitizeVersion = %q", got)
	}
	if got := unsanitizeVersion(""); got != "" {
		t.Errorf("unsanitizeVersion(empty) = %q", got)
	}
	if got := unsanitizeCost("0-13402284"); got != "0.13402284" {
		t.Errorf("unsanitizeCost = %q", got)
	}
	if got := unsanitizeExpires("2025-12-06t15_11_15-07_00"); got != "2025-12-06T15:11:15-07:00" {
		t.Errorf("unsanitizeExpires = %q", got)
	}
}

func TestNormalizeAerospikeVersionGCP(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"7.1.0.0c", "7.1.0.0-community"},
		{"7.1.0.0f", "7.1.0.0-federal"},
		{"7.1.0.0", "7.1.0.0-enterprise"},
	}
	for _, c := range cases {
		if got := normalizeAerospikeVersion(c.in); got != c.want {
			t.Errorf("normalizeAerospikeVersion(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
