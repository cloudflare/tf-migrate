package testhelpers

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/registry"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/stretchr/testify/assert"
)

// StateTestCase represents a test case for state transformations
type StateTestCase struct {
	Name     string
	Input    string
	Expected string
}

func runStateTransformTest(t *testing.T, tt StateTestCase) {
	t.Helper()
	a := assert.New(t)
	registry.RegisterAllMigrations()

	_, statePipeline := SetupPipelines()
	ctx := &transform.Context{
		StateJSON:     tt.Input,
		Resources:     make([]string, 0),
		TargetVersion: 5,
		SourceVersion: 4,
		Metadata:      make(map[string]interface{}),
		Filename:      "test.tf",
	}

	result, err := statePipeline.Transform(ctx)
	a.Nil(err)

	// Compare JSON (normalize both for comparison)
	a.JSONEq(tt.Expected, string(result), "State transformation mismatch")
}

// RunStateTransformTests runs multiple state transformation tests
func RunStateTransformTests(t *testing.T, tests []StateTestCase) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			runStateTransformTest(t, tt)
		})
	}
}
