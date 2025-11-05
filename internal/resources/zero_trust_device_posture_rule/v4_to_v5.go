package zero_trust_device_posture_rule

import (
	"encoding/json"
	"reflect"

	"github.com/cloudflare/tf-migrate/internal/hcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
	"github.com/cloudflare/tf-migrate/internal/transform/structural"
)

// V4ToV5Migrator handles migration of device posture rule resources from v4 to v5
type V4ToV5Migrator struct {
	typeUpdater *structural.ResourceTypeUpdater
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{
		typeUpdater: &structural.ResourceTypeUpdater{
			OldType: "cloudflare_device_posture_rule",
			NewType: "cloudflare_zero_trust_device_posture_rule",
		},
	}
	internal.RegisterMigrator("cloudflare_device_posture_rule", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_device_posture_rule", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_zero_trust_device_posture_rule"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_device_posture_rule" ||
		resourceType == "cloudflare_zero_trust_device_posture_rule"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - all transformations done at HCL level
	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Rename resource type
	tfhcl.RenameResourceType(block, "cloudflare_device_posture_rule", "cloudflare_zero_trust_device_posture_rule")

	body := block.Body()

	// Convert optional input block to attribute, handling nested locations attribute
	inputBlock := tfhcl.FindBlockByType(body, "input")
	if inputBlock != nil {
		m.convertInputBlockToAttribute(body, inputBlock)
	}

	// Convert optional match blocks to attribute array
	matchBlocks := tfhcl.FindBlocksByType(body, "match")
	if len(matchBlocks) > 0 {
		m.convertMatchBlocksToArray(body, matchBlocks)
	}

	// IF there is no name attribute, check in state for a name value and if so, use it
	if !tfhcl.HasAttribute(body, "name") {
		if ctx.StateJSON != "" {
			labels := block.Labels()
			resourceName := labels[1]
			gjson.Parse(ctx.StateJSON).Get("resources").ForEach(func(key, resource gjson.Result) bool {
				if m.CanHandle(resource.Get("type").String()) && resource.Get("name").String() == resourceName {
					if resource.Get("instances.0.attributes.name").Exists() {
						body.SetAttributeValue("name", cty.StringVal(resource.Get("instances.0.attributes.name").String()))
					}
					return false
				}
				return true
			})
		}
	}

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

// convertInputBlockToAttribute converts input block to attribute syntax
// Also handles nested locations block conversion and removal of running field
func (m *V4ToV5Migrator) convertInputBlockToAttribute(body *hclwrite.Body, inputBlock *hclwrite.Block) {
	inputBody := inputBlock.Body()

	// Remove running attribute
	tfhcl.RemoveAttributes(inputBody, "running")

	// If it exists, convert nested locations block to attribute
	if locationsBlock := tfhcl.FindBlockByType(inputBody, "locations"); locationsBlock != nil {
		locationsTokens := hcl.BuildObjectFromBlock(locationsBlock)
		inputBody.SetAttributeRaw("locations", locationsTokens)
		inputBody.RemoveBlock(locationsBlock)
	}

	// Conver input block itself to an attribute
	inputTokens := hcl.BuildObjectFromBlock(inputBlock)
	body.SetAttributeRaw("input", inputTokens)
	body.RemoveBlock(inputBlock)
}

// convertMatchBlocksToArray converts match blocks to an array attribute
func (m *V4ToV5Migrator) convertMatchBlocksToArray(body *hclwrite.Body, matchBlocks []*hclwrite.Block) {
	// Build array of object tokens from blocks
	var objectTokens []hclwrite.Tokens

	for _, matchBlock := range matchBlocks {
		matchBody := matchBlock.Body()

		// Build compact single-line object tokens manually
		var objTokens hclwrite.Tokens

		// Extract platform attribute if it exists
		if platformAttr := matchBody.GetAttribute("platform"); platformAttr != nil {
			valueTokens := platformAttr.Expr().BuildTokens(nil)

			// Manually construct compact object: { platform = "value" }
			objTokens = hclwrite.Tokens{
				{Type: hclsyntax.TokenOBrace, Bytes: []byte{'{'}},
				{Type: hclsyntax.TokenIdent, Bytes: []byte(" platform")},
				{Type: hclsyntax.TokenEqual, Bytes: []byte(" =")},
			}

			// Add space before value
			objTokens = append(objTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})

			// Add value tokens (without leading space since we added it above)
			for i, tok := range valueTokens {
				if i == 0 && tok.Type == hclsyntax.TokenIdent {
					// Remove any leading space from first token
					objTokens = append(objTokens, tok)
				} else {
					objTokens = append(objTokens, tok)
				}
			}

			// Close object
			objTokens = append(objTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(" ")})
			objTokens = append(objTokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte{'}'}})

			objectTokens = append(objectTokens, objTokens)
		}
	}

	// Build multi-line array tokens manually for proper formatting
	if len(objectTokens) > 0 {
		var arrayTokens hclwrite.Tokens

		// Opening bracket
		arrayTokens = append(arrayTokens, &hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte{'['}})
		arrayTokens = append(arrayTokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})

		// Add each object with indentation and comma
		for i, objTokens := range objectTokens {
			// Add indentation (4 spaces to match HCL formatting)
			arrayTokens = append(arrayTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("    ")})

			// Add object tokens
			arrayTokens = append(arrayTokens, objTokens...)

			// Add comma only if not the last item
			if i < len(objectTokens)-1 {
				arrayTokens = append(arrayTokens, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte{','}})
			}

			// Add newline after each object
			arrayTokens = append(arrayTokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})
		}

		// Add indentation before closing bracket
		arrayTokens = append(arrayTokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("  ")})

		// Closing bracket
		arrayTokens = append(arrayTokens, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte{']'}})

		body.SetAttributeRaw("match", arrayTokens)
	}

	// Remove the old match blocks
	for _, matchBlock := range matchBlocks {
		body.RemoveBlock(matchBlock)
	}
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	result := stateJSON.String()

	if stateJSON.Get("resources").Exists() {
		return m.transformFullState(result, stateJSON)
	}

	if !stateJSON.Exists() || !stateJSON.Get("attributes").Exists() {
		return result, nil
	}

	attrs := stateJSON.Get("attributes")

	result = m.transformSingleInstance(result, attrs)

	result, _ = sjson.Set(result, "schema_version", 0)

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
			return true
		}

		if resourceType == "cloudflare_device_posture_rule" {
			resourcePath := "resources." + key.String() + ".type"
			result, _ = sjson.Set(result, resourcePath, "cloudflare_zero_trust_device_posture_rule")
		}

		instances := resource.Get("instances")
		instances.ForEach(func(instKey, instance gjson.Result) bool {
			instPath := "resources." + key.String() + ".instances." + instKey.String()

			attrs := instance.Get("attributes")
			if attrs.Exists() {
				instJSON := instance.String()
				transformedInst := m.transformSingleInstance(instJSON, attrs)

				transformedInst, _ = sjson.Set(transformedInst, "schema_version", 0)

				result, _ = sjson.SetRaw(result, instPath, transformedInst)
			} else {
				schemaPath := instPath + ".schema_version"
				result, _ = sjson.Set(result, schemaPath, 0)
			}
			return true
		})

		return true
	})

	return result, nil
}

