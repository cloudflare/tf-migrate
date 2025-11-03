package v4_to_v5

import (
	"fmt"
	"os"
	"testing"

	"github.com/cloudflare/tf-migrate/integration"
	"github.com/cloudflare/tf-migrate/internal/registry"

	// Explicitly import the migrations we want to test
	_ "github.com/cloudflare/tf-migrate/internal/resources/access_application"
	_ "github.com/cloudflare/tf-migrate/internal/resources/access_policy"
	_ "github.com/cloudflare/tf-migrate/internal/resources/account_member"
	_ "github.com/cloudflare/tf-migrate/internal/resources/api_token"
	_ "github.com/cloudflare/tf-migrate/internal/resources/argo"
	_ "github.com/cloudflare/tf-migrate/internal/resources/cloudflare_ruleset"
	_ "github.com/cloudflare/tf-migrate/internal/resources/dns_record"
	_ "github.com/cloudflare/tf-migrate/internal/resources/list"
	_ "github.com/cloudflare/tf-migrate/internal/resources/load_balancer"
	_ "github.com/cloudflare/tf-migrate/internal/resources/load_balancer_monitor"
	_ "github.com/cloudflare/tf-migrate/internal/resources/load_balancer_pool"
	_ "github.com/cloudflare/tf-migrate/internal/resources/managed_transforms"
	_ "github.com/cloudflare/tf-migrate/internal/resources/page_rule"
	_ "github.com/cloudflare/tf-migrate/internal/resources/regional_hostname"
	_ "github.com/cloudflare/tf-migrate/internal/resources/snippet"
	_ "github.com/cloudflare/tf-migrate/internal/resources/snippet_rules"
	_ "github.com/cloudflare/tf-migrate/internal/resources/spectrum_application"
	_ "github.com/cloudflare/tf-migrate/internal/resources/tiered_cache"
	_ "github.com/cloudflare/tf-migrate/internal/resources/workers_cron_trigger"
	_ "github.com/cloudflare/tf-migrate/internal/resources/workers_domain"
	_ "github.com/cloudflare/tf-migrate/internal/resources/workers_route"
	_ "github.com/cloudflare/tf-migrate/internal/resources/workers_script"
	_ "github.com/cloudflare/tf-migrate/internal/resources/zero_trust_access_group"
	_ "github.com/cloudflare/tf-migrate/internal/resources/zero_trust_access_identity_provider"
	_ "github.com/cloudflare/tf-migrate/internal/resources/zero_trust_access_mtls_hostname_settings"
	_ "github.com/cloudflare/tf-migrate/internal/resources/zone"
	_ "github.com/cloudflare/tf-migrate/internal/resources/zone_settings"
)

// TestMain explicitly registers migrations for this version path
func TestMain(m *testing.M) {
	// Explicitly register the migrations for v4 to v5
	// This is called once before all tests in this package
	registry.RegisterAllMigrations()

	// Run the tests
	code := m.Run()
	os.Exit(code)
}

