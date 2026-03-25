package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/spf13/cobra"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/logger"
	"github.com/cloudflare/tf-migrate/internal/pipeline"
	"github.com/cloudflare/tf-migrate/internal/registry"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/cloudflare/tf-migrate/internal/verifydrift"
)

// version is set at build time via ldflags.
var version = "dev"

type config struct {
	// Input paths
	configDir string

	// Output paths
	outputDir string

	// Migration options
	resourcesToMigrate []string
	sourceVersion      string
	targetVersion      string
	dryRun             bool
	backup             bool
	recursive          bool
	logLevel           string
	yes                bool // skip interactive prompts, assume yes (for CI/e2e)

	// Diagnostic output options
	quiet   bool // Suppress warnings, only show errors
	verbose bool // Show all diagnostics including informational messages
}

var (
	rootCmd = &cobra.Command{
		Use:   "tf-migrate",
		Short: "Terraform configuration migration tool",
		Long: `tf-migrate is a CLI tool for migrating Terraform configurations between
different provider versions or resource schemas.

This tool provides automated transformations for:
- Resource type changes
- Attribute migrations
- Import generation for new resources
- Moved blocks for resource renames`,
		Example: `  # Migrate all .tf files in current directory
  tf-migrate migrate

  # Migrate specific directory
  tf-migrate --config-dir ./terraform migrate

  # Migrate only specific resources
  tf-migrate --resources dns_record,load_balancer migrate

  # Dry run to preview changes
  tf-migrate --dry-run migrate

  # Run with debug logging
  tf-migrate --log-level debug migrate`,
	}
)

var validVersionPath = map[string]struct{}{
	"v4-v5": {},
	"v5-v5": {}, // Allow same-version "migrations" (bypass mode - generates moved blocks only)
}

func main() {
	// Register all resource migrations
	registry.RegisterAllMigrations()

	cfg := &config{}
	rootCmd.PersistentFlags().StringVar(&cfg.configDir, "config-dir", "", "Directory containing Terraform configuration files")
	rootCmd.PersistentFlags().StringSliceVar(&cfg.resourcesToMigrate, "resources", []string{}, "Comma-separated list of resources to migrate (empty = all)")
	rootCmd.PersistentFlags().BoolVar(&cfg.dryRun, "dry-run", false, "Perform a dry run without making changes")
	rootCmd.PersistentFlags().StringVar(&cfg.sourceVersion, "source-version", "", "Source provider version (e.g., v4, v5)")
	rootCmd.PersistentFlags().StringVar(&cfg.targetVersion, "target-version", "", "Target provider version (e.g., v5, v6)")

	rootCmd.PersistentFlags().StringVarP(&cfg.logLevel, "log-level", "l", "warn", "Set log level (debug, info, warn, error, off)")

	// Create logger instance
	log := logger.New(cfg.logLevel)
	rootCmd.AddCommand(newMigrateCommand(log, cfg))
	rootCmd.AddCommand(newVersionCommand())
	rootCmd.AddCommand(newVerifyDriftCommand())
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newMigrateCommand(log hclog.Logger, cfg *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run the migration on configuration files",
		Long: `Migrate Terraform configuration files using registered transformers.
Uses the global flags --config-dir and --resources to determine what to migrate.`,
		Example: `  # Migrate configuration files in current directory
  tf-migrate migrate

  # Migrate specific directory
  tf-migrate --config-dir ./terraform migrate

  # Migrate with output to different directory
  tf-migrate migrate --output-dir ./migrated

  # Migrate only specific resources
  tf-migrate --resources dns_record,load_balancer migrate

  # Dry run to preview changes
  tf-migrate --dry-run migrate

  # Run with debug logging
  tf-migrate --log-level debug migrate`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.configDir == "" {
				cfg.configDir = "."
			}
			if cfg.sourceVersion == "" {
				cfg.sourceVersion = "v4"
			}
			if cfg.targetVersion == "" {
				cfg.targetVersion = "v5"
			}

			fmt.Println("Cloudflare Terraform Provider Migration Tool")
			fmt.Println("============================================")
			fmt.Println()

			fmt.Printf("Configuration directory: %s\n", cfg.configDir)
			if cfg.outputDir != "" {
				fmt.Printf("Output directory: %s\n", cfg.outputDir)
			} else {
				fmt.Println("Output directory: in-place")
			}

			if cfg.dryRun {
				fmt.Println("\n DRY RUN MODE - No changes will be made")
			}

			return runMigration(log, *cfg)
		},
	}

	cmd.Flags().StringVar(&cfg.outputDir, "output-dir", "", "Output directory for migrated configuration files (default: in-place)")
	cmd.Flags().BoolVar(&cfg.backup, "backup", true, "Create backup of original files before migration")
	cmd.Flags().BoolVar(&cfg.recursive, "recursive", false, "Recursively process subdirectories (useful for module structures)")

	// --no-backup is a convenience alias for --backup=false
	var noBackup bool
	cmd.Flags().BoolVar(&noBackup, "no-backup", false, "Skip creating backup files before migration (alias for --backup=false)")
	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		if noBackup {
			cfg.backup = false
		}
	}
	cmd.Flags().BoolVarP(&cfg.yes, "yes", "y", false, "Skip interactive prompts and assume yes (for CI/non-interactive use)")

	// Diagnostic output options
	cmd.Flags().BoolVarP(&cfg.quiet, "quiet", "q", false, "Suppress warnings, only show errors")
	cmd.Flags().BoolVar(&cfg.verbose, "verbose", false, "Show all diagnostics including informational messages")

	return cmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("tf-migrate version %s\n", version)
		},
	}
}

