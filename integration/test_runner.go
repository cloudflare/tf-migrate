package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// TestCase represents a single integration test case
type TestCase struct {
	Resource string // Resource directory name in testdata
}

// TestRunner manages integration test execution for any version migration
type TestRunner struct {
	BaseDir       string
	TfMigrateDir  string
	SourceVersion string
	TargetVersion string
	TestDataPath  string
}

// NewTestRunner creates a new test runner for any version migration
func NewTestRunner(sourceVersion, targetVersion string) (*TestRunner, error) {
	baseDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	// Determine the tf-migrate directory based on current location
	// We could be in integration/v4_to_v5, integration/v5_to_v6, etc.
	var tfMigrateDir string
	if strings.Contains(baseDir, "/integration/") {
		// Find the tf-migrate root by going up from integration
		parts := strings.Split(baseDir, "/integration/")
		tfMigrateDir = parts[0]
	} else {
		// Fallback: assume we're somewhere under tf-migrate
		tfMigrateDir = filepath.Dir(filepath.Dir(baseDir))
	}

	return &TestRunner{
		BaseDir:       baseDir,
		TfMigrateDir:  tfMigrateDir,
		SourceVersion: sourceVersion,
		TargetVersion: targetVersion,
	}, nil
}

// RunTest executes a single integration test
func (r *TestRunner) RunTest(t *testing.T, test TestCase) {
	t.Run(test.Resource, func(t *testing.T) {
		// Create temp directory
		tempDir := t.TempDir()

		// Copy input files
		inputDir := filepath.Join(r.BaseDir, "testdata", test.Resource, "input")
		if err := r.copyDirectory(inputDir, tempDir); err != nil {
			t.Fatalf("Failed to copy input files: %v", err)
		}

		// Run tf-migrate
		if err := r.runMigration(tempDir); err != nil {
			t.Fatalf("Migration failed: %v", err)
		}

		// Compare outputs
		expectedDir := filepath.Join(r.BaseDir, "testdata", test.Resource, "expected")
		if err := r.compareDirectories(expectedDir, tempDir); err != nil {
			t.Errorf("Output comparison failed: %v", err)
		}
	})
}

// copyDirectory copies all files from src to dst
func (r *TestRunner) copyDirectory(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("reading source directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			return fmt.Errorf("subdirectories are not yet supported: %s", entry.Name())
		}

		srcFile := filepath.Join(src, entry.Name())
		dstFile := filepath.Join(dst, entry.Name())

		if err := copyFile(srcFile, dstFile); err != nil {
			return fmt.Errorf("copying %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// runMigration executes tf-migrate on the given directory
func (r *TestRunner) runMigration(dir string) error {
	// Build the binary first to avoid compilation issues
	buildCmd := exec.Command("go", "build", "-o", filepath.Join(r.TfMigrateDir, "tf-migrate"), "./cmd/tf-migrate")
	buildCmd.Dir = r.TfMigrateDir
	if output, err := buildCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("building tf-migrate: %w\nOutput: %s", err, output)
	}

	// Run migration
	migrateCmd := exec.Command(
		filepath.Join(r.TfMigrateDir, "tf-migrate"),
		"migrate",
		"--config-dir", dir,
		"--state-file", filepath.Join(dir, "terraform.tfstate"),
		"--source-version", r.SourceVersion,
		"--target-version", r.TargetVersion,
		"--backup=false",
	)
	// Set GODEBUG to make map iteration deterministic for consistent test output
	migrateCmd.Env = append(os.Environ(), "GODEBUG=randommapseed=0")

	output, err := migrateCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("running migration: %w\nOutput: %s", err, output)
	}

	return nil
}

// compareDirectories compares all files in expected vs actual directories
func (r *TestRunner) compareDirectories(expectedDir, actualDir string) error {
	entries, err := os.ReadDir(expectedDir)
	if err != nil {
		return fmt.Errorf("reading expected directory: %w", err)
	}

	var errors []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		expectedFile := filepath.Join(expectedDir, entry.Name())
		actualFile := filepath.Join(actualDir, entry.Name())

		// Handle different file types
		switch filepath.Ext(entry.Name()) {
		case ".tfstate":
			if err := r.compareJSONFiles(expectedFile, actualFile); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", entry.Name(), err))
			}
		case ".tf":
			if err := r.compareTextFiles(expectedFile, actualFile); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", entry.Name(), err))
			}
		default:
			return fmt.Errorf("unsupported file type: %s", entry.Name())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("comparison failed:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// compareJSONFiles compares two JSON files
func (r *TestRunner) compareJSONFiles(expectedFile, actualFile string) error {
	expected, err := os.ReadFile(expectedFile)
	if err != nil {
		return fmt.Errorf("reading expected file: %w", err)
	}

	actual, err := os.ReadFile(actualFile)
	if err != nil {
		return fmt.Errorf("reading actual file: %w", err)
	}

	// Parse and normalize JSON
	var expectedData, actualData interface{}
	if err := json.Unmarshal(expected, &expectedData); err != nil {
		return fmt.Errorf("parsing expected JSON: %w", err)
	}
	if err := json.Unmarshal(actual, &actualData); err != nil {
		return fmt.Errorf("parsing actual JSON: %w", err)
	}

	// Re-marshal with indentation for comparison
	expectedNorm, err := json.MarshalIndent(expectedData, "", "  ")
	if err != nil {
		return err
	}
	actualNorm, err := json.MarshalIndent(actualData, "", "  ")
	if err != nil {
		return err
	}

	if !bytes.Equal(expectedNorm, actualNorm) {
		// Generate diff for better error messages
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(string(expectedNorm), string(actualNorm), false)
		return fmt.Errorf("JSON mismatch:\n%s", dmp.DiffPrettyText(diffs))
	}

	return nil
}

// compareTextFiles compares two text files
func (r *TestRunner) compareTextFiles(expectedFile, actualFile string) error {
	expected, err := os.ReadFile(expectedFile)
	if err != nil {
		return fmt.Errorf("reading expected file: %w", err)
	}

	actual, err := os.ReadFile(actualFile)
	if err != nil {
		return fmt.Errorf("reading actual file: %w", err)
	}

	// For .tf files, normalize using HCL formatting to handle map ordering differences
	if filepath.Ext(expectedFile) == ".tf" {
		expected = normalizeHCL(expected)
		actual = normalizeHCL(actual)
	}

	// Normalize line endings
	expectedStr := normalizeLineEndings(string(expected))
	actualStr := normalizeLineEndings(string(actual))

	if expectedStr != actualStr {
		// Generate diff for better error messages
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(expectedStr, actualStr, false)
		return fmt.Errorf("content mismatch:\n%s", dmp.DiffPrettyText(diffs))
	}

	return nil
}

// normalizeHCL normalizes HCL content using hclwrite to handle map ordering
func normalizeHCL(content []byte) []byte {
	// Use hclwrite.Format to normalize the HCL, which will produce consistent formatting
	// but may not fix map ordering. For now, just use the formatting to normalize whitespace.
	formatted := hclwrite.Format(content)
	return formatted
}

// normalizeLineEndings converts all line endings to \n
func normalizeLineEndings(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.TrimSpace(s)
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
