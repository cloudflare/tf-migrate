package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"

	"github.com/cloudflare/tf-migrate/internal/infrastructure/backup"
	"github.com/cloudflare/tf-migrate/internal/processing"
	"github.com/cloudflare/tf-migrate/internal/core"
	"github.com/cloudflare/tf-migrate/internal/resources"
)

// Migrator handles the migration process
type Migrator struct {
	config   *Config
	registry *core.Registry
	log      hclog.Logger
	backup   *backup.Manager
}

// New creates a new Migrator instance with all transformers registered
func New(config *Config, logger hclog.Logger) *Migrator {
	// Create registry with all needed transformers upfront
	reg := core.NewRegistry()
	
	// Get the migration key from the config (e.g., "v4_to_v5")
	migrationKey := config.GetMigrationKey()
	
	if err := resources.RegisterAll(reg, migrationKey, config.Resources...); err != nil {
		// Log error but continue with empty registry
		logger.Error("Failed to register resources", "error", err, "migration", migrationKey)
	}
	
	return &Migrator{
		config:   config,
		registry: reg,
		log:      logger,
		backup:   backup.New(),
	}
}

// Run performs the migration based on the configuration with automatic rollback on failure
func (m *Migrator) Run() error {
	if len(m.config.Resources) > 0 {
		m.log.Debug("Filtering for resources", "resources", m.config.Resources)
	}

	// Create backups first if we're modifying in-place
	if m.shouldCreateBackup() {
		if err := m.createAllBackups(); err != nil {
			return core.NewError(core.BackupError).
				WithOperation("creating backups").
				WithCause(err).
				Build()
		}
		
		fmt.Println("✓ Created backups of all files to be modified")
	}

	// Track if we've made any changes
	var migrationErr error

	// Process configuration files if specified
	if m.config.ConfigDir != "" {
		if err := m.processConfigFiles(); err != nil {
			migrationErr = core.NewError(core.ConfigError).
				WithOperation("processing configuration files").
				WithFile(m.config.ConfigDir).
				WithCause(err).
				Build()
		}
	}

	// Process state file if specified (only if configs succeeded)
	if migrationErr == nil && m.config.StateFile != "" {
		if err := m.processStateFile(); err != nil {
			migrationErr = core.NewError(core.StateError).
				WithOperation("processing state file").
				WithFile(m.config.StateFile).
				WithCause(err).
				Build()
		}
	}

	// Handle migration result
	if migrationErr != nil {
		// Rollback on error
		if m.backup.HasBackups() && !m.config.DryRun {
			fmt.Println("\n⚠️  Migration failed, rolling back changes...")
			
			if rollbackErr := m.backup.Rollback(); rollbackErr != nil {
				// Rollback failed - this is critical
				return core.NewError(core.BackupError).
					WithOperation("rolling back changes").
					WithContext("migration_error", migrationErr).
					WithCause(rollbackErr).
					Build()
			}
			
			fmt.Println("✓ Successfully rolled back all changes")
			
			// Clean up backup files after successful rollback
			_ = m.backup.Cleanup()
		}
		
		return migrationErr
	}

	// Success - clean up backup files if requested
	if m.backup.HasBackups() && !m.config.DryRun {
		if m.config.Backup {
			// Keep backups but clear tracking
			fmt.Println("\n✓ Migration successful. Backup files preserved with .backup extension")
		} else {
			// Remove backup files
			if err := m.backup.Cleanup(); err != nil {
				m.log.Warn("Failed to clean up some backup files", "error", err)
			}
		}
	}

	return nil
}

// createAllBackups creates backups of all files that will be modified
func (m *Migrator) createAllBackups() error {
	// Backup config files if we're modifying in place
	if m.config.ConfigDir != "" && m.config.OutputDir == m.config.ConfigDir {
		if err := m.backup.BackupDirectory(m.config.ConfigDir, ".tf"); err != nil {
			return core.NewError(core.BackupError).
				WithOperation("backing up configuration files").
				WithFile(m.config.ConfigDir).
				WithCause(err).
				Build()
		}
	}

	// Backup state file if we're modifying in place
	if m.config.StateFile != "" && m.config.OutputState == m.config.StateFile {
		if err := m.backup.Backup(m.config.StateFile); err != nil {
			return core.NewError(core.BackupError).
				WithOperation("backing up state file").
				WithFile(m.config.StateFile).
				WithCause(err).
				Build()
		}
	}

	return nil
}

