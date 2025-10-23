package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl/v2"

	"github.com/spf13/cobra"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/logger"
	"github.com/cloudflare/tf-migrate/internal/pipeline"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type config struct {
	// Input paths
	configDir string
	stateFile string

	// Output paths
	outputDir   string
	outputState string

	// Migration options
	resourcesToMigrate []string
	sourceVersion      string
	targetVersion      string
	dryRun             bool
	backup             bool
	logLevel           string
}

var (
	rootCmd = &cobra.Command{
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
  tf-migrate --dry-run migrate

  # Run with debug logging
  tf-migrate --log-level debug migrate`,
	}
)

var validVersionPath = map[string]struct{}{
	"4-5": {},
}

func main() {
	cfg := &config{}
	rootCmd.PersistentFlags().StringVar(&cfg.configDir, "config-dir", "", "Directory containing Terraform configuration files")
	rootCmd.PersistentFlags().StringVar(&cfg.stateFile, "state-file", "", "Path to Terraform state file")
	rootCmd.PersistentFlags().StringSliceVar(&cfg.resourcesToMigrate, "resources", []string{}, "Comma-separated list of resources to migrate (empty = all)")
	rootCmd.PersistentFlags().BoolVar(&cfg.dryRun, "dry-run", false, "Perform a dry run without making changes")
	rootCmd.PersistentFlags().StringVar(&cfg.sourceVersion, "source-version", "", "Source provider version (e.g., v4, v5)")
	rootCmd.PersistentFlags().StringVar(&cfg.targetVersion, "target-version", "", "Target provider version (e.g., v5, v6)")

	rootCmd.PersistentFlags().StringVarP(&cfg.logLevel, "log-level", "l", "warn", "Set log level (debug, info, warn, error, off)")

	// Create logger instance
	log := logger.New(cfg.logLevel)
	fmt.Println(fmt.Sprintf("MAIN %+v", cfg))
	rootCmd.AddCommand(newMigrateCommand(log, cfg))
	rootCmd.AddCommand(newVersionCommand())
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newMigrateCommand(log hclog.Logger, cfg *config) *cobra.Command {
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
  tf-migrate --dry-run migrate

  # Run with debug logging
  tf-migrate --log-level debug migrate`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(fmt.Sprintf("CFG %+v", cfg))
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
	cmd.Flags().StringVar(&cfg.outputState, "output-state", "", "Output path for migrated state file (default: in-place)")
	cmd.Flags().BoolVar(&cfg.backup, "backup", true, "Create backup of original files before migration")

	return cmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("tf-migrate version 0.1.0")
		},
	}
}

// runMigration performs the actual migration using the pipeline
func runMigration(log hclog.Logger, cfg config) error {
	err := validateVersions(cfg)
	if err != nil {
		return err
	}

	providers := getProviders(cfg.resourcesToMigrate...)
	configPipeline := pipeline.BuildConfigPipeline(log, providers)
	if cfg.configDir != "" {
		if err := processConfigFiles(log, configPipeline, cfg); err != nil {
			return fmt.Errorf("failed to process configuration files: %w", err)
		}
	}
	log.Debug("Finished processing configuration files")

	statePipeline := pipeline.BuildStatePipeline(log, providers)
	if cfg.stateFile != "" {
		if err := processStateFile(log, statePipeline, cfg); err != nil {
			return fmt.Errorf("failed to process state file: %w", err)
		}
	}
	log.Debug("Finished processing state file")

	return nil
}

