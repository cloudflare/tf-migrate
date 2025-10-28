package testhelpers

import (
	"strings"
	"testing"

	"github.com/cloudflare/tf-migrate/internal"
	"github.com/cloudflare/tf-migrate/internal/pipeline"
	"github.com/cloudflare/tf-migrate/internal/registry"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

// ConfigTestCase represents a test case for configuration transformations
type ConfigTestCase struct {
	Name     string
	Input    string
	Expected string
}

func SetupPipelines(resources ...string) (*pipeline.Pipeline, *pipeline.Pipeline) {
	logger := hclog.New(&hclog.LoggerOptions{})
	getFunc := func(resourceType string, source, target int) transform.ResourceTransformer {
		return internal.GetMigrator(resourceType, source, target)
	}
	getAllFunc := func(source, target int, resourcesToMigrate ...string) []transform.ResourceTransformer {
		return internal.GetAllMigrators(source, target, resources...)
	}
	providers := transform.NewMigratorProvider(getFunc, getAllFunc)

	return pipeline.BuildConfigPipeline(logger, providers), pipeline.BuildStatePipeline(logger, providers)
}

func runV4ToV5ConfigTransformTest(t *testing.T, tt ConfigTestCase) {
	t.Helper()
	registry.RegisterAllMigrations()
	a := assert.New(t)
	cfgPipeline, _ := SetupPipelines()
	ctx := &transform.Context{
		Content:       []byte(tt.Input),
		Resources:     make([]string, 0),
		TargetVersion: 5,
		SourceVersion: 4,
		Metadata:      make(map[string]interface{}),
		Filename:      "test.tf",
	}

	result, err := cfgPipeline.Transform(ctx)
	a.Nil(err)
	
	expectedFile, diags := hclwrite.ParseConfig([]byte(tt.Expected), "expected.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "Failed to parse expected HCL: %v", diags)
	expectedOutput := strings.TrimSpace(string(hclwrite.Format(expectedFile.Bytes())))

	a.Equal(expectedOutput, strings.TrimSpace(string(result)))
}

// RunConfigTransformTests runs multiple configuration transformation tests
func RunConfigTransformTests(t *testing.T, tests []ConfigTestCase) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			runV4ToV5ConfigTransformTest(t, tt)
		})
	}
}
