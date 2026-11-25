package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/spf13/cobra"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/logger"
	"github.com/cloudflare/tf-migrate/internal/pipeline"
	"github.com/cloudflare/tf-migrate/internal/registry"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
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
	resourcesToMigrate    []string
	sourceVersion         string
	targetVersion         string
	targetProviderVersion string // explicit provider version to write into required_providers (overrides API fetch)
	dryRun                bool
	backup                bool
	recursive             bool
	exclude               []string // directories to exclude from migration (relative to configDir)
	logLevel              string
	skipPhaseCheck        bool // skip phased migration prompt and run full migration directly (for CI/e2e)
	skipVersionCheck      bool // skip minimum provider version check (for testing/CI only)

	// Diagnostic output options
	quiet   bool // Suppress warnings, only show errors
	verbose bool // Show all diagnostics including informational messages

	// fileAliases maps each migrated input file to the other paths (symlinks)
	// that resolve to it. Set by processConfigFiles for global postprocessing.
	fileAliases map[string][]string
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
	rootCmd.AddCommand(newPreserveZeroTrustSettingsCommand(cfg))
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

			if cfg.verbose {
				fmt.Println("Cloudflare Terraform Provider Migration Tool")
				fmt.Println("============================================")
				fmt.Println()
				fmt.Printf("Configuration directory: %s\n", cfg.configDir)
				if cfg.outputDir != "" {
					fmt.Printf("Output directory: %s\n", cfg.outputDir)
				} else {
					fmt.Println("Output directory: in-place")
				}
				fmt.Println()
			}

			if cfg.dryRun {
				fmt.Println("\n DRY RUN MODE - No changes will be made")
			}

			// Check minimum provider version early to provide clear error without help output
			if err := checkMinimumProviderVersion(*cfg); err != nil {
				cmd.SilenceUsage = true
				return err
			}

			return runMigration(log, *cfg)
		},
	}

	cmd.Flags().StringVar(&cfg.outputDir, "output-dir", "", "Output directory for migrated configuration files (default: in-place)")
	cmd.Flags().BoolVar(&cfg.backup, "backup", true, "Create backup of original files before migration")
	cmd.Flags().BoolVar(&cfg.recursive, "recursive", false, "Recursively process subdirectories (useful for module structures)")
	cmd.Flags().StringSliceVar(&cfg.exclude, "exclude", []string{}, "Directories to exclude from migration (relative to config-dir, can be specified multiple times)")
	cmd.Flags().StringVar(&cfg.targetProviderVersion, "target-provider-version", "", "Explicit provider version to set in required_providers (e.g. 5.19.0-beta.3); skips GitHub API lookup")

	// --no-backup is a convenience alias for --backup=false
	var noBackup bool
	cmd.Flags().BoolVar(&noBackup, "no-backup", false, "Skip creating backup files before migration (alias for --backup=false)")
	cmd.Flags().BoolVar(&cfg.skipPhaseCheck, "skip-phase-check", false, "Skip the phased migration confirmation prompt and run the full migration directly (for CI/non-interactive use)")
	cmd.Flags().BoolVar(&cfg.skipVersionCheck, "skip-version-check", false, "Skip the minimum provider version check (for testing/CI only)")
	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		if noBackup {
			cfg.backup = false
		}
	}

	// Diagnostic output options
	cmd.Flags().BoolVarP(&cfg.quiet, "quiet", "q", false, "Suppress warnings, only show errors")
	cmd.Flags().BoolVarP(&cfg.verbose, "verbose", "v", false, "Show verbose output: per-file progress, rename tables, and all diagnostics")

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

	// Check that the installed provider version meets minimum requirements
	if err := checkMinimumProviderVersion(cfg); err != nil {
		return err
	}

	// Check for same-version migration (bypass mode)
	if cfg.sourceVersion == cfg.targetVersion {
		fmt.Printf("\n⚠ Same-version migration detected (%s → %s)\n", cfg.sourceVersion, cfg.targetVersion)
		fmt.Println("Running in bypass mode: Config will be processed but transformations will be minimal")
		fmt.Println()
	}

	// Run pre-migration scan to classify resources and detect issues.
	if cfg.configDir != "" {
		report, scanErr := runPreMigrationScan(log, cfg)
		if scanErr != nil {
			log.Warn("Pre-migration scan failed", "error", scanErr)
		} else {
			printPreflightReport(report, cfg)

			// If there are warnings about conflicting moved blocks and we're in
			// interactive mode, give the user a chance to abort.
			if len(report.Warnings) > 0 && !cfg.skipPhaseCheck {
				fmt.Print("Proceed with migration? [Y/n]: ")
				var answer string
				if _, err := fmt.Scanln(&answer); err == nil {
					answer = strings.ToLower(strings.TrimSpace(answer))
					if answer == "n" || answer == "no" {
						fmt.Println("Migration aborted. Please resolve the warnings above and re-run tf-migrate.")
						return nil
					}
				}
			}
		}
	}

	// --yes skips straight to full migration (used by e2e runner and CI).
	if cfg.skipPhaseCheck {
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

	// Update the provider version constraint in required_providers blocks
	// and print instructions for regenerating the lock file.
	if cfg.configDir != "" && !cfg.dryRun {
		if err := updateProviderVersionConstraint(log, cfg, allDiagnostics); err != nil {
			log.Warn("Failed to update provider version constraint", "error", err)
		}
	}
	printAccessPolicyMigrationReminder(allDiagnostics, cfg)

	return nil
}

// targetProviderVersion fetches the latest provider release for the given
// target migration version from the GitHub releases API.
//
// Pre-releases (betas) are included because v5 is currently in beta. Draft
// releases are excluded as they are unpublished works-in-progress.
//
// Returns ("", false) if the API is unreachable or rate-limited — callers
// should warn the user to update the version manually rather than silently
// writing a stale hardcoded version.
func targetProviderVersion(targetVersion string) (string, bool) {
	majorPrefix := strings.TrimPrefix(targetVersion, "v") + "."

	type release struct {
		TagName    string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		PreRelease bool   `json:"prerelease"`
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/cloudflare/terraform-provider-cloudflare/releases?per_page=50")
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var releases []release
		if json.NewDecoder(resp.Body).Decode(&releases) == nil {
			for _, r := range releases {
				// Skip unpublished drafts; include pre-releases (betas).
				if r.Draft {
					continue
				}
				version := strings.TrimPrefix(r.TagName, "v")
				if strings.HasPrefix(version, majorPrefix) {
					return version, true
				}
			}
		}
	}

	return "", false
}

