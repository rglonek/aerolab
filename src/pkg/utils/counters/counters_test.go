package counters

import (
	"sort"
	"sync"
	"testing"
)

func TestIntBasics(t *testing.T) {
	c := NewInt(5)
	if c.Get() != 5 {
		t.Fatalf("NewInt(5).Get() = %d", c.Get())
	}
	c.Inc()
	c.Inc()
	if c.Get() != 7 {
		t.Fatalf("after 2 Inc, Get() = %d, want 7", c.Get())
	}
	c.Dec()
	if c.Get() != 6 {
		t.Fatalf("after Dec, Get() = %d, want 6", c.Get())
	}
	c.Set(100)
	if c.Get() != 100 {
		t.Fatalf("after Set(100), Get() = %d", c.Get())
	}
}

func TestIntConcurrent(t *testing.T) {
	c := NewInt(0)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Inc()
		}()
	}
	wg.Wait()
	if c.Get() != 100 {
		t.Fatalf("concurrent Inc = %d, want 100", c.Get())
	}
}

func TestMapBasics(t *testing.T) {
	m := NewMap()
	m.Inc("a")
	m.Inc("a")
	m.Inc("b")
	if m.Get("a") != 2 {
		t.Errorf("Get(a) = %d, want 2", m.Get("a"))
	}
	if m.Get("missing") != 0 {
		t.Errorf("Get(missing) = %d, want 0", m.Get("missing"))
	}
	m.Dec("b")
	if m.Get("b") != 0 {
		t.Errorf("Get(b) = %d, want 0", m.Get("b"))
	}
	m.Set("c", 9)
	if m.GetTotal() != 2+0+9 {
		t.Errorf("GetTotal = %d, want 11", m.GetTotal())
	}
	if m.GetMapSize() != 3 {
		t.Errorf("GetMapSize = %d, want 3", m.GetMapSize())
	}
	keys := m.GetKeys()
	sort.Strings(keys)
	if len(keys) != 3 || keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Errorf("GetKeys = %v", keys)
	}
}

func TestMapCloneIsIndependent(t *testing.T) {
	m := NewMap()
	m.Set("a", 1)
	clone := m.Clone()
	clone.Set("a", 42)
	if m.Get("a") != 1 {
		t.Errorf("original mutated after clone edit: Get(a) = %d, want 1", m.Get("a"))
	}
	if clone.Get("a") != 42 {
		t.Errorf("clone Get(a) = %d, want 42", clone.Get("a"))
	}

	cp := m.GetMapCopy()
	cp["a"] = 99
	if m.Get("a") != 1 {
		t.Errorf("original mutated after map-copy edit: Get(a) = %d, want 1", m.Get("a"))
	}
}
