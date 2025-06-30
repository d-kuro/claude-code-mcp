# Claude Code MCP Server

A Model Context Protocol (MCP) server that exposes Claude Code's built-in tools as MCP tools, enabling external applications to access file operations, system commands, web functionality, and task management through the standardized MCP interface.

## Features

- **Complete Claude Code Tool Compatibility**: All 16 Claude Code tools implemented with identical functionality
- **Zero Configuration**: No configuration required - all tools available by default
- **Built-in Security**: Path validation and command sanitization for safe operation
- **Multiple Transport Support**: Stdio, HTTP/SSE, and in-memory transports
- **High Performance**: Optimized implementations using native system commands

## Quick Start

### Installation

#### From Binary

Download the latest release for your platform:

```bash
# Linux
curl -L https://github.com/d-kuro/claude-code-mcp/releases/latest/download/claude-code-mcp-linux-amd64 -o claude-code-mcp
chmod +x claude-code-mcp

# macOS
curl -L https://github.com/d-kuro/claude-code-mcp/releases/latest/download/claude-code-mcp-darwin-amd64 -o claude-code-mcp
chmod +x claude-code-mcp

# Windows
curl -L https://github.com/d-kuro/claude-code-mcp/releases/latest/download/claude-code-mcp-windows-amd64.exe -o claude-code-mcp.exe
```

#### From Source

```bash
# Requires Go 1.23+
git clone https://github.com/d-kuro/claude-code-mcp
cd claude-code-mcp
go build -o claude-code-mcp ./cmd/claude-code-mcp
```

### Basic Usage

**All tools are available immediately - no configuration needed:**

```bash
# Download and run
./claude-code-mcp
```

That's it! All Claude Code tools are now available through the MCP interface with built-in security.

## Available Tools

### File Operations
- **Read**: Read file contents with line offset/limit support
- **Write**: Write files with atomic operations and directory creation
- **Edit**: String replacement with automatic backup
- **MultiEdit**: Multiple edits in a single atomic operation
- **LS**: List directory contents using native `ls` command
- **Glob**: Pattern-based file discovery using `find`
- **Grep**: Content search using `ripgrep` (`rg`)

### System Operations
- **Bash**: Execute commands with persistent shell sessions
- **Task**: Launch agents for complex multi-step operations

### Extended Tools
- **NotebookRead/NotebookEdit**: Jupyter notebook support
- **WebFetch**: Fetch and process web content
- **WebSearch**: Web search with domain filtering
- **TodoRead/TodoWrite**: Session-based task management
- **exit_plan_mode**: Development workflow support

## Configuration

**No configuration required!** All tools work out of the box.

### Optional Environment Variables

You can customize logging if needed:

```bash
# Set log level (debug, info, warn, error)
export LOG_LEVEL=debug
./claude-code-mcp
```

### Built-in Security Features

- **Path Validation**: All file paths are validated and sanitized
- **Command Sanitization**: Dangerous commands and patterns are blocked
- **URL Validation**: Web requests are validated with basic checks
- **Resource Limits**: File size, output size, and timeout limits enforced
- **Session Isolation**: Todo tasks are isolated per MCP session

## Docker Deployment

```bash
# Build image
docker build -t claude-code-mcp .

# Run with configuration
docker run -v ~/.config/claude-code-mcp:/root/.config/claude-code-mcp \
           -v ~/projects:/projects \
           claude-code-mcp

# Run with HTTP transport
docker run -p 8080:8080 \
           -v ~/.config/claude-code-mcp:/root/.config/claude-code-mcp \
           claude-code-mcp --http :8080
```

## Integration Examples

### With Claude Desktop

Add to your Claude Desktop configuration (`mcp.json`):

```json
{
  "mcpServers": {
    "claude-code-mcp": {
      "command": "/path/to/claude-code-mcp",
      "args": [],
      "env": {
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

See [MCP Configuration Guide](docs/mcp-configuration.md) for detailed configuration options.

### With Custom MCP Client

```go
import (
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Create client
client := mcp.NewClient()

// Connect to server
transport := mcp.NewStdioTransport()
err := client.Connect(transport)

// Call a tool
result, err := client.CallTool(ctx, "Read", map[string]any{
    "file_path": "/path/to/file.txt",
})
```

## Development

### Building from Source

```bash
# Clone repository
git clone https://github.com/d-kuro/claude-code-mcp
cd claude-code-mcp

# Install dependencies
go mod download

# Build
go build -o claude-code-mcp ./cmd/claude-code-mcp

# Run tests
go test ./...

# Run with race detector
go test -race ./...
```

### Running Tests

```bash
# Unit tests
go test ./...

# Integration tests
go test ./internal/integration/...

# Specific package tests
go test ./internal/tools/file/...

# With coverage
go test -cover ./...
```

### Project Structure

```
claude-code-mcp/
├── cmd/claude-code-mcp/     # Main server entry point
├── pkg/                     # Public packages
│   ├── config/             # Configuration management
│   └── errors/            # Error types
├── internal/              # Private implementation
│   ├── server/           # MCP server core
│   ├── tools/           # Tool implementations
│   │   ├── file/       # File operations
│   │   ├── bash/       # System operations
│   │   ├── web/        # Web tools
│   │   ├── notebook/   # Jupyter support
│   │   ├── todo/       # Task management
│   │   └── workflow/   # Workflow tools
│   ├── security/       # Security validation
│   └── logging/        # Logging infrastructure
└── examples/           # Usage examples
```

## Troubleshooting

### Common Issues

**Q: Are all tools really available by default?**
A: Yes! All 16 Claude Code tools work immediately without any configuration.

**Q: What security measures are in place?**
A: Built-in path validation, command sanitization, and resource limits protect your system.

**Q: Can I use relative paths?**
A: No, all paths must be absolute for security reasons.

**Q: How do I enable debug logging?**
A: Set the LOG_LEVEL environment variable:
```bash
export LOG_LEVEL=debug
./claude-code-mcp
```

## Contributing

Contributions are welcome! Please read our contributing guidelines and submit pull requests to our repository.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Built on the [Model Context Protocol](https://github.com/modelcontextprotocol/go-sdk)
- Inspired by [Claude Code](https://claude.ai/code) tools
- Uses [ripgrep](https://github.com/BurntSushi/ripgrep) for fast searching