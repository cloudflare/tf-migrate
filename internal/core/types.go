package core

// MigrationVersion represents a provider version
type MigrationVersion string

const (
	V4 MigrationVersion = "v4"
	V5 MigrationVersion = "v5"
	V6 MigrationVersion = "v6"
)

// MigrationKey represents a migration path (e.g., "v4_to_v5")
type MigrationKey string

// GetMigrationKey creates a migration key from source and target versions
func GetMigrationKey(source, target MigrationVersion) MigrationKey {
	return MigrationKey(string(source) + "_to_" + string(target))
}

// ResourceType represents a Terraform resource type
type ResourceType string