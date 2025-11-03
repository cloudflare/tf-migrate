package zero_trust_access_group

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_zero_trust_access_group", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_access_group"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_zero_trust_access_group"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Config transformation is extremely complex (524 lines with heavy AST manipulation)
	// Users should use the provider's migrate tool for config transformation
	// This migrator focuses on state transformation only
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	result := stateJSON.String()

	// Transform include, exclude, and require arrays
	ruleTypes := []string{"include", "exclude", "require"}
	for _, ruleType := range ruleTypes {
		rulePath := "attributes." + ruleType
		rules := stateJSON.Get(rulePath)
		if rules.Exists() && rules.IsArray() {
			ruleArray := rules.Array()
			if len(ruleArray) == 0 {
				// Preserve empty arrays as empty arrays
				result, _ = sjson.Set(result, rulePath, []interface{}{})
			} else {
				transformed := m.transformRuleArray(ruleArray)
				result, _ = sjson.Set(result, rulePath, transformed)
			}
		}
	}

	return result, nil
}

// transformRuleArray transforms an array of rule objects
func (m *V4ToV5Migrator) transformRuleArray(rules []gjson.Result) []interface{} {
	var expandedRules []interface{}

	for _, rule := range rules {
		// Check if already in v5 format (all fields are objects)
		if m.isV5Format(rule) {
			expandedRules = append(expandedRules, rule.Value())
			continue
		}

		// Expand this rule into multiple rules
		expanded := m.expandRule(rule)
		expandedRules = append(expandedRules, expanded...)
	}

	return expandedRules
}

// isV5Format checks if a rule is already in v5 format
func (m *V4ToV5Migrator) isV5Format(rule gjson.Result) bool {
	// In v5 format, each field value should be an object (not array or boolean)
	// Quick heuristic: if we see any array or boolean values, it's v4 format
	ruleMap := rule.Map()
	for _, value := range ruleMap {
		if value.IsArray() || (value.Type == gjson.True || value.Type == gjson.False) {
			return false
		}
	}
	return true
}

// expandRule expands a single v4 rule into multiple v5 rules
func (m *V4ToV5Migrator) expandRule(rule gjson.Result) []interface{} {
	var expandedRules []interface{}

	// Process attributes in fixed order for consistency
	attributeOrder := []string{
		"email", "email_domain", "ip", "geo", "group", "service_token",
		"email_list", "ip_list", "login_method", "device_posture",
		"common_names", "azure", "github", "gsuite", "okta",
		"saml", "external_evaluation",
		"everyone", "certificate", "any_valid_service_token",
	}

	ruleMap := rule.Map()
	for _, attr := range attributeOrder {
		value, exists := ruleMap[attr]
		if !exists {
			continue
		}

		switch attr {
		// Boolean attributes → empty objects
		case "everyone", "certificate", "any_valid_service_token":
			expandedRules = append(expandedRules, map[string]interface{}{
				attr: map[string]interface{}{},
			})

		// Array attributes with simple field mapping
		case "email", "email_domain", "ip", "geo", "group", "service_token",
			"email_list", "ip_list", "login_method", "device_posture":
			if value.IsArray() {
				fieldName := m.getFieldName(attr)
				for _, item := range value.Array() {
					expandedRules = append(expandedRules, map[string]interface{}{
						attr: map[string]interface{}{
							fieldName: item.Value(),
						},
					})
				}
			}

		// common_names → common_name
		case "common_names":
			if value.IsArray() {
				for _, item := range value.Array() {
					expandedRules = append(expandedRules, map[string]interface{}{
						"common_name": map[string]interface{}{
							"common_name": item.Value(),
						},
					})
				}
			}

		// Complex blocks that need special handling
		case "azure":
			expandedRules = append(expandedRules, m.expandAzureBlocks(value)...)
		case "github":
			expandedRules = append(expandedRules, m.expandGithubBlocks(value)...)
		case "gsuite":
			expandedRules = append(expandedRules, m.expandGsuiteBlocks(value)...)
		case "okta":
			expandedRules = append(expandedRules, m.expandOktaBlocks(value)...)
		case "saml":
			expandedRules = append(expandedRules, m.expandSamlBlocks(value)...)
		case "external_evaluation":
			expandedRules = append(expandedRules, m.expandExternalEvaluationBlocks(value)...)
		}
	}

	return expandedRules
}

// getFieldName returns the nested field name for array attributes
func (m *V4ToV5Migrator) getFieldName(attr string) string {
	fieldMap := map[string]string{
		"email":          "email",
		"email_domain":   "domain",
		"ip":             "ip",
		"geo":            "country_code",
		"group":          "id",
		"service_token":  "token_id",
		"email_list":     "id",
		"ip_list":        "id",
		"login_method":   "id",
		"device_posture": "integration_uid",
	}
	return fieldMap[attr]
}

