# Claude Code MCP Server

A Model Context Protocol (MCP) server that brings Claude Code's powerful development tools to any application. Get instant access to file operations, command execution, web tools, and task management through the standardized MCP interface.

## What It Does

Transform any MCP-compatible application into a powerful development environment with:

- **File Operations**: Read, write, edit, search, and navigate files with ease
- **Command Execution**: Run shell commands with persistent sessions
- **Web Tools**: Fetch web content and perform searches
- **Jupyter Support**: Work with notebooks programmatically
- **Task Management**: Organize and track your work
- **Built-in Security**: Safe operations with automatic validation

## Quick Start

### 1. Install

Download the latest binary for your platform:

**Linux/macOS:**
```bash
curl -L https://github.com/d-kuro/claude-code-mcp/releases/latest/download/claude-code-mcp-$(uname -s | tr '[:upper:]' '[:lower:]')-amd64 -o claude-code-mcp
chmod +x claude-code-mcp
```

**Windows:**
```powershell
curl -L https://github.com/d-kuro/claude-code-mcp/releases/latest/download/claude-code-mcp-windows-amd64.exe -o claude-code-mcp.exe
```

### 2. Run

```bash
./claude-code-mcp
```

That's it! All tools are now available through the MCP interface.

## Available Tools

### 📁 File Operations
- **Read** - View file contents with optional line ranges
- **Write** - Create or overwrite files safely
- **Edit** - Make precise string replacements
- **MultiEdit** - Apply multiple edits atomically
- **LS** - List directory contents
- **Glob** - Find files by patterns
- **Grep** - Search file contents

### ⚡ System Tools
- **Bash** - Execute shell commands with persistent sessions

### 🌐 Web Tools
- **WebFetch** - Retrieve and process web content
- **WebSearch** - Search the web with filtering options

### 📓 Notebook Support
- **NotebookRead** - Read Jupyter notebook cells
- **NotebookEdit** - Modify notebook content

### ✅ Task Management
- **TodoRead/TodoWrite** - Organize tasks within sessions

## Integration

### With Claude Desktop

Add to your `~/.config/claude-desktop/mcp.json`:

```json
{
  "mcpServers": {
    "claude-code-mcp": {
      "command": "/path/to/claude-code-mcp"
    }
  }
}
```

Restart Claude Desktop and the tools will appear automatically.

### With Other MCP Clients

The server works with any MCP-compatible application. Connect using:
- **stdio** transport (default)
- **HTTP/SSE** transport (planned)

## Configuration

### Zero Configuration Required

All tools work immediately with sensible defaults and built-in security.

### Optional Settings

Control logging level:
```bash
export LOG_LEVEL=debug
./claude-code-mcp
```

## Security Features

- **Path Validation** - All file paths are validated and sanitized
- **Command Safety** - Dangerous commands are blocked
- **Resource Limits** - File sizes and timeouts are controlled
- **Session Isolation** - Each MCP session is independent

## Use Cases

### Development Workflows
- **Code Review**: Read and analyze code across multiple files
- **Refactoring**: Make coordinated changes with MultiEdit
- **Debugging**: Execute commands and inspect outputs
- **Documentation**: Fetch web content and organize information

### Data Analysis
- **Research**: Search files and web content
- **Notebook Work**: Read and modify Jupyter notebooks
- **File Processing**: Batch operations on multiple files

### Task Management
- **Project Planning**: Organize work with todo lists
- **Progress Tracking**: Maintain session-based task states

## Troubleshooting

**Q: Which tools are available?**  
A: 14 of Claude Code's 16 tools. The Task and exit_plan_mode tools are not supported in MCP context.

**Q: Do I need to configure anything?**  
A: No! Everything works out of the box with built-in security.

**Q: Can I use relative paths?**  
A: No, all paths must be absolute for security reasons.

**Q: How do I see debug information?**  
A: Set `LOG_LEVEL=debug` before running the server.

## License

MIT License - see LICENSE file for details.