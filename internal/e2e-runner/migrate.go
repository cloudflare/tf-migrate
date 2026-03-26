// migrate.go orchestrates the Terraform provider migration process.
//
// This file implements the core migration workflow that converts Terraform
// configurations from one provider version to another (e.g., v4 to v5).
// Key responsibilities include:
//   - Building the tf-migrate binary with latest code changes
//   - Copying and preparing source configurations for migration
//   - Executing the migration tool with appropriate parameters
//   - Updating provider versions and backend configurations
//
// The migration process preserves .terraform directories and lock files
// to avoid unnecessary provider re-downloads during testing.
package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RunMigrate copies v4/ to migrated-v4_to_v5/ and runs migration.
// When yes is true, --yes is passed to tf-migrate to auto-confirm the phase-1
// completion prompt (used for the phase-2 call in the e2e runner).
func RunMigrate(resources string, yes bool) error {
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

	fmt.Println()

	// Run migration (config only — state is handled by the provider's UpgradeState/MoveState)
	printYellow("Migrating configuration files...")
	args := []string{
		"--config-dir", generatedDir,
		"--source-version", "v4",
		"--target-version", "v5",
		"migrate",
		"--backup=false",
		"--recursive",
	}
	if yes {
		args = append(args, "--skip-phase-check")
	}

	cmd := exec.Command(binary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		printError("Migration failed")
		return err
	}

	printSuccess("Migration complete (config transformed; state will be upgraded by the provider)")

	// Apply post-migration patches (e.g., adding required fields that tf-migrate can't auto-generate)
	// Looks for integration/v4_to_v5/testdata/<resource>/postmigrate/ directories
	if err := applyPostMigrationPatches(repoRoot, generatedDir, resourceList); err != nil {
		printYellow("Warning: Failed to apply post-migration patches: %v", err)
	}

	// Hoist import blocks from module subdirectories to the root main.tf.
	// Terraform only supports import blocks in the root module, so any import
	// blocks generated inside a module directory must be moved to root with the
	// module address prepended to the `to` attribute.
	if err := hoistImportBlocksToRoot(generatedDir, resourceList); err != nil {
		printYellow("Warning: Failed to hoist import blocks to root: %v", err)
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

	printSuccess("Copied tf/v4/ to migrated-v4_to_v5/")
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

// hoistImportBlocksToRoot extracts import blocks from module subdirectories and
// appends them to the root main.tf with the module address prepended to `to`.
//
// Terraform only supports import blocks in the root module. tf-migrate generates
// import blocks inside the resource module files (e.g. zone_setting/zone_setting.tf).
// This function moves them to root so they are valid.
//
// Example: an import block inside zone_setting/ with
//
//	to = cloudflare_zone_setting.minimal_always_online
//
// becomes at root:
//
//	to = module.zone_setting.cloudflare_zone_setting.minimal_always_online
func hoistImportBlocksToRoot(generatedDir string, resourceList []string) error {
	mainTFPath := filepath.Join(generatedDir, "main.tf")

	var hoisted []string

	for _, resource := range resourceList {
		resource = strings.TrimSpace(resource)
		moduleDir := filepath.Join(generatedDir, resource)
		if _, err := os.Stat(moduleDir); os.IsNotExist(err) {
			continue
		}

		tfFiles, err := filepath.Glob(filepath.Join(moduleDir, "*.tf"))
		if err != nil {
			continue
		}

		for _, tfFile := range tfFiles {
			content, err := os.ReadFile(tfFile)
			if err != nil {
				continue
			}

			cleaned, extracted := extractImportBlocks(string(content), resource)
			if len(extracted) == 0 {
				continue
			}

			if err := os.WriteFile(tfFile, []byte(cleaned), permFile); err != nil {
				return fmt.Errorf("failed to rewrite %s after extracting import blocks: %w", tfFile, err)
			}

			hoisted = append(hoisted, extracted...)
			printBlue("  ✓ Hoisted %d import block(s) from %s/%s to root",
				len(extracted), resource, filepath.Base(tfFile))
		}
	}

	if len(hoisted) == 0 {
		return nil
	}

	// Append all hoisted import blocks to root main.tf
	f, err := os.OpenFile(mainTFPath, os.O_APPEND|os.O_WRONLY, permFile)
	if err != nil {
		return fmt.Errorf("failed to open root main.tf for appending import blocks: %w", err)
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "\n# Import blocks hoisted from module subdirectories\n"); err != nil {
		return err
	}
	for _, block := range hoisted {
		if _, err := fmt.Fprintf(f, "%s\n", block); err != nil {
			return err
		}
	}

	printSuccess("Hoisted %d import block(s) to root main.tf", len(hoisted))
	return nil
}

// extractImportBlocks removes all import blocks from content and returns:
// - the content with import blocks removed
// - the import blocks rewritten with "module.<moduleName>." prepended to the `to` address
func extractImportBlocks(content, moduleName string) (cleaned string, blocks []string) {
	lines := strings.Split(content, "\n")
	var cleanedLines []string
	var currentBlock []string
	inImport := false
	braceDepth := 0

	for _, line := range lines {
		if !inImport && strings.TrimSpace(line) == "import {" {
			inImport = true
			braceDepth = 1
			currentBlock = []string{line}
			continue
		}

		if inImport {
			currentBlock = append(currentBlock, line)
			braceDepth += strings.Count(line, "{")
			braceDepth -= strings.Count(line, "}")

			if braceDepth == 0 {
				blockStr := strings.Join(currentBlock, "\n")

				// Skip hoisting if the id expression references module-local values
				// (e.g. local.X) that are not accessible at root level. Leave the
				// block in the module file with a comment explaining why it was skipped.
				if strings.Contains(blockStr, "local.") {
					cleanedLines = append(cleanedLines, "# NOTE: import block not hoisted to root — id references a module-local value.")
					cleanedLines = append(cleanedLines, "# Add this import block manually in your root module with the resolved id value:")
					for _, bl := range currentBlock {
						cleanedLines = append(cleanedLines, "# "+bl)
					}
					currentBlock = nil
					inImport = false
					braceDepth = 0
					continue
				}

				// Rewrite the `to` attribute to include the module prefix
				var rewritten []string
				for _, blockLine := range currentBlock {
					trimmed := strings.TrimSpace(blockLine)
					if strings.HasPrefix(trimmed, "to = ") {
						addr := strings.TrimPrefix(trimmed, "to = ")
						indent := blockLine[:len(blockLine)-len(strings.TrimLeft(blockLine, " \t"))]
						blockLine = indent + "to = module." + moduleName + "." + addr
					}
					rewritten = append(rewritten, blockLine)
				}
				blocks = append(blocks, strings.Join(rewritten, "\n"))
				currentBlock = nil
				inImport = false
				braceDepth = 0
			}
			continue
		}

		cleanedLines = append(cleanedLines, line)
	}

	// Trim trailing blank lines left by removed import blocks
	for len(cleanedLines) > 0 && strings.TrimSpace(cleanedLines[len(cleanedLines)-1]) == "" {
		cleanedLines = cleanedLines[:len(cleanedLines)-1]
	}

	return strings.Join(cleanedLines, "\n") + "\n", blocks
}

// completeBYOIPPrefixMigration adds asn and cidr fields to migrated byo_ip_prefix resources.
// This simulates the manual intervention step users must perform after migration.
// It replaces MIGRATION WARNING comments with actual field values from environment variables.
func completeBYOIPPrefixMigration(migratedDir string, env *E2EEnv) error {
	// Only run if env vars are set
	if env.BYOIPASN == "" || env.BYOIPCidr == "" {
		return nil // Skip if not configured
	}

	// Find all .tf files in migrated directory
	files, err := filepath.Glob(filepath.Join(migratedDir, "**", "*.tf"))
	if err != nil {
		return fmt.Errorf("finding tf files: %w", err)
	}

	// Also check root level
	rootFiles, err := filepath.Glob(filepath.Join(migratedDir, "*.tf"))
	if err != nil {
		return fmt.Errorf("finding root tf files: %w", err)
	}
	files = append(files, rootFiles...)

	warningPattern := "# MIGRATION WARNING: This resource requires manual intervention to add v5 required fields 'asn' and 'cidr'"
	modified := 0

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("reading %s: %w", file, err)
		}

		// Check if file contains byo_ip_prefix with migration warning
		if !strings.Contains(string(content), "cloudflare_byo_ip_prefix") {
			continue
		}
		if !strings.Contains(string(content), warningPattern) {
			continue
		}

		// Replace warning comment with actual fields
		replacement := fmt.Sprintf("asn  = %s\n  cidr = \"%s\"", env.BYOIPASN, env.BYOIPCidr)
		newContent := strings.ReplaceAll(string(content), warningPattern, replacement)

		if err := os.WriteFile(file, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", file, err)
		}
		modified++
	}

	if modified > 0 {
		printSuccess("Completed byo_ip_prefix migration (%d file(s) updated)", modified)
	}

	return nil
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