// expandAzureBlocks expands azure blocks (renamed to azure_ad with ID expansion)
func (m *V4ToV5Migrator) expandAzureBlocks(value gjson.Result) []interface{} {
	var expanded []interface{}

	// Azure can be array of blocks or single block
	blocks := value.Array()
	if !value.IsArray() {
		blocks = []gjson.Result{value}
	}

	for _, block := range blocks {
		identityProviderID := block.Get("identity_provider_id")
		idArray := block.Get("id")

		if idArray.IsArray() {
			for _, id := range idArray.Array() {
				obj := map[string]interface{}{
					"azure_ad": map[string]interface{}{
						"id": id.Value(),
					},
				}
				if identityProviderID.Exists() {
					obj["azure_ad"].(map[string]interface{})["identity_provider_id"] = identityProviderID.Value()
				}
				expanded = append(expanded, obj)
			}
		}
	}

	return expanded
}

// expandGithubBlocks expands github blocks (renamed to github_organization with teams expansion)
func (m *V4ToV5Migrator) expandGithubBlocks(value gjson.Result) []interface{} {
	var expanded []interface{}

	blocks := value.Array()
	if !value.IsArray() {
		blocks = []gjson.Result{value}
	}

	for _, block := range blocks {
		name := block.Get("name")
		identityProviderID := block.Get("identity_provider_id")
		teams := block.Get("teams")

		if teams.IsArray() {
			for _, team := range teams.Array() {
				obj := map[string]interface{}{
					"github_organization": map[string]interface{}{
						"team": team.Value(),
					},
				}
				githubOrg := obj["github_organization"].(map[string]interface{})
				if name.Exists() {
					githubOrg["name"] = name.Value()
				}
				if identityProviderID.Exists() {
					githubOrg["identity_provider_id"] = identityProviderID.Value()
				}
				expanded = append(expanded, obj)
			}
		}
	}

	return expanded
}

// expandGsuiteBlocks expands gsuite blocks with email expansion
func (m *V4ToV5Migrator) expandGsuiteBlocks(value gjson.Result) []interface{} {
	var expanded []interface{}

	blocks := value.Array()
	if !value.IsArray() {
		blocks = []gjson.Result{value}
	}

	for _, block := range blocks {
		identityProviderID := block.Get("identity_provider_id")
		emails := block.Get("email")

		if emails.IsArray() {
			for _, email := range emails.Array() {
				obj := map[string]interface{}{
					"gsuite": map[string]interface{}{
						"email": email.Value(),
					},
				}
				if identityProviderID.Exists() {
					obj["gsuite"].(map[string]interface{})["identity_provider_id"] = identityProviderID.Value()
				}
				expanded = append(expanded, obj)
			}
		}
	}

	return expanded
}

// expandOktaBlocks expands okta blocks with name expansion
func (m *V4ToV5Migrator) expandOktaBlocks(value gjson.Result) []interface{} {
	var expanded []interface{}

	blocks := value.Array()
	if !value.IsArray() {
		blocks = []gjson.Result{value}
	}

	for _, block := range blocks {
		identityProviderID := block.Get("identity_provider_id")
		names := block.Get("name")

		if names.IsArray() {
			for _, name := range names.Array() {
				obj := map[string]interface{}{
					"okta": map[string]interface{}{
						"name": name.Value(),
					},
				}
				if identityProviderID.Exists() {
					obj["okta"].(map[string]interface{})["identity_provider_id"] = identityProviderID.Value()
				}
				expanded = append(expanded, obj)
			}
		}
	}

	return expanded
}

// expandSamlBlocks expands saml blocks (no array expansion, just wrap)
func (m *V4ToV5Migrator) expandSamlBlocks(value gjson.Result) []interface{} {
	var expanded []interface{}

	blocks := value.Array()
	if !value.IsArray() {
		blocks = []gjson.Result{value}
	}

	for _, block := range blocks {
		expanded = append(expanded, map[string]interface{}{
			"saml": block.Value(),
		})
	}

	return expanded
}

// expandExternalEvaluationBlocks expands external_evaluation blocks (no array expansion, just wrap)
func (m *V4ToV5Migrator) expandExternalEvaluationBlocks(value gjson.Result) []interface{} {
	var expanded []interface{}

	blocks := value.Array()
	if !value.IsArray() {
		blocks = []gjson.Result{value}
	}

	for _, block := range blocks {
		expanded = append(expanded, map[string]interface{}{
			"external_evaluation": block.Value(),
		})
	}

	return expanded
}
