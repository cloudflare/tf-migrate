package v4_to_v5

import (
	"fmt"
	"os"
	"testing"

	"github.com/cloudflare/tf-migrate/integration"
	"github.com/cloudflare/tf-migrate/internal/registry"
	// Explicitly import the migrations we want to test
	_ "github.com/cloudflare/tf-migrate/internal/resources/dns_record"
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

	tests := []integration.TestCase{
		{
			Name:        "DNSRecord",
			Description: "Migrate cloudflare_record to cloudflare_dns_record",
			Resource:    "dns_record",
		},
		// Add more v4 to v5 migrations here as they are implemented
	}

	for _, test := range tests {
		runner.RunTest(t, test)
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

	test := integration.TestCase{
		Name:        resource,
		Description: fmt.Sprintf("Testing %s migration from v4 to v5", resource),
		Resource:    resource,
	}

	runner.RunTest(t, test)
}
