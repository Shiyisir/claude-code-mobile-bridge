package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// GenerateToken creates a random 32-byte hex token.
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("token: rand: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// SaveToken writes a token to a file with restricted permissions.
func SaveToken(path, token string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("token: mkdir %s: %w", dir, err)
	}
	return os.WriteFile(path, []byte(token), 0600)
}

// LoadToken reads a token from a file.
func LoadToken(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("token: read %s: %w", path, err)
	}
	return string(data), nil
}
