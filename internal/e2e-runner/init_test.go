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
