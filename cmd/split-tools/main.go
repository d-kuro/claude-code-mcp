package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <input-file> <output-dir>\n", os.Args[0])
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputDir := os.Args[2]

	if err := splitToolsFile(inputFile, outputDir); err != nil {
		log.Fatalf("Error splitting tools file: %v", err)
	}

	fmt.Printf("Successfully split tools into %s\n", outputDir)
}

func splitToolsFile(inputFile, outputDir string) error {
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	scanner := bufio.NewScanner(file)
	var currentTool string
	var currentContent strings.Builder
	headerRegex := regexp.MustCompile(`^# (.+)$`)

	for scanner.Scan() {
		line := scanner.Text()

		if matches := headerRegex.FindStringSubmatch(line); matches != nil {
			if currentTool != "" {
				if err := writeToolFile(outputDir, currentTool, currentContent.String()); err != nil {
					return err
				}
			}

			currentTool = strings.ToLower(strings.ReplaceAll(matches[1], "_", "-"))
			currentContent.Reset()
			currentContent.WriteString(line + "\n")
		} else if currentTool != "" {
			currentContent.WriteString(line + "\n")
		}
	}

	if currentTool != "" {
		if err := writeToolFile(outputDir, currentTool, currentContent.String()); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return nil
}

func writeToolFile(outputDir, toolName, content string) error {
	filename := filepath.Join(outputDir, toolName+".md")
	content = strings.TrimSpace(content) + "\n"

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}

	fmt.Printf("Created: %s\n", filename)
	return nil
}
