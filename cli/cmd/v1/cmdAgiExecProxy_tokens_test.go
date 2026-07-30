//go:build !noagi

package cmd

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTokensHasMatchesOnlyKnownTokens(t *testing.T) {
	known := strings.Repeat("a", 64)
	other := strings.Repeat("b", 64)

	tk := &tokens{hashes: [][sha256.Size]byte{
		sha256.Sum256([]byte(known)),
		sha256.Sum256([]byte(other)),
	}}

	if !tk.has(known) {
		t.Error("a known token should be accepted")
	}
	if !tk.has(other) {
		t.Error("every known token should be accepted, not just the first")
	}
	if tk.has(strings.Repeat("c", 64)) {
		t.Error("an unknown token should be rejected")
	}
	if tk.has("") {
		t.Error("an empty token should be rejected")
	}
	// A prefix of a valid token must not be accepted; that is exactly the
	// byte-at-a-time guess the constant-time comparison exists to defeat.
	if tk.has(known[:63]) {
		t.Error("a prefix of a valid token should be rejected")
	}
}

func TestTokensHasOnEmptyStore(t *testing.T) {
	tk := &tokens{}
	if tk.has("anything") {
		t.Error("an empty token store should accept nothing")
	}
}

func TestLoadTokensDoStoresHashesNotPlaintext(t *testing.T) {
	dir := t.TempDir()
	token := strings.Repeat("z", 96)
	if err := os.WriteFile(filepath.Join(dir, "tok-abc"), []byte(token), 0600); err != nil {
		t.Fatalf("could not write the test token: %s", err)
	}
	// Files below the minimum length are ignored.
	if err := os.WriteFile(filepath.Join(dir, "tok-short"), []byte("tooshort"), 0600); err != nil {
		t.Fatalf("could not write the short test token: %s", err)
	}

	c := &AgiExecProxyCmd{TokenAuthLocation: dir, tokens: &tokens{}}
	c.loadTokensDo(true)

	if len(c.tokens.hashes) != 1 {
		t.Fatalf("expected exactly one loaded token, got %d", len(c.tokens.hashes))
	}
	if !c.tokens.has(token) {
		t.Error("the loaded token should authenticate")
	}
	if c.tokens.has("tooshort") {
		t.Error("a token file below the minimum length should have been skipped")
	}
}
