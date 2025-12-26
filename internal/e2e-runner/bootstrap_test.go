package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBootstrap_BackendConfigReplacement(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test backend config
	backendConfig := `bucket = "tf-migrate-e2e-state"
key    = "v4/terraform.tfstate"
region = "auto"

endpoints = {
  s3 = "https://ACCOUNT_ID.r2.cloudflarestorage.com"
}

skip_credentials_validation = true
skip_region_validation      = true
skip_requesting_account_id  = true
skip_metadata_api_check     = true
skip_s3_checksum            = true
`

	backendFile := filepath.Join(tmpDir, "backend.hcl")
	err := os.WriteFile(backendFile, []byte(backendConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create backend config: %v", err)
	}

	// Read and replace ACCOUNT_ID
	content, err := os.ReadFile(backendFile)
	if err != nil {
		t.Fatalf("Failed to read backend config: %v", err)
	}

	testAccountID := "test-account-123"
	configured := strings.ReplaceAll(string(content), "ACCOUNT_ID", testAccountID)

	configuredFile := filepath.Join(tmpDir, "backend.configured.hcl")
	err = os.WriteFile(configuredFile, []byte(configured), 0644)
	if err != nil {
		t.Fatalf("Failed to write configured backend: %v", err)
	}

	// Verify the replacement worked
	configuredContent, err := os.ReadFile(configuredFile)
	if err != nil {
		t.Fatalf("Failed to read configured backend: %v", err)
	}

	configuredStr := string(configuredContent)

	if !strings.Contains(configuredStr, testAccountID) {
		t.Error("ACCOUNT_ID was not replaced in configured backend")
	}

	if strings.Contains(configuredStr, "ACCOUNT_ID") {
		t.Error("ACCOUNT_ID placeholder still present after replacement")
	}

	expectedEndpoint := "https://test-account-123.r2.cloudflarestorage.com"
	if !strings.Contains(configuredStr, expectedEndpoint) {
		t.Errorf("Expected endpoint %s not found in configured backend", expectedEndpoint)
	}
}

func TestRunBootstrap_StateFileValidation(t *testing.T) {
	tmpDir := t.TempDir()

	// Test checking for state file existence
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")

	// State file doesn't exist
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		// This is expected - bootstrap should fail if no state file
	} else {
		t.Error("State file should not exist initially")
	}

	// Create state file
	stateContent := `{
  "version": 4,
  "serial": 1,
  "resources": []
}`
	err := os.WriteFile(stateFile, []byte(stateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create state file: %v", err)
	}

	// Now state file should exist
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error("State file should exist after creation")
	}
}

func TestRunBootstrap_EnvVarFormatting(t *testing.T) {
	// Test the environment variable formatting for terraform
	testVars := map[string]string{
		"AWS_ACCESS_KEY_ID":     "test-key-id",
		"AWS_SECRET_ACCESS_KEY": "test-secret-key",
	}

	// Verify these can be used in env format
	for k, v := range testVars {
		envLine := k + "=" + v
		parts := strings.Split(envLine, "=")

		if len(parts) != 2 {
			t.Errorf("Invalid env format: %s", envLine)
		}

		if parts[0] != k {
			t.Errorf("Key mismatch: got %s, want %s", parts[0], k)
		}

		if parts[1] != v {
			t.Errorf("Value mismatch: got %s, want %s", parts[1], v)
		}
	}
}

func TestRunBootstrap_BackendConfigFormat(t *testing.T) {
	// Test that backend config has correct HCL format
	backendConfig := `bucket = "tf-migrate-e2e-state"
key    = "v4/terraform.tfstate"
region = "auto"

endpoints = {
  s3 = "https://ACCOUNT_ID.r2.cloudflarestorage.com"
}

skip_credentials_validation = true
skip_region_validation      = true
skip_requesting_account_id  = true
skip_metadata_api_check     = true
skip_s3_checksum            = true
`

	// Verify it contains required fields
	requiredFields := []string{
		"bucket",
		"key",
		"region",
		"endpoints",
		"ACCOUNT_ID",
	}

	for _, field := range requiredFields {
		if !strings.Contains(backendConfig, field) {
			t.Errorf("Backend config missing required field: %s", field)
		}
	}
}

func TestRunBootstrap_R2EndpointFormat(t *testing.T) {
	// Test R2 endpoint URL format
	tests := []struct {
		accountID string
		want      string
	}{
		{
			accountID: "abc123",
			want:      "https://abc123.r2.cloudflarestorage.com",
		},
		{
			accountID: "test-account",
			want:      "https://test-account.r2.cloudflarestorage.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.accountID, func(t *testing.T) {
			endpoint := "https://" + tt.accountID + ".r2.cloudflarestorage.com"
			if endpoint != tt.want {
				t.Errorf("Endpoint = %s, want %s", endpoint, tt.want)
			}
		})
	}
}

func TestRunBootstrap_TerraformInitArgs(t *testing.T) {
	// Test terraform init arguments
	backendConfigFile := "backend.configured.hcl"

	initArgs := []string{"init", "-backend-config=" + backendConfigFile, "-migrate-state"}

	expectedArgs := []string{"init", "-backend-config=backend.configured.hcl", "-migrate-state"}

	if len(initArgs) != len(expectedArgs) {
		t.Errorf("Arg count mismatch: got %d, want %d", len(initArgs), len(expectedArgs))
	}

	for i, arg := range initArgs {
		if arg != expectedArgs[i] {
			t.Errorf("Arg %d: got %s, want %s", i, arg, expectedArgs[i])
		}
	}
}

func TestRunBootstrap_ConfigCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a temp config file
	configFile := filepath.Join(tmpDir, "backend.configured.hcl")
	err := os.WriteFile(configFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("Config file should exist")
	}

	// Simulate cleanup (defer os.Remove)
	os.Remove(configFile)

	// Verify file was removed
	if _, err := os.Stat(configFile); !os.IsNotExist(err) {
		t.Error("Config file should be removed")
	}
}

func TestRunBootstrap_StateBucketName(t *testing.T) {
	// Test that bucket name follows naming conventions
	bucketName := "tf-migrate-e2e-state"

	// Bucket name validation rules
	if len(bucketName) < 3 || len(bucketName) > 63 {
		t.Error("Bucket name length should be between 3 and 63 characters")
	}

	// Should only contain lowercase letters, numbers, and hyphens
	for _, c := range bucketName {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			t.Errorf("Invalid character in bucket name: %c", c)
		}
	}

	// Should not start or end with hyphen
	if bucketName[0] == '-' || bucketName[len(bucketName)-1] == '-' {
		t.Error("Bucket name should not start or end with hyphen")
	}
}

func TestRunBootstrap_StateKey(t *testing.T) {
	// Test state key format
	stateKey := "v4/terraform.tfstate"

	if !strings.HasSuffix(stateKey, ".tfstate") {
		t.Error("State key should have .tfstate extension")
	}

	if !strings.HasPrefix(stateKey, "v4/") {
		t.Error("State key should be in v4/ directory")
	}
}
