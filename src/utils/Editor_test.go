package utils

import (
	"slices"
	"testing"
)

func TestEditorCommand(t *testing.T) {
	cases := []struct {
		name       string
		visual     string
		editor     string
		wantSuffix []string
	}{
		{
			name:       "VISUAL wins over EDITOR",
			visual:     "nvim",
			editor:     "ed",
			wantSuffix: []string{"nvim", "compose.yaml"},
		},
		{
			name:       "EDITOR is used when VISUAL is unset",
			editor:     "nano",
			wantSuffix: []string{"nano", "compose.yaml"},
		},
		{
			// The case that makes splitting worth doing: an editor that
			// needs a flag to block until the file is closed.
			name:       "a multi-word value becomes command and args",
			editor:     "code --wait",
			wantSuffix: []string{"code", "--wait", "compose.yaml"},
		},
		{
			name:       "neither set falls back to vi",
			wantSuffix: []string{FallbackEditor, "compose.yaml"},
		},
		{
			// An exported-but-empty EDITOR is common in shell profiles, and
			// exec.Command("") fails in a way the user cannot act on.
			name:       "a whitespace-only value is treated as unset",
			editor:     "   ",
			wantSuffix: []string{FallbackEditor, "compose.yaml"},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("VISUAL", testCase.visual)
			t.Setenv("EDITOR", testCase.editor)

			cmd := EditorCommand("compose.yaml")

			// Path is resolved against PATH by exec.Command, so compare the
			// args, which hold what was actually asked for.
			if !slices.Equal(cmd.Args, testCase.wantSuffix) {
				t.Errorf("args: got %v, want %v", cmd.Args, testCase.wantSuffix)
			}
		})
	}
}
