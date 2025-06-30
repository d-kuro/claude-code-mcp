// Package storage provides filesystem-based credential storage implementation.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// DefaultConfigDir is the default configuration directory
const DefaultConfigDir = ".claude-code-mcp"

// DefaultCredentialFile is the default credential file name
const DefaultCredentialFile = "oauth_creds.json"

// FileSystemStore implements CredentialStore using filesystem storage
type FileSystemStore struct {
	baseDir     string
	credFile    string
	mu          sync.RWMutex
	cachedToken *oauth2.Token
	cacheTime   time.Time
	cacheTTL    time.Duration
}

// NewFileSystemStore creates a new filesystem-based credential store
func NewFileSystemStore(baseDir string) (*FileSystemStore, error) {
	if baseDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		baseDir = filepath.Join(homeDir, DefaultConfigDir)
	}

	credFile := filepath.Join(baseDir, DefaultCredentialFile)

	store := &FileSystemStore{
		baseDir:  baseDir,
		credFile: credFile,
		cacheTTL: 5 * time.Minute, // Cache tokens for 5 minutes
	}

	// Ensure the directory exists
	if err := store.ensureDir(); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return store, nil
}

// StoreToken stores an OAuth2 token to the filesystem
func (fs *FileSystemStore) StoreToken(token *oauth2.Token) error {
	if token == nil {
		return ErrTokenInvalid
	}

	// Validate token before storing
	if err := ValidateToken(token); err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Ensure directory exists
	if err := fs.ensureDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create a copy of the token with additional metadata
	tokenData := struct {
		*oauth2.Token
		StoredAt time.Time `json:"stored_at"`
		Version  int       `json:"version"`
	}{
		Token:    token,
		StoredAt: time.Now(),
		Version:  1,
	}

	// Marshal token to JSON
	data, err := json.MarshalIndent(tokenData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Write to temporary file first, then rename for atomic operation
	tempFile := fs.credFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, fs.credFile); err != nil {
		// Clean up temp file on error
		_ = os.Remove(tempFile)
		return fmt.Errorf("failed to rename token file: %w", err)
	}

	// Update cache
	fs.cachedToken = CloneToken(token)
	fs.cacheTime = time.Now()

	return nil
}

// LoadToken loads an OAuth2 token from the filesystem
func (fs *FileSystemStore) LoadToken() (*oauth2.Token, error) {
	fs.mu.RLock()

	// Check cache first
	if fs.cachedToken != nil && time.Since(fs.cacheTime) < fs.cacheTTL {
		token := CloneToken(fs.cachedToken)
		fs.mu.RUnlock()
		return token, nil
	}

	fs.mu.RUnlock()

	// Load from file
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Double-check cache after acquiring write lock
	if fs.cachedToken != nil && time.Since(fs.cacheTime) < fs.cacheTTL {
		return CloneToken(fs.cachedToken), nil
	}

	// Read from file
	data, err := os.ReadFile(fs.credFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	// Parse the token data
	var tokenData struct {
		*oauth2.Token
		StoredAt time.Time `json:"stored_at"`
		Version  int       `json:"version"`
	}

	if err := json.Unmarshal(data, &tokenData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	if tokenData.Token == nil {
		return nil, ErrTokenInvalid
	}

	// Validate the loaded token
	if err := ValidateToken(tokenData.Token); err != nil {
		return nil, fmt.Errorf("invalid token in storage: %w", err)
	}

	// Update cache
	fs.cachedToken = CloneToken(tokenData.Token)
	fs.cacheTime = time.Now()

	return CloneToken(tokenData.Token), nil
}

// DeleteToken removes an OAuth2 token from the filesystem
func (fs *FileSystemStore) DeleteToken() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Clear cache
	fs.cachedToken = nil
	fs.cacheTime = time.Time{}

	// Remove file
	if err := os.Remove(fs.credFile); err != nil {
		if os.IsNotExist(err) {
			return ErrTokenNotFound
		}
		return fmt.Errorf("failed to delete token file: %w", err)
	}

	return nil
}

// HasToken checks if a token exists in the filesystem
func (fs *FileSystemStore) HasToken() bool {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Check cache first
	if fs.cachedToken != nil && time.Since(fs.cacheTime) < fs.cacheTTL {
		return true
	}

	// Check file existence (without loading the token to avoid deadlock)
	if _, err := os.Stat(fs.credFile); err != nil {
		return false
	}

	return true
}

// GetTokenInfo returns basic information about the stored token
func (fs *FileSystemStore) GetTokenInfo() (*TokenInfo, error) {
	token, err := fs.LoadToken()
	if err != nil {
		return nil, err
	}

	return NewTokenInfo(token), nil
}

// Close closes the credential store and clears the cache
func (fs *FileSystemStore) Close() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Clear cache
	fs.cachedToken = nil
	fs.cacheTime = time.Time{}

	return nil
}

// ensureDir creates the configuration directory if it doesn't exist
func (fs *FileSystemStore) ensureDir() error {
	if err := os.MkdirAll(fs.baseDir, 0700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", fs.baseDir, err)
	}
	return nil
}

// GetBaseDir returns the base directory for the credential store
func (fs *FileSystemStore) GetBaseDir() string {
	return fs.baseDir
}

// GetCredentialFile returns the path to the credential file
func (fs *FileSystemStore) GetCredentialFile() string {
	return fs.credFile
}

// RefreshCache clears the token cache, forcing a reload from disk
func (fs *FileSystemStore) RefreshCache() {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.cachedToken = nil
	fs.cacheTime = time.Time{}
}

// SetCacheTTL sets the cache time-to-live duration
func (fs *FileSystemStore) SetCacheTTL(ttl time.Duration) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.cacheTTL = ttl
}

// GetCacheTTL returns the current cache time-to-live duration
func (fs *FileSystemStore) GetCacheTTL() time.Duration {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	return fs.cacheTTL
}

// IsCacheValid returns true if the cache is still valid
func (fs *FileSystemStore) IsCacheValid() bool {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	return fs.cachedToken != nil && time.Since(fs.cacheTime) < fs.cacheTTL
}

// GetStats returns statistics about the credential store
func (fs *FileSystemStore) GetStats() map[string]interface{} {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	stats := map[string]interface{}{
		"base_dir":    fs.baseDir,
		"cred_file":   fs.credFile,
		"cache_ttl":   fs.cacheTTL.String(),
		"cache_valid": fs.cachedToken != nil && time.Since(fs.cacheTime) < fs.cacheTTL,
		"file_exists": false,
	}

	if _, err := os.Stat(fs.credFile); err == nil {
		stats["file_exists"] = true
		if info, err := os.Stat(fs.credFile); err == nil {
			stats["file_size"] = info.Size()
			stats["file_modified"] = info.ModTime()
		}
	}

	return stats
}
