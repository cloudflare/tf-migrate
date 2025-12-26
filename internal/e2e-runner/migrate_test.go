package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractModuleBlock(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		moduleName string
		want       string
	}{
		{
			name: "simple module",
			content: `
module "zone_dnssec" {
  source = "./zone_dnssec"

  cloudflare_account_id = var.cloudflare_account_id
  cloudflare_zone_id    = var.cloudflare_zone_id
}

module "argo" {
  source = "./argo"
}
`,
			moduleName: "zone_dnssec",
			want: `module "zone_dnssec" {
  source = "./zone_dnssec"

  cloudflare_account_id = var.cloudflare_account_id
  cloudflare_zone_id    = var.cloudflare_zone_id
}`,
		},
		{
			name: "module with nested braces",
			content: `
module "complex" {
  source = "./complex"

  config = {
    key = "value"
  }
}
`,
			moduleName: "complex",
			want: `module "complex" {
  source = "./complex"

  config = {
    key = "value"
  }
}`,
		},
		{
			name:       "nonexistent module",
			content:    `module "exists" { source = "./exists" }`,
			moduleName: "nonexistent",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractModuleBlock(tt.content, tt.moduleName)
			gotTrimmed := strings.TrimSpace(got)
			wantTrimmed := strings.TrimSpace(tt.want)

			if gotTrimmed != wantTrimmed {
				t.Errorf("extractModuleBlock() =\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}


func TestFilterStateFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test state file
	state := map[string]interface{}{
		"version": 4,
		"serial":  1,
		"resources": []interface{}{
			map[string]interface{}{
				"module": "module.zone_dnssec",
				"type":   "cloudflare_zone_dnssec",
				"name":   "example",
				"instances": []interface{}{
					map[string]interface{}{"attributes": map[string]interface{}{}},
				},
			},
			map[string]interface{}{
				"module": "module.argo",
				"type":   "cloudflare_argo",
				"name":   "example",
				"instances": []interface{}{
					map[string]interface{}{"attributes": map[string]interface{}{}},
				},
			},
			map[string]interface{}{
				"module": "module.other",
				"type":   "cloudflare_other",
				"name":   "example",
				"instances": []interface{}{
					map[string]interface{}{"attributes": map[string]interface{}{}},
				},
			},
		},
	}

	stateJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal state: %v", err)
	}

	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	err = os.WriteFile(stateFile, stateJSON, 0644)
	if err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	// Filter to only zone_dnssec and argo
	resources := []string{"zone_dnssec", "argo"}
	err = filterStateFile(tmpDir, resources)
	if err != nil {
		t.Fatalf("filterStateFile() error = %v", err)
	}

	// Read filtered state
	filteredData, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("Failed to read filtered state: %v", err)
	}

	var filteredState map[string]interface{}
	err = json.Unmarshal(filteredData, &filteredState)
	if err != nil {
		t.Fatalf("Failed to unmarshal filtered state: %v", err)
	}

	// Verify only 2 resources remain
	filteredResources := filteredState["resources"].([]interface{})
	if len(filteredResources) != 2 {
		t.Errorf("Expected 2 resources after filtering, got %d", len(filteredResources))
	}

	// Verify the correct resources remain
	for _, res := range filteredResources {
		resMap := res.(map[string]interface{})
		module := resMap["module"].(string)
		if module != "module.zone_dnssec" && module != "module.argo" {
			t.Errorf("Unexpected module in filtered state: %s", module)
		}
	}
}

func TestFilterStateFile_NoStateFile(t *testing.T) {
	tmpDir := t.TempDir()

	// filterStateFile should handle missing state file gracefully
	err := filterStateFile(tmpDir, []string{"test"})
	if err != nil {
		t.Errorf("filterStateFile() should not error on missing state file: %v", err)
	}
}

func TestUpdateProviderTF(t *testing.T) {
	tmpDir := t.TempDir()

	providerContent := `terraform {
  required_version = ">= 1.0"

  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }

  # Remote state backend (R2)
  # Configuration provided via backend.hcl or -backend-config flags
  # See e2e/scripts/init for backend initialization
  backend "s3" {}
}

provider "cloudflare" {}
`

	providerFile := filepath.Join(tmpDir, "provider.tf")
	err := os.WriteFile(providerFile, []byte(providerContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create provider.tf: %v", err)
	}

	err = updateProviderTF(tmpDir)
	if err != nil {
		t.Fatalf("updateProviderTF() error = %v", err)
	}

	// Read updated file
	updatedContent, err := os.ReadFile(providerFile)
	if err != nil {
		t.Fatalf("Failed to read updated provider.tf: %v", err)
	}

	updatedStr := string(updatedContent)

	// Verify version was updated
	if !strings.Contains(updatedStr, "~> 5.0") {
		t.Error("Version was not updated to ~> 5.0")
	}

	// Verify backend config was removed
	if strings.Contains(updatedStr, "backend \"s3\"") {
		t.Error("Backend configuration was not removed")
	}

	// Verify provider block still exists
	if !strings.Contains(updatedStr, "provider \"cloudflare\"") {
		t.Error("Provider block was removed")
	}
}

func TestUpdateProviderTF_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Should not error if file doesn't exist
	err := updateProviderTF(tmpDir)
	if err != nil {
		t.Errorf("updateProviderTF() should not error when file doesn't exist: %v", err)
	}
}

func TestCreateFilteredMainTF(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source main.tf
	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(srcDir, 0755)

	mainTFContent := `
module "zone_dnssec" {
  source = "./zone_dnssec"

  cloudflare_account_id = var.cloudflare_account_id
}

module "argo" {
  source = "./argo"

  cloudflare_zone_id = var.cloudflare_zone_id
}

module "other" {
  source = "./other"
}
`

	srcMainTF := filepath.Join(srcDir, "main.tf")
	err := os.WriteFile(srcMainTF, []byte(mainTFContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create source main.tf: %v", err)
	}

	// Create filtered main.tf
	dstDir := filepath.Join(tmpDir, "dst")
	os.MkdirAll(dstDir, 0755)

	resources := []string{"zone_dnssec", "argo"}
	err = createFilteredMainTF(srcDir, dstDir, resources)
	if err != nil {
		t.Fatalf("createFilteredMainTF() error = %v", err)
	}

	// Read filtered main.tf
	dstMainTF := filepath.Join(dstDir, "main.tf")
	filteredContent, err := os.ReadFile(dstMainTF)
	if err != nil {
		t.Fatalf("Failed to read filtered main.tf: %v", err)
	}

	filteredStr := string(filteredContent)

	// Verify targeted modules are present
	if !strings.Contains(filteredStr, `module "zone_dnssec"`) {
		t.Error("zone_dnssec module not in filtered main.tf")
	}
	if !strings.Contains(filteredStr, `module "argo"`) {
		t.Error("argo module not in filtered main.tf")
	}

	// Verify excluded module is not present
	if strings.Contains(filteredStr, `module "other"`) {
		t.Error("other module should not be in filtered main.tf")
	}
}

func TestBuildBinary_Logic(t *testing.T) {
	// This test just verifies the buildBinary function exists and can be called
	// We can't actually test it without a proper Go environment
	// But we can test that the function signature is correct

	// The function should exist and be callable
	// We're not actually calling it here as it would try to build
	_ = buildBinary
}

func TestCopyAllResources(t *testing.T) {
	// Create temporary directories
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	// Create source structure
	os.MkdirAll(filepath.Join(srcDir, "module1"), 0755)
	os.MkdirAll(filepath.Join(srcDir, ".terraform"), 0755)
	os.WriteFile(filepath.Join(srcDir, "provider.tf"), []byte("provider content"), 0644)
	os.WriteFile(filepath.Join(srcDir, "variables.tf"), []byte("variables content"), 0644)
	os.WriteFile(filepath.Join(srcDir, "backend.hcl"), []byte("backend config"), 0644)
	os.WriteFile(filepath.Join(srcDir, ".terraform.lock.hcl"), []byte("lock file"), 0644)
	os.WriteFile(filepath.Join(srcDir, "module1", "main.tf"), []byte("module1 content"), 0644)

	// Create destination directory
	os.MkdirAll(dstDir, 0755)

	// Run copyAllResources
	err := copyAllResources(srcDir, dstDir)
	if err != nil {
		t.Fatalf("copyAllResources() failed: %v", err)
	}

	// Verify copied files exist
	tests := []struct {
		path       string
		shouldExist bool
		content     string
	}{
		{filepath.Join(dstDir, "provider.tf"), true, "provider content"},
		{filepath.Join(dstDir, "variables.tf"), true, "variables content"},
		{filepath.Join(dstDir, "module1", "main.tf"), true, "module1 content"},
		{filepath.Join(dstDir, "backend.hcl"), false, ""}, // Should be excluded
		{filepath.Join(dstDir, ".terraform.lock.hcl"), false, ""}, // Should be excluded
		{filepath.Join(dstDir, ".terraform"), false, ""}, // Should be excluded
	}

	for _, tt := range tests {
		_, err := os.Stat(tt.path)
		exists := err == nil

		if exists != tt.shouldExist {
			t.Errorf("File %s: exists=%v, want=%v", tt.path, exists, tt.shouldExist)
		}

		if tt.shouldExist && tt.content != "" {
			content, err := os.ReadFile(tt.path)
			if err != nil {
				t.Errorf("Failed to read %s: %v", tt.path, err)
			} else if string(content) != tt.content {
				t.Errorf("File %s content = %q, want %q", tt.path, content, tt.content)
			}
		}
	}
}

func TestCopyAllResources_NonexistentSource(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "nonexistent")
	dstDir := filepath.Join(tmpDir, "dst")

	err := copyAllResources(srcDir, dstDir)
	if err == nil {
		t.Error("Expected error for nonexistent source directory")
	}
}

func TestCopyTargetedResources(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	// Create source structure
	os.MkdirAll(filepath.Join(srcDir, "module1"), 0755)
	os.MkdirAll(filepath.Join(srcDir, "module2"), 0755)
	os.MkdirAll(filepath.Join(srcDir, "module3"), 0755)
	os.WriteFile(filepath.Join(srcDir, "provider.tf"), []byte("provider \"cloudflare\" {}"), 0644)
	os.WriteFile(filepath.Join(srcDir, "variables.tf"), []byte("variable \"x\" {}"), 0644)
	os.WriteFile(filepath.Join(srcDir, "terraform.tfstate"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(srcDir, "main.tf"), []byte(`
module "module1" {
  source = "./module1"
}

module "module2" {
  source = "./module2"
}

module "module3" {
  source = "./module3"
}
`), 0644)
	os.WriteFile(filepath.Join(srcDir, "module1", "main.tf"), []byte("resource \"cloudflare_zone\" \"test\" {}"), 0644)
	os.WriteFile(filepath.Join(srcDir, "module2", "main.tf"), []byte("resource \"cloudflare_record\" \"test\" {}"), 0644)
	os.WriteFile(filepath.Join(srcDir, "module3", "main.tf"), []byte("resource \"cloudflare_list\" \"test\" {}"), 0644)

	// Create destination directory
	os.MkdirAll(dstDir, 0755)

	// Run copyTargetedResources for module1 and module2 only
	resources := []string{"module1", "module2"}
	err := copyTargetedResources(srcDir, dstDir, resources)
	if err != nil {
		t.Fatalf("copyTargetedResources() failed: %v", err)
	}

	// Verify root files were copied
	rootFiles := []string{"provider.tf", "variables.tf", "terraform.tfstate"}
	for _, file := range rootFiles {
		path := filepath.Join(dstDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Root file %s was not copied", file)
		}
	}

	// Verify targeted modules were copied
	if _, err := os.Stat(filepath.Join(dstDir, "module1", "main.tf")); os.IsNotExist(err) {
		t.Error("module1 was not copied")
	}
	if _, err := os.Stat(filepath.Join(dstDir, "module2", "main.tf")); os.IsNotExist(err) {
		t.Error("module2 was not copied")
	}

	// Verify non-targeted module was NOT copied
	if _, err := os.Stat(filepath.Join(dstDir, "module3")); err == nil {
		t.Error("module3 should not have been copied")
	}

	// Verify filtered main.tf was created
	mainTFPath := filepath.Join(dstDir, "main.tf")
	if _, err := os.Stat(mainTFPath); os.IsNotExist(err) {
		t.Error("main.tf was not created")
	} else {
		content, _ := os.ReadFile(mainTFPath)
		mainTF := string(content)

		// Should contain module1 and module2
		if !strings.Contains(mainTF, "module1") {
			t.Error("main.tf should contain module1")
		}
		if !strings.Contains(mainTF, "module2") {
			t.Error("main.tf should contain module2")
		}
		// Should NOT contain module3
		if strings.Contains(mainTF, "module3") {
			t.Error("main.tf should not contain module3")
		}
	}
}

func TestCopyTargetedResources_MissingModule(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	// Create source structure with only one module
	os.MkdirAll(filepath.Join(srcDir, "module1"), 0755)
	os.WriteFile(filepath.Join(srcDir, "provider.tf"), []byte("provider {}"), 0644)
	os.WriteFile(filepath.Join(srcDir, "main.tf"), []byte(`module "module1" {}`), 0644)
	os.WriteFile(filepath.Join(srcDir, "module1", "main.tf"), []byte("resource {}"), 0644)

	os.MkdirAll(dstDir, 0755)

	// Request module1 (exists) and module2 (doesn't exist)
	resources := []string{"module1", "module2"}
	err := copyTargetedResources(srcDir, dstDir, resources)

	// Should not error - just skip missing modules
	if err != nil {
		t.Errorf("copyTargetedResources() should not error on missing modules: %v", err)
	}

	// Verify module1 was copied
	if _, err := os.Stat(filepath.Join(dstDir, "module1")); os.IsNotExist(err) {
		t.Error("module1 should have been copied")
	}

	// Verify module2 was not copied (doesn't exist)
	if _, err := os.Stat(filepath.Join(dstDir, "module2")); err == nil {
		t.Error("module2 should not exist")
	}
}

func TestCopyTargetedResources_OptionalRootFiles(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	// Create source with only provider.tf (not all root files)
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "provider.tf"), []byte("provider {}"), 0644)
	os.WriteFile(filepath.Join(srcDir, "main.tf"), []byte(""), 0644)
	// No variables.tf, terraform.tfvars, or terraform.tfstate

	os.MkdirAll(dstDir, 0755)

	// Should not error even if optional files are missing
	resources := []string{}
	err := copyTargetedResources(srcDir, dstDir, resources)
	if err != nil {
		t.Errorf("copyTargetedResources() should handle missing optional files: %v", err)
	}

	// Verify provider.tf was copied
	if _, err := os.Stat(filepath.Join(dstDir, "provider.tf")); os.IsNotExist(err) {
		t.Error("provider.tf should have been copied")
	}

	// Verify missing optional files don't cause errors
	if _, err := os.Stat(filepath.Join(dstDir, "variables.tf")); err == nil {
		t.Error("variables.tf should not exist if it wasn't in source")
	}
}
