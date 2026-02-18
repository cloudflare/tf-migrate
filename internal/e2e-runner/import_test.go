package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseImportAnnotations(t *testing.T) {
	tests := []struct {
		name           string
		fileContent    string
		moduleName     string
		expectedSpecs  int
		expectedFirst  *ImportSpec
	}{
		{
			name: "single import annotation",
			fileContent: `# Some comment
# tf-migrate:import-address=${var.cloudflare_account_id}
resource "cloudflare_access_organization" "test" {
  account_id = var.cloudflare_account_id
}
`,
			moduleName:    "zero_trust_organization",
			expectedSpecs: 1,
			expectedFirst: &ImportSpec{
				ResourceType:    "cloudflare_access_organization",
				ResourceName:    "test",
				ResourceAddress: "cloudflare_access_organization.test",
				ImportAddress:   "${var.cloudflare_account_id}",
				ModuleName:      "zero_trust_organization",
			},
		},
		{
			name: "multiple import annotations",
			fileContent: `# tf-migrate:import-address=zones/${var.cloudflare_zone_id}/settings/waf
resource "cloudflare_waf_package" "test" {
  zone_id = var.cloudflare_zone_id
}

# tf-migrate:import-address=${var.cloudflare_account_id}
resource "cloudflare_access_organization" "test" {
  account_id = var.cloudflare_account_id
}
`,
			moduleName:    "test_module",
			expectedSpecs: 2,
			expectedFirst: &ImportSpec{
				ResourceType:    "cloudflare_waf_package",
				ResourceName:    "test",
				ResourceAddress: "cloudflare_waf_package.test",
				ImportAddress:   "zones/${var.cloudflare_zone_id}/settings/waf",
				ModuleName:      "test_module",
			},
		},
		{
			name: "no import annotations",
			fileContent: `# Regular comment
resource "cloudflare_record" "test" {
  zone_id = var.cloudflare_zone_id
  name    = "test"
}
`,
			moduleName:    "dns_record",
			expectedSpecs: 0,
		},
		{
			name: "annotation with spaces",
			fileContent: `  #  tf-migrate:import-address=${var.cloudflare_account_id}
resource "cloudflare_access_organization" "test" {
  account_id = var.cloudflare_account_id
}
`,
			moduleName:    "zero_trust_organization",
			expectedSpecs: 1,
			expectedFirst: &ImportSpec{
				ResourceType:    "cloudflare_access_organization",
				ResourceName:    "test",
				ResourceAddress: "cloudflare_access_organization.test",
				ImportAddress:   "${var.cloudflare_account_id}",
				ModuleName:      "zero_trust_organization",
			},
		},
		{
			name: "annotation not followed by resource",
			fileContent: `# tf-migrate:import-address=${var.cloudflare_account_id}
# Some other comment
variable "test" {
  type = string
}
`,
			moduleName:    "test_module",
			expectedSpecs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			if err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Parse annotations
			specs, err := parseImportAnnotations(tmpFile, tt.moduleName)
			if err != nil {
				t.Fatalf("parseImportAnnotations() error = %v", err)
			}

			// Check number of specs
			if len(specs) != tt.expectedSpecs {
				t.Errorf("parseImportAnnotations() got %d specs, want %d", len(specs), tt.expectedSpecs)
			}

			// Check first spec if expected
			if tt.expectedFirst != nil && len(specs) > 0 {
				got := specs[0]
				want := tt.expectedFirst

				if got.ResourceType != want.ResourceType {
					t.Errorf("ResourceType = %q, want %q", got.ResourceType, want.ResourceType)
				}
				if got.ResourceName != want.ResourceName {
					t.Errorf("ResourceName = %q, want %q", got.ResourceName, want.ResourceName)
				}
				if got.ResourceAddress != want.ResourceAddress {
					t.Errorf("ResourceAddress = %q, want %q", got.ResourceAddress, want.ResourceAddress)
				}
				if got.ImportAddress != want.ImportAddress {
					t.Errorf("ImportAddress = %q, want %q", got.ImportAddress, want.ImportAddress)
				}
				if got.ModuleName != want.ModuleName {
					t.Errorf("ModuleName = %q, want %q", got.ModuleName, want.ModuleName)
				}
			}
		})
	}
}

