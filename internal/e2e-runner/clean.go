// clean.go provides utilities for managing and cleaning Terraform remote state.
//
// This file implements functionality to remove specific modules from remote
// Terraform state stored in R2, which is useful for cleaning up test resources
// or preparing for targeted migrations. It safely handles state manipulation
// with proper validation and error handling.
package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// RunClean removes specified modules from remote Terraform state in R2
func RunClean(modules []string) error {
	if len(modules) == 0 {
		return fmt.Errorf("no modules specified\nUsage: e2e clean --modules <module1,module2,...>")
	}

	// Load required environment variables
	env, err := LoadEnv(EnvForClean)
	if err != nil {
		return err
	}

	printHeader("Cleaning Modules from Remote State")

	// Get paths
	repoRoot := getRepoRoot()
	e2eRoot := filepath.Join(repoRoot, "e2e")
	v4Dir := filepath.Join(e2eRoot, "tf", "v4")

	// Check if v4 directory exists
	if _, err := os.Stat(v4Dir); os.IsNotExist(err) {
		return fmt.Errorf("v4 directory not found at %s", v4Dir)
	}

	printYellow("Modules to clean:")
	for _, module := range modules {
		printYellow("  - %s", module)
	}
	fmt.Println()

	// Set up terraform runner with R2 credentials
	tf := NewTerraformRunner(v4Dir)
	tf.EnvVars["AWS_ACCESS_KEY_ID"] = env.R2AccessKeyID
	tf.EnvVars["AWS_SECRET_ACCESS_KEY"] = env.R2SecretAccessKey

	// Pull latest state from remote
	printYellow("Pulling latest state from R2...")
	stateJSON, err := tf.Run("state", "pull")
	if err != nil {
		printError("Failed to pull state from R2")
		fmt.Println(stateJSON)
		printError("\nMake sure terraform is initialized:")
		printYellow("  cd %s", v4Dir)
		printYellow("  terraform init -reconfigure -backend-config=backend.configured.hcl")
		return err
	}
	printSuccess("State pulled successfully")

	// Parse state
	var state map[string]interface{}
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return fmt.Errorf("failed to parse state: %w", err)
	}

	// Count resources before cleanup
	resourcesArray, ok := state["resources"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid state format: resources array not found")
	}
	totalBefore := len(resourcesArray)
	printCyan("Total resources before cleanup: %d", totalBefore)
	fmt.Println()

	// Show resources that will be removed
	printYellow("Resources to be removed:")
	for _, module := range modules {
		modulePrefix := "module." + module
		var foundResources []string

		for _, res := range resourcesArray {
			resMap, ok := res.(map[string]interface{})
			if !ok {
				continue
			}

			resModule, ok := resMap["module"].(string)
			if !ok {
				continue
			}

			if resModule == modulePrefix {
				resType, ok := resMap["type"].(string)
				if !ok {
					printYellow("Warning: Skipping resource with invalid type in module %s", modulePrefix)
					continue
				}
				resName, ok := resMap["name"].(string)
				if !ok {
					printYellow("Warning: Skipping resource with invalid name in module %s", modulePrefix)
					continue
				}
				foundResources = append(foundResources, fmt.Sprintf("  - %s.%s", resType, resName))
			}
		}

		if len(foundResources) > 0 {
			printCyan("From %s:", modulePrefix)
			for _, res := range foundResources {
				fmt.Println(res)
			}
		} else {
			printYellow("  (no resources found in %s)", modulePrefix)
		}
	}
	fmt.Println()

	// Clean the state - remove specified modules
	printYellow("Cleaning state...")
	var filteredResources []interface{}
	removed := 0

	for _, res := range resourcesArray {
		resMap, ok := res.(map[string]interface{})
		if !ok {
			filteredResources = append(filteredResources, res)
			continue
		}

		resModule, ok := resMap["module"].(string)
		if !ok {
			filteredResources = append(filteredResources, res)
			continue
		}

		// Check if this resource belongs to any of the modules to clean
		shouldRemove := false
		for _, module := range modules {
			if resModule == "module."+module {
				shouldRemove = true
				removed++
				break
			}
		}

		if !shouldRemove {
			filteredResources = append(filteredResources, res)
		}
	}

	state["resources"] = filteredResources

	// Increment serial number
	if serial, ok := state["serial"].(float64); ok {
		state["serial"] = serial + 1
	}

	// Count resources after cleanup
	totalAfter := len(filteredResources)

	printSuccess("State cleaned")
	printCyan("  Before: %d resources", totalBefore)
	printCyan("  After:  %d resources", totalAfter)
	printSuccess("  Removed: %d resources", removed)
	fmt.Println()

	// Verify modules are gone
	printYellow("Verifying modules removed...")
	for _, module := range modules {
		modulePrefix := "module." + module
		count := 0
		for _, res := range filteredResources {
			if resMap, ok := res.(map[string]interface{}); ok {
				if resModule, ok := resMap["module"].(string); ok && resModule == modulePrefix {
					count++
				}
			}
		}

		if count == 0 {
			printSuccess("%s: 0 resources", modulePrefix)
		} else {
			printError("%s: %d resources remaining", modulePrefix, count)
		}
	}
	fmt.Println()

	// Save cleaned state to temp file
	cleanedStateFile := filepath.Join(v4Dir, "terraform.tfstate.cleaned")
	cleanedStateJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cleaned state: %w", err)
	}

	// Write with restrictive permissions (state files contain sensitive data)
	if err := os.WriteFile(cleanedStateFile, cleanedStateJSON, permSecretFile); err != nil {
		return fmt.Errorf("failed to write cleaned state to %s: %w", cleanedStateFile, err)
	}

	// Push cleaned state back to remote
	printYellow("Pushing cleaned state to R2...")
	output, err := tf.Run("state", "push", cleanedStateFile)
	if err != nil {
		printError("Failed to push state to R2")
		fmt.Println(output)
		printError("\nThe local state has been cleaned but not pushed to remote.")
		printYellow("You can manually push it with: terraform state push %s", cleanedStateFile)
		return err
	}
	printSuccess("State pushed successfully")

	// Only remove temp file after successful push
	if err := os.Remove(cleanedStateFile); err != nil {
		printYellow("Warning: Failed to remove temp state file %s: %v", cleanedStateFile, err)
	}

	fmt.Println()
	printHeader("Remote State Cleaned!")
	fmt.Println()

	printYellow("Summary:")
	printYellow("  - Modules cleaned: %d", len(modules))
	printYellow("  - Resources removed: %d", removed)
	printYellow("  - Resources remaining: %d", totalAfter)
	fmt.Println()

	printYellow("Next steps:")
	printYellow("  The next time you run terraform plan/apply, it will create")
	printYellow("  the resources in these modules from scratch.")
	fmt.Println()

	return nil
}
