package e2e

import (
	"regexp"
	"strings"
	"testing"
)

func TestSanitizeOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		config   *SanitizationConfig
		expected string
	}{
		{
			name:     "redact generic API key",
			input:    `Error: api_key = "abc123secret456" is invalid`,
			config:   DefaultSanitizationConfig(),
			expected: `Error: api_key = "[REDACTED]" is invalid`,
		},
		{
			name:  "redact AWS access key",
			input: `AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE"`,
			config: DefaultSanitizationConfig(),
			expected: `AWS_ACCESS_KEY_ID="[REDACTED]"`,
		},
		{
			name:  "redact AWS secret key",
			input: `AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"`,
			config: DefaultSanitizationConfig(),
			expected: `AWS_SECRET_ACCESS_KEY="[REDACTED]"`,
		},
		{
			name:  "redact Cloudflare API key",
			input: `CLOUDFLARE_API_KEY="abc123def456ghi789jkl012mno345pqr678"`,
			config: DefaultSanitizationConfig(),
			expected: `CLOUDFLARE_API_KEY="[REDACTED]"`,
		},
		{
			name:  "redact R2 credentials",
			input: `CLOUDFLARE_R2_ACCESS_KEY_ID="abc123def456ghi789jk" CLOUDFLARE_R2_SECRET_ACCESS_KEY="secret456789012345678901234567890abc"`,
			config: DefaultSanitizationConfig(),
			expected: `CLOUDFLARE_R2_ACCESS_KEY_ID="[REDACTED]" CLOUDFLARE_R2_SECRET_ACCESS_KEY="[REDACTED]"`,
		},
		{
			name:  "redact sensitive terraform attribute",
			input: `  ~ sensitive_value = "old-secret" -> "new-secret"`,
			config: DefaultSanitizationConfig(),
			expected: `  ~ sensitive_value = "[REDACTED]"`,
		},
		{
			name:  "redact password",
			input: `password = "mySecretPass123"`,
			config: DefaultSanitizationConfig(),
			expected: `password = "[REDACTED]"`,
		},
		{
			name:  "preserve non-sensitive data",
			input: `account_id = "abc123" zone_id = "def456"`,
			config: DefaultSanitizationConfig(),
			expected: `account_id = "abc123" zone_id = "def456"`,
		},
		{
			name:  "redact email when configured",
			input: `cloudflare_email = "user@example.com"`,
			config: &SanitizationConfig{
				RedactEmails: true,
				RedactedText: "[REDACTED]",
			},
			expected: `cloudflare_email = "[REDACTED]"`,
		},
		{
			name:  "preserve email when not configured",
			input: `cloudflare_email = "user@example.com"`,
			config: DefaultSanitizationConfig(),
			expected: `cloudflare_email = "user@example.com"`,
		},
		{
			name: "redact multiple credentials in one line",
			input: `Configuring with AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`,
			config: DefaultSanitizationConfig(),
			expected: `Configuring with AWS_ACCESS_KEY_ID=[REDACTED] AWS_SECRET_ACCESS_KEY=[REDACTED]`,
		},
		{
			name:  "redact environment variable format",
			input: `export CLOUDFLARE_API_KEY=abc123secret456token789key012value`,
			config: DefaultSanitizationConfig(),
			expected: `export CLOUDFLARE_API_KEY=[REDACTED]`,
		},
		{
			name:  "handle multiline output",
			input: "Line 1: normal text\nAWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE\nLine 3: more text",
			config: DefaultSanitizationConfig(),
			expected: "Line 1: normal text\nAWS_ACCESS_KEY_ID=[REDACTED]\nLine 3: more text",
		},
		{
			name:  "custom redacted text",
			input: `api_key = "secret1234567890"`,
			config: &SanitizationConfig{
				RedactAPIKeys: true,
				RedactedText:  "***HIDDEN***",
			},
			expected: `api_key = "***HIDDEN***"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeOutput(tt.input, tt.config)
			if got != tt.expected {
				t.Errorf("SanitizeOutput() =\n%q\nwant:\n%q", got, tt.expected)
			}
		})
	}
}

func TestSanitizeOutput_NilConfig(t *testing.T) {
	input := `AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE"`
	got := SanitizeOutput(input, nil)

	// Should use default config
	if strings.Contains(got, "AKIAIOSFODNN7EXAMPLE") {
		t.Error("Expected credentials to be redacted with default config")
	}
}

func TestSanitizeLines(t *testing.T) {
	input := []string{
		"Line 1: normal text",
		`AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE"`,
		"Line 3: more text",
		`CLOUDFLARE_API_KEY="secret123456789012345678901234567890"`,
	}

	expected := []string{
		"Line 1: normal text",
		`AWS_ACCESS_KEY_ID="[REDACTED]"`,
		"Line 3: more text",
		`CLOUDFLARE_API_KEY="[REDACTED]"`,
	}

	got := SanitizeLines(input, DefaultSanitizationConfig())

	if len(got) != len(expected) {
		t.Fatalf("Expected %d lines, got %d", len(expected), len(got))
	}

	for i := range got {
		if got[i] != expected[i] {
			t.Errorf("Line %d: got %q, want %q", i, got[i], expected[i])
		}
	}
}

func TestMaskSensitiveString(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		showPrefix int
		showSuffix int
		expected   string
	}{
		{
			name:       "mask middle of string",
			input:      "abc123secret456",
			showPrefix: 3,
			showSuffix: 3,
			expected:   "abc...456",
		},
		{
			name:       "mask entire short string",
			input:      "short",
			showPrefix: 3,
			showSuffix: 3,
			expected:   "*****",
		},
		{
			name:       "show only suffix",
			input:      "abc123secret456",
			showPrefix: 0,
			showSuffix: 3,
			expected:   "...456",
		},
		{
			name:       "show only prefix",
			input:      "abc123secret456",
			showPrefix: 3,
			showSuffix: 0,
			expected:   "abc...",
		},
		{
			name:       "empty string",
			input:      "",
			showPrefix: 3,
			showSuffix: 3,
			expected:   "",
		},
		{
			name:       "single character",
			input:      "a",
			showPrefix: 1,
			showSuffix: 1,
			expected:   "*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskSensitiveString(tt.input, tt.showPrefix, tt.showSuffix)
			if got != tt.expected {
				t.Errorf("MaskSensitiveString() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRedactSensitiveEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "redact TF_VAR variables",
			input:    "TF_VAR_api_key=secret TF_VAR_token=token123",
			expected: "TF_VAR_api_key=[REDACTED] TF_VAR_token=[REDACTED]",
		},
		{
			name:     "preserve non-TF_VAR env vars",
			input:    "CLOUDFLARE_ACCOUNT_ID=abc123 CLOUDFLARE_ZONE_ID=def456",
			expected: "CLOUDFLARE_ACCOUNT_ID=abc123 CLOUDFLARE_ZONE_ID=def456",
		},
		{
			name:     "preserve AWS vars (handled by specific patterns)",
			input:    "AWS_ACCESS_KEY_ID=key123",
			expected: "AWS_ACCESS_KEY_ID=key123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactSensitiveEnvVars(tt.input, "[REDACTED]")
			if got != tt.expected {
				t.Errorf("redactSensitiveEnvVars() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSanitizationConfig_CustomPatterns(t *testing.T) {
	// Test with custom regex pattern
	customPattern := regexp.MustCompile(`custom_secret=[^\s]+`)
	config := &SanitizationConfig{
		CustomPatterns: []*regexp.Regexp{customPattern},
		RedactedText:   "[HIDDEN]",
	}

	input := "Some text with custom_secret=verysecret and more text"
	expected := "Some text with [HIDDEN] and more text"

	got := SanitizeOutput(input, config)
	if got != expected {
		t.Errorf("Custom pattern: got %q, want %q", got, expected)
	}
}

func TestSanitizeOutput_RealWorldExamples(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		mustNotContain []string
	}{
		{
			name: "terraform init output with R2 backend",
			input: `Initializing the backend...
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
Successfully configured backend!`,
			mustNotContain: []string{"AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI", "bPxRfiCY"},
		},
		{
			name: "terraform error with credentials",
			input: `Error: Provider configuration error
  on provider.tf line 5:
  5:   api_key = "abc123secret456token789"
Invalid API key format`,
			mustNotContain: []string{"abc123secret456token789"},
		},
		{
			name: "terraform plan with sensitive attributes",
			input: `  ~ resource "cloudflare_api_token" "test" {
      ~ sensitive_value = "old-token-12345" -> "new-token-67890"
    }`,
			mustNotContain: []string{"old-token-12345", "new-token-67890"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeOutput(tt.input, DefaultSanitizationConfig())
			for _, secret := range tt.mustNotContain {
				if strings.Contains(got, secret) {
					t.Errorf("Output still contains secret %q:\n%s", secret, got)
				}
			}
		})
	}
}
