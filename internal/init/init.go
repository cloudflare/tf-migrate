package init

// Import all resource packages to trigger their init() functions
import (
	_ "github.com/cloudflare/tf-migrate/internal/resources/dns_record"
	// Future resources will be added here:
	// _ "github.com/cloudflare/tf-migrate/internal/resources/load_balancer"
	// _ "github.com/cloudflare/tf-migrate/internal/resources/zone_settings"
	// etc.
)