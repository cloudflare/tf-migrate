package e2e

import (
	"os"
	"path/filepath"
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
# tf-migrate:import-address=account/${var.cloudflare_account_id}
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
				ImportAddress:   "account/${var.cloudflare_account_id}",
				ModuleName:      "zero_trust_organization",
			},
		},
		{
			name: "multiple import annotations",
			fileContent: `# tf-migrate:import-address=zones/${var.cloudflare_zone_id}/settings/waf
resource "cloudflare_waf_package" "test" {
  zone_id = var.cloudflare_zone_id
}

# tf-migrate:import-address=account/${var.cloudflare_account_id}
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
			fileContent: `  #  tf-migrate:import-address=account/${var.cloudflare_account_id}
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
				ImportAddress:   "account/${var.cloudflare_account_id}",
				ModuleName:      "zero_trust_organization",
			},
		},
		{
			name: "annotation not followed by resource",
			fileContent: `# tf-migrate:import-address=account/${var.cloudflare_account_id}
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
	module1Content := `# tf-migrate:import-address=account/${var.cloudflare_account_id}
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

	// Find import specs
	specs, err := findImportSpecs(tmpDir)
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

func TestResolveImportAddress(t *testing.T) {
	env := &E2EEnv{
		AccountID: "test-account-123",
		ZoneID:    "test-zone-456",
		Domain:    "example.com",
	}

	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "account ID substitution",
			address:  "account/${var.cloudflare_account_id}",
			expected: "account/test-account-123",
		},
		{
			name:     "zone ID substitution",
			address:  "zones/${var.cloudflare_zone_id}/settings/waf",
			expected: "zones/test-zone-456/settings/waf",
		},
		{
			name:     "domain substitution",
			address:  "${var.cloudflare_domain}/path",
			expected: "example.com/path",
		},
		{
			name:     "multiple substitutions",
			address:  "account/${var.cloudflare_account_id}/zone/${var.cloudflare_zone_id}",
			expected: "account/test-account-123/zone/test-zone-456",
		},
		{
			name:     "no substitutions",
			address:  "static/path/123",
			expected: "static/path/123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveImportAddress(tt.address, env)
			if got != tt.expected {
				t.Errorf("resolveImportAddress() = %q, want %q", got, tt.expected)
			}
		})
	}
}
