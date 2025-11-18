package registry

import (
	"github.com/cloudflare/tf-migrate/internal/resources/account_member"
	"github.com/cloudflare/tf-migrate/internal/resources/api_token"
	"github.com/cloudflare/tf-migrate/internal/resources/dns_record"
	"github.com/cloudflare/tf-migrate/internal/resources/logpull_retention"
	"github.com/cloudflare/tf-migrate/internal/resources/notification_policy_webhooks"
	"github.com/cloudflare/tf-migrate/internal/resources/r2_bucket"
	"github.com/cloudflare/tf-migrate/internal/resources/regional_hostname"
	"github.com/cloudflare/tf-migrate/internal/resources/workers_kv"
	"github.com/cloudflare/tf-migrate/internal/resources/workers_kv_namespace"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_access_service_token"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_dlp_custom_profile"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_gateway_policy"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_device_posture_rule"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_list"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_tunnel_cloudflared"
	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_tunnel_cloudflared_route"
	"github.com/cloudflare/tf-migrate/internal/resources/zone_dnssec"
)

// RegisterAllMigrations registers all resource migrations with the internal registry.
// This function should be called once during initialization to set up all available migrations.
// Each resource package's NewV4ToV5Migrator function registers itself with the internal registry.
func RegisterAllMigrations() {
	account_member.NewV4ToV5Migrator()
	api_token.NewV4ToV5Migrator()
	dns_record.NewV4ToV5Migrator()
	zone_dnssec.NewV4ToV5Migrator()
	logpull_retention.NewV4ToV5Migrator()
	notification_policy_webhooks.NewV4ToV5Migrator()
	r2_bucket.NewV4ToV5Migrator()
	regional_hostname.NewV4ToV5Migrator()
	workers_kv.NewV4ToV5Migrator()
	workers_kv_namespace.NewV4ToV5Migrator()
	zero_trust_access_service_token.NewV4ToV5Migrator()
	zero_trust_dlp_custom_profile.NewV4ToV5Migrator()
	zero_trust_gateway_policy.NewV4ToV5Migrator()
	zero_trust_device_posture_rule.NewV4ToV5Migrator()
	zero_trust_list.NewV4ToV5Migrator()
	zero_trust_tunnel_cloudflared.NewV4ToV5Migrator()
	zero_trust_tunnel_cloudflared_route.NewV4ToV5Migrator()
}
