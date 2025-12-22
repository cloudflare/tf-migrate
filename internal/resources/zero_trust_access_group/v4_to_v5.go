package zero_trust_access_group

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

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
func (m *V4ToV5Migrator) GetResourceRename() (string, string) {
	return m.oldType, m.newType
}

// Preprocess transforms v4 block syntax to v5 attribute syntax with array explosion
// This is the most complex part of the migration
func (m *V4ToV5Migrator) Preprocess(content string) string {
	// Process each rule type (include, exclude, require)
	content = m.preprocessRuleBlocks(content, "include")
	content = m.preprocessRuleBlocks(content, "exclude")
	content = m.preprocessRuleBlocks(content, "require")

	return content
}

// preprocessRuleBlocks finds and transforms rule blocks (include/exclude/require)
func (m *V4ToV5Migrator) preprocessRuleBlocks(content, ruleName string) string {
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
		idpPattern := regexp.MustCompile(`identity_provider_id\s*=\s*"([^"]*)"`)
		idpMatch := idpPattern.FindStringSubmatch(githubContent)
		idp := ""
		if len(idpMatch) >= 2 {
			idp = idpMatch[1]
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
        identity_provider_id = "%s"`, idp)
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
        identity_provider_id = "%s"`, idp)
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
	// Rename resource type: cloudflare_access_group → cloudflare_zero_trust_access_group
	tfhcl.RenameResourceType(block, m.oldType, m.newType)

	// Note: Most transformation happens in Preprocess() due to complexity
	// This function handles any remaining HCL-level transformations

	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath, resourceName string) (string, error) {
	// This function receives a single instance and returns the transformed instance JSON
	result := stateJSON.String()

	// Transform the instance attributes
	attrs := stateJSON.Get("attributes")
	if !attrs.Exists() {
		// Set schema_version even for instances without attributes
		result, _ = sjson.Set(result, "schema_version", 0)
		return result, nil
	}

	// Transform include rules
	result = m.transformRule(result, attrs, "include")

	// Transform exclude rules
	result = m.transformRule(result, attrs, "exclude")

	// Transform require rules
	result = m.transformRule(result, attrs, "require")

	// Ensure schema_version is 0 (MANDATORY for v5)
	result, _ = sjson.Set(result, "schema_version", 0)

	return result, nil
}

// transformRule transforms a single rule (include, exclude, or require) from v4 to v5 format
func (m *V4ToV5Migrator) transformRule(result string, attrs gjson.Result, ruleName string) string {
	ruleField := attrs.Get(ruleName)
	if !ruleField.Exists() || !ruleField.IsArray() {
		return result
	}

	// v4 has a single rule object containing arrays
	// v5 needs multiple rule objects, each containing one selector
	var v5Selectors []interface{}

	ruleField.ForEach(func(_, v4Rule gjson.Result) bool {
		// Explode this v4 rule into multiple v5 selectors
		selectors := m.explodeRuleSelectors(v4Rule)
		v5Selectors = append(v5Selectors, selectors...)
		return true
	})

	// Set or delete based on whether we have selectors
	rulePath := "attributes." + ruleName
	if len(v5Selectors) > 0 {
		result, _ = sjson.Set(result, rulePath, v5Selectors)
	} else {
		result, _ = sjson.Delete(result, rulePath)
	}

	return result
}

// explodeRuleSelectors takes a v4 rule object and returns a slice of v5 selector objects
// This is the core transformation logic for state
func (m *V4ToV5Migrator) explodeRuleSelectors(v4Rule gjson.Result) []interface{} {
	var selectors []interface{}

	// Phase 1: Simple string array selectors (no field rename)
	// Note: Parse email_list and ip_list before email and ip to match expected ordering
	selectors = append(selectors, m.explodeStringArrayState(v4Rule, "email", "email")...)
	selectors = append(selectors, m.explodeStringArrayState(v4Rule, "email_list", "id")...)
	selectors = append(selectors, m.explodeStringArrayState(v4Rule, "ip_list", "id")...)
	selectors = append(selectors, m.explodeStringArrayState(v4Rule, "ip", "ip")...)
	selectors = append(selectors, m.explodeStringArrayState(v4Rule, "group", "id")...)
	selectors = append(selectors, m.explodeStringArrayState(v4Rule, "login_method", "id")...)

	// Phase 2: String array selectors WITH field rename
	selectors = append(selectors, m.explodeStringArrayState(v4Rule, "email_domain", "domain")...)
	selectors = append(selectors, m.explodeStringArrayState(v4Rule, "geo", "country_code")...)
	selectors = append(selectors, m.explodeStringArrayState(v4Rule, "device_posture", "integration_uid")...)
	selectors = append(selectors, m.explodeStringArrayState(v4Rule, "service_token", "token_id")...)

	// Phase 3: String scalar selectors
	selectors = append(selectors, m.explodeStringScalarState(v4Rule, "common_name", "common_name")...)
	selectors = append(selectors, m.explodeStringScalarState(v4Rule, "auth_method", "auth_method")...)

	// Phase 4: Boolean selectors
	selectors = append(selectors, m.explodeBooleanState(v4Rule, "everyone")...)
	selectors = append(selectors, m.explodeBooleanState(v4Rule, "certificate")...)
	selectors = append(selectors, m.explodeBooleanState(v4Rule, "any_valid_service_token")...)

	// Phase 5: common_names overflow
	selectors = append(selectors, m.explodeCommonNamesState(v4Rule)...)

	// Phase 6: Complex nested selectors
	selectors = append(selectors, m.explodeGitHubState(v4Rule)...)
	selectors = append(selectors, m.explodeGSuiteState(v4Rule)...)
	selectors = append(selectors, m.explodeAzureState(v4Rule)...)
	selectors = append(selectors, m.explodeOktaState(v4Rule)...)
	selectors = append(selectors, m.explodeSAMLState(v4Rule)...)
	selectors = append(selectors, m.explodeExternalEvaluationState(v4Rule)...)
	selectors = append(selectors, m.explodeAuthContextState(v4Rule)...)

	return selectors
}

