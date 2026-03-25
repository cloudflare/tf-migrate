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

	// --yes skips straight to full migration (used by e2e runner and CI).
	if cfg.yes {
		return runFullMigration(log, cfg)
	}

	// Check whether phase 1 has already run by looking for commented-out
	// resource blocks with the tf-migrate phase-1 marker in the files.
	commentedFiles := findFilesWithPhaseOneComments(cfg)
	if len(commentedFiles) > 0 {
		// Phase 1 already ran — ask the user whether the apply succeeded.
		confirmed, promptErr := confirmPhaseOneApplied(cfg)
		if promptErr != nil {
			return promptErr
		}
		if confirmed {
			// User confirms state is clean — uncomment resource blocks,
			// remove the removed {} blocks, then run the full v5 migration.
			return runPhaseTwo(log, cfg, commentedFiles)
		}
		// User says not yet — print instructions and exit.
		fmt.Println()
		fmt.Println("No problem. Once the apply completes successfully, re-run tf-migrate:")
		fmt.Printf("  tf-migrate migrate --config-dir %s\n", cfg.configDir)
		return nil
	}

	// Scan for resources that require phased migration.
	phaseOneResources, err := detectPhaseOneResources(log, cfg)
	if err != nil {
		return fmt.Errorf("failed to scan for phase-1 resources: %w", err)
	}

	if len(phaseOneResources) > 0 {
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

// findFilesWithPhaseOneComments returns a list of .tf files that contain
// commented-out resource blocks from a previous phase-1 run, identified by
// the phaseOneCommentPrefix marker.
func findFilesWithPhaseOneComments(cfg config) []string {
	files, err := findTerraformFilesWithRecursion(cfg.configDir, cfg.recursive)
	if err != nil {
		return nil
	}
	var found []string
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		if strings.Contains(string(content), phaseOneCommentPrefix+"resource ") {
			found = append(found, file)
		}
	}
	return found
}

// confirmPhaseOneApplied asks the user whether they have committed and applied
// the phase-1 changes. When --yes is set it auto-confirms (for CI).
func confirmPhaseOneApplied(cfg config) (bool, error) {
	if cfg.yes {
		return true, nil
	}

	fmt.Println()
	fmt.Println("It looks like phase 1 has already run (resource blocks are commented out).")
	fmt.Println()
	fmt.Print("Did you apply the v4 config and remove the resources from state? [y/N]: ")

	var answer string
	if _, err := fmt.Scanln(&answer); err != nil {
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

// phaseOneCommentPrefix is the line prefix used to comment out resource blocks
// during phase 1. It acts as a marker so phase 2 can identify and uncomment them.
const phaseOneCommentPrefix = "# tf-migrate: "

// runPhaseOne rewrites each .tf file containing PhaseOneTransformer resources:
//   - Comments out each resource block using phaseOneCommentPrefix so Terraform
//     ignores it, but tf-migrate can recover the definition in phase 2.
//   - Appends a removed {} block immediately after the commented-out block so
//     Terraform drops the state entry on the next plan/apply.
//
// Since the resource block is commented out, Terraform only sees the removed {}
// block — no coexistence error. The original definitions are preserved as
// comments for phase 2 to uncomment and transform.
//
// User workflow:
//  1. Commit and push the modified .tf files
//  2. Atlantis plans and applies with v4 provider → state entries dropped
//  3. Re-run tf-migrate → detects commented blocks, prompts for confirmation,
//     uncoments and runs the full v5 migration
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

		// Collect (block text, removed block) pairs for this file.
		type phaseOnePair struct {
			blockText    string // raw bytes of the original resource block
			removedBlock *hclwrite.Block
		}
		var pairs []phaseOnePair

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
			if result != nil && len(result.Blocks) > 0 {
				blockText := string(block.BuildTokens(nil).Bytes())
				pairs = append(pairs, phaseOnePair{
					blockText:    blockText,
					removedBlock: result.Blocks[0],
				})
				totalBlocks++
			}
		}

		if len(pairs) == 0 {
			continue
		}

		if cfg.dryRun {
			fmt.Printf("[%s] (dry run — would comment out %d block(s) and add removed {} blocks)\n",
				filepath.Base(file), len(pairs))
			continue
		}

		// Build new file content: for each phase-1 resource block, comment it
		// out (with the marker prefix) and append a removed {} block after it.
		newContent := string(content)
		for _, pair := range pairs {
			commented := commentOutBlock(pair.blockText)
			// Build the removed {} block text
			scratch := hclwrite.NewEmptyFile()
			scratch.Body().AppendNewline()
			scratch.Body().AppendBlock(pair.removedBlock)
			removedText := string(hclwrite.Format(scratch.Bytes()))
			newContent = strings.Replace(newContent, pair.blockText, commented+removedText, 1)
		}

		if cfg.backup {
			if err := os.WriteFile(file+".backup", content, 0644); err != nil {
				return fmt.Errorf("failed to create backup for %s: %w", file, err)
			}
		}
		if err := os.WriteFile(file, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", file, err)
		}
		fmt.Printf("✓ %s — commented out %d resource block(s), added removed {} block(s)\n",
			filepath.Base(file), len(pairs))
	}

	if cfg.dryRun {
		fmt.Println()
		fmt.Println("(dry run — no files modified)")
		return nil
	}

	if totalBlocks == 0 {
		return nil
	}

	fmt.Println()
	fmt.Println("Phase 1 complete.")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println()
	fmt.Println("  1. Commit and push the modified .tf files.")
	fmt.Println("     Your CI/Atlantis pipeline will run terraform plan and apply using the")
	fmt.Println("     CURRENT (v4) provider. Terraform will process the removed {} blocks")
	fmt.Println("     and drop the old state entries without destroying any infrastructure.")
	fmt.Println()
	fmt.Println("  2. Wait for the apply to complete successfully.")
	fmt.Println()
	fmt.Println("  3. Re-run tf-migrate to complete the full migration:")
	fmt.Printf("       tf-migrate migrate --config-dir %s\n", cfg.configDir)
	fmt.Println()
	fmt.Println("     tf-migrate will detect the commented-out blocks, ask you to confirm")
	fmt.Println("     the apply succeeded, uncomment them, and run the full v5 migration.")

	return nil
}

