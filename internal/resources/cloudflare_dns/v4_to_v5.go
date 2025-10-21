package cloudflare_dns

import (
	"github.com/cloudflare/tf-migrate/internal/factory"
	"github.com/cloudflare/tf-migrate/internal/core/transform"
)

// V4ToV5 returns the transformer for migrating cloudflare_record from v4 to v5
func V4ToV5() transform.Transformer {
	// In v4 to v5 migration:
	// - cloudflare_record becomes cloudflare_dns_record
	// - 'value' attribute is renamed to 'content'
	return factory.Default.RenameAndModifyAttributes(
		"cloudflare_record",
		"cloudflare_dns_record",
		map[string]string{
			"value": "content",
		},
	)
}