package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindInputDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test directory structure
	testStructure := []string{
		"resource1/input",
		"resource2/input",
		"resource3/expected",
		"resource4/input/nested", // nested input should not be found as separate
		"no_input/other",
	}

	for _, path := range testStructure {
		fullPath := filepath.Join(tmpDir, path)
		err := os.MkdirAll(fullPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}
	}

	inputDirs, err := findInputDirs(tmpDir)
	if err != nil {
		t.Fatalf("findInputDirs() error = %v", err)
	}

	// Should find 2 input directories (resource1/input and resource2/input)
	// resource4/input/nested's parent is also "input", so it finds that too
	expectedCount := 3
	if len(inputDirs) != expectedCount {
		t.Errorf("Found %d input dirs, want %d: %v", len(inputDirs), expectedCount, inputDirs)
	}

	// Verify paths contain "input"
	for _, dir := range inputDirs {
		if !strings.HasSuffix(dir, "input") {
			t.Errorf("Directory doesn't end with 'input': %v", dir)
		}
	}
}

func TestFindInputDirs_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	inputDirs, err := findInputDirs(tmpDir)
	if err != nil {
		t.Fatalf("findInputDirs() error = %v", err)
	}

	if len(inputDirs) != 0 {
		t.Errorf("Expected 0 input dirs in empty directory, got %d", len(inputDirs))
	}
}

func TestFindInputDirs_NonexistentDir(t *testing.T) {
	nonexistent := "/path/that/does/not/exist"

	_, err := findInputDirs(nonexistent)
	if err == nil {
		t.Error("Expected error for nonexistent directory")
	}
}

func TestRunInit_ResourceFiltering(t *testing.T) {
	// This is more of an integration test
	// We can test the logic of resource filtering separately

	resources := "zone_dnssec,argo"
	resourceList := strings.Split(resources, ",")

	expectedResources := []string{"zone_dnssec", "argo"}
	for i, r := range resourceList {
		trimmed := strings.TrimSpace(r)
		if trimmed != expectedResources[i] {
			t.Errorf("Resource %d: got %q, want %q", i, trimmed, expectedResources[i])
		}
	}
}

func TestShouldSkipE2EResource(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		expectedSkip bool
	}{
		{
			name: "skip marker present in first line",
			content: `# E2E-SKIP: Cannot be created via Terraform
# More details here
resource "cloudflare_example" "test" {}`,
			expectedSkip: true,
		},
		{
			name: "skip marker present within first 20 lines",
			content: `# Terraform v4 to v5 Migration
#
# E2E-SKIP: Lifecycle constraints
#
# Additional documentation
resource "cloudflare_example" "test" {}`,
			expectedSkip: true,
		},
		{
			name:         "skip marker at line 20",
			content:      strings.Repeat("# Comment\n", 19) + "# E2E-SKIP: Test\nresource \"test\" {}",
			expectedSkip: true,
		},
		{
			name:         "skip marker after line 20",
			content:      strings.Repeat("# Comment\n", 20) + "# E2E-SKIP: Test\nresource \"test\" {}",
			expectedSkip: false,
		},
		{
			name: "no skip marker",
			content: `# Regular comment
# More comments
resource "cloudflare_example" "test" {}`,
			expectedSkip: false,
		},
		{
			name: "skip marker not in comment",
			content: `variable "name" {
  default = "E2E-SKIP: not a skip marker"
}
resource "cloudflare_example" "test" {}`,
			expectedSkip: false,
		},
		{
			name: "skip marker after resource block starts",
			content: `resource "cloudflare_example" "test" {
  # E2E-SKIP: too late
  name = "test"
}`,
			expectedSkip: false,
		},
		{
			name:         "empty file",
			content:      "",
			expectedSkip: false,
		},
		{
			name: "only comments with skip marker",
			content: `# E2E-SKIP: Test resource
# This resource cannot be tested
# in E2E environments`,
			expectedSkip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpFile, err := os.CreateTemp("", "e2e_test_*.tf")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write content
			if _, err := tmpFile.WriteString(tt.content); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			// Test
			skip := shouldSkipE2EResource(tmpFile.Name())
			if skip != tt.expectedSkip {
				t.Errorf("shouldSkipE2EResource() = %v, want %v", skip, tt.expectedSkip)
			}
		})
	}
}

func TestShouldSkipE2EResource_FileNotFound(t *testing.T) {
	skip := shouldSkipE2EResource("/nonexistent/file.tf")
	if skip {
		t.Error("shouldSkipE2EResource() should return false for nonexistent file")
	}
}
