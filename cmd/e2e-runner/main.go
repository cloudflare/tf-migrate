package main

import (
	"fmt"
	"os"
	"strings"

	e2e "github.com/cloudflare/tf-migrate/internal/e2e-runner"
	"github.com/cloudflare/tf-migrate/internal/registry"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "e2e",
	Short: "E2E testing tool for tf-migrate",
	Long:  `End-to-end testing framework for validating v4 to v5 migrations`,
}

var initCmd = &cobra.Command{
	Use:          "init",
	Short:        "Initialize test resources",
	Long:         `Syncs resource files from integration testdata to e2e/v4`,
	SilenceUsage: true, // Don't show usage on error
	RunE: func(cmd *cobra.Command, args []string) error {
		resources, _ := cmd.Flags().GetString("resources")
		phase, _ := cmd.Flags().GetString("phase")
		if phase != "" {
			if resources != "" {
				return fmt.Errorf("--phase and --resources are mutually exclusive; use one or the other")
			}
			phaseResources, err := e2e.ResolvePhases(phase)
			if err != nil {
				return err
			}
			resources = strings.Join(phaseResources, ",")
		}
		return e2e.RunInit(resources)
	},
}

var migrateCmd = &cobra.Command{
	Use:          "migrate",
	Short:        "Run migration from v4 to v5",
	Long:         `Copies v4/ to migrated-v4_to_v5/ and runs migration`,
	SilenceUsage: true, // Don't show usage on error
	RunE: func(cmd *cobra.Command, args []string) error {
		resources, _ := cmd.Flags().GetString("resources")
		phase, _ := cmd.Flags().GetString("phase")
		if phase != "" {
			if resources != "" {
				return fmt.Errorf("--phase and --resources are mutually exclusive; use one or the other")
			}
			phaseResources, err := e2e.ResolvePhases(phase)
			if err != nil {
				return err
			}
			resources = strings.Join(phaseResources, ",")
		}
		return e2e.RunMigrate(resources, false)
	},
}

var runCmd = &cobra.Command{
	Use:          "run",
	Short:        "Run full e2e test suite",
	Long:         `Runs the complete e2e test: init, v4 apply, migrate, v5 apply, drift check`,
	SilenceUsage: true, // Don't show usage on error
	RunE: func(cmd *cobra.Command, args []string) error {
		parallelism, _ := cmd.Flags().GetInt("parallelism")

		cfg := &e2e.RunConfig{
			SkipV4Test:        cmd.Flag("skip-v4-test").Changed,
			ApplyExemptions:   cmd.Flag("apply-exemptions").Changed,
			NoRefreshSnapshot: cmd.Flag("no-refresh-snapshot").Changed,
			Parallelism:       parallelism,
			Resources:         cmd.Flag("resources").Value.String(),
			Exclude:           cmd.Flag("exclude").Value.String(),
			Phase:             cmd.Flag("phase").Value.String(),
			ProviderPath:      cmd.Flag("provider").Value.String(),
		}
		return e2e.RunE2ETests(cfg)
	},
}

var bootstrapCmd = &cobra.Command{
	Use:          "bootstrap",
	Short:        "Migrate local state to R2 remote backend",
	Long:         `One-time operation to migrate local terraform.tfstate to R2 remote backend`,
	SilenceUsage: true, // Don't show usage on error
	RunE: func(cmd *cobra.Command, args []string) error {
		return e2e.RunBootstrap()
	},
}

var cleanCmd = &cobra.Command{
	Use:          "clean",
	Short:        "Remove modules from remote state",
	Long:         `Removes specified modules from the v4 remote Terraform state in R2. Useful for cleaning up resources that have been manually deleted from Cloudflare but still exist in the Terraform state.`,
	SilenceUsage: true, // Don't show usage on error
	RunE: func(cmd *cobra.Command, args []string) error {
		modulesStr, _ := cmd.Flags().GetString("modules")
		if modulesStr == "" {
			return fmt.Errorf("no modules specified\nUsage: e2e clean --modules <module1,module2,...>")
		}
		modules := strings.Split(modulesStr, ",")
		for i := range modules {
			modules[i] = strings.TrimSpace(modules[i])
			if modules[i] == "" {
				return fmt.Errorf("empty module name in list")
			}
		}
		return e2e.RunClean(modules)
	},
}

// ============================================================================
// V5 Upgrade Commands - Tests provider v5 upgrades for v5→v5 migrations
// ============================================================================

