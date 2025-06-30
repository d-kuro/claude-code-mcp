// Package file provides shared utilities for file operation tools.
package file

import (
	"time"
)

// FileMatchInfo represents a file with its modification time for sorting.
type FileMatchInfo struct {
	Path    string
	ModTime time.Time
}
