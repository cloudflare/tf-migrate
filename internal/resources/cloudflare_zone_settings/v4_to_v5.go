package cloudflare_zone_settings

import (
	"github.com/cloudflare/tf-migrate/internal/factory"
	"github.com/cloudflare/tf-migrate/internal/core/transform"
)

// V4ToV5 returns the transformer for migrating cloudflare_zone_settings from v4 to v5
func V4ToV5() transform.Transformer {
	// In v4 to v5 migration:
	// - Some attributes were renamed for clarity
	return factory.Default.RenameAttributes(
		"cloudflare_zone_settings",
		map[string]string{
			"mobile_redirect": "mobile_optimization",
			"waf":            "web_application_firewall",
			"ssl":            "ssl_mode",
		},
	)
}