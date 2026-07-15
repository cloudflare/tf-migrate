package workers_secret

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

func TestV4ToV5Transformation(t *testing.T) {
	t.Run("ConfigTransformation", func(t *testing.T) {
		t.Run("WorkersSecretRemoved", testWorkersSecretRemoved)
		t.Run("WorkerSecretSingularRemoved", testWorkerSecretSingularRemoved)
	})

	t.Run("PhaseOne", func(t *testing.T) {
		t.Run("WorkersSecretPhaseOne", testWorkersSecretPhaseOne)
		t.Run("WorkerSecretSingularPhaseOne", testWorkerSecretSingularPhaseOne)
	})

	t.Run("CrossResourceMigration", func(t *testing.T) {
		t.Run("SingleSecretMergedIntoScript", testSingleSecretMergedIntoScript)
		t.Run("MultipleSecretsMergedIntoScript", testMultipleSecretsMergedIntoScript)
		t.Run("SecretMergedIntoScriptWithExistingBindings", testSecretMergedIntoScriptWithExistingBindings)
		t.Run("OrphanSecretRemovedWhenNoParent", testOrphanSecretRemovedWhenNoParent)
		t.Run("SecretMatchedByLiteralScriptName", testSecretMatchedByLiteralScriptName)
		t.Run("SingularWorkerSecretMergedIntoScript", testSingularWorkerSecretMergedIntoScript)
		t.Run("SecretWithReferenceToSingularWorkerScript", testSecretWithReferenceToSingularWorkerScript)
	})
}

