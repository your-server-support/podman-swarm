package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

// Token represents a join token
type Token struct {
	Value     string
	Hash      string
	CreatedAt time.Time
	ExpiresAt *time.Time
}

// TokenManager manages join tokens
type TokenManager struct {
	tokens map[string]*Token
	secret []byte
}

// GetSecret returns the secret key (for encryption)
func (tm *TokenManager) GetSecret() []byte {
	return tm.secret
}

// NewTokenManager creates a new token manager
func NewTokenManager(secret []byte) *TokenManager {
	if secret == nil || len(secret) == 0 {
		// Generate a random secret if not provided
		secret = make([]byte, 32)
		rand.Read(secret)
	}

	return &TokenManager{
		tokens: make(map[string]*Token),
		secret: secret,
	}
}

// GenerateToken generates a new join token
func (tm *TokenManager) GenerateToken() (string, error) {
	// Generate random token (similar to Docker Swarm format)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)
	
	// Create hash for validation
	hash := sha256.Sum256(append(tm.secret, tokenBytes...))
	hashStr := hex.EncodeToString(hash[:])

	tm.tokens[token] = &Token{
		Value:     token,
		Hash:      hashStr,
		CreatedAt: time.Now(),
	}

	return token, nil
}

// ValidateToken validates a join token
func (tm *TokenManager) ValidateToken(token string) bool {
	t, ok := tm.tokens[token]
	if !ok {
		// Try to validate against secret
		tokenBytes, err := base64.URLEncoding.DecodeString(token)
		if err != nil {
			return false
		}

		hash := sha256.Sum256(append(tm.secret, tokenBytes...))
		hashStr := hex.EncodeToString(hash[:])

		// Check if hash matches
		for _, storedToken := range tm.tokens {
			if storedToken.Hash == hashStr {
				return true
			}
		}

		// If no tokens exist, allow first node (bootstrap)
		if len(tm.tokens) == 0 {
			return true
		}

		return false
	}

	// Check expiration
	if t.ExpiresAt != nil && time.Now().After(*t.ExpiresAt) {
		delete(tm.tokens, token)
		return false
	}

	return true
}

// RevokeToken revokes a token
func (tm *TokenManager) RevokeToken(token string) {
	delete(tm.tokens, token)
}

// ListTokens returns all active tokens
func (tm *TokenManager) ListTokens() []string {
	tokens := make([]string, 0, len(tm.tokens))
	for token := range tm.tokens {
		tokens = append(tokens, token)
	}
	return tokens
}
