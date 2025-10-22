package internal

import (
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// Migrators is the explicit map of all available migrators
// Key is the resource type, value is the factory function
// DNS resources
// "cloudflare_record": resources.NewDNSRecordMigrator,
// "cloudflare_zone":   resources.NewDNSZoneMigrator,
// Add new migrators here explicitly
var migrators = make(map[string]func() transform.ResourceTransformer, 0)

// GetMigrator returns a new instance of the migrator for the given resource type
func GetMigrator(resourceType string) transform.ResourceTransformer {
	if factory, ok := migrators[resourceType]; ok {
		return factory()
	}
	return nil
}

// GetAllMigrators returns new instances of all migrators
func GetAllMigrators() []transform.ResourceTransformer {
	result := make([]transform.ResourceTransformer, 0, len(migrators))
	for _, factory := range migrators {
		result = append(result, factory())
	}
	return result
}

func Register(resourceType string, f func() transform.ResourceTransformer) {
	migrators[resourceType] = f
}
