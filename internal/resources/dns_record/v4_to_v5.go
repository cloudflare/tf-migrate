package dns_record

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"

	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of DNS record resources from v4 to v5
type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_record", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_dns_record"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_record"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed for DNS records
	return content
}

func (m *V4ToV5Migrator) Postprocess(content string) string {
	// No postprocessing needed - cross-file references are handled by global postprocessing
	return content
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return "cloudflare_record", "cloudflare_dns_record"
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Get the resource name before renaming (for moved block generation)
	resourceName := tfhcl.GetResourceName(block)

	// Rename cloudflare_record to cloudflare_dns_record
	tfhcl.RenameResourceType(block, "cloudflare_record", "cloudflare_dns_record")

	body := block.Body()

	// Ensure TTL is present for v5 (required field)
	tfhcl.EnsureAttribute(body, "ttl", 1)

	// Get the record type
	typeAttr := body.GetAttribute("type")
	recordType := ""
	if typeAttr != nil {
		// Extract the record type value
		recordType = tfhcl.ExtractStringFromAttribute(typeAttr)
	}

	// Complex record types that use the data field instead of content/value
	complexDataTypes := map[string]bool{
		"SRV": true, "CAA": true, "CERT": true, "DNSKEY": true, "DS": true,
		"LOC": true, "NAPTR": true, "SMIMEA": true, "SSHFP": true, "SVCB": true,
		"HTTPS": true, "TLSA": true, "URI": true,
	}

	// Rename value to content for all record types EXCEPT those that use data field
	// This handles simple types (A, AAAA, CNAME, etc.) as well as dynamic type expressions
	// where we can't determine the type statically
	if !complexDataTypes[recordType] {
		// Rename value to content
		if valueAttr := body.GetAttribute("value"); valueAttr != nil {
			// Get the expression from value attribute
			tokens := valueAttr.Expr().BuildTokens(nil)
			// Set content with the same expression
			body.SetAttributeRaw("content", tokens)
			// Remove the old value attribute
			body.RemoveAttribute("value")
		}
	}

	// Remove deprecated attributes
	tfhcl.RemoveAttributes(body, "allow_overwrite", "hostname")

	// Process data blocks
	m.processDataBlocks(block, recordType)

	// Process data attribute for CAA records
	m.processDataAttribute(block, recordType)

	// Generate moved block for state migration
	oldType, newType := m.GetResourceRename()
	from := oldType + "." + resourceName
	to := newType + "." + resourceName
	movedBlock := tfhcl.CreateMovedBlock(from, to)

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block, movedBlock},
		RemoveOriginal: true,
	}, nil
}

// processDataBlocks converts data blocks to attribute format
func (m *V4ToV5Migrator) processDataBlocks(block *hclwrite.Block, recordType string) {
	body := block.Body()

	// For SRV, MX, and URI records, hoist priority from data block to root
	// Note: SRV will keep priority in BOTH places (root and data)
	if recordType == "SRV" || recordType == "MX" || recordType == "URI" {
		tfhcl.HoistAttributeFromBlock(body, "data", "priority")
	}

	// Convert data blocks to attribute, with preprocessing for CAA records
	tfhcl.ConvertBlocksToAttribute(body, "data", "data", func(dataBlock *hclwrite.Block) {
		if recordType == "CAA" {
			// Rename content to value in CAA data blocks
			tfhcl.RenameAttribute(dataBlock.Body(), "content", "value")
			// In v5, flags format is preserved as-is (string stays string, number stays number)
		}
		// Remove priority from data block for MX/URI since it's hoisted to root only
		// SRV keeps priority in BOTH the data block AND root
		if recordType == "MX" || recordType == "URI" {
			dataBlock.Body().RemoveAttribute("priority")
		}
		// Note: For SRV, we do NOT remove priority from data block
	})
}

// processDataAttribute handles data as an attribute (not a block) for CAA records
func (m *V4ToV5Migrator) processDataAttribute(block *hclwrite.Block, recordType string) {
	dataAttr := block.Body().GetAttribute("data")
	if dataAttr != nil && recordType == "CAA" {
		expr := dataAttr.Expr()
		tokens := expr.BuildTokens(nil)

		newTokens := make(hclwrite.Tokens, 0, len(tokens))
		for i := 0; i < len(tokens); i++ {
			token := tokens[i]

			// Check if this is "content" identifier inside data - rename to "value"
			if token.Type == hclsyntax.TokenIdent && string(token.Bytes) == "content" {
				if i+1 < len(tokens) && (tokens[i+1].Type == hclsyntax.TokenEqual ||
					(string(tokens[i+1].Bytes) == " " && i+2 < len(tokens) && tokens[i+2].Type == hclsyntax.TokenEqual)) {
					valueToken := &hclwrite.Token{
						Type:  hclsyntax.TokenIdent,
						Bytes: []byte("value"),
					}
					newTokens = append(newTokens, valueToken)
				} else {
					newTokens = append(newTokens, token)
				}
			} else {
				newTokens = append(newTokens, token)
			}
		}

		block.Body().SetAttributeRaw("data", newTokens)
	}
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// State transformation is handled by the provider's StateUpgraders (MoveState/UpgradeState)
	// The moved block generated in TransformConfig triggers the provider's migration logic
	// This function is a no-op for dns_record migration
	return stateJSON.String(), nil
}

// UsesProviderStateUpgrader indicates that this resource uses provider-based state migration
func (m *V4ToV5Migrator) UsesProviderStateUpgrader() bool {
	return true
}
