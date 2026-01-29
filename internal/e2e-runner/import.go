// import.go handles parsing import annotations and generating import blocks.
//
// This file provides functionality to:
//   - Parse import annotations from Terraform configuration files
//   - Generate import blocks in the root main.tf for resources that cannot be created
//   - Support variable interpolation in import addresses
//
// Import annotations use the format:
//   # tf-migrate:import-address=<address>
//   resource "type" "name" { ... }
//
// Where <address> can include variable references like ${var.cloudflare_account_id}
//
// The generated import blocks are placed in the root module's main.tf:
//   import {
//     to = module.<module_name>.<resource_type>.<resource_name>
//     id = <resolved_address>
//   }
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

// findImportSpecs scans specific module directories for import annotations and returns import specifications
func findImportSpecs(dir string, moduleNames []string) ([]ImportSpec, error) {
	var specs []ImportSpec

	// If no module names specified, scan all modules
	if len(moduleNames) == 0 {
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

	// Scan only specified module directories
	for _, moduleName := range moduleNames {
		moduleDir := filepath.Join(dir, moduleName)

		// Check if module directory exists
		if _, err := os.Stat(moduleDir); os.IsNotExist(err) {
			continue // Skip if module doesn't exist
		}

		// Find all .tf files in this module directory
		tfFiles, err := filepath.Glob(filepath.Join(moduleDir, "*.tf"))
		if err != nil {
			return nil, fmt.Errorf("failed to glob tf files in %s: %w", moduleDir, err)
		}

		// Parse each .tf file for import annotations
		for _, tfFile := range tfFiles {
			fileSpecs, err := parseImportAnnotations(tfFile, moduleName)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s: %w", tfFile, err)
			}
			specs = append(specs, fileSpecs...)
		}
	}

	return specs, nil
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

// convertToTerraformVar converts ${var.X} syntax to var.X for use in import blocks
func convertToTerraformVar(address string) string {
	// Convert ${var.cloudflare_account_id} to var.cloudflare_account_id
	address = strings.ReplaceAll(address, "${var.cloudflare_account_id}", "var.cloudflare_account_id")
	address = strings.ReplaceAll(address, "${var.cloudflare_zone_id}", "var.cloudflare_zone_id")
	address = strings.ReplaceAll(address, "${var.cloudflare_domain}", "var.cloudflare_domain")

	return address
}

// generateImportBlocks generates import block declarations for the root main.tf
func generateImportBlocks(specs []ImportSpec) string {
	if len(specs) == 0 {
		return ""
	}

	var blocks strings.Builder
	blocks.WriteString("\n# Import blocks for resources that cannot be created via Terraform\n")
	blocks.WriteString("# These resources must be imported from existing infrastructure\n\n")

	for _, spec := range specs {
		// Convert variable syntax for Terraform (${var.X} -> var.X)
		importID := convertToTerraformVar(spec.ImportAddress)

		// Build full resource address including module prefix
		fullResourceAddress := fmt.Sprintf("module.%s.%s", spec.ModuleName, spec.ResourceAddress)

		blocks.WriteString(fmt.Sprintf("import {\n"))
		blocks.WriteString(fmt.Sprintf("  to = %s\n", fullResourceAddress))

		// Check if importID looks like a variable reference (starts with var.)
		if strings.HasPrefix(importID, "var.") || strings.Contains(importID, "var.") {
			// Don't quote variable references
			blocks.WriteString(fmt.Sprintf("  id = %s\n", importID))
		} else {
			// Quote literal strings
			blocks.WriteString(fmt.Sprintf("  id = %q\n", importID))
		}
		blocks.WriteString(fmt.Sprintf("}\n\n"))
	}

	return blocks.String()
}
