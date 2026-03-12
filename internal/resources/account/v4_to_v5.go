package account

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
)

type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_account", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_account"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_account"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface.
// cloudflare_account is not renamed between v4 and v5.
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_account", "cloudflare_account"
}

// TransformConfig handles HCL configuration transformations for cloudflare_account.
//
// v4 → v5 changes:
//   - enforce_twofactor moves from top-level into a settings nested object
//
// Example:
//
//	Before (v4):
//	  resource "cloudflare_account" "example" {
//	    name              = "My Account"
//	    enforce_twofactor = true
//	  }
//
//	After (v5):
//	  resource "cloudflare_account" "example" {
//	    name = "My Account"
//	    settings = {
//	      enforce_twofactor = true
//	    }
//	  }
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Move enforce_twofactor into settings nested object
	tfhcl.MoveAttributesToNestedObject(body, "settings", []string{"enforce_twofactor"})

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// TransformState returns the state unchanged (no-op).
//
// State migration is now handled by the provider's StateUpgrader (v5.19+).
// The provider implements UpgradeState with slot 0 handling v4 SDKv2 state
// (schema_version=0) and transforming enforce_twofactor into settings.enforce_twofactor.
//
// tf-migrate only needs to transform the HCL configuration; Terraform will
// invoke the provider's state upgrader when it detects schema_version mismatch.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// No-op: Provider's StateUpgrader handles v4→v5 state transformation
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration.
// This tells tf-migrate that the provider handles state transformation, not tf-migrate.
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
