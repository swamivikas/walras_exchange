package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type GeneratedAPIKey struct {
	RawKey    string // Shown ONCE to the user (e.g. walras_live_org_9b1deb4d_7f8a12b...)
	KeyHash   string // SHA-256 hash saved in MySQL & Go Hot RAM
	KeyPrefix string // Formatted string for UI display
}

// HashPassword hashes a plaintext password using bcrypt (Cost: 12)
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword verifies a plaintext password against a bcrypt hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateAPIKey creates a cryptographically secure, scoped API key
func GenerateAPIKey(ownerID string) (*GeneratedAPIKey, error) {
	// 1. Generate 16 secure random bytes (32 hex characters)
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	randomHex := hex.EncodeToString(randomBytes)
	cleanOwnerID := strings.ReplaceAll(ownerID, "-", "")
	if len(cleanOwnerID) > 8 {
		cleanOwnerID = cleanOwnerID[:8]
	}

	// 2. Format: walras_live_org_<shortOwnerID>_<randomHex> (56 chars total)
	rawKey := fmt.Sprintf("walras_live_org_%s_%s", cleanOwnerID, randomHex)

	// 3. Compute SHA-256 hash for database and in-memory matching
	hasher := sha256.New()
	hasher.Write([]byte(rawKey))
	keyHash := hex.EncodeToString(hasher.Sum(nil))

	// 4. Store the complete key so UI Copy & Tunnel Cmd buttons copy the real runnable key!
	keyPrefix := rawKey

	return &GeneratedAPIKey{
		RawKey:    rawKey,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
	}, nil
}
