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
	"time"
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
	// Add parallelism limit to reduce API rate limit issues
	// Check if this is a plan or apply command and parallelism not already set
	if len(args) > 0 && (args[0] == "plan" || args[0] == "apply") {
		hasParallelism := false
		for _, arg := range args {
			if strings.HasPrefix(arg, "-parallelism=") {
				hasParallelism = true
				break
			}
		}
		if !hasParallelism {
			// Insert parallelism flag after the command but before any positional args (like plan file)
			// Find the last flag (starts with -) or insert after command
			insertIndex := 1
			for i := 1; i < len(args); i++ {
				if strings.HasPrefix(args[i], "-") {
					insertIndex = i + 1
				} else {
					// Found first positional argument, insert before it
					break
				}
			}
			// Insert parallelism flag at the correct position
			// Using parallelism=3 to reduce API rate limit issues (default is 10)
			newArgs := make([]string, 0, len(args)+1)
			newArgs = append(newArgs, args[:insertIndex]...)
			newArgs = append(newArgs, "-parallelism=5")
			newArgs = append(newArgs, args[insertIndex:]...)
			args = newArgs
		}
	}

	// Retry logic for rate limiting (429 errors)
	maxRetries := 3
	retryDelay := 5 * time.Second

	var err error
	var output string

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry with exponential backoff
			waitTime := retryDelay * time.Duration(attempt)
			printYellow("Rate limit detected, waiting %v before retry %d/%d...", waitTime, attempt, maxRetries)
			time.Sleep(waitTime)
		}

		// Create command for each attempt
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

		err = cmd.Run()
		output = stdout.String() + stderr.String()

		// Check if this is a rate limit error
		if err != nil && strings.Contains(output, "429 Too Many Requests") {
			if attempt < maxRetries {
				continue // Retry
			}
			// Max retries reached, will return error below
		} else {
			// Success or non-rate-limit error
			break
		}
	}

	// Add a small delay after plan/apply to avoid rate limiting
	if len(args) > 0 && (args[0] == "plan" || args[0] == "apply") {
		time.Sleep(1 * time.Second)
	}

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
