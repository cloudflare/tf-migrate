// migrate.go orchestrates the Terraform provider migration process.
//
// This file implements the core migration workflow that converts Terraform
// configurations from one provider version to another (e.g., v4 to v5).
// Key responsibilities include:
//   - Building the tf-migrate binary with latest code changes
//   - Copying and preparing source configurations for migration
//   - Executing the migration tool with appropriate parameters
//   - Filtering and managing state files for targeted migrations
//   - Updating provider versions and backend configurations
//
// The migration process preserves .terraform directories and lock files
// to avoid unnecessary provider re-downloads during testing.
package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RunMigrate copies v4/ to migrated-v4_to_v5/ and runs migration
func RunMigrate(resources string) error {
	repoRoot := getRepoRoot()
	e2eRoot := filepath.Join(repoRoot, "e2e")
	v4Dir := filepath.Join(e2eRoot, "tf", "v4")
	generatedDir := filepath.Join(e2eRoot, "migrated-v4_to_v5")
	binary := filepath.Join(repoRoot, "bin", "tf-migrate")

	// Build the binary
	printYellow("Building tf-migrate binary...")
	if err := buildBinary(repoRoot); err != nil {
		return fmt.Errorf("failed to build binary: %w", err)
	}

	// Check if v4 directory exists
	if _, err := os.Stat(v4Dir); os.IsNotExist(err) {
		return fmt.Errorf("v4 directory not found at %s\nrun ./scripts/init first", v4Dir)
	}

	printHeader("Running v4 to v5 Migration")

	// Clean and prepare generated directory
	printYellow("Preparing output directory...")
	if err := os.MkdirAll(generatedDir, permDir); err != nil {
		return fmt.Errorf("failed to create generated directory %s: %w", generatedDir, err)
	}

	// Clean existing contents but preserve .terraform directory and lock file
	entries, err := os.ReadDir(generatedDir)
	if err != nil {
		return fmt.Errorf("failed to read generated directory: %w", err)
	}

	for _, entry := range entries {
		if entry.Name() == ".terraform" || entry.Name() == ".terraform.lock.hcl" {
			continue
		}
		path := filepath.Join(generatedDir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}

	// Check if .terraform exists
	if _, err := os.Stat(filepath.Join(generatedDir, ".terraform")); err == nil {
		printBlue("  ✓ Preserved v5 provider installation (.terraform/)")
	}
	if _, err := os.Stat(filepath.Join(generatedDir, ".terraform.lock.hcl")); err == nil {
		printBlue("  ✓ Preserved v5 dependency lock file (.terraform.lock.hcl)")
	}

	// Copy files based on whether specific resources are targeted
	var resourceList []string
	if resources != "" {
		resourceList = strings.Split(resources, ",")
		if err := copyTargetedResources(v4Dir, generatedDir, resourceList); err != nil {
			return fmt.Errorf("failed to copy targeted resources: %w", err)
		}
	} else {
		if err := copyAllResources(v4Dir, generatedDir); err != nil {
			return fmt.Errorf("failed to copy all resources: %w", err)
		}
	}

	// Update provider.tf to use v5.0 and remove backend config
	if err := updateProviderTF(generatedDir); err != nil {
		return fmt.Errorf("failed to update provider.tf: %w", err)
	}

	// Filter state file if targeting specific resources
	if resources != "" {
		if err := filterStateFile(generatedDir, resourceList); err != nil {
			return fmt.Errorf("failed to filter state file: %w", err)
		}
	}

	fmt.Println()

	// Run migration.
	//
	// Why tf-migrate must rename resource types in the state even for provider state upgraders:
	//
	// Terraform reads and decodes every resource in the state file using the current provider's
	// schema BEFORE it processes any `moved` blocks in the configuration. This means that even
	// when a perfect `moved { from = cloudflare_tunnel.x, to = cloudflare_zero_trust_tunnel_cloudflared.x }`
	// block exists in the transformed config, and even when the provider implements MoveState
	// with a SourceSchema for the old type, Terraform will error with "no schema available for
	// <old_type>" during the state read phase — before MoveState is ever called.
	//
	// The MoveState provider hook therefore cannot rescue resources whose old type has no schema
	// in the new provider version. The state type rename MUST happen before `terraform apply`.
	//
	// For resources with UsesProviderStateUpgrader(), TransformState is a deliberate no-op:
	// tf-migrate renames the resource type via CanHandle/GetResourceType (so Terraform can read
	// the state), but leaves all attributes untouched. The provider's UpgradeState handler then
	// transforms the attributes during terraform apply.
	printYellow("Migrating configuration and state files...")
	args := []string{
		"--config-dir", generatedDir,
		"--source-version", "v4",
		"--target-version", "v5",
		"migrate",
		"--backup=false",
		"--recursive",
	}

	// Always include --state-file when the state file exists
	stateFile := filepath.Join(generatedDir, "terraform.tfstate")
	if _, err := os.Stat(stateFile); err == nil {
		args = append([]string{args[0], args[1], "--state-file", stateFile}, args[2:]...)
	}

	cmd := exec.Command(binary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		printError("Migration failed")
		return err
	}

	printSuccess("Migration complete (config transformed, state will be upgraded by provider)")

	// Apply post-migration patches (e.g., adding required fields that tf-migrate can't auto-generate)
	// Looks for integration/v4_to_v5/testdata/<resource>/postmigrate/ directories
	if err := applyPostMigrationPatches(repoRoot, generatedDir, resourceList); err != nil {
		printYellow("Warning: Failed to apply post-migration patches: %v", err)
	}

	// Clean up backup files
	if err := filepath.Walk(generatedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".backup") || strings.HasSuffix(info.Name(), ".tfstate.backup") {
			if err := os.Remove(path); err != nil {
				printYellow("Warning: Failed to remove backup file %s: %v", path, err)
			}
		}
		return nil
	}); err != nil {
		printYellow("Warning: Failed to walk directory for backup cleanup: %v", err)
	}

	fmt.Println()
	printHeader("✓ Migration Complete!")

	printYellow("Results:")
	printBlue("  Input (v4):  %s", v4Dir)
	printBlue("  Output (v5): %s", generatedDir)
	fmt.Println()

	printYellow("Next steps:")
	printYellow("  cd %s", generatedDir)
	printYellow("  terraform init")
	printYellow("  terraform plan")
	fmt.Println()

	return nil
}

