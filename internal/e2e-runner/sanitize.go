// sanitize.go provides utilities for redacting sensitive data from command output.
//
// This file implements regex-based patterns for detecting and sanitizing
// sensitive information in Terraform and test output, including:
//   - API keys and authentication tokens
//   - AWS-style credentials (access keys and secret keys)
//   - Email addresses
//   - Passwords and secrets
//
// Sanitization ensures that test output and logs don't inadvertently expose
// credentials or other sensitive data.
package e2e

import (
	"regexp"
	"strings"
)

// Sensitive patterns to redact
var (
	// API keys and tokens (generic patterns) - matches key="value" and key = "value"
	apiKeyPattern = regexp.MustCompile(`(?i)(api[_-]?key|token|secret)\s*=\s*"([a-zA-Z0-9_\-]{10,})"`)

	// AWS-style credentials - matches KEY="value" and KEY=value
	// AWS Access Keys are 20 chars: A-Z and 2-7 only (no 0,1,8,9 to avoid confusion with O,I,L)
	awsAccessKeyPattern = regexp.MustCompile(`(?i)(AWS_ACCESS_KEY_ID|access[_-]?key)=("?)([A-Z2-7]{20})("?)`)
	// AWS Secret Keys are exactly 40 base64 chars: letters, numbers, +, /
	awsSecretKeyPattern = regexp.MustCompile(`(?i)(AWS_SECRET_ACCESS_KEY|secret[_-]?key)=("?)([A-Za-z0-9/+]{40})("?)`)

	// Cloudflare specific - matches with and without quotes
	// Cloudflare API tokens are typically 40+ characters
	cfAPIKeyPattern   = regexp.MustCompile(`(?i)(CLOUDFLARE_API_KEY|cloudflare[_-]?api[_-]?key)=("?)([a-zA-Z0-9_\-]{32,})("?)`)
	cfAPITokenPattern = regexp.MustCompile(`(?i)(CLOUDFLARE_API_TOKEN|cloudflare[_-]?api[_-]?token)=("?)([a-zA-Z0-9_\-]{32,})("?)`)
	cfEmailPattern    = regexp.MustCompile(`(?i)(cloudflare[_-]?email)\s*=\s*"([^"]+@[^"]+)"`)
	// R2 uses S3-compatible API: Access Key ~32 chars, Secret is SHA-256 hash (64 hex or 43-44 base64)
	cfR2AccessPattern = regexp.MustCompile(`(?i)(CLOUDFLARE_R2_ACCESS_KEY_ID|r2[_-]?access[_-]?key)=("?)([a-zA-Z0-9]{20,})("?)`)
	cfR2SecretPattern = regexp.MustCompile(`(?i)(CLOUDFLARE_R2_SECRET_ACCESS_KEY|r2[_-]?secret[_-]?key)=("?)([a-zA-Z0-9/+=]{32,})("?)`)

	// Generic sensitive values
	passwordPattern = regexp.MustCompile(`(?i)(password|passwd|pwd)\s*=\s*"([^"]{6,})"`)

	// Sensitive attributes in terraform output - handles -> (before and after)
	tfSensitiveAttrPattern = regexp.MustCompile(`(?i)(sensitive[_-]?value|secret|credential)\s*=\s*"[^"]*"(\s*->\s*"[^"]*")?`)

	// Environment variable patterns (key=value format)
	envVarPattern = regexp.MustCompile(`([A-Z_]+)=([^\s]+)`)
)

// SanitizationConfig controls what gets redacted
type SanitizationConfig struct {
	RedactAPIKeys        bool
	RedactAWSCredentials bool
	RedactEmails         bool
	RedactPasswords      bool
	RedactEnvVars        bool
	CustomPatterns       []*regexp.Regexp
	RedactedText         string
}

