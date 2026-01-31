package zero_trust_split_tunnel

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of cloudflare_split_tunnel resources from v4 to v5.
// In v5, split tunnel settings are embedded in device profile resources.
// This migrator marks split_tunnel resources for removal - the actual merging
// is done by the device profile migrators.
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_split_tunnel", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	// Split tunnel is removed in v5, so return empty string
	return ""
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_split_tunnel"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// TransformConfig does nothing - removal is handled by ProcessCrossResourceConfigMigration.
// The device profile migrator calls ProcessCrossResourceConfigMigration which merges
// split tunnels into profiles and removes the split_tunnel blocks.
// We return RemoveOriginal=false because the cross-resource handler manages removal.
func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Don't mark for removal - ProcessCrossResourceConfigMigration handles it
	return &transform.TransformResult{
		Blocks:         nil,
		RemoveOriginal: false,
	}, nil
}

// TransformState returns empty state - split tunnels are merged into device profiles via ProcessCrossResourceStateMigration.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	return "", nil
}

// ProcessCrossResourceConfigMigration merges split_tunnel resources into device profile resources.
// This function is called by the device profile migrators.
func ProcessCrossResourceConfigMigration(file *hclwrite.File) {
	body := file.Body()

	// Step 1: Collect all device profiles and split tunnels
	var defaultProfileBlock *hclwrite.Block
	customProfiles := make(map[string]*hclwrite.Block)  // resource_name -> block
	splitTunnelsByParent := make(map[string][]*hclwrite.Block)  // parent_name -> []tunnel_blocks
	orphanedSplitTunnels := []*hclwrite.Block{}  // Unparseable policy_id references

	for _, block := range body.Blocks() {
		if block.Type() == "resource" && len(block.Labels()) >= 2 {
			resourceType := tfhcl.GetResourceType(block)
			resourceName := tfhcl.GetResourceName(block)

			if resourceType == "cloudflare_zero_trust_device_default_profile" {
				defaultProfileBlock = block
			} else if resourceType == "cloudflare_zero_trust_device_custom_profile" {
				customProfiles[resourceName] = block
			} else if resourceType == "cloudflare_zero_trust_device_profiles" {
				// Handle v4 resource name - after device_profiles migration runs,
				// this will be either default or custom profile
				// We check for 'default' attribute to determine type
				blockBody := block.Body()

				isExplicitDefault := false
				if tfhcl.HasAttribute(blockBody, "default") {
					defaultValue, _ := tfhcl.ExtractBoolFromAttribute(blockBody.GetAttribute("default"))
					isExplicitDefault = defaultValue
				}

				isCustomProfile := !isExplicitDefault &&
					tfhcl.HasAttribute(blockBody, "match") &&
					tfhcl.HasAttribute(blockBody, "precedence")

				if isCustomProfile {
					customProfiles[resourceName] = block
				} else {
					defaultProfileBlock = block
				}
			} else if resourceType == "cloudflare_split_tunnel" {
				parentName := extractParentProfileName(block)
				blockBody := block.Body()

				if parentName == "" && tfhcl.HasAttribute(blockBody, "policy_id") {
					// Has policy_id but couldn't parse = orphaned
					orphanedSplitTunnels = append(orphanedSplitTunnels, block)
				} else {
					// Either no policy_id (default profile) or parsed successfully
					splitTunnelsByParent[parentName] = append(splitTunnelsByParent[parentName], block)
				}
			}
		}
	}

	// Step 2: Merge default profile split tunnels
	if defaultProfileBlock != nil {
		defaultTunnels := splitTunnelsByParent[""]  // Empty string = default profile
		mergeSplitTunnelsIntoProfile(defaultTunnels, defaultProfileBlock)
	} else if len(splitTunnelsByParent[""]) > 0 {
		// Have split tunnels for default profile but no default profile resource
		for _, tunnelBlock := range splitTunnelsByParent[""] {
			tfhcl.AppendWarningComment(tunnelBlock.Body(),
				"No default device profile found - create cloudflare_zero_trust_device_default_profile resource first")
		}
	}

	// Step 3: Merge custom profile split tunnels
	for profileName, profileBlock := range customProfiles {
		tunnels := splitTunnelsByParent[profileName]
		mergeSplitTunnelsIntoProfile(tunnels, profileBlock)
	}

	// Step 4: Remove ALL split_tunnel blocks from the file
	// This must be done BEFORE adding warnings (which rebuilds the body)
	// Collect all split_tunnel blocks first, then remove them
	var splitTunnelBlocks []*hclwrite.Block
	for _, block := range body.Blocks() {
		if block.Type() == "resource" && len(block.Labels()) >= 2 {
			if tfhcl.GetResourceType(block) == "cloudflare_split_tunnel" {
				splitTunnelBlocks = append(splitTunnelBlocks, block)
			}
		}
	}
	// Now remove them in a separate pass
	for _, block := range splitTunnelBlocks {
		body.RemoveBlock(block)
	}

	// Step 5 & 6: Collect all migration warnings
	type migrationWarning struct {
		message string
		block   *hclwrite.Block
	}
	var migrationWarnings []migrationWarning

	// Step 5: Handle orphaned split tunnels (unparseable references)
	for _, tunnelBlock := range orphanedSplitTunnels {
		tunnelResourceName := tfhcl.GetResourceName(tunnelBlock)
		warningMsg := fmt.Sprintf("Split tunnel %q has unparseable policy_id reference - manual migration required", tunnelResourceName)
		migrationWarnings = append(migrationWarnings, migrationWarning{
			message: warningMsg,
			block:   tunnelBlock,
		})
	}

	// Step 6: Handle split tunnels referencing non-existent profiles
	for profileName, tunnels := range splitTunnelsByParent {
		if profileName == "" {
			continue  // Default profile already handled
		}
		if customProfiles[profileName] == nil {
			// Referenced profile doesn't exist in this file
			for _, tunnelBlock := range tunnels {
				tunnelResourceName := tfhcl.GetResourceName(tunnelBlock)
				warningMsg := fmt.Sprintf("Split tunnel %q references profile %q which was not found - manual migration required",
					tunnelResourceName, profileName)
				migrationWarnings = append(migrationWarnings, migrationWarning{
					message: warningMsg,
					block:   tunnelBlock,
				})
			}
		}
	}

	// Add all warnings at the end of the file (only if not already added)
	// Check if warnings were already added in a previous call (idempotency check)
	fileContent := string(file.Bytes())
	if !strings.Contains(fileContent, "/** MIGRATION_WARNING:") {
		for _, warning := range migrationWarnings {
			addMigrationCommentAtEndOfFile(file, warning.message, warning.block)
		}
	}
}