// explodeStringArrayState explodes a string array into multiple wrapped selectors
func (m *V4ToV5Migrator) explodeStringArrayState(v4Rule gjson.Result, selectorName, innerFieldName string) []interface{} {
	var selectors []interface{}

	arr := v4Rule.Get(selectorName)
	if !arr.Exists() || !arr.IsArray() {
		return selectors
	}

	arr.ForEach(func(_, value gjson.Result) bool {
		selectors = append(selectors, map[string]interface{}{
			selectorName: map[string]interface{}{
				innerFieldName: value.String(),
			},
		})
		return true
	})

	return selectors
}

// explodeStringScalarState wraps a scalar string value
func (m *V4ToV5Migrator) explodeStringScalarState(v4Rule gjson.Result, selectorName, innerFieldName string) []interface{} {
	var selectors []interface{}

	value := v4Rule.Get(selectorName)
	if !value.Exists() || value.String() == "" {
		return selectors
	}

	selectors = append(selectors, map[string]interface{}{
		selectorName: map[string]interface{}{
			innerFieldName: value.String(),
		},
	})

	return selectors
}

// explodeBooleanState converts boolean true to empty object
func (m *V4ToV5Migrator) explodeBooleanState(v4Rule gjson.Result, selectorName string) []interface{} {
	var selectors []interface{}

	value := v4Rule.Get(selectorName)
	if !value.Exists() || !value.Bool() {
		return selectors
	}

	selectors = append(selectors, map[string]interface{}{
		selectorName: map[string]interface{}{},
	})

	return selectors
}

// explodeCommonNamesState handles common_names overflow array
func (m *V4ToV5Migrator) explodeCommonNamesState(v4Rule gjson.Result) []interface{} {
	var selectors []interface{}

	arr := v4Rule.Get("common_names")
	if !arr.Exists() || !arr.IsArray() {
		return selectors
	}

	arr.ForEach(func(_, value gjson.Result) bool {
		selectors = append(selectors, map[string]interface{}{
			"common_name": map[string]interface{}{
				"common_name": value.String(),
			},
		})
		return true
	})

	return selectors
}

// explodeGitHubState handles github → github_organization with teams explosion
func (m *V4ToV5Migrator) explodeGitHubState(v4Rule gjson.Result) []interface{} {
	var selectors []interface{}

	githubArr := v4Rule.Get("github")
	if !githubArr.Exists() || !githubArr.IsArray() {
		return selectors
	}

	githubArr.ForEach(func(_, github gjson.Result) bool {
		name := github.Get("name").String()
		if name == "" {
			return true
		}

		idp := github.Get("identity_provider_id").String()

		// Check for teams array
		teams := github.Get("teams")
		if teams.Exists() && teams.IsArray() && len(teams.Array()) > 0 {
			// Create one selector per team
			teams.ForEach(func(_, team gjson.Result) bool {
				selector := map[string]interface{}{
					"github_organization": map[string]interface{}{
						"name": name,
						"team": team.String(),
					},
				}
				if idp != "" {
					selector["github_organization"].(map[string]interface{})["identity_provider_id"] = idp
				}
				selectors = append(selectors, selector)
				return true
			})
		} else {
			// No teams, create single selector
			selector := map[string]interface{}{
				"github_organization": map[string]interface{}{
					"name": name,
				},
			}
			if idp != "" {
				selector["github_organization"].(map[string]interface{})["identity_provider_id"] = idp
			}
			selectors = append(selectors, selector)
		}

		return true
	})

	return selectors
}

// explodeGSuiteState handles gsuite (take first email only)
func (m *V4ToV5Migrator) explodeGSuiteState(v4Rule gjson.Result) []interface{} {
	var selectors []interface{}

	gsuiteArr := v4Rule.Get("gsuite")
	if !gsuiteArr.Exists() || !gsuiteArr.IsArray() {
		return selectors
	}

	gsuiteArr.ForEach(func(_, gsuite gjson.Result) bool {
		emailArr := gsuite.Get("email")
		if !emailArr.Exists() || !emailArr.IsArray() || len(emailArr.Array()) == 0 {
			return true
		}

		// Take first email only
		email := emailArr.Array()[0].String()
		idp := gsuite.Get("identity_provider_id").String()

		selector := map[string]interface{}{
			"gsuite": map[string]interface{}{
				"email": email,
			},
		}
		if idp != "" {
			selector["gsuite"].(map[string]interface{})["identity_provider_id"] = idp
		}
		selectors = append(selectors, selector)

		return true
	})

	return selectors
}

