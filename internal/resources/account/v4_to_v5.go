package account

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
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
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_account"}, "cloudflare_account"
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

