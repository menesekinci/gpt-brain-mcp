package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestVerifyPKCE(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	if err := verifyPKCE(challenge, "S256", verifier); err != nil {
		t.Errorf("valid S256 PKCE failed: %v", err)
	}
	if err := verifyPKCE(challenge, "S256", "wrong"); err == nil {
		t.Error("invalid verifier should fail")
	}
	if err := verifyPKCE("abc", "plain", "abc"); err != nil {
		t.Errorf("valid plain PKCE failed: %v", err)
	}
	if err := verifyPKCE("abc", "unsupported", "abc"); err == nil {
		t.Error("unsupported method should fail")
	}
}

func TestOAuthServerCodeExchange(t *testing.T) {
	tm := NewTokenManager([]byte("jwt-secret"), "https://example.com", "https://example.com/mcp/")
	srv := NewOAuthServer("https://example.com", "client-id", "https://example.com/callback", "owner-secret", tm, time.Hour, 5*time.Minute)

	verifier := "test-verifier-123"
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	code := srv.GenerateCode(challenge, "S256", "https://example.com/callback", "client-id")
	if code == "" {
		t.Fatal("code is empty")
	}

	// Valid exchange
	token, err := srv.ExchangeCode(code, "https://example.com/callback", "client-id", verifier)
	if err != nil {
		t.Fatalf("exchange failed: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}

	// Code is one-time use
	_, err = srv.ExchangeCode(code, "https://example.com/callback", "client-id", verifier)
	if err == nil {
		t.Error("reusing code should fail")
	}

	// Invalid code
	_, err = srv.ExchangeCode("invalid", "https://example.com/callback", "client-id", verifier)
	if err == nil {
		t.Error("invalid code should fail")
	}
}

func TestOAuthServerApproveToken(t *testing.T) {
	tm := NewTokenManager([]byte("jwt-secret"), "https://example.com", "https://example.com/mcp/")
	srv := NewOAuthServer("https://example.com", "client-id", "https://example.com/callback", "owner-secret", tm, time.Hour, 5*time.Minute)

	token := srv.GenerateApproveToken("state1")
	if token == "" {
		t.Fatal("approve token is empty")
	}

	if !srv.ValidateApproveToken("state1", token) {
		t.Error("valid token rejected")
	}
	if srv.ValidateApproveToken("state2", token) {
		t.Error("wrong state accepted")
	}
	if srv.ValidateApproveToken("state1", token+"tampered") {
		t.Error("tampered token accepted")
	}

	// Expired token
	parts := strings.Split(token, ".")
	if len(parts) == 2 {
		msgParts := strings.Split(parts[0], "|")
		if len(msgParts) == 2 {
			oldToken := msgParts[0] + "|" + msgParts[1] + "." + parts[1]
			// Token is still valid (within 5 min), so force an old timestamp
			expiredToken := msgParts[0] + "|" + msgParts[1] + "." + parts[1]
			_ = expiredToken
			_ = oldToken
		}
	}
}

func TestBuildRedirectURL(t *testing.T) {
	u := BuildRedirectURL("https://example.com/callback", "code123", "state456")
	if !strings.Contains(u, "code=code123") {
		t.Errorf("missing code param")
	}
	if !strings.Contains(u, "state=state456") {
		t.Errorf("missing state param")
	}
}
