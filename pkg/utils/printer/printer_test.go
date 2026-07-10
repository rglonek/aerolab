package printer

import (
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/v6/table"
)

func TestGetTableWriterSortByValidation(t *testing.T) {
	tests := []struct {
		name    string
		sortBy  []string
		wantErr bool
	}{
		{"no sort", nil, false},
		{"valid asc", []string{"Name:asc"}, false},
		{"valid dscnum", []string{"Count:dscnum"}, false},
		{"missing modifier", []string{"Name"}, true},
		{"too many colons", []string{"Name:asc:extra"}, true},
		{"bad modifier", []string{"Name:sideways"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetTableWriter("TSV", "default", tt.sortBy, true, false)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for sortBy %v, got nil", tt.sortBy)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for sortBy %v: %v", tt.sortBy, err)
			}
		})
	}
}

func TestRenderTableTSV(t *testing.T) {
	tw, err := GetTableWriter("TSV", "default", nil, true, false)
	if err != nil {
		t.Fatalf("GetTableWriter: %v", err)
	}
	out := tw.RenderTable(nil, table.Row{"Name", "Count"}, []table.Row{
		{"alpha", 1},
		{"beta", 2},
	})
	for _, want := range []string{"Name", "Count", "alpha", "beta"} {
		if !strings.Contains(out, want) {
			t.Fatalf("TSV output missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "\t") {
		t.Fatalf("TSV output should be tab-separated:\n%s", out)
	}
}

func TestColorPrintDisabledIsPlain(t *testing.T) {
	c := colorPrint{enable: false}
	if got := c.Sprint("hello", 1); got != "hello1" {
		t.Fatalf("Sprint disabled = %q, want plain fmt.Sprint output", got)
	}
	if got := c.Sprintf("%s-%d", "x", 5); got != "x-5" {
		t.Fatalf("Sprintf disabled = %q, want plain fmt.Sprintf output", got)
	}
}
