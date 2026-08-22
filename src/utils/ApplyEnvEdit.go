package utils

import (
	"fmt"
	"os"
	"strings"
)

// EnvEditOp describes a mutation to apply to the .env file.
type EnvEditOp struct {
	Type  string // "set" (add or update key), "delete"
	Key   string // The variable name
	Value string // For "set" operations
}

// ApplyEnvEdit applies a series of edits to an .env file, preserving formatting.
// Comments, blank lines, key order, quoting, export prefixes, CRLF and BOM survive.
func ApplyEnvEdit(filePath string, ops []EnvEditOp) error {
	// Read the current file
	bytes, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read failed: %w", err)
	}

	content := string(bytes)
	lines := parseLines(content)

	// Track which keys have been modified or deleted. setOrder keeps the order
	// the caller asked for, because the keys that turn out to be new are
	// appended to the end of the file below: ranging over keysToSet to do that
	// would write them in Go's randomized map order, so the same edit would
	// produce a different file - and a different diff - on every run.
	keysToDelete := make(map[string]bool)
	keysToSet := make(map[string]string)
	var setOrder []string

	for _, op := range ops {
		switch op.Type {
		case "set":
			if _, alreadySet := keysToSet[op.Key]; !alreadySet {
				setOrder = append(setOrder, op.Key)
			}
			keysToSet[op.Key] = op.Value
		case "delete":
			keysToDelete[op.Key] = true
		}
	}

	// Apply edits to existing lines, preserving formatting
	var newLines []string
	seenKeys := make(map[string]bool)

	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "#") {
			// Preserve blank lines and comments as-is
			newLines = append(newLines, line)
			continue
		}

		// Try to parse this line as a key=value
		// Preserve the raw line format (export, quoting, etc.)
		key, err := extractKeyFromLine(line)
		if err != nil || key == "" {
			// Parse error or no key found - preserve the line
			newLines = append(newLines, line)
			continue
		}

		seenKeys[key] = true

		if keysToDelete[key] {
			// Skip deleted keys (leave blanks to match the delete without disturbing line count)
			// Actually, we should preserve the file structure. Comment out or remove?
			// Per D5.2, we preserve everything and only rewrite changed lines.
			// So: skip this line entirely for deletions
			continue
		}

		if newValue, hasNewValue := keysToSet[key]; hasNewValue {
			// Rewrite this line with the new value, preserving the format
			newLines = append(newLines, formatEnvLine(key, newValue, line))
			delete(keysToSet, key) // Mark as handled
		} else {
			// Keep the line as-is
			newLines = append(newLines, line)
		}
	}

	// Append any new keys that weren't in the file, in the order they were
	// asked for. Keys already rewritten in place were dropped from keysToSet
	// by the walk above, so what is left here is exactly the new ones.
	for _, key := range setOrder {
		value, isNew := keysToSet[key]
		if !isNew {
			continue
		}

		newLines = append(newLines, formatEnvLine(key, value, ""))
	}

	// Reconstruct the file with preserved line endings
	newContent := linesToContent(newLines, content)

	// Ensure trailing newline (idiomatic for .env files)
	if newContent != "" && !strings.HasSuffix(newContent, "\n") && !strings.HasSuffix(newContent, "\r\n") {
		newContent += "\n"
	}

	// Write atomically with mode 0600 for new files
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return ReplaceFileAtomicallyWithMode(filePath, []byte(newContent), 0o600)
	}
	return ReplaceFileAtomically(filePath, []byte(newContent))
}

// extractKeyFromLine extracts the key from a line like "KEY=value" or "export KEY=value"
func extractKeyFromLine(line string) (string, error) {
	// Remove export prefix if present
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "export ") {
		trimmed = strings.TrimSpace(trimmed[7:])
	}

	// Find the = sign
	eqIdx := strings.IndexByte(trimmed, '=')
	if eqIdx <= 0 {
		return "", fmt.Errorf("no key found")
	}

	return trimmed[:eqIdx], nil
}

// formatEnvLine formats a key=value pair, trying to match the original line's format if possible.
// If originalLine is empty, it creates a new line.
func formatEnvLine(key, value, originalLine string) string {
	// If we have an original line, try to preserve its format
	if originalLine != "" {
		// Check if original had export prefix
		hasExport := strings.HasPrefix(strings.TrimSpace(originalLine), "export ")

		// Build the new line with the same export prefix
		var sb strings.Builder
		if hasExport {
			sb.WriteString("export ")
		}
		sb.WriteString(key)
		sb.WriteString("=")

		// Quote the value if needed (contains space, #, quote, backslash, or is empty)
		if value == "" || strings.ContainsAny(value, " #\"\\") {
			sb.WriteString(quoteEnvValue(value))
		} else {
			sb.WriteString(value)
		}

		return sb.String()
	}

	// New line - just use the basic format
	if value == "" || strings.ContainsAny(value, " #\"\\") {
		return key + "=" + quoteEnvValue(value)
	}
	return key + "=" + value
}

// quoteEnvValue quotes and escapes an env value per dotenv rules.
// Strings containing spaces, #, or special chars are quoted; internal quotes and backslashes are escaped.
func quoteEnvValue(value string) string {
	// If it contains problematic chars, quote it
	// Note: always quote if it contains quotes or backslashes (need escaping), or spaces/hash (syntax), or empty
	if value == "" || strings.ContainsAny(value, " #\"\\") {
		// Escape backslashes first, then quotes, then quote the whole thing
		escaped := strings.ReplaceAll(value, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		return "\"" + escaped + "\""
	}
	return value
}

// linesToContent joins lines back into content, preserving the dominant line ending (LF or CRLF).
// If the original content ended with a newline, the result will too.
func linesToContent(lines []string, originalContent string) string {
	// Detect line ending from original content
	lineEnding := "\n"
	if strings.Contains(originalContent, "\r\n") {
		lineEnding = "\r\n"
	}

	// Join lines with the detected ending
	result := strings.Join(lines, lineEnding)

	// Preserve the trailing newline if the original had one
	if len(originalContent) > 0 && (originalContent[len(originalContent)-1] == '\n' || originalContent[len(originalContent)-1] == '\r') {
		result += lineEnding
	}

	return result
}

// parseLines splits content by newline, preserving line text without the newline.
// (This duplicates the one in GetEnvFileContents.go for now; consolidate later if needed.)
func parseLines(content string) []string {
	var lines []string
	var current []byte

	for _, b := range []byte(content) {
		if b == '\n' {
			lines = append(lines, string(current))
			current = nil
		} else if b != '\r' {
			// Skip CR for CRLF, keep others
			current = append(current, b)
		}
	}

	// Don't forget the last line if it doesn't end with newline
	if len(current) > 0 {
		lines = append(lines, string(current))
	}

	return lines
}