// DefaultSanitizationConfig returns safe defaults
func DefaultSanitizationConfig() *SanitizationConfig {
	return &SanitizationConfig{
		RedactAPIKeys:        true,
		RedactAWSCredentials: true,
		RedactEmails:         false, // Usually safe to show
		RedactPasswords:      true,
		RedactEnvVars:        true,
		CustomPatterns:       []*regexp.Regexp{},
		RedactedText:         "[REDACTED]",
	}
}

// SanitizeOutput removes sensitive data from command output
func SanitizeOutput(output string, config *SanitizationConfig) string {
	if config == nil {
		config = DefaultSanitizationConfig()
	}

	sanitized := output

	// Redact AWS credentials first (R2 uses AWS-compatible API)
	if config.RedactAWSCredentials {
		sanitized = awsAccessKeyPattern.ReplaceAllString(sanitized, `$1=$2`+config.RedactedText+`$4`)
		sanitized = awsSecretKeyPattern.ReplaceAllString(sanitized, `$1=$2`+config.RedactedText+`$4`)
	}

	// Redact API keys and tokens (specific patterns before generic)
	if config.RedactAPIKeys {
		// Cloudflare-specific patterns first (preserve quotes if present)
		sanitized = cfAPIKeyPattern.ReplaceAllString(sanitized, `$1=$2`+config.RedactedText+`$4`)
		sanitized = cfAPITokenPattern.ReplaceAllString(sanitized, `$1=$2`+config.RedactedText+`$4`)
		sanitized = cfR2AccessPattern.ReplaceAllString(sanitized, `$1=$2`+config.RedactedText+`$4`)
		sanitized = cfR2SecretPattern.ReplaceAllString(sanitized, `$1=$2`+config.RedactedText+`$4`)
		// Generic pattern last (catches non-prefixed keys)
		sanitized = apiKeyPattern.ReplaceAllString(sanitized, `$1 = "`+config.RedactedText+`"`)
	}

	// Redact emails
	if config.RedactEmails {
		sanitized = cfEmailPattern.ReplaceAllString(sanitized, `$1 = "`+config.RedactedText+`"`)
	}

	// Redact passwords
	if config.RedactPasswords {
		sanitized = passwordPattern.ReplaceAllString(sanitized, `$1 = "`+config.RedactedText+`"`)
	}

	// Redact sensitive terraform attributes - handles both sides of ->
	sanitized = tfSensitiveAttrPattern.ReplaceAllString(sanitized, `$1 = "`+config.RedactedText+`"`)

	// Redact environment variables (be selective)
	if config.RedactEnvVars {
		sanitized = redactSensitiveEnvVars(sanitized, config.RedactedText)
	}

	// Apply custom patterns
	for _, pattern := range config.CustomPatterns {
		sanitized = pattern.ReplaceAllString(sanitized, config.RedactedText)
	}

	return sanitized
}

// redactSensitiveEnvVars only redacts TF_VAR env vars
// AWS and Cloudflare vars are handled by specific patterns that preserve quotes
func redactSensitiveEnvVars(output, redactedText string) string {
	sensitiveVars := []string{
		"TF_VAR_api_key",
		"TF_VAR_secret",
		"TF_VAR_token",
	}

	result := output
	for _, varName := range sensitiveVars {
		pattern := regexp.MustCompile(varName + `=([^\s]+)`)
		result = pattern.ReplaceAllString(result, varName+"="+redactedText)
	}

	return result
}

// SanitizeLines sanitizes each line individually (for streaming output)
func SanitizeLines(lines []string, config *SanitizationConfig) []string {
	if config == nil {
		config = DefaultSanitizationConfig()
	}

	sanitized := make([]string, len(lines))
	for i, line := range lines {
		sanitized[i] = SanitizeOutput(line, config)
	}
	return sanitized
}

// MaskSensitiveString partially masks a sensitive string
// Example: "abc123secret456" -> "abc...456"
func MaskSensitiveString(s string, showPrefix, showSuffix int) string {
	if len(s) <= showPrefix+showSuffix {
		return strings.Repeat("*", len(s))
	}

	return s[:showPrefix] + "..." + s[len(s)-showSuffix:]
}
