// bootstrap.go handles migration of local Terraform state to remote R2 backend.
//
// This file provides functionality to transition from local state files to
// remote state storage in Cloudflare R2, which is necessary for collaborative
// workflows and state management across different environments.
package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RunBootstrap migrates local state to R2 remote backend
func RunBootstrap() error {
	// Load required environment variables
	env, err := LoadEnv(EnvForBootstrap)
	if err != nil {
		return err
	}

	printHeader("Bootstrap Remote State")

	printGreen("Using credentials:")
	printCyan("  User:       %s", env.Email)
	printCyan("  Account ID: %s", env.AccountID)
	printCyan("  Zone ID:    %s", env.ZoneID)
	printCyan("  R2 Key ID:  %s", env.R2AccessKeyID)
	fmt.Println()

	// Get paths
	repoRoot := getRepoRoot()
	e2eRoot := filepath.Join(repoRoot, "e2e")
	v4Dir := filepath.Join(e2eRoot, "tf", "v4")

	// Check if local state exists
	stateFile := filepath.Join(v4Dir, "terraform.tfstate")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		return fmt.Errorf("local state file not found at %s\nplease run terraform apply first to create initial state", stateFile)
	}

	printYellow("Current state file:")
	printYellow("  %s", stateFile)
	fmt.Println()

	// Create backend config with account ID
	backendConfig := filepath.Join(v4Dir, "backend.hcl")
	backendConfigTmp := filepath.Join(v4Dir, "backend.configured.hcl")

	printYellow("Configuring backend with account ID...")
	backendContent, err := os.ReadFile(backendConfig)
	if err != nil {
		return fmt.Errorf("failed to read backend config: %w", err)
	}

	configuredContent := strings.ReplaceAll(string(backendContent), "ACCOUNT_ID", env.AccountID)
	if err := os.WriteFile(backendConfigTmp, []byte(configuredContent), permFile); err != nil {
		return fmt.Errorf("failed to write backend config to %s: %w", backendConfigTmp, err)
	}
	defer func() {
		if err := os.Remove(backendConfigTmp); err != nil && !os.IsNotExist(err) {
			printYellow("Warning: Failed to remove temp backend config %s: %v", backendConfigTmp, err)
		}
	}()

	// Initialize terraform with remote backend
	printYellow("Initializing Terraform with remote backend...")

	tf := NewTerraformRunner(v4Dir)
	tf.EnvVars["AWS_ACCESS_KEY_ID"] = env.R2AccessKeyID
	tf.EnvVars["AWS_SECRET_ACCESS_KEY"] = env.R2SecretAccessKey

	// Run init with backend config and migrate state
	initArgs := []string{"init", "-backend-config=" + backendConfigTmp, "-migrate-state"}
	output, err := tf.Run(initArgs...)
	if err != nil {
		printError("Failed to initialize with remote backend")
		fmt.Println(output)
		return err
	}

	fmt.Println()
	printSuccess("State successfully migrated to remote backend!")
	fmt.Println()

	printCyan("Remote state location:")
	printYellow("  Bucket: tf-migrate-e2e-state")
	printYellow("  Key: v4/terraform.tfstate")
	printYellow("  Endpoint: https://%s.r2.cloudflarestorage.com", env.AccountID)
	fmt.Println()

	printYellow("Note: Local state file should now be deleted (Terraform does this automatically).")
	printYellow("The state is now managed remotely in R2.")
	fmt.Println()

	printSuccess("Bootstrap complete!")
	return nil
}