// applyPostMigrationPatches applies patches from testdata/<resource>/postmigrate/ to
// migrated config files. Each patch file contains lines to inject (one attribute per line)
// and targets the .tf file matching the resource name in the migrated module directory.
// This supports resources that need manual config additions after tf-migrate runs
// (e.g., predefined profiles needing profile_id).
func applyPostMigrationPatches(repoRoot, generatedDir string, resources []string) error {
	testdataDir := filepath.Join(repoRoot, "integration", "v4_to_v5", "testdata")

	for _, resource := range resources {
		patchDir := filepath.Join(testdataDir, resource, "postmigrate")
		if _, err := os.Stat(patchDir); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(patchDir)
		if err != nil {
			return fmt.Errorf("failed to read postmigrate dir for %s: %w", resource, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".patch") {
				continue
			}

			patchFile := filepath.Join(patchDir, entry.Name())
			patchData, err := os.ReadFile(patchFile)
			if err != nil {
				return fmt.Errorf("failed to read patch %s: %w", patchFile, err)
			}

			// Parse patch: first line is target resource type.name, rest are lines to inject
			lines := strings.Split(strings.TrimSpace(string(patchData)), "\n")
			if len(lines) < 2 {
				continue
			}
			targetResource := strings.TrimSpace(lines[0])
			patchLines := strings.Join(lines[1:], "\n")

			// Find and patch the migrated .tf file
			moduleDir := filepath.Join(generatedDir, resource)
			tfFiles, _ := filepath.Glob(filepath.Join(moduleDir, "*.tf"))
			for _, tfFile := range tfFiles {
				content, err := os.ReadFile(tfFile)
				if err != nil {
					continue
				}
				fileContent := string(content)
				// Find the resource block and inject after the opening brace
				marker := fmt.Sprintf("resource \"%s\"", targetResource)
				if idx := strings.Index(fileContent, marker); idx >= 0 {
					// Find the opening brace after the resource declaration
					braceIdx := strings.Index(fileContent[idx:], "{")
					if braceIdx >= 0 {
						insertPos := idx + braceIdx + 1
						patched := fileContent[:insertPos] + "\n" + patchLines + fileContent[insertPos:]
						if err := os.WriteFile(tfFile, []byte(patched), permFile); err != nil {
							return fmt.Errorf("failed to write patched file %s: %w", tfFile, err)
						}
						printBlue("  ✓ Applied post-migration patch to %s/%s", resource, filepath.Base(tfFile))
					}
				}
			}
		}
	}

	return nil
}

