package page_rule

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

type V4ToV5Migrator struct{}

func NewV4ToV5Migrator() transform.ResourceTransformer {
	migrator := &V4ToV5Migrator{}
	internal.RegisterMigrator("cloudflare_page_rule", "v4", "v5", migrator)
	return migrator
}

func (m *V4ToV5Migrator) GetResourceType() string {
	return "cloudflare_page_rule"
}

func (m *V4ToV5Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_page_rule"
}

// Preprocess handles string-level transformations before HCL parsing
// This is necessary for consolidating duplicate cache_ttl_by_status entries
func (m *V4ToV5Migrator) Preprocess(content string) string {
	// Remove minify completely - it's not supported in v5
	content = m.removeMinifyFromActions(content)

	// Consolidate multiple cache_ttl_by_status entries into a single map
	content = m.consolidateCacheTTLByStatus(content)

	return content
}

func (m *V4ToV5Migrator) removeMinifyFromActions(content string) string {
	// Pattern to match minify blocks
	// Match minify attribute with its block, including the newline before it
	// but preserve the newline at the end to keep attributes separated
	minifyPattern := regexp.MustCompile(`(?ms)\n\s*minify\s*=\s*\{[^{}]*\}`)
	content = minifyPattern.ReplaceAllString(content, "")
	return content
}

func (m *V4ToV5Migrator) consolidateCacheTTLByStatus(content string) string {
	// Find page_rule resources and process their actions
	pageRulePattern := regexp.MustCompile(`(?ms)resource\s+"cloudflare_page_rule"[^{]+\{`)

	// Check if we have any page rules to process
	if !pageRulePattern.MatchString(content) {
		return content
	}

	// First, find all cache_ttl_by_status blocks and collect the data
	// Pattern matches the multiline structure:
	// cache_ttl_by_status = {
	//   codes = "XXX"
	//   ttl   = YYY
	// }
	ttlBlockPattern := regexp.MustCompile(`(?ms)cache_ttl_by_status\s*=\s*\{\s*codes\s*=\s*"([^"]+)"\s*ttl\s*=\s*(\d+)\s*\}`)

	matches := ttlBlockPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return content
	}

	// Collect all the code-ttl pairs
	pairs := make(map[string]string)
	for _, match := range matches {
		if len(match) >= 3 {
			code := match[1]
			ttl := match[2]
			pairs[code] = ttl
		}
	}

	if len(pairs) == 0 {
		return content
	}

	// Sort codes for consistent output
	var codes []string
	for code := range pairs {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	// Build the consolidated map
	var mapEntries []string
	for _, code := range codes {
		mapEntries = append(mapEntries, fmt.Sprintf(`"%s" = %s`, code, pairs[code]))
	}

	newTTL := "cache_ttl_by_status = { " + strings.Join(mapEntries, ", ") + " }"

	// Remove all old cache_ttl_by_status blocks
	// Match newline + whitespace + the block itself, preserving newline at end
	ttlRemovePattern := regexp.MustCompile(`(?ms)\n\s*cache_ttl_by_status\s*=\s*\{[^{}]*\}`)
	content = ttlRemovePattern.ReplaceAllString(content, "")

	// Now insert the consolidated one after cache_level
	// Find the cache_level line and add our new TTL after it
	cacheLevelPattern := regexp.MustCompile(`(cache_level\s*=\s*"[^"]+")`)
	replaced := false
	content = cacheLevelPattern.ReplaceAllStringFunc(content, func(match string) string {
		if !replaced {
			replaced = true
			return match + "\n    " + newTTL
		}
		return match
	})

	return content
}

func (m *V4ToV5Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Preprocessing handles all transformations for page_rule
	// No additional AST transformation needed
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *V4ToV5Migrator) TransformState(ctx *transform.Context, stateJSON gjson.Result, resourcePath string) (string, error) {
	// No state transformation needed for page_rule
	return stateJSON.String(), nil
}