// updateProviderVersionConstraint scans all .tf files for a required_providers
// block containing cloudflare/cloudflare and updates the version constraint to
// match the target provider version. Prints next-step instructions if updated.
func updateProviderVersionConstraint(log hclog.Logger, cfg config, diags hcl.Diagnostics) error {
	var targetVersion string

	if cfg.targetProviderVersion != "" {
		// User supplied an explicit version via --target-provider-version; use it directly.
		targetVersion = cfg.targetProviderVersion
		log.Debug("Using explicit provider version from --target-provider-version flag", "version", targetVersion)
	} else {
		var fromAPI bool
		targetVersion, fromAPI = targetProviderVersion(cfg.targetVersion)
		if !fromAPI {
			// GitHub API was unreachable (rate-limited, offline, etc.).
			// Do not write a stale hardcoded version — tell the user to update manually.
			log.Debug("GitHub API unavailable — skipping provider version update")
			printVersionFetchFailure(cfg, diags)
			return nil
		}
		log.Debug("Provider version fetched from GitHub API", "version", targetVersion)
	}

	// Update the migrated files: with --output-dir these are not the input files.
	migratedDir := cfg.outputDir
	if migratedDir == "" {
		migratedDir = cfg.configDir
	}
	files, err := findTerraformFilesWithRecursion(migratedDir, cfg.recursive, cfg.exclude)
	if err != nil {
		return err
	}

	// Track which directories had their provider version updated
	updatedDirs := make(map[string]bool)

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		parsed, diags := hclwrite.ParseConfig(content, filepath.Base(file), hcl.InitialPos)
		if diags.HasErrors() {
			continue
		}

		updated := false
		for _, block := range parsed.Body().Blocks() {
			if block.Type() != "terraform" {
				continue
			}
			for _, inner := range block.Body().Blocks() {
				if inner.Type() != "required_providers" {
					continue
				}
				cfAttr := inner.Body().GetAttribute("cloudflare")
				if cfAttr == nil {
					continue
				}
				// The attribute value is an object expression — check if it
				// contains source = "cloudflare/cloudflare"
				attrStr := string(cfAttr.BuildTokens(nil).Bytes())
				if !strings.Contains(attrStr, "cloudflare/cloudflare") {
					continue
				}
				// Rewrite the version value using raw token replacement.
				// We replace any existing version = "..." with the target.
				versionRe := regexp.MustCompile(`version\s*=\s*"[^"]*"`)
				newAttrStr := versionRe.ReplaceAllString(attrStr, `version = "`+targetVersion+`"`)
				// Only add a constraint when there is none. Comparing the result
				// with the input instead added a second version line whenever the
				// existing one already matched the target (re-runs, symlinks).
				if !versionRe.MatchString(attrStr) {
					// No version constraint existed — add one
					newAttrStr = strings.Replace(attrStr,
						`source  = "cloudflare/cloudflare"`,
						`source  = "cloudflare/cloudflare"`+"\n      version = \""+targetVersion+`"`,
						1)
					if newAttrStr == attrStr {
						// Try without double space
						newAttrStr = strings.Replace(attrStr,
							`source = "cloudflare/cloudflare"`,
							`source  = "cloudflare/cloudflare"`+"\n      version = \""+targetVersion+`"`,
							1)
					}
				}
				if newAttrStr != attrStr {
					// Re-parse the file with the updated attribute text and write it.
					newContent := strings.Replace(string(content), attrStr, newAttrStr, 1)
					if err := os.WriteFile(file, []byte(newContent), 0644); err != nil {
						log.Warn("Failed to write updated provider version", "file", file, "error", err)
						continue
					}
					updated = true
					if cfg.verbose {
						fmt.Printf("✓ Updated cloudflare provider version to %s in %s\n",
							targetVersion, filepath.Base(file))
					}
				}
			}
		}
		if updated {
			// Track the directory containing this file
			dir := filepath.Dir(file)
			updatedDirs[dir] = true
		}
	}

	fmt.Println()
	fmt.Printf("Provider version updated to %s.\n", targetVersion)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println()

	step := 1

	// If there were any actionable warnings (DiagWarning), remind the user
	// to address them before proceeding.
	hasActionableWarnings := false
	for _, d := range diags {
		if d.Severity == hcl.DiagWarning {
			hasActionableWarnings = true
			break
		}
	}
	if hasActionableWarnings {
		fmt.Printf("  %d. Review the warnings above and complete any required manual actions\n", step)
		fmt.Println("     before running terraform plan/apply.")
		fmt.Println()
		step++
	}

	fmt.Printf("  %d. Regenerate the lock file locally in each directory:\n", step)
	if len(updatedDirs) <= 1 {
		// Single directory (or none) - use the original format
		fmt.Printf("       cd %s\n", migratedDir)
		fmt.Println("       terraform init -upgrade -backend=false")
	} else {
		// Multiple directories with provider updates
		// Sort for consistent output
		var sortedDirs []string
		for dir := range updatedDirs {
			sortedDirs = append(sortedDirs, dir)
		}
		// Sort by length then lexically for consistent ordering
		for i := 0; i < len(sortedDirs)-1; i++ {
			for j := i + 1; j < len(sortedDirs); j++ {
				if len(sortedDirs[i]) > len(sortedDirs[j]) ||
					(len(sortedDirs[i]) == len(sortedDirs[j]) && sortedDirs[i] > sortedDirs[j]) {
					sortedDirs[i], sortedDirs[j] = sortedDirs[j], sortedDirs[i]
				}
			}
		}
		for _, dir := range sortedDirs {
			fmt.Printf("       cd %s && terraform init -upgrade -backend=false\n", dir)
		}
	}
	fmt.Println()
	step++

	// Provider version note (verbose only)
	if cfg.verbose {
		fmt.Printf("  %d. Provider version set to %s (latest available)\n", step, targetVersion)
		fmt.Println()
		step++
	}

	fmt.Printf("  %d. Commit the changed files and open a pull/merge request. Read the\n", step)
	fmt.Println("     whole terraform plan before you apply: check that nothing is replaced")
	fmt.Println("     or destroyed, and that every change is one you expect.")

	return nil
}

