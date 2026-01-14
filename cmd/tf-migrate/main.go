package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudflare/cloudflare-go/v6"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/spf13/cobra"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/logger"
	"github.com/cloudflare/tf-migrate/internal/pipeline"
	"github.com/cloudflare/tf-migrate/internal/registry"
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
	recursive          bool
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
	"v4-v5": {},
}

func main() {
	// Register all resource migrations
	registry.RegisterAllMigrations()

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
	cmd.Flags().BoolVar(&cfg.recursive, "recursive", false, "Recursively process subdirectories (useful for module structures)")

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

// initAPIClient initializes a Cloudflare API client if credentials are available
// It checks for CLOUDFLARE_API_TOKEN first, then falls back to CLOUDFLARE_API_KEY + CLOUDFLARE_EMAIL
func initAPIClient() *cloudflare.Client {
	apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	apiKey := os.Getenv("CLOUDFLARE_API_KEY")
	apiEmail := os.Getenv("CLOUDFLARE_EMAIL")

	if apiToken != "" {
		fmt.Println("✓ Using Cloudflare API credentials (API token)")
		return cloudflare.NewClient()
	}

	if apiKey != "" && apiEmail != "" {
		fmt.Println("✓ Using Cloudflare API credentials (API key + email)")
		return cloudflare.NewClient()
	}

	fmt.Println("ℹ No Cloudflare API credentials found")
	fmt.Println("  Some migrations may require manual intervention after completion")
	fmt.Println("  Set CLOUDFLARE_API_TOKEN or (CLOUDFLARE_API_KEY + CLOUDFLARE_EMAIL) for full automation")
	fmt.Println()
	return nil
}

// runMigration performs the actual migration using the pipeline
func runMigration(log hclog.Logger, cfg config) error {
	err := validateVersions(cfg)
	if err != nil {
		return err
	}

	// Load state file first if present (needed for cross-referencing in config transformations)
	var stateJSON string
	if cfg.stateFile != "" {
		content, err := os.ReadFile(cfg.stateFile)
		if err != nil {
			log.Warn("Failed to read state file", "file", cfg.stateFile, "error", err)
			fmt.Printf("⚠  Failed to read state file: %s (error: %v)\n", cfg.stateFile, err)
		} else {
			stateJSON = string(content)
			log.Info("Loaded state file for cross-referencing", "file", cfg.stateFile, "size", len(stateJSON))
			fmt.Printf("✓ Loaded state file for config cross-referencing: %s\n", cfg.stateFile)
		}
	} else {
		log.Info("No state file specified")
		fmt.Println("ℹ No state file specified - config transformations will proceed without state")
	}

	// Initialize API client if credentials are available
	apiClient := initAPIClient()

	providers := getProviders(cfg.resourcesToMigrate...)
	configPipeline := pipeline.BuildConfigPipeline(log, providers)
	parsedConfigs := make(map[string]*hclwrite.File)
	if cfg.configDir != "" {
		parsedConfigs, err = processConfigFiles(log, configPipeline, cfg, stateJSON, apiClient)
		if err != nil {
			return fmt.Errorf("failed to process configuration files: %w", err)
		}
	}
	log.Debug("Finished processing configuration files")

	statePipeline := pipeline.BuildStatePipeline(log, providers)
	if cfg.stateFile != "" {
		if err := processStateFile(log, statePipeline, cfg, apiClient, parsedConfigs); err != nil {
			return fmt.Errorf("failed to process state file: %w", err)
		}
	}
	log.Debug("Finished processing state file")

	return nil
}

func processConfigFiles(log hclog.Logger, p *pipeline.Pipeline, cfg config, stateJSON string, apiClient *cloudflare.Client) (map[string]*hclwrite.File, error) {
	if cfg.outputDir == "" {
		cfg.outputDir = cfg.configDir
	}

	files, err := findTerraformFilesWithRecursion(cfg.configDir, cfg.recursive)
	if err != nil {
		return nil, fmt.Errorf("failed to list .tf files: %w", err)
	}

	if len(files) == 0 {
		fmt.Printf("No .tf files found in %s\n", cfg.configDir)
		return nil, nil
	}

	fmt.Printf("\nFound %d configuration files to migrate\n", len(files))

	// Store file paths for global postprocessing
	outputPaths := make([]string, 0, len(files))

	parsedConfigs := make(map[string]*hclwrite.File)
	for i, file := range files {
		fmt.Printf("[%d/%d] Processing %s... ", i+1, len(files), filepath.Base(file))
		log.Debug("Processing file", "file", file, "index", i+1)

		content, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", file, err)
		}

		if cfg.backup && !cfg.dryRun && cfg.outputDir == cfg.configDir {
			backupPath := file + ".backup"
			if err := os.WriteFile(backupPath, content, 0644); err != nil {
				return nil, fmt.Errorf("failed to create backup %s: %w", backupPath, err)
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
			StateJSON:     stateJSON, // For cross-referencing in config transformations
			APIClient:     apiClient,
		}
		transformed, err := p.Transform(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to transform %s: %w", file, err)
		}

		if ctx.CFGFile != nil {
			parsedConfigs[file] = ctx.CFGFile
		}

		// Calculate output path maintaining directory structure when recursive
		var outputPath string
		if cfg.recursive {
			// Preserve directory structure relative to config dir
			relPath, err := filepath.Rel(cfg.configDir, file)
			if err != nil {
				return nil, fmt.Errorf("failed to compute relative path: %w", err)
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
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}

		if err := os.WriteFile(outputPath, transformed, 0644); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", outputPath, err)
		}
		fmt.Println("✓")
		log.Debug("Migrated file", "output", outputPath)
		outputPaths = append(outputPaths, outputPath)

	}

	// Apply global postprocessing for cross-file reference updates
	if !cfg.dryRun && len(outputPaths) > 0 {
		if err := applyGlobalPostprocessing(log, cfg, outputPaths); err != nil {
			return nil, fmt.Errorf("failed to apply global postprocessing: %w", err)
		}
	}

	return parsedConfigs, nil
}

// applyGlobalPostprocessing applies cross-file reference updates for resource and attribute renames
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
			oldType, newType := renamer.GetResourceRename()
			if oldType != "" && newType != "" {
				// Only add to renames map if the types are different (actual rename)
				if oldType != newType {
					renames[oldType] = newType
					log.Debug("Collected resource rename", "old", oldType, "new", newType)
				} else {
					log.Debug("Resource type unchanged", "type", oldType)
				}
			} else {
				// Warn if migrator implements interface but returns empty values
				log.Warn("Migrator implements ResourceRenamer but returned empty type names",
					"old", oldType, "new", newType)
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

	// Apply renames to all files
	for _, outputPath := range outputPaths {
		content, err := os.ReadFile(outputPath)
		if err != nil {
			log.Warn("Failed to read file for global postprocessing", "file", outputPath, "error", err)
			continue
		}

		contentStr := string(content)
		modified := false

		// Apply all resource type renames
		for oldType, newType := range renames {
			newContent := strings.ReplaceAll(contentStr, oldType+".", newType+".")
			if newContent != contentStr {
				modified = true
				contentStr = newContent
				log.Debug("Updated references", "file", filepath.Base(outputPath), "old", oldType, "new", newType)
			}
		}

		// Apply all attribute renames
		// Pattern: data.cloudflare_zones.<instance_name>.zones → data.cloudflare_zones.<instance_name>.result
		// We need to match: <ResourceType>.<instance_name>.<OldAttribute>
		for _, rename := range attributeRenames {
			// Build regex pattern: data\.cloudflare_zones\.([a-zA-Z0-9_-]+)\.zones
			// The instance name can contain letters, numbers, underscores, and hyphens
			pattern := regexp.QuoteMeta(rename.ResourceType) + `\.([a-zA-Z0-9_-]+)\.` + regexp.QuoteMeta(rename.OldAttribute)
			re := regexp.MustCompile(pattern)

			// Replace with: data.cloudflare_zones.$1.result (preserving instance name)
			replacement := rename.ResourceType + ".$1." + rename.NewAttribute
			newContent := re.ReplaceAllString(contentStr, replacement)

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

func processStateFile(log hclog.Logger, p *pipeline.Pipeline, cfg config, apiClient *cloudflare.Client, parsedConfigs map[string]*hclwrite.File) error {
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
		StateJSON:     string(content),
		Filename:      filepath.Base(cfg.stateFile),
		Diagnostics:   make(hcl.Diagnostics, 0),
		Metadata:      make(map[string]interface{}),
		SourceVersion: cfg.sourceVersion,
		TargetVersion: cfg.targetVersion,
		Resources:     cfg.resourcesToMigrate,
		APIClient:     apiClient,
		CFGFiles:      parsedConfigs,
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
