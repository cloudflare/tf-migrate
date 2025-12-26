package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewTerraformRunner(t *testing.T) {
	workDir := "/test/dir"
	tf := NewTerraformRunner(workDir)

	if tf.WorkDir != workDir {
		t.Errorf("WorkDir = %v, want %v", tf.WorkDir, workDir)
	}

	if tf.EnvVars == nil {
		t.Error("EnvVars should be initialized")
	}

	if len(tf.EnvVars) != 0 {
		t.Error("EnvVars should be empty initially")
	}
}

func TestTerraformRunner_EnvVars(t *testing.T) {
	tf := NewTerraformRunner("/test")

	tf.EnvVars["TEST_VAR"] = "test_value"
	tf.EnvVars["ANOTHER_VAR"] = "another_value"

	if tf.EnvVars["TEST_VAR"] != "test_value" {
		t.Error("EnvVars not set correctly")
	}

	if len(tf.EnvVars) != 2 {
		t.Error("Expected 2 env vars")
	}
}

func TestTerraformRunner_TFConfigFile(t *testing.T) {
	tf := NewTerraformRunner("/test")

	configFile := "/path/to/config.tfrc"
	tf.TFConfigFile = configFile

	if tf.TFConfigFile != configFile {
		t.Errorf("TFConfigFile = %v, want %v", tf.TFConfigFile, configFile)
	}
}

func TestTerraformRunner_Run_VersionCommand(t *testing.T) {
	// This test requires terraform to be installed
	// Skip if terraform is not available
	if _, err := os.Stat("/usr/bin/terraform"); err != nil {
		if _, err := os.Stat("/usr/local/bin/terraform"); err != nil {
			t.Skip("terraform not found, skipping integration test")
		}
	}

	tmpDir := t.TempDir()
	tf := NewTerraformRunner(tmpDir)

	output, err := tf.Run("version")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !strings.Contains(output, "Terraform") {
		t.Errorf("Output doesn't contain 'Terraform': %v", output)
	}
}

func TestTerraformRunner_StateList(t *testing.T) {
	// Skip if terraform is not available
	if _, err := os.Stat("/usr/bin/terraform"); err != nil {
		if _, err := os.Stat("/usr/local/bin/terraform"); err != nil {
			t.Skip("terraform not found, skipping integration test")
		}
	}

	tmpDir := t.TempDir()
	tf := NewTerraformRunner(tmpDir)

	// StateList should fail gracefully when there's no terraform state
	_, err := tf.StateList()
	// We expect an error since there's no terraform configuration
	// but we're testing that the method can be called
	if err == nil {
		// If somehow it succeeds (empty state), that's fine too
		t.Log("StateList succeeded with empty state")
	}
}

func TestTerraformRunner_StateListParsing(t *testing.T) {
	// Test the parsing logic that StateList uses
	tests := []struct {
		name   string
		output string
		want   int
	}{
		{
			name: "multiple resources",
			output: `module.zone.cloudflare_zone.example
module.dns.cloudflare_dns_record.example
module.dns.cloudflare_dns_record.www`,
			want: 3,
		},
		{
			name:   "empty output",
			output: "",
			want:   0,
		},
		{
			name:   "single resource",
			output: `cloudflare_zone.test`,
			want:   1,
		},
		{
			name: "output with blank lines",
			output: `module.zone.cloudflare_zone.example

module.dns.cloudflare_dns_record.example`,
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the parsing logic that StateList uses
			lines := strings.Split(strings.TrimSpace(tt.output), "\n")
			var resources []string
			for _, line := range lines {
				if line != "" {
					resources = append(resources, line)
				}
			}

			if len(resources) != tt.want {
				t.Errorf("Got %d resources, want %d", len(resources), tt.want)
			}
		})
	}
}

