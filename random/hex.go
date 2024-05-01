package random

import (
	"crypto/rand"
	"fmt"
)

// SecureRandomHexString returns a securely random hex string of length l.
func SecureRandomHexString(l int) (string, error) {
	bytes := make([]byte, l)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", bytes), nil
}
