package utils

import "strings"

// SecretHint reports whether a value under this env key should be masked by
// default when something other than the Env page renders it. Key name only —
// value-shape heuristics are deliberately not used.
//
// It is a hint, not a guarantee: the Env page masks everything regardless,
// and this exists so the next panel to render arbitrary env values does the
// safe thing without its author having to think about it.
func SecretHint(key string) bool {
	upper := strings.ToUpper(key)

	// Secret patterns: suffix/prefix/contains
	secretPatterns := []string{
		"KEY",
		"TOKEN",
		"SECRET",
		"PASSWORD",
		"PASS",
		"CREDENTIAL",
		"APIKEY",
		"APISECRET",
		"AUTH",
	}

	for _, pattern := range secretPatterns {
		if strings.Contains(upper, pattern) {
			return true
		}
	}

	return false
}