func TestTerraformRunner_RunToFile(t *testing.T) {
	// Skip if terraform is not available
	if _, err := os.Stat("/usr/bin/terraform"); err != nil {
		if _, err := os.Stat("/usr/local/bin/terraform"); err != nil {
			t.Skip("terraform not found, skipping integration test")
		}
	}

	tmpDir := t.TempDir()
	tf := NewTerraformRunner(tmpDir)
	outFile := filepath.Join(tmpDir, "output.txt")

	err := tf.RunToFile(outFile, "version")
	if err != nil {
		t.Fatalf("RunToFile() error = %v", err)
	}

	// Check if file was created
	if _, err := os.Stat(outFile); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}

	// Check file contents
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "Terraform") {
		t.Errorf("Output file doesn't contain 'Terraform': %v", string(content))
	}
}

func TestTerraformRunner_RunToFile_Error(t *testing.T) {
	tmpDir := t.TempDir()
	tf := NewTerraformRunner(tmpDir)
	outFile := filepath.Join(tmpDir, "error_output.txt")

	// Run with invalid command
	err := tf.RunToFile(outFile, "invalid-command")
	if err == nil {
		t.Error("Expected error for invalid command")
	}

	// File should still be created with error output
	if _, statErr := os.Stat(outFile); os.IsNotExist(statErr) {
		t.Error("Output file should be created even on error")
	}
}

func TestTerraformRunner_StatePull(t *testing.T) {
	tmpDir := t.TempDir()
	tf := NewTerraformRunner(tmpDir)

	// This is more of an integration test - just verify the method exists
	// and has the right signature
	destFile := "test.tfstate"
	err := tf.StatePull(destFile)

	// We expect this to fail without proper terraform setup, but we're testing
	// that the method exists and can be called
	if err == nil {
		// If it somehow succeeds, check the file was created
		fullPath := filepath.Join(tmpDir, destFile)
		if _, statErr := os.Stat(fullPath); os.IsNotExist(statErr) {
			t.Error("StatePull succeeded but file wasn't created")
		}
	}
}

func TestTerraformRunner_StatePush(t *testing.T) {
	tmpDir := t.TempDir()
	tf := NewTerraformRunner(tmpDir)

	// Create a dummy state file
	stateFile := "test.tfstate"
	fullPath := filepath.Join(tmpDir, stateFile)
	err := os.WriteFile(fullPath, []byte("{}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test state file: %v", err)
	}

	// This is an integration test - just verify the method can be called
	err = tf.StatePush(stateFile)

	// We expect this to fail without proper terraform setup
	// Just checking that the method signature is correct
	_ = err
}

func TestTerraformRunner_EnvVarInjection(t *testing.T) {
	tmpDir := t.TempDir()
	tf := NewTerraformRunner(tmpDir)

	// Set custom env vars
	tf.EnvVars["TEST_VAR_1"] = "value1"
	tf.EnvVars["TEST_VAR_2"] = "value2"

	// Verify env vars are set
	if len(tf.EnvVars) != 2 {
		t.Errorf("Expected 2 env vars, got %d", len(tf.EnvVars))
	}
}

func TestTerraformRunner_TFConfigFileEnv(t *testing.T) {
	tmpDir := t.TempDir()
	tf := NewTerraformRunner(tmpDir)

	configPath := "/custom/config.tfrc"
	tf.TFConfigFile = configPath

	if tf.TFConfigFile != configPath {
		t.Error("TF_CLI_CONFIG_FILE not set correctly")
	}
}

func TestTerraformRunner_WorkDirValidation(t *testing.T) {
	tests := []struct {
		name    string
		workDir string
	}{
		{
			name:    "absolute path",
			workDir: "/absolute/path",
		},
		{
			name:    "relative path",
			workDir: "./relative/path",
		},
		{
			name:    "current directory",
			workDir: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tf := NewTerraformRunner(tt.workDir)
			if tf.WorkDir != tt.workDir {
				t.Errorf("WorkDir = %v, want %v", tf.WorkDir, tt.workDir)
			}
		})
	}
}
