// terraform.go provides a wrapper for executing Terraform commands.
//
// This file implements TerraformRunner, a type that simplifies running
// Terraform commands with consistent configuration:
//   - Environment variable management (credentials, config)
//   - Working directory handling
//   - Output sanitization for security
//   - Standardized error handling and reporting
//
// TerraformRunner is used throughout the e2e tests to interact with
// Terraform in a safe, consistent manner.
package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TerraformRunner handles terraform command execution
type TerraformRunner struct {
	WorkDir            string
	EnvVars            map[string]string
	TFConfigFile       string // For provider overrides
	SanitizeOutput     bool   // Enable output sanitization
	SanitizationConfig *SanitizationConfig
}

// NewTerraformRunner creates a new terraform runner
func NewTerraformRunner(workDir string) *TerraformRunner {
	return &TerraformRunner{
		WorkDir:            workDir,
		EnvVars:            make(map[string]string),
		SanitizeOutput:     true, // Enable by default for security
		SanitizationConfig: DefaultSanitizationConfig(),
	}
}

// Run executes a terraform command
func (tr *TerraformRunner) Run(args ...string) (string, error) {
	cmd := exec.Command("terraform", args...)
	cmd.Dir = tr.WorkDir

	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range tr.EnvVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Add TF_CLI_CONFIG_FILE if set
	if tr.TFConfigFile != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("TF_CLI_CONFIG_FILE=%s", tr.TFConfigFile))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	// Sanitize output if enabled
	if tr.SanitizeOutput {
		output = SanitizeOutput(output, tr.SanitizationConfig)
	}

	if err != nil {
		// Output is already sanitized, safe to include in error
		return output, fmt.Errorf("terraform command failed: %w\nOutput: %s", err, output)
	}

	return output, nil
}

// RunToFile executes a terraform command and writes output to a file
func (tr *TerraformRunner) RunToFile(outFile string, args ...string) error {
	output, err := tr.Run(args...)
	if err != nil {
		// Write output even on error for debugging
		if writeErr := os.WriteFile(outFile, []byte(output), permFile); writeErr != nil {
			// Log but don't override the original error
			printYellow("Warning: Failed to write output to %s: %v", outFile, writeErr)
		}
		return err
	}

	return os.WriteFile(outFile, []byte(output), permFile)
}

// StateList lists resources in terraform state
func (tr *TerraformRunner) StateList() ([]string, error) {
	output, err := tr.Run("state", "list")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var resources []string
	for _, line := range lines {
		if line != "" {
			resources = append(resources, line)
		}
	}

	return resources, nil
}

// StatePull pulls state to local file
func (tr *TerraformRunner) StatePull(destFile string) error {
	output, err := tr.Run("state", "pull")
	if err != nil {
		return err
	}

	// Use restrictive permissions for state files (contain sensitive data)
	destPath := filepath.Join(tr.WorkDir, destFile)
	return os.WriteFile(destPath, []byte(output), permSecretFile)
}

// StatePush pushes local state file to remote
func (tr *TerraformRunner) StatePush(stateFile string) error {
	_, err := tr.Run("state", "push", stateFile)
	return err
}
