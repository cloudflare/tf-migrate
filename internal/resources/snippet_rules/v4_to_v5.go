package snippet_rules

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_snippet_rules", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_snippet_rules"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_snippet_rules"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	body := block.Body()

	// Find and collect all rules blocks
	rulesBlocks := body.Blocks()
	var blocksToRemove []*hclwrite.Block
	var hasRules bool

	// First pass: check if we have rules blocks
	for _, rulesBlock := range rulesBlocks {
		if rulesBlock.Type() == "rules" {
			hasRules = true
			blocksToRemove = append(blocksToRemove, rulesBlock)
		}
	}

	// Remove all rules blocks
	for _, rulesBlock := range blocksToRemove {
		body.RemoveBlock(rulesBlock)
	}

	// Add rules as a list attribute if we have any rules
	if hasRules {
		// Build the rules list using raw tokens to preserve formatting
		var rulesTokens hclwrite.Tokens
		rulesTokens = append(rulesTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenOBrack,
			Bytes: []byte("["),
		})
		rulesTokens = append(rulesTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		})

		// Process each rules block
		for i, rulesBlock := range blocksToRemove {
			if rulesBlock.Type() == "rules" {
				// Add indentation
				rulesTokens = append(rulesTokens, &hclwrite.Token{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("    "),
				})
				rulesTokens = append(rulesTokens, &hclwrite.Token{
					Type:  hclsyntax.TokenOBrace,
					Bytes: []byte("{"),
				})
				rulesTokens = append(rulesTokens, &hclwrite.Token{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				})

				ruleBody := rulesBlock.Body()

				// Process attributes in order: enabled, expression, description, snippet_name
				// This maintains a consistent order in the output
				attributeOrder := []string{"enabled", "expression", "description", "snippet_name"}

				for _, attrName := range attributeOrder {
					if attr := ruleBody.GetAttribute(attrName); attr != nil {
						rulesTokens = append(rulesTokens, &hclwrite.Token{
							Type:  hclsyntax.TokenIdent,
							Bytes: []byte("      " + attrName),
						})

						// Calculate padding for alignment
						padding := 14 - len(attrName) // Align to "snippet_name" length
						if padding < 1 {
							padding = 1
						}
						paddingBytes := make([]byte, padding)
						for j := range paddingBytes {
							paddingBytes[j] = ' '
						}
						rulesTokens = append(rulesTokens, &hclwrite.Token{
							Type:  hclsyntax.TokenIdent,
							Bytes: paddingBytes,
						})

						rulesTokens = append(rulesTokens, &hclwrite.Token{
							Type:  hclsyntax.TokenEqual,
							Bytes: []byte("="),
						})
						rulesTokens = append(rulesTokens, &hclwrite.Token{
							Type:  hclsyntax.TokenIdent,
							Bytes: []byte(" "),
						})

						// Special handling for snippet_name attribute
						if attrName == "snippet_name" {
							// Get the expression and check if it's a reference to cloudflare_snippet.*.name
							exprTokens := attr.Expr().BuildTokens(nil)
							exprStr := string(exprTokens.Bytes())

							// Check if this references a cloudflare_snippet resource's name attribute
							// Pattern: cloudflare_snippet.<resource_name>.name
							if strings.Contains(exprStr, "cloudflare_snippet.") && strings.HasSuffix(strings.TrimSpace(exprStr), ".name") {
								// Replace .name with .snippet_name
								modifiedExprStr := strings.TrimSuffix(strings.TrimSpace(exprStr), ".name") + ".snippet_name"
								rulesTokens = append(rulesTokens, &hclwrite.Token{
									Type:  hclsyntax.TokenIdent,
									Bytes: []byte(modifiedExprStr),
								})
							} else {
								// Use the original expression tokens
								rulesTokens = append(rulesTokens, exprTokens...)
							}
						} else {
							// For other attributes, use the original tokens
							rulesTokens = append(rulesTokens, attr.Expr().BuildTokens(nil)...)
						}

						rulesTokens = append(rulesTokens, &hclwrite.Token{
							Type:  hclsyntax.TokenNewline,
							Bytes: []byte("\n"),
						})
					}
				}

				// Close object
				rulesTokens = append(rulesTokens, &hclwrite.Token{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("    "),
				})
				rulesTokens = append(rulesTokens, &hclwrite.Token{
					Type:  hclsyntax.TokenCBrace,
					Bytes: []byte("}"),
				})

				// Add comma if not the last item
				if i < len(blocksToRemove)-1 {
					rulesTokens = append(rulesTokens, &hclwrite.Token{
						Type:  hclsyntax.TokenComma,
						Bytes: []byte(","),
					})
				}
				rulesTokens = append(rulesTokens, &hclwrite.Token{
					Type:  hclsyntax.TokenNewline,
					Bytes: []byte("\n"),
				})
			}
		}

		// Close list
		rulesTokens = append(rulesTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("  "),
		})
		rulesTokens = append(rulesTokens, &hclwrite.Token{
			Type:  hclsyntax.TokenCBrack,
			Bytes: []byte("]"),
		})

		// Set the rules attribute using raw tokens
		body.SetAttributeRaw("rules", rulesTokens)
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	result := stateJSON.String()

	if stateJSON.Get("type").Exists() && stateJSON.Get("instances").Exists() {
		return m.transformFullResource(result, stateJSON)
	}

	if !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	return m.transformSingleInstance(result, stateJSON), nil
}

