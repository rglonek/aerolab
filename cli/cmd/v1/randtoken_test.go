package cmd

import (
	"strings"
	"testing"
)

func TestGenerateRandomTokenShapeAndUniqueness(t *testing.T) {
	const size = 128
	seen := make(map[string]bool, 200)
	for i := 0; i < 200; i++ {
		tok, err := generateRandomToken(size)
		if err != nil {
			t.Fatalf("generateRandomToken returned an error: %s", err)
		}
		if len(tok) != size {
			t.Fatalf("expected a %d character token, got %d", size, len(tok))
		}
		for _, c := range tok {
			if !strings.ContainsRune(tokenAlphabet, c) {
				t.Fatalf("token contains %q, which is outside the alphabet", c)
			}
		}
		if seen[tok] {
			t.Fatalf("generateRandomToken produced a duplicate token after %d draws", i)
		}
		seen[tok] = true
	}
}

// TestGenerateRandomTokenAlphabetCoverage guards against a rejection-sampling
// bug that silently drops part of the alphabet, which would shrink the keyspace
// without changing the token's shape.
func TestGenerateRandomTokenAlphabetCoverage(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 200; i++ {
		tok, err := generateRandomToken(128)
		if err != nil {
			t.Fatalf("generateRandomToken returned an error: %s", err)
		}
		b.WriteString(tok)
	}
	drawn := b.String()
	for _, c := range tokenAlphabet {
		if !strings.ContainsRune(drawn, c) {
			t.Errorf("character %q never appeared in 25600 drawn characters", c)
		}
	}
}

func TestGenerateRandomTokenRejectsNonPositiveSize(t *testing.T) {
	for _, size := range []int{0, -1} {
		if _, err := generateRandomToken(size); err == nil {
			t.Errorf("expected an error for size %d, got none", size)
		}
	}
}

func TestGenerateRandomIDIsHexAndUnique(t *testing.T) {
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		id, err := generateRandomID(8)
		if err != nil {
			t.Fatalf("generateRandomID returned an error: %s", err)
		}
		if len(id) != 16 {
			t.Fatalf("expected 16 hex characters for 8 bytes, got %d", len(id))
		}
		if strings.Trim(id, "0123456789abcdef") != "" {
			t.Fatalf("id %q is not lowercase hex", id)
		}
		if seen[id] {
			t.Fatalf("generateRandomID produced a duplicate after %d draws", i)
		}
		seen[id] = true
	}
}