// printVersionFetchFailure is called when the GitHub API is unreachable.
// Instead of silently writing a stale hardcoded version it tells the user
// to look up the latest version and update required_providers manually.
func printVersionFetchFailure(cfg config, diags hcl.Diagnostics) {
	fmt.Println()
	fmt.Println("⚠ Could not fetch the latest provider version from GitHub (network unavailable or rate-limited).")
	fmt.Println("  The provider version in required_providers has NOT been updated.")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println()

	step := 1

	// Always remind the user to review warnings — even when the version fetch
	// fails there may be actionable migration warnings that need attention.
	hasActionableWarnings := false
	for _, d := range diags {
		if d.Severity == hcl.DiagWarning {
			hasActionableWarnings = true
			break
		}
	}
	if hasActionableWarnings {
		fmt.Printf("  %d. Review the warnings above and complete any required manual actions\n", step)
		fmt.Println("     before running terraform plan/apply.")
		fmt.Println()
		step++
	}

	fmt.Printf("  %d. Find the latest Cloudflare provider version:\n", step)
	fmt.Println("       https://github.com/cloudflare/terraform-provider-cloudflare/releases")
	fmt.Println()
	step++

	fmt.Printf("  %d. Update required_providers in your main.tf (or equivalent):\n", step)
	fmt.Println("       cloudflare = {")
	fmt.Println("         source  = \"cloudflare/cloudflare\"")
	fmt.Println("         version = \"~> <latest>\"")
	fmt.Println("       }")
	fmt.Println()
	step++

	fmt.Printf("  %d. Regenerate the lock file locally:\n", step)
	fmt.Printf("       cd %s\n", cfg.configDir)
	fmt.Println("       terraform init -upgrade -backend=false")
	fmt.Println()
	step++

	fmt.Printf("  %d. Commit the changed files and open a pull/merge request. Read the\n", step)
	fmt.Println("     whole terraform plan before you apply: check that nothing is replaced")
	fmt.Println("     or destroyed, and that every change is one you expect.")
}

