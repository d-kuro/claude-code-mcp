// Package web provides web operation tools using the MCP SDK patterns.
package web

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/d-kuro/geminiwebtools"
	"github.com/d-kuro/geminiwebtools/pkg/storage"
	"github.com/d-kuro/geminiwebtools/pkg/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/prompts"
	"github.com/d-kuro/claude-code-mcp/internal/tools"
	"github.com/d-kuro/claude-code-mcp/internal/tools/auth"
)

// WebFetchArgs represents the arguments for the WebFetch tool.
type WebFetchArgs struct {
	URL    string `json:"url"`
	Prompt string `json:"prompt"`
}

// WebSearchArgs represents the arguments for the WebSearch tool.
type WebSearchArgs struct {
	Query          string   `json:"query"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	BlockedDomains []string `json:"blocked_domains,omitempty"`
}

// CreateWebFetchTool creates the WebFetch tool using geminiwebtools library.
func CreateWebFetchTool(ctx *tools.Context) *tools.ServerTool {
	handler := func(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[WebFetchArgs]) (*mcp.CallToolResultFor[any], error) {
		args := params.Arguments

		// Validate URL
		if err := ctx.Validator.ValidateURL(args.URL); err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Invalid URL: " + err.Error()}},
				IsError: true,
			}, nil
		}

		// Validate prompt
		if strings.TrimSpace(args.Prompt) == "" {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Prompt cannot be empty"}},
				IsError: true,
			}, nil
		}

		// Create geminiwebtools client with MCP credential sharing
		credStore, err := createGeminiCredentialStore()
		if err != nil {
			ctx.Logger.WithTool("WebFetch").Error("Failed to create credential store", "error", err)
			return createErrorResponse("Failed to initialize credential store: " + err.Error()), nil
		}

		client, err := geminiwebtools.NewClient(
			geminiwebtools.WithCredentialStore(credStore),
		)
		if err != nil {
			ctx.Logger.WithTool("WebFetch").Error("Failed to create geminiwebtools client", "error", err)
			return createErrorResponse("Failed to initialize web fetch client: " + err.Error()), nil
		}

		// Construct prompt that includes the URL and user's processing instructions
		// This matches the gemini-cli interface expectation
		fetchPrompt := fmt.Sprintf("%s\n\nPlease process the content from: %s", args.Prompt, args.URL)

		// Perform the fetch
		result, err := client.Fetch(ctxReq, fetchPrompt)
		if err != nil {
			ctx.Logger.WithTool("WebFetch").Error("Web fetch failed", "error", err, "url", args.URL)
			return createErrorResponse("Error: " + err.Error()), nil
		}

		// Convert result to MCP response format
		return convertWebFetchResult(result, args), nil
	}

	tool := &mcp.Tool{
		Name:        "WebFetch",
		Description: prompts.WebFetchToolDoc,
	}

	return &tools.ServerTool{
		Tool: tool,
		RegisterFunc: func(server *mcp.Server) {
			mcp.AddTool(server, tool, handler)
		},
	}
}

// CreateWebSearchTool creates the WebSearch tool using geminiwebtools library.
func CreateWebSearchTool(ctx *tools.Context) *tools.ServerTool {
	handler := func(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[WebSearchArgs]) (*mcp.CallToolResultFor[any], error) {
		args := params.Arguments

		// Validate query
		if strings.TrimSpace(args.Query) == "" {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Query cannot be empty"}},
				IsError: true,
			}, nil
		}

		if len(args.Query) < 2 {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Query must be at least 2 characters long"}},
				IsError: true,
			}, nil
		}

		// Create geminiwebtools client with MCP credential sharing
		credStore, err := createGeminiCredentialStore()
		if err != nil {
			ctx.Logger.WithTool("WebSearch").Error("Failed to create credential store", "error", err)
			return createErrorResponse("Failed to initialize credential store: " + err.Error()), nil
		}

		client, err := geminiwebtools.NewClient(
			geminiwebtools.WithCredentialStore(credStore),
		)
		if err != nil {
			ctx.Logger.WithTool("WebSearch").Error("Failed to create geminiwebtools client", "error", err)
			return createErrorResponse("Failed to initialize web search client: " + err.Error()), nil
		}

		// Perform the search
		result, err := client.Search(ctxReq, args.Query)
		if err != nil {
			ctx.Logger.WithTool("WebSearch").Error("Web search failed", "error", err, "query", args.Query)
			return createErrorResponse("Error: " + err.Error()), nil
		}

		// Apply domain filtering as post-processing
		filteredResult := applyDomainFiltering(result, args.AllowedDomains, args.BlockedDomains)

		// Convert result to MCP response format
		return convertWebSearchResult(filteredResult, args), nil
	}

	tool := &mcp.Tool{
		Name:        "WebSearch",
		Description: prompts.WebSearchToolDoc,
	}

	return &tools.ServerTool{
		Tool: tool,
		RegisterFunc: func(server *mcp.Server) {
			mcp.AddTool(server, tool, handler)
		},
	}
}

// Helper functions

// createErrorResponse creates a standardized error response.
func createErrorResponse(message string) *mcp.CallToolResultFor[any] {
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}
}

// convertWebFetchResult converts geminiwebtools WebFetchResult to MCP response format.
func convertWebFetchResult(result *types.WebFetchResult, args WebFetchArgs) *mcp.CallToolResultFor[any] {
	// Create metadata compatible with original implementation
	metadata := map[string]any{
		"url":           args.URL,
		"prompt":        args.Prompt,
		"api_used":      result.Metadata.APIUsed,
		"has_grounding": result.Metadata.HasGrounding,
	}

	// Add conditional metadata fields
	if result.Metadata.ContentType != "" {
		metadata["content_type"] = result.Metadata.ContentType
	}
	if result.Metadata.ContentSize > 0 {
		metadata["content_size"] = result.Metadata.ContentSize
	}
	if result.Metadata.ProcessingTime != "" {
		metadata["processing_time"] = result.Metadata.ProcessingTime
	}
	if result.Metadata.SourceCount > 0 {
		metadata["source_count"] = result.Metadata.SourceCount
	}
	if result.Metadata.SupportCount > 0 {
		metadata["support_count"] = result.Metadata.SupportCount
	}
	if result.Metadata.UsedFallback {
		metadata["used_fallback"] = true
	}

	// Use DisplayText as the main content (includes formatting and citations)
	content := result.DisplayText
	if content == "" {
		content = result.Content
	}
	if content == "" {
		content = "No content received"
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: content}},
		Meta:    metadata,
	}
}

// convertWebSearchResult converts geminiwebtools WebSearchResult to MCP response format.
func convertWebSearchResult(result *types.WebSearchResult, args WebSearchArgs) *mcp.CallToolResultFor[any] {
	// Create metadata compatible with original implementation
	metadata := map[string]any{
		"query":         args.Query,
		"search_region": "US", // Default region
		"has_grounding": result.Metadata.HasGrounding,
		"api_used":      result.Metadata.APIUsed,
	}

	// Add conditional metadata fields
	if result.Metadata.ProcessingTime != "" {
		metadata["processing_time"] = result.Metadata.ProcessingTime
	}
	if result.Metadata.SourceCount > 0 {
		metadata["source_count"] = result.Metadata.SourceCount
	}
	if result.Metadata.SupportCount > 0 {
		metadata["support_count"] = result.Metadata.SupportCount
	}
	if len(result.Metadata.WebSearchQueries) > 0 {
		metadata["web_search_queries"] = result.Metadata.WebSearchQueries
	}
	if len(args.AllowedDomains) > 0 {
		metadata["allowed_domains"] = args.AllowedDomains
	}
	if len(args.BlockedDomains) > 0 {
		metadata["blocked_domains"] = args.BlockedDomains
	}

	// Use DisplayText as the main content (includes formatting and citations)
	content := result.DisplayText
	if content == "" {
		content = result.Content
	}
	if content == "" {
		content = fmt.Sprintf("No search results found for query: %s", args.Query)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: content}},
		Meta:    metadata,
	}
}

// applyDomainFiltering applies domain filtering to search results as post-processing.
func applyDomainFiltering(result *types.WebSearchResult, allowedDomains, blockedDomains []string) *types.WebSearchResult {
	// If no domain filtering is requested, return as-is
	if len(allowedDomains) == 0 && len(blockedDomains) == 0 {
		return result
	}

	// Filter sources based on domain restrictions
	var filteredSources []types.GroundingChunk
	for _, source := range result.Sources {
		if source.Web.URI == "" {
			continue // Skip sources without URI
		}

		// Extract domain from URI
		domain := extractDomain(source.Web.URI)
		if domain == "" {
			continue // Skip if we can't extract domain
		}

		// Check blocked domains first
		if isBlocked(domain, blockedDomains) {
			continue
		}

		// If allowed domains specified, check if domain is allowed
		if len(allowedDomains) > 0 && !isAllowed(domain, allowedDomains) {
			continue
		}

		filteredSources = append(filteredSources, source)
	}

	// Create a new result with filtered sources
	filteredResult := *result // Copy the result
	filteredResult.Sources = filteredSources

	// Update metadata to reflect filtering
	filteredResult.Metadata.SourceCount = len(filteredSources)
	filteredResult.Metadata.AllowedDomains = allowedDomains
	filteredResult.Metadata.BlockedDomains = blockedDomains

	// If we filtered out all sources, add a note
	if len(result.Sources) > 0 && len(filteredSources) == 0 {
		filteredResult.DisplayText += "\n\n**Note:** All search results were filtered out by domain restrictions."
	} else if len(filteredSources) < len(result.Sources) {
		removedCount := len(result.Sources) - len(filteredSources)
		filteredResult.DisplayText += fmt.Sprintf("\n\n**Note:** %d search result(s) were filtered out by domain restrictions.", removedCount)
	}

	return &filteredResult
}

// extractDomain extracts the domain from a URL.
func extractDomain(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsedURL.Hostname())
}

// isBlocked checks if a domain is in the blocked list.
func isBlocked(domain string, blockedDomains []string) bool {
	for _, blocked := range blockedDomains {
		normalizedBlocked := strings.ToLower(blocked)
		if domain == normalizedBlocked || strings.HasSuffix(domain, "."+normalizedBlocked) {
			return true
		}
	}
	return false
}

// isAllowed checks if a domain is in the allowed list.
func isAllowed(domain string, allowedDomains []string) bool {
	for _, allowed := range allowedDomains {
		normalizedAllowed := strings.ToLower(allowed)
		if domain == normalizedAllowed || strings.HasSuffix(domain, "."+normalizedAllowed) {
			return true
		}
	}
	return false
}

// createGeminiCredentialStore creates a geminiwebtools credential store
// using the same directory as the MCP server for credential sharing
func createGeminiCredentialStore() (storage.CredentialStore, error) {
	return storage.NewFileSystemStore(auth.GetDefaultConfigDir())
}
