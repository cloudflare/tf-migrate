package e2e

import (
	"os"
	"testing"
)

func TestLoadEnv(t *testing.T) {
	// Save original env vars and restore after test
	originalEnv := map[string]string{
		"CLOUDFLARE_ACCOUNT_ID":           os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		"CLOUDFLARE_ZONE_ID":              os.Getenv("CLOUDFLARE_ZONE_ID"),
		"CLOUDFLARE_DOMAIN":               os.Getenv("CLOUDFLARE_DOMAIN"),
		"CLOUDFLARE_EMAIL":                os.Getenv("CLOUDFLARE_EMAIL"),
		"CLOUDFLARE_API_KEY":              os.Getenv("CLOUDFLARE_API_KEY"),
		"CLOUDFLARE_R2_ACCESS_KEY_ID":     os.Getenv("CLOUDFLARE_R2_ACCESS_KEY_ID"),
		"CLOUDFLARE_R2_SECRET_ACCESS_KEY": os.Getenv("CLOUDFLARE_R2_SECRET_ACCESS_KEY"),
	}
	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	tests := []struct {
		name     string
		setup    func()
		required []string
		wantErr  bool
		validate func(t *testing.T, env *E2EEnv)
	}{
		{
			name: "all required vars present",
			setup: func() {
				os.Setenv("CLOUDFLARE_ACCOUNT_ID", "test-account")
				os.Setenv("CLOUDFLARE_ZONE_ID", "test-zone")
				os.Setenv("CLOUDFLARE_DOMAIN", "test.example.com")
			},
			required: []string{"CLOUDFLARE_ACCOUNT_ID", "CLOUDFLARE_ZONE_ID", "CLOUDFLARE_DOMAIN"},
			wantErr:  false,
			validate: func(t *testing.T, env *E2EEnv) {
				if env.AccountID != "test-account" {
					t.Errorf("AccountID = %v, want test-account", env.AccountID)
				}
				if env.ZoneID != "test-zone" {
					t.Errorf("ZoneID = %v, want test-zone", env.ZoneID)
				}
				if env.Domain != "test.example.com" {
					t.Errorf("Domain = %v, want test.example.com", env.Domain)
				}
			},
		},
		{
			name: "missing required var",
			setup: func() {
				os.Setenv("CLOUDFLARE_ACCOUNT_ID", "test-account")
				os.Unsetenv("CLOUDFLARE_ZONE_ID")
			},
			required: []string{"CLOUDFLARE_ACCOUNT_ID", "CLOUDFLARE_ZONE_ID"},
			wantErr:  true,
		},
		{
			name: "empty string treated as missing",
			setup: func() {
				os.Setenv("CLOUDFLARE_ACCOUNT_ID", "test-account")
				os.Setenv("CLOUDFLARE_ZONE_ID", "")
			},
			required: []string{"CLOUDFLARE_ACCOUNT_ID", "CLOUDFLARE_ZONE_ID"},
			wantErr:  true,
		},
		{
			name: "unknown variable name",
			setup: func() {
				os.Setenv("CLOUDFLARE_ACCOUNT_ID", "test-account")
			},
			required: []string{"CLOUDFLARE_UNKNOWN_VAR"},
			wantErr:  true,
		},
		{
			name: "r2 credentials",
			setup: func() {
				os.Setenv("CLOUDFLARE_R2_ACCESS_KEY_ID", "test-key-id")
				os.Setenv("CLOUDFLARE_R2_SECRET_ACCESS_KEY", "test-secret-key")
			},
			required: []string{"CLOUDFLARE_R2_ACCESS_KEY_ID", "CLOUDFLARE_R2_SECRET_ACCESS_KEY"},
			wantErr:  false,
			validate: func(t *testing.T, env *E2EEnv) {
				if env.R2AccessKeyID != "test-key-id" {
					t.Errorf("R2AccessKeyID = %v, want test-key-id", env.R2AccessKeyID)
				}
				if env.R2SecretAccessKey != "test-secret-key" {
					t.Errorf("R2SecretAccessKey = %v, want test-secret-key", env.R2SecretAccessKey)
				}
			},
		},
		{
			name: "no required vars",
			setup: func() {
				os.Setenv("CLOUDFLARE_ACCOUNT_ID", "test-account")
			},
			required: []string{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars
			for k := range originalEnv {
				os.Unsetenv(k)
			}

			// Setup test env
			tt.setup()

			env, err := LoadEnv(tt.required)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, env)
			}
		})
	}
}

func TestEnvForPredefinedSets(t *testing.T) {
	tests := []struct {
		name     string
		envVars  []string
		expected int
	}{
		{"EnvForInit", EnvForInit, 3},
		{"EnvForBackend", EnvForBackend, 7},
		{"EnvForRunner", EnvForRunner, 3},
		{"EnvForBootstrap", EnvForBootstrap, 3},
		{"EnvForClean", EnvForClean, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.envVars) != tt.expected {
				t.Errorf("%s has %d vars, expected %d", tt.name, len(tt.envVars), tt.expected)
			}
		})
	}
}
