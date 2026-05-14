package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/modelcontextprotocol/go-sdk/auth"
)

// TokenClaims represents the JWT claims for MCP access tokens.
type TokenClaims struct {
	jwt.RegisteredClaims
	Scopes []string `json:"scopes"`
}

// TokenManager handles JWT creation and verification.
type TokenManager struct {
	signingKey []byte
	issuer     string
	audience   string
}

// NewTokenManager creates a token manager.
func NewTokenManager(signingKey []byte, issuer, audience string) *TokenManager {
	return &TokenManager{
		signingKey: signingKey,
		issuer:     issuer,
		audience:   audience,
	}
}

// GenerateAccessToken creates a new access token for the given subject and scopes.
func (tm *TokenManager) GenerateAccessToken(subject string, scopes []string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tm.issuer,
			Subject:   subject,
			Audience:  jwt.ClaimStrings{tm.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
		Scopes: scopes,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(tm.signingKey)
}

// VerifyAccessToken validates a token string and returns the parsed claims.
func (tm *TokenManager) VerifyAccessToken(tokenStr string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return tm.signingKey, nil
	}, jwt.WithIssuer(tm.issuer), jwt.WithAudience(tm.audience))
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token claims")
}

// ToAuthTokenInfo converts TokenClaims to SDK TokenInfo.
func (c *TokenClaims) ToAuthTokenInfo() *auth.TokenInfo {
	exp := time.Time{}
	if c.ExpiresAt != nil {
		exp = c.ExpiresAt.Time
	}
	return &auth.TokenInfo{
		Scopes:     c.Scopes,
		Expiration: exp,
		UserID:     c.Subject,
	}
}
