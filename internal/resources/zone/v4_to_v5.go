package zone

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of Zone resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_zone", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zone"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_zone"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// 1. Rename zone → name
	tfhcl.RenameAttribute(body, "zone", "name")

	// 2. Transform account_id to account = { id = "..." }
	if accountIDAttr := body.GetAttribute("account_id"); accountIDAttr != nil {
		// Get the current expression
		expr := accountIDAttr.Expr()
		tokens := expr.BuildTokens(nil)

		// Create nested object: account = { id = <value> }
		// Build the new expression manually
		objTokens := []*hclwrite.Token{
			{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")},
			{Type: hclsyntax.TokenNewline, Bytes: []byte("\n    ")},
			{Type: hclsyntax.TokenIdent, Bytes: []byte("id")},
			{Type: hclsyntax.TokenEqual, Bytes: []byte(" = ")},
		}
		objTokens = append(objTokens, tokens...)
		objTokens = append(objTokens,
			&hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n  ")},
			&hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")},
		)

		body.SetAttributeRaw("account", objTokens)
		body.RemoveAttribute("account_id")
	}

	// 3. Remove obsolete attributes
	body.RemoveAttribute("jump_start")
	body.RemoveAttribute("plan")

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	// This function can receive either:
	// 1. A full resource (in unit tests) - has "type", "name", "instances"
	// 2. A single instance (in actual migration framework) - has "attributes"
	// We need to handle both cases

	result := stateJSON.String()

	// Check if this is a full resource (has "type" and "instances") or a single instance
	if stateJSON.Get("type").Exists() && stateJSON.Get("instances").Exists() {
		// Full resource - transform all instances
		return m.transformFullResource(result, stateJSON)
	}

	// Single instance - transform just the attributes
	if !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	return m.transformSingleInstance(result, stateJSON), nil
}

// transformFullResource handles transformation of a full resource with instances
func (m *V4ToV5Migrator) transformFullResource(result string, resource gjson.Result) (string, error) {
	resourceType := resource.Get("type").String()
	if resourceType != "cloudflare_zone" {
		return result, nil
	}

	// Transform all instances
	instances := resource.Get("instances")
	instances.ForEach(func(key, instance gjson.Result) bool {
		instPath := "instances." + key.String()
		instJSON := instance.String()
		transformedInst := m.transformSingleInstance(instJSON, instance)
		result, _ = sjson.SetRaw(result, instPath, transformedInst)
		return true
	})

	return result, nil
}

// transformSingleInstance transforms a single instance's attributes
func (m *V4ToV5Migrator) transformSingleInstance(result string, instance gjson.Result) string {
	if !instance.Get("attributes").Exists() {
		return result
	}

	// 1. zone → name
	if instance.Get("attributes.zone").Exists() {
		zoneValue := instance.Get("attributes.zone")
		result, _ = sjson.Set(result, "attributes.name", zoneValue.Value())
		result, _ = sjson.Delete(result, "attributes.zone")
	}

	// 2. account_id → account = { id = "..." }
	if instance.Get("attributes.account_id").Exists() {
		accountID := instance.Get("attributes.account_id")
		result, _ = sjson.Set(result, "attributes.account", map[string]interface{}{
			"id": accountID.Value(),
		})
		result, _ = sjson.Delete(result, "attributes.account_id")
	}

	// 3. Remove jump_start
	if instance.Get("attributes.jump_start").Exists() {
		result, _ = sjson.Delete(result, "attributes.jump_start")
	}

	// 4. Remove plan
	if instance.Get("attributes.plan").Exists() {
		result, _ = sjson.Delete(result, "attributes.plan")
	}

	return result
}
