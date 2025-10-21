package main

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/cloudflare/tf-migrate/internal/infrastructure/logging"
	"github.com/cloudflare/tf-migrate/internal/migration"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	rootCmd := buildRootCommand()
	return rootCmd.Execute()
}

// buildRootCommand creates the root command with all subcommands
func buildRootCommand() *cobra.Command {
	cfg := &migration.Config{
		Backup: true, // Default value
	}

	var log hclog.Logger

	rootCmd := &cobra.Command{
		Use:   "tf-migrate",
		Short: "Terraform configuration migration tool",
		Long: `tf-migrate is a CLI tool for migrating Terraform configurations and state files
between different provider versions or resource schemas.

This tool provides automated transformations for:
- Resource type changes
- Attribute migrations
- State file updates
- Import generation for new resources
- Moved blocks for resource renames`,
		Example: `  # Migrate all .tf files in current directory
  tf-migrate migrate

  # Migrate specific directory with state file
  tf-migrate --config-dir ./terraform --state-file terraform.tfstate migrate

  # Migrate only specific resources
  tf-migrate --resources dns_record,load_balancer migrate

  # Dry run to preview changes
  tf-migrate --dry-run migrate`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfg.ConfigDir, "config-dir", "", "Directory containing Terraform configuration files")
	rootCmd.PersistentFlags().StringVar(&cfg.StateFile, "state-file", "", "Path to Terraform state file")
	rootCmd.PersistentFlags().StringSliceVar(&cfg.Resources, "resources", []string{}, "Comma-separated list of resources to migrate (empty = all)")
	rootCmd.PersistentFlags().StringVar(&cfg.SourceVersion, "source-version", "", "Source provider version (e.g., v4, v5)")
	rootCmd.PersistentFlags().StringVar(&cfg.TargetVersion, "target-version", "", "Target provider version (e.g., v5, v6)")
	rootCmd.PersistentFlags().BoolVar(&cfg.DryRun, "dry-run", false, "Perform a dry run without making changes")

	// Initialize logger before running any command
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// Always use default logger with full output
		log = logging.New()
	}

	// Add subcommands
	rootCmd.AddCommand(newMigrateCommand(cfg, &log))
	rootCmd.AddCommand(newVersionCommand())

	return rootCmd
}

// newMigrateCommand creates the migrate subcommand
func newMigrateCommand(cfg *migration.Config, log *hclog.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run the migration on configuration and/or state files",
		Long: `Migrate Terraform configuration and state files using registered transformers.
Uses the global flags --config-dir, --state-file, and --resources to determine what to migrate.`,
		Example: `  # Migrate configuration files in current directory
  tf-migrate migrate

  # Migrate specific directory with state file
  tf-migrate --config-dir ./terraform --state-file terraform.tfstate migrate

  # Migrate with output to different directory
  tf-migrate migrate --output-dir ./migrated

  # Migrate only specific resources
  tf-migrate --resources dns_record,load_balancer migrate

  # Dry run to preview changes
  tf-migrate --dry-run migrate`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set defaults
			cfg.SetDefaults()

			// Validate version configuration
			if err := cfg.ValidateVersions(); err != nil {
				return err
			}

			// Always display migration info
			displayMigrationInfo(cfg)

			// Run the migration with the logger instance
			migrator := migration.New(cfg, *log)
			return migrator.Run()
		},
	}

	// Command-specific flags
	cmd.Flags().StringVar(&cfg.OutputDir, "output-dir", "", "Output directory for migrated configuration files (default: in-place)")
	cmd.Flags().StringVar(&cfg.OutputState, "output-state", "", "Output path for migrated state file (default: in-place)")
	cmd.Flags().BoolVar(&cfg.Backup, "backup", true, "Create backup of original files before migration")

	return cmd
}

// newVersionCommand creates the version subcommand
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("tf-migrate version 0.1.0")
		},
	}
}

// displayMigrationInfo shows the migration configuration to the user
// Uses fmt for UI output as recommended
func displayMigrationInfo(cfg *migration.Config) {
	fmt.Println("Cloudflare Terraform Provider Migration Tool")
	fmt.Println("============================================")
	fmt.Println()

	fmt.Printf("Migration: %s → %s\n", cfg.SourceVersion, cfg.TargetVersion)
	fmt.Printf("Configuration directory: %s\n", cfg.ConfigDir)
	if cfg.OutputDir != "" && cfg.OutputDir != cfg.ConfigDir {
		fmt.Printf("Output directory: %s\n", cfg.OutputDir)
	} else {
		fmt.Println("Output directory: in-place")
	}

	if cfg.StateFile != "" {
		fmt.Printf("State file: %s\n", cfg.StateFile)
		if cfg.OutputState != "" && cfg.OutputState != cfg.StateFile {
			fmt.Printf("Output state: %s\n", cfg.OutputState)
		} else {
			fmt.Println("Output state: in-place")
		}
	}

	if cfg.DryRun {
		fmt.Println("\n⚠️  DRY RUN MODE - No changes will be made")
	}
	fmt.Println()
}
