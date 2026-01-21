package security

import (
	"testing"
	"time"
)

func TestNewAPITokenManager(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!")
	manager := NewAPITokenManager(secret)

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	if len(manager.secret) != len(secret) {
		t.Errorf("Expected secret length %d, got %d", len(secret), len(manager.secret))
	}
}

func TestNewAPITokenManagerWithNilSecret(t *testing.T) {
	manager := NewAPITokenManager(nil)

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	// Should generate random secret
	if len(manager.secret) != 32 {
		t.Errorf("Expected secret length 32, got %d", len(manager.secret))
	}
}

func TestGenerateToken(t *testing.T) {
	manager := NewAPITokenManager([]byte("test-secret"))

	token, err := manager.GenerateToken("test-token", nil)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("Expected non-empty token")
	}

	// Token should be at least 32 characters (base64 of 24 bytes)
	if len(token) < 32 {
		t.Errorf("Token too short: %d", len(token))
	}
}

func TestValidateToken(t *testing.T) {
	manager := NewAPITokenManager([]byte("test-secret"))

	token, err := manager.GenerateToken("test-token", nil)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Valid token should validate
	if !manager.ValidateToken(token) {
		t.Error("Expected valid token to validate")
	}

	// Invalid token should not validate
	if manager.ValidateToken("invalid-token") {
		t.Error("Expected invalid token to fail validation")
	}

	// Empty token should not validate
	if manager.ValidateToken("") {
		t.Error("Expected empty token to fail validation")
	}
}

func TestRevokeToken(t *testing.T) {
	manager := NewAPITokenManager([]byte("test-secret"))

	token, err := manager.GenerateToken("test-token", nil)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Token should be valid before revocation
	if !manager.ValidateToken(token) {
		t.Error("Expected valid token")
	}

	// Revoke token
	err = manager.RevokeToken(token)
	if err != nil {
		t.Errorf("Failed to revoke token: %v", err)
	}

	// Token should be invalid after revocation
	if manager.ValidateToken(token) {
		t.Error("Expected revoked token to be invalid")
	}
}

func TestListTokens(t *testing.T) {
	manager := NewAPITokenManager([]byte("test-secret"))

	// Create multiple tokens
	_, err := manager.GenerateToken("token1", nil)
	if err != nil {
		t.Fatalf("Failed to generate token1: %v", err)
	}

	_, err = manager.GenerateToken("token2", nil)
	if err != nil {
		t.Fatalf("Failed to generate token2: %v", err)
	}

	tokens := manager.ListTokens()
	if len(tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(tokens))
	}
}

func TestTokenExpiration(t *testing.T) {
	manager := NewAPITokenManager([]byte("test-secret"))

	// Create token that expires in 100ms
	expiresAt := time.Now().Add(100 * time.Millisecond)
	token, err := manager.GenerateToken("expiring-token", &expiresAt)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Should be valid initially
	if !manager.ValidateToken(token) {
		t.Error("Expected token to be valid initially")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be invalid after expiration
	if manager.ValidateToken(token) {
		t.Error("Expected expired token to be invalid")
	}
}

func TestCleanupExpiredTokens(t *testing.T) {
	manager := NewAPITokenManager([]byte("test-secret"))

	// Create expired token
	expiresAt := time.Now().Add(-1 * time.Hour) // Already expired
	token, err := manager.GenerateToken("expired-token", &expiresAt)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Create non-expired token
	_, err = manager.GenerateToken("valid-token", nil)
	if err != nil {
		t.Fatalf("Failed to generate valid token: %v", err)
	}

	// Should have 2 tokens before cleanup
	if len(manager.ListTokens()) != 2 {
		t.Errorf("Expected 2 tokens before cleanup, got %d", len(manager.ListTokens()))
	}

	// Cleanup expired tokens
	manager.CleanupExpiredTokens()

	// Should have 1 token after cleanup (expired one removed)
	tokens := manager.ListTokens()
	if len(tokens) != 1 {
		t.Errorf("Expected 1 token after cleanup, got %d", len(tokens))
	}

	// The remaining token should not be the expired one
	if manager.ValidateToken(token) {
		t.Error("Expired token should have been cleaned up")
	}
}

func TestConcurrentTokenOperations(t *testing.T) {
	manager := NewAPITokenManager([]byte("test-secret"))

	done := make(chan bool)

	// Concurrent token generation
	for i := 0; i < 10; i++ {
		go func(idx int) {
			_, err := manager.GenerateToken("concurrent-token", nil)
			if err != nil {
				t.Errorf("Failed to generate token: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all operations
	for i := 0; i < 10; i++ {
		<-done
	}

	tokens := manager.ListTokens()
	if len(tokens) != 10 {
		t.Errorf("Expected 10 tokens, got %d", len(tokens))
	}
}

func TestTokenMetadata(t *testing.T) {
	manager := NewAPITokenManager([]byte("test-secret"))

	name := "metadata-test"
	expiresAt := time.Now().Add(1 * time.Hour)

	token, err := manager.GenerateToken(name, &expiresAt)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	tokens := manager.ListTokens()
	if len(tokens) != 1 {
		t.Fatalf("Expected 1 token, got %d", len(tokens))
	}

	apiToken := tokens[0]
	if apiToken.Name != name {
		t.Errorf("Expected name %s, got %s", name, apiToken.Name)
	}

	// Verify token can be validated (more important than exact token match)
	if !manager.ValidateToken(token) {
		t.Error("Generated token should validate")
	}

	if apiToken.ExpiresAt == nil {
		t.Error("Expected ExpiresAt to be set")
	}

	if apiToken.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
}

func TestRevokeNonExistentToken(t *testing.T) {
	manager := NewAPITokenManager([]byte("test-secret"))

	err := manager.RevokeToken("non-existent-token")
	if err == nil {
		t.Error("Expected error when revoking non-existent token")
	}
}

func TestTokenHash(t *testing.T) {
	manager := NewAPITokenManager([]byte("test-secret"))

	token1, _ := manager.GenerateToken("token1", nil)
	token2, _ := manager.GenerateToken("token2", nil)

	// Tokens should be different
	if token1 == token2 {
		t.Error("Expected different tokens")
	}

	// Both should validate
	if !manager.ValidateToken(token1) {
		t.Error("Expected token1 to be valid")
	}

	if !manager.ValidateToken(token2) {
		t.Error("Expected token2 to be valid")
	}
}
