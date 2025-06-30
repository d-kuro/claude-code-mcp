.PHONY: help split-tools build test clean

# Default target
help:
	@echo "Available targets:"
	@echo "  split-tools  - Split claude-code-tools.md into individual tool files"
	@echo "  build        - Build the project"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean generated files"
	@echo "  help         - Show this help message"

# Split the tools documentation into individual files
split-tools:
	@echo "Splitting claude-code-tools.md into individual tool files..."
	@go run cmd/split-tools/main.go docs/claude-code/claude-code-tools.md internal/prompts/tools
	@echo "Tool files updated successfully"

# Build the project
build:
	@echo "Building claude-code-mcp..."
	@go build -o bin/claude-code-mcp cmd/claude-code-mcp/main.go
	@echo "Build completed successfully"

# Run tests
test:
	@echo "Running tests..."
	@go test ./...

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	@rm -rf internal/prompts/tools/
	@rm -rf bin/
	@echo "Clean completed"

# Ensure tools directory exists and files are up to date
ensure-tools: split-tools

# Development target to rebuild tools and test
dev: split-tools test
	@echo "Development build completed"