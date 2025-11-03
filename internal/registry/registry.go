package registry

import (
	"github.com/cloudflare/tf-migrate/internal/resources/access_application"
	"github.com/cloudflare/tf-migrate/internal/resources/access_policy"
	"github.com/cloudflare/tf-migrate/internal/resources/account_member"
	"github.com/cloudflare/tf-migrate/internal/resources/api_token"
	"github.com/cloudflare/tf-migrate/internal/resources/argo"
	"github.com/cloudflare/tf-migrate/internal/resources/cloudflare_ruleset"
	"github.com/cloudflare/tf-migrate/internal/resources/dns_record"
	"github.com/cloudflare/tf-migrate/internal/resources/list"
	"github.com/cloudflare/tf-migrate/internal/resources/load_balancer"
	"github.com/cloudflare/tf-migrate/internal/resources/load_balancer_monitor"
	"github.com/cloudflare/tf-migrate/internal/resources/load_balancer_pool"
	"github.com/cloudflare/tf-migrate/internal/resources/managed_transforms"
	"github.com/cloudflare/tf-migrate/internal/resources/page_rule"
	"github.com/cloudflare/tf-migrate/internal/resources/regional_hostname"
	"github.com/cloudflare/tf-migrate/internal/resources/snippet"
	"github.com/cloudflare/tf-migrate/internal/resources/snippet_rules"
	"github.com/cloudflare/tf-migrate/internal/resources/spectrum_application"
	"github.com/cloudflare/tf-migrate/internal/resources/tiered_cache"
	"github.com/cloudflare/tf-migrate/internal/resources/workers_cron_trigger"
	"github.com/cloudflare/tf-migrate/internal/resources/workers_domain"
	"github.com/cloudflare/tf-migrate/internal/resources/workers_route"
	"github.com/cloudflare/tf-migrate/internal/resources/workers_script"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_access_group"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_access_identity_provider"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_access_mtls_hostname_settings"
	"github.com/cloudflare/tf-migrate/internal/resources/zone"
	"github.com/cloudflare/tf-migrate/internal/resources/zone_settings"
)

// RegisterAllMigrations registers all resource migrations with the internal registry.
// This function should be called once during initialization to set up all available migrations.
// Each resource package has its own RegisterMigrations function that defines how to register
// its specific migrations.
func RegisterAllMigrations() {
	access_application.NewV4ToV5Migrator()
	access_policy.NewV4ToV5Migrator()
	account_member.NewV4ToV5Migrator()
	api_token.NewV4ToV5Migrator()
	argo.NewV4ToV5Migrator()
	cloudflare_ruleset.NewV4ToV5Migrator()
	dns_record.NewV4ToV5Migrator()
	list.NewV4ToV5Migrator()
	load_balancer.NewV4ToV5Migrator()
	load_balancer_monitor.NewV4ToV5Migrator()
	load_balancer_pool.NewV4ToV5Migrator()
	managed_transforms.NewV4ToV5Migrator()
	page_rule.NewV4ToV5Migrator()
	regional_hostname.NewV4ToV5Migrator()
	snippet.NewV4ToV5Migrator()
	snippet_rules.NewV4ToV5Migrator()
	spectrum_application.NewV4ToV5Migrator()
	tiered_cache.NewV4ToV5Migrator()
	workers_cron_trigger.NewV4ToV5Migrator()
	workers_domain.NewV4ToV5Migrator()
	workers_route.NewV4ToV5Migrator()
	workers_script.NewV4ToV5Migrator()
	zero_trust_access_group.NewV4ToV5Migrator()
	zero_trust_access_identity_provider.NewV4ToV5Migrator()
	zero_trust_access_mtls_hostname_settings.NewV4ToV5Migrator()
	zone.NewV4ToV5Migrator()
	zone_settings.NewV4ToV5Migrator()
}
