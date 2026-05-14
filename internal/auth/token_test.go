package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestTokenManager(t *testing.T) {
	tm := NewTokenManager([]byte("test-secret"), "https://example.com", "https://example.com/mcp/")

	token, err := tm.GenerateAccessToken("owner", []string{ScopeRead, ScopeWritePlans}, time.Hour)
	if err != nil {
		t.Fatalf("generate token failed: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}

	claims, err := tm.VerifyAccessToken(token)
	if err != nil {
		t.Fatalf("verify token failed: %v", err)
	}
	if claims.Subject != "owner" {
		t.Errorf("subject = %q, want owner", claims.Subject)
	}
	if len(claims.Scopes) != 2 {
		t.Errorf("scopes len = %d, want 2", len(claims.Scopes))
	}

	// Expired token should fail
	token2, _ := tm.GenerateAccessToken("owner", []string{ScopeRead}, -time.Hour)
	_, err = tm.VerifyAccessToken(token2)
	if err == nil {
		t.Error("expected error for expired token")
	}

	// Bad signature should fail
	_, err = tm.VerifyAccessToken(token + "tampered")
	if err == nil {
		t.Error("expected error for tampered token")
	}
}

func TestTokenClaimsToAuthTokenInfo(t *testing.T) {
	now := time.Now()
	claims := &TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
		Scopes: []string{ScopeRead},
	}
	info := claims.ToAuthTokenInfo()
	if len(info.Scopes) != 1 || info.Scopes[0] != ScopeRead {
		t.Errorf("scopes mismatch")
	}
	if info.UserID != "" {
		t.Errorf("expected empty user id, got %q", info.UserID)
	}
}
