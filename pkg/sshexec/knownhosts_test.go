package sshexec

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/ssh"
)

func testPublicKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("could not generate a test key: %s", err)
	}
	_ = priv
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("could not convert the test key: %s", err)
	}
	return sshPub
}

func newTestStore(t *testing.T) *HostKeyStore {
	t.Helper()
	return NewHostKeyStore(filepath.Join(t.TempDir(), "sub", "known-hosts.json"))
}

func TestHostKeyStoreLearnsOnFirstUse(t *testing.T) {
	store := newTestStore(t)
	key := testPublicKey(t)
	cb := store.callback("aws/uuid-1/1", false, nil)

	if err := cb("10.0.0.1:22", &net.TCPAddr{}, key); err != nil {
		t.Fatalf("first connection should be allowed, got: %s", err)
	}

	entry, err := store.Lookup("aws/uuid-1/1")
	if err != nil {
		t.Fatalf("Lookup returned an error: %s", err)
	}
	if entry == nil {
		t.Fatal("expected the key to be remembered after the first connection")
	}
	if entry.Fingerprint != ssh.FingerprintSHA256(key) {
		t.Errorf("stored fingerprint %q does not match the presented key", entry.Fingerprint)
	}
	if entry.Host != "10.0.0.1:22" {
		t.Errorf("stored host = %q, want %q", entry.Host, "10.0.0.1:22")
	}

	// The same key on a later connection is accepted unchanged.
	if err := cb("10.0.0.9:22", &net.TCPAddr{}, key); err != nil {
		t.Fatalf("a matching key should be allowed, got: %s", err)
	}
}

func TestHostKeyStoreMismatchWarnsAndRelearnsByDefault(t *testing.T) {
	store := newTestStore(t)
	original := testPublicKey(t)
	replacement := testPublicKey(t)

	var warnings []string
	cb := store.callback("gcp/uuid-2/3", false, func(format string, args ...any) {
		warnings = append(warnings, format)
	})

	if err := cb("host", &net.TCPAddr{}, original); err != nil {
		t.Fatalf("first connection should be allowed, got: %s", err)
	}
	if err := cb("host", &net.TCPAddr{}, replacement); err != nil {
		t.Fatalf("a changed key should be allowed in non-strict mode, got: %s", err)
	}
	if len(warnings) == 0 {
		t.Error("expected a warning when the host key changed")
	}

	entry, err := store.Lookup("gcp/uuid-2/3")
	if err != nil {
		t.Fatalf("Lookup returned an error: %s", err)
	}
	if entry.Fingerprint != ssh.FingerprintSHA256(replacement) {
		t.Error("the new key should have been relearned")
	}
	if entry.PreviousFingerprint != ssh.FingerprintSHA256(original) {
		t.Error("the superseded fingerprint should be retained as an audit trail")
	}
	if entry.ReplacedAt.IsZero() {
		t.Error("ReplacedAt should be set when a key is superseded")
	}
}

func TestHostKeyStoreMismatchFailsInStrictMode(t *testing.T) {
	store := newTestStore(t)
	original := testPublicKey(t)
	replacement := testPublicKey(t)

	learn := store.callback("docker/uuid-3/1", false, nil)
	if err := learn("host", &net.TCPAddr{}, original); err != nil {
		t.Fatalf("first connection should be allowed, got: %s", err)
	}

	strict := store.callback("docker/uuid-3/1", true, nil)
	err := strict("host", &net.TCPAddr{}, replacement)
	if err == nil {
		t.Fatal("strict mode should refuse a changed host key")
	}
	if !strings.Contains(err.Error(), ssh.FingerprintSHA256(original)) || !strings.Contains(err.Error(), ssh.FingerprintSHA256(replacement)) {
		t.Errorf("the error should name both fingerprints, got: %s", err)
	}

	// The refused key must not be recorded.
	entry, err := store.Lookup("docker/uuid-3/1")
	if err != nil {
		t.Fatalf("Lookup returned an error: %s", err)
	}
	if entry.Fingerprint != ssh.FingerprintSHA256(original) {
		t.Error("strict mode must not overwrite the remembered key")
	}
}

