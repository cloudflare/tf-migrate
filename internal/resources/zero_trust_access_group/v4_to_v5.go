package zero_trust_access_group

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
	tfhcl "github.com/cloudflare/tf-migrate/internal/transform/hcl"
)

// V4ToV5Migrator handles migration of Zero Trust Access Group resources from v4 to v5
type V4ToV5Migrator struct {
	oldType string
	newType string
}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{
		oldType: "cloudflare_access_group",
		newType: "cloudflare_zero_trust_access_group",
	}

	// Register BOTH old and new resource names
	internal.RegisterMigrator("cloudflare_access_group", "v4", "v5", migrator)
	internal.RegisterMigrator("cloudflare_zero_trust_access_group", "v4", "v5", migrator)

	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return m.newType
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == m.oldType || resourceType == m.newType
}

// GetResourceRename implements the ResourceRenamer interface
// This allows the migration tool to collect all resource renames and apply them globally
func (m *V4ToV5Migrator) GetResourceRename() ([]string, string) {
	return []string{"cloudflare_access_group", "cloudflare_zero_trust_access_group"}, m.newType
}

// Preprocess is now a no-op - all transformations are done in TransformConfig using HCL AST manipulation
// This approach properly handles both v4 formats:
// - Format A (original v4): Single block with arrays: include { email = ["a@example.com"] }
// - Format B (cf-terraforming): Multiple blocks with nested objects: include { email = { email = "a@example.com" } }
// The old regex-based approach is kept below as preprocessRuleBlocksLegacy for reference/fallback
func (m *V4ToV5Migrator) Preprocess(content string) string {
	// No preprocessing needed - all transformations done in TransformConfig with HCL AST
	return content
}

// preprocessRuleBlocksLegacy is the old regex-based approach - kept as fallback/reference
// This function has a bug: it processes each block independently, causing "Attribute redefined" errors
// when there are multiple include/exclude/require blocks (like cf-terraforming generates)
func (m *V4ToV5Migrator) preprocessRuleBlocksLegacy(content, ruleName string) string {
	// Find all rule blocks for this rule type
	// Pattern: ruleName { ... }
	// We need to match the entire block including nested content
	pattern := regexp.MustCompile(fmt.Sprintf(`(?s)(%s)\s*\{([^{}]*(?:\{[^{}]*\}[^{}]*)*)\}`, ruleName))

	// Replace each match
	content = pattern.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the block content (everything between { })
		blockPattern := regexp.MustCompile(fmt.Sprintf(`(?s)%s\s*\{(.+)\}`, ruleName))
		blockMatches := blockPattern.FindStringSubmatch(match)
		if len(blockMatches) < 2 {
			return match
		}

		blockContent := blockMatches[1]

		// Parse the block content and extract selectors
		selectors := m.parseSelectorsFromBlock(blockContent)

		// Convert selectors to v5 format
		if len(selectors) == 0 {
			// Empty rule - return as empty array
			return fmt.Sprintf("%s = []", ruleName)
		}

		// Check if we have a for expression
		if len(selectors) == 1 && strings.HasPrefix(selectors[0], "FOR_EXPRESSION:") {
			// Extract the for expression and use it directly (no array wrapper needed)
			forExpr := strings.TrimPrefix(selectors[0], "FOR_EXPRESSION:")
			return fmt.Sprintf("%s = %s", ruleName, forExpr)
		}

		// Build v5 attribute syntax with trailing comma
		v5Selectors := make([]string, 0, len(selectors))
		for _, selector := range selectors {
			v5Selectors = append(v5Selectors, selector)
		}

		// Format as attribute with array (with trailing comma for Terraform style)
		return fmt.Sprintf("%s = [\n%s,\n  ]", ruleName, strings.Join(v5Selectors, ",\n"))
	})

	return content
}

// parseSelectorsFromBlock parses a v4 rule block and returns v5 selector strings
func (m *V4ToV5Migrator) parseSelectorsFromBlock(blockContent string) []string {
	var selectors []string

	// Phase 1: Simple string array selectors (no field rename)
	// Note: Parse email_list and ip_list before email and ip to match expected ordering
	selectors = append(selectors, m.parseStringArraySelector(blockContent, "email", "email")...)
	selectors = append(selectors, m.parseStringArraySelector(blockContent, "email_list", "id")...)
	selectors = append(selectors, m.parseStringArraySelector(blockContent, "ip_list", "id")...)
	selectors = append(selectors, m.parseStringArraySelector(blockContent, "ip", "ip")...)
	selectors = append(selectors, m.parseStringArraySelector(blockContent, "group", "id")...)
	selectors = append(selectors, m.parseStringArraySelector(blockContent, "login_method", "id")...)

	// Phase 2: Simple string array selectors WITH field rename
	selectors = append(selectors, m.parseStringArraySelector(blockContent, "email_domain", "domain")...)
	selectors = append(selectors, m.parseStringArraySelector(blockContent, "geo", "country_code")...)
	selectors = append(selectors, m.parseStringArraySelector(blockContent, "device_posture", "integration_uid")...)
	selectors = append(selectors, m.parseStringArraySelector(blockContent, "service_token", "token_id")...)

	// Phase 3: Simple string scalar selectors (wrap single value)
	selectors = append(selectors, m.parseStringScalarSelector(blockContent, "common_name", "common_name")...)
	selectors = append(selectors, m.parseStringScalarSelector(blockContent, "auth_method", "auth_method")...)

	// Phase 4: Boolean selectors (convert to empty objects)
	selectors = append(selectors, m.parseBooleanSelector(blockContent, "everyone")...)
	selectors = append(selectors, m.parseBooleanSelector(blockContent, "certificate")...)
	selectors = append(selectors, m.parseBooleanSelector(blockContent, "any_valid_service_token")...)

	// Phase 5: Special case - common_names overflow array
	selectors = append(selectors, m.parseCommonNamesOverflow(blockContent)...)

	// Phase 6: Complex nested object selectors
	selectors = append(selectors, m.parseGitHubSelector(blockContent)...)
	selectors = append(selectors, m.parseGSuiteSelector(blockContent)...)
	selectors = append(selectors, m.parseAzureSelector(blockContent)...)
	selectors = append(selectors, m.parseOktaSelector(blockContent)...)
	selectors = append(selectors, m.parseSAMLSelector(blockContent)...)
	selectors = append(selectors, m.parseExternalEvaluationSelector(blockContent)...)
	selectors = append(selectors, m.parseAuthContextSelector(blockContent)...)

	return selectors
}

// parseStringArraySelector parses a string array selector and returns v5 format strings
// Example: email = ["a@example.com", "b@example.com"]
// Returns: ["    {\n      email = {\n        email = \"a@example.com\"\n      }\n    }", ...]
// Also handles for expressions: email = [for i in range(2) : "user${i}@example.com"]
func (m *V4ToV5Migrator) parseStringArraySelector(blockContent, selectorName, innerFieldName string) []string {
	var selectors []string

	// First, check if this is a for expression
	forExpr := m.detectForExpression(blockContent, selectorName)
	if forExpr != "" {
		// Transform the for expression to v5 format
		v5ForExpr := m.transformForExpression(forExpr, selectorName, innerFieldName)
		// Return as a special marker that will be handled differently
		selectors = append(selectors, "FOR_EXPRESSION:"+v5ForExpr)
		return selectors
	}

	// Pattern to match: selectorName = ["value1", "value2"]
	pattern := regexp.MustCompile(fmt.Sprintf(`%s\s*=\s*\[([^\]]+)\]`, selectorName))
	matches := pattern.FindStringSubmatch(blockContent)
	if len(matches) < 2 {
		return selectors
	}

	// Extract array values
	arrayContent := matches[1]

	// Parse individual string values
	valuePattern := regexp.MustCompile(`"([^"]*)"`)
	valueMatches := valuePattern.FindAllStringSubmatch(arrayContent, -1)

	for _, valueMatch := range valueMatches {
		if len(valueMatch) < 2 {
			continue
		}

		value := valueMatch[1]

		// Build v5 selector object
		selector := fmt.Sprintf(`    {
      %s = {
        %s = "%s"
      }
    }`, selectorName, innerFieldName, value)

		selectors = append(selectors, selector)
	}

	return selectors
}