// extractParentProfileName extracts the device profile resource name from the policy_id reference.
// Returns empty string for default profile (no policy_id) or if reference cannot be parsed.
func extractParentProfileName(splitTunnelBlock *hclwrite.Block) string {
	body := splitTunnelBlock.Body()

	if !tfhcl.HasAttribute(body, "policy_id") {
		// No policy_id attribute = applies to default profile
		return ""
	}

	policyIdAttr := body.GetAttribute("policy_id")
	tokens := policyIdAttr.Expr().BuildTokens(nil)
	tokenStr := string(tokens.Bytes())

	// Try to extract resource name from various resource type references
	resourceTypes := []string{
		"cloudflare_zero_trust_device_custom_profile", // v5 name
		"cloudflare_zero_trust_device_profiles",       // v4 name
		"cloudflare_device_settings_policy",           // v4 deprecated name
	}

	for _, resourceType := range resourceTypes {
		if name := extractResourceNameFromReference(tokenStr, resourceType); name != "" {
			return name
		}
	}

	// Cannot parse - return empty string
	// This will cause the split tunnel to be added to orphanedSplitTunnels
	return ""
}

// extractResourceNameFromReference extracts the resource name from a Terraform reference string.
// Supports both dot notation (resource_type.name.id) and bracket notation (resource_type["name"].id).
func extractResourceNameFromReference(tokenStr, resourceType string) string {
	// Handle dot notation: resource_type.name.id
	dotPattern := resourceType + "."
	if strings.Contains(tokenStr, dotPattern) {
		parts := strings.Split(tokenStr, ".")
		if len(parts) >= 2 {
			return strings.TrimSpace(parts[1])
		}
	}

	// Handle bracket notation: resource_type["name"].id
	bracketPattern := resourceType + `["`
	if strings.Contains(tokenStr, bracketPattern) {
		start := strings.Index(tokenStr, `["`) + 2
		end := strings.Index(tokenStr[start:], `"`)
		if end > 0 {
			return tokenStr[start : start+end]
		}
	}

	return ""
}

