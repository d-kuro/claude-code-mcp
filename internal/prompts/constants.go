// Package prompts contains all prompt strings and descriptions used by the tools.
package prompts

import _ "embed"

// Embedded tool documentation files
//

//go:embed tools/bash.md
var BashToolDoc string

//go:embed tools/glob.md
var GlobToolDoc string

//go:embed tools/grep.md
var GrepToolDoc string

//go:embed tools/ls.md
var LSToolDoc string

//go:embed tools/read.md
var ReadToolDoc string

//go:embed tools/edit.md
var EditToolDoc string

//go:embed tools/multiedit.md
var MultiEditToolDoc string

//go:embed tools/write.md
var WriteToolDoc string

//go:embed tools/notebookread.md
var NotebookReadToolDoc string

//go:embed tools/notebookedit.md
var NotebookEditToolDoc string

//go:embed tools/webfetch.md
var WebFetchToolDoc string

//go:embed tools/todoread.md
var TodoReadToolDoc string

//go:embed tools/todowrite.md
var TodoWriteToolDoc string

//go:embed tools/websearch.md
var WebSearchToolDoc string
