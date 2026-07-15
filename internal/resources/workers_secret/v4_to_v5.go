package workers_secret

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of cloudflare_worker_secret and
// cloudflare_workers_secret resources from v4 to v5.
//
// In v5, standalone worker secret resources no longer exist. Secrets are
// managed as bindings on the cloudflare_workers_script resource:
//
//	bindings = [
//	  {
//	    type = "secret_text"
//	    name = "MY_SECRET"
//	    text = "secret-value"
//	  }
//	]
//
// The workers_script migrator calls ProcessCrossResourceConfigMigration to
// merge secret resources into their parent workers_script bindings list.
type V4ToV5Migrator struct{}

type secretBinding struct {
	block *hclwrite.Block
	name  string // the secret name expression
	text  string // the secret_text expression
}

type parentScriptInfo struct {
	block      *hclwrite.Block
	scriptName string // the script_name or name attribute value
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_workers_secret", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_worker_secret", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return ""
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_workers_secret" || resourceType == "cloudflare_worker_secret"
}

func (m *V4ToV5Migrator) Preprocess(content string) string {
	return content
}

// GetResourceRename implements the ResourceRenamer interface.
// workers_secret is removed in v5 (folded into workers_script bindings).
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_workers_secret", "cloudflare_worker_secret"}, ""
}

// TransformPhaseOne implements the PhaseOneTransformer interface.
// Because cloudflare_workers_secret does not exist in the v5 provider, Terraform
// cannot read existing state entries after the provider upgrade. Phase 1 appends
// a removed {} block while the v4 provider is still active so Terraform can drop
// the state entry without destroying infrastructure. The caller comments out the
// original resource block.
func (m *V4ToV5Migrator) TransformPhaseOne(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	if len(block.Labels()) < 2 {
		return nil, fmt.Errorf("invalid resource block: expected 2 labels, got %d", len(block.Labels()))
	}
	resourceType := block.Labels()[0]
	resourceName := block.Labels()[1]

	removedAddr := resourceType + "." + resourceName
	removedBlock := tfhcl.CreateRemovedBlock(removedAddr)
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{removedBlock},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	originalResourceType := tfhcl.GetResourceType(block)
	resourceName := tfhcl.GetResourceName(block)

	// Generate a removed block for the secret resource
	from := originalResourceType + "." + resourceName
	removedBlock := tfhcl.CreateRemovedBlock(from)

	// Build the binding snippet for the diagnostic message
	body := block.Body()
	bindingSnippet := buildBindingSnippet(body)

	scriptRef := extractScriptReference(body)
	if scriptRef == "" {
		scriptRef = "(unknown)"
	}

	ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  fmt.Sprintf("Resource removed: %s.%s", originalResourceType, resourceName),
		Detail: fmt.Sprintf(`The %s resource has been removed in v5. Secrets are now managed as
bindings on the cloudflare_workers_script resource (script_name = %s).

If the parent cloudflare_workers_script is in the same file, the secret has
been automatically merged into its bindings list. Otherwise, add the
following binding to the parent resource manually:

%s

After applying, run 'terraform state rm %s' to remove the old state entry.`,
			originalResourceType, scriptRef, bindingSnippet, from),
	})

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{removedBlock},
		RemoveOriginal: true,
	}, nil
}

// ProcessCrossResourceConfigMigration merges workers_secret resources into
// their parent cloudflare_workers_script resources' bindings lists.
//
// Called from the workers_script migrator after its own transformation is
// complete, following the same pattern as zero_trust_split_tunnel.
func ProcessCrossResourceConfigMigration(file *hclwrite.File) {
	body := file.Body()

	// Collect workers_script resources and workers_secret resources
	scriptResources := make(map[string]*parentScriptInfo) // keyed by resource name
	var secretBlocks []*hclwrite.Block

	for _, block := range body.Blocks() {
		if block.Type() != "resource" || len(block.Labels()) < 2 {
			continue
		}
		resourceType := block.Labels()[0]
		resourceName := block.Labels()[1]

		switch resourceType {
		case "cloudflare_workers_script", "cloudflare_worker_script":
			// In v5 the attribute is script_name; in v4 it's name.
			// After workers_script migration runs, it will be script_name.
			sn := extractAttrString(block.Body(), "script_name")
			if sn == "" {
				sn = extractAttrString(block.Body(), "name")
			}
			scriptResources[resourceName] = &parentScriptInfo{
				block:      block,
				scriptName: sn,
			}
		case "cloudflare_workers_secret", "cloudflare_worker_secret":
			secretBlocks = append(secretBlocks, block)
		}
	}

	if len(secretBlocks) == 0 {
		return
	}

	// Group secrets by their parent script resource name
	secretsByScript := make(map[string][]secretBinding) // keyed by script resource name
	var orphanSecrets []*hclwrite.Block

	for _, secretBlock := range secretBlocks {
		secretBody := secretBlock.Body()
		parentScript := findParentScriptResource(secretBody, scriptResources)

		nameVal := extractAttrExpr(secretBody, "name")
		textVal := extractAttrExpr(secretBody, "secret_text")

		if parentScript != "" {
			secretsByScript[parentScript] = append(secretsByScript[parentScript], secretBinding{
				block: secretBlock,
				name:  nameVal,
				text:  textVal,
			})
		} else {
			orphanSecrets = append(orphanSecrets, secretBlock)
		}
	}

	// Merge secrets into parent script bindings.
	// Only merge into scripts that have already been migrated (have script_name
	// attribute). Scripts still using the v4 "name" attribute haven't had their
	// transformBindings run yet, so merging now would be overwritten. Those
	// scripts will trigger another call to this function after their own
	// transformation completes.
	for scriptName, secrets := range secretsByScript {
		info := scriptResources[scriptName]
		if info == nil {
			continue
		}

		// Check if the script has already been migrated by looking for script_name.
		// If it still has "name" (v4), skip -- it will be handled on a later call.
		if info.block.Body().GetAttribute("script_name") == nil {
			continue
		}

		mergeSecretsIntoScript(info.block, secrets)

		// Remove the secret resource blocks
		for _, s := range secrets {
			body.RemoveBlock(s.block)
		}
	}

	// For orphan secrets (parent not in same file), just remove the block.
	// The TransformConfig already emitted a removed block and diagnostic.
	for _, block := range orphanSecrets {
		body.RemoveBlock(block)
	}
}

