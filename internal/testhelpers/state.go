package testhelpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

// StateTestCase represents a test case for state transformations
type StateTestCase struct {
	Name     string
	Input    string
	Expected string
}

// RunStateTransformTest runs a single state transformation test
func RunStateTransformTest(t *testing.T, tt StateTestCase, migrator transform.ResourceTransformer) {
	t.Helper()

	// Create context
	ctx := &transform.Context{
		StateJSON: tt.Input,
	}

	// Transform the state
	result, err := migrator.TransformState(ctx, gjson.Result{}, "")
	require.NoError(t, err, "Failed to transform state")

	// Compare JSON (normalize both for comparison)
	assert.JSONEq(t, tt.Expected, result, "State transformation mismatch")
}

// RunStateTransformTests runs multiple state transformation tests
func RunStateTransformTests(t *testing.T, tests []StateTestCase, migrator transform.ResourceTransformer) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			RunStateTransformTest(t, tt, migrator)
		})
	}
}

// RunStateTransformTestWithFactory runs a test with a migrator factory function
func RunStateTransformTestWithFactory(t *testing.T, tt StateTestCase, factory func() transform.ResourceTransformer) {
	t.Helper()
	migrator := factory()
	RunStateTransformTest(t, tt, migrator)
}

// RunStateTransformTestsWithFactory runs multiple tests with a migrator factory function
func RunStateTransformTestsWithFactory(t *testing.T, tests []StateTestCase, factory func() transform.ResourceTransformer) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			RunStateTransformTestWithFactory(t, tt, factory)
		})
	}
}