// parseStringScalarSelector parses a string scalar selector (single value, not array)
// Example: common_name = "example.com"
// Returns: "    {\n      common_name = {\n        common_name = \"example.com\"\n      }\n    }"
func (m *V4ToV5Migrator) parseStringScalarSelector(blockContent, selectorName, innerFieldName string) []string {
	var selectors []string

	// Pattern to match: selectorName = "value"
	pattern := regexp.MustCompile(fmt.Sprintf(`%s\s*=\s*"([^"]*)"`, selectorName))
	matches := pattern.FindStringSubmatch(blockContent)
	if len(matches) < 2 {
		return selectors
	}

	value := matches[1]

	// Build v5 selector object
	selector := fmt.Sprintf(`    {
      %s = {
        %s = "%s"
      }
    }`, selectorName, innerFieldName, value)

	selectors = append(selectors, selector)
	return selectors
}

// parseBooleanSelector parses a boolean selector and converts to empty object
// Example: everyone = true
// Returns: "    {\n      everyone = {}\n    }"
func (m *V4ToV5Migrator) parseBooleanSelector(blockContent, selectorName string) []string {
	var selectors []string

	// Pattern to match: selectorName = true
	pattern := regexp.MustCompile(fmt.Sprintf(`%s\s*=\s*true`, selectorName))
	if !pattern.MatchString(blockContent) {
		return selectors
	}

	// Build v5 selector with empty object
	selector := fmt.Sprintf(`    {
      %s = {}
    }`, selectorName)

	selectors = append(selectors, selector)
	return selectors
}

// parseCommonNamesOverflow handles the special common_names overflow array
// v4 has both common_name (string) and common_names (array)
// v5 only has common_name (wrapped), so we need to expand common_names into multiple selectors
func (m *V4ToV5Migrator) parseCommonNamesOverflow(blockContent string) []string {
	var selectors []string

	// Pattern to match: common_names = ["a", "b"]
	pattern := regexp.MustCompile(`common_names\s*=\s*\[([^\]]+)\]`)
	matches := pattern.FindStringSubmatch(blockContent)
	if len(matches) < 2 {
		return selectors
	}

	// Extract array values
	arrayContent := matches[1]

	// Parse individual string values
	valuePattern := regexp.MustCompile(`"([^"]*)"`)
	valueMatches := valuePattern.FindAllStringSubmatch(arrayContent, -1)

	for _, valueMatch := range valueMatches {
		if len(valueMatch) < 2 {
			continue
		}

		value := valueMatch[1]

		// Build v5 selector object (same as common_name)
		selector := fmt.Sprintf(`    {
      common_name = {
        common_name = "%s"
      }
    }`, value)

		selectors = append(selectors, selector)
	}

	return selectors
}

// parseGitHubSelector parses github blocks and converts to github_organization with team explosion
// v4: github { name = "org", teams = ["team1", "team2"], identity_provider_id = "idp" }
// v5: Multiple selectors, one per team
func (m *V4ToV5Migrator) parseGitHubSelector(blockContent string) []string {
	var selectors []string

	// Pattern to match github blocks
	pattern := regexp.MustCompile(`(?s)github\s*\{([^}]+)\}`)
	matches := pattern.FindAllStringSubmatch(blockContent, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		githubContent := match[1]

		// Extract name
		namePattern := regexp.MustCompile(`name\s*=\s*"([^"]*)"`)
		nameMatch := namePattern.FindStringSubmatch(githubContent)
		if len(nameMatch) < 2 {
			continue
		}
		name := nameMatch[1]

		// Extract identity_provider_id (optional)
		// Matches both quoted strings and unquoted references (e.g., cloudflare_access_identity_provider.foo.id)
		idpPattern := regexp.MustCompile(`identity_provider_id\s*=\s*(.+?)(?:\n|\})`)
		idpMatch := idpPattern.FindStringSubmatch(githubContent)
		idp := ""
		if len(idpMatch) >= 2 {
			idp = strings.TrimSpace(idpMatch[1])
		}

		// Extract teams array
		teamsPattern := regexp.MustCompile(`teams\s*=\s*\[([^\]]+)\]`)
		teamsMatch := teamsPattern.FindStringSubmatch(githubContent)

		var teams []string
		if len(teamsMatch) >= 2 {
			// Parse team values
			teamValuePattern := regexp.MustCompile(`"([^"]*)"`)
			teamMatches := teamValuePattern.FindAllStringSubmatch(teamsMatch[1], -1)
			for _, tm := range teamMatches {
				if len(tm) >= 2 {
					teams = append(teams, tm[1])
				}
			}
		}

		// If no teams, create one selector without team field
		if len(teams) == 0 {
			selector := fmt.Sprintf(`    {
      github_organization = {
        name = "%s"`, name)
			if idp != "" {
				selector += fmt.Sprintf(`
        identity_provider_id = %s`, idp)
			}
			selector += `
      }
    }`
			selectors = append(selectors, selector)
		} else {
			// Create one selector per team
			for _, team := range teams {
				selector := fmt.Sprintf(`    {
      github_organization = {
        name = "%s"
        team = "%s"`, name, team)
				if idp != "" {
					selector += fmt.Sprintf(`
        identity_provider_id = %s`, idp)
				}
				selector += `
      }
    }`
				selectors = append(selectors, selector)
			}
		}
	}

	return selectors
}

// parseGSuiteSelector parses gsuite blocks
// v4: gsuite { email = ["group@example.com"], identity_provider_id = "idp" }
// v5: Takes first email only (API limitation)
func (m *V4ToV5Migrator) parseGSuiteSelector(blockContent string) []string {
	var selectors []string

	// Pattern to match gsuite blocks
	pattern := regexp.MustCompile(`(?s)gsuite\s*\{([^}]+)\}`)
	matches := pattern.FindAllStringSubmatch(blockContent, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		gsuiteContent := match[1]

		// Extract email array (take first only)
		emailPattern := regexp.MustCompile(`email\s*=\s*\[([^\]]+)\]`)
		emailMatch := emailPattern.FindStringSubmatch(gsuiteContent)
		if len(emailMatch) < 2 {
			continue
		}

		// Get first email
		emailValuePattern := regexp.MustCompile(`"([^"]*)"`)
		emailValues := emailValuePattern.FindAllStringSubmatch(emailMatch[1], -1)
		if len(emailValues) == 0 {
			continue
		}
		email := emailValues[0][1]

		// Extract identity_provider_id (optional)
		idpPattern := regexp.MustCompile(`identity_provider_id\s*=\s*"([^"]*)"`)
		idpMatch := idpPattern.FindStringSubmatch(gsuiteContent)
		idp := ""
		if len(idpMatch) >= 2 {
			idp = idpMatch[1]
		}

		selector := fmt.Sprintf(`    {
      gsuite = {
        email = "%s"`, email)
		if idp != "" {
			selector += fmt.Sprintf(`
        identity_provider_id = "%s"`, idp)
		}
		selector += `
      }
    }`
		selectors = append(selectors, selector)
	}

	return selectors
}

