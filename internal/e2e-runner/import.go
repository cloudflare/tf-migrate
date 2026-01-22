// import.go handles Terraform import operations for import-only resources.
//
// This file provides functionality to:
//   - Parse import annotations from Terraform configuration files
//   - Execute terraform import commands for resources that cannot be created
//   - Support variable interpolation in import addresses
//
// Import annotations use the format:
//   # tf-migrate:import-address=<address>
//   resource "type" "name" { ... }
//
// Where <address> can include variable references like ${var.cloudflare_account_id}
package e2e

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ImportSpec represents a resource that needs to be imported
type ImportSpec struct {
	ResourceType    string // e.g., "cloudflare_access_organization"
	ResourceName    string // e.g., "test"
	ResourceAddress string // e.g., "cloudflare_access_organization.test"
	ImportAddress   string // e.g., "account/${var.cloudflare_account_id}"
	ModuleName      string // e.g., "zero_trust_organization" (extracted from file path)
}

// findImportSpecs scans a directory for import annotations and returns import specifications
func findImportSpecs(dir string) ([]ImportSpec, error) {
	var specs []ImportSpec

	// Walk through all .tf files in subdirectories (modules)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a .tf file
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".tf") {
			return nil
		}

		// Skip root directory files (only look in modules)
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if !strings.Contains(relPath, string(filepath.Separator)) {
			return nil // Skip files in root directory
		}

		// Extract module name from path (first directory component)
		moduleName := strings.Split(relPath, string(filepath.Separator))[0]

		// Parse file for import annotations
		fileSpecs, err := parseImportAnnotations(path, moduleName)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}

		specs = append(specs, fileSpecs...)
		return nil
	})

	return specs, err
}

// parseImportAnnotations parses a single .tf file for import annotations
func parseImportAnnotations(filePath, moduleName string) ([]ImportSpec, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var specs []ImportSpec
	scanner := bufio.NewScanner(file)

	// Regex patterns
	importAnnotation := regexp.MustCompile(`^\s*#\s*tf-migrate:import-address=(.+)$`)
	resourceDeclaration := regexp.MustCompile(`^\s*resource\s+"([^"]+)"\s+"([^"]+)"\s*{`)

	var pendingImportAddress string

	for scanner.Scan() {
		line := scanner.Text()

		// Check for import annotation
		if matches := importAnnotation.FindStringSubmatch(line); matches != nil {
			pendingImportAddress = strings.TrimSpace(matches[1])
			continue
		}

		// Check for resource declaration following an import annotation
		if pendingImportAddress != "" {
			if matches := resourceDeclaration.FindStringSubmatch(line); matches != nil {
				resourceType := matches[1]
				resourceName := matches[2]

				specs = append(specs, ImportSpec{
					ResourceType:    resourceType,
					ResourceName:    resourceName,
					ResourceAddress: fmt.Sprintf("%s.%s", resourceType, resourceName),
					ImportAddress:   pendingImportAddress,
					ModuleName:      moduleName,
				})

				pendingImportAddress = "" // Reset
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return specs, nil
}

// resolveImportAddress replaces variable references in import address with actual values
func resolveImportAddress(address string, env *E2EEnv) string {
	// Replace common variable references
	address = strings.ReplaceAll(address, "${var.cloudflare_account_id}", env.AccountID)
	address = strings.ReplaceAll(address, "${var.cloudflare_zone_id}", env.ZoneID)
	address = strings.ReplaceAll(address, "${var.cloudflare_domain}", env.Domain)

	return address
}

// executeImports runs terraform import commands for all import specs
func executeImports(ctx *testContext, specs []ImportSpec) error {
	if len(specs) == 0 {
		return nil // No imports needed
	}

	printHeader("Importing Resources")
	printYellow("Found %d resource(s) marked for import", len(specs))
	fmt.Println()

	tf := NewTerraformRunner(ctx.v4Dir)

	// Set R2 credentials for terraform commands
	r2AccessKey := os.Getenv("CLOUDFLARE_R2_ACCESS_KEY_ID")
	r2SecretKey := os.Getenv("CLOUDFLARE_R2_SECRET_ACCESS_KEY")
	if r2AccessKey != "" && r2SecretKey != "" {
		tf.EnvVars["AWS_ACCESS_KEY_ID"] = r2AccessKey
		tf.EnvVars["AWS_SECRET_ACCESS_KEY"] = r2SecretKey
	}

	for _, spec := range specs {
		// Resolve variables in import address
		importAddress := resolveImportAddress(spec.ImportAddress, ctx.env)

		// Build full resource address including module prefix
		fullResourceAddress := fmt.Sprintf("module.%s.%s", spec.ModuleName, spec.ResourceAddress)

		printYellow("Importing %s...", fullResourceAddress)
		printBlue("  Import address: %s", importAddress)

		// Run terraform import
		output, err := tf.Run("import", "-no-color", "-input=false", fullResourceAddress, importAddress)
		if err != nil {
			// Check if resource already exists in state
			if strings.Contains(output, "Resource already managed by Terraform") ||
				strings.Contains(output, "already exists in state") {
				printGreen("  ✓ Resource already imported")
				continue
			}

			printError("Failed to import %s", fullResourceAddress)
			fmt.Println()
			printRed("Error output:")
			fmt.Println(output)
			return fmt.Errorf("import failed for %s: %w", fullResourceAddress, err)
		}

		printSuccess("Successfully imported %s", fullResourceAddress)
		fmt.Println()
	}

	printSuccess("All imports completed")
	fmt.Println()

	return nil
}
