package cache

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type sample struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestCacheStoreGetRoundTrip(t *testing.T) {
	c := &Cache{Enabled: true, Dir: filepath.Join(t.TempDir(), "cache")}
	in := sample{Name: "clusterA", Count: 3}
	if err := c.Store("inventory", in); err != nil {
		t.Fatalf("Store: %v", err)
	}
	var out sample
	if err := c.Get("inventory", &out); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if out != in {
		t.Fatalf("round trip mismatch: got %+v, want %+v", out, in)
	}
}

func TestCacheGetMissing(t *testing.T) {
	c := &Cache{Enabled: true, Dir: filepath.Join(t.TempDir(), "cache")}
	var out sample
	if err := c.Get("nope", &out); !errors.Is(err, ErrNoCacheFile) {
		t.Fatalf("Get missing = %v, want ErrNoCacheFile", err)
	}
}

func TestCacheDelete(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cache")
	c := &Cache{Enabled: true, Dir: dir}
	if err := c.Store("x", sample{Name: "y"}); err != nil {
		t.Fatalf("Store: %v", err)
	}
	if err := c.Delete(); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("Delete should remove dir, stat err = %v", err)
	}
}

func TestCacheDisabledIsNoOp(t *testing.T) {
	for _, c := range []*Cache{
		nil,
		{Enabled: false, Dir: filepath.Join(t.TempDir(), "d")},
		{Enabled: true, Dir: ""},
	} {
		if err := c.Store("k", sample{Name: "z"}); err != nil {
			t.Fatalf("disabled Store returned %v, want nil", err)
		}
		out := sample{Name: "unchanged"}
		if err := c.Get("k", &out); err != nil {
			t.Fatalf("disabled Get returned %v, want nil", err)
		}
		if out.Name != "unchanged" {
			t.Fatalf("disabled Get mutated output: %+v", out)
		}
		if err := c.Delete(); err != nil {
			t.Fatalf("disabled Delete returned %v, want nil", err)
		}
	}
}