func (m *V4ToV5Migrator) transformFullResource(result string, resource gjson.Result) (string, error) {
	resourceType := resource.Get("type").String()
	if resourceType != "cloudflare_snippet_rules" {
		return result, nil
	}

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

func (m *V4ToV5Migrator) transformSingleInstance(result string, instance gjson.Result) string {
	if !instance.Get("attributes").Exists() {
		return result
	}

	// Check if this is v4 indexed format vs v5 array format
	// In v4 format: no "rules" key exists, just "rules.#", "rules.0.x", etc. (indexed format)
	// In v5 format: "rules" key exists as an array
	rulesValue := instance.Get("attributes.rules")
	if rulesValue.Exists() && rulesValue.IsArray() {
		// Already in v5 format (rules is an array) - just update schema_version
		if instance.Get("schema_version").Int() == 1 {
			result, _ = sjson.Set(result, "schema_version", 0)
		}
		return result
	}

	// V4 indexed format - need to convert
	// Use escaped path to access the literal "rules.#" key (with dot in the key name)
	rulesCountKey := "attributes.rules\\.#"
	rulesCountResult := instance.Get(rulesCountKey)
	if !rulesCountResult.Exists() {
		// No rules at all
		if instance.Get("schema_version").Int() == 1 {
			result, _ = sjson.Set(result, "schema_version", 0)
		}
		return result
	}

	// Convert from indexed format to list format
	rulesCount := int(rulesCountResult.Int())

	var rules []interface{}
	for i := 0; i < rulesCount; i++ {
		rule := make(map[string]interface{})
		indexStr := fmt.Sprintf("%d", i)

		// enabled defaults to true if not present
		// Use escaped dots to access literal "rules.N.enabled" keys
		enabledPath := "attributes.rules\\." + indexStr + "\\.enabled"
		if instance.Get(enabledPath).Exists() {
			rule["enabled"] = instance.Get(enabledPath).Bool()
		} else {
			rule["enabled"] = true
		}

		// expression is required
		expressionPath := "attributes.rules\\." + indexStr + "\\.expression"
		if instance.Get(expressionPath).Exists() {
			rule["expression"] = instance.Get(expressionPath).String()
		}

		// snippet_name is required
		snippetNamePath := "attributes.rules\\." + indexStr + "\\.snippet_name"
		if instance.Get(snippetNamePath).Exists() {
			rule["snippet_name"] = instance.Get(snippetNamePath).String()
		}

		// description defaults to "" if not present
		descriptionPath := "attributes.rules\\." + indexStr + "\\.description"
		if instance.Get(descriptionPath).Exists() {
			rule["description"] = instance.Get(descriptionPath).String()
		} else {
			rule["description"] = ""
		}

		rules = append(rules, rule)
	}

	// Set the rules list (ensure it's an array, not nil for empty case)
	if len(rules) == 0 {
		result, _ = sjson.Set(result, "attributes.rules", []interface{}{})
	} else {
		result, _ = sjson.Set(result, "attributes.rules", rules)
	}

	// Update schema_version to 0
	result, _ = sjson.Set(result, "schema_version", 0)

	// Clean up indexed keys by parsing as map and removing them
	// This is necessary because sjson.Delete doesn't handle keys with dots well
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		return result // Return as-is if we can't parse
	}

	// Remove indexed format keys from attributes
	if attrs, ok := data["attributes"].(map[string]interface{}); ok {
		// Remove all keys that match the indexed format pattern
		// but keep the "rules" array itself
		keysToDelete := []string{}
		for key := range attrs {
			if key == "rules" {
				continue // Keep the rules array
			}
			if strings.HasPrefix(key, "rules.") {
				// This is an indexed key (rules.#, rules.%, rules.N, rules.N.field)
				keysToDelete = append(keysToDelete, key)
			}
		}
		for _, key := range keysToDelete {
			delete(attrs, key)
		}
	}

	// Marshal back to JSON
	resultBytes, err := json.Marshal(data)
	if err != nil {
		return result // Return original if marshaling fails
	}

	return string(resultBytes)
}
