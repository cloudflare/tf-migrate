package zero_trust_split_tunnel

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of cloudflare_split_tunnel resources from v4 to v5.
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

// TransformState is a no-op — state transformation is handled by the provider's StateUpgraders.
// cloudflare_split_tunnel does not exist in v5; users must run `terraform state rm` for each
// split tunnel entry after migration. The provider's MoveState + UpgradeState handlers on
// device profile resources handle the device profile state migration.
func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders (MoveState/UpgradeState).
	// The moved blocks generated in TransformConfig (on device profiles) trigger the provider's
	// migration logic. split_tunnel has no v5 equivalent resource; its state entries are removed
	// by the user via `terraform state rm` after migration.
	return stateJSON.String(), nil
}

// ProcessCrossResourceConfigMigration merges split_tunnel resources into device profile resources.
// This function is called by the device profile migrators.
func ProcessCrossResourceConfigMigration(file *hclwrite.File) {
	body := file.Body()

	// Step 1: Collect all device profiles and split tunnels
	var defaultProfileBlock *hclwrite.Block
	customProfiles := make(map[string]*hclwrite.Block)         // resource_name -> block
	splitTunnelsByParent := make(map[string][]*hclwrite.Block) // parent_name -> []tunnel_blocks
	orphanedSplitTunnels := []*hclwrite.Block{}                // Unparseable policy_id references
	orphanedDefaultTunnels := []*hclwrite.Block{}              // Default-profile tunnels with no default profile in file

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
		defaultTunnels := splitTunnelsByParent[""] // Empty string = default profile
		mergeSplitTunnelsIntoProfile(defaultTunnels, defaultProfileBlock)
	} else if len(splitTunnelsByParent[""]) > 0 {
		// Have split tunnels for the default profile but no default profile resource in this file.
		// Collect them for end-of-file warnings — the blocks will be removed in Step 4.
		orphanedDefaultTunnels = append(orphanedDefaultTunnels, splitTunnelsByParent[""]...)
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

	// Step 5b: Handle split tunnels that targeted the implicit default profile but no default profile
	// resource was declared in this file. The tunnel config is preserved in the comment so the user
	// can manually add it to a cloudflare_zero_trust_device_default_profile resource.
	for _, tunnelBlock := range orphanedDefaultTunnels {
		tunnelResourceName := tfhcl.GetResourceName(tunnelBlock)
		warningMsg := fmt.Sprintf(
			"Split tunnel %q was configured on the implicit default device profile. "+
				"Its configuration has been removed during migration. "+
				"To manage these settings in Terraform, declare a cloudflare_zero_trust_device_default_profile resource "+
				"and run `terraform import cloudflare_zero_trust_device_default_profile.<name> <account_id>`, "+
				"then add the following tunnel entries to its exclude or include attribute.",
			tunnelResourceName,
		)
		migrationWarnings = append(migrationWarnings, migrationWarning{
			message: warningMsg,
			block:   tunnelBlock,
		})
	}

	// Step 6: Handle split tunnels referencing non-existent profiles
	for profileName, tunnels := range splitTunnelsByParent {
		if profileName == "" {
			continue // Default profile already handled
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
