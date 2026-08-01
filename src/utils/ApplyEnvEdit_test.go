package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyEnvEditRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		ops      []EnvEditOp
		expected string
	}{
		{
			name:    "set simple value",
			content: "KEY1=value1\nKEY2=value2\n",
			ops: []EnvEditOp{
				{Type: "set", Key: "KEY1", Value: "newvalue"},
			},
			expected: "KEY1=newvalue\nKEY2=value2\n",
		},
		{
			name:    "add new key",
			content: "KEY1=value1\n",
			ops: []EnvEditOp{
				{Type: "set", Key: "KEY2", Value: "value2"},
			},
			expected: "KEY1=value1\nKEY2=value2\n",
		},
		{
			name:    "delete key",
			content: "KEY1=value1\nKEY2=value2\nKEY3=value3\n",
			ops: []EnvEditOp{
				{Type: "delete", Key: "KEY2"},
			},
			expected: "KEY1=value1\nKEY3=value3\n",
		},
		{
			name:    "preserve comments and blanks",
			content: "# Comment\nKEY1=value1\n\nKEY2=value2\n",
			ops: []EnvEditOp{
				{Type: "set", Key: "KEY1", Value: "newvalue"},
			},
			expected: "# Comment\nKEY1=newvalue\n\nKEY2=value2\n",
		},
		{
			name:    "preserve export prefix",
			content: "export KEY1=value1\nKEY2=value2\n",
			ops: []EnvEditOp{
				{Type: "set", Key: "KEY1", Value: "newvalue"},
			},
			expected: "export KEY1=newvalue\nKEY2=value2\n",
		},
		{
			name:    "quote value with spaces",
			content: "KEY1=value1\n",
			ops: []EnvEditOp{
				{Type: "set", Key: "KEY2", Value: "value with spaces"},
			},
			expected: "KEY1=value1\nKEY2=\"value with spaces\"\n",
		},
		{
			name:    "empty value",
			content: "KEY1=value1\n",
			ops: []EnvEditOp{
				{Type: "set", Key: "KEY2", Value: ""},
			},
			expected: "KEY1=value1\nKEY2=\"\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			tmpDir := t.TempDir()
			envPath := filepath.Join(tmpDir, ".env")
			if err := os.WriteFile(envPath, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}

			// Apply the edits
			if err := ApplyEnvEdit(envPath, tt.ops); err != nil {
				t.Fatalf("ApplyEnvEdit: %v", err)
			}

			// Read back the result
			result, err := os.ReadFile(envPath)
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("got:\n%q\n\nwant:\n%q", string(result), tt.expected)
			}
		})
	}
}

func TestApplyEnvEditCreateNewFile(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	// Apply edits to non-existent file
	ops := []EnvEditOp{
		{Type: "set", Key: "KEY1", Value: "value1"},
		{Type: "set", Key: "KEY2", Value: "value2"},
	}

	if err := ApplyEnvEdit(envPath, ops); err != nil {
		t.Fatalf("ApplyEnvEdit: %v", err)
	}

	// Verify file was created with 0600 mode
	info, err := os.Stat(envPath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("expected mode 0600, got %o", mode)
	}

	// Verify content
	result, _ := os.ReadFile(envPath)
	if string(result) != "KEY1=value1\nKEY2=value2\n" {
		t.Errorf("unexpected content: %q", string(result))
	}
}

func TestFormatEnvLineQuoting(t *testing.T) {
	tests := []struct {
		key    string
		value  string
		want   string
	}{
		{"KEY", "simple", "KEY=simple"},
		{"KEY", "has space", "KEY=\"has space\""},
		{"KEY", "has#hash", "KEY=\"has#hash\""},
		{"KEY", "", "KEY=\"\""},
		{"KEY", "has\"quote", "KEY=\"has\\\"quote\""},
		{"KEY", "has\\backslash", "KEY=\"has\\\\backslash\""},
	}

	for _, tt := range tests {
		t.Run(tt.key+"="+tt.value, func(t *testing.T) {
			result := formatEnvLine(tt.key, tt.value, "")
			if result != tt.want {
				t.Errorf("got %q, want %q", result, tt.want)
			}
		})
	}
}