// findFilesWithPhaseOneComments returns a list of .tf files that contain
// commented-out resource blocks from a previous phase-1 run, identified by
// the phaseOneCommentPrefix marker.
func findFilesWithPhaseOneComments(cfg config) []string {
	files, err := findTerraformFilesWithRecursion(cfg.configDir, cfg.recursive, cfg.exclude)
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
	if cfg.skipPhaseCheck {
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
	files, err := findTerraformFilesWithRecursion(cfg.configDir, cfg.recursive, cfg.exclude)
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
			FilePath:      file,
			Diagnostics:   make(hcl.Diagnostics, 0),
			Metadata:      make(map[string]interface{}),
			SourceVersion: cfg.sourceVersion,
			TargetVersion: cfg.targetVersion,
			Resources:     cfg.resourcesToMigrate,
		}

		// Snapshot the normalized file bytes BEFORE calling BuildTokens on
		// any block. hclwrite.File.Bytes() normalizes whitespace around
		// inline comments as a side effect on the shared token list, while
		// BuildTokens preserves original spacing. Calling Bytes() first
		// ensures both representations use the same normalized form so the
		// subsequent strings.Replace can find the match.
		normalizedFile := string(parsed.Bytes())

		// Collect (block text, removed block) pairs for this file.
		type phaseOnePair struct {
			blockText    string // normalized bytes of the original resource block
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

		newContent := normalizedFile
		for _, pair := range pairs {
			commented := commentOutBlock(pair.blockText)
			// Build the removed {} block text
			scratch := hclwrite.NewEmptyFile()
			scratch.Body().AppendNewline()
			scratch.Body().AppendBlock(pair.removedBlock)
			removedText := string(hclwrite.Format(scratch.Bytes()))
			before := newContent
			newContent = strings.Replace(newContent, pair.blockText, commented+removedText, 1)
			if newContent == before {
				return fmt.Errorf(
					"phase-1 replacement failed for block in %s: "+
						"could not locate block text in normalized file content — "+
						"this is a tf-migrate bug, please file an issue at "+
						"https://github.com/cloudflare/tf-migrate/issues",
					filepath.Base(file),
				)
			}
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
	fmt.Println("     Run terraform plan and apply using the current (v4) provider.")
	fmt.Println("     Terraform will process the removed {} blocks")
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
	if cfg.verbose {
		fmt.Println("===========================================")
	}
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
			// Check if the previous non-empty input line was a commented block.
			// Look backwards in lines, not out: out has already been uncommented.
			lastNonEmpty := ""
			for j := i - 1; j >= 0; j-- {
				if strings.TrimSpace(lines[j]) != "" {
					lastNonEmpty = lines[j]
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

	files, aliases, err := findTerraformFilesWithAliases(cfg.configDir, cfg.recursive, cfg.exclude)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list .tf files: %w", err)
	}

	if len(files) == 0 {
		fmt.Printf("No .tf files found in %s\n", cfg.configDir)
		return nil, nil, nil
	}

	if cfg.verbose {
		fmt.Printf("Found %d configuration files to migrate\n", len(files))
	}

	// Store file paths for global postprocessing
	outputPaths := make([]string, 0, len(files))

	// Collect diagnostics from all files
	var allDiagnostics hcl.Diagnostics

	parsedConfigs := make(map[string]*hclwrite.File)
	for i, file := range files {
		if cfg.verbose {
			fmt.Printf("[%d/%d] Processing %s... ", i+1, len(files), filepath.Base(file))
		}
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
			FilePath:      file,
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
		if d := symlinkOutsideDirWarning(cfg, file); d != nil {
			allDiagnostics = append(allDiagnostics, d)
		}

		if ctx.CFGFile != nil {
			parsedConfigs[file] = ctx.CFGFile
		}

		outputPath, err := outputPathFor(cfg, file)
		if err != nil {
			return nil, allDiagnostics, err
		}

		if cfg.dryRun {
			if cfg.verbose {
				fmt.Println("(dry run)")
			}
			log.Debug("Would write file", "output", outputPath)
			outputPaths = append(outputPaths, outputPath)
			continue
		}

		if err := writeOutputFile(outputPath, transformed); err != nil {
			return nil, allDiagnostics, err
		}
		if cfg.verbose {
			fmt.Println("✓")
		}
		log.Debug("Migrated file", "output", outputPath)
		outputPaths = append(outputPaths, outputPath)

		// Other paths to the same file (symlinks). In place, writing through
		// one path already updated them. With a separate output directory,
		// each path gets its own copy of the migrated file.
		for _, alias := range aliases[file] {
			aliasOutput, err := outputPathFor(cfg, alias)
			if err != nil {
				return nil, allDiagnostics, err
			}
			if sameFile(aliasOutput, outputPath) {
				log.Debug("Skipped symlink to an already migrated file", "path", alias, "target", file)
				continue
			}
			if err := writeOutputFile(aliasOutput, transformed); err != nil {
				return nil, allDiagnostics, err
			}
			log.Debug("Migrated file", "output", aliasOutput, "same_as", outputPath)
			outputPaths = append(outputPaths, aliasOutput)
		}

	}

	// Apply global postprocessing for cross-file reference updates
	if !cfg.dryRun && len(outputPaths) > 0 {
		cfg.fileAliases = aliases
		postDiags, err := applyGlobalPostprocessing(log, cfg, outputPaths)
		allDiagnostics = append(allDiagnostics, postDiags...)
		if err != nil {
			return nil, allDiagnostics, fmt.Errorf("failed to apply global postprocessing: %w", err)
		}
		if cfg.recursive {
			allDiagnostics = hoistModuleImports(log, cfg, outputPaths, allDiagnostics)
		}
	}

	return parsedConfigs, allDiagnostics, nil
}

// outputPathFor returns where the migrated version of file is written,
// keeping the directory structure relative to the config dir when recursive.
func outputPathFor(cfg config, file string) (string, error) {
	if !cfg.recursive {
		return filepath.Join(cfg.outputDir, filepath.Base(file)), nil
	}
	relPath, err := filepath.Rel(cfg.configDir, file)
	if err != nil {
		return "", fmt.Errorf("failed to compute relative path: %w", err)
	}
	return filepath.Join(cfg.outputDir, relPath), nil
}

func writeOutputFile(outputPath string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputPath, err)
	}
	return nil
}

// sameFile reports whether a and b exist and are the same file.
func sameFile(a, b string) bool {
	ia, errA := os.Stat(a)
	ib, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(ia, ib)
}

func applyGlobalPostprocessing(log hclog.Logger, cfg config, outputPaths []string) (hcl.Diagnostics, error) {
	var diags hcl.Diagnostics

	// Collect resource renames, attribute renames, computed attribute mappings,
	// and invalid attribute reference detectors from all migrators.
	providers := getProviders(cfg.resourcesToMigrate...)
	migrators := providers.GetAllMigrators(cfg.sourceVersion, cfg.targetVersion, cfg.resourcesToMigrate...)

	// Map to store old type -> new type mappings
	renames := make(map[string]string)
	// Slice to store attribute renames
	var attributeRenames []transform.AttributeRename
	// Slice to store computed attribute mappings (for when both resource type AND attribute change)
	var computedAttrMappings []transform.ComputedAttributeMapping
	// Slice to store invalid attribute references to warn about
	var invalidAttrRefs []transform.InvalidAttributeReference
	// Slice to store per-resource renames (one v4 type splitting into several v5 types)
	var instanceRenames []transform.InstanceRename

	for _, migrator := range migrators {
		// Check if this migrator implements ResourceRenamer interface
		if renamer, ok := migrator.(transform.ResourceRenamer); ok {
			oldTypes, newType := renamer.GetResourceRename()
			if len(oldTypes) > 0 && newType != "" {
				for _, oldType := range oldTypes {
					if oldType != newType {
						renames[oldType] = newType
						log.Debug("Collected resource rename", "old", oldType, "new", newType)
					} else {
						log.Debug("Resource type unchanged", "type", oldType)
					}
				}
			} else {
				log.Debug("Migrator implements ResourceRenamer but returned empty type names",
					"oldTypes", oldTypes, "newType", newType)
			}
		} else {
			log.Debug("Migrator does not implement ResourceRenamer interface - cross-file references may not be updated",
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

		// Check if this migrator implements ComputedAttributeMapper interface
		if computedAttrMapper, ok := migrator.(transform.ComputedAttributeMapper); ok {
			mappings := computedAttrMapper.GetComputedAttributeMappings()
			if len(mappings) > 0 {
				computedAttrMappings = append(computedAttrMappings, mappings...)
				for _, m := range mappings {
					log.Debug("Collected computed attribute mapping",
						"old_type", m.OldResourceType,
						"old_attr", m.OldAttribute,
						"new_type", m.NewResourceType,
						"new_attr", m.NewAttribute)
				}
			}
		}

		// Check if this migrator implements InstanceRenamer interface
		if instanceRenamer, ok := migrator.(transform.InstanceRenamer); ok {
			for _, r := range instanceRenamer.GetInstanceRenames() {
				instanceRenames = append(instanceRenames, r)
				log.Debug("Collected instance rename",
					"old", r.OldType+"."+r.Name, "new", r.NewType+"."+r.Name, "dir", r.Dir)
			}
		}

		// Check if this migrator implements InvalidAttributeReferenceDetector interface
		if detector, ok := migrator.(transform.InvalidAttributeReferenceDetector); ok {
			refs := detector.GetInvalidAttributeReferences()
			if len(refs) > 0 {
				invalidAttrRefs = append(invalidAttrRefs, refs...)
				for _, r := range refs {
					log.Debug("Collected invalid attribute reference detector",
						"resource_type", r.ResourceType,
						"attribute", r.Attribute)
				}
			}
		}
	}

	// If no renames or detectors found, skip global postprocessing
	if len(renames) == 0 && len(attributeRenames) == 0 && len(computedAttrMappings) == 0 && len(invalidAttrRefs) == 0 && len(instanceRenames) == 0 {
		log.Debug("No renames found, skipping global postprocessing")
		return diags, nil
	}

	// Track resources that were intentionally converted to removed {} blocks.
	// References to these addresses must NOT be rewritten to renamed types.
	removedRefsByType, err := collectRemovedRefsByType(outputPaths)
	if err != nil {
		log.Warn("Failed to collect removed block references for postprocessing", "error", err)
	}

	// Which resources are declared with count or for_each, so indexed and
	// iterated references to them can be told apart from attribute access.
	iterable := collectIterableResources(log, outputPaths, cfg.fileAliases)
	instanceRenames = expandInstanceRenamesToAliases(instanceRenames, cfg.fileAliases)

	// Track which renames were actually applied (content changed)
	appliedRenames := make(map[string]string)
	appliedAttrRenames := make(map[string]transform.AttributeRename)               // key: ResourceType.OldAttribute
	appliedComputedMappings := make(map[string]transform.ComputedAttributeMapping) // key: OldResourceType.OldAttribute

	if cfg.verbose {
		fmt.Printf("\nApplying cross-file reference updates across %d files...\n", len(outputPaths))
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

		// Apply per-resource renames before anything else: they only touch the
		// exact addresses a migrator reported, in files of the declaring module.
		if len(instanceRenames) > 0 {
			newContent := applyInstanceRenames(contentStr, instanceRenames, inputDirForOutput(cfg, outputPath))
			if newContent != contentStr {
				modified = true
				contentStr = newContent
				log.Debug("Updated per-resource references", "file", filepath.Base(outputPath))
			}
		}

		// Apply computed attribute mappings FIRST (for when both resource type AND attribute name change)
		// Example: cloudflare_tunnel.<name>.cname → cloudflare_zero_trust_tunnel_cloudflared.<name>.name
		// This must happen BEFORE resource type renames so the pattern matches the old resource type
		for _, mapping := range computedAttrMappings {
			// Build regex pattern to match old resource type + old attribute.
			// Use regexp.QuoteMeta so resource types containing "." (e.g.
			// "data.cloudflare_zone") are treated as literals, not wildcards.
			// The trailing \b ensures ".zone" does not match the prefix of
			// ".zone_id" (which would produce the nonsense attribute "name_id").
			// Pattern example: cloudflare_tunnel\.([a-zA-Z0-9_-]+)\.cname\b
			pattern := regexp.QuoteMeta(mapping.OldResourceType) + `\.([a-zA-Z0-9_-]+)\.` + regexp.QuoteMeta(mapping.OldAttribute) + `\b`
			replacement := mapping.NewResourceType + ".$1." + mapping.NewAttribute
			newContent := regexReplaceSkippingMovedBlocks(contentStr, pattern, replacement)

			if newContent != contentStr {
				modified = true
				contentStr = newContent
				key := mapping.OldResourceType + "." + mapping.OldAttribute
				appliedComputedMappings[key] = mapping
				log.Debug("Updated computed attribute references",
					"file", filepath.Base(outputPath),
					"old_type", mapping.OldResourceType,
					"old_attr", mapping.OldAttribute,
					"new_type", mapping.NewResourceType,
					"new_attr", mapping.NewAttribute)
			}
		}

		// Apply all resource type renames, but skip content within moved blocks
		for oldType, newType := range renames {
			newContent := replaceResourceTypeRefsSkippingMovedBlocks(contentStr, oldType, newType, removedRefsByType[oldType])
			if newContent != contentStr {
				modified = true
				contentStr = newContent
				appliedRenames[oldType] = newType
				log.Debug("Updated references", "file", filepath.Base(outputPath), "old", oldType, "new", newType)
			}
		}

		// Apply all attribute renames, but skip content within moved blocks.
		// data.cloudflare_zones.<name>.zones -> data.cloudflare_zones.<name>.result,
		// including indexed, splat and for/for_each iteration forms.
		if len(attributeRenames) > 0 {
			newContent, applied := renameAttributeReferences(log, contentStr, outputPath, attributeRenames, iterable)
			if newContent != contentStr {
				modified = true
				contentStr = newContent
				for _, rename := range applied {
					appliedAttrRenames[rename.ResourceType+"."+rename.OldAttribute] = rename
					log.Debug("Updated attribute references",
						"file", filepath.Base(outputPath),
						"resource_type", rename.ResourceType,
						"old_attr", rename.OldAttribute,
						"new_attr", rename.NewAttribute)
				}
			}
		}

		// Write back if modified
		if modified {
			if err := os.WriteFile(outputPath, []byte(contentStr), 0644); err != nil {
				return diags, fmt.Errorf("failed to write updated file %s: %w", outputPath, err)
			}
		}
	}

	// Scan all output files for invalid attribute references and emit DiagWarnings.
	// This runs after all rewrites so that already-fixed references (e.g. secret →
	// tunnel_secret) don't produce false positives.
	if len(invalidAttrRefs) > 0 {
		invalidAttrDiags := scanForInvalidAttributeReferences(log, outputPaths, invalidAttrRefs, cfg.fileAliases)
		diags = append(diags, invalidAttrDiags...)
	}

	if cfg.verbose {
		totalApplied := len(appliedRenames) + len(appliedAttrRenames) + len(appliedComputedMappings)
		if totalApplied > 0 {
			fmt.Printf("✓ Updated cross-file references (%d of %d rules applied)\n",
				totalApplied, len(renames)+len(attributeRenames)+len(computedAttrMappings))

			if len(appliedRenames) > 0 {
				fmt.Println("\n  Resource type renames applied:")
				for oldType, newType := range appliedRenames {
					fmt.Printf("    %s → %s\n", oldType, newType)
				}
			}

			if len(appliedAttrRenames) > 0 {
				fmt.Println("\n  Attribute renames applied:")
				for _, rename := range appliedAttrRenames {
					fmt.Printf("    %s.*.%s → %s.*.%s\n",
						rename.ResourceType, rename.OldAttribute,
						rename.ResourceType, rename.NewAttribute)
				}
			}

			if len(appliedComputedMappings) > 0 {
				fmt.Println("\n  Computed attribute mappings applied:")
				for _, mapping := range appliedComputedMappings {
					fmt.Printf("    %s.*.%s → %s.*.%s\n",
						mapping.OldResourceType, mapping.OldAttribute,
						mapping.NewResourceType, mapping.NewAttribute)
				}
			}
		} else {
			fmt.Println("✓ No cross-file references needed updating")
		}
	}
	return diags, nil
}

// scanForInvalidAttributeReferences scans output files for cross-file references
// to known-invalid attributes and returns a DiagWarning for each match found.
func scanForInvalidAttributeReferences(log hclog.Logger, outputPaths []string, refs []transform.InvalidAttributeReference, aliases map[string][]string) hcl.Diagnostics {
	var diags hcl.Diagnostics
	if len(refs) == 0 {
		return diags
	}

	types := make([]string, 0, len(refs))
	for _, ref := range refs {
		types = append(types, ref.ResourceType)
	}
	iterable := collectIterableResources(log, outputPaths, aliases)

	for _, outputPath := range outputPaths {
		content, err := os.ReadFile(outputPath)
		if err != nil {
			log.Warn("Failed to read file for invalid attribute scan", "file", outputPath, "error", err)
			continue
		}
		body, ok := parseSyntaxBody(log, content, outputPath)
		if !ok {
			continue
		}
		found := tfhcl.FindInstanceAttributeReferences(body, content, types, iterable(outputPath))
		for _, f := range found {
			for _, ref := range refs {
				if f.Type != ref.ResourceType || f.Attribute != ref.Attribute {
					continue
				}
				r := f.Range
				r.Filename = filepath.Base(outputPath)
				match := string(r.SliceBytes(content))
				summary := "Unknown attribute reference: " + match
				if ref.ReviewOnly {
					summary = "Review changed attribute reference: " + match
				}
				log.Debug("Found invalid attribute reference",
					"file", filepath.Base(outputPath),
					"match", match)
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  summary,
					Detail:   fmt.Sprintf("In %s\n\n  %s", filepath.Base(outputPath), ref.Suggestion),
					Subject:  &r,
				})
			}
		}
	}

	return diags
}

// parseSyntaxBody parses content for read-only AST scans.
func parseSyntaxBody(log hclog.Logger, content []byte, path string) (*hclsyntax.Body, bool) {
	parsed, parseDiags := hclsyntax.ParseConfig(content, filepath.Base(path), hcl.InitialPos)
	if parseDiags.HasErrors() || parsed == nil {
		log.Warn("Failed to parse file for reference scan", "file", path, "error", parseDiags.Error())
		return nil, false
	}
	body, ok := parsed.Body.(*hclsyntax.Body)
	return body, ok && body != nil
}

// collectIterableResources returns, for an output file, a function reporting
// whether <type>.<name> may be indexed or iterated: false only when the
// resource is declared in the same directory without count or for_each.
// Undeclared resources are assumed iterable. A file migrated in place and
// reached through symlinks (aliases) declares its resources in each of
// their directories too.
func collectIterableResources(log hclog.Logger, outputPaths []string, aliases map[string][]string) func(outputPath string) func(resourceType, name string) bool {
	singles := make(map[string]map[string]bool) // dir -> "type.name" -> declared without count/for_each
	for _, outputPath := range outputPaths {
		content, err := os.ReadFile(outputPath)
		if err != nil {
			continue
		}
		body, ok := parseSyntaxBody(log, content, outputPath)
		if !ok {
			continue
		}
		dirs := []string{filepath.Dir(outputPath)}
		for _, alias := range aliases[outputPath] {
			dirs = append(dirs, filepath.Dir(alias))
		}
		for _, block := range body.Blocks {
			if len(block.Labels) != 2 || (block.Type != "resource" && block.Type != "data") {
				continue
			}
			addr := block.Labels[0] + "." + block.Labels[1]
			if block.Type == "data" {
				addr = "data." + addr
			}
			_, hasCount := block.Body.Attributes["count"]
			_, hasForEach := block.Body.Attributes["for_each"]
			for _, dir := range dirs {
				if singles[dir] == nil {
					singles[dir] = make(map[string]bool)
				}
				singles[dir][addr] = !hasCount && !hasForEach
			}
		}
	}
	return func(outputPath string) func(string, string) bool {
		inDir := singles[filepath.Dir(outputPath)]
		return func(resourceType, name string) bool {
			return !inDir[resourceType+"."+name]
		}
	}
}

// renameAttributeReferences rewrites references to renamed attributes in
// content, skipping moved and removed blocks. It returns the renames that
// changed something. If content does not parse, it falls back to a regex
// that only handles <type>.<name>.<attribute>.
func renameAttributeReferences(log hclog.Logger, content, outputPath string, renames []transform.AttributeRename, iterable func(string) func(string, string) bool) (string, []transform.AttributeRename) {
	var applied []transform.AttributeRename

	body, ok := parseSyntaxBody(log, []byte(content), outputPath)
	if !ok {
		for _, rename := range renames {
			pattern := regexp.QuoteMeta(rename.ResourceType) + `\.([a-zA-Z0-9_-]+)\.` + regexp.QuoteMeta(rename.OldAttribute) + `\b`
			replacement := rename.ResourceType + ".$1." + rename.NewAttribute
			if newContent := regexReplaceSkippingMovedBlocks(content, pattern, replacement); newContent != content {
				content = newContent
				applied = append(applied, rename)
			}
		}
		return content, applied
	}

	types := make([]string, 0, len(renames))
	for _, rename := range renames {
		types = append(types, rename.ResourceType)
	}
	found := tfhcl.FindInstanceAttributeReferences(body, []byte(content), types, iterable(outputPath))
	protected := findProtectedBlockRanges(content)

	type edit struct {
		start, end int
		text       string
	}
	var edits []edit
	used := make(map[int]bool)
	for _, f := range found {
		if inRanges(protected, f.NameRange.Start.Byte) {
			continue
		}
		for i, rename := range renames {
			if f.Type == rename.ResourceType && f.Attribute == rename.OldAttribute {
				edits = append(edits, edit{f.NameRange.Start.Byte, f.NameRange.End.Byte, rename.NewAttribute})
				used[i] = true
				break
			}
		}
	}
	// found is sorted by position; apply from the end so offsets stay valid.
	for i := len(edits) - 1; i >= 0; i-- {
		e := edits[i]
		content = content[:e.start] + e.text + content[e.end:]
	}
	for i, rename := range renames {
		if used[i] {
			applied = append(applied, rename)
		}
	}
	return content, applied
}

func inRanges(ranges [][2]int, pos int) bool {
	for _, r := range ranges {
		if pos >= r[0] && pos < r[1] {
			return true
		}
	}
	return false
}

// collectRemovedRefsByType scans transformed files and returns addresses found in
// removed { from = <type>.<name> } blocks as a map[type]set(name).
func collectRemovedRefsByType(outputPaths []string) (map[string]map[string]struct{}, error) {
	removed := make(map[string]map[string]struct{})

	for _, outputPath := range outputPaths {
		content, err := os.ReadFile(outputPath)
		if err != nil {
			return nil, fmt.Errorf("failed reading %s: %w", outputPath, err)
		}

		parsed, diags := hclwrite.ParseConfig(content, filepath.Base(outputPath), hcl.InitialPos)
		if diags.HasErrors() {
			continue
		}

		for _, block := range parsed.Body().Blocks() {
			if block.Type() != "removed" {
				continue
			}
			fromAttr := block.Body().GetAttribute("from")
			if fromAttr == nil {
				continue
			}

			from := extractTraversalString(fromAttr)
			parts := strings.SplitN(from, ".", 2)
			if len(parts) != 2 {
				continue
			}

			resourceType, resourceName := parts[0], parts[1]
			if _, ok := removed[resourceType]; !ok {
				removed[resourceType] = make(map[string]struct{})
			}
			removed[resourceType][resourceName] = struct{}{}
		}
	}

	return removed, nil
}

// replaceResourceTypeRefsSkippingMovedBlocks rewrites <oldType>.<name> to
// <newType>.<name>, except when <name> is in excludedNames. Replacements are
// skipped inside moved/removed blocks.
func replaceResourceTypeRefsSkippingMovedBlocks(content, oldType, newType string, excludedNames map[string]struct{}) string {
	pattern := regexp.MustCompile(regexp.QuoteMeta(oldType) + `\.([a-zA-Z0-9_-]+)`)
	return regexReplaceFuncSkippingMovedBlocks(content, pattern, func(match string) string {
		sub := pattern.FindStringSubmatch(match)
		if len(sub) != 2 {
			return match
		}
		name := sub[1]
		if excludedNames != nil {
			if _, skip := excludedNames[name]; skip {
				return match
			}
		}
		return newType + "." + name
	})
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

// expandInstanceRenamesToAliases adds, for each rename whose declaring file is
// also reached through symlinks, a copy scoped to each symlink's directory.
// Each of those directories is a module that declares the resource, so its
// references must be renamed too.
func expandInstanceRenamesToAliases(renames []transform.InstanceRename, aliases map[string][]string) []transform.InstanceRename {
	if len(aliases) == 0 {
		return renames
	}
	expanded := renames
	for _, r := range renames {
		if r.File == "" || r.Dir == "" {
			continue
		}
		for _, alias := range aliases[r.File] {
			copied := r
			copied.Dir = filepath.Clean(filepath.Dir(alias))
			if !sameDir(copied.Dir, r.Dir) {
				expanded = append(expanded, copied)
			}
		}
	}
	return expanded
}

// inputDirForOutput returns the directory of the input file that outputPath was
// written from, so it can be compared with InstanceRename.Dir.
func inputDirForOutput(cfg config, outputPath string) string {
	if cfg.recursive {
		if rel, err := filepath.Rel(cfg.outputDir, filepath.Dir(outputPath)); err == nil {
			return filepath.Clean(filepath.Join(cfg.configDir, rel))
		}
	}
	return filepath.Clean(cfg.configDir)
}

// sameDir reports whether two directory paths refer to the same directory,
// tolerating one being relative and the other absolute.
func sameDir(a, b string) bool {
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	return errA == nil && errB == nil && absA == absB
}

// applyInstanceRenames rewrites references to the resources in renames whose
// Dir matches dir (or is empty), e.g. old_type.name.id -> new_type.name.id.
// Other resources of the same old type, data sources, and moved/removed blocks
// are left unchanged.
func applyInstanceRenames(content string, renames []transform.InstanceRename, dir string) string {
	byType := make(map[string]map[string]string) // old type -> name -> new type
	for _, r := range renames {
		if r.Dir != "" && !sameDir(r.Dir, dir) {
			continue
		}
		if byType[r.OldType] == nil {
			byType[r.OldType] = make(map[string]string)
		}
		byType[r.OldType][r.Name] = r.NewType
	}

	for oldType, names := range byType {
		pattern := regexp.MustCompile(`(data\.)?\b` + regexp.QuoteMeta(oldType) + `\.([a-zA-Z0-9_-]+)`)
		content = regexReplaceFuncSkippingMovedBlocks(content, pattern, func(match string) string {
			sub := pattern.FindStringSubmatch(match)
			if len(sub) != 3 || sub[1] != "" {
				return match
			}
			newType, ok := names[sub[2]]
			if !ok {
				return match
			}
			return newType + "." + sub[2]
		})
	}
	return content
}

// regexReplaceFuncSkippingMovedBlocks replaces regex matches in content with a
// callback, skipping moved and removed blocks.
func regexReplaceFuncSkippingMovedBlocks(content string, re *regexp.Regexp, replacer func(string) string) string {
	protected := findProtectedBlockRanges(content)
	if len(protected) == 0 {
		return re.ReplaceAllStringFunc(content, replacer)
	}

	var result strings.Builder
	lastEnd := 0
	for _, r := range protected {
		result.WriteString(re.ReplaceAllStringFunc(content[lastEnd:r[0]], replacer))
		result.WriteString(content[r[0]:r[1]])
		lastEnd = r[1]
	}
	result.WriteString(re.ReplaceAllStringFunc(content[lastEnd:], replacer))
	return result.String()
}

func findTerraformFiles(dir string) ([]string, error) {
	return findTerraformFilesWithRecursion(dir, false, nil)
}

// findTerraformFilesWithRecursion returns the .tf files under dir, listing
// each file once even when it is reachable through several paths. See
// findTerraformFilesWithAliases.
func findTerraformFilesWithRecursion(dir string, recursive bool, exclude []string) ([]string, error) {
	files, _, err := findTerraformFilesWithAliases(dir, recursive, exclude)
	return files, err
}

// findTerraformFilesWithAliases returns the .tf files under dir with each
// real file listed once, plus, for each listed path, the other paths
// (symlinks) that resolve to the same file.
//
// Modules often share a file by symlinking it into each module directory.
// Migrating every path would transform the same file several times, and the
// transforms are not idempotent (e.g. the provider version constraint was
// added once per path). The listed path is the real file when it is inside
// dir, otherwise the first symlink found.
func findTerraformFilesWithAliases(dir string, recursive bool, exclude []string) ([]string, map[string][]string, error) {
	all, err := listTerraformFiles(dir, recursive, exclude)
	if err != nil {
		return nil, nil, err
	}

	realPaths := make([]string, len(all))
	canonical := make(map[string]string) // real path -> listed path
	for i, path := range all {
		realPaths[i] = resolvedPath(path)
		first, seen := canonical[realPaths[i]]
		if !seen || (isSymlink(first) && !isSymlink(path)) {
			canonical[realPaths[i]] = path
		}
	}

	var files []string
	aliases := make(map[string][]string)
	for i, path := range all {
		listed := canonical[realPaths[i]]
		if path == listed {
			files = append(files, path)
		} else {
			aliases[listed] = append(aliases[listed], path)
		}
	}
	return files, aliases, nil
}

// resolvedPath returns the absolute path of the file path refers to after
// following symlinks, or path itself if it cannot be resolved (for example a
// dangling symlink, which fails later when it is read).
func resolvedPath(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	if abs, err := filepath.Abs(resolved); err == nil {
		return abs
	}
	return resolved
}

// symlinkOutsideDirWarning warns when file is a symlink to a file outside the
// config dir and is migrated in place: the shared file is rewritten for every
// configuration that uses it, including ones still on v4.
func symlinkOutsideDirWarning(cfg config, file string) *hcl.Diagnostic {
	if !isSymlink(file) || !sameDirPath(cfg.outputDir, cfg.configDir) {
		return nil
	}
	target := resolvedPath(file)
	root := resolvedPath(cfg.configDir)
	if rel, err := filepath.Rel(root, target); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil
	}
	return &hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  fmt.Sprintf("Migrated a shared file outside the config directory: %s", target),
		Detail: fmt.Sprintf("%s is a symlink to %s, which is outside %s. It was migrated in place, so "+
			"every configuration that links to it now sees the v5 version. Check that none of them "+
			"are still on the v4 provider, or copy the file into this configuration instead of linking it.",
			file, target, root),
	}
}