var v5UpgradeCleanCmd = &cobra.Command{
	Use:          "v5-upgrade-clean",
	Short:        "Clean up v5 upgrade test resources",
	Long:         `Destroys resources and removes them from the v5 upgrade test state. Use --modules to target specific modules.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := &e2e.V5UpgradeConfig{
			FromVersion: cmd.Flag("from-version").Value.String(),
			ToVersion:   cmd.Flag("to-version").Value.String(),
		}

		var modules []string
		modulesStr, _ := cmd.Flags().GetString("modules")
		if modulesStr != "" {
			for _, m := range strings.Split(modulesStr, ",") {
				m = strings.TrimSpace(m)
				if m != "" {
					modules = append(modules, m)
				}
			}
		}

		return e2e.RunV5UpgradeClean(cfg, modules)
	},
}

var v5UpgradeCmd = &cobra.Command{
	Use:          "v5-upgrade",
	Short:        "Test provider v5 upgrades for v5→v5 migrations",
	SilenceUsage: true,
	Long: `Tests that provider v5 upgrades work correctly when upgrading between v5 versions.

This command validates that users can safely upgrade from older v5 provider versions
(e.g., 5.18.0) to newer versions without state corruption or unexpected drift.

Unlike v4→v5 migration tests, these tests:
  - Do NOT use tf-migrate (configs are already v5 syntax)
  - Test the provider's built-in UpgradeState mechanism
  - Create resources with an older v5 version, then upgrade the provider

Examples:
  # Run with local provider (recommended)
  e2e v5-upgrade --from-version 5.18.0 --provider ../provider --apply-exemptions

  # Run with registry provider
  e2e v5-upgrade --from-version 5.18.0 --to-version latest --apply-exemptions

  # Run and clean up after
  e2e v5-upgrade --from-version 5.18.0 --provider ../provider --clean

  # Test specific resources
  e2e v5-upgrade --from-version 5.18.0 --resources dns_record,ruleset --provider ../provider`,
	RunE: func(cmd *cobra.Command, args []string) error {
		parallelism, _ := cmd.Flags().GetInt("parallelism")

		cfg := &e2e.V5UpgradeConfig{
			FromVersion:     cmd.Flag("from-version").Value.String(),
			ToVersion:       cmd.Flag("to-version").Value.String(),
			Resources:       cmd.Flag("resources").Value.String(),
			Exclude:         cmd.Flag("exclude").Value.String(),
			ApplyExemptions: cmd.Flag("apply-exemptions").Changed,
			Parallelism:     parallelism,
			SkipCreate:      cmd.Flag("skip-create").Changed,
			ProviderPath:    cmd.Flag("provider").Value.String(),
			Clean:           cmd.Flag("clean").Changed,
		}
		return e2e.RunV5UpgradeTests(cfg)
	},
}

func init() {
	// Init command flags
	initCmd.Flags().String("resources", "", "Target specific resources (comma-separated)")
	initCmd.Flags().String("phase", "", "Run predefined phase(s) (comma-separated numbers, e.g., '0' or '0,1')")

	// Migrate command flags
	migrateCmd.Flags().String("resources", "", "Target specific resources (comma-separated)")
	migrateCmd.Flags().String("phase", "", "Run predefined phase(s) (comma-separated numbers, e.g., '0' or '0,1')")

	// Run command flags
	runCmd.Flags().Bool("skip-v4-test", false, "Skip v4 testing phase")
	runCmd.Flags().Bool("apply-exemptions", false, "Apply drift exemptions from global and resource-specific configs")
	runCmd.Flags().String("resources", "", "Target specific resources (comma-separated)")
	runCmd.Flags().String("exclude", "", "Exclude specific resources (comma-separated)")
	runCmd.Flags().String("phase", "", "Run predefined phase(s) (comma-separated numbers, e.g., '0' or '0,1')")
	runCmd.Flags().String("provider", "", "Path to provider source directory (will be built automatically)")
	runCmd.Flags().Int("parallelism", 0, "Terraform parallelism for plan/apply (0 uses Terraform default)")
	runCmd.Flags().Bool("no-refresh-snapshot", false, "Run an additional diagnostic terraform plan with -refresh=false before the authoritative refresh plan")

	// Clean command flags
	cleanCmd.Flags().String("modules", "", "Modules to remove from state (comma-separated)")

	// V5 Upgrade command flags
	v5UpgradeCmd.Flags().String("from-version", e2e.DefaultFromVersion, "Source provider version (e.g., 5.18.0)")
	v5UpgradeCmd.Flags().String("to-version", e2e.DefaultToVersion, "Target provider version (e.g., latest, 5.20.0)")
	v5UpgradeCmd.Flags().String("resources", "", "Target specific resources (comma-separated)")
	v5UpgradeCmd.Flags().String("exclude", "", "Exclude specific resources (comma-separated)")
	v5UpgradeCmd.Flags().Bool("apply-exemptions", false, "Apply drift exemptions from global and resource-specific configs")
	v5UpgradeCmd.Flags().Int("parallelism", 0, "Terraform parallelism for plan/apply (0 uses Terraform default)")
	v5UpgradeCmd.Flags().Bool("skip-create", false, "Skip resource creation, use existing state")
	v5UpgradeCmd.Flags().String("provider", "", "Path to local provider source directory (will be built automatically)")
	v5UpgradeCmd.Flags().Bool("clean", false, "Destroy resources and clean up state after test completes")

	// V5 Upgrade Clean command flags
	v5UpgradeCleanCmd.Flags().String("from-version", e2e.DefaultFromVersion, "Source provider version (e.g., 5.18.0)")
	v5UpgradeCleanCmd.Flags().String("to-version", e2e.DefaultToVersion, "Target provider version (e.g., latest, 5.20.0)")
	v5UpgradeCleanCmd.Flags().String("modules", "", "Modules to clean (comma-separated, e.g., 'page_rule,dns_record')")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(bootstrapCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(v5UpgradeCmd)
	rootCmd.AddCommand(v5UpgradeCleanCmd)
}

func main() {
	// Initialize the migration registry once at startup
	registry.RegisterAllMigrations()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