func TestHostKeyStoreForget(t *testing.T) {
	store := newTestStore(t)
	keyA, keyB := testPublicKey(t), testPublicKey(t)

	if err := store.Remember("aws/uuid/1", "h1", keyA, ""); err != nil {
		t.Fatalf("Remember returned an error: %s", err)
	}
	if err := store.Remember("aws/uuid/2", "h2", keyB, ""); err != nil {
		t.Fatalf("Remember returned an error: %s", err)
	}

	if err := store.Forget("aws/uuid/1", "aws/uuid/never-learned"); err != nil {
		t.Fatalf("Forget returned an error: %s", err)
	}
	if e, _ := store.Lookup("aws/uuid/1"); e != nil {
		t.Error("the forgotten entry should be gone")
	}
	if e, _ := store.Lookup("aws/uuid/2"); e == nil {
		t.Error("Forget removed an entry it was not asked to remove")
	}

	// After forgetting, the identity is learned fresh rather than mismatching.
	replacement := testPublicKey(t)
	if err := store.callback("aws/uuid/1", true, nil)("h1", &net.TCPAddr{}, replacement); err != nil {
		t.Fatalf("a forgotten identity should be relearned even in strict mode, got: %s", err)
	}

	if err := store.ForgetAll(); err != nil {
		t.Fatalf("ForgetAll returned an error: %s", err)
	}
	entries, err := store.List()
	if err != nil {
		t.Fatalf("List returned an error: %s", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected an empty store after ForgetAll, got %d entries", len(entries))
	}
}

func TestHostKeyStoreListIsSorted(t *testing.T) {
	store := newTestStore(t)
	for _, id := range []string{"aws/u/3", "aws/u/1", "aws/u/2"} {
		if err := store.Remember(id, "h", testPublicKey(t), ""); err != nil {
			t.Fatalf("Remember returned an error: %s", err)
		}
	}
	entries, err := store.List()
	if err != nil {
		t.Fatalf("List returned an error: %s", err)
	}
	want := []string{"aws/u/1", "aws/u/2", "aws/u/3"}
	for i, e := range entries {
		if e.ID != want[i] {
			t.Errorf("entry %d = %q, want %q", i, e.ID, want[i])
		}
	}
}

func TestHostKeyStoreFilePermissions(t *testing.T) {
	store := newTestStore(t)
	if err := store.Remember("aws/u/1", "h", testPublicKey(t), ""); err != nil {
		t.Fatalf("Remember returned an error: %s", err)
	}

	info, err := os.Stat(store.Path())
	if err != nil {
		t.Fatalf("could not stat the store: %s", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("store file mode = %o, want 0600", perm)
	}

	dirInfo, err := os.Stat(filepath.Dir(store.Path()))
	if err != nil {
		t.Fatalf("could not stat the store directory: %s", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0700 {
		t.Errorf("store directory mode = %o, want 0700", perm)
	}
}

// TestHostKeyStoreConcurrentRemember checks that parallel writers do not
// corrupt the file or lose entries; the store is shared by every SSH
// connection AeroLab opens, and those run concurrently.
func TestHostKeyStoreConcurrentRemember(t *testing.T) {
	store := newTestStore(t)
	const workers = 16

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := "aws/u/" + string(rune('a'+n))
			if err := store.Remember(id, "h", testPublicKey(t), ""); err != nil {
				t.Errorf("Remember(%s) returned an error: %s", id, err)
			}
		}(i)
	}
	wg.Wait()

	entries, err := store.List()
	if err != nil {
		t.Fatalf("List returned an error: %s", err)
	}
	if len(entries) != workers {
		t.Errorf("expected %d entries after concurrent writes, got %d", workers, len(entries))
	}
}

func TestNilHostKeyStoreIsInert(t *testing.T) {
	var store *HostKeyStore
	if store.Path() != "" {
		t.Error("a nil store should report an empty path")
	}
	if e, err := store.Lookup("id"); e != nil || err != nil {
		t.Error("a nil store should return nothing from Lookup")
	}
	if err := store.Remember("id", "h", testPublicKey(t), ""); err != nil {
		t.Errorf("Remember on a nil store should be a no-op, got: %s", err)
	}
	if err := store.Forget("id"); err != nil {
		t.Errorf("Forget on a nil store should be a no-op, got: %s", err)
	}
	if NewHostKeyStore("") != nil {
		t.Error("NewHostKeyStore with an empty path should return nil")
	}
}

// TestMakeClientConfigHostKeyPolicy checks that verification is only enabled
// when both the store and the identity are present, so connections to external
// hosts AeroLab does not manage keep working.
func TestMakeClientConfigHostKeyPolicy(t *testing.T) {
	store := newTestStore(t)
	key := testPublicKey(t)
	if err := store.Remember("aws/u/1", "h", key, ""); err != nil {
		t.Fatalf("Remember returned an error: %s", err)
	}

	verified, err := makeClientConfig(&ClientConf{Username: "root", HostKeyStore: store, HostKeyID: "aws/u/1", HostKeyStrict: true})
	if err != nil {
		t.Fatalf("makeClientConfig returned an error: %s", err)
	}
	if err := verified.HostKeyCallback("h", &net.TCPAddr{}, testPublicKey(t)); err == nil {
		t.Error("expected the configured callback to reject a mismatched key")
	}

	// No identity: external host, unverified as before.
	unverified, err := makeClientConfig(&ClientConf{Username: "root", HostKeyStore: store})
	if err != nil {
		t.Fatalf("makeClientConfig returned an error: %s", err)
	}
	if err := unverified.HostKeyCallback("h", &net.TCPAddr{}, testPublicKey(t)); err != nil {
		t.Errorf("expected no verification without a host key ID, got: %s", err)
	}
}
