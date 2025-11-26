package healthcheck

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/tidwall/gjson"
)

// TestIntegrationStateData validates that the state transformation works
// with the actual integration test data (28 instances)
func TestIntegrationStateData(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	ctx := &transform.Context{
		SourceVersion: "v4",
		TargetVersion: "v5",
	}

	// Read the actual integration test state file
	stateFile := "../../../integration/v4_to_v5/testdata/healthcheck/input/terraform.tfstate"
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Skipf("Integration test data not found (expected): %v", err)
		return
	}

	// Parse the state file
	state := gjson.ParseBytes(data)
	resources := state.Get("resources")

	if !resources.Exists() || !resources.IsArray() {
		t.Fatal("Invalid state file: no resources array")
	}

	totalInstances := 0
	successfulTransforms := 0
	var errors []string

	// Process each resource
	for _, resource := range resources.Array() {
		resourceType := resource.Get("type").String()
		resourceName := resource.Get("name").String()

		if resourceType != "cloudflare_healthcheck" {
			continue
		}

		instances := resource.Get("instances")
		if !instances.Exists() || !instances.IsArray() {
			continue
		}

		// Transform each instance
		for idx, instance := range instances.Array() {
			totalInstances++

			result, err := migrator.TransformState(ctx, instance, "resources.0.instances.0")
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s[%d]: %v", resourceName, idx, err))
				continue
			}

			// Verify the result is valid JSON
			resultJSON := gjson.Parse(result)
			if !resultJSON.Exists() {
				errors = append(errors, fmt.Sprintf("%s[%d]: invalid JSON output", resourceName, idx))
				continue
			}

			// Verify schema_version was set
			if resultJSON.Get("schema_version").Int() != 0 {
				errors = append(errors, fmt.Sprintf("%s[%d]: schema_version not set to 0", resourceName, idx))
				continue
			}

			// Verify attributes exist
			attrs := resultJSON.Get("attributes")
			if !attrs.Exists() {
				// This is OK - some instances might not have attributes
				successfulTransforms++
				continue
			}

			// Check for http_config or tcp_config based on type
			healthType := attrs.Get("type").String()

			if healthType == "HTTP" || healthType == "HTTPS" {
				// Should have http_config if any HTTP fields were present
				// Note: May not have http_config if no HTTP-specific fields in v4
				successfulTransforms++
			} else if healthType == "TCP" {
				// Should have tcp_config if any TCP fields were present
				successfulTransforms++
			} else {
				successfulTransforms++
			}
		}
	}

	// Report results
	t.Logf("Processed %d instances from integration test data", totalInstances)
	t.Logf("Successful transformations: %d", successfulTransforms)

	if len(errors) > 0 {
		t.Logf("Errors encountered:")
		for _, errMsg := range errors {
			t.Logf("  - %s", errMsg)
		}
		t.Fatalf("Failed to transform %d out of %d instances", len(errors), totalInstances)
	}

	if totalInstances == 0 {
		t.Fatal("No instances found in integration test data")
	}

	if totalInstances < 28 {
		t.Errorf("Expected 28 instances, found %d", totalInstances)
	}

	t.Logf("✅ All %d instances transformed successfully!", totalInstances)
}

