package notebook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTestNotebook creates a test notebook file.
func createTestNotebook(t *testing.T) string {
	notebook := JupyterNotebook{
		NBFormat:      4,
		NBFormatMinor: 4,
		Metadata:      map[string]interface{}{},
		Cells: []JupyterCell{
			{
				ID:       "markdown-cell-1",
				CellType: "markdown",
				Source:   []string{"# Test Notebook", "", "This is a test."},
				Metadata: map[string]interface{}{},
			},
			{
				ID:             "code-cell-1",
				CellType:       "code",
				Source:         []string{"print('Hello World')"},
				Metadata:       map[string]interface{}{},
				Outputs:        []interface{}{},
				ExecutionCount: nil,
			},
		},
	}

	tempDir := t.TempDir()
	notebookPath := filepath.Join(tempDir, "test.ipynb")

	data, err := json.MarshalIndent(notebook, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test notebook: %v", err)
	}

	if err := os.WriteFile(notebookPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test notebook: %v", err)
	}

	return notebookPath
}

func TestReadNotebookContent(t *testing.T) {
	// Test reading entire notebook
	notebookPath := createTestNotebook(t)

	content, err := readNotebookContent(notebookPath, nil)
	if err != nil {
		t.Fatalf("Failed to read notebook content: %v", err)
	}

	if !strings.Contains(content, "# Test Notebook") {
		t.Errorf("Expected content to contain '# Test Notebook'")
	}
	if !strings.Contains(content, "print('Hello World')") {
		t.Errorf("Expected content to contain 'print('Hello World')'")
	}

	// Test reading specific cell
	cellID := "markdown-cell-1"
	content, err = readNotebookContent(notebookPath, &cellID)
	if err != nil {
		t.Fatalf("Failed to read specific cell: %v", err)
	}

	if !strings.Contains(content, "# Test Notebook") {
		t.Errorf("Expected content to contain '# Test Notebook'")
	}
	if strings.Contains(content, "print('Hello World')") {
		t.Errorf("Expected content to NOT contain 'print('Hello World')' when reading specific cell")
	}

	// Test nonexistent cell
	nonexistentID := "nonexistent"
	_, err = readNotebookContent(notebookPath, &nonexistentID)
	if err == nil {
		t.Errorf("Expected error when reading nonexistent cell")
	}
}

func TestEditNotebookContent(t *testing.T) {
	// Test replacing cell content
	notebookPath := createTestNotebook(t)
	cellID := "markdown-cell-1"
	newSource := "# Updated Notebook\n\nThis has been updated."

	result, err := editNotebookContent(notebookPath, &cellID, newSource, nil, "replace")
	if err != nil {
		t.Fatalf("Failed to edit notebook: %v", err)
	}

	if !strings.Contains(result, "Successfully replaced") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify the notebook was updated
	data, err := os.ReadFile(notebookPath)
	if err != nil {
		t.Fatalf("Failed to read modified notebook: %v", err)
	}

	var notebook JupyterNotebook
	if err := json.Unmarshal(data, &notebook); err != nil {
		t.Fatalf("Failed to parse modified notebook: %v", err)
	}

	// Find the modified cell
	found := false
	for _, cell := range notebook.Cells {
		if cell.ID == "markdown-cell-1" {
			source := strings.Join(extractSourceLines(cell.Source), "\n")
			if !strings.Contains(source, "# Updated Notebook") {
				t.Errorf("Expected cell to contain '# Updated Notebook'")
			}
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Could not find cell with ID 'markdown-cell-1'")
	}
}

func TestNotebookEditInsert(t *testing.T) {
	notebookPath := createTestNotebook(t)
	cellID := "markdown-cell-1"
	newSource := "x = 42\nprint(x)"
	cellType := "code"

	result, err := editNotebookContent(notebookPath, &cellID, newSource, &cellType, "insert")
	if err != nil {
		t.Fatalf("Failed to insert cell: %v", err)
	}

	if !strings.Contains(result, "Successfully inserted") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify the notebook has an additional cell
	data, err := os.ReadFile(notebookPath)
	if err != nil {
		t.Fatalf("Failed to read modified notebook: %v", err)
	}

	var notebook JupyterNotebook
	if err := json.Unmarshal(data, &notebook); err != nil {
		t.Fatalf("Failed to parse modified notebook: %v", err)
	}

	if len(notebook.Cells) != 3 {
		t.Errorf("Expected 3 cells after insert, got %d", len(notebook.Cells))
	}

	// Check that the new cell was inserted after the first cell
	if notebook.Cells[1].CellType != "code" {
		t.Errorf("Expected inserted cell to be code type, got %s", notebook.Cells[1].CellType)
	}

	source := strings.Join(extractSourceLines(notebook.Cells[1].Source), "\n")
	if !strings.Contains(source, "x = 42") {
		t.Errorf("Expected inserted cell to contain 'x = 42'")
	}
}

func TestNotebookEditDelete(t *testing.T) {
	notebookPath := createTestNotebook(t)
	cellID := "code-cell-1"

	result, err := editNotebookContent(notebookPath, &cellID, "", nil, "delete")
	if err != nil {
		t.Fatalf("Failed to delete cell: %v", err)
	}

	if !strings.Contains(result, "Successfully deleted") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify the notebook has one less cell
	data, err := os.ReadFile(notebookPath)
	if err != nil {
		t.Fatalf("Failed to read modified notebook: %v", err)
	}

	var notebook JupyterNotebook
	if err := json.Unmarshal(data, &notebook); err != nil {
		t.Fatalf("Failed to parse modified notebook: %v", err)
	}

	if len(notebook.Cells) != 1 {
		t.Errorf("Expected 1 cell after delete, got %d", len(notebook.Cells))
	}

	// Check that the remaining cell is the markdown cell
	if notebook.Cells[0].ID != "markdown-cell-1" {
		t.Errorf("Expected remaining cell to be 'markdown-cell-1', got %s", notebook.Cells[0].ID)
	}
}

func TestNotebookEditErrors(t *testing.T) {
	notebookPath := createTestNotebook(t)

	// Test missing cell_id for replace mode
	_, err := editNotebookContent(notebookPath, nil, "test", nil, "replace")
	if err == nil {
		t.Errorf("Expected error for missing cell_id in replace mode")
	}

	// Test nonexistent cell
	nonexistentID := "nonexistent"
	_, err = editNotebookContent(notebookPath, &nonexistentID, "test", nil, "replace")
	if err == nil {
		t.Errorf("Expected error for nonexistent cell")
	}

	// Test invalid edit_mode
	cellID := "markdown-cell-1"
	_, err = editNotebookContent(notebookPath, &cellID, "test", nil, "invalid")
	if err == nil {
		t.Errorf("Expected error for invalid edit_mode")
	}
}
