package dns_record

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

// init registers the DNS record migrators
func init() {
	// Register v4 to v5 migrator with explicit version
	internal.Register("cloudflare_dns_record", "v4", "v5", func() transform.ResourceTransformer {
		return NewV4ToV5Migrator()
	})

	// Also register for the legacy cloudflare_record type (v4 to v5)
	internal.Register("cloudflare_record", "v4", "v5", func() transform.ResourceTransformer {
		return NewV4ToV5Migrator()
	})

	// Example: Future v5 to v6 migration would be registered like this:
	// internal.RegisterVersioned("cloudflare_dns_record", "v5", "v6", func() transform.ResourceTransformer {
	//     return NewV5ToV6Migrator()
	// })
}

// GetMigrator returns the appropriate migrator based on source and target versions
func GetMigrator(sourceVersion, targetVersion string) transform.ResourceTransformer {
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