// parseAzureSelector parses azure blocks and renames to azure_ad
// v4: azure { id = ["id1"], identity_provider_id = "idp" }
// v5: Takes first id only, renames to azure_ad
func (m *V4ToV5Migrator) parseAzureSelector(blockContent string) []string {
	var selectors []string

	// Pattern to match azure blocks
	pattern := regexp.MustCompile(`(?s)azure\s*\{([^}]+)\}`)
	matches := pattern.FindAllStringSubmatch(blockContent, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		azureContent := match[1]

		// Extract id array (take first only)
		idPattern := regexp.MustCompile(`id\s*=\s*\[([^\]]+)\]`)
		idMatch := idPattern.FindStringSubmatch(azureContent)
		if len(idMatch) < 2 {
			continue
		}

		// Get first id
		idValuePattern := regexp.MustCompile(`"([^"]*)"`)
		idValues := idValuePattern.FindAllStringSubmatch(idMatch[1], -1)
		if len(idValues) == 0 {
			continue
		}
		id := idValues[0][1]

		// Extract identity_provider_id (optional)
		idpPattern := regexp.MustCompile(`identity_provider_id\s*=\s*"([^"]*)"`)
		idpMatch := idpPattern.FindStringSubmatch(azureContent)
		idp := ""
		if len(idpMatch) >= 2 {
			idp = idpMatch[1]
		}

		selector := fmt.Sprintf(`    {
      azure_ad = {
        id = "%s"`, id)
		if idp != "" {
			selector += fmt.Sprintf(`
        identity_provider_id = "%s"`, idp)
		}
		selector += `
      }
    }`
		selectors = append(selectors, selector)
	}

	return selectors
}

// parseOktaSelector parses okta blocks
// v4: okta { name = ["group1"], identity_provider_id = "idp" }
// v5: Takes first name only
func (m *V4ToV5Migrator) parseOktaSelector(blockContent string) []string {
	var selectors []string

	// Pattern to match okta blocks
	pattern := regexp.MustCompile(`(?s)okta\s*\{([^}]+)\}`)
	matches := pattern.FindAllStringSubmatch(blockContent, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		oktaContent := match[1]

		// Extract name array (take first only)
		namePattern := regexp.MustCompile(`name\s*=\s*\[([^\]]+)\]`)
		nameMatch := namePattern.FindStringSubmatch(oktaContent)
		if len(nameMatch) < 2 {
			continue
		}

		// Get first name
		nameValuePattern := regexp.MustCompile(`"([^"]*)"`)
		nameValues := nameValuePattern.FindAllStringSubmatch(nameMatch[1], -1)
		if len(nameValues) == 0 {
			continue
		}
		name := nameValues[0][1]

		// Extract identity_provider_id (optional)
		idpPattern := regexp.MustCompile(`identity_provider_id\s*=\s*"([^"]*)"`)
		idpMatch := idpPattern.FindStringSubmatch(oktaContent)
		idp := ""
		if len(idpMatch) >= 2 {
			idp = idpMatch[1]
		}

		selector := fmt.Sprintf(`    {
      okta = {
        name = "%s"`, name)
		if idp != "" {
			selector += fmt.Sprintf(`
        identity_provider_id = "%s"`, idp)
		}
		selector += `
      }
    }`
		selectors = append(selectors, selector)
	}

	return selectors
}

// parseSAMLSelector parses saml blocks (no structural changes)
func (m *V4ToV5Migrator) parseSAMLSelector(blockContent string) []string {
	var selectors []string

	// Pattern to match saml blocks
	pattern := regexp.MustCompile(`(?s)saml\s*\{([^}]+)\}`)
	matches := pattern.FindAllStringSubmatch(blockContent, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		samlContent := match[1]

		// Extract attribute_name
		attrPattern := regexp.MustCompile(`attribute_name\s*=\s*"([^"]*)"`)
		attrMatch := attrPattern.FindStringSubmatch(samlContent)
		if len(attrMatch) < 2 {
			continue
		}
		attrName := attrMatch[1]

		// Extract attribute_value
		valPattern := regexp.MustCompile(`attribute_value\s*=\s*"([^"]*)"`)
		valMatch := valPattern.FindStringSubmatch(samlContent)
		if len(valMatch) < 2 {
			continue
		}
		attrValue := valMatch[1]

		// Extract identity_provider_id (optional)
		idpPattern := regexp.MustCompile(`identity_provider_id\s*=\s*"([^"]*)"`)
		idpMatch := idpPattern.FindStringSubmatch(samlContent)
		idp := ""
		if len(idpMatch) >= 2 {
			idp = idpMatch[1]
		}

		selector := fmt.Sprintf(`    {
      saml = {
        attribute_name  = "%s"
        attribute_value = "%s"`, attrName, attrValue)
		if idp != "" {
			selector += fmt.Sprintf(`
        identity_provider_id = "%s"`, idp)
		}
		selector += `
      }
    }`
		selectors = append(selectors, selector)
	}

	return selectors
}

// parseExternalEvaluationSelector parses external_evaluation blocks (no structural changes)
func (m *V4ToV5Migrator) parseExternalEvaluationSelector(blockContent string) []string {
	var selectors []string

	// Pattern to match external_evaluation blocks
	pattern := regexp.MustCompile(`(?s)external_evaluation\s*\{([^}]+)\}`)
	matches := pattern.FindAllStringSubmatch(blockContent, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		extEvalContent := match[1]

		// Extract evaluate_url
		urlPattern := regexp.MustCompile(`evaluate_url\s*=\s*"([^"]*)"`)
		urlMatch := urlPattern.FindStringSubmatch(extEvalContent)
		if len(urlMatch) < 2 {
			continue
		}
		evaluateURL := urlMatch[1]

		// Extract keys_url
		keysPattern := regexp.MustCompile(`keys_url\s*=\s*"([^"]*)"`)
		keysMatch := keysPattern.FindStringSubmatch(extEvalContent)
		if len(keysMatch) < 2 {
			continue
		}
		keysURL := keysMatch[1]

		selector := fmt.Sprintf(`    {
      external_evaluation = {
        evaluate_url = "%s"
        keys_url     = "%s"
      }
    }`, evaluateURL, keysURL)
		selectors = append(selectors, selector)
	}

	return selectors
}

// parseAuthContextSelector parses auth_context blocks (no structural changes)
func (m *V4ToV5Migrator) parseAuthContextSelector(blockContent string) []string {
	var selectors []string

	// Pattern to match auth_context blocks
	pattern := regexp.MustCompile(`(?s)auth_context\s*\{([^}]+)\}`)
	matches := pattern.FindAllStringSubmatch(blockContent, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		authContent := match[1]

		// Extract ac_id
		idPattern := regexp.MustCompile(`ac_id\s*=\s*"([^"]*)"`)
		idMatch := idPattern.FindStringSubmatch(authContent)
		if len(idMatch) < 2 {
			continue
		}
		acID := idMatch[1]

		// Extract id
		idPattern2 := regexp.MustCompile(`id\s*=\s*"([^"]*)"`)
		idMatch2 := idPattern2.FindStringSubmatch(authContent)
		if len(idMatch2) < 2 {
			continue
		}
		id := idMatch2[1]

		// Extract identity_provider_id
		idpPattern := regexp.MustCompile(`identity_provider_id\s*=\s*"([^"]*)"`)
		idpMatch := idpPattern.FindStringSubmatch(authContent)
		if len(idpMatch) < 2 {
			continue
		}
		idp := idpMatch[1]

		selector := fmt.Sprintf(`    {
      auth_context = {
        ac_id                = "%s"
        id                   = "%s"
        identity_provider_id = "%s"
      }
    }`, acID, id, idp)
		selectors = append(selectors, selector)
	}

	return selectors
}

