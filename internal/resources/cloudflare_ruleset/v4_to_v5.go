package cloudflare_ruleset

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_ruleset", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_ruleset"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_ruleset"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Config transformation is extremely complex (815 lines with many specialized helpers)
	// Users should use the provider's migrate tool for config transformation
	// This migrator focuses on state transformation only
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	result := stateJSON.String()

	// Set schema_version to 0 for v5
	result, _ = sjson.Set(result, "schema_version", 0)

	// Check if rules is already a JSON array (v5 format)
	rulesValue := stateJSON.Get("attributes.rules")
	if rulesValue.Exists() && rulesValue.IsArray() {
		// Already in v5 format, just ensure action_parameters are properly structured
		var rules []interface{}
		for _, ruleVal := range rulesValue.Array() {
			if ruleMap, ok := ruleVal.Value().(map[string]interface{}); ok {
				// Remove disable_railgun if present (removed in v5)
				if ap, ok := ruleMap["action_parameters"].(map[string]interface{}); ok {
					delete(ap, "disable_railgun")
				}
				rules = append(rules, ruleMap)
			}
		}
		result, _ = sjson.Set(result, "attributes.rules", rules)
		return result, nil
	}

	// Transform from indexed format to array
	// Note: "rules.#" is a literal key name with a dot, not a path, so we need to escape it
	rulesCount := stateJSON.Get("attributes.rules\\.#")
	if !rulesCount.Exists() || rulesCount.Int() == 0 {
		return result, nil
	}

	var rules []interface{}
	for i := int64(0); i < rulesCount.Int(); i++ {
		rule := m.transformRule(stateJSON, i)
		if rule != nil {
			rules = append(rules, rule)
		}
	}

	// Rebuild the attributes map without indexed rule keys
	attrsResult := gjson.Get(result, "attributes")
	if attrsResult.Exists() && attrsResult.IsObject() {
		newAttrs := make(map[string]interface{})
		attrsMap := attrsResult.Map()

		// Copy all attributes except those starting with "rules."
		for key, val := range attrsMap {
			// Skip keys that are "rules.#" or start with "rules."
			if key == "rules.#" || (len(key) > 6 && key[:6] == "rules.") {
				continue
			}
			newAttrs[key] = val.Value()
		}

		// Add the new rules array
		newAttrs["rules"] = rules

		// Replace the entire attributes object
		result, _ = sjson.Set(result, "attributes", newAttrs)
	}

	return result, nil
}

// transformRule transforms a single rule from indexed format to object
func (m *V4ToV5Migrator) transformRule(stateJSON gjson.Result, ruleIdx int64) map[string]interface{} {
	rule := make(map[string]interface{})
	// Escape dots in the path since "rules.0" is a literal key
	basePath := fmt.Sprintf("attributes.rules\\.%d", ruleIdx)

	// Copy basic rule attributes
	simpleFields := []string{"id", "ref", "enabled", "description", "expression", "action"}
	for _, field := range simpleFields {
		val := stateJSON.Get(basePath + "\\." + field)
		if val.Exists() {
			rule[field] = val.Value()
		}
	}

	// Transform action_parameters if present
	actionParamsCount := stateJSON.Get(basePath + "\\.action_parameters\\.#")
	if actionParamsCount.Exists() && actionParamsCount.Int() > 0 {
		actionParams := m.transformActionParameters(stateJSON, basePath)
		if len(actionParams) > 0 {
			// Remove disable_railgun (removed in v5)
			delete(actionParams, "disable_railgun")
			rule["action_parameters"] = actionParams
		}
	}

	// Transform ratelimit if present
	rateLimitCount := stateJSON.Get(basePath + "\\.ratelimit\\.#")
	if rateLimitCount.Exists() && rateLimitCount.Int() > 0 {
		ratelimit := m.transformRatelimit(stateJSON, basePath)
		if len(ratelimit) > 0 {
			rule["ratelimit"] = ratelimit
		}
	}

	// Transform exposed_credential_check if present
	exposedCredCount := stateJSON.Get(basePath + "\\.exposed_credential_check\\.#")
	if exposedCredCount.Exists() && exposedCredCount.Int() > 0 {
		exposedCred := m.transformExposedCredentialCheck(stateJSON, basePath)
		if len(exposedCred) > 0 {
			rule["exposed_credential_check"] = exposedCred
		}
	}

	// Transform logging if present
	loggingCount := stateJSON.Get(basePath + "\\.logging\\.#")
	if loggingCount.Exists() && loggingCount.Int() > 0 {
		logging := m.transformLogging(stateJSON, basePath)
		if len(logging) > 0 {
			rule["logging"] = logging
		}
	}

	return rule
}

