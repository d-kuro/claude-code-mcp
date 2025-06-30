// Package storage provides credential storage abstractions.
package storage

import (
	"errors"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// ErrTokenNotFound is returned when a token is not found in storage
var ErrTokenNotFound = errors.New("token not found")

// ErrTokenInvalid is returned when a token is invalid or corrupted
var ErrTokenInvalid = errors.New("token is invalid or corrupted")

// ErrStorageUnavailable is returned when storage is temporarily unavailable
var ErrStorageUnavailable = errors.New("storage is temporarily unavailable")

// CredentialStore defines the interface for storing and retrieving OAuth2 credentials
type CredentialStore interface {
	// StoreToken stores an OAuth2 token securely
	StoreToken(token *oauth2.Token) error

	// LoadToken loads an OAuth2 token from storage
	LoadToken() (*oauth2.Token, error)

	// DeleteToken removes an OAuth2 token from storage
	DeleteToken() error

	// HasToken checks if a token exists in storage
	HasToken() bool

	// GetTokenInfo returns basic information about the stored token
	GetTokenInfo() (*TokenInfo, error)

	// Close closes the credential store and performs cleanup
	Close() error
}

// TokenInfo provides basic information about a stored token
type TokenInfo struct {
	AccessToken  string    `json:"access_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
	Email        string    `json:"email,omitempty"`
	IsExpired    bool      `json:"is_expired"`
	ExpiresIn    int64     `json:"expires_in_seconds"`
}

// TokenStore is an alias for CredentialStore for backward compatibility
type TokenStore = CredentialStore

// NewTokenInfo creates a TokenInfo from an OAuth2 token
func NewTokenInfo(token *oauth2.Token) *TokenInfo {
	if token == nil {
		return nil
	}

	info := &TokenInfo{
		AccessToken:  maskToken(token.AccessToken),
		TokenType:    token.TokenType,
		RefreshToken: maskToken(token.RefreshToken),
		Expiry:       token.Expiry,
		IsExpired:    token.Expiry.Before(time.Now()),
	}

	if !info.IsExpired {
		info.ExpiresIn = int64(time.Until(token.Expiry).Seconds())
	}

	return info
}

// maskToken masks a token string for safe display
func maskToken(token string) string {
	if len(token) <= 8 {
		return strings.Repeat("*", len(token))
	}
	return token[:4] + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
}

// IsTokenExpired checks if a token is expired
func IsTokenExpired(token *oauth2.Token) bool {
	if token == nil {
		return true
	}
	return token.Expiry.Before(time.Now())
}

// NeedsRefresh checks if a token needs to be refreshed soon
func NeedsRefresh(token *oauth2.Token, threshold time.Duration) bool {
	if token == nil {
		return true
	}
	return token.Expiry.Before(time.Now().Add(threshold))
}

// ValidateToken performs basic validation on a token
func ValidateToken(token *oauth2.Token) error {
	if token == nil {
		return ErrTokenInvalid
	}

	if token.AccessToken == "" {
		return ErrTokenInvalid
	}

	if token.TokenType == "" {
		token.TokenType = "Bearer" // Default token type
	}

	return nil
}

// CloneToken creates a deep copy of an OAuth2 token
func CloneToken(token *oauth2.Token) *oauth2.Token {
	if token == nil {
		return nil
	}

	clone := &oauth2.Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}

	// Note: Extra fields are not copied as oauth2.Token doesn't provide
	// a way to enumerate all extra keys

	return clone
}