func newVerifyDriftCommand() *cobra.Command {
	var planFile string
	cmd := &cobra.Command{
		Use:   "verify-drift",
		Short: "Verify a terraform plan output against known migration drift exemptions",
		Long: `Reads a terraform plan output file and checks each change against Cloudflare's
known migration drift exemptions. Prints a report of expected vs unexpected changes.

Exit code 0: all drift is expected or none detected.
Exit code 1: unexpected drift requires attention.`,
		Example: `  # Export plan output and verify
  terraform plan > plan.txt
  tf-migrate verify-drift --file plan.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := os.ReadFile(planFile)
			if err != nil {
				return fmt.Errorf("reading plan file: %w", err)
			}
			result, err := verifydrift.Verify(string(content))
			if err != nil {
				return fmt.Errorf("verifying drift: %w", err)
			}
			verifydrift.PrintReport(result, planFile)
			if result.HasUnexpected {
				os.Exit(1)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&planFile, "file", "", "Path to terraform plan output file (required)")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

// runMigration performs the actual migration using the pipeline.
// It automatically detects whether a phased migration is needed (e.g. for
// cloudflare_zone_settings_override in Atlantis-managed workspaces) and handles
// it transparently without requiring any additional flags from the user.
func runMigration(log hclog.Logger, cfg config) error {
	err := validateVersions(cfg)
	if err != nil {
		return err
	}

	// Check for same-version migration (bypass mode)
	if cfg.sourceVersion == cfg.targetVersion {
		fmt.Printf("\n⚠ Same-version migration detected (%s → %s)\n", cfg.sourceVersion, cfg.targetVersion)
		fmt.Println("Running in bypass mode: Config will be processed but transformations will be minimal")
		fmt.Println()
	}

	// --yes with no removed blocks present means: skip straight to full migration.
	// This is the e2e runner's second-call path: fresh v4 files, state already
	// cleaned by the phase-1 apply, just run the full migration.
	if cfg.yes {
		return runFullMigration(log, cfg)
	}

	// Scan for resources that require phased migration.
	phaseOneResources, err := detectPhaseOneResources(log, cfg)
	if err != nil {
		return fmt.Errorf("failed to scan for phase-1 resources: %w", err)
	}

	if len(phaseOneResources) > 0 {
		// Check whether removed {} blocks have already been written to the files,
		// meaning phase 1 ran previously. Ask the user whether they have committed
		// and applied those blocks so we know if state is clean.
		if filesAlreadyHaveRemovedBlocks(phaseOneResources) {
			confirmed, promptErr := confirmPhaseOneApplied(cfg)
			if promptErr != nil {
				return promptErr
			}
			if confirmed {
				// Delete all cleanup files so the original resource blocks can
				// coexist cleanly with the full migration output.
				for file := range phaseOneResources {
					cleanupPath := filepath.Join(filepath.Dir(file), phaseOneCleanupFilename)
					if err := os.Remove(cleanupPath); err != nil && !os.IsNotExist(err) {
						return fmt.Errorf("failed to delete %s: %w", cleanupPath, err)
					}
				}
				// User confirms state is clean — run the full v5 migration.
				return runPhaseTwo(log, cfg)
			}
			// User says not yet — remind them what to do and exit.
			fmt.Println()
			fmt.Printf("No problem. Once the apply completes, delete the %s file(s) and re-run tf-migrate:\n", phaseOneCleanupFilename)
			for file := range phaseOneResources {
				fmt.Printf("  rm %s\n", filepath.Join(filepath.Dir(file), phaseOneCleanupFilename))
			}
			fmt.Printf("  tf-migrate migrate --config-dir %s\n", cfg.configDir)
			return nil
		}

		// No removed blocks yet — run phase 1.
		return runPhaseOne(log, cfg, phaseOneResources)
	}

	// No phase-1 resources — run the standard full migration.
	return runFullMigration(log, cfg)
}

// runFullMigration runs the standard pipeline migration without any phasing.
func runFullMigration(log hclog.Logger, cfg config) error {
	providers := getProviders(cfg.resourcesToMigrate...)
	configPipeline := pipeline.BuildConfigPipeline(log, providers)
	var allDiagnostics hcl.Diagnostics
	if cfg.configDir != "" {
		var err error
		_, allDiagnostics, err = processConfigFiles(log, configPipeline, cfg)
		if err != nil {
			return fmt.Errorf("failed to process configuration files: %w", err)
		}
	}
	log.Debug("Finished processing configuration files")
	printDiagnostics(allDiagnostics, cfg)
	return nil
}

// filesAlreadyHaveRemovedBlocks returns true if any _phase1_cleanup.tf file
// exists in the directories containing phase-1 resources. This detects re-runs
// after phase 1 has written the cleanup files but before the user has deleted
// them and confirmed the apply succeeded.
func filesAlreadyHaveRemovedBlocks(phaseOneResources map[string][]string) bool {
	for file := range phaseOneResources {
		cleanupPath := filepath.Join(filepath.Dir(file), phaseOneCleanupFilename)
		if _, err := os.Stat(cleanupPath); err == nil {
			return true
		}
	}
	return false
}

// confirmPhaseOneApplied asks the user whether they have committed and applied
// the phase-1 removed {} blocks. When --yes is set it auto-confirms (for CI).
func confirmPhaseOneApplied(cfg config) (bool, error) {
	if cfg.yes {
		return true, nil
	}

	fmt.Println()
	fmt.Printf("It looks like %s already exists from a previous phase-1 run.\n", phaseOneCleanupFilename)
	fmt.Println()
	fmt.Printf("Did you commit %s, apply it via Atlantis/CI, and confirm the\n", phaseOneCleanupFilename)
	fmt.Print("zone_settings_override entries were removed from state? [y/N]: ")

	var answer string
	if _, err := fmt.Scanln(&answer); err != nil {
		// Non-interactive environment (e.g. piped input) — treat as "no"
		return false, nil
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}

// detectPhaseOneResources scans all .tf files and returns a map of
// filename → resource addresses for resources whose migrator implements PhaseOneTransformer.
func detectPhaseOneResources(log hclog.Logger, cfg config) (map[string][]string, error) {
	files, err := findTerraformFilesWithRecursion(cfg.configDir, cfg.recursive)
	if err != nil {
		return nil, err
	}

	providers := getProviders(cfg.resourcesToMigrate...)
	found := make(map[string][]string)

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			log.Warn("Failed to read file during phase-1 scan", "file", file, "error", err)
			continue
		}

		parsed, diags := hclwrite.ParseConfig(content, filepath.Base(file), hcl.InitialPos)
		if diags.HasErrors() {
			log.Warn("Failed to parse file during phase-1 scan", "file", file, "error", diags)
			continue
		}

		for _, block := range parsed.Body().Blocks() {
			if block.Type() != "resource" || len(block.Labels()) < 2 {
				continue
			}
			resourceType := block.Labels()[0]
			resourceName := block.Labels()[1]

			migrator := providers.GetMigrator(resourceType, cfg.sourceVersion, cfg.targetVersion)
			if migrator == nil {
				continue
			}
			if _, ok := migrator.(transform.PhaseOneTransformer); ok {
				addr := resourceType + "." + resourceName
				found[file] = append(found[file], addr)
			}
		}
	}

	return found, nil
}

// phaseOneCleanupFilename is the name of the temporary file written during
// phase 1 that contains the removed {} blocks. It is written into the same
// directory as the source .tf file (which may be a module subdirectory), so
// that the removed {} block addresses match the resources exactly — no module
// prefix needed since Terraform resolves addresses relative to the module.
// The file must be deleted before re-running tf-migrate for the full migration.
const phaseOneCleanupFilename = "_phase1_cleanup.tf"

// runPhaseOne writes removed {} blocks for all PhaseOneTransformer resources
// into _phase1_cleanup.tf files co-located with the source .tf files.
// Writing the cleanup file into the same directory as the resources ensures the
// removed {} addresses are correct (bare resource addresses, no module prefix),
// and that Terraform processes them in the right module context.
//
// The original .tf files are left completely untouched — this is essential so
// that tf-migrate can perform the full v5 migration in phase 2 using the
// original resource definitions.
//
// User workflow:
//  1. Commit and push _phase1_cleanup.tf files (originals unchanged)
//  2. Atlantis applies with v4 provider → state entries dropped
//  3. Delete _phase1_cleanup.tf files
//  4. Re-run tf-migrate → full v5 migration from intact original files
func runPhaseOne(log hclog.Logger, cfg config, phaseOneResources map[string][]string) error {
	fmt.Println("Phased Migration — Phase 1: State Cleanup")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("The following resources have no schema in the v5 provider and must be")
	fmt.Println("removed from Terraform state before the full migration can proceed:")
	fmt.Println()

	for _, addrs := range phaseOneResources {
		for _, addr := range addrs {
			fmt.Printf("  • %s\n", addr)
		}
	}
	fmt.Println()

	providers := getProviders(cfg.resourcesToMigrate...)

	// Group removed {} blocks by directory so each dir gets its own cleanup file.
	// This ensures the removed {} addresses are correct within each module context.
	type dirCleanup struct {
		file  *hclwrite.File
		body  *hclwrite.Body
		count int
	}
	byDir := make(map[string]*dirCleanup)

	totalBlocks := 0

	for file := range phaseOneResources {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		parsed, diags := hclwrite.ParseConfig(content, filepath.Base(file), hcl.InitialPos)
		if diags.HasErrors() {
			return fmt.Errorf("failed to parse %s: %w", file, diags)
		}

		ctx := &transform.Context{
			Content:       content,
			Filename:      filepath.Base(file),
			Diagnostics:   make(hcl.Diagnostics, 0),
			Metadata:      make(map[string]interface{}),
			SourceVersion: cfg.sourceVersion,
			TargetVersion: cfg.targetVersion,
			Resources:     cfg.resourcesToMigrate,
		}

		dir := filepath.Dir(file)
		if _, ok := byDir[dir]; !ok {
			f := hclwrite.NewEmptyFile()
			byDir[dir] = &dirCleanup{file: f, body: f.Body()}
		}
		dc := byDir[dir]

		for _, block := range parsed.Body().Blocks() {
			if block.Type() != "resource" || len(block.Labels()) < 2 {
				continue
			}
			resourceType := block.Labels()[0]
			migrator := providers.GetMigrator(resourceType, cfg.sourceVersion, cfg.targetVersion)
			if migrator == nil {
				continue
			}
			p1, ok := migrator.(transform.PhaseOneTransformer)
			if !ok {
				continue
			}
			result, err := p1.TransformPhaseOne(ctx, block)
			if err != nil {
				return fmt.Errorf("phase-1 transform failed for %s in %s: %w", block.Labels()[1], file, err)
			}
			if result != nil {
				for _, b := range result.Blocks {
					dc.body.AppendNewline()
					dc.body.AppendBlock(b)
					dc.count++
					totalBlocks++
				}
			}
		}
	}

	if totalBlocks == 0 {
		return nil
	}

	if cfg.dryRun {
		fmt.Printf("(dry run — would write %d %s file(s) with %d removed block(s) total)\n",
			len(byDir), phaseOneCleanupFilename, totalBlocks)
		fmt.Println("Original .tf files would NOT be modified.")
		return nil
	}

	var cleanupPaths []string
	for dir, dc := range byDir {
		if dc.count == 0 {
			continue
		}
		cleanupPath := filepath.Join(dir, phaseOneCleanupFilename)
		formatted := hclwrite.Format(dc.file.Bytes())
		if err := os.WriteFile(cleanupPath, formatted, 0644); err != nil {
			return fmt.Errorf("failed to write cleanup file in %s: %w", dir, err)
		}
		fmt.Printf("✓ Written %s with %d removed block(s)\n",
			filepath.Join(filepath.Base(dir), phaseOneCleanupFilename), dc.count)
		cleanupPaths = append(cleanupPaths, cleanupPath)
	}
	fmt.Println("  Original .tf files are unchanged.")
	fmt.Println()
	fmt.Println("Phase 1 complete.")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println()
	fmt.Printf("  1. Commit and push %s (your other .tf files are unchanged).\n", phaseOneCleanupFilename)
	fmt.Println("     Your CI/Atlantis pipeline will run terraform plan and apply using the")
	fmt.Println("     CURRENT (v4) provider. Terraform will process the removed {} blocks")
	fmt.Println("     and drop the old state entries without destroying any infrastructure.")
	fmt.Println()
	fmt.Println("  2. Wait for the apply to complete successfully.")
	fmt.Println()
	fmt.Printf("  3. Delete the %s file(s):\n", phaseOneCleanupFilename)
	for _, p := range cleanupPaths {
		fmt.Printf("       rm %s\n", p)
	}
	fmt.Println()
	fmt.Println("  4. Re-run tf-migrate to complete the full migration:")
	fmt.Printf("       tf-migrate migrate --config-dir %s\n", cfg.configDir)
	fmt.Println()
	fmt.Println("     tf-migrate will ask you to confirm the apply succeeded, then run the")
	fmt.Println("     full v5 migration from your original (unchanged) config files.")

	return nil
}

// runPhaseTwo runs the standard full migration after the user has confirmed
// that the phase-1 removed {} blocks were applied successfully.
func runPhaseTwo(log hclog.Logger, cfg config) error {
	fmt.Println("Phased Migration — Phase 2: Full Migration")
	fmt.Println("===========================================")
	fmt.Println()
	fmt.Println("Running full v4 → v5 migration...")
	fmt.Println()

	return runFullMigration(log, cfg)
}

func processConfigFiles(log hclog.Logger, p *pipeline.Pipeline, cfg config) (map[string]*hclwrite.File, hcl.Diagnostics, error) {
	if cfg.outputDir == "" {
		cfg.outputDir = cfg.configDir
	}

	files, err := findTerraformFilesWithRecursion(cfg.configDir, cfg.recursive)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list .tf files: %w", err)
	}

	if len(files) == 0 {
		fmt.Printf("No .tf files found in %s\n", cfg.configDir)
		return nil, nil, nil
	}

	fmt.Printf("\nFound %d configuration files to migrate\n", len(files))

	// Store file paths for global postprocessing
	outputPaths := make([]string, 0, len(files))

	// Collect diagnostics from all files
	var allDiagnostics hcl.Diagnostics

	parsedConfigs := make(map[string]*hclwrite.File)
	for i, file := range files {
		fmt.Printf("[%d/%d] Processing %s... ", i+1, len(files), filepath.Base(file))
		log.Debug("Processing file", "file", file, "index", i+1)

		content, err := os.ReadFile(file)
		if err != nil {
			return nil, allDiagnostics, fmt.Errorf("failed to read %s: %w", file, err)
		}

		if cfg.backup && !cfg.dryRun && cfg.outputDir == cfg.configDir {
			backupPath := file + ".backup"
			if err := os.WriteFile(backupPath, content, 0644); err != nil {
				return nil, allDiagnostics, fmt.Errorf("failed to create backup %s: %w", backupPath, err)
			}
			log.Debug("Created backup", "path", backupPath)
		}

		ctx := &transform.Context{
			Content:       content,
			Filename:      filepath.Base(file),
			Diagnostics:   make(hcl.Diagnostics, 0),
			Metadata:      make(map[string]interface{}),
			SourceVersion: cfg.sourceVersion,
			TargetVersion: cfg.targetVersion,
			Resources:     cfg.resourcesToMigrate,
		}
		transformed, err := p.Transform(ctx)
		if err != nil {
			return nil, allDiagnostics, fmt.Errorf("failed to transform %s: %w", file, err)
		}

		// Collect diagnostics from this file's context
		allDiagnostics = append(allDiagnostics, ctx.Diagnostics...)

		if ctx.CFGFile != nil {
			parsedConfigs[file] = ctx.CFGFile
		}

		// Calculate output path maintaining directory structure when recursive
		var outputPath string
		if cfg.recursive {
			// Preserve directory structure relative to config dir
			relPath, err := filepath.Rel(cfg.configDir, file)
			if err != nil {
				return nil, allDiagnostics, fmt.Errorf("failed to compute relative path: %w", err)
			}
			outputPath = filepath.Join(cfg.outputDir, relPath)
		} else {
			outputPath = filepath.Join(cfg.outputDir, filepath.Base(file))
		}

		if cfg.dryRun {
			fmt.Println("(dry run)")
			log.Debug("Would write file", "output", outputPath)
			outputPaths = append(outputPaths, outputPath)
			continue
		}

		// Create output directory (including subdirectories if needed)
		outputDir := filepath.Dir(outputPath)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, allDiagnostics, fmt.Errorf("failed to create output directory: %w", err)
		}

		if err := os.WriteFile(outputPath, transformed, 0644); err != nil {
			return nil, allDiagnostics, fmt.Errorf("failed to write %s: %w", outputPath, err)
		}
		fmt.Println("✓")
		log.Debug("Migrated file", "output", outputPath)
		outputPaths = append(outputPaths, outputPath)

	}

	// Apply global postprocessing for cross-file reference updates
	if !cfg.dryRun && len(outputPaths) > 0 {
		if err := applyGlobalPostprocessing(log, cfg, outputPaths); err != nil {
			return nil, allDiagnostics, fmt.Errorf("failed to apply global postprocessing: %w", err)
		}
	}

	return parsedConfigs, allDiagnostics, nil
}

func applyGlobalPostprocessing(log hclog.Logger, cfg config, outputPaths []string) error {
	// Collect resource renames and attribute renames from all migrators
	providers := getProviders(cfg.resourcesToMigrate...)
	migrators := providers.GetAllMigrators(cfg.sourceVersion, cfg.targetVersion, cfg.resourcesToMigrate...)

	// Map to store old type -> new type mappings
	renames := make(map[string]string)
	// Slice to store attribute renames
	var attributeRenames []transform.AttributeRename

	for _, migrator := range migrators {
		// Check if this migrator implements ResourceRenamer interface
		if renamer, ok := migrator.(transform.ResourceRenamer); ok {
			oldTypes, newType := renamer.GetResourceRename()
			if len(oldTypes) > 0 && newType != "" {
				// Process each old type
				for _, oldType := range oldTypes {
					// Only add to renames map if the types are different (actual rename)
					if oldType != newType {
						renames[oldType] = newType
						log.Debug("Collected resource rename", "old", oldType, "new", newType)
					} else {
						log.Debug("Resource type unchanged", "type", oldType)
					}
				}
			} else {
				// Warn if migrator implements interface but returned empty values
				log.Warn("Migrator implements ResourceRenamer but returned empty type names",
					"oldTypes", oldTypes, "newType", newType)
			}
		} else {
			// Warn if migrator doesn't implement ResourceRenamer interface
			log.Warn("Migrator does not implement ResourceRenamer interface - cross-file references may not be updated",
				"migrator", fmt.Sprintf("%T", migrator))
		}

		// Check if this migrator implements AttributeRenamer interface
		if attrRenamer, ok := migrator.(transform.AttributeRenamer); ok {
			renames := attrRenamer.GetAttributeRenames()
			if len(renames) > 0 {
				attributeRenames = append(attributeRenames, renames...)
				for _, r := range renames {
					log.Debug("Collected attribute rename",
						"resource_type", r.ResourceType,
						"old_attr", r.OldAttribute,
						"new_attr", r.NewAttribute)
				}
			}
		}
	}

	// If no renames found, skip global postprocessing
	if len(renames) == 0 && len(attributeRenames) == 0 {
		log.Debug("No renames found, skipping global postprocessing")
		return nil
	}

	totalUpdates := len(renames) + len(attributeRenames)
	fmt.Printf("\nApplying cross-file reference updates (%d updates across %d files)...\n", totalUpdates, len(outputPaths))

	// Print summary of resource type renames (always shown)
	if len(renames) > 0 {
		fmt.Println("\nResource type renames:")
		for oldType, newType := range renames {
			fmt.Printf("  %s → %s\n", oldType, newType)
		}
	}

	// Print summary of attribute renames (always shown)
	if len(attributeRenames) > 0 {
		fmt.Println("\nAttribute renames:")
		for _, rename := range attributeRenames {
			fmt.Printf("  %s.*.%s → %s.*.%s\n", rename.ResourceType, rename.OldAttribute, rename.ResourceType, rename.NewAttribute)
		}
	}

	// Apply renames to all files
	for _, outputPath := range outputPaths {
		content, err := os.ReadFile(outputPath)
		if err != nil {
			log.Warn("Failed to read file for global postprocessing", "file", outputPath, "error", err)
			continue
		}

		contentStr := string(content)
		modified := false

		// Apply all resource type renames, but skip content within moved blocks
		for oldType, newType := range renames {
			newContent := replaceSkippingMovedBlocks(contentStr, oldType+".", newType+".")
			if newContent != contentStr {
				modified = true
				contentStr = newContent
				log.Debug("Updated references", "file", filepath.Base(outputPath), "old", oldType, "new", newType)
			}
		}

		// Apply all attribute renames, but skip content within moved blocks
		// Pattern: data.cloudflare_zones.<instance_name>.zones → data.cloudflare_zones.<instance_name>.result
		// We need to match: <ResourceType>.<instance_name>.<OldAttribute>
		for _, rename := range attributeRenames {
			// Build regex pattern: data\.cloudflare_zones\.([a-zA-Z0-9_-]+)\.zones
			// The instance name can contain letters, numbers, underscores, and hyphens
			pattern := rename.ResourceType + `\.([a-zA-Z0-9_-]+)\.` + rename.OldAttribute
			replacement := rename.ResourceType + ".$1." + rename.NewAttribute
			newContent := regexReplaceSkippingMovedBlocks(contentStr, pattern, replacement)

			if newContent != contentStr {
				modified = true
				contentStr = newContent
				log.Debug("Updated attribute references",
					"file", filepath.Base(outputPath),
					"resource_type", rename.ResourceType,
					"old_attr", rename.OldAttribute,
					"new_attr", rename.NewAttribute)
			}
		}

		// Write back if modified
		if modified {
			if err := os.WriteFile(outputPath, []byte(contentStr), 0644); err != nil {
				return fmt.Errorf("failed to write updated file %s: %w", outputPath, err)
			}
		}
	}

	fmt.Printf("✓ Updated cross-file references (%d updates applied)\n", totalUpdates)
	return nil
}

// findProtectedBlockRanges returns the [start, end) byte ranges of all moved and
// removed blocks in content. Uses brace counting to correctly handle nested braces
// (e.g. a removed block containing a lifecycle { } sub-block).
func findProtectedBlockRanges(content string) [][2]int {
	var ranges [][2]int
	i := 0
	for i < len(content) {
		// Look for "moved {" or "removed {" at the start of a line (possibly indented)
		for _, keyword := range []string{"moved", "removed"} {
			if !strings.HasPrefix(content[i:], keyword) {
				continue
			}
			// Must be followed by optional whitespace then "{"
			j := i + len(keyword)
			for j < len(content) && (content[j] == ' ' || content[j] == '\t' || content[j] == '\n' || content[j] == '\r') {
				j++
			}
			if j >= len(content) || content[j] != '{' {
				continue
			}
			// Found a block — count braces to find the end
			start := i
			depth := 0
			k := j
			for k < len(content) {
				if content[k] == '{' {
					depth++
				} else if content[k] == '}' {
					depth--
					if depth == 0 {
						ranges = append(ranges, [2]int{start, k + 1})
						i = k + 1
						goto nextChar
					}
				}
				k++
			}
		}
		i++
	nextChar:
	}
	return ranges
}

// replaceSkippingMovedBlocks replaces old with new in content, skipping moved and
// removed blocks. Uses brace counting to correctly handle nested braces.
func replaceSkippingMovedBlocks(content, old, new string) string {
	protected := findProtectedBlockRanges(content)
	if len(protected) == 0 {
		return strings.ReplaceAll(content, old, new)
	}

	var result strings.Builder
	lastEnd := 0
	for _, r := range protected {
		result.WriteString(strings.ReplaceAll(content[lastEnd:r[0]], old, new))
		result.WriteString(content[r[0]:r[1]])
		lastEnd = r[1]
	}
	result.WriteString(strings.ReplaceAll(content[lastEnd:], old, new))
	return result.String()
}

// regexReplaceSkippingMovedBlocks replaces regex matches in content, skipping moved
// and removed blocks. Uses brace counting to correctly handle nested braces.
func regexReplaceSkippingMovedBlocks(content, pattern, replacement string) string {
	protected := findProtectedBlockRanges(content)
	re := regexp.MustCompile(pattern)
	if len(protected) == 0 {
		return re.ReplaceAllString(content, replacement)
	}

	var result strings.Builder
	lastEnd := 0
	for _, r := range protected {
		result.WriteString(re.ReplaceAllString(content[lastEnd:r[0]], replacement))
		result.WriteString(content[r[0]:r[1]])
		lastEnd = r[1]
	}
	result.WriteString(re.ReplaceAllString(content[lastEnd:], replacement))
	return result.String()
}

func findTerraformFiles(dir string) ([]string, error) {
	return findTerraformFilesWithRecursion(dir, false)
}

func findTerraformFilesWithRecursion(dir string, recursive bool) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() && recursive {
			// Recursively search subdirectories
			subFiles, err := findTerraformFilesWithRecursion(path, recursive)
			if err != nil {
				// Log the error but continue processing other directories
				continue
			}
			files = append(files, subFiles...)
		} else if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tf") {
			files = append(files, path)
		}
	}

	return files, nil
}

func validateVersions(c config) error {
	versionPath := fmt.Sprintf("%s-%s", c.sourceVersion, c.targetVersion)
	if _, ok := validVersionPath[versionPath]; !ok {
		return fmt.Errorf("unsupported migration path: %s", versionPath)
	}
	return nil
}

func getProviders(resources ...string) transform.MigrationProvider {
	getFunc := func(resourceType string, source string, target string) transform.ResourceTransformer {
		return internal.GetMigrator(resourceType, source, target)
	}
	getAllFunc := func(source string, target string, resourcesToMigrate ...string) []transform.ResourceTransformer {
		return internal.GetAllMigrators(source, target, resources...)
	}
	return transform.NewMigrationProvider(getFunc, getAllFunc)
}

// printDiagnostics prints diagnostics to the user based on verbosity settings
// Default (medium verbosity): show warnings and errors
// Quiet mode (--quiet): only show errors
// Verbose mode (--verbose): show all diagnostics including informational messages
func printDiagnostics(diags hcl.Diagnostics, cfg config) {
	if diags == nil || len(diags) == 0 {
		return
	}

	// Filter diagnostics based on verbosity level
	var filtered []*hcl.Diagnostic
	var errors, warnings, infos int

	for _, diag := range diags {
		switch diag.Severity {
		case hcl.DiagError:
			errors++
			// Errors are always shown
			filtered = append(filtered, diag)
		case hcl.DiagWarning:
			warnings++
			// Warnings are shown unless quiet mode is enabled
			if !cfg.quiet {
				filtered = append(filtered, diag)
			}
		default:
			// Informational messages (DiagInvalid or any other)
			infos++
			// Only shown in verbose mode
			if cfg.verbose {
				filtered = append(filtered, diag)
			}
		}
	}

	if len(filtered) == 0 {
		// In quiet mode, still show a summary if there were warnings
		if cfg.quiet && warnings > 0 {
			fmt.Printf("\n⚠ %d warning(s) suppressed (use without --quiet to see details)\n", warnings)
		}
		return
	}

	fmt.Println()
	// Print summary header
	var parts []string
	if errors > 0 {
		parts = append(parts, fmt.Sprintf("%d error(s)", errors))
	}
	if warnings > 0 {
		if cfg.quiet {
			parts = append(parts, fmt.Sprintf("%d warning(s) suppressed", warnings))
		} else {
			parts = append(parts, fmt.Sprintf("%d warning(s)", warnings))
		}
	}
	if cfg.verbose && infos > 0 {
		parts = append(parts, fmt.Sprintf("%d info message(s)", infos))
	}
	fmt.Printf("Migration diagnostics: %s\n", strings.Join(parts, ", "))
	fmt.Println(strings.Repeat("─", 70))

	for i, diag := range filtered {
		if i > 0 {
			fmt.Println()
		}

		// Print severity icon
		var icon string
		switch diag.Severity {
		case hcl.DiagError:
			icon = "✗"
		case hcl.DiagWarning:
			icon = "⚠"
		default:
			icon = "ℹ"
		}

		fmt.Printf("\n%s %s\n", icon, diag.Summary)
		if diag.Detail != "" {
			fmt.Printf("\n%s\n", diag.Detail)
		}
	}
	fmt.Println(strings.Repeat("─", 70))
}
