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
		return e2e.RunMigrate(resources)
	},
}

var runCmd = &cobra.Command{
	Use:          "run",
	Short:        "Run full e2e test suite",
	Long:         `Runs the complete e2e test: init, v4 apply, migrate, v5 apply, drift check`,
	SilenceUsage: true, // Don't show usage on error
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := &e2e.RunConfig{
			SkipV4Test:                cmd.Flag("skip-v4-test").Changed,
			ApplyExemptions:           cmd.Flag("apply-exemptions").Changed,
			Resources:                 cmd.Flag("resources").Value.String(),
			Phase:                     cmd.Flag("phase").Value.String(),
			ProviderPath:              cmd.Flag("provider").Value.String(),
			UsesProviderStateUpgrader: cmd.Flag("uses-provider-state-upgrader").Changed,
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
	runCmd.Flags().String("phase", "", "Run predefined phase(s) (comma-separated numbers, e.g., '0' or '0,1')")
	runCmd.Flags().String("provider", "", "Path to provider source directory (will be built automatically)")
	runCmd.Flags().Bool("uses-provider-state-upgrader", false, "Only test resources that use provider-based state migration")

	// Clean command flags
	cleanCmd.Flags().String("modules", "", "Modules to remove from state (comma-separated)")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(bootstrapCmd)
	rootCmd.AddCommand(cleanCmd)
}

func main() {
	// Initialize the migration registry once at startup
	registry.RegisterAllMigrations()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
