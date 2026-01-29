package registry

import (
	lbpoolsdata "github.com/cloudflare/tf-migrate/internal/datasources/load_balancer_pools"
	rulesetsdata "github.com/cloudflare/tf-migrate/internal/datasources/rulesets"
	zonedata "github.com/cloudflare/tf-migrate/internal/datasources/zone"
	zonesdata "github.com/cloudflare/tf-migrate/internal/datasources/zones"
	"github.com/cloudflare/tf-migrate/internal/resources/access_rule"
	"github.com/cloudflare/tf-migrate/internal/resources/account_member"
	"github.com/cloudflare/tf-migrate/internal/resources/api_shield"
	"github.com/cloudflare/tf-migrate/internal/resources/api_token"
	"github.com/cloudflare/tf-migrate/internal/resources/argo"
	"github.com/cloudflare/tf-migrate/internal/resources/bot_management"
	"github.com/cloudflare/tf-migrate/internal/resources/certificate_pack"
	"github.com/cloudflare/tf-migrate/internal/resources/custom_pages"
	"github.com/cloudflare/tf-migrate/internal/resources/dns_record"
	"github.com/cloudflare/tf-migrate/internal/resources/healthcheck"
	"github.com/cloudflare/tf-migrate/internal/resources/leaked_credential_check"
	"github.com/cloudflare/tf-migrate/internal/resources/leaked_credential_check_rule"
	"github.com/cloudflare/tf-migrate/internal/resources/list"
	"github.com/cloudflare/tf-migrate/internal/resources/list_item"
	"github.com/cloudflare/tf-migrate/internal/resources/load_balancer"
	"github.com/cloudflare/tf-migrate/internal/resources/load_balancer_monitor"
	"github.com/cloudflare/tf-migrate/internal/resources/load_balancer_pool"
	"github.com/cloudflare/tf-migrate/internal/resources/logpull_retention"
	"github.com/cloudflare/tf-migrate/internal/resources/logpush_job"
	"github.com/cloudflare/tf-migrate/internal/resources/logpush_ownership_challenge"
	"github.com/cloudflare/tf-migrate/internal/resources/managed_transforms"
	"github.com/cloudflare/tf-migrate/internal/resources/mtls_certificate"
	"github.com/cloudflare/tf-migrate/internal/resources/notification_policy"
	"github.com/cloudflare/tf-migrate/internal/resources/notification_policy_webhooks"
	"github.com/cloudflare/tf-migrate/internal/resources/page_rule"
	"github.com/cloudflare/tf-migrate/internal/resources/pages_project"
	"github.com/cloudflare/tf-migrate/internal/resources/queue"
	"github.com/cloudflare/tf-migrate/internal/resources/r2_bucket"
	"github.com/cloudflare/tf-migrate/internal/resources/regional_hostname"
	"github.com/cloudflare/tf-migrate/internal/resources/regional_tiered_cache"
	"github.com/cloudflare/tf-migrate/internal/resources/ruleset"
	"github.com/cloudflare/tf-migrate/internal/resources/snippet"
	"github.com/cloudflare/tf-migrate/internal/resources/spectrum_application"
	"github.com/cloudflare/tf-migrate/internal/resources/tiered_cache"
	"github.com/cloudflare/tf-migrate/internal/resources/turnstile_widget"
	"github.com/cloudflare/tf-migrate/internal/resources/url_normalization_settings"
	"github.com/cloudflare/tf-migrate/internal/resources/worker_route"
	"github.com/cloudflare/tf-migrate/internal/resources/workers_for_platforms_dispatch_namespace"
	"github.com/cloudflare/tf-migrate/internal/resources/workers_kv"
	"github.com/cloudflare/tf-migrate/internal/resources/workers_kv_namespace"
	"github.com/cloudflare/tf-migrate/internal/resources/workers_script"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_access_application"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_access_group"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_access_identity_provider"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_access_mtls_certificate"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_access_mtls_hostname_settings"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_access_policy"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_access_service_token"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_device_default_profile"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_device_posture_integration"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_device_managed_networks"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_device_posture_rule"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_split_tunnel"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_dex_test"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_dlp_custom_profile"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_gateway_certificate"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_gateway_policy"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_gateway_settings"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_list"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_local_fallback_domain"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_organization"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_tunnel_cloudflared"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_tunnel_cloudflared_config"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_tunnel_cloudflared_route"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_tunnel_cloudflared_virtual_network"
	"github.com/cloudflare/tf-migrate/internal/resources/zone"
	"github.com/cloudflare/tf-migrate/internal/resources/zone_dnssec"
	"github.com/cloudflare/tf-migrate/internal/resources/zone_setting"
)