// detectForExpression checks if the array content contains a for expression
// Returns the for expression string if found, empty string otherwise
func (m *V4ToV5Migrator) detectForExpression(blockContent, selectorName string) string {
	// Pattern to match: selectorName = [for ... : ...]
	pattern := regexp.MustCompile(fmt.Sprintf(`%s\s*=\s*\[(for\s+.+?:.+?)\]`, selectorName))
	matches := pattern.FindStringSubmatch(blockContent)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

// transformForExpression converts a for expression from v4 to v5 format
// v4: email = [for i in range(2) : "user${i}@example.com"]
// v5: [for i in range(2) : { email = { email = "user${i}@example.com" } }]
func (m *V4ToV5Migrator) transformForExpression(forExpr, selectorName, innerFieldName string) string {
	// Parse the for expression using HCL parser to extract components
	// Wrap the entire expression in an attribute to make it parseable
	testCode := fmt.Sprintf("test = [%s]", forExpr)

	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(testCode), "test.hcl")

	if diags.HasErrors() {
		// Fallback: Use regex-based extraction if parsing fails
		return m.transformForExpressionRegex(forExpr, selectorName, innerFieldName)
	}

	// Extract the for expression details from the parsed AST
	body, ok := file.Body.(*hclsyntax.Body)
	if !ok || len(body.Attributes) == 0 {
		return m.transformForExpressionRegex(forExpr, selectorName, innerFieldName)
	}

	attr := body.Attributes["test"]
	tupleExpr, ok := attr.Expr.(*hclsyntax.TupleConsExpr)
	if !ok || len(tupleExpr.Exprs) == 0 {
		return m.transformForExpressionRegex(forExpr, selectorName, innerFieldName)
	}

	forExprParsed, ok := tupleExpr.Exprs[0].(*hclsyntax.ForExpr)
	if !ok {
		return m.transformForExpressionRegex(forExpr, selectorName, innerFieldName)
	}

	// Extract the variable name, collection, and value expression
	varName := forExprParsed.KeyVar
	if varName == "" {
		varName = forExprParsed.ValVar
	}

	// Get the collection expression as string
	collectionBytes := forExprParsed.CollExpr.Range().SliceBytes([]byte(testCode))
	collectionStr := string(collectionBytes)

	// Get the value expression as string
	valueBytes := forExprParsed.ValExpr.Range().SliceBytes([]byte(testCode))
	valueStr := string(valueBytes)

	// Build the v5 for expression
	// [for VAR in COLLECTION : { selector = { field = VALUE } }]
	v5ForExpr := fmt.Sprintf(`[for %s in %s : {
    %s = {
      %s = %s
    }
  }]`, varName, collectionStr, selectorName, innerFieldName, valueStr)

	return v5ForExpr
}

// transformForExpressionRegex is a fallback regex-based transformation
func (m *V4ToV5Migrator) transformForExpressionRegex(forExpr, selectorName, innerFieldName string) string {
	// Simple regex extraction for common patterns like: for i in range(N) : "value${i}"
	// Pattern: for VAR in COLLECTION : VALUE
	pattern := regexp.MustCompile(`for\s+(\w+)\s+in\s+(.+?)\s*:\s*(.+)`)
	matches := pattern.FindStringSubmatch(forExpr)

	if len(matches) < 4 {
		// If we can't parse it, return a comment indicating manual migration needed
		return fmt.Sprintf(`# MIGRATION WARNING: Complex for expression detected
    # Original: %s = [%s]
    # Please manually migrate this expression`, selectorName, forExpr)
	}

	varName := matches[1]
	collection := matches[2]
	value := strings.TrimSpace(matches[3])

	// Build the v5 for expression
	v5ForExpr := fmt.Sprintf(`[for %s in %s : {
    %s = {
      %s = %s
    }
  }]`, varName, collection, selectorName, innerFieldName, value)

	return v5ForExpr
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Capture original resource type before any modifications (for moved block generation)
	originalResourceType := tfhcl.GetResourceType(block)
	resourceName := tfhcl.GetResourceName(block)

	// Track if we need to generate a moved block
	var movedBlock *hclwrite.Block

	// Rename resource type: cloudflare_access_group → cloudflare_zero_trust_access_group
	if originalResourceType == m.oldType {
		tfhcl.RenameResourceType(block, m.oldType, m.newType)

		// Generate moved block for state migration
		_, newType := m.GetResourceRename()
		from := originalResourceType + "." + resourceName
		to := newType + "." + resourceName
		movedBlock = tfhcl.CreateMovedBlock(from, to)
	}

	body := block.Body()

	// Check for dynamic blocks for include/exclude/require which can't be auto-migrated
	dynamicConditions := m.detectDynamicConditionBlocks(body)
	if len(dynamicConditions) > 0 {
		ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  fmt.Sprintf("Dynamic block requires manual migration: cloudflare_zero_trust_access_group.%s", resourceName),
			Detail: fmt.Sprintf(`Dynamic blocks for %s cannot be automatically migrated to v5.

The v5 provider uses list attributes for include/exclude/require instead of blocks.
Dynamic blocks are not supported with list attributes.

To migrate manually:
  1. Convert dynamic blocks to static lists, OR
  2. Use for_each at the resource level instead of dynamic blocks`, strings.Join(dynamicConditions, ", ")),
		})
	}

	// 1. First convert include/exclude/require blocks to attributes
	// This handles both v4 formats:
	// - Format A: include { email = ["a@example.com"] }
	// - Format B: include { email = { email = "a@example.com" } } (cf-terraforming)
	// ConvertBlocksToArrayAttribute collects ALL blocks first, then creates single attribute
	m.convertConditionBlocksToAttributes(body)

	// 2. Transform condition attributes (boolean selectors, array expansion, etc.)
	m.transformConditionAttributes(body)

	// 3. Remove empty exclude and require arrays (v5 provider normalizes them to null)
	m.removeEmptyConditionArrays(body)

	// Build result blocks
	blocks := []*hclwrite.Block{block}
	if movedBlock != nil {
		blocks = append(blocks, movedBlock)
	}

	return &transform.TransformResult{
		Blocks:         blocks,
		RemoveOriginal: movedBlock != nil, // Remove original if we generated a moved block
	}, nil
}

// detectDynamicConditionBlocks checks for dynamic blocks for include/exclude/require
// Returns the list of dynamic block names found
func (m *V4ToV5Migrator) detectDynamicConditionBlocks(body *hclwrite.Body) []string {
	var dynamicConditions []string
	conditionNames := map[string]bool{"include": true, "exclude": true, "require": true}

	for _, block := range body.Blocks() {
		if block.Type() == "dynamic" && len(block.Labels()) > 0 {
			if conditionNames[block.Labels()[0]] {
				dynamicConditions = append(dynamicConditions, block.Labels()[0])
			}
		}
	}

	return dynamicConditions
}

// convertConditionBlocksToAttributes converts include/exclude/require blocks to attribute arrays
// This handles both v4 formats by collecting ALL blocks of the same type first
// It also handles nested blocks (like github, gsuite, etc.) by converting them to attributes
func (m *V4ToV5Migrator) convertConditionBlocksToAttributes(body *hclwrite.Body) {
	conditionNames := []string{"include", "exclude", "require"}

	for _, condName := range conditionNames {
		blocks := tfhcl.FindBlocksByType(body, condName)
		if len(blocks) == 0 {
			continue
		}

		// Build array of objects from blocks, handling nested blocks
		var objectTokens []hclwrite.Tokens
		for _, block := range blocks {
			// Convert nested blocks to attributes first (github, gsuite, azure, okta, saml)
			m.convertNestedBlocksToAttributes(block.Body())

			// Now build object from the block (which only has attributes now)
			objTokens := tfhcl.BuildObjectFromBlock(block)
			objectTokens = append(objectTokens, objTokens)
		}

		// Build array from objects and set as attribute
		arrayTokens := tfhcl.BuildArrayFromObjects(objectTokens)
		body.SetAttributeRaw(condName, arrayTokens)

		// Remove original blocks
		tfhcl.RemoveBlocksByType(body, condName)
	}
}

