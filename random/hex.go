package random

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// SecureBytes returns a securely random byte slice of length l.
func SecureBytes(l int) ([]byte, error) {
	bytes := make([]byte, l)
	_, err := rand.Read(bytes)
	if err != nil {
		return bytes, fmt.Errorf("random: error reading random bytes: %w", err)
	}

	return bytes, nil
}

// SecureHexString returns a securely random hex string of length 2*l.
func SecureHexString(l int) (string, error) {
	bytes, err := SecureBytes(l)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}
