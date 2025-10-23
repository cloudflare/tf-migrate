package internal

import (
	"fmt"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// MigratorFactory is a function that creates a new ResourceTransformer
type MigratorFactory func() transform.ResourceTransformer

// MigratorRegistration holds information about a registered migrator
type MigratorRegistration struct {
	Factory       MigratorFactory
	SourceVersion string
	TargetVersion string
}

// Migrators is the map of all available migrators
// Key format: "resourceType:sourceVersion:targetVersion"
var migrators = make(map[string]*MigratorRegistration)

// GetMigrator returns a new instance of the migrator for the given resource type and versions
func GetMigrator(resourceType string, sourceVersion string, targetVersion string) transform.ResourceTransformer {
	key := fmt.Sprintf("%s:%s:%s", resourceType, sourceVersion, targetVersion)
	if reg, ok := migrators[key]; ok {
		return reg.Factory()
	}
	return nil
}

// GetAllMigrators returns new instances of all migrators for the specified versions
func GetAllMigrators(sourceVersion string, targetVersion string) []transform.ResourceTransformer {
	result := make([]transform.ResourceTransformer, 0)
	
	// Collect version-specific migrators
	for _, reg := range migrators {
		if reg.SourceVersion == sourceVersion && reg.TargetVersion == targetVersion {
			result = append(result, reg.Factory())
		}
	}
	
	return result
}

// RegisterVersioned registers a migrator for a specific version transition
func RegisterVersioned(resourceType string, sourceVersion string, targetVersion string, factory MigratorFactory) {
	key := fmt.Sprintf("%s:%s:%s", resourceType, sourceVersion, targetVersion)
	migrators[key] = &MigratorRegistration{
		Factory:       factory,
		SourceVersion: sourceVersion,
		TargetVersion: targetVersion,
	}
}

// Register is a convenience function that defaults to v4->v5 migration
func Register(resourceType string, factory MigratorFactory) {
	RegisterVersioned(resourceType, "v4", "v5", factory)
}
