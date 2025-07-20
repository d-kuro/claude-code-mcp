// Package notebook provides Jupyter notebook operation tools using the MCP SDK patterns.
package notebook

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/prompts"
	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// NotebookReadArgs represents the arguments for the NotebookRead tool.
type NotebookReadArgs struct {
	NotebookPath string  `json:"notebook_path"`
	CellID       *string `json:"cell_id,omitempty"`
}

// NotebookEditArgs represents the arguments for the NotebookEdit tool.
type NotebookEditArgs struct {
	NotebookPath string  `json:"notebook_path"`
	NewSource    string  `json:"new_source"`
	CellID       *string `json:"cell_id,omitempty"`
	CellType     *string `json:"cell_type,omitempty"`
	EditMode     *string `json:"edit_mode,omitempty"`
}

// JupyterNotebook represents the structure of a Jupyter notebook.
type JupyterNotebook struct {
	Cells         []JupyterCell `json:"cells"`
	Metadata      interface{}   `json:"metadata"`
	NBFormat      int           `json:"nbformat"`
	NBFormatMinor int           `json:"nbformat_minor"`
}

// JupyterCell represents a cell in a Jupyter notebook.
type JupyterCell struct {
	ID             string                 `json:"id,omitempty"`
	CellType       string                 `json:"cell_type"`
	Source         interface{}            `json:"source"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Outputs        []interface{}          `json:"outputs,omitempty"`
	ExecutionCount *int                   `json:"execution_count,omitempty"`
}

// CreateNotebookReadTool creates the NotebookRead tool using MCP SDK patterns.
func CreateNotebookReadTool(ctx *tools.Context) *tools.ServerTool {
	handler := func(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[NotebookReadArgs]) (*mcp.CallToolResultFor[any], error) {
		args := params.Arguments

		sanitizedPath, err := ctx.Validator.SanitizePath(args.NotebookPath)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Invalid notebook path: " + err.Error()}},
				IsError: true,
			}, nil
		}

		if err := ctx.Validator.ValidatePath(sanitizedPath); err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Path validation failed: " + err.Error()}},
				IsError: true,
			}, nil
		}

		// Validate .ipynb extension
		if !strings.HasSuffix(strings.ToLower(sanitizedPath), ".ipynb") {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: File must have .ipynb extension"}},
				IsError: true,
			}, nil
		}

		content, err := readNotebookContent(sanitizedPath, args.CellID)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: " + err.Error()}},
				IsError: true,
			}, nil
		}

		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: content}},
		}, nil
	}

	// Create a wrapper handler that converts from map[string]any to typed args
	wrapperHandler := func(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[map[string]any]) (*mcp.CallToolResultFor[any], error) {
		// Convert map[string]any to typed args
		var args NotebookReadArgs
		data, err := json.Marshal(params.Arguments)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Failed to marshal arguments: " + err.Error()}},
				IsError: true,
			}, nil
		}

		if err := json.Unmarshal(data, &args); err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Failed to unmarshal arguments: " + err.Error()}},
				IsError: true,
			}, nil
		}

		// Create typed params and call the original handler
		typedParams := &mcp.CallToolParamsFor[NotebookReadArgs]{
			Name:      params.Name,
			Arguments: args,
		}

		return handler(ctx, session, typedParams)
	}

	tool := &mcp.Tool{
		Name:        "NotebookRead",
		Description: prompts.NotebookReadToolDoc,
	}

	return &tools.ServerTool{
		Tool: tool,
		RegisterFunc: func(server *mcp.Server) {
			mcp.AddTool(server, tool, wrapperHandler)
		},
	}
}

// CreateNotebookEditTool creates the NotebookEdit tool using MCP SDK patterns.
func CreateNotebookEditTool(ctx *tools.Context) *tools.ServerTool {
	handler := func(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[NotebookEditArgs]) (*mcp.CallToolResultFor[any], error) {
		args := params.Arguments

		sanitizedPath, err := ctx.Validator.SanitizePath(args.NotebookPath)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Invalid notebook path: " + err.Error()}},
				IsError: true,
			}, nil
		}

		if err := ctx.Validator.ValidatePath(sanitizedPath); err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Path validation failed: " + err.Error()}},
				IsError: true,
			}, nil
		}

		// Validate .ipynb extension
		if !strings.HasSuffix(strings.ToLower(sanitizedPath), ".ipynb") {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: File must have .ipynb extension"}},
				IsError: true,
			}, nil
		}

		// Validate edit mode
		editMode := "replace"
		if args.EditMode != nil {
			editMode = *args.EditMode
			if editMode != "replace" && editMode != "insert" && editMode != "delete" {
				return &mcp.CallToolResultFor[any]{
					Content: []mcp.Content{&mcp.TextContent{Text: "Error: edit_mode must be one of: replace, insert, delete"}},
					IsError: true,
				}, nil
			}
		}

		// Validate cell type for insert mode
		if editMode == "insert" && (args.CellType == nil || *args.CellType == "") {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: cell_type is required when edit_mode is insert"}},
				IsError: true,
			}, nil
		}

		if args.CellType != nil && *args.CellType != "code" && *args.CellType != "markdown" {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: cell_type must be either 'code' or 'markdown'"}},
				IsError: true,
			}, nil
		}

		// Validate new_source for delete mode
		if editMode == "delete" && args.NewSource != "" {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: new_source should be empty when edit_mode is delete"}},
				IsError: true,
			}, nil
		}

		result, err := editNotebookContent(sanitizedPath, args.CellID, args.NewSource, args.CellType, editMode)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: " + err.Error()}},
				IsError: true,
			}, nil
		}

		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, nil
	}

	// Create a wrapper handler that converts from map[string]any to typed args
	wrapperHandler := func(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[map[string]any]) (*mcp.CallToolResultFor[any], error) {
		// Convert map[string]any to typed args
		var args NotebookEditArgs
		data, err := json.Marshal(params.Arguments)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Failed to marshal arguments: " + err.Error()}},
				IsError: true,
			}, nil
		}

		if err := json.Unmarshal(data, &args); err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Failed to unmarshal arguments: " + err.Error()}},
				IsError: true,
			}, nil
		}

		// Create typed params and call the original handler
		typedParams := &mcp.CallToolParamsFor[NotebookEditArgs]{
			Name:      params.Name,
			Arguments: args,
		}

		return handler(ctx, session, typedParams)
	}

	tool := &mcp.Tool{
		Name:        "NotebookEdit",
		Description: prompts.NotebookEditToolDoc,
	}

	return &tools.ServerTool{
		Tool: tool,
		RegisterFunc: func(server *mcp.Server) {
			mcp.AddTool(server, tool, wrapperHandler)
		},
	}
}

// readNotebookContent reads and formats the content of a Jupyter notebook.
func readNotebookContent(notebookPath string, cellID *string) (string, error) {
	// Check if file exists
	stat, err := os.Stat(notebookPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat notebook file: %w", err)
	}

	if stat.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file")
	}

	// Read the notebook file
	data, err := os.ReadFile(notebookPath)
	if err != nil {
		return "", fmt.Errorf("failed to read notebook file: %w", err)
	}

	// Parse JSON
	var notebook JupyterNotebook
	if err := json.Unmarshal(data, &notebook); err != nil {
		return "", fmt.Errorf("failed to parse notebook JSON: %w", err)
	}

	// If specific cell ID requested, find and return only that cell
	if cellID != nil && *cellID != "" {
		for i, cell := range notebook.Cells {
			if cell.ID == *cellID {
				return formatNotebookCell(cell, i), nil
			}
		}
		return "", fmt.Errorf("cell with ID '%s' not found", *cellID)
	}

	// Format all cells
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Jupyter Notebook: %s\n", filepath.Base(notebookPath)))
	output.WriteString(fmt.Sprintf("Format: v%d.%d\n", notebook.NBFormat, notebook.NBFormatMinor))
	output.WriteString(fmt.Sprintf("Total cells: %d\n\n", len(notebook.Cells)))

	for i, cell := range notebook.Cells {
		output.WriteString(formatNotebookCell(cell, i))
		if i < len(notebook.Cells)-1 {
			output.WriteString("\n" + strings.Repeat("-", 80) + "\n\n")
		}
	}

	return output.String(), nil
}

// formatNotebookCell formats a single notebook cell for display.
func formatNotebookCell(cell JupyterCell, index int) string {
	var output strings.Builder

	// Cell header
	output.WriteString(fmt.Sprintf("Cell %d", index))
	if cell.ID != "" {
		output.WriteString(fmt.Sprintf(" (ID: %s)", cell.ID))
	}
	output.WriteString(fmt.Sprintf(" [%s]", cell.CellType))
	if cell.ExecutionCount != nil {
		output.WriteString(fmt.Sprintf(" [%d]", *cell.ExecutionCount))
	}
	output.WriteString(":\n\n")

	// Source content
	sourceLines := extractSourceLines(cell.Source)
	if len(sourceLines) > 0 {
		output.WriteString("Source:\n")
		for i, line := range sourceLines {
			output.WriteString(fmt.Sprintf("%3d: %s\n", i+1, line))
		}
	} else {
		output.WriteString("Source: (empty)\n")
	}

	// Outputs (for code cells)
	if cell.CellType == "code" && len(cell.Outputs) > 0 {
		output.WriteString("\nOutputs:\n")
		for i, outputData := range cell.Outputs {
			output.WriteString(fmt.Sprintf("  Output %d: %s\n", i+1, formatOutputData(outputData)))
		}
	}

	return output.String()
}

// extractSourceLines extracts source lines from various formats.
func extractSourceLines(source interface{}) []string {
	switch s := source.(type) {
	case string:
		if s == "" {
			return nil
		}
		return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	case []interface{}:
		var lines []string
		for _, line := range s {
			if str, ok := line.(string); ok {
				// Remove trailing newline for display
				str = strings.TrimSuffix(str, "\n")
				lines = append(lines, str)
			}
		}
		return lines
	case []string:
		var lines []string
		for _, line := range s {
			lines = append(lines, strings.TrimSuffix(line, "\n"))
		}
		return lines
	default:
		return []string{fmt.Sprintf("%v", source)}
	}
}

// formatOutputData formats output data for display.
func formatOutputData(output interface{}) string {
	if outputMap, ok := output.(map[string]interface{}); ok {
		if outputType, exists := outputMap["output_type"]; exists {
			switch outputType {
			case "stream":
				if text, exists := outputMap["text"]; exists {
					return fmt.Sprintf("stream: %v", text)
				}
			case "execute_result", "display_data":
				if data, exists := outputMap["data"]; exists {
					if dataMap, ok := data.(map[string]interface{}); ok {
						if textPlain, exists := dataMap["text/plain"]; exists {
							return fmt.Sprintf("%s: %v", outputType, textPlain)
						}
					}
				}
			case "error":
				if ename, exists := outputMap["ename"]; exists {
					if evalue, evalueExists := outputMap["evalue"]; evalueExists {
						return fmt.Sprintf("error: %v: %v", ename, evalue)
					}
					return fmt.Sprintf("error: %v", ename)
				}
			}
		}
	}
	return fmt.Sprintf("%v", output)
}

// editNotebookContent edits a notebook cell based on the specified operation.
func editNotebookContent(notebookPath string, cellID *string, newSource string, cellType *string, editMode string) (string, error) {
	// Check if file exists
	stat, err := os.Stat(notebookPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat notebook file: %w", err)
	}

	if stat.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file")
	}

	// Read the notebook file
	data, err := os.ReadFile(notebookPath)
	if err != nil {
		return "", fmt.Errorf("failed to read notebook file: %w", err)
	}

	// Parse JSON
	var notebook JupyterNotebook
	if err := json.Unmarshal(data, &notebook); err != nil {
		return "", fmt.Errorf("failed to parse notebook JSON: %w", err)
	}

	// Create backup
	backupPath := notebookPath + ".backup"
	if err := os.WriteFile(backupPath, data, stat.Mode()); err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}

	var result string
	var modified bool

	switch editMode {
	case "replace":
		result, modified, err = replaceNotebookCell(&notebook, cellID, newSource, cellType)
	case "insert":
		result, modified, err = insertNotebookCell(&notebook, cellID, newSource, *cellType)
	case "delete":
		result, modified, err = deleteNotebookCell(&notebook, cellID)
	default:
		return "", fmt.Errorf("invalid edit mode: %s", editMode)
	}

	if err != nil {
		// Restore backup on error
		_ = os.Rename(backupPath, notebookPath)
		return "", err
	}

	if !modified {
		// Clean up backup if no changes were made
		_ = os.Remove(backupPath)
		return result, nil
	}

	// Write modified notebook back to file
	modifiedData, err := json.MarshalIndent(notebook, "", "  ")
	if err != nil {
		// Restore backup on error
		_ = os.Rename(backupPath, notebookPath)
		return "", fmt.Errorf("failed to marshal modified notebook: %w", err)
	}

	if err := os.WriteFile(notebookPath, modifiedData, stat.Mode()); err != nil {
		// Restore backup on error
		_ = os.Rename(backupPath, notebookPath)
		return "", fmt.Errorf("failed to write modified notebook: %w", err)
	}

	// Clean up backup on success
	_ = os.Remove(backupPath)

	return result, nil
}

// replaceNotebookCell replaces the content of an existing cell.
func replaceNotebookCell(notebook *JupyterNotebook, cellID *string, newSource string, cellType *string) (string, bool, error) {
	if cellID == nil || *cellID == "" {
		return "", false, fmt.Errorf("cell_id is required for replace mode")
	}

	// Find the cell by ID
	for i := range notebook.Cells {
		if notebook.Cells[i].ID == *cellID {
			// Update cell type if specified
			if cellType != nil && *cellType != "" {
				notebook.Cells[i].CellType = *cellType
			}

			// Update source
			notebook.Cells[i].Source = strings.Split(newSource, "\n")

			// Clear outputs and execution count for code cells when replacing content
			if notebook.Cells[i].CellType == "code" {
				notebook.Cells[i].Outputs = nil
				notebook.Cells[i].ExecutionCount = nil
			}

			return fmt.Sprintf("Successfully replaced content of cell %s", *cellID), true, nil
		}
	}

	return "", false, fmt.Errorf("cell with ID '%s' not found", *cellID)
}

// insertNotebookCell inserts a new cell at the specified position.
func insertNotebookCell(notebook *JupyterNotebook, cellID *string, newSource string, cellType string) (string, bool, error) {
	// Generate a unique cell ID
	newCellID := generateCellID()

	// Create new cell
	newCell := JupyterCell{
		ID:       newCellID,
		CellType: cellType,
		Source:   strings.Split(newSource, "\n"),
		Metadata: make(map[string]interface{}),
	}

	// Initialize outputs for code cells
	if cellType == "code" {
		newCell.Outputs = []interface{}{}
		newCell.ExecutionCount = nil
	}

	// Determine insertion position
	insertIndex := 0
	if cellID != nil && *cellID != "" {
		// Find the cell to insert after
		for i, cell := range notebook.Cells {
			if cell.ID == *cellID {
				insertIndex = i + 1
				break
			}
		}
	}

	// Insert the new cell
	if insertIndex >= len(notebook.Cells) {
		// Append at the end
		notebook.Cells = append(notebook.Cells, newCell)
	} else {
		// Insert at the specified position
		notebook.Cells = append(notebook.Cells[:insertIndex], append([]JupyterCell{newCell}, notebook.Cells[insertIndex:]...)...)
	}

	position := "at the beginning"
	if cellID != nil && *cellID != "" {
		position = fmt.Sprintf("after cell %s", *cellID)
	}

	return fmt.Sprintf("Successfully inserted new %s cell (ID: %s) %s", cellType, newCellID, position), true, nil
}

// deleteNotebookCell deletes the specified cell.
func deleteNotebookCell(notebook *JupyterNotebook, cellID *string) (string, bool, error) {
	if cellID == nil || *cellID == "" {
		return "", false, fmt.Errorf("cell_id is required for delete mode")
	}

	// Find and remove the cell
	for i, cell := range notebook.Cells {
		if cell.ID == *cellID {
			// Remove the cell
			notebook.Cells = append(notebook.Cells[:i], notebook.Cells[i+1:]...)
			return fmt.Sprintf("Successfully deleted cell %s", *cellID), true, nil
		}
	}

	return "", false, fmt.Errorf("cell with ID '%s' not found", *cellID)
}

// generateCellID generates a unique cell ID.
func generateCellID() string {
	// Generate a random 8-byte ID similar to Jupyter's format
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("cell-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}
