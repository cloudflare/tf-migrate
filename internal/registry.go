package internal

import (
	"fmt"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

// ResourceMigrator is a function that returns a transformer for a resource
type ResourceMigrator func() transform.ResourceTransformer

// Migrator holds information about a registered migrator
type Migrator struct {
	ResourceMigrator ResourceMigrator
	SourceVersion    string
	TargetVersion    string
}

// Migrators is the map of all available migrators
// Key format: "resourceType:sourceVersion:targetVersion"
var migrators = make(map[string]*Migrator)

// GetMigrator returns a new instance of the migrator for the given resource type and versions
func GetMigrator(resourceType string, sourceVersion string, targetVersion string) transform.ResourceTransformer {
	key := fmt.Sprintf("%s:%s:%s", resourceType, sourceVersion, targetVersion)
	if reg, ok := migrators[key]; ok {
		return reg.ResourceMigrator()
	}
	return nil
}

// GetAllMigrators returns new instances of all migrators for the specified versions
func GetAllMigrators(sourceVersion string, targetVersion string, resources ...string) []transform.ResourceTransformer {
	result := make([]transform.ResourceTransformer, 0)

	// Only return the migrators for the resources specified
	if len(resources) > 0 {
		for _, r := range resources {
			key := fmt.Sprintf("%s:%s:%s", r, sourceVersion, targetVersion)
			if reg, ok := migrators[key]; ok {
				result = append(result, reg.ResourceMigrator())
			}
		}
		return result
	}
	for _, reg := range migrators {
		if reg.SourceVersion == sourceVersion && reg.TargetVersion == targetVersion {
			result = append(result, reg.ResourceMigrator())
		}
	}
	return result
}

// Register registers a migrator for a specific version transition
func Register(resourceType string, sourceVersion string, targetVersion string, resourceMigrator ResourceMigrator) {
	key := fmt.Sprintf("%s:%s:%s", resourceType, sourceVersion, targetVersion)
	migrators[key] = &Migrator{
		ResourceMigrator: resourceMigrator,
		SourceVersion:    sourceVersion,
		TargetVersion:    targetVersion,
	}
}