// sameDirPath reports whether a and b name the same directory. An empty a
// means the default, the config dir.
func sameDirPath(a, b string) bool {
	return a == "" || resolvedPath(a) == resolvedPath(b)
}

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func listTerraformFiles(dir string, recursive bool, exclude []string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() && recursive {
			// Check if this directory should be excluded
			if shouldExclude(entry.Name(), exclude) {
				continue
			}
			// Recursively search subdirectories
			subFiles, err := listTerraformFiles(path, recursive, exclude)
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

// shouldExclude checks if a directory name matches any of the exclusion patterns
func shouldExclude(dirName string, exclude []string) bool {
	for _, pattern := range exclude {
		// Support both exact match and wildcard patterns
		if matched, _ := filepath.Match(pattern, dirName); matched {
			return true
		}
		// Also check for exact match
		if pattern == dirName {
			return true
		}
	}
	return false
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

// consolidatedDiagnostic groups multiple diagnostics with the same summary
type consolidatedDiagnostic struct {
	Summary  string
	Severity hcl.DiagnosticSeverity
	Details  []string // Multiple detail messages for resources with same issue
	Sample   string   // Full detail from first occurrence (for reference)
}

// consolidateDiagnostics groups diagnostics by summary to avoid repetition
func consolidateDiagnostics(diags hcl.Diagnostics) []consolidatedDiagnostic {
	groups := make(map[string]*consolidatedDiagnostic)

	for _, diag := range diags {
		key := fmt.Sprintf("%s:%d", diag.Summary, diag.Severity)

		if existing, ok := groups[key]; ok {
			// Extract resource name from detail (first line typically contains it)
			existing.Details = append(existing.Details, extractResourceName(diag.Detail))
		} else {
			groups[key] = &consolidatedDiagnostic{
				Summary:  diag.Summary,
				Severity: diag.Severity,
				Details:  []string{extractResourceName(diag.Detail)},
				Sample:   diag.Detail,
			}
		}
	}

	// Convert map to slice
	result := make([]consolidatedDiagnostic, 0, len(groups))
	for _, g := range groups {
		result = append(result, *g)
	}
	return result
}

// extractResourceName extracts the resource identifier from diagnostic detail.
// Returns the entire first line so that context such as file paths is preserved
// in the consolidated bullet list.
// Expected formats:
//   - "cloudflare_type.name  (/path/to/file.tf)"  — new consolidated format
//   - "Resource cloudflare_type.name has ..."      — legacy format
func extractResourceName(detail string) string {
	lines := strings.Split(detail, "\n")
	if len(lines) == 0 {
		return detail
	}

	firstLine := strings.TrimSpace(lines[0])

	// Legacy format: "Resource cloudflare_type.name ..."
	if strings.HasPrefix(firstLine, "Resource ") {
		parts := strings.SplitN(firstLine, " ", 3)
		if len(parts) >= 2 {
			return parts[1]
		}
	}

	// New consolidated format: "cloudflare_type.name  (/path/to/file)"
	// Return the whole first line so the file path is visible in the bullet list.
	if strings.Contains(firstLine, "cloudflare_") {
		return firstLine
	}

	return firstLine
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
		// Even when nothing actionable is shown, hint about suppressed info messages
		if !cfg.verbose && infos > 0 {
			fmt.Printf("\nℹ %d informational message(s) suppressed (use --verbose to see details)\n", infos)
		}
		return
	}

	// ANSI color codes for diagnostic output
	const (
		diagColorRed    = "\033[0;31m"
		diagColorYellow = "\033[1;33m"
		diagColorCyan   = "\033[0;36m"
		diagColorReset  = "\033[0m"
	)

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
	if infos > 0 {
		if cfg.verbose {
			parts = append(parts, fmt.Sprintf("%d info message(s)", infos))
		} else {
			parts = append(parts, fmt.Sprintf("%d info message(s) suppressed (use --verbose to see)", infos))
		}
	}
	headerColor := diagColorYellow
	if errors > 0 {
		headerColor = diagColorRed
	}
	fmt.Printf("%sMigration diagnostics: %s%s\n", headerColor, strings.Join(parts, ", "), diagColorReset)
	fmt.Println(strings.Repeat("─", 70))

	// Consolidate diagnostics by summary
	consolidated := consolidateDiagnostics(filtered)

	for i, cd := range consolidated {
		if i > 0 {
			fmt.Println()
		}

		// Print severity icon with color
		var icon, color string
		switch cd.Severity {
		case hcl.DiagError:
			icon = "✗"
			color = diagColorRed
		case hcl.DiagWarning:
			icon = "⚠"
			color = diagColorYellow
		default:
			icon = "ℹ"
			color = diagColorCyan
		}

		// If there are multiple resources with the same issue, consolidate them
		if len(cd.Details) > 1 {
			fmt.Printf("\n%s%s %s%s (%d resources)\n", color, icon, cd.Summary, diagColorReset, len(cd.Details))
			fmt.Println()
			// List affected resources
			for _, detail := range cd.Details {
				fmt.Printf("  - %s\n", detail)
			}
			// Show the full instructions once
			if cd.Sample != "" {
				// Extract the instructions part (after the first line/resource description)
				lines := strings.Split(cd.Sample, "\n")
				if len(lines) > 1 {
					fmt.Println()
					// Find where the actual instructions start (skip resource-specific lines)
					for j := 1; j < len(lines); j++ {
						line := lines[j]
						// Skip empty lines at the start
						if strings.TrimSpace(line) == "" && j < len(lines)-1 {
							continue
						}
						// Print remaining lines
						fmt.Println(strings.Join(lines[j:], "\n"))
						break
					}
				}
			}
		} else {
			// Single resource - show full detail
			fmt.Printf("\n%s%s %s%s\n", color, icon, cd.Summary, diagColorReset)
			if cd.Sample != "" {
				fmt.Printf("\n%s\n", cd.Sample)
			}
		}
	}
	fmt.Println(strings.Repeat("─", 70))
}
