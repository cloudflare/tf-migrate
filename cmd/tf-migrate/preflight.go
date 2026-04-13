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
	files, err := findTerraformFilesWithRecursion(cfg.configDir, cfg.recursive, cfg.exclude)
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

	// Auto-fix moved blocks that use v4 type names before detecting conflicts
	autoFixMovedBlocks(report, renames, files, log, cfg.verbose)

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

	// cloudflare_access_policy resources with application_id are a special manual
	// path in v4->v5: they become removed {} blocks and must be rewritten inline
	// on cloudflare_zero_trust_access_application.policies.
	if cfg.sourceVersion == "v4" && cfg.targetVersion == "v5" &&
		resourceType == "cloudflare_access_policy" && block.Body().GetAttribute("application_id") != nil {
		return &scannedResource{
			File:         file,
			ResourceType: resourceType,
			ResourceName: resourceName,
			Class:        classManualIntervention,
			Detail:       "application_id detected -- tf-migrate will generate removed {}; migrate policy inline to cloudflare_zero_trust_access_application.policies (do not use moved block)",
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

// autoFixMovedBlocks updates moved blocks that use v4 resource type names to use v5 type names.
// This is done automatically so users don't need to manually update their moved blocks.
// Only fixes moved blocks where BOTH from and to are v4 types (conflicting case).
// Moved blocks that already correctly map v4->v5 are left untouched.
func autoFixMovedBlocks(report *preflightReport, renames map[string]string, files []string, log hclog.Logger, verbose bool) {
	// Track which files need updating and what changes were made
	type fileUpdate struct {
		path        string
		oldBlocks   []existingMovedBlock
		newBlocks   []existingMovedBlock
		needsUpdate bool
	}

	// Build file map using basenames (report.MovedBlocks use basename, files use full path)
	fileMap := make(map[string]*fileUpdate)
	for _, f := range files {
		basename := filepath.Base(f)
		fileMap[basename] = &fileUpdate{path: f}
	}

	// Find moved blocks that need updating
	for i := range report.MovedBlocks {
		mb := &report.MovedBlocks[i]
		fu := fileMap[mb.File]
		if fu == nil {
			continue
		}

		oldFromType := mb.FromType
		oldToType := mb.ToType
		needsUpdate := false

		// Check if from-type is a v4 name that gets renamed
		newFromType, fromIsV4 := renames[mb.FromType]
		// Check if to-type is a v4 name that gets renamed
		newToType, toIsV4 := renames[mb.ToType]

		// Only auto-fix if BOTH from and to are v4 types (the problematic case)
		// If from is v4 and to is already v5, that's a correct mapping - don't touch it
		if fromIsV4 && toIsV4 {
			if newFromType != mb.FromType {
				mb.FromType = newFromType
				needsUpdate = true
			}
			if newToType != mb.ToType {
				mb.ToType = newToType
				needsUpdate = true
			}
		}

		if needsUpdate {
			fu.needsUpdate = true
			fu.oldBlocks = append(fu.oldBlocks, existingMovedBlock{
				File:     mb.File,
				FromType: oldFromType,
				FromName: mb.FromName,
				ToType:   oldToType,
				ToName:   mb.ToName,
			})
			fu.newBlocks = append(fu.newBlocks, *mb)
		}
	}

	// Apply updates to files
	for _, fu := range fileMap {
		if !fu.needsUpdate {
			continue
		}

		content, err := os.ReadFile(fu.path)
		if err != nil {
			log.Warn("Failed to read file for moved block auto-fix", "file", fu.path, "error", err)
			continue
		}

		parsed, diags := hclwrite.ParseConfig(content, filepath.Base(fu.path), hcl.InitialPos)
		if diags.HasErrors() {
			log.Warn("Failed to parse file for moved block auto-fix", "file", fu.path, "error", diags)
			continue
		}

		// Update moved blocks in the parsed HCL
		for _, block := range parsed.Body().Blocks() {
			if block.Type() != "moved" {
				continue
			}

			body := block.Body()
			fromAttr := body.GetAttribute("from")
			toAttr := body.GetAttribute("to")
			if fromAttr == nil || toAttr == nil {
				continue
			}

			fromStr := extractTraversalString(fromAttr)
			toStr := extractTraversalString(toAttr)
			if fromStr == "" || toStr == "" {
				continue
			}

			fromParts := strings.SplitN(fromStr, ".", 2)
			toParts := strings.SplitN(toStr, ".", 2)
			if len(fromParts) != 2 || len(toParts) != 2 {
				continue
			}

			// Check if this moved block needs updating
			fromType := fromParts[0]
			toType := toParts[0]
			fromName := fromParts[1]
			toName := toParts[1]

			newFromType, fromNeedsUpdate := renames[fromType]
			newToType, toNeedsUpdate := renames[toType]

			if fromNeedsUpdate {
				// Rebuild the from traversal with new type
				fromTraversal := hcl.Traversal{
					hcl.TraverseRoot{Name: newFromType},
					hcl.TraverseAttr{Name: fromName},
				}
				body.SetAttributeTraversal("from", fromTraversal)
			}

			if toNeedsUpdate {
				// Rebuild the to traversal with new type
				toTraversal := hcl.Traversal{
					hcl.TraverseRoot{Name: newToType},
					hcl.TraverseAttr{Name: toName},
				}
				body.SetAttributeTraversal("to", toTraversal)
			}
		}

		// Write the updated file
		if err := os.WriteFile(fu.path, parsed.Bytes(), 0644); err != nil {
			log.Warn("Failed to write file for moved block auto-fix", "file", fu.path, "error", err)
			continue
		}

		// Log the changes
		if verbose {
			for i, old := range fu.oldBlocks {
				new := fu.newBlocks[i]
				log.Info("Auto-fixed moved block",
					"file", filepath.Base(fu.path),
					"old_from", old.FromType+"."+old.FromName,
					"old_to", old.ToType+"."+old.ToName,
					"new_from", new.FromType+"."+new.FromName,
					"new_to", new.ToType+"."+new.ToName)
			}
		}
	}
}

// detectMovedBlockConflicts checks for hand-written moved blocks that may conflict
// with what tf-migrate would generate.
func detectMovedBlockConflicts(report *preflightReport, renames map[string]string) []string {
	var warnings []string

	manualAddrs := make(map[string]struct{})
	for _, r := range report.Resources {
		if r.Class == classManualIntervention {
			manualAddrs[r.ResourceType+"."+r.ResourceName] = struct{}{}
		}
	}

	for _, mb := range report.MovedBlocks {
		addr := mb.FromType + "." + mb.FromName

		if _, isManual := manualAddrs[addr]; isManual {
			warnings = append(warnings, fmt.Sprintf(
				"Conflicting moved block in %s: %s → %s.%s\n"+
					"  This resource requires manual migration (application_id path).\n"+
					"  Do not use a moved block; keep the generated removed {} block and migrate inline policies manually.",
				mb.File, addr, mb.ToType, mb.ToName))
			continue
		}

		// Check if the from-type is a v4 name that tf-migrate would rename
		if newType, ok := renames[mb.FromType]; ok {
			if mb.ToType == newType {
				// User's moved block matches what tf-migrate would generate -- duplicate
				warnings = append(warnings, fmt.Sprintf(
					"Pre-existing moved block in %s: %s → %s.%s\n"+
						"  tf-migrate will generate this moved block automatically.\n"+
						"  Consider removing it to avoid duplicates.",
					mb.File, addr, mb.ToType, mb.ToName))
			}
			// Note: We no longer warn about "Conflicting moved block" with wrong target type
			// because autoFixMovedBlocks already updates those to the correct v5 types.
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

	// Print warnings ALWAYS (these are important for all users)
	if len(report.Warnings) > 0 {
		fmt.Println()
		fmt.Println("⚠ Warnings:")
		for _, w := range report.Warnings {
			for _, line := range strings.Split(w, "\n") {
				fmt.Printf("  %s\n", line)
			}
		}
	}

	// Only show detailed scan output in verbose mode
	if cfg.verbose {
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
		}

		if len(autoMigrated) > 0 {
			fmt.Printf("  Resources with config-only changes (%d):\n", len(autoMigrated))
			for _, r := range autoMigrated {
				fmt.Printf("    %s.%s\n", r.ResourceType, r.ResourceName)
			}
			fmt.Println()
		}

		if len(manual) > 0 {
			fmt.Printf("  Resources requiring manual intervention (%d):\n", len(manual))
			for _, r := range manual {
				fmt.Printf("    %s.%s: %s\n", r.ResourceType, r.ResourceName, r.Detail)
			}
			fmt.Println()
		}

		if len(unsupported) > 0 {
			fmt.Printf("  Unsupported resources -- no transformer registered (%d):\n", len(unsupported))
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
	} else {
		// Minimal output in non-verbose mode
		fmt.Printf("  Migrated %d resources (%d renamed, %d config-only)\n",
			len(report.Resources), len(renamed), len(autoMigrated))
	}
}
