package file

import (
	"os"
	"regexp"
	"testing"
)

func TestMatchIncludePattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		fileName string
		want     bool
		wantErr  bool
	}{
		{
			name:     "simple glob match",
			pattern:  "*.go",
			fileName: "main.go",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "simple glob no match",
			pattern:  "*.go",
			fileName: "main.js",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "brace expansion match first",
			pattern:  "*.{ts,tsx}",
			fileName: "component.ts",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "brace expansion match second",
			pattern:  "*.{ts,tsx}",
			fileName: "component.tsx",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "brace expansion no match",
			pattern:  "*.{ts,tsx}",
			fileName: "component.js",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "complex brace pattern",
			pattern:  "test.{js,ts,jsx,tsx}",
			fileName: "test.jsx",
			want:     true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matchIncludePattern(tt.pattern, tt.fileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("matchIncludePattern() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("matchIncludePattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchBracePattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		fileName string
		want     bool
		wantErr  bool
	}{
		{
			name:     "simple brace match",
			pattern:  "*.{go}",
			fileName: "main.go",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "multiple alternatives match",
			pattern:  "*.{js,ts}",
			fileName: "app.ts",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "whitespace in alternatives",
			pattern:  "*.{js, ts, jsx}",
			fileName: "app.jsx",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "no match",
			pattern:  "*.{js,ts}",
			fileName: "app.py",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "malformed pattern",
			pattern:  "*.{js,ts",
			fileName: "app.js",
			want:     false, // Falls back to standard matching which won't match
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matchBracePattern(tt.pattern, tt.fileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("matchBracePattern() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("matchBracePattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsBinaryContent(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "plain text",
			data: []byte("Hello, world!\nThis is plain text."),
			want: false,
		},
		{
			name: "text with tabs and newlines",
			data: []byte("package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}"),
			want: false,
		},
		{
			name: "binary with null bytes",
			data: []byte{0x00, 0x01, 0x02, 0x03, 0x00, 0x00, 0x48, 0x65, 0x6c, 0x6c, 0x6f},
			want: true,
		},
		{
			name: "mostly non-printable",
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x0b, 0x0c},
			want: true,
		},
		{
			name: "empty data",
			data: []byte{},
			want: false,
		},
		{
			name: "mostly text with few control chars",
			data: []byte("Hello\x01World\x02Test"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBinaryContent(tt.data); got != tt.want {
				t.Errorf("isBinaryContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSearchFileContent(t *testing.T) {
	// Create a temporary file for testing
	tempFile := "/tmp/test_grep_content.txt"
	content := `package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	fmt.Println("Hello, world!")
	log.Println("This is a log message")
	
	if err := os.Chdir("."); err != nil {
		log.Fatal("Error changing directory")
	}
}

// TestFunction demonstrates function matching
func TestFunction(param string) error {
	return fmt.Errorf("test error: %s", param)
}`

	// Write test content to file
	if _, err := writeFileContent(tempFile, content); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() {
		// Clean up
		if err := os.Remove(tempFile); err != nil {
			t.Logf("Failed to clean up test file: %v", err)
		}
	}()

	tests := []struct {
		name    string
		pattern string
		want    bool
		wantErr bool
	}{
		{
			name:    "find import statement",
			pattern: `import\s*\(`,
			want:    true,
			wantErr: false,
		},
		{
			name:    "find function definition",
			pattern: `func\s+\w+`,
			want:    true,
			wantErr: false,
		},
		{
			name:    "find specific function",
			pattern: `func main`,
			want:    true,
			wantErr: false,
		},
		{
			name:    "find fmt usage",
			pattern: `fmt\..*`,
			want:    true,
			wantErr: false,
		},
		{
			name:    "pattern not found",
			pattern: `nonexistent_pattern`,
			want:    false,
			wantErr: false,
		},
		{
			name:    "case sensitive match",
			pattern: `MAIN`,
			want:    false,
			wantErr: false,
		},
		{
			name:    "regex special chars",
			pattern: `log\..*Error`,
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regex, err := regexp.Compile(tt.pattern)
			if err != nil {
				t.Fatalf("Failed to compile regex: %v", err)
			}

			got, err := searchFileContent(tempFile, regex)
			if (err != nil) != tt.wantErr {
				t.Errorf("searchFileContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("searchFileContent() = %v, want %v", got, tt.want)
			}
		})
	}
}
