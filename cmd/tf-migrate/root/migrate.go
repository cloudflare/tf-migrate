package root

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudflare/tf-migrate/internal/logger"
	"github.com/cloudflare/tf-migrate/internal/pipeline"
	"github.com/cloudflare/tf-migrate/internal/registry"
	"github.com/cloudflare/tf-migrate/internal/resources"
)

// runMigration performs the actual migration using the pipeline
func runMigration(configDir string, stateFile string, outputDir string, outputState string,
	resourcesToMigrate []string, dryRun bool, backup bool) error {

	reg := registry.NewStrategyRegistry()

	resources.RegisterFromFactories(reg, resourcesToMigrate...)

	// TODO:: select resources to migrate
	if len(resourcesToMigrate) > 0 {
		logger.Info("Filtering for resources", "resources", resourcesToMigrate)
	}

	configPipeline := pipeline.BuildConfigPipeline(reg)
	if configDir != "" {
		if err := processConfigFiles(configPipeline, configDir, outputDir, dryRun, backup); err != nil {
			return fmt.Errorf("failed to process configuration files: %w", err)
		}
	}

	statePipeline := pipeline.BuildStatePipeline(reg)
	if stateFile != "" {
		if err := processStateFile(statePipeline, stateFile, outputState, dryRun, backup); err != nil {
			return fmt.Errorf("failed to process state file: %w", err)
		}
	}

	return nil
}

func processConfigFiles(p *pipeline.Pipeline, configDir string, outputDir string, dryRun bool, backup bool) error {
	if outputDir == "" {
		outputDir = configDir
	}

	files, err := findTerraformFiles(configDir)
	if err != nil {
		return fmt.Errorf("failed to list .tf files: %w", err)
	}

	if len(files) == 0 {
		logger.Warn("No .tf files found", "directory", configDir)
		return nil
	}

	logger.Info("Found configuration files to migrate", "count", len(files))

	for i, file := range files {
		logger.Info("Processing file", "file", filepath.Base(file), "progress", fmt.Sprintf("%d/%d", i+1, len(files)))

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		if backup && !dryRun && outputDir == configDir {
			backupPath := file + ".backup"
			if err := os.WriteFile(backupPath, content, 0644); err != nil {
				return fmt.Errorf("failed to create backup %s: %w", backupPath, err)
			}
			logger.Debug("Created backup", "path", backupPath)
		}

		transformed, err := p.Transform(content, filepath.Base(file))
		if err != nil {
			return fmt.Errorf("failed to transform %s: %w", file, err)
		}

		outputPath := filepath.Join(outputDir, filepath.Base(file))

		if dryRun {
			logger.Info("Would write file", "output", outputPath)
		} else {
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}

			if err := os.WriteFile(outputPath, transformed, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", outputPath, err)
			}
			logger.Info("Migrated file", "output", outputPath)
		}
	}

	return nil
}

func processStateFile(p *pipeline.Pipeline, stateFile string, outputStatePath string, dryRun bool, backup bool) error {
	if p == nil {
		return fmt.Errorf("state pipeline is nil")
	}

	logger.Info("Processing state file", "file", stateFile)

	content, err := os.ReadFile(stateFile)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	// If no output path specified, use input path (in-place)
	if outputStatePath == "" {
		outputStatePath = stateFile
	}

	if backup && !dryRun && outputStatePath == stateFile {
		backupPath := stateFile + ".backup"
		if err := os.WriteFile(backupPath, content, 0644); err != nil {
			return fmt.Errorf("failed to create state backup %s: %w", backupPath, err)
		}
		logger.Debug("Created state backup", "path", backupPath)
	}

	transformedContent, err := p.Transform(content, filepath.Base(stateFile))
	if err != nil {
		return fmt.Errorf("failed to transform state file: %w", err)
	}

	if dryRun {
		logger.Info("Would write transformed state", "output", outputStatePath)
	} else {
		if err := os.WriteFile(outputStatePath, transformedContent, 0644); err != nil {
			return fmt.Errorf("failed to write state %s: %w", outputStatePath, err)
		}
		logger.Info("Wrote transformed state", "output", outputStatePath)
	}

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
