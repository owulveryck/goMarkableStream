package jwtutil

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Pre-encoded JWT header for HS256: {"alg":"HS256","typ":"JWT"}
const jwtHeaderBase64 = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"

// Claims represents the JWT payload claims.
type Claims struct {
	Subject   string `json:"sub"`           // Username
	IssuedAt  int64  `json:"iat"`           // Issued at (Unix timestamp)
	ExpiresAt int64  `json:"exp"`           // Expiration time (Unix timestamp)
	TokenID   string `json:"jti,omitempty"` // Unique token ID
	// Role scopes what the bearer may do (see RoleAdmin, RoleFunnel).
	Role string `json:"role,omitempty"`
	// FunnelGen ties a RoleFunnel token to the temporary credentials it was
	// issued for, so tokens die with those credentials.
	FunnelGen uint64 `json:"fgen,omitempty"`
}

const (
	// RoleAdmin is granted to the configured owner credentials.
	RoleAdmin = "admin"
	// RoleFunnel is granted to temporary Funnel viewers: they may watch the
	// stream but not manage the device.
	RoleFunnel = "funnel"
)

// Token represents a validated JWT with its claims.
type Token struct {
	Raw    string
	Claims Claims
}

// ValidationError represents a JWT validation error.
type ValidationError struct {
	Reason string
}

func (e *ValidationError) Error() string {
	return e.Reason
}

var (
	// ErrTokenExpired is returned when the token has expired.
	ErrTokenExpired = &ValidationError{Reason: "token has expired"}
	// ErrInvalidSignature is returned when the signature verification fails.
	ErrInvalidSignature = &ValidationError{Reason: "invalid signature"}
	// ErrMalformedToken is returned when the token format is invalid.
	ErrMalformedToken = &ValidationError{Reason: "malformed token"}
	// ErrInvalidClaims is returned when the claims cannot be decoded.
	ErrInvalidClaims = &ValidationError{Reason: "invalid claims"}
)

// base64URLEncode encodes data using URL-safe base64 without padding.
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// base64URLDecode decodes URL-safe base64 data without padding.
func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// generateTokenID generates a random token ID.
func generateTokenID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateToken creates a new signed JWT token with the admin role.
func CreateToken(subject string, lifetime time.Duration, secret []byte) (string, error) {
	return CreateScopedToken(subject, RoleAdmin, 0, lifetime, secret)
}

// CreateScopedToken creates a new signed JWT token carrying a role and, for
// RoleFunnel tokens, the generation of the credentials it was issued for.
func CreateScopedToken(subject, role string, funnelGen uint64, lifetime time.Duration, secret []byte) (string, error) {
	now := time.Now()

	tokenID, err := generateTokenID()
	if err != nil {
		return "", fmt.Errorf("failed to generate token ID: %w", err)
	}

	claims := Claims{
		Subject:   subject,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(lifetime).Unix(),
		TokenID:   tokenID,
		Role:      role,
		FunnelGen: funnelGen,
	}

	return SignToken(claims, secret)
}

// SignToken signs claims and returns a complete JWT string.
func SignToken(claims Claims, secret []byte) (string, error) {
	// Encode claims as JSON
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to encode claims: %w", err)
	}

	// Base64URL encode the claims
	payloadBase64 := base64URLEncode(claimsJSON)

	// Create the data to sign: header.payload
	signingInput := jwtHeaderBase64 + "." + payloadBase64

	// Sign with HMAC-SHA256
	signature := sign(signingInput, secret)

	// Return the complete token: header.payload.signature
	return signingInput + "." + signature, nil
}

// sign creates an HMAC-SHA256 signature and returns it as base64url.
func sign(data string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(data))
	return base64URLEncode(h.Sum(nil))
}

// ValidateToken validates a JWT token and returns the parsed claims.
func ValidateToken(tokenString string, secret []byte) (*Token, error) {
	// Split the token into parts
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, ErrMalformedToken
	}

	header := parts[0]
	payload := parts[1]
	signature := parts[2]

	// Verify the header matches our expected header
	if header != jwtHeaderBase64 {
		return nil, ErrMalformedToken
	}

	// Verify the signature
	signingInput := header + "." + payload
	expectedSignature := sign(signingInput, secret)

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return nil, ErrInvalidSignature
	}

	// Decode the claims
	claimsJSON, err := base64URLDecode(payload)
	if err != nil {
		return nil, ErrInvalidClaims
	}

	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, ErrInvalidClaims
	}

	// Check expiration
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, ErrTokenExpired
	}

	return &Token{
		Raw:    tokenString,
		Claims: claims,
	}, nil
}

// IsValidationError checks if an error is a JWT validation error.
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}
