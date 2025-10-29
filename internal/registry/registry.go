package registry

import (
	"github.com/cloudflare/tf-migrate/internal/resources/dns_record"
	"github.com/cloudflare/tf-migrate/internal/resources/zone_dnssec"
)

// RegisterAllMigrations registers all resource migrations with the internal registry.
// This function should be called once during initialization to set up all available migrations.
// Each resource package's NewV4ToV5Migrator function registers itself with the internal registry.
func RegisterAllMigrations() {
	dns_record.NewV4ToV5Migrator()
	zone_dnssec.NewV4ToV5Migrator()
}
