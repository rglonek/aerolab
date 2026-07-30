package sshexec

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// HostKeyEntry is a single remembered SSH host key.
type HostKeyEntry struct {
	// ID is the stable identity the key was learned against, e.g.
	// "aws/<clusterUUID>/3". It deliberately does not include the address:
	// cloud instances change IP across stop/start while remaining the same
	// machine with the same host key.
	ID string `json:"id"`
	// KeyType is the SSH key algorithm, e.g. "ssh-ed25519".
	KeyType string `json:"keyType"`
	// Key is the base64-encoded wire format of the public key.
	Key string `json:"key"`
	// Fingerprint is the SHA256 fingerprint, for display and error messages.
	Fingerprint string `json:"fingerprint"`
	// Host is the address the key was last seen on. Informational only.
	Host string `json:"host"`
	// LearnedAt is when the current key was recorded.
	LearnedAt time.Time `json:"learnedAt"`
	// PreviousFingerprint is the fingerprint this entry replaced, set when a
	// key changed while strict checking was off. It leaves an audit trail of
	// a change that was allowed through with only a warning.
	PreviousFingerprint string `json:"previousFingerprint,omitempty"`
	// ReplacedAt is when PreviousFingerprint was superseded.
	ReplacedAt time.Time `json:"replacedAt,omitempty"`
}

// HostKeyStore remembers SSH host keys on a trust-on-first-use basis.
//
// Keys are indexed by a caller-supplied stable identity rather than by
// hostname, so an instance keeps its entry across address changes, and a
// rebuilt instance that reuses an address does not inherit a stale one.
//
// The store is safe for concurrent use. It re-reads the file under lock before
// every mutation and replaces it via a temporary file plus rename, so parallel
// AeroLab processes do not clobber each other's entries or leave a truncated
// file behind on failure.
type HostKeyStore struct {
	mu   sync.Mutex
	path string
}

// NewHostKeyStore returns a store backed by the given file path. The file and
// its parent directory are created lazily on first write.
func NewHostKeyStore(path string) *HostKeyStore {
	if path == "" {
		return nil
	}
	return &HostKeyStore{path: path}
}

// Path returns the file backing the store.
func (s *HostKeyStore) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// load reads the store file. A missing file is not an error: it just means
// nothing has been learned yet.
func (s *HostKeyStore) load() (map[string]*HostKeyEntry, error) {
	entries := make(map[string]*HostKeyEntry)
	contents, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return entries, nil
		}
		return nil, fmt.Errorf("could not read host key store %s: %w", s.path, err)
	}
	if len(contents) == 0 {
		return entries, nil
	}
	if err := json.Unmarshal(contents, &entries); err != nil {
		return nil, fmt.Errorf("could not parse host key store %s: %w", s.path, err)
	}
	return entries, nil
}

// save writes the store atomically with owner-only permissions.
func (s *HostKeyStore) save(entries map[string]*HostKeyEntry) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("could not create host key store directory %s: %w", dir, err)
	}
	contents, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode host key store: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".known-hosts-*.tmp")
	if err != nil {
		return fmt.Errorf("could not create temporary host key store: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return fmt.Errorf("could not set permissions on host key store: %w", err)
	}
	if _, err := tmp.Write(contents); err != nil {
		tmp.Close()
		return fmt.Errorf("could not write host key store: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("could not close host key store: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("could not replace host key store %s: %w", s.path, err)
	}
	return nil
}

// Lookup returns the remembered entry for id, or nil if none is known.
func (s *HostKeyStore) Lookup(id string) (*HostKeyEntry, error) {
	if s == nil || id == "" {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.load()
	if err != nil {
		return nil, err
	}
	return entries[id], nil
}

// List returns all remembered entries, sorted by ID.
func (s *HostKeyStore) List() ([]*HostKeyEntry, error) {
	if s == nil {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.load()
	if err != nil {
		return nil, err
	}
	out := make([]*HostKeyEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Remember records key as the trusted host key for id, replacing any previous
// entry. previousFingerprint, when non-empty, is retained on the new entry as
// an audit trail of the key that was superseded.
func (s *HostKeyStore) Remember(id string, host string, key ssh.PublicKey, previousFingerprint string) error {
	if s == nil || id == "" || key == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.load()
	if err != nil {
		return err
	}
	entry := &HostKeyEntry{
		ID:                  id,
		KeyType:             key.Type(),
		Key:                 base64.StdEncoding.EncodeToString(key.Marshal()),
		Fingerprint:         ssh.FingerprintSHA256(key),
		Host:                host,
		LearnedAt:           time.Now(),
		PreviousFingerprint: previousFingerprint,
	}
	if previousFingerprint != "" {
		entry.ReplacedAt = entry.LearnedAt
	}
	entries[id] = entry
	return s.save(entries)
}

// Forget drops the remembered keys for the given identities. It is safe to
// call with identities that were never learned.
func (s *HostKeyStore) Forget(ids ...string) error {
	if s == nil || len(ids) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.load()
	if err != nil {
		return err
	}
	changed := false
	for _, id := range ids {
		if _, ok := entries[id]; ok {
			delete(entries, id)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return s.save(entries)
}

// ForgetAll empties the store.
func (s *HostKeyStore) ForgetAll() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save(make(map[string]*HostKeyEntry))
}

// callback returns an ssh.HostKeyCallback that verifies against the store.
//
// An unknown identity is learned and allowed (trust on first use). A key that
// differs from the remembered one is refused when strict is set; otherwise it
// is reported through logf and relearned, so a node rebuilt outside AeroLab
// does not wedge every subsequent command.
func (s *HostKeyStore) callback(id string, strict bool, logf func(string, ...any)) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		known, err := s.Lookup(id)
		if err != nil {
			if strict {
				return fmt.Errorf("could not verify host key for %s: %w", id, err)
			}
			if logf != nil {
				logf("could not read the SSH host key store, skipping verification for %s: %s", id, err)
			}
			return nil
		}

		if known == nil {
			if err := s.Remember(id, hostname, key, ""); err != nil {
				if strict {
					return fmt.Errorf("could not record host key for %s: %w", id, err)
				}
				if logf != nil {
					logf("could not record the SSH host key for %s: %s", id, err)
				}
			}
			return nil
		}

		if known.Fingerprint == ssh.FingerprintSHA256(key) {
			return nil
		}

		newFingerprint := ssh.FingerprintSHA256(key)
		if strict {
			return fmt.Errorf("SSH host key mismatch for %s (%s): expected %s, got %s. If this node was rebuilt or replaced outside AeroLab, run 'aerolab config host-keys forget' for it; otherwise the connection may be intercepted",
				id, hostname, known.Fingerprint, newFingerprint)
		}
		if logf != nil {
			logf("SECURITY: SSH host key for %s (%s) changed from %s to %s. This is expected if the node was rebuilt, and indicates interception otherwise. Enable 'aerolab config backend --ssh-strict-host-key' to refuse the connection instead",
				id, hostname, known.Fingerprint, newFingerprint)
		}
		if err := s.Remember(id, hostname, key, known.Fingerprint); err != nil && logf != nil {
			logf("could not record the new SSH host key for %s: %s", id, err)
		}
		return nil
	}
}
