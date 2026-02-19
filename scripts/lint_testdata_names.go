package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

const (
	testdataDir = "integration/v4_to_v5/testdata"
	prefix      = "cftftest"
)

type LintError struct {
	File      string
	Line      int
	Column    int
	Resource  string
	Field     string
	Value     string
	Message   string
}

func (e LintError) String() string {
	return fmt.Sprintf("%s:%d:%d: %s (resource: %s, field: %s, value: %q)",
		e.File, e.Line, e.Column, e.Message, e.Resource, e.Field, e.Value)
}

type Linter struct {
	errors []LintError
	parser *hclparse.Parser
}

func NewLinter() *Linter {
	return &Linter{
		errors: []LintError{},
		parser: hclparse.NewParser(),
	}
}

func (l *Linter) AddError(err LintError) {
	l.errors = append(l.errors, err)
}

func (l *Linter) HasErrors() bool {
	return len(l.errors) > 0
}

func (l *Linter) PrintErrors() {
	for _, err := range l.errors {
		fmt.Println(err.String())
	}
}

// shouldCheckField determines if a field should be checked for the prefix
func shouldCheckField(resourceType, fieldName string) bool {
	// Fields that should be checked for prefix
	checkFields := map[string][]string{
		"cloudflare_zone":                     {"zone", "name"},
		"cloudflare_r2_bucket":                {"name"},
		"cloudflare_workers_kv_namespace":     {"title"},
		"cloudflare_pages_project":            {"name"},
		"cloudflare_api_token":                {"name"},
		"cloudflare_record":                   {"name"},
		"cloudflare_dns_record":               {"name"},
		"cloudflare_zero_trust_tunnel":        {"name"},
		"cloudflare_tunnel":                   {"name"},
		"cloudflare_notification_policy":      {"name"},
		"cloudflare_logpush_job":              {"name"},
		"cloudflare_zero_trust_list":          {"name"},
		"cloudflare_zero_trust_gateway_policy": {"name"},
		"cloudflare_dlp_profile":              {"name"},
		"cloudflare_device_posture_rule":      {"name"},
		"cloudflare_access_service_token":     {"name"},
	}

	fields, ok := checkFields[resourceType]
	if !ok {
		return false
	}

	for _, f := range fields {
		if f == fieldName {
			return true
		}
	}
	return false
}

// shouldCheckLocal determines if a local value should be checked
func shouldCheckLocal(localName string) bool {
	// Locals that commonly contain resource name prefixes
	checkLocals := []string{
		"name_prefix",
		"namespace_prefix",
		"subdomain_prefix",
	}

	for _, l := range checkLocals {
		if l == localName {
			return true
		}
	}
	return false
}

// hasPrefix checks if a value contains the required prefix
// It handles various cases: direct strings, interpolations, functions, etc.
func hasPrefix(value string) bool {
	// Already has the prefix
	if strings.Contains(value, prefix) {
		return true
	}

	// Special cases that are allowed without prefix
	allowedPatterns := []string{
		"@",                    // Root domain in DNS
		"^_",                   // DNS service records
		"^\\$\\{",              // Pure interpolation (will get prefix from local)
		"^join\\(",             // Function calls that might use locals
		"^concat\\(",
		"^format\\(",
		"each\\.value",         // Dynamic values from for_each
		"each\\.key",
		"count\\.index",        // Dynamic values from count
		"var\\.",               // Variable references
		"local\\.",             // Local references
		"\\$\\{local",          // Template with local reference
		"\\$\\{var",            // Template with variable reference
		"\\$\\{each",           // Template with each reference
	}

	for _, pattern := range allowedPatterns {
		matched, _ := regexp.MatchString(pattern, value)
		if matched {
			return true
		}
	}

	return false
}

