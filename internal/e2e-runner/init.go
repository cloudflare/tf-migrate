// init.go handles initialization of test infrastructure and Terraform workspaces.
//
// This file provides functionality to set up e2e test environments, including:
//   - Initializing Terraform workspaces with appropriate providers
//   - Configuring backend for remote state storage
//   - Listing available test modules
//   - Preparing directories and dependencies for test execution
//
// Proper initialization is critical for ensuring tests run in a clean,
// reproducible environment.
package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunInit syncs resource files from integration testdata to e2e/v4
func RunInit(resources string) error {
	// Load required environment variables
	env, err := LoadEnv(EnvForInit)
	if err != nil {
		return err
	}

	// Parse resource filter if provided
	var targetResources []string
	if resources != "" {
		for _, r := range strings.Split(resources, ",") {
			targetResources = append(targetResources, strings.TrimSpace(r))
		}
	}

	// Get paths
	repoRoot := getRepoRoot()
	e2eRoot := filepath.Join(repoRoot, "e2e")
	v4Dir := filepath.Join(e2eRoot, "tf", "v4")
	testdataRoot := filepath.Join(repoRoot, "integration", "v4_to_v5", "testdata")

	// Check if testdata directory exists
	if _, err := os.Stat(testdataRoot); os.IsNotExist(err) {
		return fmt.Errorf("testdata directory not found at %s", testdataRoot)
	}

	printHeader("Syncing Test Resources")

	if len(targetResources) > 0 {
		printYellow("Filtering to specific resources: %s", strings.Join(targetResources, ", "))
	}

	// Create v4 directory if it doesn't exist
	if err := os.MkdirAll(v4Dir, permDir); err != nil {
		return fmt.Errorf("failed to create v4 directory %s: %w", v4Dir, err)
	}

	// Sync resource files from testdata
	printYellow("Syncing resource files from testdata...")
	fileCount := 0
	var moduleNames []string

	// Find all input directories
	inputDirs, err := findInputDirs(testdataRoot)
	if err != nil {
		return fmt.Errorf("failed to find input directories: %w", err)
	}

	// Filter directories if resources specified
	if len(targetResources) > 0 {
		var filteredDirs []string
		for _, dir := range inputDirs {
			// Extract resource type from parent directory (e.g., "zone_dnssec" from ".../zone_dnssec/input")
			resourceType := filepath.Base(filepath.Dir(dir))
			for _, targetResource := range targetResources {
				if resourceType == targetResource {
					filteredDirs = append(filteredDirs, dir)
					break
				}
			}
		}
		inputDirs = filteredDirs
	}

	for _, inputDir := range inputDirs {
		resourceType := filepath.Base(filepath.Dir(inputDir))
		moduleNames = append(moduleNames, resourceType)

		// Create/clean module directory
		moduleDir := filepath.Join(v4Dir, resourceType)
		if err := os.RemoveAll(moduleDir); err != nil {
			return fmt.Errorf("failed to clean module directory: %w", err)
		}
		if err := os.MkdirAll(moduleDir, permDir); err != nil {
			return fmt.Errorf("failed to create module directory %s: %w", moduleDir, err)
		}

		// Check for *_e2e.tf files first
		e2eFiles, err := filepath.Glob(filepath.Join(inputDir, "*_e2e.tf"))
		if err != nil {
			return fmt.Errorf("failed to glob e2e files: %w", err)
		}

		if len(e2eFiles) > 0 {
			// Use e2e-specific files
			for _, tfFile := range e2eFiles {
				filename := filepath.Base(tfFile)
				destFilename := strings.Replace(filename, "_e2e.tf", ".tf", 1)
				destFile := filepath.Join(moduleDir, destFilename)

				if err := copyFile(tfFile, destFile); err != nil {
					return fmt.Errorf("failed to copy file: %w", err)
				}

				printGreen("  ✓ %s/%s (from %s)", resourceType, destFilename, filename)
				fileCount++
			}
		} else {
			// Fall back to regular .tf files
			tfFiles, err := filepath.Glob(filepath.Join(inputDir, "*.tf"))
			if err != nil {
				return fmt.Errorf("failed to glob tf files: %w", err)
			}

			for _, tfFile := range tfFiles {
				filename := filepath.Base(tfFile)
				destFile := filepath.Join(moduleDir, filename)

				if err := copyFile(tfFile, destFile); err != nil {
					return fmt.Errorf("failed to copy file: %w", err)
				}

				printGreen("  ✓ %s/%s", resourceType, filename)
				fileCount++
			}
		}

		// Create versions.tf for each module
		versionsFile := filepath.Join(moduleDir, "versions.tf")
		versionsContent := `terraform {
  required_providers {
    cloudflare = {
      source = "cloudflare/cloudflare"
    }
  }
}
`
		if err := os.WriteFile(versionsFile, []byte(versionsContent), permFile); err != nil {
			return fmt.Errorf("failed to create versions.tf at %s: %w", versionsFile, err)
		}

		printGreen("  ✓ %s/versions.tf", resourceType)
		fileCount++
	}

	fmt.Println()
	printGreen("  Total: %d files synced", fileCount)
	fmt.Println()

	// Configure terraform variables
	printYellow("Configuring terraform variables...")
	fmt.Println()

	// Check if provider.tf already has cloudflare_domain variable
	providerTfPath := filepath.Join(v4Dir, "provider.tf")
	providerContent, err := os.ReadFile(providerTfPath)
	if err == nil && !strings.Contains(string(providerContent), "variable \"cloudflare_domain\"") {
		// Append cloudflare_domain variable to provider.tf
		domainVarContent := `
variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}
`
		f, err := os.OpenFile(providerTfPath, os.O_APPEND|os.O_WRONLY, permFile)
		if err != nil {
			return fmt.Errorf("failed to open provider.tf at %s: %w", providerTfPath, err)
		}
		defer f.Close()

		if _, err := f.WriteString(domainVarContent); err != nil {
			return fmt.Errorf("failed to append to provider.tf: %w", err)
		}
		printGreen("  ✓ Added cloudflare_domain variable to provider.tf")
	}

	tfvarsContent := fmt.Sprintf(`# Auto-generated by init script
# Edit and re-run ./scripts/init to update these values

cloudflare_account_id     = "%s"
cloudflare_zone_id        = "%s"
cloudflare_domain         = "%s"
crowdstrike_client_id     = "%s"
crowdstrike_client_secret = "%s"
crowdstrike_api_url       = "%s"
crowdstrike_customer_id   = "%s"
`, env.AccountID, env.ZoneID, env.Domain, env.CrowdstrikeClientID, env.CrowdstrikeClientSecret, env.CrowdstrikeAPIURL, env.CrowdstrikeCustomerID)

	tfvarsPath := filepath.Join(v4Dir, "terraform.tfvars")
	if err := os.WriteFile(tfvarsPath, []byte(tfvarsContent), permFile); err != nil {
		return fmt.Errorf("failed to write terraform.tfvars to %s: %w", tfvarsPath, err)
	}

	fmt.Println()
	printSuccess("Saved configuration")
	printBlue("    Account ID: %s", env.AccountID)
	printBlue("    Zone ID: %s", env.ZoneID)
	printBlue("    Domain: %s", env.Domain)
	printGreen("    File: v4/terraform.tfvars")
	fmt.Println()

	// Scan for import annotations before generating main.tf
	printYellow("Scanning for import annotations...")
	importSpecs, err := findImportSpecs(v4Dir, moduleNames)
	if err != nil {
		return fmt.Errorf("failed to scan for import annotations: %w", err)
	}

	if len(importSpecs) > 0 {
		printGreen("  ✓ Found %d resource(s) requiring import", len(importSpecs))
		for _, spec := range importSpecs {
			printBlue("    - module.%s.%s", spec.ModuleName, spec.ResourceAddress)
		}
	}
	fmt.Println()

	// Update main.tf with all discovered modules
	printYellow("Updating main.tf with module references...")

	mainTfContent := `# Main configuration file that imports all resource modules
# This file ties together all the resource-specific configurations

# Each resource type is in its own subdirectory

`

	// Add import blocks if any resources need importing
	if len(importSpecs) > 0 {
		mainTfContent += generateImportBlocks(importSpecs)
	}

	for _, moduleName := range moduleNames {
		// Scan module directory for declared variables
		moduleDir := filepath.Join(v4Dir, moduleName)
		moduleVars, err := getModuleVariables(moduleDir)
		if err != nil {
			return fmt.Errorf("failed to scan variables for module %s: %w", moduleName, err)
		}

		// Build module block with only variables that are declared in the module
		moduleBlock := fmt.Sprintf("\nmodule \"%s\" {\n  source = \"./%s\"\n\n", moduleName, moduleName)

		// Always pass cloudflare variables if they exist
		if contains(moduleVars, "cloudflare_account_id") {
			moduleBlock += "  cloudflare_account_id = var.cloudflare_account_id\n"
		}
		if contains(moduleVars, "cloudflare_zone_id") {
			moduleBlock += "  cloudflare_zone_id    = var.cloudflare_zone_id\n"
		}
		if contains(moduleVars, "cloudflare_domain") {
			moduleBlock += "  cloudflare_domain     = var.cloudflare_domain\n"
		}

		// Only pass crowdstrike variables if the module declares them
		if contains(moduleVars, "crowdstrike_client_id") {
			moduleBlock += "  crowdstrike_client_id     = var.crowdstrike_client_id\n"
		}
		if contains(moduleVars, "crowdstrike_client_secret") {
			moduleBlock += "  crowdstrike_client_secret = var.crowdstrike_client_secret\n"
		}
		if contains(moduleVars, "crowdstrike_api_url") {
			moduleBlock += "  crowdstrike_api_url       = var.crowdstrike_api_url\n"
		}
		if contains(moduleVars, "crowdstrike_customer_id") {
			moduleBlock += "  crowdstrike_customer_id   = var.crowdstrike_customer_id\n"
		}

		moduleBlock += "}\n"
		mainTfContent += moduleBlock
	}

	mainTfPath := filepath.Join(v4Dir, "main.tf")
	if err := os.WriteFile(mainTfPath, []byte(mainTfContent), permFile); err != nil {
		return fmt.Errorf("failed to write main.tf to %s: %w", mainTfPath, err)
	}

	if len(importSpecs) > 0 {
		printGreen("  ↻ Updated main.tf with %d module references and %d import blocks", len(moduleNames), len(importSpecs))
	} else {
		printGreen("  ↻ Updated main.tf with %d module references", len(moduleNames))
	}

	fmt.Println()
	printHeader("✓ Sync Complete!")

	printYellow("Summary:")
	printBlue("  - Terraform v4 configs: %s", v4Dir)
	printGreen("  - Modules: %d", len(moduleNames))
	printGreen("  - Files synced: %d", fileCount)
	fmt.Println()

	// Validate that we found resources to test
	if len(moduleNames) == 0 && fileCount == 0 {
		fmt.Println()
		printError("No resources found to test!")
		fmt.Println()
		if len(targetResources) > 0 {
			printYellow("The specified resources were not found in testdata:")
			for _, r := range targetResources {
				printRed("  - %s", r)
			}
			fmt.Println()
			printYellow("Available resources in testdata:")
			allDirs, _ := findInputDirs(testdataRoot)
			if len(allDirs) > 0 {
				for _, dir := range allDirs {
					resourceType := filepath.Base(filepath.Dir(dir))
					printBlue("  - %s", resourceType)
				}
			} else {
				printRed("  No resources found in testdata")
			}
		} else {
			printYellow("No testdata found in %s", testdataRoot)
		}
		fmt.Println()
		return fmt.Errorf("cannot run e2e tests with 0 modules and 0 files")
	}

	// Configure remote backend
	printYellow("Configuring remote backend...")

	// Validate additional environment variables needed for backend
	_, err = LoadEnv(EnvForBackend)
	if err != nil {
		return err
	}

	printSuccess("Backend configured")
	fmt.Println()

	// Check if terraform is initialized
	terraformDir := filepath.Join(v4Dir, ".terraform")
	if _, err := os.Stat(terraformDir); err == nil {
		printBlue("  ✓ Provider installation preserved")
		printBlue("  ✓ Backend already configured")
	} else {
		printYellow("  Note: Run 'cd tf/v4 && terraform init -backend-config=backend.configured.hcl' to initialize")
	}

	fmt.Println()
	printYellow("Next steps:")
	if _, err := os.Stat(terraformDir); err != nil {
		printGreen("  cd tf/v4 && terraform init -backend-config=backend.configured.hcl && terraform apply")
	} else {
		printGreen("  cd tf/v4 && terraform apply")
	}
	fmt.Println()
	printBlue("Note: Configuration is automatically loaded from terraform.tfvars")
	printBlue("      State is managed remotely in R2")
	fmt.Println()

	return nil
}

// findInputDirs finds all "input" directories in testdata
func findInputDirs(root string) ([]string, error) {
	var inputDirs []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && info.Name() == "input" {
			inputDirs = append(inputDirs, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Strings(inputDirs)
	return inputDirs, nil
}

// getModuleVariables scans a module directory for variable declarations
func getModuleVariables(moduleDir string) ([]string, error) {
	var variables []string

	// Read all .tf files in the module directory
	entries, err := os.ReadDir(moduleDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read module directory %s: %w", moduleDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tf") {
			continue
		}

		filePath := filepath.Join(moduleDir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue // Skip files we can't read
		}

		// Simple pattern matching for variable declarations
		// Look for: variable "name" {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "variable") {
				// Extract variable name from: variable "name" {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					varName := strings.Trim(parts[1], `"`)
					variables = append(variables, varName)
				}
			}
		}
	}

	return variables, nil
}