func testWorkersSecretRemoved(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "workers_secret should produce removed block",
			Input: `resource "cloudflare_workers_secret" "my_secret" {
  account_id  = "abc123"
  script_name = "my-worker"
  name        = "MY_SECRET"
  secret_text = "super-secret"
}`,
			Expected: `removed {
  from = cloudflare_workers_secret.my_secret

  lifecycle {
    destroy = false
  }
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func testWorkerSecretSingularRemoved(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "worker_secret (singular) should produce removed block",
			Input: `resource "cloudflare_worker_secret" "my_secret" {
  account_id  = "abc123"
  script_name = "my-worker"
  name        = "MY_SECRET"
  secret_text = "super-secret"
}`,
			Expected: `removed {
  from = cloudflare_worker_secret.my_secret

  lifecycle {
    destroy = false
  }
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

// Phase-one tests

func testWorkersSecretPhaseOne(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	p1, ok := migrator.(transform.PhaseOneTransformer)
	require.True(t, ok, "migrator must implement PhaseOneTransformer")

	input := `resource "cloudflare_workers_secret" "my_secret" {
  account_id  = "abc123"
  script_name = "my-worker"
  name        = "MY_SECRET"
  secret_text = "super-secret"
}`
	file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	block := file.Body().Blocks()[0]
	ctx := &transform.Context{Filename: "test.tf", CFGFile: file}
	result, err := p1.TransformPhaseOne(ctx, block)
	require.NoError(t, err)
	assert.False(t, result.RemoveOriginal, "PhaseOne must not remove the original block")
	require.Len(t, result.Blocks, 1)
	assert.Equal(t, "removed", result.Blocks[0].Type())
}

func testWorkerSecretSingularPhaseOne(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	p1, ok := migrator.(transform.PhaseOneTransformer)
	require.True(t, ok, "migrator must implement PhaseOneTransformer")

	input := `resource "cloudflare_worker_secret" "my_secret" {
  account_id  = "abc123"
  script_name = "my-worker"
  name        = "MY_SECRET"
  secret_text = "super-secret"
}`
	file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors())

	block := file.Body().Blocks()[0]
	ctx := &transform.Context{Filename: "test.tf", CFGFile: file}
	result, err := p1.TransformPhaseOne(ctx, block)
	require.NoError(t, err)
	assert.False(t, result.RemoveOriginal)
	require.Len(t, result.Blocks, 1)

	// Verify the removed block uses the original (singular) resource type
	removedBody := result.Blocks[0].Body()
	fromAttr := removedBody.GetAttribute("from")
	require.NotNil(t, fromAttr)
	fromExpr := strings.TrimSpace(string(fromAttr.Expr().BuildTokens(nil).Bytes()))
	assert.Contains(t, fromExpr, "cloudflare_worker_secret.my_secret")
}

// Cross-resource migration tests

func testSingleSecretMergedIntoScript(t *testing.T) {
	input := `resource "cloudflare_workers_script" "my_worker" {
  account_id  = "abc123"
  script_name = "my-worker"
  content     = "addEventListener('fetch', event => {});"
}

resource "cloudflare_workers_secret" "my_secret" {
  account_id  = "abc123"
  script_name = cloudflare_workers_script.my_worker.script_name
  name        = "MY_SECRET"
  secret_text = "super-secret"
}`

	expected := `resource "cloudflare_workers_script" "my_worker" {
  account_id  = "abc123"
  script_name = "my-worker"
  content     = "addEventListener('fetch', event => {});"
  bindings = [
  {
    type = "secret_text"
    name = "MY_SECRET"
    text = "super-secret"
  }
]
}`

	runCrossResourceTest(t, input, expected)
}

func testMultipleSecretsMergedIntoScript(t *testing.T) {
	input := `resource "cloudflare_workers_script" "my_worker" {
  account_id  = "abc123"
  script_name = "my-worker"
  content     = "addEventListener('fetch', event => {});"
}

resource "cloudflare_workers_secret" "secret_one" {
  account_id  = "abc123"
  script_name = cloudflare_workers_script.my_worker.script_name
  name        = "SECRET_ONE"
  secret_text = "first-secret"
}

resource "cloudflare_workers_secret" "secret_two" {
  account_id  = "abc123"
  script_name = cloudflare_workers_script.my_worker.script_name
  name        = "SECRET_TWO"
  secret_text = "second-secret"
}`

	expected := `resource "cloudflare_workers_script" "my_worker" {
  account_id  = "abc123"
  script_name = "my-worker"
  content     = "addEventListener('fetch', event => {});"
  bindings = [
  {
    type = "secret_text"
    name = "SECRET_ONE"
    text = "first-secret"
  }, {
    type = "secret_text"
    name = "SECRET_TWO"
    text = "second-secret"
  }
]
}`

	runCrossResourceTest(t, input, expected)
}

func testSecretMergedIntoScriptWithExistingBindings(t *testing.T) {
	input := `resource "cloudflare_workers_script" "my_worker" {
  account_id  = "abc123"
  script_name = "my-worker"
  content     = "addEventListener('fetch', event => {});"
  bindings = [
    {
      type = "kv_namespace"
      name = "MY_KV"
      namespace_id = "kv-id-123"
    }
  ]
}

resource "cloudflare_workers_secret" "my_secret" {
  account_id  = "abc123"
  script_name = cloudflare_workers_script.my_worker.script_name
  name        = "MY_SECRET"
  secret_text = "super-secret"
}`

	expected := `resource "cloudflare_workers_script" "my_worker" {
  account_id  = "abc123"
  script_name = "my-worker"
  content     = "addEventListener('fetch', event => {});"
  bindings = concat([
    {
      type = "kv_namespace"
      name = "MY_KV"
      namespace_id = "kv-id-123"
    }
  ], [
  {
    type = "secret_text"
    name = "MY_SECRET"
    text = "super-secret"
  }
])
}`

	runCrossResourceTest(t, input, expected)
}

func testOrphanSecretRemovedWhenNoParent(t *testing.T) {
	input := `resource "cloudflare_workers_secret" "orphan_secret" {
  account_id  = "abc123"
  script_name = "some-other-worker"
  name        = "ORPHAN_SECRET"
  secret_text = "orphan-value"
}`

	// Orphan secret should be removed (no parent script in file)
	expected := ``

	runCrossResourceTest(t, input, expected)
}

func testSecretMatchedByLiteralScriptName(t *testing.T) {
	input := `resource "cloudflare_workers_script" "my_worker" {
  account_id  = "abc123"
  script_name = "my-worker"
  content     = "addEventListener('fetch', event => {});"
}

resource "cloudflare_workers_secret" "my_secret" {
  account_id  = "abc123"
  script_name = "my-worker"
  name        = "MY_SECRET"
  secret_text = "super-secret"
}`

	expected := `resource "cloudflare_workers_script" "my_worker" {
  account_id  = "abc123"
  script_name = "my-worker"
  content     = "addEventListener('fetch', event => {});"
  bindings = [
  {
    type = "secret_text"
    name = "MY_SECRET"
    text = "super-secret"
  }
]
}`

	runCrossResourceTest(t, input, expected)
}

func testSingularWorkerSecretMergedIntoScript(t *testing.T) {
	input := `resource "cloudflare_workers_script" "my_worker" {
  account_id  = "abc123"
  script_name = "my-worker"
  content     = "addEventListener('fetch', event => {});"
}

resource "cloudflare_worker_secret" "my_secret" {
  account_id  = "abc123"
  script_name = cloudflare_workers_script.my_worker.script_name
  name        = "MY_SECRET"
  secret_text = "super-secret"
}`

	expected := `resource "cloudflare_workers_script" "my_worker" {
  account_id  = "abc123"
  script_name = "my-worker"
  content     = "addEventListener('fetch', event => {});"
  bindings = [
  {
    type = "secret_text"
    name = "MY_SECRET"
    text = "super-secret"
  }
]
}`

	runCrossResourceTest(t, input, expected)
}

func testSecretWithReferenceToSingularWorkerScript(t *testing.T) {
	// After the workers_script migrator runs, cloudflare_worker_script is renamed
	// to cloudflare_workers_script and "name" becomes "script_name".
	// ProcessCrossResourceConfigMigration only merges into already-migrated scripts.
	input := `resource "cloudflare_workers_script" "my_worker" {
  account_id  = "abc123"
  script_name = "my-worker"
  content     = "addEventListener('fetch', event => {});"
}

resource "cloudflare_workers_secret" "my_secret" {
  account_id  = "abc123"
  script_name = cloudflare_worker_script.my_worker.name
  name        = "MY_SECRET"
  secret_text = "super-secret"
}`

	// The secret references cloudflare_worker_script (v4 name) but the script
	// has already been renamed to cloudflare_workers_script. The cross-resource
	// merge should still match because it checks both prefixes.
	expected := `resource "cloudflare_workers_script" "my_worker" {
  account_id  = "abc123"
  script_name = "my-worker"
  content     = "addEventListener('fetch', event => {});"
  bindings = [
  {
    type = "secret_text"
    name = "MY_SECRET"
    text = "super-secret"
  }
]
}`

	runCrossResourceTest(t, input, expected)
}

// runCrossResourceTest parses input HCL, runs ProcessCrossResourceConfigMigration,
// and compares the output to expected.
func runCrossResourceTest(t *testing.T, input, expected string) {
	t.Helper()

	file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "Failed to parse input HCL: %v", diags)

	ProcessCrossResourceConfigMigration(file)

	output := string(hclwrite.Format(file.Bytes()))
	output = strings.TrimSpace(output)

	if expected == "" {
		assert.Empty(t, output, "Expected empty output but got:\n%s", output)
		return
	}

	expectedFile, diags := hclwrite.ParseConfig([]byte(expected), "expected.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), "Failed to parse expected HCL: %v", diags)
	expectedOutput := string(hclwrite.Format(expectedFile.Bytes()))
	expectedOutput = strings.TrimSpace(expectedOutput)

	assert.Equal(t, expectedOutput, output)
}