func TestV4ToV5Migration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	runner, err := integration.NewTestRunner("v4", "v5")
	if err != nil {
		t.Fatalf("Failed to create test runner: %v", err)
	}

	tests := []integration.TestCase{
		{
			Name:        "AccessApplication",
			Description: "Migrate cloudflare_access_application policies list from IDs to objects",
			Resource:    "access_application",
		},
		{
			Name:        "AccessPolicy",
			Description: "Migrate cloudflare_access_policy with array expansion and boolean-to-object transformations",
			Resource:    "access_policy",
		},
		{
			Name:        "AccountMember",
			Description: "Migrate cloudflare_account_member email_address to email and role_ids to roles",
			Resource:    "account_member",
		},
		{
			Name:        "APIToken",
			Description: "Migrate cloudflare_api_token policy blocks to policies list",
			Resource:    "api_token",
		},
		{
			Name:        "Argo",
			Description: "Migrate cloudflare_argo to cloudflare_argo_smart_routing and cloudflare_argo_tiered_caching",
			Resource:    "argo",
		},
		{
			Name:        "CloudflareRuleset",
			Description: "Transform cloudflare_ruleset from indexed to array state format",
			Resource:    "cloudflare_ruleset",
		},
		{
			Name:        "DNSRecord",
			Description: "Migrate cloudflare_record to cloudflare_dns_record",
			Resource:    "dns_record",
		},
		{
			Name:        "List",
			Description: "Transform cloudflare_list item flattening and boolean string conversions",
			Resource:    "list",
		},
		{
			Name:        "LoadBalancer",
			Description: "Transform cloudflare_load_balancer state field renames",
			Resource:    "load_balancer",
		},
		{
			Name:        "LoadBalancerMonitor",
			Description: "Transform header array to map format in load balancer monitor",
			Resource:    "load_balancer_monitor",
		},
		{
			Name:        "LoadBalancerPool",
			Description: "Transform cloudflare_load_balancer_pool dynamic origins to for expressions",
			Resource:    "load_balancer_pool",
		},
		{
			Name:        "ManagedTransforms",
			Description: "Ensure both managed_request_headers and managed_response_headers exist",
			Resource:    "managed_transforms",
		},
		{
			Name:        "PageRule",
			Description: "No transformation for cloudflare_page_rule",
			Resource:    "page_rule",
		},
		{
			Name:        "RegionalHostname",
			Description: "Remove timeouts blocks from cloudflare_regional_hostname",
			Resource:    "regional_hostname",
		},
		{
			Name:        "Snippet",
			Description: "No transformation for cloudflare_snippet",
			Resource:    "snippet",
		},
		{
			Name:        "SnippetRules",
			Description: "No transformation for cloudflare_snippet_rules",
			Resource:    "snippet_rules",
		},
		{
			Name:        "SpectrumApplication",
			Description: "No transformation for cloudflare_spectrum_application",
			Resource:    "spectrum_application",
		},
		{
			Name:        "TieredCache",
			Description: "Transform cache_type to value and convert generic to argo_tiered_caching",
			Resource:    "tiered_cache",
		},
		{
			Name:        "WorkersCronTrigger",
			Description: "Rename cloudflare_worker_cron_trigger to cloudflare_workers_cron_trigger",
			Resource:    "workers_cron_trigger",
		},
		{
			Name:        "WorkersDomain",
			Description: "Rename cloudflare_worker_domain to cloudflare_workers_custom_domain",
			Resource:    "workers_domain",
		},
		{
			Name:        "WorkersRoute",
			Description: "Rename cloudflare_worker_route to cloudflare_workers_route and script_name to script",
			Resource:    "workers_route",
		},
		{
			Name:        "WorkersScript",
			Description: "Transform cloudflare_workers_script binding blocks to unified bindings list",
			Resource:    "workers_script",
		},
		{
			Name:        "ZeroTrustAccessGroup",
			Description: "Transform cloudflare_access_group with array expansion and boolean-to-object conversions",
			Resource:    "zero_trust_access_group",
		},
		{
			Name:        "ZeroTrustAccessIdentityProvider",
			Description: "No transformation for cloudflare_access_identity_provider",
			Resource:    "zero_trust_access_identity_provider",
		},
		{
			Name:        "ZeroTrustAccessMTLSHostnameSettings",
			Description: "Transform cloudflare_access_mtls_hostname_settings dynamic blocks to for expressions",
			Resource:    "zero_trust_access_mtls_hostname_settings",
		},
		{
			Name:        "Zone",
			Description: "Transform zone to name, account_id to account object, and remove jump_start and plan",
			Resource:    "zone",
		},
		{
			Name:        "ZoneSettings",
			Description: "No transformation for cloudflare_zone_settings_override",
			Resource:    "zone_settings",
		},
	}

	for _, test := range tests {
		runner.RunTest(t, test)
	}
}

// TestSingleResource allows testing a specific resource during development
func TestSingleResource(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Set environment variable TEST_RESOURCE to test a specific resource
	// e.g., TEST_RESOURCE=dns_record go test ./integration/v4_to_v5
	resource := os.Getenv("TEST_RESOURCE")
	if resource == "" {
		t.Skip("TEST_RESOURCE not set, skipping single resource test")
	}

	// For v4_to_v5 tests, we always use version v4 to v5
	runner, err := integration.NewTestRunner("v4", "v5")
	if err != nil {
		t.Fatalf("Failed to create test runner: %v", err)
	}

	test := integration.TestCase{
		Name:        resource,
		Description: fmt.Sprintf("Testing %s migration from v4 to v5", resource),
		Resource:    resource,
	}

	runner.RunTest(t, test)
}
