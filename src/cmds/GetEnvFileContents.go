package cmds

import (
	"bytes"
	"fmt"
	"os"

	"github.com/compose-spec/compose-go/v2/dotenv"

	tea "charm.land/bubbletea/v2"
)

// EnvEntry represents one line from the .env file.
type EnvEntry struct {
	Key   string // "" for comments/blank lines
	Value string
	Raw   string // The original line as it appears in the file
	// Source indicates what this entry is: "var", "comment", "blank", "parse_error"
	Source string
}

type EnvFileContentsMsg struct {
	Path    string       // Resolved .env path
	Entries []EnvEntry   // Parsed entries in file order
	Err     error
}

// GetEnvFileContents reads and parses the .env file at the given path.
// It uses compose-go's dotenv parser to match compose semantics.
func GetEnvFileContents(envPath string) tea.Cmd {
	return func() tea.Msg {
		// Check if file exists
		if _, err := os.Stat(envPath); err != nil {
			if os.IsNotExist(err) {
				// File doesn't exist - return empty with no error
				return EnvFileContentsMsg{
					Path:    envPath,
					Entries: []EnvEntry{},
				}
			}
			return EnvFileContentsMsg{
				Path: envPath,
				Err:  fmt.Errorf("stat failed: %w", err),
			}
		}

		// Read the file
		bytes, err := os.ReadFile(envPath)
		if err != nil {
			return EnvFileContentsMsg{
				Path: envPath,
				Err:  fmt.Errorf("read failed: %w", err),
			}
		}

		content := string(bytes)
		entries, err := parseEnvFile(content)
		if err != nil {
			// Parse errors are non-fatal; return what we got
			return EnvFileContentsMsg{
				Path:    envPath,
				Entries: entries,
				Err:     err,
			}
		}

		return EnvFileContentsMsg{
			Path:    envPath,
			Entries: entries,
		}
	}
}

// parseEnvFile parses the raw .env content line by line.
// It preserves comments and blank lines and uses compose-go's parser for values.
func parseEnvFile(content string) ([]EnvEntry, error) {
	lines := parseLines(content)
	var entries []EnvEntry
	var parseErr error

	for _, line := range lines {
		if line == "" {
			// Blank line
			entries = append(entries, EnvEntry{
				Key:    "",
				Value:  "",
				Raw:    line,
				Source: "blank",
			})
			continue
		}

		if line[0] == '#' {
			// Comment
			entries = append(entries, EnvEntry{
				Key:    "",
				Value:  "",
				Raw:    line,
				Source: "comment",
			})
			continue
		}

		// Try to parse as a key=value pair using compose-go's parser
		mapping, err := dotenv.Parse(bytes.NewReader([]byte(line)))
		if err != nil || len(mapping) == 0 {
			// Parse error - record it as such
			if parseErr == nil {
				parseErr = fmt.Errorf("parse error on line: %s", line)
			}
			entries = append(entries, EnvEntry{
				Key:    "",
				Value:  "",
				Raw:    line,
				Source: "parse_error",
			})
			continue
		}

		// Extract the parsed key-value (should be one pair per line)
		for key, value := range mapping {
			entries = append(entries, EnvEntry{
				Key:    key,
				Value:  value,
				Raw:    line,
				Source: "var",
			})
		}
	}

	return entries, parseErr
}

// parseLines splits content by newline, preserving the line text without the newline.
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
