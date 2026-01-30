package zero_trust_tunnel_cloudflared_route

import (
	"context"

	"github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/zero_trust"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of zero trust tunnel cloudflared route resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	// Register BOTH v4 resource names (deprecated and preferred)
	internal.RegisterMigrator("cloudflare_tunnel_route", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_tunnel_route", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Return the NEW (v5) resource name
	return "cloudflare_zero_trust_tunnel_cloudflared_route"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	// Handle both the deprecated name and the preferred v4 name
	return resourceType == "cloudflare_tunnel_route" || resourceType == "cloudflare_zero_trust_tunnel_route"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - HCL parser can handle all transformations
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_tunnel_route", "cloudflare_zero_trust_tunnel_cloudflared_route"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	resourceType := tfhcl.GetResourceType(block)

	// Only rename if it's the deprecated cloudflare_tunnel_route
	// cloudflare_zero_trust_tunnel_route only needs the _cloudflared suffix added
	if resourceType == "cloudflare_tunnel_route" {
		tfhcl.RenameResourceType(block, "cloudflare_tunnel_route", "cloudflare_zero_trust_tunnel_cloudflared_route")
	} else if resourceType == "cloudflare_zero_trust_tunnel_route" {
		tfhcl.RenameResourceType(block, "cloudflare_zero_trust_tunnel_route", "cloudflare_zero_trust_tunnel_cloudflared_route")
	}

	// All fields remain the same - no field renames or transformations needed
	// Fields: account_id, tunnel_id, network, comment, virtual_network_id

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	// Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	// Update the type field if it exists (for unit tests that pass instance-level type)
	if stateJSON.Get("type").Exists() {
		result, _ = sjson.Set(result, "type", "cloudflare_zero_trust_tunnel_cloudflared_route")
	}

	// Try to update the ID if API client is available
	// In v4, the ID was the network CIDR (or a checksum of it)
	// In v5, the ID is a UUID from the API
	attrs := stateJSON.Get("attributes")
	if attrs.Exists() && ctx.APIClient != nil {
		accountID := attrs.Get("account_id").String()
		tunnelID := attrs.Get("tunnel_id").String()
		network := attrs.Get("network").String()
		virtualNetworkID := attrs.Get("virtual_network_id").String()

		// Query for the tunnel route using v6 API with pagination support
		params := zero_trust.NetworkRouteListParams{
			AccountID: cloudflare.F(accountID),
			IsDeleted: cloudflare.F(false),
			TunnelID:  cloudflare.F(tunnelID),
		}
		if virtualNetworkID != "" {
			params.VirtualNetworkID = cloudflare.F(virtualNetworkID)
		}

		// Iterate through all pages using AutoPaging iterator
		// fix mee!!!!!!
		iter := ctx.APIClient.ZeroTrust.Networks.Routes.ListAutoPaging(context.Background(), params)
		for iter.Next() {
			route := iter.Current()
			if route.Network == network {
				// Update the ID to the UUID from the API
				result, _ = sjson.Set(result, "attributes.id", route.ID)
				break
			}
		}
	}

	return result, nil
}
