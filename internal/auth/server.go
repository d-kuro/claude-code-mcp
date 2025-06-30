// Package auth provides OAuth2 callback server implementation.
package auth

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// AuthResult represents the result of an OAuth2 authentication flow
type AuthResult struct {
	Token *oauth2.Token
	Error error
}

// CallbackServer handles OAuth2 callback requests
type CallbackServer struct {
	port     int
	state    string
	result   chan AuthResult
	server   *http.Server
	config   *oauth2.Config
	mu       sync.Mutex
	started  bool
	shutdown bool
}

// NewCallbackServer creates a new OAuth2 callback server
func NewCallbackServer(port int, state string, config *oauth2.Config) *CallbackServer {
	return &CallbackServer{
		port:   port,
		state:  state,
		result: make(chan AuthResult, 1),
		config: config,
	}
}

// Start starts the OAuth2 callback server
func (s *CallbackServer) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started || s.shutdown {
		return fmt.Errorf("server already started or shutdown")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2callback", s.handleCallback)
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.started = true

	// Start server in a goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.result <- AuthResult{Error: fmt.Errorf("callback server failed: %w", err)}
		}
	}()

	// Wait a moment for the server to start
	time.Sleep(100 * time.Millisecond)

	return nil
}

// Stop gracefully stops the callback server
func (s *CallbackServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started || s.shutdown {
		return nil
	}

	s.shutdown = true

	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown callback server: %w", err)
		}
	}

	return nil
}

// WaitForResult waits for the OAuth2 authentication result
func (s *CallbackServer) WaitForResult(ctx context.Context) (*oauth2.Token, error) {
	select {
	case result := <-s.result:
		return result.Token, result.Error
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// handleCallback handles the OAuth2 callback request
func (s *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := r.URL.Query()

	// Check for error parameter
	if errorParam := query.Get("error"); errorParam != "" {
		errorDesc := query.Get("error_description")
		if errorDesc == "" {
			errorDesc = errorParam
		}

		s.sendResponse(w, "Authentication failed", fmt.Sprintf("Authentication failed: %s", errorDesc), false)
		s.result <- AuthResult{Error: fmt.Errorf("authentication failed: %s", errorDesc)}
		return
	}

	// Validate state parameter (CSRF protection)
	receivedState := query.Get("state")
	if receivedState != s.state {
		s.sendResponse(w, "Authentication failed", "Invalid state parameter", false)
		s.result <- AuthResult{Error: fmt.Errorf("invalid state parameter")}
		return
	}

	// Extract authorization code
	code := query.Get("code")
	if code == "" {
		s.sendResponse(w, "Authentication failed", "No authorization code received", false)
		s.result <- AuthResult{Error: fmt.Errorf("no authorization code received")}
		return
	}

	// Exchange authorization code for access token
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	token, err := s.config.Exchange(ctx, code)
	if err != nil {
		s.sendResponse(w, "Authentication failed", "Failed to exchange authorization code", false)
		s.result <- AuthResult{Error: fmt.Errorf("failed to exchange authorization code: %w", err)}
		return
	}

	// Validate the token
	if err := ValidateToken(ctx, token); err != nil {
		s.sendResponse(w, "Authentication failed", "Token validation failed", false)
		s.result <- AuthResult{Error: fmt.Errorf("token validation failed: %w", err)}
		return
	}

	// Success! Send success response and token result
	s.sendResponse(w, "Authentication successful", "You can now close this window and return to the application", true)
	s.result <- AuthResult{Token: token}
}

// handleHealth handles health check requests
func (s *CallbackServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `{"status": "ok", "port": %d}`, s.port)
}

// sendResponse sends an HTML response to the user
func (s *CallbackServer) sendResponse(w http.ResponseWriter, title, message string, success bool) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	status := http.StatusOK
	if !success {
		status = http.StatusBadRequest
	}
	w.WriteHeader(status)

	// Determine colors based on success/failure
	titleColor := "#28a745"   // Green for success
	messageColor := "#495057" // Dark gray for message
	if !success {
		titleColor = "#dc3545" // Red for error
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>%s</title>
    <meta charset="utf-8">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            max-width: 600px;
            margin: 50px auto;
            padding: 20px;
            background-color: #f8f9fa;
            color: #333;
        }
        .container {
            background: white;
            padding: 40px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            text-align: center;
        }
        h1 {
            color: %s;
            margin-bottom: 20px;
        }
        p {
            color: %s;
            font-size: 16px;
            line-height: 1.5;
        }
        .footer {
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #e9ecef;
            color: #6c757d;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>%s</h1>
        <p>%s</p>
        <div class="footer">
            Claude Code MCP Server - OAuth2 Authentication
        </div>
    </div>
</body>
</html>
`, title, titleColor, messageColor, title, message)

	_, _ = fmt.Fprint(w, html)
}

// GetPort returns the port the server is listening on
func (s *CallbackServer) GetPort() int {
	return s.port
}

// GetState returns the OAuth2 state parameter
func (s *CallbackServer) GetState() string {
	return s.state
}

// IsStarted returns true if the server has been started
func (s *CallbackServer) IsStarted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}

// IsShutdown returns true if the server has been shutdown
func (s *CallbackServer) IsShutdown() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shutdown
}
