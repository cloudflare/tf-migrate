package zero_trust_gateway_certificate

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
)

func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.ConfigTestCase{
		{
			Name: "basic unchanged fields",
			Input: `
resource "cloudflare_zero_trust_gateway_certificate" "test" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  validity_period_days = 1
  activate             = true
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_certificate" "test" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  validity_period_days = 1
  activate             = true
}`,
		},
		{
			Name: "removes custom field",
			Input: `
resource "cloudflare_zero_trust_gateway_certificate" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  custom     = true
  activate   = true
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_certificate" "test" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  activate   = true
}`,
		},
		{
			Name: "removes gateway_managed field",
			Input: `
resource "cloudflare_zero_trust_gateway_certificate" "test" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  gateway_managed      = true
  validity_period_days = 1826
  activate             = true
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_certificate" "test" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  validity_period_days = 1826
  activate             = true
}`,
		},
		{
			Name: "removes id field from custom certificate config",
			Input: `
resource "cloudflare_zero_trust_gateway_certificate" "custom" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  custom     = true
  id         = "existing-cert-uuid-123"
  activate   = false
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_certificate" "custom" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  activate   = false
}`,
		},
		{
			Name: "multiple resources in one file",
			Input: `
resource "cloudflare_zero_trust_gateway_certificate" "first" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  custom     = true
  id         = "cert-1"
}

resource "cloudflare_zero_trust_gateway_certificate" "second" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  gateway_managed      = true
  validity_period_days = 3650
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_certificate" "first" {
  account_id = "f037e56e89293a057740de681ac9abbe"
}

resource "cloudflare_zero_trust_gateway_certificate" "second" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  validity_period_days = 3650
}`,
		},
		{
			Name: "comprehensive - all user-defined fields",
			Input: `
resource "cloudflare_zero_trust_gateway_certificate" "comprehensive" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  custom               = true
  gateway_managed      = false
  id                   = "test-cert-uuid-789"
  validity_period_days = 1826
  activate             = true
}`,
			Expected: `
resource "cloudflare_zero_trust_gateway_certificate" "comprehensive" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  validity_period_days = 1826
  activate             = true
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, testCases, migrator)
}

func TestStateTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.StateTestCase{
		{
			Name: "basic unchanged fields",
			Input: `{
				"attributes": {
					"account_id": "f037e56e89293a057740de681ac9abbe",
					"validity_period_days": 1,
					"activate": true,
					"id": "test-id-123"
				}
			}`,
			Expected: `{
				"attributes": {
					"account_id": "f037e56e89293a057740de681ac9abbe",
					"activate": true,
					"id": "test-id-123"
				}
			}`,
		},
		{
			Name: "removes custom field",
			Input: `{
				"attributes": {
					"account_id": "f037e56e89293a057740de681ac9abbe",
					"custom": true,
					"activate": true,
					"id": "test-id-456"
				}
			}`,
			Expected: `{
				"attributes": {
					"account_id": "f037e56e89293a057740de681ac9abbe",
					"activate": true,
					"id": "test-id-456"
				}
			}`,
		},
		{
			Name: "removes gateway_managed field",
			Input: `{
				"attributes": {
					"account_id": "f037e56e89293a057740de681ac9abbe",
					"gateway_managed": true,
					"validity_period_days": 1826,
					"activate": true,
					"id": "test-id-789"
				}
			}`,
			Expected: `{
				"attributes": {
					"account_id": "f037e56e89293a057740de681ac9abbe",
					"activate": true,
					"id": "test-id-789"
				}
			}`,
		},
		{
			Name: "keeps id field in state (unlike config)",
			Input: `{
				"attributes": {
					"account_id": "f037e56e89293a057740de681ac9abbe",
					"custom": true,
					"id": "existing-cert-uuid-123",
					"activate": false
				}
			}`,
			Expected: `{
				"attributes": {
					"account_id": "f037e56e89293a057740de681ac9abbe",
					"id": "existing-cert-uuid-123",
					"activate": false
				}
			}`,
		},
		{
			Name: "removes qs_pack_id field",
			Input: `{
				"attributes": {
					"account_id": "f037e56e89293a057740de681ac9abbe",
					"qs_pack_id": "qs-pack-123",
					"validity_period_days": 1826,
					"activate": true,
					"id": "test-id-qs"
				}
			}`,
			Expected: `{
				"attributes": {
					"account_id": "f037e56e89293a057740de681ac9abbe",
					"activate": true,
					"id": "test-id-qs"
				}
			}`,
		},
		{
			Name: "comprehensive - all user-defined fields",
			Input: `{
				"attributes": {
					"account_id": "f037e56e89293a057740de681ac9abbe",
					"custom": true,
					"gateway_managed": false,
					"id": "test-cert-uuid-789",
					"qs_pack_id": "qs-pack-456",
					"validity_period_days": 1826,
					"activate": true
				}
			}`,
			Expected: `{
				"attributes": {
					"account_id": "f037e56e89293a057740de681ac9abbe",
					"id": "test-cert-uuid-789",
					"activate": true
				}
			}`,
		},
		{
			Name: "preserves all v4 computed fields except qs_pack_id",
			Input: `{
				"attributes": {
					"account_id": "f037e56e89293a057740de681ac9abbe",
					"id": "test-cert-uuid-computed",
					"validity_period_days": 1826,
					"activate": true,
					"in_use": true,
					"binding_status": "active",
					"uploaded_on": "2024-01-15T10:30:00Z",
					"created_at": "2024-01-15T10:30:00Z",
					"expires_on": "2029-01-15T10:30:00Z",
					"qs_pack_id": "qs-pack-to-remove"
				}
			}`,
			Expected: `{
				"attributes": {
					"account_id": "f037e56e89293a057740de681ac9abbe",
					"id": "test-cert-uuid-computed",
					"activate": true,
					"in_use": true,
					"binding_status": "active",
					"uploaded_on": "2024-01-15T10:30:00Z",
					"created_at": "2024-01-15T10:30:00Z",
					"expires_on": "2029-01-15T10:30:00Z"
				}
			}`,
		},
	}

	testhelpers.RunStateTransformTests(t, testCases, migrator)
}

func TestCanHandle(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []struct {
		name         string
		resourceType string
		expected     bool
	}{
		{
			name:         "handles cloudflare_zero_trust_gateway_certificate",
			resourceType: "cloudflare_zero_trust_gateway_certificate",
			expected:     true,
		},
		{
			name:         "does not handle other resources",
			resourceType: "cloudflare_teams_list",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := migrator.CanHandle(tt.resourceType)
			if result != tt.expected {
				t.Errorf("CanHandle(%q) = %v, want %v", tt.resourceType, result, tt.expected)
			}
		})
	}
}

func TestGetResourceType(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	expected := "cloudflare_zero_trust_gateway_certificate"
	result := migrator.GetResourceType()

	if result != expected {
		t.Errorf("GetResourceType() = %q, want %q", result, expected)
	}
}

// TestValidityPeriodDaysDefaultHandling tests the combined config+state transformation
// for handling the v4 default value of validity_period_days
func TestValidityPeriodDaysDefaultHandling(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []struct {
		name          string
		config        string
		state         string
		expectedState string
		description   string
	}{
		{
			name: "explicit validity_period_days in config - keeps in state",
			config: `
resource "cloudflare_zero_trust_gateway_certificate" "test" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  gateway_managed      = true
  validity_period_days = 3650
  activate             = true
}`,
			state: `{
				"resources": [{
					"type": "cloudflare_zero_trust_gateway_certificate",
					"name": "test",
					"instances": [{
						"attributes": {
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"gateway_managed": true,
							"validity_period_days": 3650,
							"activate": true,
							"id": "cert-123"
						}
					}]
				}]
			}`,
			expectedState: `{
				"resources": [{
					"type": "cloudflare_zero_trust_gateway_certificate",
					"name": "test",
					"instances": [{
						"attributes": {
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"validity_period_days": 3650.0,
							"activate": true,
							"id": "cert-123"
						}
					}]
				}]
			}`,
			description: "When validity_period_days is explicitly set in config, it should be kept in state (converted to float64)",
		},
		{
			name: "no validity_period_days in config - removes from state",
			config: `
resource "cloudflare_zero_trust_gateway_certificate" "test" {
  account_id      = "f037e56e89293a057740de681ac9abbe"
  gateway_managed = true
  activate        = true
}`,
			state: `{
				"resources": [{
					"type": "cloudflare_zero_trust_gateway_certificate",
					"name": "test",
					"instances": [{
						"attributes": {
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"gateway_managed": true,
							"validity_period_days": 1826,
							"activate": true,
							"id": "cert-456"
						}
					}]
				}]
			}`,
			expectedState: `{
				"resources": [{
					"type": "cloudflare_zero_trust_gateway_certificate",
					"name": "test",
					"instances": [{
						"attributes": {
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"activate": true,
							"id": "cert-456"
						}
					}]
				}]
			}`,
			description: "When validity_period_days is NOT in config (v4 default), it should be removed from state",
		},
		{
			name: "default value 1826 explicitly set - keeps in state",
			config: `
resource "cloudflare_zero_trust_gateway_certificate" "test" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  gateway_managed      = true
  validity_period_days = 1826
}`,
			state: `{
				"resources": [{
					"type": "cloudflare_zero_trust_gateway_certificate",
					"name": "test",
					"instances": [{
						"attributes": {
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"gateway_managed": true,
							"validity_period_days": 1826,
							"id": "cert-789"
						}
					}]
				}]
			}`,
			expectedState: `{
				"resources": [{
					"type": "cloudflare_zero_trust_gateway_certificate",
					"name": "test",
					"instances": [{
						"attributes": {
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"validity_period_days": 1826.0,
							"id": "cert-789"
						}
					}]
				}]
			}`,
			description: "Even if the value is 1826 (v4 default), keep it if explicitly set in config",
		},
		{
			name: "multiple resources with mixed explicit/implicit",
			config: `