// RegisterAllMigrations registers all resource migrations with the internal registry.
// This function should be called once during initialization to set up all available migrations.
// Each resource package's NewV4ToV5Migrator function registers itself with the internal registry.
func RegisterAllMigrations() {
	// Datasources
	lbpoolsdata.NewV4ToV5Migrator()
	rulesetsdata.NewV4ToV5Migrator()
	zonedata.NewV4ToV5Migrator()
	zonesdata.NewV4ToV5Migrator()

	// Resources
	access_rule.NewV4ToV5Migrator()
	account_member.NewV4ToV5Migrator()
	api_shield.NewV4ToV5Migrator()
	api_token.NewV4ToV5Migrator()
	argo.NewV4ToV5Migrator()
	bot_management.NewV4ToV5Migrator()
	certificate_pack.NewV4ToV5Migrator()
	custom_pages.NewV4ToV5Migrator()
	dns_record.NewV4ToV5Migrator()
	healthcheck.NewV4ToV5Migrator()
	leaked_credential_check.NewV4ToV5Migrator()
	leaked_credential_check_rule.NewV4ToV5Migrator()
	load_balancer.NewV4ToV5Migrator()
	load_balancer_monitor.NewV4ToV5Migrator()
	load_balancer_pool.NewV4ToV5Migrator()
	list.NewV4ToV5Migrator()
	list_item.NewV4ToV5Migrator()
	zone.NewV4ToV5Migrator()
	zone_dnssec.NewV4ToV5Migrator()
	zone_setting.NewV4ToV5Migrator()
	logpull_retention.NewV4ToV5Migrator()
	logpush_job.NewV4ToV5Migrator()
	logpush_ownership_challenge.NewV4ToV5Migrator()
	managed_transforms.NewV4ToV5Migrator()
	mtls_certificate.NewV4ToV5Migrator()
	notification_policy.NewV4ToV5Migrator()
	notification_policy_webhooks.NewV4ToV5Migrator()
	page_rule.NewV4ToV5Migrator()
	pages_project.NewV4ToV5Migrator()
	queue.NewV4ToV5Migrator()
	r2_bucket.NewV4ToV5Migrator()
	regional_hostname.NewV4ToV5Migrator()
	regional_tiered_cache.NewV4ToV5Migrator()
	ruleset.NewV4ToV5Migrator()
	snippet.NewV4ToV5Migrator()
	tiered_cache.NewV4ToV5Migrator()
	spectrum_application.NewV4ToV5Migrator()
	turnstile_widget.NewV4ToV5Migrator()
	url_normalization_settings.NewV4ToV5Migrator()
	worker_route.NewV4ToV5Migrator()
	workers_kv.NewV4ToV5Migrator()
	workers_kv_namespace.NewV4ToV5Migrator()
	workers_script.NewV4ToV5Migrator()
	workers_for_platforms_dispatch_namespace.NewV4ToV5Migrator()
	zero_trust_access_application.NewV4ToV5Migrator()
	zero_trust_access_group.NewV4ToV5Migrator()
	zero_trust_access_identity_provider.NewV4ToV5Migrator()
	zero_trust_access_mtls_certificate.NewV4ToV5Migrator()
	zero_trust_access_mtls_hostname_settings.NewV4ToV5Migrator()
	zero_trust_access_policy.NewV4ToV5Migrator()
	zero_trust_access_service_token.NewV4ToV5Migrator()
	zero_trust_device_default_profile.NewV4ToV5Migrator()
	zero_trust_device_managed_networks.NewV4ToV5Migrator()
	zero_trust_device_posture_integration.NewV4ToV5Migrator()
	zero_trust_device_posture_rule.NewV4ToV5Migrator()
	zero_trust_split_tunnel.NewV4ToV5Migrator()
	zero_trust_dex_test.NewV4ToV5Migrator()
	zero_trust_dlp_custom_profile.NewV4ToV5Migrator()
	zero_trust_gateway_certificate.NewV4ToV5Migrator()
	zero_trust_gateway_policy.NewV4ToV5Migrator()
	zero_trust_gateway_settings.NewV4ToV5Migrator()
	zero_trust_list.NewV4ToV5Migrator()
	zero_trust_local_fallback_domain.NewV4ToV5Migrator()
	zero_trust_organization.NewV4ToV5Migrator()
	zero_trust_tunnel_cloudflared.NewV4ToV5Migrator()
	zero_trust_tunnel_cloudflared_config.NewV4ToV5Migrator()
	zero_trust_tunnel_cloudflared_route.NewV4ToV5Migrator()
	zero_trust_tunnel_cloudflared_virtual_network.NewV4ToV5Migrator()
}
