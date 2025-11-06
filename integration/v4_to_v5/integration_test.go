package v4_to_v5

import (
	"os"
	"testing"

	"github.com/cloudflare/tf-migrate/integration"
	"github.com/cloudflare/tf-migrate/internal/registry"

	// Explicitly import the migrations we want to test
	_ "github.com/cloudflare/tf-migrate/internal/resources/account_member"
	_ "github.com/cloudflare/tf-migrate/internal/resources/api_token"
	_ "github.com/cloudflare/tf-migrate/internal/resources/dns_record"
	_ "github.com/cloudflare/tf-migrate/internal/resources/workers_kv_namespace"
	_ "github.com/cloudflare/tf-migrate/internal/resources/logpull_retention"
	_ "github.com/cloudflare/tf-migrate/internal/resources/zero_trust_access_service_token"
	_ "github.com/cloudflare/tf-migrate/internal/resources/zero_trust_gateway_policy"
	_ "github.com/cloudflare/tf-migrate/internal/resources/zero_trust_list"
)

// TestMain explicitly registers migrations for this version path
func TestMain(m *testing.M) {
	// Explicitly register the migrations for v4 to v5
	// This is called once before all tests in this package
	registry.RegisterAllMigrations()

	// Run the tests
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

	// List of resources to test for v4 to v5 migration
	resources := []string{
		"account_member",
		"api_token",
		"dns_record",
		"logpull_retention",
		"workers_kv_namespace",
		"zero_trust_access_service_token",
		"zero_trust_gateway_policy",
		"zero_trust_list",
	}

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
