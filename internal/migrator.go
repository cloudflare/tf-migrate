package internal

import (
	"fmt"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

// Migrator holds information about a registered migrator
type Migrator struct {
	ResourceMigrator transform.ResourceTransformer
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
		return reg.ResourceMigrator
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
				result = append(result, reg.ResourceMigrator)
			}
		}
		return result
	}
	for _, reg := range migrators {
		if reg.SourceVersion == sourceVersion && reg.TargetVersion == targetVersion {
			result = append(result, reg.ResourceMigrator)
		}
	}
	return result
}

// RegisterMigrator registers a migrator for a specific version transition
func RegisterMigrator(sourceVersionResourceType string, sourceVersion string, targetVersion string, resourceMigrator transform.ResourceTransformer) {
	key := fmt.Sprintf("%s:%s:%s", sourceVersionResourceType, sourceVersion, targetVersion)
	migrators[key] = &Migrator{
		ResourceMigrator: resourceMigrator,
		SourceVersion:    sourceVersion,
		TargetVersion:    targetVersion,
	}
}
