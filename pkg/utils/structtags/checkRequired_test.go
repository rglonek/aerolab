package structtags

import (
	"testing"
	"time"
)

func TestCheckRequiredNonStruct(t *testing.T) {
	if err := CheckRequired(42); err == nil {
		t.Fatal("expected error for non-struct input")
	}
}

func TestCheckRequiredStrings(t *testing.T) {
	type S struct {
		Name     string `required:"true"`
		Optional string
	}
	if err := CheckRequired(S{Name: "set"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := CheckRequired(S{}); err == nil {
		t.Fatal("expected error for empty required string")
	}
	// Works via pointer too.
	if err := CheckRequired(&S{Name: "set"}); err != nil {
		t.Fatalf("unexpected error via pointer: %v", err)
	}
}

func TestCheckRequiredNumbersAndCollections(t *testing.T) {
	type S struct {
		Count int               `required:"true"`
		Items []string          `required:"true"`
		Meta  map[string]string `required:"true"`
	}
	if err := CheckRequired(S{Count: 1, Items: []string{"a"}, Meta: map[string]string{"k": "v"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := CheckRequired(S{Count: 0, Items: []string{"a"}, Meta: map[string]string{"k": "v"}}); err == nil {
		t.Fatal("expected error for zero required int")
	}
	if err := CheckRequired(S{Count: 1, Items: nil, Meta: map[string]string{"k": "v"}}); err == nil {
		t.Fatal("expected error for nil required slice")
	}
}

func TestCheckRequiredPointer(t *testing.T) {
	type Inner struct {
		V string `required:"true"`
	}
	type S struct {
		Ptr *Inner `required:"true"`
	}
	if err := CheckRequired(S{Ptr: nil}); err == nil {
		t.Fatal("expected error for nil required pointer")
	}
	if err := CheckRequired(S{Ptr: &Inner{V: "x"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// nested required field enforced recursively
	if err := CheckRequired(S{Ptr: &Inner{V: ""}}); err == nil {
		t.Fatal("expected error for empty nested required field")
	}
}

func TestCheckRequiredTime(t *testing.T) {
	type S struct {
		When time.Time `required:"true"`
	}
	if err := CheckRequired(S{}); err == nil {
		t.Fatal("expected error for zero time")
	}
	if err := CheckRequired(S{When: time.Now()}); err != nil {
		t.Fatalf("unexpected error for non-zero time: %v", err)
	}
}