// explodeAzureState handles azure → azure_ad (take first id only)
func (m *V4ToV5Migrator) explodeAzureState(v4Rule gjson.Result) []interface{} {
	var selectors []interface{}

	azureArr := v4Rule.Get("azure")
	if !azureArr.Exists() || !azureArr.IsArray() {
		return selectors
	}

	azureArr.ForEach(func(_, azure gjson.Result) bool {
		idArr := azure.Get("id")
		if !idArr.Exists() || !idArr.IsArray() || len(idArr.Array()) == 0 {
			return true
		}

		// Take first id only
		id := idArr.Array()[0].String()
		idp := azure.Get("identity_provider_id").String()

		selector := map[string]interface{}{
			"azure_ad": map[string]interface{}{
				"id": id,
			},
		}
		if idp != "" {
			selector["azure_ad"].(map[string]interface{})["identity_provider_id"] = idp
		}
		selectors = append(selectors, selector)

		return true
	})

	return selectors
}

// explodeOktaState handles okta (take first name only)
func (m *V4ToV5Migrator) explodeOktaState(v4Rule gjson.Result) []interface{} {
	var selectors []interface{}

	oktaArr := v4Rule.Get("okta")
	if !oktaArr.Exists() || !oktaArr.IsArray() {
		return selectors
	}

	oktaArr.ForEach(func(_, okta gjson.Result) bool {
		nameArr := okta.Get("name")
		if !nameArr.Exists() || !nameArr.IsArray() || len(nameArr.Array()) == 0 {
			return true
		}

		// Take first name only
		name := nameArr.Array()[0].String()
		idp := okta.Get("identity_provider_id").String()

		selector := map[string]interface{}{
			"okta": map[string]interface{}{
				"name": name,
			},
		}
		if idp != "" {
			selector["okta"].(map[string]interface{})["identity_provider_id"] = idp
		}
		selectors = append(selectors, selector)

		return true
	})

	return selectors
}

// explodeSAMLState handles saml (no structural changes)
func (m *V4ToV5Migrator) explodeSAMLState(v4Rule gjson.Result) []interface{} {
	var selectors []interface{}

	samlArr := v4Rule.Get("saml")
	if !samlArr.Exists() || !samlArr.IsArray() {
		return selectors
	}

	samlArr.ForEach(func(_, saml gjson.Result) bool {
		attrName := saml.Get("attribute_name").String()
		attrValue := saml.Get("attribute_value").String()
		if attrName == "" || attrValue == "" {
			return true
		}

		idp := saml.Get("identity_provider_id").String()

		selector := map[string]interface{}{
			"saml": map[string]interface{}{
				"attribute_name":  attrName,
				"attribute_value": attrValue,
			},
		}
		if idp != "" {
			selector["saml"].(map[string]interface{})["identity_provider_id"] = idp
		}
		selectors = append(selectors, selector)

		return true
	})

	return selectors
}

// explodeExternalEvaluationState handles external_evaluation (no structural changes)
func (m *V4ToV5Migrator) explodeExternalEvaluationState(v4Rule gjson.Result) []interface{} {
	var selectors []interface{}

	extEvalArr := v4Rule.Get("external_evaluation")
	if !extEvalArr.Exists() || !extEvalArr.IsArray() {
		return selectors
	}

	extEvalArr.ForEach(func(_, extEval gjson.Result) bool {
		evaluateURL := extEval.Get("evaluate_url").String()
		keysURL := extEval.Get("keys_url").String()
		if evaluateURL == "" || keysURL == "" {
			return true
		}

		selectors = append(selectors, map[string]interface{}{
			"external_evaluation": map[string]interface{}{
				"evaluate_url": evaluateURL,
				"keys_url":     keysURL,
			},
		})

		return true
	})

	return selectors
}

// explodeAuthContextState handles auth_context (no structural changes)
func (m *V4ToV5Migrator) explodeAuthContextState(v4Rule gjson.Result) []interface{} {
	var selectors []interface{}

	authArr := v4Rule.Get("auth_context")
	if !authArr.Exists() || !authArr.IsArray() {
		return selectors
	}

	authArr.ForEach(func(_, auth gjson.Result) bool {
		acID := auth.Get("ac_id").String()
		id := auth.Get("id").String()
		idp := auth.Get("identity_provider_id").String()
		if acID == "" || id == "" || idp == "" {
			return true
		}

		selectors = append(selectors, map[string]interface{}{
			"auth_context": map[string]interface{}{
				"ac_id":                acID,
				"id":                   id,
				"identity_provider_id": idp,
			},
		})

		return true
	})

	return selectors
}
