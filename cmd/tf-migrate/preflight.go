package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

// resourceClassification describes how a resource will be handled during migration.
type resourceClassification int

const (
	// classAutoMigrated means the resource has a transformer and will be fully auto-migrated.
	classAutoMigrated resourceClassification = iota
	// classRenamed means the resource will be renamed and a moved block generated.
	classRenamed
	// classManualIntervention means the resource has a transformer but requires manual steps.
	classManualIntervention
	// classUnsupported means no transformer is registered for this resource type.
	classUnsupported
)

// scannedResource holds information about a resource found during pre-migration scanning.
type scannedResource struct {
	File         string
	ResourceType string
	ResourceName string
	Class        resourceClassification
	OldType      string // only set for classRenamed
	NewType      string // only set for classRenamed
	Detail       string // human-readable detail for manual intervention / unsupported
}

// existingMovedBlock represents a hand-written moved block found in the user's config.
type existingMovedBlock struct {
	File     string
	FromType string
	FromName string
	ToType   string
	ToName   string
}

// preflightReport is the result of a pre-migration scan.
type preflightReport struct {
	Resources   []scannedResource
	MovedBlocks []existingMovedBlock
	Warnings    []string
}

// runPreMigrationScan scans all .tf files and classifies resources and detects
// existing moved blocks. It prints a summary and returns warnings for any issues.
func runPreMigrationScan(log hclog.Logger, cfg config) (*preflightReport, error) {
	files, err := findTerraformFilesWithRecursion(cfg.configDir, cfg.recursive)
	if err != nil {
		return nil, fmt.Errorf("failed to find terraform files: %w", err)
	}

	providers := getProviders(cfg.resourcesToMigrate...)
	report := &preflightReport{}

	// Build rename map from all migrators (same logic as applyGlobalPostprocessing)
	migrators := providers.GetAllMigrators(cfg.sourceVersion, cfg.targetVersion, cfg.resourcesToMigrate...)
	renames := make(map[string]string) // old type -> new type
	for _, migrator := range migrators {
		if renamer, ok := migrator.(transform.ResourceRenamer); ok {
			oldTypes, newType := renamer.GetResourceRename()
			for _, oldType := range oldTypes {
				if oldType != newType {
					renames[oldType] = newType
				}
			}
		}
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			log.Warn("Failed to read file during pre-migration scan", "file", file, "error", err)
			continue
		}

		parsed, diags := hclwrite.ParseConfig(content, filepath.Base(file), hcl.InitialPos)
		if diags.HasErrors() {
			log.Warn("Failed to parse file during pre-migration scan", "file", file, "error", diags)
			continue
		}

		relFile := filepath.Base(file)

		for _, block := range parsed.Body().Blocks() {
			switch block.Type() {
			case "resource":
				if len(block.Labels()) < 2 {
					continue
				}
				sr := classifyResource(block, relFile, providers, renames, cfg)
				if sr != nil {
					report.Resources = append(report.Resources, *sr)
				}

			case "moved":
				mb := parseMovedBlock(block, relFile)
				if mb != nil {
					report.MovedBlocks = append(report.MovedBlocks, *mb)
				}
			}
		}
	}

	// Check for conflicting moved blocks
	report.Warnings = detectMovedBlockConflicts(report, renames)

	return report, nil
}

// classifyResource determines how a resource will be handled during migration.
func classifyResource(block *hclwrite.Block, file string, providers transform.MigrationProvider, renames map[string]string, cfg config) *scannedResource {
	resourceType := block.Labels()[0]
	resourceName := block.Labels()[1]

	// Only scan cloudflare resources
	if !strings.HasPrefix(resourceType, "cloudflare_") {
		return nil
	}

	migrator := providers.GetMigrator(resourceType, cfg.sourceVersion, cfg.targetVersion)
	if migrator == nil {
		return &scannedResource{
			File:         file,
			ResourceType: resourceType,
			ResourceName: resourceName,
			Class:        classUnsupported,
			Detail:       "no transformer registered -- manual migration required",
		}
	}

	// Check if this resource will be renamed
	if newType, ok := renames[resourceType]; ok {
		return &scannedResource{
			File:         file,
			ResourceType: resourceType,
			ResourceName: resourceName,
			Class:        classRenamed,
			OldType:      resourceType,
			NewType:      newType,
		}
	}

	// Resource has a transformer but no rename -- config-only changes
	return &scannedResource{
		File:         file,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Class:        classAutoMigrated,
	}
}

// parseMovedBlock extracts from/to information from a moved block.
func parseMovedBlock(block *hclwrite.Block, file string) *existingMovedBlock {
	body := block.Body()

	fromAttr := body.GetAttribute("from")
	toAttr := body.GetAttribute("to")
	if fromAttr == nil || toAttr == nil {
		return nil
	}

	fromStr := extractTraversalString(fromAttr)
	toStr := extractTraversalString(toAttr)
	if fromStr == "" || toStr == "" {
		return nil
	}

	fromParts := strings.SplitN(fromStr, ".", 2)
	toParts := strings.SplitN(toStr, ".", 2)

	// Only care about cloudflare resources
	if !strings.HasPrefix(fromStr, "cloudflare_") && !strings.HasPrefix(toStr, "cloudflare_") {
		return nil
	}

	mb := &existingMovedBlock{
		File: file,
	}
	if len(fromParts) == 2 {
		mb.FromType = fromParts[0]
		mb.FromName = fromParts[1]
	}
	if len(toParts) == 2 {
		mb.ToType = toParts[0]
		mb.ToName = toParts[1]
	}

	return mb
}