// convertNestedBlocksToAttributes converts nested blocks (github, gsuite, etc.) to attributes
// These blocks have array semantics in v4 but are represented as blocks
func (m *V4ToV5Migrator) convertNestedBlocksToAttributes(body *hclwrite.Body) {
	// List of block types that should be converted to array attributes
	nestedBlockTypes := []string{"github", "gsuite", "azure", "okta", "saml", "external_evaluation"}

	for _, blockType := range nestedBlockTypes {
		blocks := tfhcl.FindBlocksByType(body, blockType)
		if len(blocks) == 0 {
			continue
		}

		// Convert blocks to array of objects
		var objectTokens []hclwrite.Tokens
		for _, block := range blocks {
			objTokens := tfhcl.BuildObjectFromBlock(block)
			objectTokens = append(objectTokens, objTokens)
		}

		// Build array and set as attribute
		arrayTokens := tfhcl.BuildArrayFromObjects(objectTokens)
		body.SetAttributeRaw(blockType, arrayTokens)

		// Remove original blocks
		tfhcl.RemoveBlocksByType(body, blockType)
	}
}

// removeEmptyConditionArrays removes empty exclude and require arrays
// The v5 provider normalizes empty arrays to null, so we should omit them
func (m *V4ToV5Migrator) removeEmptyConditionArrays(body *hclwrite.Body) {
	// Only remove exclude and require, keep include even if empty
	conditionsToCheck := []string{"exclude", "require"}

	for _, attrName := range conditionsToCheck {
		attr := body.GetAttribute(attrName)
		if attr == nil {
			continue
		}

		// Parse the attribute expression to check if it's an empty array
		expr := attr.Expr()
		src := hclwrite.Format(expr.BuildTokens(nil).Bytes())

		// Parse as syntax expression
		syntaxExpr, diags := hclsyntax.ParseExpression(src, attrName, hcl.InitialPos)
		if diags.HasErrors() {
			continue
		}

		// Check if it's an empty tuple (array)
		if tup, ok := syntaxExpr.(*hclsyntax.TupleConsExpr); ok {
			if len(tup.Exprs) == 0 {
				// It's an empty array, remove the attribute
				body.RemoveAttribute(attrName)
			}
		}
	}
}

// transformConditionAttributes transforms include/exclude/require attributes
// Handles:
// 1. Boolean attributes (everyone, certificate, any_valid_service_token) -> empty objects
// 2. Array attributes (email, group, ip, email_domain, geo) -> split into multiple objects
// 3. Nested object attributes (cf-terraforming format) -> preserve and normalize
// 4. For expressions inside selectors -> transform to wrap values in v5 structure
func (m *V4ToV5Migrator) transformConditionAttributes(body *hclwrite.Body) {
	conditionAttrs := []string{"include", "exclude", "require"}

	for _, attrName := range conditionAttrs {
		attr := body.GetAttribute(attrName)
		if attr == nil {
			continue
		}

		// Parse the attribute expression
		expr := attr.Expr()
		originalTokens := expr.BuildTokens(nil)
		src := hclwrite.Format(originalTokens.Bytes())

		// Normalize IP addresses in the source before parsing
		src = []byte(m.normalizeIPsInSource(string(src)))

		// Parse as syntax expression to manipulate
		syntaxExpr, diags := hclsyntax.ParseExpression(src, attrName, hcl.InitialPos)
		if diags.HasErrors() {
			// Can't parse - leave as is
			continue
		}

		// Check if this contains a for expression that needs special handling
		// e.g., [{email = [for i in range(2) : "user${i}@example.com"]}]
		if forExprTransformed := m.tryTransformForExpressionInCondition(syntaxExpr, string(src)); forExprTransformed != "" {
			// Successfully transformed a for expression - use the result
			body.SetAttributeRaw(attrName, hclwrite.Tokens{
				&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(forExprTransformed)},
			})
			continue
		}

		// Check if this is a simple tuple/array that we can transform
		// For other complex expressions, preserve as-is
		if !m.canTransformConditionExpression(syntaxExpr) {
			continue
		}

		// Transform the expression
		transformedExpr := m.transformConditionExpression(syntaxExpr)
		if transformedExpr == nil {
			continue
		}

		// Convert back to tokens and set the attribute
		tokens := m.exprToTokens(transformedExpr)
		body.SetAttributeRaw(attrName, tokens)
	}
}

// tryTransformForExpressionInCondition checks if the condition contains an object
// with a for expression selector value and transforms it if so.
// Input: [{email = [for i in range(2) : "user${i}@example.com"]}]
// Output: [for i in range(2) : { email = { email = "user${i}@example.com" } }]
func (m *V4ToV5Migrator) tryTransformForExpressionInCondition(expr hclsyntax.Expression, src string) string {
	tup, ok := expr.(*hclsyntax.TupleConsExpr)
	if !ok || len(tup.Exprs) != 1 {
		// Only handle single-element arrays with for expressions
		return ""
	}

	obj, ok := tup.Exprs[0].(*hclsyntax.ObjectConsExpr)
	if !ok || len(obj.Items) != 1 {
		// Only handle objects with a single attribute
		return ""
	}

	item := obj.Items[0]
	selectorName := m.getKeyString(item.KeyExpr)

	// Check if the value is a for expression (directly, not wrapped in tuple)
	forExpr, ok := item.ValueExpr.(*hclsyntax.ForExpr)
	if !ok {
		return ""
	}

	// Get the inner field name for this selector
	innerFieldName := m.getSelectorInnerFieldName(selectorName)
	if innerFieldName == "" {
		return ""
	}

	// Extract components from the for expression
	varName := forExpr.ValVar
	if varName == "" {
		varName = forExpr.KeyVar
	}

	// Get the collection expression as string from the source
	collectionBytes := forExpr.CollExpr.Range().SliceBytes([]byte(src))
	collectionStr := string(collectionBytes)

	// Get the value expression as string from the source
	valueBytes := forExpr.ValExpr.Range().SliceBytes([]byte(src))
	valueStr := string(valueBytes)

	// Build the transformed for expression
	// [for VAR in COLLECTION : { selector = { field = VALUE } }]
	return fmt.Sprintf(`[for %s in %s : {
%s = {
%s = %s
}
}]`, varName, collectionStr, selectorName, innerFieldName, valueStr)
}

// getSelectorInnerFieldName returns the inner field name for a selector type
func (m *V4ToV5Migrator) getSelectorInnerFieldName(selectorName string) string {
	mapping := map[string]string{
		"email":          "email",
		"group":          "id",
		"ip":             "ip",
		"ip_list":        "id",
		"email_domain":   "domain",
		"email_list":     "id",
		"geo":            "country_code",
		"common_name":    "common_name",
		"auth_method":    "auth_method",
		"login_method":   "id",
		"device_posture": "integration_uid",
		"service_token":  "token_id",
	}
	return mapping[selectorName]
}

// canTransformConditionExpression checks if an expression can be safely transformed
// Returns false for complex expressions like for expressions that should be preserved
func (m *V4ToV5Migrator) canTransformConditionExpression(expr hclsyntax.Expression) bool {
	tup, ok := expr.(*hclsyntax.TupleConsExpr)
	if !ok {
		// Not a tuple/array - can't transform
		return false
	}

	// Check each element - if any is complex (not an object literal), don't transform
	for _, itemExpr := range tup.Exprs {
		if !m.isSimpleObjectExpression(itemExpr) {
			return false
		}
	}

	return true
}

// isSimpleObjectExpression checks if an expression is a simple object literal
// that can be safely transformed. Returns false for complex expressions.
func (m *V4ToV5Migrator) isSimpleObjectExpression(expr hclsyntax.Expression) bool {
	obj, ok := expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return false
	}

	// Check each value in the object
	for _, item := range obj.Items {
		if !m.isSimpleValue(item.ValueExpr) {
			return false
		}
	}

	return true
}

