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

// runStateTransformTest runs a single state transformation test
func runStateTransformTest(t *testing.T, tt StateTestCase, migrator transform.ResourceTransformer) {
	t.Helper()

	// Parse the input JSON to create the gjson.Result
	inputResult := gjson.Parse(tt.Input)
	
	// Create context
	ctx := &transform.Context{
		StateJSON: tt.Input,
	}

	// Transform the state - pass the parsed input as the instance
	result, err := migrator.TransformState(ctx, inputResult, "")
	require.NoError(t, err, "Failed to transform state")

	// Compare JSON (normalize both for comparison)
	assert.JSONEq(t, tt.Expected, result, "State transformation mismatch")
}

// RunStateTransformTests runs multiple state transformation tests
func RunStateTransformTests(t *testing.T, tests []StateTestCase, migrator transform.ResourceTransformer) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			runStateTransformTest(t, tt, migrator)
		})
	}
}

