package testhelpers

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

// ResourceTestSuite contains both config and state tests for a resource
type ResourceTestSuite struct {
	ConfigTests []ConfigTestCase
	StateTests  []StateTestCase
}

// RunResourceTestSuite runs a complete test suite for a resource migrator
func RunResourceTestSuite(t *testing.T, suite ResourceTestSuite, factory func() transform.ResourceTransformer) {
	t.Run("ConfigTransformation", func(t *testing.T) {
		RunConfigTransformTestsWithFactory(t, suite.ConfigTests, factory)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		RunStateTransformTestsWithFactory(t, suite.StateTests, factory)
	})
}

// AssertResourceType verifies that a migrator handles the expected resource types
func AssertResourceType(t *testing.T, migrator transform.ResourceTransformer, expectedTypes ...string) {
	t.Helper()
	
	for _, resourceType := range expectedTypes {
		if !migrator.CanHandle(resourceType) {
			t.Errorf("Migrator should handle resource type %q but doesn't", resourceType)
		}
	}
}

// AssertNotResourceType verifies that a migrator doesn't handle certain resource types
func AssertNotResourceType(t *testing.T, migrator transform.ResourceTransformer, unexpectedTypes ...string) {
	t.Helper()
	
	for _, resourceType := range unexpectedTypes {
		if migrator.CanHandle(resourceType) {
			t.Errorf("Migrator should not handle resource type %q but does", resourceType)
		}
	}
}