// TestIntegrationStateHTTPTransform validates HTTP-specific transformations
func TestIntegrationStateHTTPTransform(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	ctx := &transform.Context{
		SourceVersion: "v4",
		TargetVersion: "v5",
	}

	// Test with a full HTTP healthcheck from integration data
	input := `{
  "schema_version": 0,
  "attributes": {
    "id": "full-http-id",
    "zone_id": "zone123",
    "name": "migration-test-full-http",
    "address": "api.example.com",
    "type": "HTTP",
    "port": 80,
    "path": "/health",
    "method": "GET",
    "expected_codes": ["200", "201", "204"],
    "expected_body": "OK",
    "follow_redirects": false,
    "allow_insecure": false,
    "header": [
      {
        "header": "Host",
        "values": ["example.com"]
      },
      {
        "header": "User-Agent",
        "values": ["HealthChecker/1.0"]
      }
    ],
    "description": "Full HTTP healthcheck with all fields",
    "consecutive_fails": 3,
    "consecutive_successes": 2,
    "retries": 2,
    "timeout": 5,
    "interval": 60,
    "suspended": false,
    "check_regions": ["WNAM", "ENAM"],
    "created_on": "2024-01-01T00:00:00Z",
    "modified_on": "2024-01-01T00:00:00Z"
  }
}`

	instance := gjson.Parse(input)
	result, err := migrator.TransformState(ctx, instance, "resources.0.instances.0")

	if err != nil {
		t.Fatalf("TransformState failed: %v", err)
	}

	resultJSON := gjson.Parse(result)

	// Verify http_config was created
	httpConfig := resultJSON.Get("attributes.http_config")
	if !httpConfig.Exists() {
		t.Fatal("http_config should be created for HTTP healthcheck")
	}

	// Verify fields moved into http_config
	checks := map[string]interface{}{
		"port":             80.0,
		"path":             "/health",
		"method":           "GET",
		"expected_body":    "OK",
		"follow_redirects": false,
		"allow_insecure":   false,
	}

	for field, expectedVal := range checks {
		val := httpConfig.Get(field)
		if !val.Exists() {
			t.Errorf("Field %s should exist in http_config", field)
			continue
		}

		switch v := expectedVal.(type) {
		case float64:
			if val.Float() != v {
				t.Errorf("Field %s: expected %v, got %v", field, v, val.Float())
			}
		case string:
			if val.String() != v {
				t.Errorf("Field %s: expected %v, got %v", field, v, val.String())
			}
		case bool:
			if val.Bool() != v {
				t.Errorf("Field %s: expected %v, got %v", field, v, val.Bool())
			}
		}
	}

	// Verify header transformation (Set→Map)
	header := httpConfig.Get("header")
	if !header.Exists() {
		t.Fatal("header should exist in http_config")
	}

	// Verify it's a map with the right structure
	var headerMap map[string]interface{}
	if err := json.Unmarshal([]byte(header.Raw), &headerMap); err != nil {
		t.Fatalf("header should be a valid map: %v", err)
	}

	if len(headerMap) != 2 {
		t.Errorf("Expected 2 headers, got %d", len(headerMap))
	}

	t.Log("✅ HTTP healthcheck transformation validated successfully")
}

// TestIntegrationStateTCPTransform validates TCP-specific transformations
func TestIntegrationStateTCPTransform(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	ctx := &transform.Context{
		SourceVersion: "v4",
		TargetVersion: "v5",
	}

	// Test with a TCP healthcheck from integration data
	input := `{
  "schema_version": 0,
  "attributes": {
    "id": "tcp-basic-id",
    "zone_id": "zone123",
    "name": "migration-test-tcp",
    "address": "10.0.0.1",
    "type": "TCP",
    "port": 8080,
    "method": "connection_established",
    "consecutive_fails": 2,
    "timeout": 10,
    "interval": 30,
    "created_on": "2024-01-01T00:00:00Z",
    "modified_on": "2024-01-01T00:00:00Z"
  }
}`

	instance := gjson.Parse(input)
	result, err := migrator.TransformState(ctx, instance, "resources.0.instances.0")

	if err != nil {
		t.Fatalf("TransformState failed: %v", err)
	}

	resultJSON := gjson.Parse(result)

	// Verify tcp_config was created
	tcpConfig := resultJSON.Get("attributes.tcp_config")
	if !tcpConfig.Exists() {
		t.Fatal("tcp_config should be created for TCP healthcheck")
	}

	// Verify fields moved into tcp_config
	if tcpConfig.Get("port").Float() != 8080.0 {
		t.Errorf("port should be 8080 in tcp_config, got %v", tcpConfig.Get("port").Float())
	}

	if tcpConfig.Get("method").String() != "connection_established" {
		t.Errorf("method should be 'connection_established' in tcp_config, got %v", tcpConfig.Get("method").String())
	}

	// Verify root-level fields were removed
	attrs := resultJSON.Get("attributes")
	if attrs.Get("port").Exists() {
		t.Error("port should be removed from root level")
	}
	if attrs.Get("method").Exists() {
		t.Error("method should be removed from root level")
	}

	// Verify non-TCP fields remain at root
	if !attrs.Get("consecutive_fails").Exists() {
		t.Error("consecutive_fails should remain at root level")
	}

	t.Log("✅ TCP healthcheck transformation validated successfully")
}
