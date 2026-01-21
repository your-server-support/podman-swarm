package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// APIToken represents an API authentication token
type APIToken struct {
	Token     string
	Hash      string
	Name      string
	CreatedAt time.Time
	ExpiresAt *time.Time
}

// APITokenManager manages API authentication tokens
type APITokenManager struct {
	mu     sync.RWMutex
	tokens map[string]*APIToken // token -> APIToken
	hashes map[string]string    // hash -> token (for fast lookup)
	secret []byte
}

// NewAPITokenManager creates a new API token manager
func NewAPITokenManager(secret []byte) *APITokenManager {
	if secret == nil || len(secret) == 0 {
		// Generate a random secret if not provided
		secret = make([]byte, 32)
		rand.Read(secret)
	}

	return &APITokenManager{
		tokens: make(map[string]*APIToken),
		hashes: make(map[string]string),
		secret: secret,
	}
}

// GenerateToken generates a new API token
func (tm *APITokenManager) GenerateToken(name string, expiresAt *time.Time) (string, error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)
	
	// Create hash for storage
	hash := sha256.Sum256(append(tm.secret, tokenBytes...))
	hashStr := hex.EncodeToString(hash[:])

	apiToken := &APIToken{
		Token:     token,
		Hash:      hashStr,
		Name:      name,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.tokens[token] = apiToken
	tm.hashes[hashStr] = token

	return token, nil
}

// ValidateToken validates an API token
func (tm *APITokenManager) ValidateToken(token string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	t, ok := tm.tokens[token]
	if !ok {
		return false
	}

	// Check expiration
	if t.ExpiresAt != nil && time.Now().After(*t.ExpiresAt) {
		return false
	}

	return true
}

// RevokeToken revokes an API token
func (tm *APITokenManager) RevokeToken(token string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	t, ok := tm.tokens[token]
	if !ok {
		return fmt.Errorf("token not found")
	}

	delete(tm.hashes, t.Hash)
	delete(tm.tokens, token)

	return nil
}

// ListTokens returns all active tokens
func (tm *APITokenManager) ListTokens() []*APIToken {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tokens := make([]*APIToken, 0, len(tm.tokens))
	for _, token := range tm.tokens {
		// Don't include actual token value in list
		tokens = append(tokens, &APIToken{
			Token:     "***", // Masked
			Hash:      token.Hash,
			Name:      token.Name,
			CreatedAt: token.CreatedAt,
			ExpiresAt: token.ExpiresAt,
		})
	}

	return tokens
}

// CleanupExpiredTokens removes expired tokens
func (tm *APITokenManager) CleanupExpiredTokens() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()
	for token, t := range tm.tokens {
		if t.ExpiresAt != nil && now.After(*t.ExpiresAt) {
			delete(tm.hashes, t.Hash)
			delete(tm.tokens, token)
		}
	}
}

// StartCleanupRoutine starts a routine that periodically cleans up expired tokens
func (tm *APITokenManager) StartCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			tm.CleanupExpiredTokens()
		}
	}()
}