func (m *V4ToV5Migrator) transformSingleInstance(result string, attrs gjson.Result) string {
	inputField := attrs.Get("input")
	if inputField.Exists() {
		if m.inputFieldIsEmpty(attrs) {
			result, _ = sjson.Delete(result, "attributes.input")
		} else {
			result = m.transformInputArrayToObject(result, attrs)
		}
	}

	// Re-parse attrs after transformation to get updated structure
	updatedInstance := gjson.Parse(result)
	updatedAttrs := updatedInstance.Get("attributes")

	// Convert numeric fields to float64 (after input is already an object)
	result = m.convertNumericFields(result, updatedAttrs)

	inputField = updatedAttrs.Get("input")
	if inputField.Exists() {
		// Transform all empty values to null
		result = m.transformInputEmptyValuesToNull(result, updatedAttrs)

		// Remove deprecated running field from input
		if inputField.Get("running").Exists() {
			result, _ = sjson.Delete(result, "attributes.input.running")
		}
	}

	return result
}

// stateHasEmptyInputAttribute checks if the state file's input attribute consists entirely of empty values
func (m *V4ToV5Migrator) inputFieldIsEmpty(attrs gjson.Result) bool {
	emptyInput := `
{
	"active_threats": 0,
	"certificate_id": "",
	"check_disks": null,
	"check_private_key": false,
	"cn": "",
	"compliance_status": "",
	"connection_id": "",
	"count_operator": "",
	"domain": "",
	"eid_last_seen": "",
	"enabled": false,
	"exists": false,
	"extended_key_usage": null,
	"id": "",
	"infected": false,
	"is_active": false,
	"issue_count": "",
	"last_seen": "",
	"locations": [],
	"network_status": "",
	"operational_state": "",
	"operator": "",
	"os": "",
	"os_distro_name": "",
	"os_distro_revision": "",
	"os_version_extra": "",
	"overall": "",
	"path": "",
	"require_all": false,
	"risk_level": "",
	"running": false,
	"score": 0,
	"sensor_config": "",
	"sha256": "",
	"state": "",
	"thumbprint": "",
	"total_score": 0,
	"version": "",
	"version_operator": ""
}`

	inputField := attrs.Get("input")
	if !inputField.Exists() {
		return false
	}

	inputObj := inputField.Array()[0]

	var actual, expected map[string]interface{}
	json.Unmarshal([]byte(inputObj.Raw), &actual)
	json.Unmarshal([]byte(emptyInput), &expected)

	return reflect.DeepEqual(actual, expected)
}

