package testing

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/cloudflare/tf-migrate/internal/processing"
	"github.com/cloudflare/tf-migrate/internal/core"
	"github.com/cloudflare/tf-migrate/internal/resources"
)

// normalizeHCL normalizes HCL formatting for comparison
func normalizeHCL(hcl string) string {
	// Trim whitespace
	hcl = strings.TrimSpace(hcl)
	
	// Normalize line endings
	hcl = strings.ReplaceAll(hcl, "\r\n", "\n")
	
	// Remove trailing whitespace from lines
	lines := strings.Split(hcl, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	hcl = strings.Join(lines, "\n")
	
	// Normalize spacing around braces (handles the closing brace on same line issue)
	// Replace "content = "1.2.3.4"\n}" with "content = "1.2.3.4" }"
	// and "{ content" with "{ content"
	re := regexp.MustCompile(`\s*\n\s*}`)
	hcl = re.ReplaceAllString(hcl, " }")
	
	// Also normalize opening braces
	re = regexp.MustCompile(`{\s+`)
	hcl = re.ReplaceAllString(hcl, "{ ")
	
	// Normalize multiple spaces to single space
	re = regexp.MustCompile(`\s+`)
	hcl = re.ReplaceAllString(hcl, " ")
	
	// Extract and sort attributes for each resource block to ignore ordering
	// This is a simple approach that works for basic test cases
	resourcePattern := regexp.MustCompile(`resource\s+"([^"]+)"\s+"([^"]+)"\s+{([^}]+)}`)
	attrPattern := regexp.MustCompile(`(\w+)\s*=\s*("[^"]*"|\S+)`)
	
	matches := resourcePattern.FindAllStringSubmatch(hcl, -1)
	for _, match := range matches {
		if len(match) == 4 {
			resourceType := match[1]
			resourceName := match[2]
			body := match[3]
			
			// Find all attributes
			attrs := attrPattern.FindAllStringSubmatch(body, -1)
			attrMap := make(map[string]string)
			for _, attr := range attrs {
				if len(attr) == 3 {
					attrMap[attr[1]] = attr[2]
				}
			}
			
			// Sort attribute names
			var attrNames []string
			for name := range attrMap {
				attrNames = append(attrNames, name)
			}
			// Use a simple sort for deterministic output
			for i := 0; i < len(attrNames); i++ {
				for j := i + 1; j < len(attrNames); j++ {
					if attrNames[i] > attrNames[j] {
						attrNames[i], attrNames[j] = attrNames[j], attrNames[i]
					}
				}
			}
			
			// Rebuild the resource block with sorted attributes
			var sortedAttrs []string
			for _, name := range attrNames {
				sortedAttrs = append(sortedAttrs, fmt.Sprintf("%s = %s", name, attrMap[name]))
			}
			
			newBlock := fmt.Sprintf(`resource "%s" "%s" { %s }`, resourceType, resourceName, strings.Join(sortedAttrs, " "))
			hcl = strings.ReplaceAll(hcl, match[0], newBlock)
		}
	}
	
	return hcl
}

// ConfigTestCase defines a configuration transformation test
type ConfigTestCase struct {
	Name         string // Test name
	ResourceType string // Resource type being tested
	Input        string // Input HCL configuration
	Expected     string // Expected output HCL
	ShouldError  bool   // Whether an error is expected
}

// StateTestCase defines a state transformation test
type StateTestCase struct {
	Name         string // Test name
	ResourceType string // Resource type being tested
	Input        string // Input JSON state
	Expected     string // Expected output JSON
	ShouldError  bool   // Whether an error is expected
}

// TestSuite provides a comprehensive test suite for a resource transformer
type TestSuite struct {
	ResourceType string
	ConfigTests  []ConfigTestCase
	StateTests   []StateTestCase
}

// RunConfigTest runs a single configuration transformation test
func RunConfigTest(t *testing.T, tc ConfigTestCase) {
	t.Helper()
	t.Run(tc.Name, func(t *testing.T) {
		// Create registry with just the resource being tested
		// Default to v4_to_v5 for backward compatibility in tests
		reg := core.NewRegistry()
		if err := resources.RegisterAll(reg, "v4_to_v5", tc.ResourceType); err != nil {
			t.Fatalf("failed to register resource %s: %v", tc.ResourceType, err)
		}

		// Process the configuration
		result, err := processing.ProcessConfig([]byte(tc.Input), "test.tf", reg)
		
		// Check error expectation
		if tc.ShouldError {
			if err == nil {
				t.Fatal("expected error but got none")
			}
			return
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Normalize whitespace for comparison
		actual := normalizeHCL(string(result))
		expected := normalizeHCL(tc.Expected)

		if actual != expected {
			t.Errorf("transformation mismatch\nInput:\n%s\n\nExpected:\n%s\n\nActual:\n%s",
				tc.Input, tc.Expected, string(result))
		}
	})
}

// RunStateTest runs a single state transformation test
func RunStateTest(t *testing.T, tc StateTestCase) {
	t.Helper()
	t.Run(tc.Name, func(t *testing.T) {
		// Create registry with just the resource being tested
		// Default to v4_to_v5 for backward compatibility in tests
		reg := core.NewRegistry()
		if err := resources.RegisterAll(reg, "v4_to_v5", tc.ResourceType); err != nil {
			t.Fatalf("failed to register resource %s: %v", tc.ResourceType, err)
		}

		// Process the state
		result, err := processing.ProcessState([]byte(tc.Input), "terraform.tfstate", reg)
		
		// Check error expectation
		if tc.ShouldError {
			if err == nil {
				t.Fatal("expected error but got none")
			}
			return
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Compare JSON (ignoring formatting and field order)
		var actualJSON, expectedJSON interface{}
		if err := json.Unmarshal(result, &actualJSON); err != nil {
			t.Fatalf("failed to parse actual JSON: %v", err)
		}
		if err := json.Unmarshal([]byte(tc.Expected), &expectedJSON); err != nil {
			t.Fatalf("failed to parse expected JSON: %v", err)
		}

		if !reflect.DeepEqual(actualJSON, expectedJSON) {
			// Pretty print for better error messages
			actualPretty, _ := json.MarshalIndent(actualJSON, "", "  ")
			expectedPretty, _ := json.MarshalIndent(expectedJSON, "", "  ")
			
			t.Errorf("state transformation mismatch\nInput:\n%s\n\nExpected:\n%s\n\nActual:\n%s",
				tc.Input, string(expectedPretty), string(actualPretty))
		}
	})
}

// RunTestSuite runs a complete test suite for a resource transformer
func RunTestSuite(t *testing.T, suite TestSuite) {
	t.Run(suite.ResourceType, func(t *testing.T) {
		// Run all config tests
		if len(suite.ConfigTests) > 0 {
			t.Run("Config", func(t *testing.T) {
				for _, tc := range suite.ConfigTests {
					tc.ResourceType = suite.ResourceType // Ensure resource type is set
					RunConfigTest(t, tc)
				}
			})
		}

		// Run all state tests
		if len(suite.StateTests) > 0 {
			t.Run("State", func(t *testing.T) {
				for _, tc := range suite.StateTests {
					tc.ResourceType = suite.ResourceType // Ensure resource type is set
					RunStateTest(t, tc)
				}
			})
		}
	})
}

// QuickTest provides a simple way to test a basic transformation
func QuickTest(t *testing.T, resourceType, input, expected string) {
	t.Helper()
	RunConfigTest(t, ConfigTestCase{
		Name:         "quick_test",
		ResourceType: resourceType,
		Input:        input,
		Expected:     expected,
		ShouldError:  false,
	})
}

// AssertTransformation is a helper for inline assertions
func AssertTransformation(t *testing.T, resourceType string, input, expected string) {
	t.Helper()
	
	reg := core.NewRegistry()
	if err := resources.RegisterAll(reg, "v4_to_v5", resourceType); err != nil {
		t.Fatalf("failed to register resource: %v", err)
	}

	result, err := processing.ProcessConfig([]byte(input), "test.tf", reg)
	if err != nil {
		t.Fatalf("transformation failed: %v", err)
	}

	actual := normalizeHCL(string(result))
	expected = normalizeHCL(expected)

	if actual != expected {
		t.Errorf("unexpected result:\ngot:\n%s\nwant:\n%s", string(result), expected)
	}
}

// BenchmarkTransformation benchmarks a resource transformation
func BenchmarkTransformation(b *testing.B, resourceType, input string) {
	reg := core.NewRegistry()
	if err := resources.RegisterAll(reg, "v4_to_v5", resourceType); err != nil {
		b.Fatalf("failed to register resource: %v", err)
	}

	inputBytes := []byte(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := processing.ProcessConfig(inputBytes, "test.tf", reg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// GenerateTestTable helps generate test cases for common patterns
func GenerateTestTable(resourceType string, oldAttrs, newAttrs []string) []ConfigTestCase {
	var tests []ConfigTestCase

	// Generate test for each attribute rename
	for i, oldAttr := range oldAttrs {
		if i < len(newAttrs) {
			newAttr := newAttrs[i]
			tests = append(tests, ConfigTestCase{
				Name:         "rename_" + oldAttr + "_to_" + newAttr,
				ResourceType: resourceType,
				Input: fmt.Sprintf(`resource "%s" "test" {
  %s = "value"
}`, resourceType, oldAttr),
				Expected: fmt.Sprintf(`resource "%s" "test" {
  %s = "value"
}`, resourceType, newAttr),
			})
		}
	}

	return tests
}