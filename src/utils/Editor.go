package utils

import (
	"os"
	"os/exec"
	"strings"
)

// FallbackEditor is used when neither $VISUAL nor $EDITOR is set. POSIX
// requires vi, so it is the one editor that can be assumed present.
const FallbackEditor = "vi"

// EditorCommand builds the command that opens path in the user's editor.
//
// $VISUAL wins over $EDITOR by long-standing convention: $EDITOR may be a
// line editor for use on a dumb terminal, while $VISUAL is the full-screen
// one, and we are handing over a full terminal.
//
// The value is split on whitespace rather than run through a shell, so
// EDITOR="code --wait" works. A shell would also make every character the
// user has in that variable executable at the moment we hand it the
// terminal, which buys nothing here.
func EditorCommand(path string) *exec.Cmd {
	parts := strings.Fields(editorFromEnv())

	return exec.Command(parts[0], append(parts[1:], path)...)
}

// editorFromEnv returns the configured editor, or the fallback when the
// variables are unset or hold nothing but whitespace.
func editorFromEnv() string {
	for _, key := range []string{"VISUAL", "EDITOR"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}

	return FallbackEditor
}
