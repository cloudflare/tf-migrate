package custom_hostname

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_custom_hostname", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_custom_hostname"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_custom_hostname"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_custom_hostname"}, "cloudflare_custom_hostname"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Removed in v5; provider migration handles state behavior.
	tfhcl.RemoveAttributes(body, "wait_for_ssl_pending_validation")

	// v4: ssl/settings are nested blocks. v5: ssl/settings are nested attributes.
	for _, sslBlock := range body.Blocks() {
		if sslBlock.Type() != "ssl" {
			continue
		}
		sslBody := sslBlock.Body()
		if sslBody.GetAttribute("wildcard") == nil {
			tfhcl.SetAttributeValue(sslBody, "wildcard", false)
		}
		for _, settingsBlock := range sslBody.Blocks() {
			if settingsBlock.Type() != "settings" {
				continue
			}
			tfhcl.RenameAttribute(settingsBlock.Body(), "tls13", "tls_1_3")
		}
		tfhcl.ConvertSingleBlockToAttribute(sslBody, "settings", "settings")
	}
	tfhcl.ConvertSingleBlockToAttribute(body, "ssl", "ssl")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by provider StateUpgraders.
	return stateJSON.String(), nil
}

func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