func (m *V4ToV5Migrator) transformInputEmptyValuesToNull(stateJSON string, attrs gjson.Result) string {
	inputField := attrs.Get("input")
	if !inputField.Exists() {
		return stateJSON
	}

	var inputObj gjson.Result
	if inputField.IsArray() {
		inputObj = inputField.Array()[0]
	} else {
		inputObj = inputField
	}

	inputObj.ForEach(func(key, value gjson.Result) bool {
		if isEmptyValue(value) {
			stateJSON, _ = sjson.Set(stateJSON, "attributes.input."+key.String(), nil)
		}

		return true
	})

	return stateJSON
}

// isEmptyValue checks if a gjson.Result value is considered "empty" (default/zero)
func isEmptyValue(value gjson.Result) bool {
	if !value.Exists() {
		return true
	}

	switch value.Type {
	case gjson.Null:
		return true
	case gjson.False:
		return true // false is an empty value
	case gjson.Number:
		return value.Num == 0 // 0 is an empty value
	case gjson.String:
		return value.Str == "" // empty string is an empty value
	case gjson.JSON:
		// Check if it's an empty array or object
		if value.IsArray() {
			return len(value.Array()) == 0
		}
		if value.IsObject() {
			// Empty object or object with all empty values
			isEmpty := true
			value.ForEach(func(_, v gjson.Result) bool {
				if !isEmptyValue(v) {
					isEmpty = false
					return false
				}
				return true
			})
			return isEmpty
		}
		return false
	default:
		return false
	}
}

// transformInputArrayToObject converts input field from array to object
func (m *V4ToV5Migrator) transformInputArrayToObject(stateJSON string, attrs gjson.Result) string {
	inputField := attrs.Get("input")

	if !inputField.Exists() {
		return stateJSON
	}

	// If input is an array with one element, convert to object
	if inputField.IsArray() && len(inputField.Array()) > 0 {
		inputObj := inputField.Array()[0]

		// First, transform nested locations if it exists and is an array
		if locationsField := inputObj.Get("locations"); locationsField.Exists() {
			if locationsField.IsArray() && len(locationsField.Array()) > 0 {
				locationsObj := locationsField.Array()[0]
				// Set locations as object instead of array
				inputObjMap := inputObj.Value().(map[string]interface{})
				inputObjMap["locations"] = locationsObj.Value()
				stateJSON, _ = sjson.Set(stateJSON, "attributes.input", inputObjMap)
				return stateJSON
			}
		}

		// Set input as object instead of array
		stateJSON, _ = sjson.Set(stateJSON, "attributes.input", inputObj.Value())
	} else if inputField.IsArray() && len(inputField.Array()) == 0 {
		// Empty array - remove it
		stateJSON, _ = sjson.Delete(stateJSON, "attributes.input")
	}

	return stateJSON
}

// convertNumericFields converts TypeInt fields to Float64Attribute
func (m *V4ToV5Migrator) convertNumericFields(stateJSON string, attrs gjson.Result) string {
	inputField := attrs.Get("input")
	if !inputField.Exists() || !inputField.IsObject() {
		return stateJSON
	}

	// Convert active_threats: int → float64
	if activeThreats := inputField.Get("active_threats"); activeThreats.Exists() {
		floatVal := state.ConvertToFloat64(activeThreats)
		stateJSON, _ = sjson.Set(stateJSON, "attributes.input.active_threats", floatVal)
	}

	// Convert total_score: int → float64
	if totalScore := inputField.Get("total_score"); totalScore.Exists() {
		floatVal := state.ConvertToFloat64(totalScore)
		stateJSON, _ = sjson.Set(stateJSON, "attributes.input.total_score", floatVal)
	}

	// Convert score: int → float64
	if score := inputField.Get("score"); score.Exists() {
		floatVal := state.ConvertToFloat64(score)
		stateJSON, _ = sjson.Set(stateJSON, "attributes.input.score", floatVal)
	}

	return stateJSON
}
