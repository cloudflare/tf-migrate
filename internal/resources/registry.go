package resources

import (
	"fmt"

	"github.com/cloudflare/tf-migrate/internal/core"
	"github.com/cloudflare/tf-migrate/internal/core/transform"
	dns "github.com/cloudflare/tf-migrate/internal/resources/cloudflare_dns"
	zone_settings "github.com/cloudflare/tf-migrate/internal/resources/cloudflare_zone_settings"
)

// ResourceDefinition describes a resource transformation
type ResourceDefinition struct {
	Type        string                // Resource type (e.g., "cloudflare_record")
	Description string                // Human-readable description
	Deprecated  bool                  // Whether this resource is deprecated
	Version     string                // Migration version (e.g., "v4_to_v5")
	Transform   transform.Transformer // The transformer implementation
}

// GetAllResources returns all resource definitions for a specific migration version.
// THIS IS THE SINGLE SOURCE OF TRUTH for all resource transformations.
func GetAllResources(migrationKey string) []ResourceDefinition {
	switch migrationKey {
	case "v4_to_v5":
		return getV4ToV5Resources()
	default:
		// Return v4_to_v5 as default for backward compatibility
		return getV4ToV5Resources()
	}
}

// getV4ToV5Resources returns all v4 to v5 migration definitions
func getV4ToV5Resources() []ResourceDefinition {
	return []ResourceDefinition{
		// DNS Resources
		{
			Type:        "cloudflare_record",
			Description: "DNS record (migrated to cloudflare_dns_record)",
			Deprecated:  true,
			Version:     "v4_to_v5",
			Transform:   dns.V4ToV5(),
		},

		// Zone Settings
		{
			Type:        "cloudflare_zone_settings",
			Description: "Zone settings (renamed attributes)",
			Version:     "v4_to_v5",
			Transform:   zone_settings.V4ToV5(),
		},

		// Add all other v4 to v5 resources here...
		// Each resource package should have a v4_to_v5.go file
	}
}

// RegisterAll registers all or filtered resources to the given registry for a specific migration version
func RegisterAll(reg *core.Registry, migrationKey string, resourceFilter ...string) error {
	// Create filter map for quick lookup
	filterMap := make(map[string]bool)
	for _, filter := range resourceFilter {
		filterMap[filter] = true
	}

	// Get resources for the specific migration version
	resources := GetAllResources(migrationKey)

	// Track registration stats
	registered := 0
	skipped := 0

	for _, def := range resources {
		// Skip if filter is specified and resource not in filter
		if len(resourceFilter) > 0 && !filterMap[def.Type] {
			skipped++
			continue
		}

		// Register the transformer
		reg.Register(def.Transform)
		registered++
	}

	// Return error if filter specified but no resources matched
	if len(resourceFilter) > 0 && registered == 0 {
		return fmt.Errorf("no transformers found for resources: %v in migration %s", resourceFilter, migrationKey)
	}

	return nil
}

// GetResourceByType returns the definition for a specific resource type and migration version
func GetResourceByType(resourceType string, migrationKey string) (*ResourceDefinition, bool) {
	for _, def := range GetAllResources(migrationKey) {
		if def.Type == resourceType {
			return &def, true
		}
	}
	return nil, false
}

// GetResourceTypes returns a list of all supported resource types for a migration version
func GetResourceTypes(migrationKey string) []string {
	resources := GetAllResources(migrationKey)
	types := make([]string, len(resources))
	for i, r := range resources {
		types[i] = r.Type
	}
	return types
}

// GetDeprecatedResources returns all deprecated resources for a migration version
func GetDeprecatedResources(migrationKey string) []ResourceDefinition {
	var deprecated []ResourceDefinition
	for _, def := range GetAllResources(migrationKey) {
		if def.Deprecated {
			deprecated = append(deprecated, def)
		}
	}
	return deprecated
}
