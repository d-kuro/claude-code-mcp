# Grep

- Fast content search tool that works with any codebase size
- Searches file contents using regular expressions
- Supports full regex syntax (eg. "log.*Error", "function\s+\w+", etc.)
- Filter files by pattern with the include parameter (eg. "*.js", "*.{ts,tsx}")
- Returns file paths with at least one match sorted by modification time
- Use this tool when you need to find files containing specific patterns
- If you need to identify/count the number of matches within files, use the Bash tool with `rg` (ripgrep) directly. Do NOT use `grep`.
- When you are doing an open ended search that may require multiple rounds of globbing and grepping, use the Agent tool instead


```typescript
{
  // The regular expression pattern to search for in file contents
  pattern: string;
  // The directory to search in. Defaults to the current working directory.
  path?: string;
  // File pattern to include in the search (e.g. "*.js", "*.{ts,tsx}")
  include?: string;
}
```