// copyTargetedResources copies only specific modules and required root files
func copyTargetedResources(srcDir, dstDir string, resources []string) error {
	printYellow("Copying only targeted resources: %s", strings.Join(resources, ","))

	// Copy required root files
	rootFiles := []string{"provider.tf", "variables.tf", "terraform.tfvars", "terraform.tfstate"}
	copiedRootFiles := 0
	for _, file := range rootFiles {
		src := filepath.Join(srcDir, file)
		dst := filepath.Join(dstDir, file)
		if _, err := os.Stat(src); err == nil {
			if err := copyFile(src, dst); err != nil {
				return err
			}
			printGreen("    ✓ Copied root file: %s", file)
			copiedRootFiles++
		}
	}
	if copiedRootFiles > 0 {
		fmt.Println()
	}

	// Copy targeted module directories
	for _, resource := range resources {
		resource = strings.TrimSpace(resource)
		srcModuleDir := filepath.Join(srcDir, resource)
		dstModuleDir := filepath.Join(dstDir, resource)

		if _, err := os.Stat(srcModuleDir); os.IsNotExist(err) {
			printRed("    ✗ Module not found: %s", resource)
			continue
		}

		if err := copyDir(srcModuleDir, dstModuleDir); err != nil {
			return fmt.Errorf("failed to copy module %s: %w", resource, err)
		}

		printGreen("    ✓ Copied module: %s", resource)
	}

	// Create filtered main.tf with only targeted modules
	printYellow("Creating filtered main.tf...")
	if err := createFilteredMainTF(srcDir, dstDir, resources); err != nil {
		return err
	}

	printSuccess("Copied targeted resources to migrated-v4_to_v5/")
	return nil
}

// copyAllResources copies all files using rsync-like behavior
func copyAllResources(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	excludes := map[string]bool{
		".terraform":             true,
		".terraform.lock.hcl":    true,
		"backend.hcl":            true,
		"backend.configured.hcl": true,
	}

	fileCount := 0
	dirCount := 0
	for _, entry := range entries {
		if excludes[entry.Name()] {
			continue
		}

		src := filepath.Join(srcDir, entry.Name())
		dst := filepath.Join(dstDir, entry.Name())

		if entry.IsDir() {
			printCyan("  Copying directory: %s/", entry.Name())
			if err := copyDir(src, dst); err != nil {
				return err
			}
			dirCount++
		} else {
			if err := copyFile(src, dst); err != nil {
				return err
			}
			fileCount++
		}
	}

	if fileCount > 0 || dirCount > 0 {
		fmt.Println()
		printGreen("  Copied %d directories and %d files", dirCount, fileCount)
	}

	printSuccess("Copied tf/v4/ to migrated-v4_to_v5/ (including state file for migration)")
	return nil
}

// createFilteredMainTF creates a main.tf with only specified modules
func createFilteredMainTF(srcDir, dstDir string, resources []string) error {
	content := `# Main configuration file that imports all resource modules
# This file ties together all the resource-specific configurations

# Each resource type is in its own subdirectory


`

	// Read original main.tf to extract module blocks
	srcMainTF := filepath.Join(srcDir, "main.tf")
	srcContent, err := os.ReadFile(srcMainTF)
	if err != nil {
		return err
	}

	// Extract each module block
	for _, resource := range resources {
		resource = strings.TrimSpace(resource)
		moduleBlock := extractModuleBlock(string(srcContent), resource)
		if moduleBlock != "" {
			content += moduleBlock + "\n"
		}
	}

	dstMainTF := filepath.Join(dstDir, "main.tf")
	return os.WriteFile(dstMainTF, []byte(content), permFile)
}

