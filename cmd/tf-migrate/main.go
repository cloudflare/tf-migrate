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
	resourcesToMigrate    []string
	sourceVersion         string
	targetVersion         string
	targetProviderVersion string // explicit provider version to write into required_providers (overrides API fetch)
	dryRun                bool
	backup                bool
	recursive             bool
	logLevel              string
	skipPhaseCheck        bool // skip phased migration prompt and run full migration directly (for CI/e2e)
	skipVersionCheck      bool // skip minimum provider version check (for testing)

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

	files, err := findTerraformFilesWithRecursion(cfg.configDir, cfg.recursive)
	if err != nil {
		return err
	}

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
				newAttrStr := regexp.MustCompile(`version\s*=\s*"[^"]*"`).
					ReplaceAllString(attrStr, `version = "`+targetVersion+`"`)
				if newAttrStr == attrStr {
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
		_ = updated
	}

	fmt.Println()
	fmt.Printf("Provider version updated to %s.\n", targetVersion)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println()

	step := 1

	// Remind users to apply with v4 first if they haven't already.
	// This normalizes state for resources like cloudflare_ruleset where the v5
	// provider's UpgradeResourceState has edge cases with older v4 state formats.
	fmt.Printf("  %d. BEFORE upgrading: ensure you have run 'terraform apply' with\n", step)
	fmt.Printf("     provider v%s on your current v4 config to normalize state.\n", minimumProviderVersion)
	fmt.Println("     This prevents state deserialization errors during the v5 upgrade")
	fmt.Println("     (especially for resources like cloudflare_ruleset).")
	fmt.Println()
	step++

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

	fmt.Printf("  %d. Regenerate the lock file locally:\n", step)
	fmt.Printf("       cd %s\n", cfg.configDir)
	fmt.Println("       terraform init -upgrade -backend=false")
	fmt.Println()
	step++

	// Note about provider version and MoveState support
	fmt.Printf("  %d. Verify the provider version supports MoveState for your resources.\n", step)
	fmt.Printf("     tf-migrate set the version to %s. If you see errors about\n", targetVersion)
	fmt.Println("     'Missing resource schema' or 'no schema available' during plan/apply,")
	fmt.Println("     try upgrading to the latest beta release:")
	fmt.Println("       https://github.com/cloudflare/terraform-provider-cloudflare/releases")
	fmt.Println()
	step++

	fmt.Printf("  %d. Commit and push all changed files:\n", step)
	fmt.Println("       git add .")
	fmt.Printf("       git commit -m \"chore: migrate Cloudflare provider to v%s\"\n", targetVersion)
	fmt.Println("       git push")

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

	fmt.Printf("  %d. Commit and push all changed files:\n", step)
	fmt.Println("       git add .")
	fmt.Println("       git commit -m \"chore: migrate Cloudflare provider to v<latest>\"")
	fmt.Println("       git push")
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
			FilePath:      file,
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

		// Use the parsed (normalized) file bytes as the base — both blockText
		// from BuildTokens and the file bytes come from the same hclwrite
		// tokenizer, so replacements match exactly.
		newContent := string(parsed.Bytes())
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

	fmt.Printf("Found %d configuration files to migrate\n", len(files))

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
			if cfg.verbose {
				fmt.Println("(dry run)")
			}
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
		if cfg.verbose {
			fmt.Println("✓")
		}
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
	}

	// If no renames found, skip global postprocessing
	if len(renames) == 0 && len(attributeRenames) == 0 {
		log.Debug("No renames found, skipping global postprocessing")
		return nil
	}

	if cfg.verbose {
		totalUpdates := len(renames) + len(attributeRenames)
		fmt.Printf("\nApplying cross-file reference updates (%d updates across %d files)...\n", totalUpdates, len(outputPaths))

		if len(renames) > 0 {
			fmt.Println("\nResource type renames:")
			for oldType, newType := range renames {
				fmt.Printf("  %s → %s\n", oldType, newType)
			}
		}

		if len(attributeRenames) > 0 {
			fmt.Println("\nAttribute renames:")
			for _, rename := range attributeRenames {
				fmt.Printf("  %s.*.%s → %s.*.%s\n", rename.ResourceType, rename.OldAttribute, rename.ResourceType, rename.NewAttribute)
			}
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

	if cfg.verbose {
		fmt.Printf("✓ Updated cross-file references (%d rule(s) applied)\n", len(renames)+len(attributeRenames))
	}
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
		// Even when nothing actionable is shown, hint about suppressed info messages
		if !cfg.verbose && infos > 0 {
			fmt.Printf("\nℹ %d informational message(s) suppressed (use --verbose to see details)\n", infos)
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
	if infos > 0 {
		if cfg.verbose {
			parts = append(parts, fmt.Sprintf("%d info message(s)", infos))
		} else {
			parts = append(parts, fmt.Sprintf("%d info message(s) suppressed (use --verbose to see)", infos))
		}
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
