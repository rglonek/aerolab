package parallelize

import (
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestForEachRunsAll(t *testing.T) {
	var count int64
	ForEach([]int{1, 2, 3, 4, 5}, func(int) {
		atomic.AddInt64(&count, 1)
	})
	if count != 5 {
		t.Fatalf("ForEach visited %d items, want 5", count)
	}
}

func TestForEachEmpty(t *testing.T) {
	called := false
	ForEach([]int{}, func(int) { called = true })
	if called {
		t.Fatal("ForEach should not call fn on empty slice")
	}
}

func TestMapPreservesOrder(t *testing.T) {
	in := []int{1, 2, 3, 4}
	out := Map(in, func(x int) int { return x * x })
	want := []int{1, 4, 9, 16}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("Map out[%d] = %d, want %d (out=%v)", i, out[i], want[i], out)
		}
	}
}

func TestMapLimitPreservesOrder(t *testing.T) {
	in := []int{5, 4, 3, 2, 1}
	out := MapLimit(in, 2, func(x int) string {
		time.Sleep(time.Millisecond)
		return string(rune('a' + x))
	})
	want := []string{"f", "e", "d", "c", "b"}
	if len(out) != len(want) {
		t.Fatalf("MapLimit len = %d, want %d", len(out), len(want))
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("MapLimit out[%d] = %q, want %q", i, out[i], want[i])
		}
	}
}

func TestForEachLimitRespectsConcurrency(t *testing.T) {
	const limit = 3
	var (
		cur, max int64
		mu       sync.Mutex
	)
	items := make([]int, 30)
	ForEachLimit(items, limit, func(int) {
		n := atomic.AddInt64(&cur, 1)
		mu.Lock()
		if n > max {
			max = n
		}
		mu.Unlock()
		time.Sleep(2 * time.Millisecond)
		atomic.AddInt64(&cur, -1)
	})
	if max > limit {
		t.Fatalf("observed concurrency %d exceeded limit %d", max, limit)
	}
	if max == 0 {
		t.Fatal("fn never ran")
	}
}

func TestForEachLimitRunsAll(t *testing.T) {
	var results []int
	var mu sync.Mutex
	ForEachLimit([]int{1, 2, 3, 4, 5, 6}, 2, func(x int) {
		mu.Lock()
		results = append(results, x)
		mu.Unlock()
	})
	sort.Ints(results)
	want := []int{1, 2, 3, 4, 5, 6}
	if len(results) != len(want) {
		t.Fatalf("ForEachLimit ran %d items, want %d", len(results), len(want))
	}
	for i := range want {
		if results[i] != want[i] {
			t.Fatalf("results[%d] = %d, want %d", i, results[i], want[i])
		}
	}
}
