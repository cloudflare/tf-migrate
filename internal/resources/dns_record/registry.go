package dns_record

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// RegisterMigrations registers all DNS record migrations with the internal registry.
// This function is called from internal/resources/register_all.go during initialization.
func RegisterMigrations() {
	// Register v4 to v5 migrator using numeric versions
	internal.Register("cloudflare_dns_record", 4, 5, NewV4ToV5Migrator)
	internal.Register("cloudflare_record", 4, 5, NewV4ToV5Migrator)
}

// GetMigrator returns the appropriate migrator based on source and target versions
func GetMigrator(sourceVersion, targetVersion int) transform.ResourceTransformer {
	// Use the registry to get the appropriate migrator
	migrator := internal.GetMigrator("cloudflare_dns_record", sourceVersion, targetVersion)
	if migrator != nil {
		return migrator
	}

	// Also check for legacy cloudflare_record type
	migrator = internal.GetMigrator("cloudflare_record", sourceVersion, targetVersion)
	if migrator != nil {
		return migrator
	}

	return nil
}