// mergeSplitTunnelsIntoProfile merges multiple split_tunnel resources into a device profile.
// This properly handles multiple split_tunnel resources by collecting all tunnels before setting attributes.
func mergeSplitTunnelsIntoProfile(splitTunnelBlocks []*hclwrite.Block, profileBlock *hclwrite.Block) {
	if len(splitTunnelBlocks) == 0 {
		return
	}

	profileBody := profileBlock.Body()

	// Group tunnels by mode (exclude vs include)
	tunnelsByMode := make(map[string][]cty.Value)

	for _, splitTunnelBlock := range splitTunnelBlocks {
		splitTunnelBody := splitTunnelBlock.Body()

		// Get mode
		mode := "exclude" // Default mode
		if modeVal := tfhcl.ExtractStringFromAttribute(splitTunnelBody.GetAttribute("mode")); modeVal != "" {
			mode = modeVal
		}

		// Extract tunnels from this split_tunnel resource
		tunnelsBlocks := tfhcl.FindBlocksByType(splitTunnelBody, "tunnels")
		for _, tunnelBlock := range tunnelsBlocks {
			tunnelMap := make(map[string]cty.Value)
			tunnelBody := tunnelBlock.Body()

			// Extract address (required)
			if addr := tfhcl.ExtractStringFromAttribute(tunnelBody.GetAttribute("address")); addr != "" {
				tunnelMap["address"] = cty.StringVal(addr)
			}

			// Extract description (optional)
			if desc := tfhcl.ExtractStringFromAttribute(tunnelBody.GetAttribute("description")); desc != "" {
				tunnelMap["description"] = cty.StringVal(desc)
			}

			// Extract host (optional)
			if host := tfhcl.ExtractStringFromAttribute(tunnelBody.GetAttribute("host")); host != "" {
				tunnelMap["host"] = cty.StringVal(host)
			}

			if len(tunnelMap) > 0 {
				tunnelsByMode[mode] = append(tunnelsByMode[mode], cty.ObjectVal(tunnelMap))
			}
		}
	}

	// Set the merged tunnels in deterministic order (include first, then exclude)
	// This matches the order typically seen in device profiles
	if tunnels, ok := tunnelsByMode["include"]; ok && len(tunnels) > 0 {
		profileBody.SetAttributeValue("include", cty.TupleVal(tunnels))
	}
	if tunnels, ok := tunnelsByMode["exclude"]; ok && len(tunnels) > 0 {
		profileBody.SetAttributeValue("exclude", cty.TupleVal(tunnels))
	}
}