// isSimpleValue checks if a value expression is simple enough to transform
func (m *V4ToV5Migrator) isSimpleValue(expr hclsyntax.Expression) bool {
	switch e := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		// Simple literals are fine
		return true
	case *hclsyntax.TemplateExpr:
		// Template expressions (string interpolations) are fine
		return true
	case *hclsyntax.ScopeTraversalExpr:
		// Variable references are fine
		return true
	case *hclsyntax.TupleConsExpr:
		// Arrays are fine if all elements are simple
		for _, elem := range e.Exprs {
			if !m.isSimpleValue(elem) {
				return false
			}
		}
		return true
	case *hclsyntax.ObjectConsExpr:
		// Objects are fine if all values are simple
		for _, item := range e.Items {
			if !m.isSimpleValue(item.ValueExpr) {
				return false
			}
		}
		return true
	case *hclsyntax.ForExpr:
		// For expressions are complex - don't transform
		return false
	case *hclsyntax.FunctionCallExpr:
		// Function calls are complex - don't transform
		return false
	case *hclsyntax.ConditionalExpr:
		// Conditionals are complex - don't transform
		return false
	default:
		// Unknown types - be conservative, don't transform
		return false
	}
}

// transformConditionExpression transforms a condition list expression
// Input: [{everyone = true, email = ["a", "b"]}]
// Output: [{everyone = {}}, {email = {email = "a"}}, {email = {email = "b"}}]
// Also handles cf-terraforming format where values are already nested objects
func (m *V4ToV5Migrator) transformConditionExpression(expr hclsyntax.Expression) hclsyntax.Expression {
	tup, ok := expr.(*hclsyntax.TupleConsExpr)
	if !ok {
		// Not a tuple/array, can't transform
		return nil
	}

	var newExprs []hclsyntax.Expression

	for _, itemExpr := range tup.Exprs {
		obj, ok := itemExpr.(*hclsyntax.ObjectConsExpr)
		if !ok {
			// Keep non-object expressions as-is
			newExprs = append(newExprs, itemExpr)
			continue
		}

		// First, transform booleans in place
		obj = m.transformBooleans(obj)

		// Then check if we need to expand arrays or handle cf-terraforming format
		expanded := m.expandObject(obj)
		if len(expanded) > 0 {
			// Object was expanded into multiple objects
			newExprs = append(newExprs, expanded...)
		} else {
			// No expansion needed, keep original object
			newExprs = append(newExprs, obj)
		}
	}

	// Return the modified tuple
	return &hclsyntax.TupleConsExpr{
		Exprs:     newExprs,
		SrcRange:  tup.SrcRange,
		OpenRange: tup.OpenRange,
	}
}

// transformBooleans transforms boolean attributes to empty objects
// everyone = true -> everyone = {}
// certificate = false -> (removed)
func (m *V4ToV5Migrator) transformBooleans(obj *hclsyntax.ObjectConsExpr) *hclsyntax.ObjectConsExpr {
	boolAttrs := map[string]bool{
		"everyone":                true,
		"certificate":             true,
		"any_valid_service_token": true,
	}

	var newItems []hclsyntax.ObjectConsItem

	for _, item := range obj.Items {
		key := m.getKeyString(item.KeyExpr)

		if boolAttrs[key] {
			// Check if it's a boolean literal
			lit, ok := item.ValueExpr.(*hclsyntax.LiteralValueExpr)
			if ok && lit.Val.Type() == cty.Bool {
				if lit.Val.False() {
					// Remove false values
					continue
				}
				// Replace true with empty object
				newItems = append(newItems, hclsyntax.ObjectConsItem{
					KeyExpr:   item.KeyExpr,
					ValueExpr: &hclsyntax.ObjectConsExpr{},
				})
				continue
			}
		}

		// Keep other items as-is
		newItems = append(newItems, item)
	}

	return &hclsyntax.ObjectConsExpr{
		Items:    newItems,
		SrcRange: obj.SrcRange,
	}
}

// expandObject checks if an object needs expansion and returns expanded objects
// Returns nil if no expansion needed
func (m *V4ToV5Migrator) expandObject(obj *hclsyntax.ObjectConsExpr) []hclsyntax.Expression {
	var allExpanded []hclsyntax.Expression
	var remainingItems []hclsyntax.ObjectConsItem

	for _, item := range obj.Items {
		key := m.getKeyString(item.KeyExpr)

		// Handle github specially
		if key == "github" {
			expanded := m.expandGithub(item)
			if len(expanded) > 0 {
				allExpanded = append(allExpanded, expanded...)
				continue
			}
		}

		// Handle gsuite specially
		if key == "gsuite" {
			expanded := m.expandGsuite(item)
			if len(expanded) > 0 {
				allExpanded = append(allExpanded, expanded...)
				continue
			}
		}

		// Handle okta specially
		if key == "okta" {
			expanded := m.expandOkta(item)
			if len(expanded) > 0 {
				allExpanded = append(allExpanded, expanded...)
				continue
			}
		}

		// Handle saml specially
		if key == "saml" {
			expanded := m.expandSaml(item)
			if len(expanded) > 0 {
				allExpanded = append(allExpanded, expanded...)
				continue
			}
		}

		// Handle azure specially
		if key == "azure" {
			expanded := m.expandAzure(item)
			if len(expanded) > 0 {
				allExpanded = append(allExpanded, expanded...)
				continue
			}
		}

		// Handle simple array attributes
		expanded := m.expandArrayAttribute(key, item)
		if len(expanded) > 0 {
			allExpanded = append(allExpanded, expanded...)
			continue
		}

		// Keep other attributes as-is
		remainingItems = append(remainingItems, item)
	}

	// If we expanded some attributes, each remaining item becomes its own object
	if len(allExpanded) > 0 {
		for _, item := range remainingItems {
			singleItemObj := &hclsyntax.ObjectConsExpr{
				Items: []hclsyntax.ObjectConsItem{item},
			}
			allExpanded = append(allExpanded, singleItemObj)
		}
		return allExpanded
	}

	// No expansion happened
	return nil
}