func processConfigFiles(log hclog.Logger, p *pipeline.Pipeline, cfg config) error {
	if cfg.outputDir == "" {
		cfg.outputDir = cfg.configDir
	}

	files, err := findTerraformFiles(cfg.configDir)
	if err != nil {
		return fmt.Errorf("failed to list .tf files: %w", err)
	}

	if len(files) == 0 {
		fmt.Printf("No .tf files found in %s\n", cfg.configDir)
		return nil
	}

	fmt.Printf("\nFound %d configuration files to migrate\n", len(files))

	for i, file := range files {
		fmt.Printf("[%d/%d] Processing %s... ", i+1, len(files), filepath.Base(file))
		log.Debug("Processing file", "file", file, "index", i+1)

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		if cfg.backup && !cfg.dryRun && cfg.outputDir == cfg.configDir {
			backupPath := file + ".backup"
			if err := os.WriteFile(backupPath, content, 0644); err != nil {
				return fmt.Errorf("failed to create backup %s: %w", backupPath, err)
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
			return fmt.Errorf("failed to transform %s: %w", file, err)
		}

		outputPath := filepath.Join(cfg.outputDir, filepath.Base(file))

		if cfg.dryRun {
			fmt.Println("(dry run)")
			log.Debug("Would write file", "output", outputPath)
			return nil
		}

		if err := os.MkdirAll(cfg.outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		if err := os.WriteFile(outputPath, transformed, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", outputPath, err)
		}
		fmt.Println("✓")
		log.Debug("Migrated file", "output", outputPath)
	}

	return nil
}

func processStateFile(log hclog.Logger, p *pipeline.Pipeline, cfg config) error {
	if p == nil {
		return fmt.Errorf("state pipeline is nil")
	}

	fmt.Printf("\nProcessing state file: %s... ", filepath.Base(cfg.stateFile))
	log.Debug("Processing state file", "file", cfg.stateFile)

	content, err := os.ReadFile(cfg.stateFile)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	// If no output path specified, use input path (in-place)
	if cfg.outputState == "" {
		cfg.outputState = cfg.stateFile
	}

	if cfg.backup && !cfg.dryRun && cfg.outputState == cfg.stateFile {
		backupPath := cfg.stateFile + ".backup"
		if err := os.WriteFile(backupPath, content, 0644); err != nil {
			return fmt.Errorf("failed to create state backup %s: %w", backupPath, err)
		}
		log.Debug("Created state backup", "path", backupPath)
	}

	ctx := &transform.Context{
		Content:       content,
		Filename:      filepath.Base(cfg.stateFile),
		Diagnostics:   make(hcl.Diagnostics, 0),
		Metadata:      make(map[string]interface{}),
		SourceVersion: cfg.sourceVersion,
		TargetVersion: cfg.targetVersion,
		Resources:     cfg.resourcesToMigrate,
	}
	transformedContent, err := p.Transform(ctx)
	if err != nil {
		return fmt.Errorf("failed to transform state file: %w", err)
	}

	if cfg.dryRun {
		fmt.Println("(dry run)")
		log.Debug("Would write transformed state", "output", cfg.outputState)
		return nil
	}

	if err := os.WriteFile(cfg.outputState, transformedContent, 0644); err != nil {
		return fmt.Errorf("failed to write state %s: %w", cfg.outputState, err)
	}
	fmt.Println("✓")
	log.Debug("Wrote transformed state", "output", cfg.outputState)
	return nil
}

func findTerraformFiles(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tf") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files, nil
}

func validateVersions(c config) error {
	source := strings.TrimPrefix(c.sourceVersion, "v")
	target := strings.TrimPrefix(c.targetVersion, "v")
	versionPath := fmt.Sprintf("%s-%s", source, target)
	if _, ok := validVersionPath[versionPath]; !ok {
		return fmt.Errorf("unsupported migration path: %s", versionPath)
	}

	return nil
}

func getProviders(resources ...string) transform.Provider {
	getFunc := func(resourceType string, source string, target string) transform.ResourceTransformer {
		return internal.GetMigrator(resourceType, source, target)
	}
	getAllFunc := func(source string, target string, resourcesToMigrate ...string) []transform.ResourceTransformer {
		return internal.GetAllMigrators(source, target, resources...)
	}
	return transform.NewMigratorProvider(getFunc, getAllFunc)
}