// mergeSecretsIntoScript adds secret_text bindings to a workers_script resource.
func mergeSecretsIntoScript(scriptBlock *hclwrite.Block, secrets []secretBinding) {
	scriptBody := scriptBlock.Body()

	// Build binding objects for each secret
	var bindingObjects []string
	for _, s := range secrets {
		obj := buildBindingObject(s.name, s.text)
		bindingObjects = append(bindingObjects, obj)
	}

	// Check if the script already has a bindings attribute
	existingBindings := scriptBody.GetAttribute("bindings")
	if existingBindings != nil {
		// Append to existing bindings using concat()
		existingExpr := strings.TrimSpace(string(existingBindings.Expr().BuildTokens(nil).Bytes()))
		newBindings := "[\n  " + strings.Join(bindingObjects, ", ") + "\n]"

		var concatExpr string
		if strings.HasPrefix(existingExpr, "concat(") {
			// Already a concat expression - add our bindings as another argument
			concatExpr = existingExpr[:len(existingExpr)-1] + ", " + newBindings + ")"
		} else {
			concatExpr = "concat(" + existingExpr + ", " + newBindings + ")"
		}
		tfhcl.SetAttributeFromExpressionString(scriptBody, "bindings", concatExpr)
	} else {
		// No existing bindings - create new list
		bindingsValue := "[\n  " + strings.Join(bindingObjects, ", ") + "\n]"
		tfhcl.SetAttributeFromExpressionString(scriptBody, "bindings", bindingsValue)
	}
}

// buildBindingObject creates a v5 binding object string for a secret.
func buildBindingObject(name, text string) string {
	var attrs []string
	attrs = append(attrs, `type = "secret_text"`)
	if name != "" {
		attrs = append(attrs, "name = "+name)
	}
	if text != "" {
		attrs = append(attrs, "text = "+text)
	}
	return "{\n    " + strings.Join(attrs, "\n    ") + "\n  }"
}

// buildBindingSnippet creates a human-readable binding snippet for diagnostics.
func buildBindingSnippet(body *hclwrite.Body) string {
	name := extractAttrExpr(body, "name")
	text := extractAttrExpr(body, "secret_text")

	var lines []string
	lines = append(lines, "  {")
	lines = append(lines, `    type = "secret_text"`)
	if name != "" {
		lines = append(lines, "    name = "+name)
	}
	if text != "" {
		lines = append(lines, "    text = "+text)
	}
	lines = append(lines, "  }")
	return strings.Join(lines, "\n")
}

// findParentScriptResource finds the parent workers_script resource name
// by matching the script_name attribute of the secret to a script resource.
func findParentScriptResource(secretBody *hclwrite.Body, scripts map[string]*parentScriptInfo) string {
	scriptNameAttr := secretBody.GetAttribute("script_name")
	if scriptNameAttr == nil {
		return ""
	}

	scriptNameExpr := strings.TrimSpace(string(scriptNameAttr.Expr().BuildTokens(nil).Bytes()))

	// Check for direct reference: cloudflare_workers_script.NAME.script_name
	// or cloudflare_workers_script.NAME.name (v4 form)
	// or cloudflare_worker_script.NAME.name (v4 singular form)
	for _, prefix := range []string{
		"cloudflare_workers_script.",
		"cloudflare_worker_script.",
	} {
		if strings.HasPrefix(scriptNameExpr, prefix) {
			rest := scriptNameExpr[len(prefix):]
			// Extract resource name (before the next dot)
			parts := strings.SplitN(rest, ".", 2)
			if len(parts) >= 1 {
				resourceName := strings.TrimSpace(parts[0])
				if _, ok := scripts[resourceName]; ok {
					return resourceName
				}
			}
		}
	}

	// Check for literal script_name match
	scriptNameLiteral := extractAttrString(secretBody, "script_name")
	if scriptNameLiteral != "" {
		for name, info := range scripts {
			if info.scriptName == scriptNameLiteral {
				return name
			}
		}
	}

	return ""
}

// extractScriptReference returns the script_name expression as a string for diagnostics.
func extractScriptReference(body *hclwrite.Body) string {
	attr := body.GetAttribute("script_name")
	if attr == nil {
		return ""
	}
	return strings.TrimSpace(string(attr.Expr().BuildTokens(nil).Bytes()))
}

// extractAttrExpr returns the raw expression string for an attribute.
func extractAttrExpr(body *hclwrite.Body, name string) string {
	attr := body.GetAttribute(name)
	if attr == nil {
		return ""
	}
	return strings.TrimSpace(string(attr.Expr().BuildTokens(nil).Bytes()))
}

// extractAttrString returns the unquoted string value for an attribute, or empty if not a literal.
func extractAttrString(body *hclwrite.Body, name string) string {
	return tfhcl.ExtractStringFromAttribute(body.GetAttribute(name))
}