// expandArrayAttribute expands array attributes like email, group, ip
// email = ["a", "b"] -> [{email = {email = "a"}}, {email = {email = "b"}}]
// Also handles cf-terraforming format where value is already nested: email = { email = "a" }
func (m *V4ToV5Migrator) expandArrayAttribute(key string, item hclsyntax.ObjectConsItem) []hclsyntax.Expression {
	// Map of attribute names to their inner field names
	arrayAttrs := map[string]string{
		"email":          "email",
		"group":          "id",
		"ip":             "ip",
		"ip_list":        "id",
		"email_domain":   "domain",
		"email_list":     "id",
		"geo":            "country_code",
		"common_name":    "common_name",
		"auth_method":    "auth_method",
		"login_method":   "id",
		"device_posture": "integration_uid",
		"service_token":  "token_id",
	}

	innerFieldName, isArrayAttr := arrayAttrs[key]
	if !isArrayAttr {
		return nil
	}

	// Check if already transformed: if value is an object with a single item matching innerFieldName, skip
	if obj, ok := item.ValueExpr.(*hclsyntax.ObjectConsExpr); ok {
		if len(obj.Items) == 1 {
			itemKey := m.getKeyString(obj.Items[0].KeyExpr)
			if itemKey == innerFieldName {
				// Already in cf-terraforming format (nested object), keep as-is
				// Just wrap it properly: {email = {email = "x"}} is correct v5 format
				return nil
			}
		}
		// It's an object but not in the expected format - might be cf-terraforming format
		// where the value is already { email = "x" }, wrap it
		return nil
	}

	var result []hclsyntax.Expression

	// Check if the value is a tuple/array
	if tup, ok := item.ValueExpr.(*hclsyntax.TupleConsExpr); ok {
		// Create a new object for each item in the array
		for _, elem := range tup.Exprs {
			// Normalize IP addresses: add /32 suffix if missing
			valueExpr := elem
			if key == "ip" {
				valueExpr = m.normalizeIPExpression(elem)
			}

			newObj := &hclsyntax.ObjectConsExpr{
				Items: []hclsyntax.ObjectConsItem{
					{
						KeyExpr: m.newKeyExpr(key),
						ValueExpr: &hclsyntax.ObjectConsExpr{
							Items: []hclsyntax.ObjectConsItem{
								{
									KeyExpr:   m.newKeyExpr(innerFieldName),
									ValueExpr: valueExpr,
								},
							},
						},
					},
				},
			}
			result = append(result, newObj)
		}
		return result
	}

	// Handle single string value (not an array)
	// common_name = "device" -> {common_name = {common_name = "device"}}
	newObj := &hclsyntax.ObjectConsExpr{
		Items: []hclsyntax.ObjectConsItem{
			{
				KeyExpr: m.newKeyExpr(key),
				ValueExpr: &hclsyntax.ObjectConsExpr{
					Items: []hclsyntax.ObjectConsItem{
						{
							KeyExpr:   m.newKeyExpr(innerFieldName),
							ValueExpr: item.ValueExpr,
						},
					},
				},
			},
		},
	}
	return []hclsyntax.Expression{newObj}
}

// expandGithub handles the special case of github blocks
// github = [{name = "org", teams = ["t1", "t2"], identity_provider_id = "id"}]
// -> Multiple github_organization objects, one per team
func (m *V4ToV5Migrator) expandGithub(item hclsyntax.ObjectConsItem) []hclsyntax.Expression {
	// Check if the value is a tuple/array
	tup, ok := item.ValueExpr.(*hclsyntax.TupleConsExpr)
	if !ok {
		return nil
	}

	var result []hclsyntax.Expression

	for _, githubExpr := range tup.Exprs {
		githubObj, ok := githubExpr.(*hclsyntax.ObjectConsExpr)
		if !ok {
			continue
		}

		// Extract fields
		var nameExpr hclsyntax.Expression
		var teamsExpr *hclsyntax.TupleConsExpr
		var identityProviderExpr hclsyntax.Expression
		var otherItems []hclsyntax.ObjectConsItem

		for _, githubItem := range githubObj.Items {
			itemKey := m.getKeyString(githubItem.KeyExpr)
			switch itemKey {
			case "name":
				nameExpr = githubItem.ValueExpr
			case "teams":
				teamsExpr, _ = githubItem.ValueExpr.(*hclsyntax.TupleConsExpr)
			case "identity_provider_id":
				identityProviderExpr = githubItem.ValueExpr
			default:
				otherItems = append(otherItems, githubItem)
			}
		}

		// Expand teams array
		if teamsExpr != nil && len(teamsExpr.Exprs) > 0 {
			for _, teamExpr := range teamsExpr.Exprs {
				items := m.buildGithubOrgItems(nameExpr, teamExpr, identityProviderExpr, otherItems)
				newObj := &hclsyntax.ObjectConsExpr{
					Items: []hclsyntax.ObjectConsItem{
						{
							KeyExpr: m.newKeyExpr("github_organization"),
							ValueExpr: &hclsyntax.ObjectConsExpr{
								Items: items,
							},
						},
					},
				}
				result = append(result, newObj)
			}
		} else {
			// No teams array, create single github_organization
			items := m.buildGithubOrgItems(nameExpr, nil, identityProviderExpr, otherItems)
			newObj := &hclsyntax.ObjectConsExpr{
				Items: []hclsyntax.ObjectConsItem{
					{
						KeyExpr: m.newKeyExpr("github_organization"),
						ValueExpr: &hclsyntax.ObjectConsExpr{
							Items: items,
						},
					},
				},
			}
			result = append(result, newObj)
		}
	}

	return result
}

// buildGithubOrgItems builds the items for a github_organization object
func (m *V4ToV5Migrator) buildGithubOrgItems(nameExpr, teamExpr, identityProviderExpr hclsyntax.Expression, otherItems []hclsyntax.ObjectConsItem) []hclsyntax.ObjectConsItem {
	var items []hclsyntax.ObjectConsItem

	if nameExpr != nil {
		items = append(items, hclsyntax.ObjectConsItem{
			KeyExpr:   m.newKeyExpr("name"),
			ValueExpr: nameExpr,
		})
	}

	if teamExpr != nil {
		items = append(items, hclsyntax.ObjectConsItem{
			KeyExpr:   m.newKeyExpr("team"),
			ValueExpr: teamExpr,
		})
	}

	if identityProviderExpr != nil {
		items = append(items, hclsyntax.ObjectConsItem{
			KeyExpr:   m.newKeyExpr("identity_provider_id"),
			ValueExpr: identityProviderExpr,
		})
	}

	items = append(items, otherItems...)
	return items
}

// expandGsuite handles gsuite blocks
// gsuite = [{email = "group@example.com", identity_provider_id = "id"}]
// -> {gsuite = {email = "group@example.com", identity_provider_id = "id"}}
func (m *V4ToV5Migrator) expandGsuite(item hclsyntax.ObjectConsItem) []hclsyntax.Expression {
	tup, ok := item.ValueExpr.(*hclsyntax.TupleConsExpr)
	if !ok {
		return nil
	}

	var result []hclsyntax.Expression

	for _, gsuiteExpr := range tup.Exprs {
		gsuiteObj, ok := gsuiteExpr.(*hclsyntax.ObjectConsExpr)
		if !ok {
			continue
		}

		newObj := &hclsyntax.ObjectConsExpr{
			Items: []hclsyntax.ObjectConsItem{
				{
					KeyExpr:   m.newKeyExpr("gsuite"),
					ValueExpr: gsuiteObj,
				},
			},
		}
		result = append(result, newObj)
	}

	return result
}

// expandOkta handles okta blocks
// okta = [{name = "group", identity_provider_id = "id"}]
// -> {okta = {name = "group", identity_provider_id = "id"}}
func (m *V4ToV5Migrator) expandOkta(item hclsyntax.ObjectConsItem) []hclsyntax.Expression {
	tup, ok := item.ValueExpr.(*hclsyntax.TupleConsExpr)
	if !ok {
		return nil
	}

	var result []hclsyntax.Expression

	for _, oktaExpr := range tup.Exprs {
		oktaObj, ok := oktaExpr.(*hclsyntax.ObjectConsExpr)
		if !ok {
			continue
		}

		newObj := &hclsyntax.ObjectConsExpr{
			Items: []hclsyntax.ObjectConsItem{
				{
					KeyExpr:   m.newKeyExpr("okta"),
					ValueExpr: oktaObj,
				},
			},
		}
		result = append(result, newObj)
	}

	return result
}

// expandSaml handles saml blocks
// saml = [{attribute_name = "name", attribute_value = "value", identity_provider_id = "id"}]
// -> {saml = {attribute_name = "name", attribute_value = "value", identity_provider_id = "id"}}
func (m *V4ToV5Migrator) expandSaml(item hclsyntax.ObjectConsItem) []hclsyntax.Expression {
	tup, ok := item.ValueExpr.(*hclsyntax.TupleConsExpr)
	if !ok {
		return nil
	}

	var result []hclsyntax.Expression

	for _, samlExpr := range tup.Exprs {
		samlObj, ok := samlExpr.(*hclsyntax.ObjectConsExpr)
		if !ok {
			continue
		}

		newObj := &hclsyntax.ObjectConsExpr{
			Items: []hclsyntax.ObjectConsItem{
				{
					KeyExpr:   m.newKeyExpr("saml"),
					ValueExpr: samlObj,
				},
			},
		}
		result = append(result, newObj)
	}

	return result
}