// transformActionParameters transforms action_parameters from indexed to object format
func (m *V4ToV5Migrator) transformActionParameters(stateJSON gjson.Result, basePath string) map[string]interface{} {
	actionParams := make(map[string]interface{})
	// basePath already has escaped dots, continue escaping
	apPath := basePath + "\\.action_parameters\\.0"

	// Simple string/number/boolean fields
	simpleFields := []string{
		"additional_cacheable_ports", "automatic_https_rewrites", "bic", "cache",
		"content", "content_type", "disable_apps", "disable_zaraz", "disable_rum",
		"fonts", "email_obfuscation", "host_header", "hotlink_protection", "id",
		"increment", "mirage", "opportunistic_encryption", "origin_cache_control",
		"polish", "products", "read_timeout", "respect_strong_etags",
		"rocket_loader", "rules", "ruleset", "rulesets", "security_level",
		"server_side_excludes", "ssl", "status_code", "sxg", "origin_error_page_passthru",
	}

	for _, field := range simpleFields {
		val := stateJSON.Get(apPath + "\\." + field)
		if val.Exists() {
			actionParams[field] = val.Value()
		}
	}

	// Array fields
	phasesCount := stateJSON.Get(apPath + "\\.phases\\.#")
	if phasesCount.Exists() && phasesCount.Int() > 0 {
		var phases []string
		for j := int64(0); j < phasesCount.Int(); j++ {
			val := stateJSON.Get(fmt.Sprintf("%s\\.phases\\.%d", apPath, j))
			if val.Exists() {
				phases = append(phases, val.String())
			}
		}
		if len(phases) > 0 {
			actionParams["phases"] = phases
		}
	}

	// Transform headers from list to map
	headersCount := stateJSON.Get(apPath + "\\.headers\\.#")
	if headersCount.Exists() && headersCount.Int() > 0 {
		headers := make(map[string]interface{})
		for j := int64(0); j < headersCount.Int(); j++ {
			headerPath := fmt.Sprintf("%s\\.headers\\.%d", apPath, j)
			name := stateJSON.Get(headerPath + "\\.name").String()
			if name != "" {
				header := make(map[string]interface{})
				if val := stateJSON.Get(headerPath + "\\.operation"); val.Exists() {
					header["operation"] = val.String()
				}
				if val := stateJSON.Get(headerPath + "\\.value"); val.Exists() {
					header["value"] = val.String()
				}
				if val := stateJSON.Get(headerPath + "\\.expression"); val.Exists() {
					header["expression"] = val.String()
				}
				headers[name] = header
			}
		}
		if len(headers) > 0 {
			actionParams["headers"] = headers
		}
	}

	// Transform cookie_fields, request_fields, response_fields (SetAttribute -> ListNestedAttribute)
	for _, field := range []string{"cookie_fields", "request_fields", "response_fields"} {
		fieldCount := stateJSON.Get(fmt.Sprintf("%s\\.%s\\.#", apPath, field))
		if fieldCount.Exists() && fieldCount.Int() > 0 {
			var fieldList []map[string]interface{}
			for j := int64(0); j < fieldCount.Int(); j++ {
				fieldVal := stateJSON.Get(fmt.Sprintf("%s\\.%s\\.%d", apPath, field, j))
				if fieldVal.Exists() {
					fieldList = append(fieldList, map[string]interface{}{
						"name": fieldVal.String(),
					})
				}
			}
			if len(fieldList) > 0 {
				actionParams[field] = fieldList
			}
		}
	}

	// Transform single nested blocks (simple approach - copy all fields)
	singleBlocks := []string{
		"algorithms", "uri", "matched_data", "response", "autominify",
		"edge_ttl", "browser_ttl", "serve_stale", "cache_key", "cache_reserve",
		"from_list", "from_value", "origin", "sni", "overrides",
	}

	for _, blockName := range singleBlocks {
		blockCount := stateJSON.Get(fmt.Sprintf("%s\\.%s\\.#", apPath, blockName))
		if blockCount.Exists() && blockCount.Int() > 0 {
			// For simplicity, extract the first element and copy all its fields
			blockPath := fmt.Sprintf("%s\\.%s\\.0", apPath, blockName)
			blockData := stateJSON.Get(blockPath)
			if blockData.Exists() && blockData.IsObject() {
				actionParams[blockName] = blockData.Value()
			}
		}
	}

	return actionParams
}

// transformRatelimit transforms ratelimit from indexed to object format
func (m *V4ToV5Migrator) transformRatelimit(stateJSON gjson.Result, basePath string) map[string]interface{} {
	ratelimit := make(map[string]interface{})
	// basePath already has escaped dots, continue escaping
	rlPath := basePath + "\\.ratelimit\\.0"

	// Handle characteristics array
	charCount := stateJSON.Get(rlPath + "\\.characteristics\\.#")
	if charCount.Exists() && charCount.Int() > 0 {
		var chars []string
		for j := int64(0); j < charCount.Int(); j++ {
			val := stateJSON.Get(fmt.Sprintf("%s\\.characteristics\\.%d", rlPath, j))
			if val.Exists() {
				chars = append(chars, val.String())
			}
		}
		ratelimit["characteristics"] = chars
	}

	// Copy simple ratelimit fields
	rlFields := []string{
		"period", "requests_per_period", "score_per_period",
		"score_response_header_name", "mitigation_timeout", "counting_expression",
		"requests_to_origin",
	}
	for _, field := range rlFields {
		val := stateJSON.Get(rlPath + "\\." + field)
		if val.Exists() {
			ratelimit[field] = val.Value()
		}
	}

	return ratelimit
}

// transformExposedCredentialCheck transforms exposed_credential_check from indexed to object format
func (m *V4ToV5Migrator) transformExposedCredentialCheck(stateJSON gjson.Result, basePath string) map[string]interface{} {
	exposedCred := make(map[string]interface{})
	// basePath already has escaped dots, continue escaping
	ecPath := basePath + "\\.exposed_credential_check\\.0"

	if val := stateJSON.Get(ecPath + "\\.username_expression"); val.Exists() {
		exposedCred["username_expression"] = val.String()
	}
	if val := stateJSON.Get(ecPath + "\\.password_expression"); val.Exists() {
		exposedCred["password_expression"] = val.String()
	}

	return exposedCred
}

// transformLogging transforms logging from indexed to object format
func (m *V4ToV5Migrator) transformLogging(stateJSON gjson.Result, basePath string) map[string]interface{} {
	logging := make(map[string]interface{})
	// basePath already has escaped dots, continue escaping
	logPath := basePath + "\\.logging\\.0"

	if val := stateJSON.Get(logPath + "\\.enabled"); val.Exists() {
		logging["enabled"] = val.Bool()
	}

	return logging
}
