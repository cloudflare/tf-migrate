package account_member

import (
	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
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

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	tfhcl.RenameAttribute(body, "email_address", "email")
	tfhcl.RenameAttribute(body, "role_ids", "roles")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	result := stateJSON.String()

	if stateJSON.Get("resources").Exists() {
		// Full state document - transform all resources
		return m.transformFullState(result, stateJSON)
	}

	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	result = m.transformSingleAccountMember(result, stateJSON)

	return result, nil
}

func (m *V4ToV5Migrator) transformFullState(result string, stateJSON gjson.Result) (string, error) {
	resources := stateJSON.Get("resources")
	if !resources.Exists() {
		return result, nil
	}

	resources.ForEach(func(key, resource gjson.Result) bool {
		resourceType := resource.Get("type").String()

		if !m.CanHandle(resourceType) {
			return true // continue loop
		}

		insts := resource.Get("instances")
		insts.ForEach(func(instKey, inst gjson.Result) bool {
			instPath := "resources." + key.String() + ".instances." + instKey.String()

			instJSON := inst.String()
			transformedInst := m.transformSingleAccountMember(instJSON, inst)
			transformedInstParsed := gjson.Parse(transformedInst)
			result, _ = sjson.SetRaw(result, instPath, transformedInstParsed.Raw)

			return true
		})

		return true
	})

	return result, nil
}

func (m *V4ToV5Migrator) transformSingleAccountMember(result string, instance gjson.Result) string {
	attrs := instance.Get("attributes")
	result = state.RenameField(result, "attributes", attrs, "email_address", "email")
	result = state.RenameField(result, "attributes", attrs, "role_ids", "roles")

	return result
}
