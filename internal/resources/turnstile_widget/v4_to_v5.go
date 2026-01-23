package turnstile_widget

import (
	"sort"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles the migration of cloudflare_turnstile_widget from v4 to v5.
// Key transformations:
// 1. Config: Remove toset() wrapper from domains field (Set → List)
// 2. State: Set schema_version = 0 (required for all v5 migrations)
// 3. State: Copy id value to sitekey field (required for v5 API calls)
//
// Field changes:
// - domains: SetAttribute in v4 → ListAttribute in v5 (remove toset wrapper)
// - sitekey: NEW in v5 (computed) - MUST be added during migration (copy from id)
// - created_on: NEW in v5 (computed) - don't add during migration
// - modified_on: NEW in v5 (computed) - don't add during migration
// - clearance_level: NEW in v5 (optional computed) - don't add during migration
// - ephemeral_id: NEW in v5 (optional computed) - don't add during migration
// - region: v5 adds Default: "world" - don't add if not in v4
// - offlabel: v4 had Default: false, v5 removed default - preserve if present
type V4ToV5Migrator struct{}

// NewV4ToV5Migrator creates a new migrator for cloudflare_turnstile_widget v4 to v5.
func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register with v4 resource name (same as v5 in this case)
	internal.RegisterMigrator("cloudflare_turnstile_widget", "v4", "v5", migrator)
	return migrator
}

// GetResourceType returns the v5 resource type this migrator handles.
func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_turnstile_widget"
}

// CanHandle determines if this migrator can handle the given resource type.
func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_turnstile_widget"
}

// Preprocess handles string-level transformations before HCL parsing.
// No preprocessing needed for turnstile_widget migration.
func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// cloudflare_turnstile_widget doesn't rename, so return the same name
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_turnstile_widget", "cloudflare_turnstile_widget"
}

// TransformConfig handles configuration file transformations.
// Main transformation: Remove toset() wrapper from domains field (Set → List)
// Also sorts domains alphabetically to match API ordering
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// v4: domains = toset(["example.com", "test.com"])
	// v5: domains = ["example.com", "test.com"]
	// Remove the toset() function wrapper
	tfhcl.RemoveFunctionWrapper(body, "domains", "toset")

	// Sort domains alphabetically to match API ordering
	// The Cloudflare API returns domains in alphabetical order, and since v5 uses
	// ListAttribute (ordered) instead of SetAttribute (unordered), we must sort
	// domains in both config and state to prevent drift
	tfhcl.SortStringArrayAttribute(body, "domains")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState handles state file transformations.
// This function receives a single resource instance and returns the transformed instance JSON.
// Main transformations:
// 1. Set schema_version = 0 (required for all v5 migrations)
// 2. Preserve all existing fields (both Set and List are JSON arrays in state)
// 3. Do NOT add new v5 computed fields (they'll be populated by provider on first read)
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	// Check if it's a valid instance
	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		// Even for invalid instances, set schema_version for v5
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	// Important: In Terraform state JSON, both SetAttribute and ListAttribute
	// are stored as JSON arrays. The difference is only in the config syntax
	// (toset() wrapper). So we don't need to transform the domains field here.

	// Fields preserved as-is:
	// - id (computed in both v4 and v5)
	// - account_id (required in both)
	// - name (required in both)
	// - domains (required in both - already a JSON array)
	// - mode (required in both)
	// - region (optional in both)
	// - bot_fight_mode (optional in both)
	// - offlabel (optional in both)
	// - secret (computed in both)

	// Special case: sitekey is a NEW field in v5, but it's required for API calls
	// In v5, both id and sitekey contain the same value (the site key)
	// Copy id value to sitekey field
	idValue := stateJSON.Get("attributes.id").String()
	if idValue != "" {
		result, _ = sjson.Set(result, "attributes.sitekey", idValue)
	}

	// Sort domains alphabetically to match API ordering (prevents drift)
	// The Cloudflare API returns domains in alphabetical order
	domainsArray := stateJSON.Get("attributes.domains")
	if domainsArray.Exists() && domainsArray.IsArray() {
		domains := []string{} // Initialize as empty slice to preserve empty arrays
		for _, domain := range domainsArray.Array() {
			domains = append(domains, domain.String())
		}
		sort.Strings(domains)
		result, _ = sjson.Set(result, "attributes.domains", domains)
	}

	// Fields NOT added (new in v5, will be populated by provider on first read):
	// - created_on (computed)
	// - modified_on (computed)
	// - clearance_level (optional computed)
	// - ephemeral_id (optional computed)

	// Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}
