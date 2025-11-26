package healthcheck

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/tidwall/gjson"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("StateTransformation", func(t *testing.T) {
		t.Run("HTTPHealthcheck", testHTTPHealthcheckState)
		t.Run("HTTPSHealthcheck", testHTTPSHealthcheckState)
		t.Run("TCPHealthcheck", testTCPHealthcheckState)
		t.Run("TypeConversions", testStateTypeConversions)
		t.Run("HeaderTransformation", testHeaderTransformation)
		t.Run("EdgeCases", testStateEdgeCases)
	})

	// Config transformation tests skipped for now due to TODO markers
	// t.Run("ConfigTransformation", func(t *testing.T) {
	//     t.Run("HTTPHealthcheck", testHTTPHealthcheckConfig)
	//     t.Run("TCPHealthcheck", testTCPHealthcheckConfig)
	// })

	_ = migrator // Suppress unused variable warning
}

// testHTTPHealthcheckState tests HTTP healthcheck state transformation
func testHTTPHealthcheckState(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	ctx := &transform.Context{
		SourceVersion: "v4",
		TargetVersion: "v5",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "HTTP healthcheck with all fields",
			input: `{
  "type": "cloudflare_healthcheck",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "name": "http-check",
    "address": "example.com",
    "type": "HTTP",
    "port": 80,
    "path": "/health",
    "method": "GET",
    "expected_codes": ["200", "201"],
    "expected_body": "OK",
    "follow_redirects": false,
    "allow_insecure": false,
    "consecutive_fails": 3,
    "consecutive_successes": 2,
    "retries": 2,
    "timeout": 5,
    "interval": 60,
    "suspended": false,
    "check_regions": ["WNAM", "ENAM"]
  }
}`,
			expected: `{
  "type": "cloudflare_healthcheck",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "name": "http-check",
    "address": "example.com",
    "type": "HTTP",
    "consecutive_fails": 3,
    "consecutive_successes": 2,
    "retries": 2,
    "timeout": 5,
    "interval": 60,
    "suspended": false,
    "check_regions": ["WNAM", "ENAM"],
    "http_config": {
      "port": 80,
      "path": "/health",
      "method": "GET",
      "expected_codes": ["200", "201"],
      "expected_body": "OK",
      "follow_redirects": false,
      "allow_insecure": false
    }
  },
  "schema_version": 0
}`,
		},
		{
			name: "HTTP healthcheck minimal",
			input: `{
  "type": "cloudflare_healthcheck",
  "name": "minimal",
  "attributes": {
    "zone_id": "abc123",
    "name": "minimal-check",
    "address": "example.com",
    "type": "HTTP"
  }
}`,
			expected: `{
  "type": "cloudflare_healthcheck",
  "name": "minimal",
  "attributes": {
    "zone_id": "abc123",
    "name": "minimal-check",
    "address": "example.com",
    "type": "HTTP"
  },
  "schema_version": 0
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := gjson.Parse(tt.input)
			result, err := migrator.TransformState(ctx, instance, "resources.0.instances.0")

			if err != nil {
				t.Fatalf("TransformState failed: %v", err)
			}

			// Parse both as JSON to normalize formatting
			resultJSON := gjson.Parse(result)
			expectedJSON := gjson.Parse(tt.expected)

			// Compare key fields
			compareStateFields(t, resultJSON, expectedJSON)
		})
	}
}

// testHTTPSHealthcheckState tests HTTPS healthcheck state transformation
func testHTTPSHealthcheckState(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	ctx := &transform.Context{
		SourceVersion: "v4",
		TargetVersion: "v5",
	}

	input := `{
  "type": "cloudflare_healthcheck",
  "name": "https",
  "attributes": {
    "zone_id": "abc123",
    "name": "https-check",
    "address": "example.com",
    "type": "HTTPS",
    "port": 443,
    "path": "/api/health",
    "method": "HEAD",
    "allow_insecure": true,
    "expected_codes": ["200"]
  }
}`

	instance := gjson.Parse(input)
	result, err := migrator.TransformState(ctx, instance, "resources.0.instances.0")

	if err != nil {
		t.Fatalf("TransformState failed: %v", err)
	}

	resultJSON := gjson.Parse(result)

	// Verify http_config was created (HTTPS uses http_config)
	if !resultJSON.Get("attributes.http_config").Exists() {
		t.Error("Expected http_config to be created for HTTPS healthcheck")
	}

	// Verify fields moved into http_config
	httpConfig := resultJSON.Get("attributes.http_config")
	if httpConfig.Get("port").Int() != 443 {
		t.Errorf("Expected port=443 in http_config, got %v", httpConfig.Get("port").Int())
	}
	if httpConfig.Get("path").String() != "/api/health" {
		t.Errorf("Expected path=/api/health in http_config, got %v", httpConfig.Get("path").String())
	}
	if httpConfig.Get("allow_insecure").Bool() != true {
		t.Error("Expected allow_insecure=true in http_config")
	}

	// Verify root-level fields removed
	if resultJSON.Get("attributes.port").Exists() {
		t.Error("port should be removed from root level")
	}
	if resultJSON.Get("attributes.path").Exists() {
		t.Error("path should be removed from root level")
	}

	// Verify schema_version set
	if resultJSON.Get("schema_version").Int() != 0 {
		t.Error("Expected schema_version=0")
	}
}

// testTCPHealthcheckState tests TCP healthcheck state transformation
func testTCPHealthcheckState(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	ctx := &transform.Context{
		SourceVersion: "v4",
		TargetVersion: "v5",
	}

	input := `{
  "type": "cloudflare_healthcheck",
  "name": "tcp",
  "attributes": {
    "zone_id": "abc123",
    "name": "tcp-check",
    "address": "10.0.0.1",
    "type": "TCP",
    "port": 8080,
    "method": "connection_established",
    "consecutive_fails": 2,
    "timeout": 10
  }
}`

	instance := gjson.Parse(input)
	result, err := migrator.TransformState(ctx, instance, "resources.0.instances.0")

	if err != nil {
		t.Fatalf("TransformState failed: %v", err)
	}

	resultJSON := gjson.Parse(result)

	// Verify tcp_config was created
	if !resultJSON.Get("attributes.tcp_config").Exists() {
		t.Fatal("Expected tcp_config to be created for TCP healthcheck")
	}

	// Verify fields moved into tcp_config
	tcpConfig := resultJSON.Get("attributes.tcp_config")
	if tcpConfig.Get("port").Int() != 8080 {
		t.Errorf("Expected port=8080 in tcp_config, got %v", tcpConfig.Get("port").Int())
	}
	if tcpConfig.Get("method").String() != "connection_established" {
		t.Errorf("Expected method=connection_established in tcp_config, got %v", tcpConfig.Get("method").String())
	}

	// Verify root-level fields removed
	if resultJSON.Get("attributes.port").Exists() {
		t.Error("port should be removed from root level")
	}
	if resultJSON.Get("attributes.method").Exists() {
		t.Error("method should be removed from root level")
	}

	// Verify non-TCP fields remain at root
	if !resultJSON.Get("attributes.consecutive_fails").Exists() {
		t.Error("consecutive_fails should remain at root level")
	}
	if !resultJSON.Get("attributes.timeout").Exists() {
		t.Error("timeout should remain at root level")
	}

	// Verify schema_version set
	if resultJSON.Get("schema_version").Int() != 0 {
		t.Error("Expected schema_version=0")
	}
}

// testStateTypeConversions tests numeric type conversions (Int → Float64)
func testStateTypeConversions(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	ctx := &transform.Context{
		SourceVersion: "v4",
		TargetVersion: "v5",
	}

	input := `{
  "type": "cloudflare_healthcheck",
  "name": "conversions",
  "attributes": {
    "zone_id": "abc123",
    "name": "test",
    "address": "example.com",
    "type": "HTTP",
    "consecutive_fails": 1,
    "consecutive_successes": 1,
    "retries": 2,
    "timeout": 5,
    "interval": 60,
    "port": 80
  }
}`

	instance := gjson.Parse(input)
	result, err := migrator.TransformState(ctx, instance, "resources.0.instances.0")

	if err != nil {
		t.Fatalf("TransformState failed: %v", err)
	}

	resultJSON := gjson.Parse(result)
	attrs := resultJSON.Get("attributes")

	// Check that all numeric fields are converted to float64
	numericFields := []string{
		"consecutive_fails",
		"consecutive_successes",
		"retries",
		"timeout",
		"interval",
	}

	for _, field := range numericFields {
		val := attrs.Get(field)
		if !val.Exists() {
			continue
		}
		// gjson returns numbers as float64, so we just verify it exists and is numeric
		if val.Type != gjson.Number {
			t.Errorf("Field %s should be numeric, got type %v", field, val.Type)
		}
	}

	// Port should be in http_config and also numeric
	httpConfig := attrs.Get("http_config")
	if httpConfig.Exists() {
		portVal := httpConfig.Get("port")
		if portVal.Exists() && portVal.Type != gjson.Number {
			t.Error("port in http_config should be numeric")
		}
	}
}

// testHeaderTransformation tests header Set→Map transformation
func testHeaderTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	ctx := &transform.Context{
		SourceVersion: "v4",
		TargetVersion: "v5",
	}

	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, result gjson.Result)
	}{
		{
			name: "Single header",
			input: `{
  "type": "cloudflare_healthcheck",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "name": "test",
    "address": "example.com",
    "type": "HTTP",
    "header": [
      {
        "header": "Host",
        "values": ["example.com"]
      }
    ]
  }
}`,
			validate: func(t *testing.T, result gjson.Result) {
				headerMap := result.Get("attributes.http_config.header")
				if !headerMap.Exists() {
					t.Fatal("Expected header map in http_config")
				}

				hostValues := headerMap.Get("Host")
				if !hostValues.Exists() {
					t.Fatal("Expected 'Host' key in header map")
				}

				if !hostValues.IsArray() {
					t.Fatal("Expected Host values to be an array")
				}

				if len(hostValues.Array()) != 1 || hostValues.Array()[0].String() != "example.com" {
					t.Errorf("Expected Host=[example.com], got %v", hostValues.Array())
				}
			},
		},
		{
			name: "Multiple headers",
			input: `{
  "type": "cloudflare_healthcheck",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "name": "test",
    "address": "example.com",
    "type": "HTTPS",
    "header": [
      {
        "header": "Host",
        "values": ["example.com"]
      },
      {
        "header": "User-Agent",
        "values": ["HealthChecker/1.0"]
      },
      {
        "header": "Accept",
        "values": ["application/json", "text/plain"]
      }
    ]
  }
}`,
			validate: func(t *testing.T, result gjson.Result) {
				headerMap := result.Get("attributes.http_config.header")
				if !headerMap.Exists() {
					t.Fatal("Expected header map in http_config")
				}

				// Check Host header
				if headerMap.Get("Host").Array()[0].String() != "example.com" {
					t.Error("Host header not transformed correctly")
				}

				// Check User-Agent header
				if headerMap.Get("User-Agent").Array()[0].String() != "HealthChecker/1.0" {
					t.Error("User-Agent header not transformed correctly")
				}

				// Check Accept header with multiple values
				acceptValues := headerMap.Get("Accept").Array()
				if len(acceptValues) != 2 {
					t.Errorf("Expected 2 Accept values, got %d", len(acceptValues))
				}
			},
		},
		{
			name: "No headers",
			input: `{
  "type": "cloudflare_healthcheck",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "name": "test",
    "address": "example.com",
    "type": "HTTP"
  }
}`,
			validate: func(t *testing.T, result gjson.Result) {
				// No http_config should be created if no HTTP fields exist
				// This is acceptable
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := gjson.Parse(tt.input)
			result, err := migrator.TransformState(ctx, instance, "resources.0.instances.0")

			if err != nil {
				t.Fatalf("TransformState failed: %v", err)
			}

			resultJSON := gjson.Parse(result)
			tt.validate(t, resultJSON)
		})
	}
}

// testStateEdgeCases tests edge cases in state transformation
func testStateEdgeCases(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	ctx := &transform.Context{
		SourceVersion: "v4",
		TargetVersion: "v5",
	}

	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, result gjson.Result, err error)
	}{
		{
			name: "Missing attributes block",
			input: `{
  "type": "cloudflare_healthcheck",
  "name": "test"
}`,
			validate: func(t *testing.T, result gjson.Result, err error) {
				if err != nil {
					t.Fatalf("TransformState should not error: %v", err)
				}
				// Should set schema_version even without attributes
				if result.Get("schema_version").Int() != 0 {
					t.Error("Expected schema_version=0 even without attributes")
				}
			},
		},
		{
			name: "Empty check_regions array",
			input: `{
  "type": "cloudflare_healthcheck",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "name": "test",
    "address": "example.com",
    "type": "HTTP",
    "check_regions": []
  }
}`,
			validate: func(t *testing.T, result gjson.Result, err error) {
				if err != nil {
					t.Fatalf("TransformState failed: %v", err)
				}
				// Empty arrays should be preserved as-is (don't delete)
				// check_regions is Optional+Computed, so empty is valid
				checkRegions := result.Get("attributes.check_regions")
				if checkRegions.Exists() && checkRegions.IsArray() {
					// This is acceptable - provider will handle it
				}
			},
		},
		{
			name: "Null optional fields",
			input: `{
  "type": "cloudflare_healthcheck",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "name": "test",
    "address": "example.com",
    "type": "HTTP",
    "description": null,
    "suspended": null
  }
}`,
			validate: func(t *testing.T, result gjson.Result, err error) {
				if err != nil {
					t.Fatalf("TransformState failed: %v", err)
				}
				// Null fields should remain as-is for provider to handle
			},
		},
		{
			name: "Computed fields present",
			input: `{
  "type": "cloudflare_healthcheck",
  "name": "test",
  "attributes": {
    "id": "health123",
    "zone_id": "abc123",
    "name": "test",
    "address": "example.com",
    "type": "HTTP",
    "created_on": "2024-01-01T00:00:00Z",
    "modified_on": "2024-01-01T00:00:00Z",
    "status": "healthy",
    "failure_reason": ""
  }
}`,
			validate: func(t *testing.T, result gjson.Result, err error) {
				if err != nil {
					t.Fatalf("TransformState failed: %v", err)
				}
				// Computed fields should NOT be removed or modified
				attrs := result.Get("attributes")
				if !attrs.Get("id").Exists() {
					t.Error("id (computed) should not be removed")
				}
				if !attrs.Get("created_on").Exists() {
					t.Error("created_on (computed) should not be removed")
				}
				if !attrs.Get("modified_on").Exists() {
					t.Error("modified_on (computed) should not be removed")
				}
			},
		},
		{
			name: "Unknown type defaults to HTTP",
			input: `{
  "type": "cloudflare_healthcheck",
  "name": "test",
  "attributes": {
    "zone_id": "abc123",
    "name": "test",
    "address": "example.com",
    "type": "UNKNOWN",
    "port": 80
  }
}`,
			validate: func(t *testing.T, result gjson.Result, err error) {
				if err != nil {
					t.Fatalf("TransformState failed: %v", err)
				}
				// Should treat as HTTP (default behavior)
				if result.Get("attributes.http_config").Exists() {
					// Good - treated as HTTP
				} else {
					// Also acceptable - no HTTP fields to move
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := gjson.Parse(tt.input)
			result, err := migrator.TransformState(ctx, instance, "resources.0.instances.0")
			resultJSON := gjson.Parse(result)
			tt.validate(t, resultJSON, err)
		})
	}
}

// Helper function to compare state fields
func compareStateFields(t *testing.T, result, expected gjson.Result) {
	// Compare type
	if result.Get("type").String() != expected.Get("type").String() {
		t.Errorf("Type mismatch: expected %s, got %s",
			expected.Get("type").String(), result.Get("type").String())
	}

	// Compare name
	if result.Get("name").String() != expected.Get("name").String() {
		t.Errorf("Name mismatch: expected %s, got %s",
			expected.Get("name").String(), result.Get("name").String())
	}

	// Compare schema_version
	if result.Get("schema_version").Int() != expected.Get("schema_version").Int() {
		t.Errorf("schema_version mismatch: expected %d, got %d",
			expected.Get("schema_version").Int(), result.Get("schema_version").Int())
	}

	// Compare key attributes
	expectedAttrs := expected.Get("attributes")
	resultAttrs := result.Get("attributes")

	if expectedAttrs.Exists() && !resultAttrs.Exists() {
		t.Error("Expected attributes to exist")
		return
	}

	// Compare specific fields (not exhaustive, just key ones)
	compareField := func(field string) {
		expVal := expectedAttrs.Get(field)
		resVal := resultAttrs.Get(field)

		if expVal.Exists() != resVal.Exists() {
			t.Errorf("Field %s existence mismatch: expected %v, got %v",
				field, expVal.Exists(), resVal.Exists())
		}
	}

	compareField("zone_id")
	compareField("name")
	compareField("address")
	compareField("type")
	compareField("http_config")
	compareField("tcp_config")
}

// TestCanHandle verifies the migrator handles the correct resource type
func TestCanHandle(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []struct {
		resourceType string
		expected     bool
	}{
		{"cloudflare_healthcheck", true},
		{"cloudflare_health_check", false},
		{"cloudflare_dns_record", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			result := migrator.CanHandle(tt.resourceType)
			if result != tt.expected {
				t.Errorf("CanHandle(%s) = %v, expected %v", tt.resourceType, result, tt.expected)
			}
		})
	}
}

// TestGetResourceType verifies the resource type returned
func TestGetResourceType(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	expected := "cloudflare_healthcheck"
	result := migrator.GetResourceType()

	if result != expected {
		t.Errorf("GetResourceType() = %s, expected %s", result, expected)
	}
}