func TestFindImportSpecs(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()

	// Create module directories
	module1Dir := filepath.Join(tmpDir, "module1")
	module2Dir := filepath.Join(tmpDir, "module2")
	if err := os.MkdirAll(module1Dir, 0755); err != nil {
		t.Fatalf("failed to create module1 dir: %v", err)
	}
	if err := os.MkdirAll(module2Dir, 0755); err != nil {
		t.Fatalf("failed to create module2 dir: %v", err)
	}

	// Create test files
	module1Content := `# tf-migrate:import-address=${var.cloudflare_account_id}
resource "cloudflare_access_organization" "test" {
  account_id = var.cloudflare_account_id
}
`
	if err := os.WriteFile(filepath.Join(module1Dir, "main.tf"), []byte(module1Content), 0644); err != nil {
		t.Fatalf("failed to write module1 file: %v", err)
	}

	module2Content := `# Regular resource, no import needed
resource "cloudflare_record" "test" {
  zone_id = var.cloudflare_zone_id
  name    = "test"
}
`
	if err := os.WriteFile(filepath.Join(module2Dir, "main.tf"), []byte(module2Content), 0644); err != nil {
		t.Fatalf("failed to write module2 file: %v", err)
	}

	// Create a root-level file (should be ignored)
	rootContent := `# tf-migrate:import-address=should_be_ignored
resource "cloudflare_something" "root" {
  id = "test"
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "provider.tf"), []byte(rootContent), 0644); err != nil {
		t.Fatalf("failed to write root file: %v", err)
	}

	// Find import specs (pass nil to scan all modules)
	specs, err := findImportSpecs(tmpDir, nil)
	if err != nil {
		t.Fatalf("findImportSpecs() error = %v", err)
	}

	// Should find only the one from module1
	if len(specs) != 1 {
		t.Errorf("findImportSpecs() got %d specs, want 1", len(specs))
	}

	if len(specs) > 0 {
		got := specs[0]
		if got.ModuleName != "module1" {
			t.Errorf("ModuleName = %q, want %q", got.ModuleName, "module1")
		}
		if got.ResourceType != "cloudflare_access_organization" {
			t.Errorf("ResourceType = %q, want %q", got.ResourceType, "cloudflare_access_organization")
		}
	}
}

func TestFindImportSpecsWithFilter(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()

	// Create module directories
	module1Dir := filepath.Join(tmpDir, "module1")
	module2Dir := filepath.Join(tmpDir, "module2")
	module3Dir := filepath.Join(tmpDir, "module3")
	if err := os.MkdirAll(module1Dir, 0755); err != nil {
		t.Fatalf("failed to create module1 dir: %v", err)
	}
	if err := os.MkdirAll(module2Dir, 0755); err != nil {
		t.Fatalf("failed to create module2 dir: %v", err)
	}
	if err := os.MkdirAll(module3Dir, 0755); err != nil {
		t.Fatalf("failed to create module3 dir: %v", err)
	}

	// Create test files with import annotations
	module1Content := `# tf-migrate:import-address=${var.cloudflare_account_id}
resource "cloudflare_access_organization" "test" {
  account_id = var.cloudflare_account_id
}
`
	if err := os.WriteFile(filepath.Join(module1Dir, "main.tf"), []byte(module1Content), 0644); err != nil {
		t.Fatalf("failed to write module1 file: %v", err)
	}

	module2Content := `# tf-migrate:import-address=${var.cloudflare_zone_id}
resource "cloudflare_waf_package" "test" {
  zone_id = var.cloudflare_zone_id
}
`
	if err := os.WriteFile(filepath.Join(module2Dir, "main.tf"), []byte(module2Content), 0644); err != nil {
		t.Fatalf("failed to write module2 file: %v", err)
	}

	module3Content := `# Regular resource, no import needed
resource "cloudflare_record" "test" {
  zone_id = var.cloudflare_zone_id
  name    = "test"
}
`
	if err := os.WriteFile(filepath.Join(module3Dir, "main.tf"), []byte(module3Content), 0644); err != nil {
		t.Fatalf("failed to write module3 file: %v", err)
	}

	t.Run("filter to single module with import", func(t *testing.T) {
		// Find import specs filtering to only module1
		specs, err := findImportSpecs(tmpDir, []string{"module1"})
		if err != nil {
			t.Fatalf("findImportSpecs() error = %v", err)
		}

		// Should find only module1's import
		if len(specs) != 1 {
			t.Errorf("findImportSpecs() got %d specs, want 1", len(specs))
		}

		if len(specs) > 0 {
			got := specs[0]
			if got.ModuleName != "module1" {
				t.Errorf("ModuleName = %q, want %q", got.ModuleName, "module1")
			}
			if got.ResourceType != "cloudflare_access_organization" {
				t.Errorf("ResourceType = %q, want %q", got.ResourceType, "cloudflare_access_organization")
			}
		}
	})

	t.Run("filter to module without import", func(t *testing.T) {
		// Find import specs filtering to only module3 (no imports)
		specs, err := findImportSpecs(tmpDir, []string{"module3"})
		if err != nil {
			t.Fatalf("findImportSpecs() error = %v", err)
		}

		// Should find no imports
		if len(specs) != 0 {
			t.Errorf("findImportSpecs() got %d specs, want 0", len(specs))
		}
	})

	t.Run("filter to multiple modules", func(t *testing.T) {
		// Find import specs filtering to module1 and module2
		specs, err := findImportSpecs(tmpDir, []string{"module1", "module2"})
		if err != nil {
			t.Fatalf("findImportSpecs() error = %v", err)
		}

		// Should find both imports
		if len(specs) != 2 {
			t.Errorf("findImportSpecs() got %d specs, want 2", len(specs))
		}

		// Check that we got imports from both modules
		foundModule1 := false
		foundModule2 := false
		for _, spec := range specs {
			if spec.ModuleName == "module1" {
				foundModule1 = true
			}
			if spec.ModuleName == "module2" {
				foundModule2 = true
			}
		}

		if !foundModule1 {
			t.Error("Expected to find import from module1")
		}
		if !foundModule2 {
			t.Error("Expected to find import from module2")
		}
	})

	t.Run("filter to nonexistent module", func(t *testing.T) {
		// Find import specs filtering to module that doesn't exist
		specs, err := findImportSpecs(tmpDir, []string{"nonexistent"})
		if err != nil {
			t.Fatalf("findImportSpecs() error = %v", err)
		}

		// Should find no imports (no error, just empty result)
		if len(specs) != 0 {
			t.Errorf("findImportSpecs() got %d specs, want 0", len(specs))
		}
	})
}

func TestConvertToTerraformVar(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "account ID variable",
			address:  "${var.cloudflare_account_id}",
			expected: "var.cloudflare_account_id",
		},
		{
			name:     "zone ID in path",
			address:  "zones/${var.cloudflare_zone_id}/settings/waf",
			expected: "zones/var.cloudflare_zone_id/settings/waf",
		},
		{
			name:     "domain variable",
			address:  "${var.cloudflare_domain}/path",
			expected: "var.cloudflare_domain/path",
		},
		{
			name:     "multiple variables",
			address:  "account/${var.cloudflare_account_id}/zone/${var.cloudflare_zone_id}",
			expected: "account/var.cloudflare_account_id/zone/var.cloudflare_zone_id",
		},
		{
			name:     "no variables",
			address:  "static/path/123",
			expected: "static/path/123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToTerraformVar(tt.address)
			if got != tt.expected {
				t.Errorf("convertToTerraformVar() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGenerateImportBlocks(t *testing.T) {
	tests := []struct {
		name     string
		specs    []ImportSpec
		expected []string // Strings that should be present in output
	}{
		{
			name:     "no imports",
			specs:    []ImportSpec{},
			expected: []string{},
		},
		{
			name: "single import with account ID variable",
			specs: []ImportSpec{
				{
					ResourceType:    "cloudflare_access_organization",
					ResourceName:    "test",
					ResourceAddress: "cloudflare_access_organization.test",
					ImportAddress:   "${var.cloudflare_account_id}",
					ModuleName:      "zero_trust_organization",
				},
			},
			expected: []string{
				"import {",
				"to = module.zero_trust_organization.cloudflare_access_organization.test",
				"id = var.cloudflare_account_id", // Variable reference, not quoted
			},
		},
		{
			name: "import with literal string ID",
			specs: []ImportSpec{
				{
					ResourceType:    "cloudflare_something",
					ResourceName:    "test",
					ResourceAddress: "cloudflare_something.test",
					ImportAddress:   "static-id-123",
					ModuleName:      "something",
				},
			},
			expected: []string{
				"import {",
				"to = module.something.cloudflare_something.test",
				`id = "static-id-123"`, // Literal string, quoted
			},
		},
		{
			name: "multiple imports with mixed types",
			specs: []ImportSpec{
				{
					ResourceType:    "cloudflare_access_organization",
					ResourceName:    "test",
					ResourceAddress: "cloudflare_access_organization.test",
					ImportAddress:   "${var.cloudflare_account_id}",
					ModuleName:      "zero_trust_organization",
				},
				{
					ResourceType:    "cloudflare_waf_package",
					ResourceName:    "test",
					ResourceAddress: "cloudflare_waf_package.test",
					ImportAddress:   "zones/${var.cloudflare_zone_id}/settings/waf",
					ModuleName:      "waf",
				},
			},
			expected: []string{
				"import {",
				"to = module.zero_trust_organization.cloudflare_access_organization.test",
				"id = var.cloudflare_account_id",
				"to = module.waf.cloudflare_waf_package.test",
				`id = "zones/${var.cloudflare_zone_id}/settings/waf"`, // Mixed literal and variable — needs string interpolation
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateImportBlocks(tt.specs)

			if len(tt.expected) == 0 {
				if got != "" {
					t.Errorf("generateImportBlocks() expected empty string, got %q", got)
				}
				return
			}

			for _, expected := range tt.expected {
				if !strings.Contains(got, expected) {
					t.Errorf("generateImportBlocks() missing expected string %q\nGot:\n%s", expected, got)
				}
			}
		})
	}
}