// extractTraversalString extracts the string representation of a traversal attribute
// (e.g., "cloudflare_access_application.admin_api" from `from = cloudflare_access_application.admin_api`).
func extractTraversalString(attr *hclwrite.Attribute) string {
	tokens := attr.Expr().BuildTokens(nil)
	var parts []string
	for _, tok := range tokens {
		s := strings.TrimSpace(string(tok.Bytes))
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "")
}

// detectMovedBlockConflicts checks for hand-written moved blocks that may conflict
// with what tf-migrate would generate.
func detectMovedBlockConflicts(report *preflightReport, renames map[string]string) []string {
	var warnings []string

	for _, mb := range report.MovedBlocks {
		addr := mb.FromType + "." + mb.FromName

		// Check if the from-type is a v4 name that tf-migrate would rename
		if newType, ok := renames[mb.FromType]; ok {
			if mb.ToType == newType {
				// User's moved block matches what tf-migrate would generate -- duplicate
				warnings = append(warnings, fmt.Sprintf(
					"Pre-existing moved block in %s: %s → %s.%s\n"+
						"  tf-migrate will generate this moved block automatically.\n"+
						"  Consider removing it to avoid duplicates.",
					mb.File, addr, mb.ToType, mb.ToName))
			} else {
				// User's moved block targets a different type than expected
				warnings = append(warnings, fmt.Sprintf(
					"Conflicting moved block in %s: %s → %s.%s\n"+
						"  tf-migrate would rename %s to %s, not %s.\n"+
						"  Please remove or correct this moved block before proceeding.",
					mb.File, addr, mb.ToType, mb.ToName,
					mb.FromType, newType, mb.ToType))
			}
		}

		// Check if from-type doesn't exist in v5 provider at all and has no transformer
		if _, ok := renames[mb.FromType]; !ok {
			// Not a known rename -- check if it's even a cloudflare type
			if strings.HasPrefix(mb.FromType, "cloudflare_") {
				// Check if this type has any migrator registered
				found := false
				for _, r := range report.Resources {
					if r.ResourceType == mb.FromType {
						found = true
						break
					}
				}
				if !found {
					// The moved block references a type that has no resources in config
					// This could be a stale moved block or a cross-module reference
					warnings = append(warnings, fmt.Sprintf(
						"Moved block in %s references %s which has no matching resource in config.\n"+
							"  Verify this moved block is still needed.",
						mb.File, addr))
				}
			}
		}
	}

	return warnings
}

// printPreflightReport prints the pre-migration scan results to stdout.
func printPreflightReport(report *preflightReport, cfg config) {
	if len(report.Resources) == 0 && len(report.MovedBlocks) == 0 {
		return
	}

	var renamed, autoMigrated, manual, unsupported []scannedResource
	for _, r := range report.Resources {
		switch r.Class {
		case classRenamed:
			renamed = append(renamed, r)
		case classAutoMigrated:
			autoMigrated = append(autoMigrated, r)
		case classManualIntervention:
			manual = append(manual, r)
		case classUnsupported:
			unsupported = append(unsupported, r)
		}
	}

	fmt.Println()
	fmt.Println("Pre-migration scan:")
	fmt.Printf("  %d Cloudflare resource(s) found\n", len(report.Resources))
	fmt.Println()

	if len(renamed) > 0 {
		fmt.Printf("  Resources with type rename + moved block (%d):\n", len(renamed))
		for _, r := range renamed {
			fmt.Printf("    %s.%s → %s.%s\n", r.OldType, r.ResourceName, r.NewType, r.ResourceName)
		}
		fmt.Println()
		fmt.Println("  IMPORTANT: Moved blocks require the v5 provider to support MoveState for")
		fmt.Println("  each resource. Ensure you use the provider version set by tf-migrate")
		fmt.Println("  (including beta releases if applicable).")
		fmt.Println()
	}

	if len(autoMigrated) > 0 && cfg.verbose {
		fmt.Printf("  Resources with config-only changes (%d):\n", len(autoMigrated))
		for _, r := range autoMigrated {
			fmt.Printf("    %s.%s\n", r.ResourceType, r.ResourceName)
		}
		fmt.Println()
	} else if len(autoMigrated) > 0 {
		fmt.Printf("  %d resource(s) with config-only changes (no rename)\n", len(autoMigrated))
	}

	if len(manual) > 0 {
		fmt.Printf("  ⚠ Resources requiring manual intervention (%d):\n", len(manual))
		for _, r := range manual {
			fmt.Printf("    %s.%s: %s\n", r.ResourceType, r.ResourceName, r.Detail)
		}
		fmt.Println()
	}

	if len(unsupported) > 0 {
		fmt.Printf("  ⚠ Unsupported resources -- no transformer registered (%d):\n", len(unsupported))
		for _, r := range unsupported {
			fmt.Printf("    %s.%s\n", r.ResourceType, r.ResourceName)
		}
		fmt.Println("  These resources must be migrated manually. Check the v5 migration guide:")
		fmt.Println("  https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs/guides/version-5-upgrade")
		fmt.Println()
	}

	if len(report.MovedBlocks) > 0 {
		fmt.Printf("  Existing moved blocks found (%d):\n", len(report.MovedBlocks))
		for _, mb := range report.MovedBlocks {
			fmt.Printf("    %s: %s.%s → %s.%s\n", mb.File, mb.FromType, mb.FromName, mb.ToType, mb.ToName)
		}
		fmt.Println()
	}

	// Print warnings
	if len(report.Warnings) > 0 {
		fmt.Println("  ⚠ Warnings:")
		for _, w := range report.Warnings {
			for _, line := range strings.Split(w, "\n") {
				fmt.Printf("    %s\n", line)
			}
			fmt.Println()
		}
	}
}