// addMigrationCommentAtEndOfFile adds a warning comment at the end of the file,
// including a multi-line comment block showing the removed resource.
func addMigrationCommentAtEndOfFile(file *hclwrite.File, message string, block *hclwrite.Block) {
	body := file.Body()

	// Add multi-line comment block with the resource content
	// Format: /** MIGRATION_WARNING: message
	//         *  resource "cloudflare_split_tunnel" "name" {
	//         *    ...
	//         *  }
	//         */

	// Get the block content as a string
	blockContent := string(block.BuildTokens(nil).Bytes())

	// Build the multi-line comment
	var multiLineComment strings.Builder
	multiLineComment.WriteString("/** MIGRATION_WARNING: ")
	multiLineComment.WriteString(message)
	multiLineComment.WriteString("\n")

	// Add each line of the block content with a "*  " prefix
	lines := strings.Split(strings.TrimSpace(blockContent), "\n")
	for _, line := range lines {
		multiLineComment.WriteString("*  ")
		multiLineComment.WriteString(line)
		multiLineComment.WriteString("\n")
	}

	multiLineComment.WriteString("*/\n\n")

	// Append the multi-line comment
	multiLineTokens := hclwrite.Tokens{
		&hclwrite.Token{
			Type:  hclsyntax.TokenComment,
			Bytes: []byte(multiLineComment.String()),
		},
	}
	body.AppendUnstructuredTokens(multiLineTokens)
}

// ProcessCrossResourceStateMigration merges split_tunnel state into device profile state.
// This function is called by the device profile migrators' Preprocess method.
func ProcessCrossResourceStateMigration(stateJSON string) string {
	result := stateJSON

	// Maps to track resources
	defaultProfiles := make(map[string]int)      // account_id -> resource index
	customProfiles := make(map[string]int)       // policy_id -> resource index
	var splitTunnels []splitTunnelStateInfo

	resources := gjson.Get(stateJSON, "resources")
	if resources.Exists() && resources.IsArray() {
		for i, resource := range resources.Array() {
			resourceType := resource.Get("type").String()

			if resourceType == "cloudflare_zero_trust_device_default_profile" {
				// Default profile - keyed by account_id
				accountID := resource.Get("instances.0.attributes.account_id").String()
				if accountID != "" {
					defaultProfiles[accountID] = i
				}
			} else if resourceType == "cloudflare_zero_trust_device_custom_profile" {
				// Custom profile - keyed by policy_id
				policyID := resource.Get("instances.0.attributes.policy_id").String()
				if policyID != "" {
					customProfiles[policyID] = i
				}
			} else if resourceType == "cloudflare_zero_trust_device_profiles" {
				// v4 device_profiles resource - need to determine if default or custom
				attrs := resource.Get("instances.0.attributes")
				if attrs.Exists() {
					defaultAttr := attrs.Get("default")
					matchAttr := attrs.Get("match")
					precedenceAttr := attrs.Get("precedence")

					isExplicitDefault := defaultAttr.Exists() && defaultAttr.Bool()
					isCustomProfile := !isExplicitDefault && matchAttr.Exists() && precedenceAttr.Exists()

					if isCustomProfile {
						// It's a custom profile - get policy_id
						// Try policy_id attribute first, then extract from compound ID if needed
						policyID := attrs.Get("policy_id").String()
						if policyID == "" {
							// Fallback: extract from ID if it has format account_id/profile_id
							id := attrs.Get("id").String()
							if slashIdx := strings.Index(id, "/"); slashIdx != -1 && slashIdx < len(id)-1 {
								policyID = id[slashIdx+1:]
							}
						}
						if policyID != "" {
							customProfiles[policyID] = i
						}
					} else {
						// It's a default profile
						accountID := attrs.Get("account_id").String()
						if accountID != "" {
							defaultProfiles[accountID] = i
						}
					}
				}
			} else if resourceType == "cloudflare_split_tunnel" {
				tunnel := extractSplitTunnelStateInfo(resource)
				if tunnel != nil {
					splitTunnels = append(splitTunnels, *tunnel)
				}
			}
		}
	}

	// Merge split tunnels into their parent profiles
	for _, tunnel := range splitTunnels {
		if tunnel.policyID == "" {
			// No policy_id = default profile
			if profileIndex, exists := defaultProfiles[tunnel.accountID]; exists {
				result = mergeTunnelsIntoProfileState(result, profileIndex, tunnel.mode, []splitTunnelStateInfo{tunnel})
			}
		} else {
			// Has policy_id = custom profile
			if profileIndex, exists := customProfiles[tunnel.policyID]; exists {
				result = mergeTunnelsIntoProfileState(result, profileIndex, tunnel.mode, []splitTunnelStateInfo{tunnel})
			}
		}
	}

	// Remove all split_tunnel resources from state
	result = removeSplitTunnelResourcesFromState(result)

	return result
}