// extractStringValue extracts the actual string value from an HCL expression
func extractStringValue(expr hclsyntax.Expression) (string, bool) {
	switch e := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		if e.Val.Type().FriendlyName() == "string" {
			return e.Val.AsString(), true
		}
	case *hclsyntax.TemplateExpr:
		// For template expressions, concatenate all literal parts
		var parts []string
		for _, part := range e.Parts {
			if lit, ok := part.(*hclsyntax.LiteralValueExpr); ok {
				if lit.Val.Type().FriendlyName() == "string" {
					parts = append(parts, lit.Val.AsString())
				}
			} else if scope, ok := part.(*hclsyntax.ScopeTraversalExpr); ok {
				// Handle variable/local references
				parts = append(parts, "${"+scope.AsTraversal().RootName()+"}")
			} else {
				// Other expressions - return a placeholder
				return "${expr}", true
			}
		}
		return strings.Join(parts, ""), true
	case *hclsyntax.FunctionCallExpr:
		// For function calls, return a representation
		return fmt.Sprintf("%s()", e.Name), true
	case *hclsyntax.ScopeTraversalExpr:
		// For traversals like var.foo or local.bar
		return fmt.Sprintf("${%s}", e.AsTraversal().RootName()), true
	}
	return "", false
}

// lintBlock checks a single HCL block for naming violations
func (l *Linter) lintBlock(file string, block *hclsyntax.Block) {
	// Check resources
	if block.Type == "resource" {
		if len(block.Labels) < 2 {
			return
		}

		resourceType := block.Labels[0]
		resourceName := block.Labels[1]

		// Skip predefined profiles — they have fixed API names that can't be prefixed
		if typeAttr, ok := block.Body.Attributes["type"]; ok {
			if typeVal, ok := extractStringValue(typeAttr.Expr); ok && typeVal == "predefined" {
				return
			}
		}

		// Check each attribute in the resource
		for _, attr := range block.Body.Attributes {
			if shouldCheckField(resourceType, attr.Name) {
				if value, ok := extractStringValue(attr.Expr); ok {
					if !hasPrefix(value) {
						l.AddError(LintError{
							File:     file,
							Line:     attr.NameRange.Start.Line,
							Column:   attr.NameRange.Start.Column,
							Resource: fmt.Sprintf("%s.%s", resourceType, resourceName),
							Field:    attr.Name,
							Value:    value,
							Message:  fmt.Sprintf("Resource field %q must contain %q prefix", attr.Name, prefix),
						})
					}
				}
			}
		}
	}

	// Check locals
	if block.Type == "locals" {
		for _, attr := range block.Body.Attributes {
			if shouldCheckLocal(attr.Name) {
				if value, ok := extractStringValue(attr.Expr); ok {
					if !hasPrefix(value) {
						l.AddError(LintError{
							File:     file,
							Line:     attr.NameRange.Start.Line,
							Column:   attr.NameRange.Start.Column,
							Resource: "locals",
							Field:    attr.Name,
							Value:    value,
							Message:  fmt.Sprintf("Local value %q should contain %q prefix", attr.Name, prefix),
						})
					}
				}
			}
		}
	}

	// Recursively check nested blocks
	for _, nested := range block.Body.Blocks {
		l.lintBlock(file, nested)
	}
}

// lintFile checks a single Terraform file
func (l *Linter) lintFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	file, diags := l.parser.ParseHCL(content, path)
	if diags.HasErrors() {
		return fmt.Errorf("failed to parse HCL: %s", diags.Error())
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return fmt.Errorf("unexpected body type")
	}

	// Check all top-level blocks
	for _, block := range body.Blocks {
		l.lintBlock(path, block)
	}

	return nil
}

// walkTestdata walks the testdata directory and lints all .tf files
func (l *Linter) walkTestdata() error {
	return filepath.Walk(testdataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".tf" {
			if err := l.lintFile(path); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to lint %s: %v\n", path, err)
			}
		}

		return nil
	})
}

func main() {
	linter := NewLinter()

	if err := linter.walkTestdata(); err != nil {
		fmt.Fprintf(os.Stderr, "Error walking testdata: %v\n", err)
		os.Exit(1)
	}

	if linter.HasErrors() {
		fmt.Println("❌ Linting failed with the following errors:")
		fmt.Println()
		linter.PrintErrors()
		fmt.Printf("\n❌ Found %d naming violations\n", len(linter.errors))
		os.Exit(1)
	}

	fmt.Println("✅ All testdata files have correct naming conventions")
	os.Exit(0)
}
