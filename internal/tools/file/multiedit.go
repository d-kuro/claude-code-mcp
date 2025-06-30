// Package file provides the MultiEdit tool implementation.
//
// The MultiEdit tool allows performing multiple atomic string replacements
// on a single file in sequence. This is more efficient than using the Edit
// tool multiple times and provides atomic all-or-nothing behavior.
//
// Key features:
// - Multiple edits applied sequentially
// - Atomic operation (all succeed or none applied)
// - Automatic backup and rollback on failure
// - Follows MCP SDK patterns and security model
package file
