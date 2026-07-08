package diff

import (
	"strings"
	"testing"
)

func TestDiffIdenticalIsNil(t *testing.T) {
	in := []byte("line1\nline2\nline3\n")
	if got := Diff("a", in, "b", in); got != nil {
		t.Fatalf("Diff of identical input = %q, want nil", got)
	}
}

func TestDiffReportsChanges(t *testing.T) {
	old := []byte("alpha\nbravo\ncharlie\n")
	updated := []byte("alpha\nBRAVO\ncharlie\n")
	out := string(Diff("old.txt", old, "new.txt", updated))
	if out == "" {
		t.Fatal("expected non-empty diff")
	}
	for _, want := range []string{"--- old.txt", "+++ new.txt", "-bravo", "+BRAVO"} {
		if !strings.Contains(out, want) {
			t.Errorf("diff output missing %q:\n%s", want, out)
		}
	}
}

func TestDiffAddedAndRemovedLines(t *testing.T) {
	old := []byte("keep\nremove-me\nkeep2\n")
	updated := []byte("keep\nkeep2\nadd-me\n")
	out := string(Diff("o", old, "n", updated))
	if !strings.Contains(out, "-remove-me") {
		t.Errorf("expected removed line marker, got:\n%s", out)
	}
	if !strings.Contains(out, "+add-me") {
		t.Errorf("expected added line marker, got:\n%s", out)
	}
}
