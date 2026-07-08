package file

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreJSONRoundTrip(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "nested", "deeper", "data.json")

	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	in := payload{Name: "aerolab", Count: 42}

	if err := StoreJSON(name, ".tmp", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644, in); err != nil {
		t.Fatalf("StoreJSON: %v", err)
	}

	raw, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("reading stored file: %v", err)
	}
	var out payload
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Fatalf("round trip mismatch: got %+v, want %+v", out, in)
	}

	if _, err := os.Stat(name + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temp file should be renamed away, stat err = %v", err)
	}
}

func TestStoreJSONOverwrites(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "data.json")

	if err := StoreJSON(name, ".tmp", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644, map[string]int{"v": 1}); err != nil {
		t.Fatalf("first StoreJSON: %v", err)
	}
	if err := StoreJSON(name, ".tmp", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644, map[string]int{"v": 2}); err != nil {
		t.Fatalf("second StoreJSON: %v", err)
	}

	raw, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	var out map[string]int
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out["v"] != 2 {
		t.Fatalf("overwrite failed: got %v, want v=2", out)
	}
}
