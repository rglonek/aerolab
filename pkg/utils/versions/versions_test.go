package versions

import "testing"

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.2.3", "1.2.3", 0},
		{"1.2.3", "1.2.4", -1},
		{"2.0.0", "1.9.9", 1},
		{"v1.2.3", "1.2.3", 0},   // leading v is trimmed
		{"1.2", "1.2.0", -1},     // fewer components sorts older
		{"1.2.0", "1.2", 1},      // more components sorts newer
		{"10.0.0", "9.0.0", 1},   // numeric, not lexical
		{"1.2.3-alpha", "1.2.3-beta", -1}, // prerelease tail compared lexically
		{"1.2.3-beta", "1.2.3-alpha", 1},
	}
	for _, c := range cases {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestLatestOldest(t *testing.T) {
	if got := Latest("1.2.3", "1.3.0"); got != "1.3.0" {
		t.Errorf("Latest = %q, want 1.3.0", got)
	}
	if got := Latest("2.0.0", "1.9.9"); got != "2.0.0" {
		t.Errorf("Latest = %q, want 2.0.0", got)
	}
	if got := Oldest("1.2.3", "1.3.0"); got != "1.2.3" {
		t.Errorf("Oldest = %q, want 1.2.3", got)
	}
	// Equal versions: Latest returns b, Oldest returns a (per implementation).
	if got := Latest("1.0.0", "1.0.0"); got != "1.0.0" {
		t.Errorf("Latest(equal) = %q", got)
	}
}
