package migration

import (
	"fmt"
	"strings"
)

// Config holds all configuration options for the migration
type Config struct {
	// Input paths
	ConfigDir  string
	StateFile  string
	
	// Output paths
	OutputDir   string
	OutputState string
	
	// Migration options
	Resources     []string
	SourceVersion string
	TargetVersion string
	DryRun        bool
	Backup        bool
}

// SetDefaults sets default values for the configuration
func (c *Config) SetDefaults() {
	if c.ConfigDir == "" {
		c.ConfigDir = "."
	}
	
	if c.OutputDir == "" {
		c.OutputDir = c.ConfigDir
	}
	
	if c.OutputState == "" {
		c.OutputState = c.StateFile
	}
	
	// Default versions if not specified
	if c.SourceVersion == "" {
		c.SourceVersion = "v4"
	}
	if c.TargetVersion == "" {
		c.TargetVersion = "v5"
	}
}

// ValidateVersions checks if the version migration is supported
func (c *Config) ValidateVersions() error {
	// Normalize versions (remove 'v' prefix if present)
	source := strings.TrimPrefix(c.SourceVersion, "v")
	target := strings.TrimPrefix(c.TargetVersion, "v")
	
	// Check if versions are valid
	validVersions := map[string]bool{
		"4": true,
		"5": true,
		"6": true,
	}
	
	if !validVersions[source] {
		return fmt.Errorf("unsupported source version: %s (supported: v4, v5, v6)", c.SourceVersion)
	}
	
	if !validVersions[target] {
		return fmt.Errorf("unsupported target version: %s (supported: v4, v5, v6)", c.TargetVersion)
	}
	
	// Check for multi-step migrations (not supported yet)
	sourceInt := 0
	targetInt := 0
	fmt.Sscanf(source, "%d", &sourceInt)
	fmt.Sscanf(target, "%d", &targetInt)
	
	if targetInt <= sourceInt {
		return fmt.Errorf("target version (%s) must be greater than source version (%s)", c.TargetVersion, c.SourceVersion)
	}
	
	if targetInt-sourceInt > 1 {
		return fmt.Errorf("multi-step migrations not supported (from %s to %s). Please migrate one version at a time", c.SourceVersion, c.TargetVersion)
	}
	
	return nil
}

// GetMigrationKey returns the migration key for the version pair (e.g., "v4_to_v5")
func (c *Config) GetMigrationKey() string {
	source := strings.TrimPrefix(c.SourceVersion, "v")
	target := strings.TrimPrefix(c.TargetVersion, "v")
	return fmt.Sprintf("v%s_to_v%s", source, target)
}