resource "cloudflare_zero_trust_gateway_certificate" "explicit" {
  account_id           = "f037e56e89293a057740de681ac9abbe"
  gateway_managed      = true
  validity_period_days = 7300
}

resource "cloudflare_zero_trust_gateway_certificate" "implicit" {
  account_id      = "f037e56e89293a057740de681ac9abbe"
  gateway_managed = true
}`,
			state: `{
				"resources": [
					{
						"type": "cloudflare_zero_trust_gateway_certificate",
						"name": "explicit",
						"instances": [{
							"attributes": {
								"account_id": "f037e56e89293a057740de681ac9abbe",
								"gateway_managed": true,
								"validity_period_days": 7300,
								"id": "cert-explicit"
							}
						}]
					},
					{
						"type": "cloudflare_zero_trust_gateway_certificate",
						"name": "implicit",
						"instances": [{
							"attributes": {
								"account_id": "f037e56e89293a057740de681ac9abbe",
								"gateway_managed": true,
								"validity_period_days": 1826,
								"id": "cert-implicit"
							}
						}]
					}
				]
			}`,
			expectedState: `{
				"resources": [
					{
						"type": "cloudflare_zero_trust_gateway_certificate",
						"name": "explicit",
						"instances": [{
							"attributes": {
								"account_id": "f037e56e89293a057740de681ac9abbe",
								"validity_period_days": 7300.0,
								"id": "cert-explicit"
							}
						}]
					},
					{
						"type": "cloudflare_zero_trust_gateway_certificate",
						"name": "implicit",
						"instances": [{
							"attributes": {
								"account_id": "f037e56e89293a057740de681ac9abbe",
								"id": "cert-implicit"
							}
						}]
					}
				]
			}`,
			description: "Multiple resources: explicit validity_period_days kept, implicit removed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Parse config file and transform to populate metadata
			configFile, diags := hclwrite.ParseConfig([]byte(tt.config), "test.tf", hcl.InitialPos)
			require.False(t, diags.HasErrors(), "Failed to parse config: %v", diags)

			// Create context and transform config to populate metadata
			ctx := &transform.Context{
				StateJSON: tt.state,
				CFGFiles: map[string]*hclwrite.File{
					"test.tf": configFile,
				},
			}

			// Transform config blocks to populate metadata
			body := configFile.Body()
			for _, block := range body.Blocks() {
				if block.Type() == "resource" && len(block.Labels()) >= 2 {
					resourceType := block.Labels()[0]
					if migrator.CanHandle(resourceType) {
						_, err := migrator.TransformConfig(ctx, block)
						require.NoError(t, err, "Failed to transform config")
					}
				}
			}

			// Step 2: Parse state
			inputResult := gjson.Parse(tt.state)

			// Step 3: Run state transformation using the context with metadata
			transformedState, err := migrator.TransformState(ctx, inputResult, "", "")
			require.NoError(t, err, "TransformState failed")

			// Step 4: Compare results
			assert.JSONEq(t, tt.expectedState, transformedState, tt.description)
		})
	}
}