// splitTunnelStateInfo holds state information extracted from a split_tunnel resource
type splitTunnelStateInfo struct {
	accountID string
	policyID  string   // empty string for default profile
	mode      string   // "exclude" or "include"
	tunnels   []map[string]interface{}
}

// extractSplitTunnelStateInfo extracts split tunnel state data from a resource
func extractSplitTunnelStateInfo(resource gjson.Result) *splitTunnelStateInfo {
	attrs := resource.Get("instances.0.attributes")
	if !attrs.Exists() {
		return nil
	}

	info := &splitTunnelStateInfo{
		accountID: attrs.Get("account_id").String(),
		policyID:  attrs.Get("policy_id").String(),
		mode:      attrs.Get("mode").String(),
		tunnels:   []map[string]interface{}{},
	}

	// Default mode if not specified
	if info.mode == "" {
		info.mode = "exclude"
	}

	// Extract tunnels array
	tunnelsData := attrs.Get("tunnels")
	if tunnelsData.Exists() && tunnelsData.IsArray() {
		for _, tunnel := range tunnelsData.Array() {
			tunnelMap := make(map[string]interface{})

			// Extract optional string fields
			extractOptionalStringField(tunnel, "address", tunnelMap)
			extractOptionalStringField(tunnel, "description", tunnelMap)
			extractOptionalStringField(tunnel, "host", tunnelMap)

			if len(tunnelMap) > 0 {
				info.tunnels = append(info.tunnels, tunnelMap)
			}
		}
	}

	return info
}

// extractOptionalStringField extracts a string field from gjson.Result if it exists and is non-empty.
func extractOptionalStringField(source gjson.Result, fieldName string, target map[string]interface{}) {
	if value := source.Get(fieldName); value.Exists() && value.String() != "" {
		target[fieldName] = value.String()
	}
}

// mergeTunnelsIntoProfileState merges split tunnel data into a device profile's state
func mergeTunnelsIntoProfileState(jsonStr string, profileIndex int, mode string, tunnels []splitTunnelStateInfo) string {
	result := jsonStr

	// Build the path to the mode attribute (exclude or include)
	modePath := fmt.Sprintf("resources.%d.instances.0.attributes.%s", profileIndex, mode)
	existingTunnels := gjson.Get(jsonStr, modePath)

	var allTunnels []interface{}

	// Get existing tunnels if any
	if existingTunnels.Exists() && existingTunnels.IsArray() {
		for _, tunnel := range existingTunnels.Array() {
			allTunnels = append(allTunnels, tunnel.Value())
		}
	}

	// Add tunnels from split_tunnel resources
	for _, splitTunnel := range tunnels {
		for _, tunnel := range splitTunnel.tunnels {
			allTunnels = append(allTunnels, tunnel)
		}
	}

	// Set the merged tunnels in the profile
	if len(allTunnels) > 0 {
		result, _ = sjson.Set(result, modePath, allTunnels)
	}

	return result
}

// removeSplitTunnelResourcesFromState removes all cloudflare_split_tunnel resources from state
func removeSplitTunnelResourcesFromState(jsonStr string) string {
	result := jsonStr

	resources := gjson.Get(jsonStr, "resources")
	if !resources.Exists() || !resources.IsArray() {
		return result
	}

	var indicesToRemove []int
	for i, resource := range resources.Array() {
		resourceType := resource.Get("type").String()
		if resourceType == "cloudflare_split_tunnel" {
			indicesToRemove = append(indicesToRemove, i)
		}
	}

	// Remove in reverse order to preserve indices
	for i := len(indicesToRemove) - 1; i >= 0; i-- {
		resourcePath := fmt.Sprintf("resources.%d", indicesToRemove[i])
		result, _ = sjson.Delete(result, resourcePath)
	}

	return result
}