// expandAzure handles azure blocks
// azure = [{id = "group-id", identity_provider_id = "id"}]
// -> {azure_ad = {id = "group-id", identity_provider_id = "id"}}
func (m *V4ToV5Migrator) expandAzure(item hclsyntax.ObjectConsItem) []hclsyntax.Expression {
	tup, ok := item.ValueExpr.(*hclsyntax.TupleConsExpr)
	if !ok {
		return nil
	}

	var result []hclsyntax.Expression

	for _, azureExpr := range tup.Exprs {
		azureObj, ok := azureExpr.(*hclsyntax.ObjectConsExpr)
		if !ok {
			continue
		}

		// Rename azure to azure_ad for v5
		newObj := &hclsyntax.ObjectConsExpr{
			Items: []hclsyntax.ObjectConsItem{
				{
					KeyExpr:   m.newKeyExpr("azure_ad"),
					ValueExpr: azureObj,
				},
			},
		}
		result = append(result, newObj)
	}

	return result
}

// Helper functions

// getKeyString extracts the string value from a key expression
func (m *V4ToV5Migrator) getKeyString(keyExpr hclsyntax.Expression) string {
	switch k := keyExpr.(type) {
	case *hclsyntax.ObjectConsKeyExpr:
		if k.ForceNonLiteral {
			return ""
		}
		// Extract the key from the wrapped expression
		if scope, ok := k.Wrapped.(*hclsyntax.ScopeTraversalExpr); ok {
			if len(scope.Traversal) > 0 {
				if root, ok := scope.Traversal[0].(hcl.TraverseRoot); ok {
					return root.Name
				}
			}
		}
	case *hclsyntax.ScopeTraversalExpr:
		if len(k.Traversal) > 0 {
			if root, ok := k.Traversal[0].(hcl.TraverseRoot); ok {
				return root.Name
			}
		}
	}
	return ""
}

// newKeyExpr creates a new key expression for an object item
func (m *V4ToV5Migrator) newKeyExpr(key string) hclsyntax.Expression {
	return &hclsyntax.ObjectConsKeyExpr{
		Wrapped: &hclsyntax.ScopeTraversalExpr{
			Traversal: hcl.Traversal{
				hcl.TraverseRoot{Name: key},
			},
		},
	}
}

// exprToTokens converts a syntax expression to write tokens
func (m *V4ToV5Migrator) exprToTokens(expr hclsyntax.Expression) hclwrite.Tokens {
	if expr == nil {
		return nil
	}
	return m.buildExprTokens(expr)
}

// buildExprTokens recursively builds hclwrite tokens from hclsyntax expression
func (m *V4ToV5Migrator) buildExprTokens(expr hclsyntax.Expression) hclwrite.Tokens {
	var tokens hclwrite.Tokens

	if expr == nil {
		return tokens
	}

	switch e := expr.(type) {
	case *hclsyntax.TupleConsExpr:
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
		for i, item := range e.Exprs {
			if i > 0 {
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")})
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
			}
			tokens = append(tokens, m.buildExprTokens(item)...)
		}
		// Add trailing comma for HCL formatting convention
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")})

	case *hclsyntax.ObjectConsExpr:
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")})
		if len(e.Items) == 0 {
			// Empty object: {}
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")})
		} else {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
			for i, item := range e.Items {
				if i > 0 {
					tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
				}
				tokens = append(tokens, m.buildExprTokens(item.KeyExpr)...)
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenEqual, Bytes: []byte("=")})
				tokens = append(tokens, m.buildExprTokens(item.ValueExpr)...)
			}
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")})
		}

	case *hclsyntax.ObjectConsKeyExpr:
		if e.ForceNonLiteral {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOParen, Bytes: []byte("(")})
			tokens = append(tokens, m.buildExprTokens(e.Wrapped)...)
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCParen, Bytes: []byte(")")})
		} else {
			tokens = append(tokens, m.buildExprTokens(e.Wrapped)...)
		}

	case *hclsyntax.ScopeTraversalExpr:
		if len(e.Traversal) > 0 {
			if root, ok := e.Traversal[0].(hcl.TraverseRoot); ok {
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(root.Name)})
			}
		}

	case *hclsyntax.LiteralValueExpr:
		if e.Val.Type() == cty.String {
			strVal := e.Val.AsString()
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOQuote, Bytes: []byte("\"")})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(strVal)})
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCQuote, Bytes: []byte("\"")})
		} else if e.Val.Type() == cty.Number {
			bf := e.Val.AsBigFloat()
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNumberLit, Bytes: []byte(bf.Text('f', -1))})
		} else if e.Val.Type() == cty.Bool {
			if e.Val.True() {
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("true")})
			} else {
				tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("false")})
			}
		} else {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(e.Val.GoString())})
		}

	case *hclsyntax.TemplateExpr:
		if len(e.Parts) == 1 {
			tokens = append(tokens, m.buildExprTokens(e.Parts[0])...)
		} else {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOQuote, Bytes: []byte("\"")})
			for _, part := range e.Parts {
				switch p := part.(type) {
				case *hclsyntax.LiteralValueExpr:
					if p.Val.Type() == cty.String {
						tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(p.Val.AsString())})
					}
				case *hclsyntax.ScopeTraversalExpr:
					tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenTemplateInterp, Bytes: []byte("${")})
					for _, part := range p.Traversal {
						switch t := part.(type) {
						case hcl.TraverseRoot:
							tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(t.Name)})
						case hcl.TraverseAttr:
							tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenDot, Bytes: []byte(".")})
							tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte(t.Name)})
						case hcl.TraverseIndex:
							tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")})
							if t.Key.Type() == cty.String {
								tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOQuote, Bytes: []byte("\"")})
								tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(t.Key.AsString())})
								tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCQuote, Bytes: []byte("\"")})
							} else if t.Key.Type() == cty.Number {
								bf := t.Key.AsBigFloat()
								tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNumberLit, Bytes: []byte(bf.String())})
							}
							tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")})
						}
					}
					tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenTemplateSeqEnd, Bytes: []byte("}")})
				default:
					tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenTemplateInterp, Bytes: []byte("${")})
					tokens = append(tokens, m.buildExprTokens(p)...)
					tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenTemplateSeqEnd, Bytes: []byte("}")})
				}
			}
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCQuote, Bytes: []byte("\"")})
		}

	default:
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenComment, Bytes: []byte("/* UNKNOWN TYPE */")})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("null")})
	}

	return tokens
}

// normalizeIPExpression normalizes IP address literals in HCL expressions
func (m *V4ToV5Migrator) normalizeIPExpression(expr hclsyntax.Expression) hclsyntax.Expression {
	if lit, ok := expr.(*hclsyntax.LiteralValueExpr); ok {
		if lit.Val.Type() == cty.String {
			ipStr := lit.Val.AsString()
			if !strings.Contains(ipStr, "/") {
				return &hclsyntax.LiteralValueExpr{
					Val:      cty.StringVal(ipStr + "/32"),
					SrcRange: lit.SrcRange,
				}
			}
		}
	}
	return expr
}

// normalizeIPsInSource normalizes IP addresses in HCL source code
func (m *V4ToV5Migrator) normalizeIPsInSource(src string) string {
	ipPattern := regexp.MustCompile(`"(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(/\d+)?)"`)
	return ipPattern.ReplaceAllStringFunc(src, func(match string) string {
		ipWithCIDR := match[1 : len(match)-1]
		if strings.Contains(ipWithCIDR, "/") {
			return match
		}
		return `"` + ipWithCIDR + `/32"`
	})
}

