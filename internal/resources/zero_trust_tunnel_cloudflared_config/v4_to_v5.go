package zero_trust_tunnel_cloudflared_config

import (
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles migration of zero trust tunnel cloudflared config resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register BOTH v4 resource names (deprecated and preferred)
	internal.RegisterMigrator("cloudflare_tunnel_config", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_tunnel_cloudflared_config", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name (same as preferred v4 name)
	return "cloudflare_zero_trust_tunnel_cloudflared_config"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both the deprecated name and the preferred v4 name
	return resourceType == "cloudflare_tunnel_config" || resourceType == "cloudflare_zero_trust_tunnel_cloudflared_config"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - HCL parser can handle all transformations
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_tunnel_config", "cloudflare_zero_trust_tunnel_cloudflared_config"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	resourceType := tfhcl.GetResourceType(block)

	// Rename resource type if using deprecated name
	if resourceType == "cloudflare_tunnel_config" {
		tfhcl.RenameResourceType(block, "cloudflare_tunnel_config", "cloudflare_zero_trust_tunnel_cloudflared_config")
	}

	body := block.Body()

	// Process config block if it exists
	configBlocks := tfhcl.FindBlocksByType(body, "config")
	for _, configBlock := range configBlocks {
		configBody := configBlock.Body()

		// Remove deprecated blocks that were removed in v5
		tfhcl.RemoveBlocksByType(configBody, "warp_routing")

		// Remove ip_rules from all origin_request blocks
		// (at config level)
		originReqBlocks := tfhcl.FindBlocksByType(configBody, "origin_request")
		for _, originReqBlock := range originReqBlocks {
			tfhcl.RemoveBlocksByType(originReqBlock.Body(), "ip_rules")
		}

		// Remove ip_rules from nested origin_request blocks within ingress_rule
		ingressBlocks := tfhcl.FindBlocksByType(configBody, "ingress_rule")
		for _, ingressBlock := range ingressBlocks {
			nestedOriginReqBlocks := tfhcl.FindBlocksByType(ingressBlock.Body(), "origin_request")
			for _, nestedOriginReqBlock := range nestedOriginReqBlocks {
				tfhcl.RemoveBlocksByType(nestedOriginReqBlock.Body(), "ip_rules")
			}
		}
	}

	// Note: Additional v4→v5 schema changes (MaxItems:1 block→attribute conversions,
	// ingress_rule→ingress rename) are handled in the state transformation.
	// HCL config transformation focuses on removing deprecated fields to ensure
	// configs remain valid. Users can update block→attribute syntax using
	// terraform fmt or the v5 provider will handle it.

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, instance gjson.Result, resourcePath, resourceName string) (string, error) {
	result := instance.String()
	attrs := instance.Get("attributes")

	if !attrs.Exists() {
		return result, nil
	}

	// Always use relative path within the instance JSON (not the full state path)
	path := "attributes"

	// 1. Transform config array [{}] → object {}
	result = state.TransformFieldArrayToObject(result, path, attrs, "config", state.ArrayToObjectOptions{})

	// Get the transformed config to work with (re-parse from root)
	resultParsed := gjson.Parse(result)
	attrs = resultParsed.Get(path)
	configObj := attrs.Get("config")

	if configObj.Exists() && configObj.IsObject() {
		configPath := path + ".config"

		// 2. Rename ingress_rule → ingress inside config
		result = state.RenameField(result, configPath, configObj, "ingress_rule", "ingress")

		// 3. Remove deprecated fields from config
		result = state.RemoveFields(result, configPath, configObj, "warp_routing")

		// Refresh config after changes
		configObj = gjson.Parse(result).Get(configPath)

		// 4. Transform config-level origin_request array → object
		originReqPath := configPath + ".origin_request"
		result = transformOriginRequest(result, originReqPath, configObj.Get("origin_request"))

		// 5. Transform ingress array elements' origin_request
		ingressArray := gjson.Parse(result).Get(configPath + ".ingress")
		if ingressArray.Exists() && ingressArray.IsArray() {
			for i, ingressItem := range ingressArray.Array() {
				originReq := ingressItem.Get("origin_request")
				if originReq.Exists() {
					ingressOriginReqPath := fmt.Sprintf("%s.ingress.%d.origin_request", configPath, i)
					result = transformOriginRequest(result, ingressOriginReqPath, originReq)
				}
			}
		}
	}

	// Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	// Update the type field if it exists (for unit tests that pass instance-level type)
	if instance.Get("type").Exists() {
		result, _ = sjson.Set(result, "type", "cloudflare_zero_trust_tunnel_cloudflared_config")
	}

	return result, nil
}

// transformOriginRequest transforms an origin_request field from array to object and handles nested transformations
func transformOriginRequest(stateJSON string, path string, originReq gjson.Result) string {
	if !originReq.Exists() {
		return stateJSON
	}

	// If it's an array, convert to object first
	if originReq.IsArray() {
		array := originReq.Array()
		if len(array) > 0 {
			stateJSON, _ = sjson.Set(stateJSON, path, array[0].Value())
		} else {
			// Empty array - delete it
			stateJSON, _ = sjson.Delete(stateJSON, path)
			return stateJSON
		}
		// Refresh origin_request after conversion
		originReq = gjson.Parse(stateJSON).Get(path)
	}

	if !originReq.IsObject() {
		return stateJSON
	}

	// Remove deprecated fields
	stateJSON = state.RemoveFields(stateJSON, path, originReq, "bastion_mode", "proxy_address", "proxy_port", "ip_rules")

	// Refresh after removals
	originReq = gjson.Parse(stateJSON).Get(path)

	// Convert duration fields from strings to int64 nanoseconds
	durationFields := []string{"connect_timeout", "tls_timeout", "tcp_keep_alive", "keep_alive_timeout"}
	for _, field := range durationFields {
		fieldValue := originReq.Get(field)
		if fieldValue.Exists() && fieldValue.Type == gjson.String {
			if nanos, err := parseDurationToNanoseconds(fieldValue.String()); err == nil {
				stateJSON, _ = sjson.Set(stateJSON, path+"."+field, nanos)
			}
		}
	}

	// Convert keep_alive_connections from int to int64
	keepAliveConns := originReq.Get("keep_alive_connections")
	if keepAliveConns.Exists() {
		stateJSON, _ = sjson.Set(stateJSON, path+".keep_alive_connections", state.ConvertToInt64(keepAliveConns))
	}

	// Refresh origin_request after duration conversions
	originReq = gjson.Parse(stateJSON).Get(path)

	// Transform nested access array → object
	accessField := originReq.Get("access")
	if accessField.Exists() {
		stateJSON = state.TransformFieldArrayToObject(stateJSON, path, originReq, "access", state.ArrayToObjectOptions{})
	}

	return stateJSON
}

// parseDurationToNanoseconds converts a Go duration string (e.g., "30s", "1m30s") to int64 nanoseconds
func parseDurationToNanoseconds(durationStr string) (int64, error) {
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration %q: %w", durationStr, err)
	}
	return duration.Nanoseconds(), nil
}
