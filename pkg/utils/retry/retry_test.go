package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWithRetrySucceedsFirstTry(t *testing.T) {
	calls := 0
	got, err := WithRetrySimple(3, time.Millisecond, func() (int, error) {
		calls++
		return 42, nil
	})
	if err != nil || got != 42 {
		t.Fatalf("got (%d,%v), want (42,nil)", got, err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestWithRetryEventuallySucceeds(t *testing.T) {
	calls := 0
	got, err := WithRetrySimple(5, time.Millisecond, func() (string, error) {
		calls++
		if calls < 3 {
			return "", errors.New("transient")
		}
		return "ok", nil
	})
	if err != nil || got != "ok" {
		t.Fatalf("got (%q,%v), want (ok,nil)", got, err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestWithRetryExhausts(t *testing.T) {
	calls := 0
	sentinel := errors.New("always")
	_, err := WithRetrySimple(2, time.Millisecond, func() (int, error) {
		calls++
		return 0, sentinel
	})
	if err == nil || !errors.Is(err, sentinel) {
		t.Fatalf("expected wrapped sentinel error, got %v", err)
	}
	if calls != 3 { // 1 initial + 2 retries
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestWithRetryShouldRetryFalseStops(t *testing.T) {
	calls := 0
	cfg := &Config{
		MaxRetries:  5,
		RetrySleep:  time.Millisecond,
		ShouldRetry: func(err error) bool { return false },
	}
	_, err := WithRetry(cfg, func() (int, error) {
		calls++
		return 0, errors.New("fatal")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("expected 1 call when ShouldRetry=false, got %d", calls)
	}
}

func TestWithRetryContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cfg := &Config{MaxRetries: 3, RetrySleep: time.Second, Context: ctx}
	_, err := WithRetry(cfg, func() (int, error) { return 0, errors.New("x") })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestCombineErrors(t *testing.T) {
	if err := CombineErrors(nil, nil); err != nil {
		t.Fatalf("CombineErrors(nil,nil) = %v, want nil", err)
	}
	e1 := errors.New("one")
	e2 := errors.New("two")
	err := CombineErrors(e1, nil, e2)
	if !errors.Is(err, e1) || !errors.Is(err, e2) {
		t.Fatalf("combined error missing components: %v", err)
	}
}

func TestRetryState(t *testing.T) {
	s := NewRetryState()
	if !s.ShouldRetryCapacity(2) {
		t.Fatal("expected capacity retry available")
	}
	s.IncrementCapacityRetry()
	s.IncrementCapacityRetry()
	if s.ShouldRetryCapacity(2) {
		t.Fatal("expected capacity exhausted")
	}
	s.IncrementTransientRetry()
	if s.TransientRetriesUsed != 1 {
		t.Fatalf("TransientRetriesUsed = %d, want 1", s.TransientRetriesUsed)
	}
	s.Reset()
	if s.CapacityRetriesUsed != 0 || s.TransientRetriesUsed != 0 {
		t.Fatal("Reset did not clear counters")
	}
}