// extractModuleBlock extracts a module block from terraform config
func extractModuleBlock(content, moduleName string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inModule := false
	braceCount := 0

	for _, line := range lines {
		if strings.Contains(line, fmt.Sprintf(`module "%s"`, moduleName)) {
			inModule = true
		}

		if inModule {
			result = append(result, line)

			// Count braces to find end of block
			braceCount += strings.Count(line, "{")
			braceCount -= strings.Count(line, "}")

			if braceCount == 0 && strings.Contains(line, "}") {
				break
			}
		}
	}

	return strings.Join(result, "\n")
}

// updateProviderTF updates provider.tf to use v5.0 and removes backend config
func updateProviderTF(dir string) error {
	providerFile := filepath.Join(dir, "provider.tf")
	if _, err := os.Stat(providerFile); os.IsNotExist(err) {
		return nil // No provider.tf to update
	}

	content, err := os.ReadFile(providerFile)
	if err != nil {
		return err
	}

	// Update version to 5.0
	updatedContent := strings.ReplaceAll(string(content), `version = "~> 4.0"`, `version = "~> 5.0"`)

	// Remove backend configuration (find and remove lines between "# Remote state backend" and "backend "s3" {}")
	lines := strings.Split(updatedContent, "\n")
	var filtered []string
	skipUntilBackendEnd := false

	for _, line := range lines {
		if strings.Contains(line, "# Remote state backend") {
			skipUntilBackendEnd = true
			continue
		}

		if skipUntilBackendEnd {
			if strings.Contains(line, `backend "s3" {}`) {
				skipUntilBackendEnd = false
				continue
			}
			continue
		}

		filtered = append(filtered, line)
	}

	updatedContent = strings.Join(filtered, "\n")

	if err := os.WriteFile(providerFile, []byte(updatedContent), permFile); err != nil {
		return fmt.Errorf("failed to write provider.tf to %s: %w", providerFile, err)
	}

	printSuccess("Updated provider.tf to use ~> 5.0 and removed backend config")
	return nil
}

// filterStateFile filters terraform.tfstate to only include targeted resources
func filterStateFile(dir string, resources []string) error {
	stateFile := filepath.Join(dir, "terraform.tfstate")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		return nil // No state file to filter
	}

	printYellow("Filtering state file to only include targeted resources...")

	// Read state file
	stateData, err := os.ReadFile(stateFile)
	if err != nil {
		return err
	}

	// Parse JSON
	var state map[string]interface{}
	if err := json.Unmarshal(stateData, &state); err != nil {
		return err
	}

	// Filter resources
	if resourcesArray, ok := state["resources"].([]interface{}); ok {
		var filtered []interface{}

		for _, res := range resourcesArray {
			resMap, ok := res.(map[string]interface{})
			if !ok {
				continue
			}

			module, ok := resMap["module"].(string)
			if !ok {
				continue
			}

			// Check if resource belongs to any targeted module
			shouldKeep := false
			for _, targetModule := range resources {
				targetModule = strings.TrimSpace(targetModule)
				if module == fmt.Sprintf("module.%s", targetModule) ||
					strings.HasPrefix(module, fmt.Sprintf("module.%s.", targetModule)) {
					shouldKeep = true
					break
				}
			}

			if shouldKeep {
				filtered = append(filtered, res)
			}
		}

		state["resources"] = filtered

		// Count instances
		instanceCount := 0
		for _, res := range filtered {
			if resMap, ok := res.(map[string]interface{}); ok {
				if instances, ok := resMap["instances"].([]interface{}); ok {
					instanceCount += len(instances)
				}
			}
		}

		printSuccess("Filtered state to %d resources from targeted modules", instanceCount)
	}

	// Write filtered state
	filteredData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	// Use restrictive permissions for state files (contain sensitive data)
	return os.WriteFile(stateFile, filteredData, permSecretFile)
}

// buildBinary builds the tf-migrate binary
func buildBinary(repoRoot string) error {
	// Ensure bin directory exists
	binDir := filepath.Join(repoRoot, "bin")
	if err := os.MkdirAll(binDir, permDir); err != nil {
		return fmt.Errorf("failed to create bin directory %s: %w", binDir, err)
	}

	// Use absolute path for output binary
	outputPath := filepath.Join(repoRoot, "bin", "tf-migrate")
	cmd := exec.Command("go", "build", "-o", outputPath, "./cmd/tf-migrate")
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	printSuccess("Binary built successfully")
	return nil
}