func (m *Migrator) processConfigFiles() error {
	errorList := core.NewErrorList(10)
	
	files, err := m.findTerraformFiles()
	if err != nil {
		return core.NewError(core.FileError).
			WithOperation("listing .tf files").
			WithFile(m.config.ConfigDir).
			WithCause(err).
			Build()
	}

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "Warning: No .tf files found in directory: %s\n", m.config.ConfigDir)
		return nil
	}

	fmt.Printf("\nFound %d configuration files to migrate\n", len(files))

	// Process all files, collecting any errors
	successCount := 0

	for i, file := range files {
		if err := m.processConfigFile(file, i+1, len(files)); err != nil {
			errorList.Add(err)
			// Stop on first error for safety
			break
		}
		successCount++
	}

	// Report results
	if errorList.HasErrors() {
		return core.NewError(core.ConfigError).
			WithOperation("processing configuration files").
			WithContext("processed_count", successCount).
			WithContext("total_count", len(files)).
			WithCause(errorList.First()).
			Build()
	}

	return nil
}

func (m *Migrator) processConfigFile(file string, current, total int) error {
	fmt.Printf("[%d/%d] Processing: %s\n", current, total, filepath.Base(file))

	content, err := os.ReadFile(file)
	if err != nil {
		return core.NewError(core.FileError).
			WithOperation("reading configuration file").
			WithFile(file).
			WithCause(err).
			Build()
	}

	// Transform the content
	transformed, err := processing.ProcessConfig(content, filepath.Base(file), m.registry)
	if err != nil {
		// Error already wrapped by ProcessConfig
		return err
	}

	// Write the output
	outputPath := filepath.Join(m.config.OutputDir, filepath.Base(file))
	return m.writeOutput(outputPath, transformed, false)
}

func (m *Migrator) processStateFile() error {
	fmt.Printf("\nProcessing state file: %s\n", m.config.StateFile)

	content, err := os.ReadFile(m.config.StateFile)
	if err != nil {
		return core.NewError(core.FileError).
			WithOperation("reading state file").
			WithFile(m.config.StateFile).
			WithCause(err).
			Build()
	}

	// Transform the content
	transformed, err := processing.ProcessState(content, filepath.Base(m.config.StateFile), m.registry)
	if err != nil {
		// Error already wrapped by ProcessState
		return err
	}

	return m.writeOutput(m.config.OutputState, transformed, true)
}

func (m *Migrator) findTerraformFiles() ([]string, error) {
	var files []string

	entries, err := os.ReadDir(m.config.ConfigDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tf") {
			files = append(files, filepath.Join(m.config.ConfigDir, entry.Name()))
		}
	}

	return files, nil
}

func (m *Migrator) shouldCreateBackup() bool {
	// Create backups if:
	// 1. Not in dry-run mode
	// 2. We're modifying files in place (not outputting to different location)
	if m.config.DryRun {
		return false
	}
	
	// Check if any operation is in-place
	inPlaceConfig := m.config.ConfigDir != "" && m.config.OutputDir == m.config.ConfigDir
	inPlaceState := m.config.StateFile != "" && m.config.OutputState == m.config.StateFile
	
	return inPlaceConfig || inPlaceState
}

func (m *Migrator) writeOutput(path string, content []byte, isState bool) error {
	if m.config.DryRun {
		action := "Would write file"
		if isState {
			action = "Would write transformed state"
		}
		fmt.Printf("%s: %s\n", action, path)
		return nil
	}

	// Create output directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return core.NewError(core.FileError).
			WithOperation("creating output directory").
			WithFile(dir).
			WithCause(err).
			Build()
	}

	if err := os.WriteFile(path, content, 0644); err != nil {
		return core.NewError(core.FileError).
			WithOperation("writing output file").
			WithFile(path).
			WithCause(err).
			Build()
	}

	action := "  ✓ Migrated"
	if isState {
		action = "  ✓ Transformed state"
	}
	
	fmt.Printf("%s: %s\n", action, path)
	
	return nil
}