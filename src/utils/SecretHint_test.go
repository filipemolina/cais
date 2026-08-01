package utils

import (
	"testing"
)

func TestSecretHint(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		// Secret patterns
		{"API_KEY", true},
		{"DATABASE_PASSWORD", true},
		{"SECRET_TOKEN", true},
		{"AUTH_TOKEN", true},
		{"STRIPE_API_KEY", true},
		{"GITHUB_TOKEN", true},
		{"JWT_SECRET", true},
		{"SLACK_TOKEN", true},
		{"AWS_SECRET_ACCESS_KEY", true},
		{"CREDENTIAL_ID", true},
		{"API_SECRET", true},
		{"AUTH_PASSWORD", true},

		// Case insensitivity
		{"api_key", true},
		{"Api_Key", true},
		{"PASSWORD", true},
		{"password", true},

		// Non-secrets explicitly
		{"DATABASE_URL", false},
		{"API_URL", false},
		{"SERVER_HOST", false},
		{"SERVER_PORT", false},
		{"TIMEZONE", false},
		{"TZ", false},
		{"PUID", false},
		{"PGID", false},

		// Innocuous keys
		{"DEBUG", false},
		{"LOG_LEVEL", false},
		{"APP_NAME", false},
		{"VERSION", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := SecretHint(tt.key)
			if got != tt.want {
				t.Errorf("SecretHint(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}
