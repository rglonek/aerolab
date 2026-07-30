package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// tokenAlphabet is the character set used for generated auth tokens.
const tokenAlphabet = "abcdefghijklmnopqrstuvwxyz1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// generateRandomToken returns a cryptographically secure random token of n
// characters drawn from tokenAlphabet.
//
// Bytes at or above tokenRejectAbove are discarded rather than folded with a
// modulo, so every character in the alphabet is equally likely.
func generateRandomToken(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("token size must be positive, got %d", n)
	}

	// Largest multiple of len(tokenAlphabet) that fits in a byte; bytes at or
	// above it would bias the modulo and are rejected.
	tokenRejectAbove := 256 - (256 % len(tokenAlphabet))

	sb := strings.Builder{}
	sb.Grow(n)

	buf := make([]byte, n)
	for sb.Len() < n {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("could not read random bytes: %w", err)
		}
		for _, b := range buf {
			if int(b) >= tokenRejectAbove {
				continue
			}
			sb.WriteByte(tokenAlphabet[int(b)%len(tokenAlphabet)])
			if sb.Len() == n {
				break
			}
		}
	}

	return sb.String(), nil
}

// generateRandomID returns a random lowercase hex string of nBytes*2
// characters. It is used for token file names, which must not encode the
// creation time: a discoverable mint time narrows the search space for anyone
// attempting to reconstruct a token.
func generateRandomID(nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("could not read random bytes: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
