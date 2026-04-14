package account_member

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
	internal.RegisterMigrator("cloudflare_account_member", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_account_member"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_account_member"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This resource does not rename, so we return the same name for both old and new
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_account_member"}, "cloudflare_account_member"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	tfhcl.RenameAttribute(body, "email_address", "email")
	tfhcl.RenameAttribute(body, "role_ids", "roles")

	// Ensure status is set to prevent drift.
	// In v5, if status is not set, the provider defaults to "pending" which can cause
	// unnecessary changes and may fail with "cannot update membership for own user"
	// when the service account tries to modify its own membership.
	// If no status is set, default to "accepted". If status is already set,
	// preserve the original value (e.g., "pending" should remain "pending").
	if body.GetAttribute("status") == nil {
		tfhcl.SetAttribute(body, "status", "accepted")
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}
