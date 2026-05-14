package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CodeData holds the data associated with an authorization code.
type CodeData struct {
	CodeChallenge string
	CodeMethod    string
	RedirectURI   string
	ClientID      string
	ExpiresAt     time.Time
}

// OAuthServer holds the state for our minimal OAuth authorization server.
type OAuthServer struct {
	IssuerURL           string
	ClientID            string
	RedirectURI         string
	AllowedRedirectHost string
	OwnerSecret         string
	TokenManager        *TokenManager
	TokenExpiry         time.Duration
	CodeExpiry          time.Duration

	codes sync.Map // map[string]CodeData
}

// NewOAuthServer creates a new OAuthServer.
func NewOAuthServer(issuerURL, clientID, redirectURI, ownerSecret string, tm *TokenManager, tokenExpiry, codeExpiry time.Duration) *OAuthServer {
	return &OAuthServer{
		IssuerURL:           issuerURL,
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		AllowedRedirectHost: "chatgpt.com",
		OwnerSecret:         ownerSecret,
		TokenManager:        tm,
		TokenExpiry:         tokenExpiry,
		CodeExpiry:          codeExpiry,
	}
}

// GenerateCode creates a new authorization code.
func (s *OAuthServer) GenerateCode(codeChallenge, codeMethod, redirectURI, clientID string) string {
	code := uuid.NewString()
	s.codes.Store(code, CodeData{
		CodeChallenge: codeChallenge,
		CodeMethod:    codeMethod,
		RedirectURI:   redirectURI,
		ClientID:      clientID,
		ExpiresAt:     time.Now().Add(s.CodeExpiry),
	})
	return code
}

// ExchangeCode validates an authorization code and PKCE verifier, then returns an access token.
func (s *OAuthServer) ExchangeCode(code, redirectURI, clientID, codeVerifier string) (string, error) {
	val, ok := s.codes.Load(code)
	if !ok {
		return "", fmt.Errorf("invalid code")
	}
	s.codes.Delete(code) // one-time use

	data := val.(CodeData)
	if time.Now().After(data.ExpiresAt) {
		return "", fmt.Errorf("code expired")
	}
	if data.RedirectURI != redirectURI {
		return "", fmt.Errorf("redirect_uri mismatch")
	}
	if data.ClientID != clientID {
		return "", fmt.Errorf("client_id mismatch")
	}

	// Verify PKCE
	if err := verifyPKCE(data.CodeChallenge, data.CodeMethod, codeVerifier); err != nil {
		return "", err
	}

	token, err := s.TokenManager.GenerateAccessToken("owner", DefaultScopes, s.TokenExpiry)
	if err != nil {
		return "", fmt.Errorf("token generation failed: %w", err)
	}
	return token, nil
}

func verifyPKCE(challenge, method, verifier string) error {
	if method == "" || method == "S256" {
		h := sha256.Sum256([]byte(verifier))
		computed := base64.RawURLEncoding.EncodeToString(h[:])
		if computed != challenge {
			return fmt.Errorf("pkce verification failed")
		}
		return nil
	}
	if method == "plain" {
		if verifier != challenge {
			return fmt.Errorf("pkce verification failed")
		}
		return nil
	}
	return fmt.Errorf("unsupported code_challenge_method: %s", method)
}

// GenerateApproveToken creates a time-limited signed token for the approve link.
// This prevents CSRF/forgery of the approval form.
func (s *OAuthServer) GenerateApproveToken(state string) string {
	msg := fmt.Sprintf("%s|%d", state, time.Now().Unix())
	mac := hmac.New(sha256.New, []byte(s.OwnerSecret))
	mac.Write([]byte(msg))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("%s.%s", msg, sig)
}

// ValidateApproveToken checks if the approve token is valid.
func (s *OAuthServer) ValidateApproveToken(state, token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	msgPart := parts[0]
	sig := parts[1]

	msgParts := strings.Split(msgPart, "|")
	if len(msgParts) != 2 {
		return false
	}
	tokenState := msgParts[0]
	timestampStr := msgParts[1]
	if tokenState != state {
		return false
	}
	ts, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return false
	}
	// Allow 5-minute window
	if time.Now().Unix()-ts > 300 || ts-time.Now().Unix() > 60 {
		return false
	}
	mac := hmac.New(sha256.New, []byte(s.OwnerSecret))
	mac.Write([]byte(msgPart))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expectedSig))
}

// BuildRedirectURL constructs the redirect URL with code and state.
func BuildRedirectURL(base, code, state string) string {
	u, _ := url.Parse(base)
	q := u.Query()
	q.Set("code", code)
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String()
}
