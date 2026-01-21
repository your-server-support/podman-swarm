package security

import (
	"testing"
)

func TestNewTokenManager(t *testing.T) {
	secret := []byte("test-secret-key")
	manager := NewTokenManager(secret)

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}
}

func TestGenerateJoinToken(t *testing.T) {
	manager := NewTokenManager([]byte("test-secret"))

	token, err := manager.GenerateToken()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("Expected non-empty token")
	}

	// Token should be base64 encoded (length varies)
	if len(token) < 20 {
		t.Errorf("Token too short: %d", len(token))
	}
}

func TestValidateJoinToken(t *testing.T) {
	manager := NewTokenManager([]byte("test-secret"))

	token, err := manager.GenerateToken()
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

func TestJoinTokenDeterministic(t *testing.T) {
	secret := []byte("test-secret")
	manager1 := NewTokenManager(secret)
	manager2 := NewTokenManager(secret)

	// Note: Tokens are random, so we can't compare them directly
	// But we can verify that a token from one manager validates with another
	token, err := manager1.GenerateToken()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Token should validate with the same secret
	if !manager2.ValidateToken(token) {
		t.Error("Expected token to validate with same secret")
	}
}

func TestJoinTokenWithDifferentSecrets(t *testing.T) {
	manager1 := NewTokenManager([]byte("secret1"))
	manager2 := NewTokenManager([]byte("secret2"))

	token1, err := manager1.GenerateToken()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Generate a token in manager2 first (so it's not in bootstrap mode)
	_, err = manager2.GenerateToken()
	if err != nil {
		t.Fatalf("Failed to generate token in manager2: %v", err)
	}

	// Now token1 should not validate in manager2 (different secrets, has tokens)
	if manager2.ValidateToken(token1) {
		t.Error("Expected token to fail validation with different manager")
	}
}

func TestMultipleTokenGeneration(t *testing.T) {
	manager := NewTokenManager([]byte("test-secret"))

	tokens := make([]string, 10)
	for i := 0; i < 10; i++ {
		token, err := manager.GenerateToken()
		if err != nil {
			t.Fatalf("Failed to generate token %d: %v", i, err)
		}
		tokens[i] = token
	}

	// All tokens should be different
	seen := make(map[string]bool)
	for _, token := range tokens {
		if seen[token] {
			t.Error("Expected unique tokens")
		}
		seen[token] = true
	}

	// All tokens should validate
	for i, token := range tokens {
		if !manager.ValidateToken(token) {
			t.Errorf("Expected token %d to validate", i)
		}
	}
}

func TestTokenFormatting(t *testing.T) {
	manager := NewTokenManager([]byte("test-secret"))

	token, err := manager.GenerateToken()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Token should be base64 encoded (contains alphanumeric + / + = + -)
	for _, c := range token {
		valid := (c >= '0' && c <= '9') ||
			(c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			c == '+' || c == '/' || c == '=' || c == '-' || c == '_'
		if !valid {
			t.Errorf("Token contains invalid base64 character: %c", c)
		}
	}
}
