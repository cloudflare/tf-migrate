package v4_to_v5

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudflare/tf-migrate/integration"
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func TestV4ToV5Migration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	runner, err := integration.NewTestRunner("v4", "v5")
	if err != nil {
		t.Fatalf("Failed to create test runner: %v", err)
	}

	// Dynamically discover resources from testdata directory
	testdataPath := "testdata"
	entries, err := os.ReadDir(testdataPath)
	if err != nil {
		t.Fatalf("Failed to read testdata directory: %v", err)
	}

	var resources []string
	for _, entry := range entries {
		if entry.IsDir() {
			resources = append(resources, entry.Name())
		}
	}

	if len(resources) == 0 {
		t.Fatal("No resources found in testdata directory")
	}

	t.Logf("Discovered %d resources to test: %v", len(resources), resources)

	for _, resource := range resources {
		runner.RunTest(t, integration.TestCase{
			Resource: resource,
		})
	}
}

// TestSingleResource allows testing a specific resource during development
func TestSingleResource(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Set environment variable TEST_RESOURCE to test a specific resource
	// e.g., TEST_RESOURCE=dns_record go test ./integration/v4_to_v5
	resource := os.Getenv("TEST_RESOURCE")
	if resource == "" {
		t.Skip("TEST_RESOURCE not set, skipping single resource test")
	}

	// For v4_to_v5 tests, we always use version v4 to v5
	runner, err := integration.NewTestRunner("v4", "v5")
	if err != nil {
		t.Fatalf("Failed to create test runner: %v", err)
	}

	runner.RunTest(t, integration.TestCase{
		Resource: resource,
	})
}

func TestGatewayPolicyPrecedenceReview(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	runner, err := integration.NewTestRunner("v4", "v5")
	if err != nil {
		t.Fatal(err)
	}
	// Duplicate precedences require manual resolution, so this config-only
	// fixture must not be discovered by the live E2E runner's testdata scan.
	runner.TestDataPath = filepath.Join(runner.BaseDir, "fixtures")
	runner.RunTest(t, integration.TestCase{
		Resource: "zero_trust_gateway_policy",
		ExpectedOutput: []string{
			"Action required: review Gateway policy precedence",
			"cloudflare_teams_rule.cftftest_gitlab_staging",
			"cloudflare_zero_trust_gateway_policy.cftftest_harbor_staging",
			"cloudflare_teams_rule.cftftest_catch_all_block",
			"assign unique precedence values",
		},
	})
}
