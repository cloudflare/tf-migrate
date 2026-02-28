package e2e

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// PhaseResources maps phase numbers to their resource lists.
// Each phase represents a group of resources that can be tested together.
// Resource names must match testdata directory names under integration/v4_to_v5/testdata/.
// Use --phase 0 or --phase 0,1 to run one or more phases.
var PhaseResources = map[int][]string{
	0: {
		"argo",
		"bot_management",
		"custom_pages",
		"dns_record",
		"list",
		"load_balancer",
		"load_balancer_monitor",
		"load_balancer_pool",
		"managed_transforms",
		"page_rule",
		"regional_hostname",
		"ruleset",
		"snippet",
		"snippet_rules",
		"spectrum_application",
		"tiered_cache",
		"url_normalization_settings",
		"worker_route",
		"workers_script",
		"zero_trust_access_application",
		"zero_trust_access_group",
		"zero_trust_access_identity_provider",
		"zero_trust_access_mtls_certificate",
		"zero_trust_access_mtls_hostname_settings",
		"zero_trust_access_policy",
		"zone",
		"zone_setting",
	},
	1: {
		"api_token",
		"healthcheck",
		"logpull_retention",
		"logpush_job",
		"notification_policy_webhooks",
		"pages_project",
		"r2_bucket",
		"workers_kv",
		"workers_kv_namespace",
		"zero_trust_access_service_token",
		"zero_trust_device_posture_rule",
		"zero_trust_dlp_custom_profile",
		"zero_trust_dlp_predefined_profile",
		"zero_trust_gateway_policy",
		"zero_trust_list",
		"zero_trust_tunnel_cloudflared_route",
		"zone_datasource",
		"zone_dnssec",
		"zones_datasource",
	},
}

// ResolvePhases takes a comma-separated string of phase numbers (e.g., "0,1")
// and returns the deduplicated, sorted list of resources for those phases.
func ResolvePhases(phases string) ([]string, error) {
	if phases == "" {
		return nil, fmt.Errorf("no phases specified")
	}

	seen := make(map[string]bool)
	var result []string

	for _, p := range strings.Split(phases, ",") {
		p = strings.TrimSpace(p)
		phaseNum, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid phase number %q: must be an integer", p)
		}

		resources, ok := PhaseResources[phaseNum]
		if !ok {
			var available []string
			for k := range PhaseResources {
				available = append(available, strconv.Itoa(k))
			}
			sort.Strings(available)
			return nil, fmt.Errorf("unknown phase %d; available phases: %s", phaseNum, strings.Join(available, ", "))
		}

		for _, r := range resources {
			if !seen[r] {
				seen[r] = true
				result = append(result, r)
			}
		}
	}

	sort.Strings(result)
	return result, nil
}