// commentOutBlock prefixes every line of blockText with phaseOneCommentPrefix.
func commentOutBlock(blockText string) string {
	lines := strings.Split(blockText, "\n")
	var out []string
	for _, line := range lines {
		if line == "" {
			out = append(out, "")
		} else {
			out = append(out, phaseOneCommentPrefix+line)
		}
	}
	return strings.Join(out, "\n")
}

// runPhaseTwo uncomments the resource blocks that were commented out during
// phase 1, removes the removed {} blocks that were added alongside them, then
// runs the full v5 migration.
func runPhaseTwo(log hclog.Logger, cfg config, commentedFiles []string) error {
	fmt.Println("Phased Migration — Phase 2: Full Migration")
	fmt.Println("===========================================")
	fmt.Println()
	fmt.Println("Restoring commented-out resource blocks and running full v5 migration...")
	fmt.Println()

	// Uncomment resource blocks and remove the adjacent removed {} blocks.
	for _, file := range commentedFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}
		restored := restorePhaseOneFile(string(content))
		if err := os.WriteFile(file, []byte(restored), 0644); err != nil {
			return fmt.Errorf("failed to restore %s: %w", file, err)
		}
	}

	return runFullMigration(log, cfg)
}

// restorePhaseOneFile removes the phaseOneCommentPrefix from commented-out
// resource lines and removes the removed {} blocks that follow them.
func restorePhaseOneFile(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	skipUntilClosingBrace := false
	depth := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// If we're inside a removed {} block to skip, track brace depth.
		if skipUntilClosingBrace {
			depth += strings.Count(line, "{") - strings.Count(line, "}")
			if depth <= 0 {
				skipUntilClosingBrace = false
			}
			continue
		}

		// Detect the start of a removed {} block that immediately follows
		// a commented-out block (i.e. preceded by phase-1 commented lines).
		// We identify it by the block type being "removed {".
		trimmed := strings.TrimSpace(line)
		if trimmed == "removed {" {
			// Check if the previous non-empty output line was a commented block.
			// Look backwards in out for the last non-empty line.
			lastNonEmpty := ""
			for j := len(out) - 1; j >= 0; j-- {
				if strings.TrimSpace(out[j]) != "" {
					lastNonEmpty = out[j]
					break
				}
			}
			if strings.HasPrefix(strings.TrimSpace(lastNonEmpty), phaseOneCommentPrefix) {
				// This removed {} block was added by phase 1 — skip it.
				depth = 1
				skipUntilClosingBrace = true
				// Also remove the blank line before removed {} if present.
				if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
					out = out[:len(out)-1]
				}
				continue
			}
		}

		// Uncomment lines that have the phase-1 marker prefix.
		if strings.HasPrefix(line, phaseOneCommentPrefix) {
			out = append(out, strings.TrimPrefix(line, phaseOneCommentPrefix))
			continue
		}

		out = append(out, line)
	}

	return strings.Join(out, "\n")
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
