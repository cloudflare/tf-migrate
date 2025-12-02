package zero_trust_device_posture_rule

import (
	"encoding/json"
	"reflect"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
	"github.com/cloudflare/tf-migrate/internal/transform/state"
)

// V4ToV5Migrator handles migration of device posture rule resources from v4 to v5
type V4ToV5Migrator struct {
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}

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
	tfhcl.RenameResourceType(block, "cloudflare_device_posture_rule", "cloudflare_zero_trust_device_posture_rule")

	body := block.Body()

	// Convert optional input block to attribute, handling nested locations attribute
	if inputBlock := tfhcl.FindBlockByType(body, "input"); inputBlock != nil {
		tfhcl.ConvertBlocksToAttribute(body, "input", "input", func(inputBlock *hclwrite.Block) {
			tfhcl.RemoveAttributes(inputBlock.Body(), "running")
			tfhcl.ConvertBlocksToAttribute(inputBlock.Body(), "locations", "locations", func(locationsBlock *hclwrite.Block) {})
		})
	}

	// Convert optional match blocks to attribute array
	if matchBlocks := tfhcl.FindBlocksByType(body, "match"); len(matchBlocks) > 0 {
		tfhcl.MergeAttributeAndBlocksToObjectArray(body, "", "match", "match", "platform", []string{}, true)
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

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	result := stateJSON.String()

	attrs := stateJSON.Get("attributes")
	if !attrs.Exists() {
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	inputField := attrs.Get("input")
	if inputField.Exists() {
		if m.inputFieldIsEmpty(attrs) {
			result, _ = sjson.Delete(result, "attributes.input")
		} else {
			result = m.transformInputArrayToObject(result, attrs)
		}
	}

	// Re-parse attrs after transformation to get updated structure
	updatedAttrs := gjson.Parse(result).Get("attributes")

	result = m.convertNumericFields(result, updatedAttrs)

	inputField = updatedAttrs.Get("input")
	if inputField.Exists() {
		// Remove input.enabled if explicitly false BEFORE transforming to null
		// This must be done before transforming empty values
		if enabled := inputField.Get("enabled"); enabled.Exists() && enabled.Type == gjson.False {
			result, _ = sjson.Delete(result, "attributes.input.enabled")
			// Re-parse updatedAttrs after deletion so transform doesn't re-add it
			updatedAttrs = gjson.Parse(result).Get("attributes")
			inputField = updatedAttrs.Get("input")
		}

		// Transform empty values to null for fields not explicitly set in config
		result = transform.TransformEmptyValuesToNull(transform.TransformEmptyValuesToNullOptions{
			Ctx:              ctx,
			Result:           result,
			FieldPath:        "attributes.input",
			FieldResult:      inputField,
			ResourceName:     resourceName,
			HCLAttributePath: "input",
			CanHandle:        m.CanHandle,
		})

		if inputField.Get("running").Exists() {
			result, _ = sjson.Delete(result, "attributes.input.running")
		}
	}

	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
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

	if !inputField.IsArray() || len(inputField.Array()) == 0 {
		return true
	}

	inputObj := inputField.Array()[0]

	var actual, expected map[string]interface{}
	json.Unmarshal([]byte(inputObj.Raw), &actual)
	json.Unmarshal([]byte(emptyInput), &expected)

	return reflect.DeepEqual(actual, expected)
}

// transformInputArrayToObject converts input field from array to object
func (m *V4ToV5Migrator) transformInputArrayToObject(stateJSON string, attrs gjson.Result) string {
	inputField := attrs.Get("input")

	if !inputField.Exists() {
		return stateJSON
	}

	if inputField.IsArray() && len(inputField.Array()) > 0 {
		inputObj := inputField.Array()[0]

		// First, transform nested locations if it exists and is an array
		if locationsField := inputObj.Get("locations"); locationsField.Exists() {
			if locationsField.IsArray() && len(locationsField.Array()) > 0 {
				locationsObj := locationsField.Array()[0]
				inputObjMap := inputObj.Value().(map[string]interface{})
				inputObjMap["locations"] = locationsObj.Value()
				stateJSON, _ = sjson.Set(stateJSON, "attributes.input", inputObjMap)
				return stateJSON
			}
		}

		stateJSON, _ = sjson.Set(stateJSON, "attributes.input", inputObj.Value())
	} else if inputField.IsArray() && len(inputField.Array()) == 0 {
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

	if activeThreats := inputField.Get("active_threats"); activeThreats.Exists() {
		floatVal := state.ConvertToFloat64(activeThreats)
		stateJSON, _ = sjson.Set(stateJSON, "attributes.input.active_threats", floatVal)
	}

	if totalScore := inputField.Get("total_score"); totalScore.Exists() {
		floatVal := state.ConvertToFloat64(totalScore)
		stateJSON, _ = sjson.Set(stateJSON, "attributes.input.total_score", floatVal)
	}

	if score := inputField.Get("score"); score.Exists() {
		floatVal := state.ConvertToFloat64(score)
		stateJSON, _ = sjson.Set(stateJSON, "attributes.input.score", floatVal)
	}

	return stateJSON
}
