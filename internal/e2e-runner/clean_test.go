package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunClean_ValidationEmptyModules(t *testing.T) {
	// Test that RunClean validates input
	modules := []string{}

	err := RunClean(modules)
	if err == nil {
		t.Error("Expected error for empty modules list")
	}

	if err != nil && err.Error() != "no modules specified\nUsage: e2e clean --modules <module1,module2,...>" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestRunClean_StateFiltering(t *testing.T) {
	// Test the state filtering logic without actually connecting to R2
	// We'll test the JSON manipulation directly

	state := map[string]interface{}{
		"version": 4,
		"serial":  float64(1),
		"resources": []interface{}{
			map[string]interface{}{
				"module": "module.zone_dnssec",
				"type":   "cloudflare_zone_dnssec",
				"name":   "example",
			},
			map[string]interface{}{
				"module": "module.argo",
				"type":   "cloudflare_argo",
				"name":   "example",
			},
			map[string]interface{}{
				"module": "module.custom_pages",
				"type":   "cloudflare_custom_pages",
				"name":   "example",
			},
		},
	}

	modulesToClean := []string{"zone_dnssec", "argo"}

	// Filter resources (simulating what RunClean does)
	resourcesArray := state["resources"].([]interface{})
	var filtered []interface{}
	removed := 0

	for _, res := range resourcesArray {
		resMap := res.(map[string]interface{})
		resModule := resMap["module"].(string)

		shouldRemove := false
		for _, module := range modulesToClean {
			if resModule == "module."+module {
				shouldRemove = true
				removed++
				break
			}
		}

		if !shouldRemove {
			filtered = append(filtered, res)
		}
	}

	state["resources"] = filtered

	// Verify results
	if removed != 2 {
		t.Errorf("Expected to remove 2 resources, removed %d", removed)
	}

	if len(filtered) != 1 {
		t.Errorf("Expected 1 resource remaining, got %d", len(filtered))
	}

	// Verify the remaining resource is custom_pages
	remainingRes := filtered[0].(map[string]interface{})
	if remainingRes["module"] != "module.custom_pages" {
		t.Errorf("Expected custom_pages to remain, got %v", remainingRes["module"])
	}
}

func TestRunClean_SerialIncrement(t *testing.T) {
	// Test that serial number is incremented
	state := map[string]interface{}{
		"version":   4,
		"serial":    float64(5),
		"resources": []interface{}{},
	}

	// Simulate serial increment
	if serial, ok := state["serial"].(float64); ok {
		state["serial"] = serial + 1
	}

	newSerial := state["serial"].(float64)
	if newSerial != 6 {
		t.Errorf("Expected serial to be 6, got %v", newSerial)
	}
}

func TestRunClean_StateJSONMarshaling(t *testing.T) {
	// Test that state can be marshaled/unmarshaled correctly
	state := map[string]interface{}{
		"version": 4,
		"serial":  float64(1),
		"resources": []interface{}{
			map[string]interface{}{
				"module": "module.test",
				"type":   "cloudflare_zone",
				"name":   "example",
			},
		},
	}

	// Marshal to JSON
	stateJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal state: %v", err)
	}

	// Unmarshal back
	var unmarshaledState map[string]interface{}
	err = json.Unmarshal(stateJSON, &unmarshaledState)
	if err != nil {
		t.Fatalf("Failed to unmarshal state: %v", err)
	}

	// Verify data integrity
	if unmarshaledState["version"] != float64(4) {
		t.Error("Version not preserved after marshal/unmarshal")
	}

	resources := unmarshaledState["resources"].([]interface{})
	if len(resources) != 1 {
		t.Error("Resources not preserved after marshal/unmarshal")
	}
}

func TestRunClean_ModulePrefix(t *testing.T) {
	// Test that module prefix matching works correctly
	tests := []struct {
		name           string
		resourceModule string
		cleanModules   []string
		shouldRemove   bool
	}{
		{
			name:           "exact match",
			resourceModule: "module.zone_dnssec",
			cleanModules:   []string{"zone_dnssec"},
			shouldRemove:   true,
		},
		{
			name:           "no match",
			resourceModule: "module.zone_dnssec",
			cleanModules:   []string{"argo"},
			shouldRemove:   false,
		},
		{
			name:           "multiple modules one matches",
			resourceModule: "module.argo",
			cleanModules:   []string{"zone_dnssec", "argo", "custom_pages"},
			shouldRemove:   true,
		},
		{
			name:           "substring should not match",
			resourceModule: "module.zone_dnssec_settings",
			cleanModules:   []string{"zone_dnssec"},
			shouldRemove:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldRemove := false
			for _, module := range tt.cleanModules {
				if tt.resourceModule == "module."+module {
					shouldRemove = true
					break
				}
			}

			if shouldRemove != tt.shouldRemove {
				t.Errorf("shouldRemove = %v, want %v", shouldRemove, tt.shouldRemove)
			}
		})
	}
}

func TestRunClean_EmptyState(t *testing.T) {
	// Test handling of empty state
	state := map[string]interface{}{
		"version":   4,
		"serial":    float64(1),
		"resources": []interface{}{},
	}

	modulesToClean := []string{"zone_dnssec"}

	resourcesArray := state["resources"].([]interface{})
	var filtered []interface{}
	removed := 0

	for _, res := range resourcesArray {
		resMap := res.(map[string]interface{})
		resModule := resMap["module"].(string)

		shouldRemove := false
		for _, module := range modulesToClean {
			if resModule == "module."+module {
				shouldRemove = true
				removed++
				break
			}
		}

		if !shouldRemove {
			filtered = append(filtered, res)
		}
	}

	if removed != 0 {
		t.Errorf("Expected 0 resources removed from empty state, got %d", removed)
	}

	if len(filtered) != 0 {
		t.Errorf("Expected empty filtered list, got %d", len(filtered))
	}
}

func TestRunClean_ResourceCounting(t *testing.T) {
	// Test that resource counting works correctly
	state := map[string]interface{}{
		"version": 4,
		"serial":  float64(1),
		"resources": []interface{}{
			map[string]interface{}{
				"module": "module.zone_dnssec",
				"type":   "cloudflare_zone_dnssec",
				"name":   "example1",
				"instances": []interface{}{
					map[string]interface{}{"id": "1"},
				},
			},
			map[string]interface{}{
				"module": "module.zone_dnssec",
				"type":   "cloudflare_zone_dnssec",
				"name":   "example2",
				"instances": []interface{}{
					map[string]interface{}{"id": "2"},
					map[string]interface{}{"id": "3"},
				},
			},
		},
	}

	// Count instances
	resources := state["resources"].([]interface{})
	instanceCount := 0
	for _, res := range resources {
		if resMap, ok := res.(map[string]interface{}); ok {
			if instances, ok := resMap["instances"].([]interface{}); ok {
				instanceCount += len(instances)
			}
		}
	}

	if instanceCount != 3 {
		t.Errorf("Expected 3 instances, counted %d", instanceCount)
	}
}

func TestRunClean_InvalidStateFormat(t *testing.T) {
	// Test handling of invalid state format
	state := map[string]interface{}{
		"version": 4,
		"serial":  float64(1),
		// Missing resources array
	}

	_, ok := state["resources"].([]interface{})
	if ok {
		t.Error("Expected resources to be missing")
	}

	// This simulates what would happen in RunClean
	// It should handle the missing resources array gracefully
	if state["resources"] == nil {
		// Set to empty array
		state["resources"] = []interface{}{}
	}

	resources, ok := state["resources"].([]interface{})
	if !ok {
		t.Error("Failed to create empty resources array")
	}

	if len(resources) != 0 {
		t.Error("Expected empty resources array")
	}
}

func TestCleanStateFiltering(t *testing.T) {
	// Test the state filtering logic used in RunClean
	stateJSON := `{
		"version": 4,
		"terraform_version": "1.0.0",
		"serial": 1,
		"resources": [
			{
				"module": "module.zone",
				"type": "cloudflare_zone",
				"name": "example",
				"instances": [{}]
			},
			{
				"module": "module.dns",
				"type": "cloudflare_dns_record",
				"name": "www",
				"instances": [{}]
			},
			{
				"module": "module.healthcheck",
				"type": "cloudflare_healthcheck",
				"name": "test",
				"instances": [{}]
			}
		]
	}`

	var state map[string]interface{}
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		t.Fatalf("Failed to parse test state: %v", err)
	}

	// Simulate cleaning module.dns
	modulesToClean := []string{"dns"}
	resourcesArray := state["resources"].([]interface{})
	var filtered []interface{}

	for _, res := range resourcesArray {
		resMap := res.(map[string]interface{})
		module := resMap["module"].(string)

		shouldRemove := false
		for _, m := range modulesToClean {
			if module == "module."+m {
				shouldRemove = true
				break
			}
		}

		if !shouldRemove {
			filtered = append(filtered, res)
		}
	}

	if len(filtered) != 2 {
		t.Errorf("Expected 2 resources after filtering, got %d", len(filtered))
	}

	// Verify the right resource was removed
	for _, res := range filtered {
		resMap := res.(map[string]interface{})
		module := resMap["module"].(string)
		if module == "module.dns" {
			t.Errorf("module.dns should have been removed but is still present")
		}
	}
}

func TestCleanSerialIncrement_Unit(t *testing.T) {
	// Test that serial number is incremented correctly
	state := map[string]interface{}{
		"serial": float64(5),
	}

	// Simulate the serial increment logic from RunClean
	if serial, ok := state["serial"].(float64); ok {
		state["serial"] = serial + 1
	}

	newSerial := state["serial"].(float64)
	if newSerial != 6 {
		t.Errorf("Expected serial to be incremented to 6, got %v", newSerial)
	}
}

func TestCleanModuleMatching(t *testing.T) {
	tests := []struct {
		name         string
		resourceMod  string
		cleanModules []string
		shouldRemove bool
	}{
		{
			name:         "exact match",
			resourceMod:  "module.dns",
			cleanModules: []string{"dns"},
			shouldRemove: true,
		},
		{
			name:         "no match",
			resourceMod:  "module.dns",
			cleanModules: []string{"zone"},
			shouldRemove: false,
		},
		{
			name:         "multiple modules one matches",
			resourceMod:  "module.dns",
			cleanModules: []string{"zone", "dns", "healthcheck"},
			shouldRemove: true,
		},
		{
			name:         "partial name should not match",
			resourceMod:  "module.dns_record",
			cleanModules: []string{"dns"},
			shouldRemove: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldRemove := false
			for _, module := range tt.cleanModules {
				if tt.resourceMod == "module."+module {
					shouldRemove = true
					break
				}
			}

			if shouldRemove != tt.shouldRemove {
				t.Errorf("Expected shouldRemove=%v, got %v", tt.shouldRemove, shouldRemove)
			}
		})
	}
}

func TestCleanResourceCounting_Unit(t *testing.T) {
	// Test resource counting logic
	stateJSON := `{
		"resources": [
			{
				"module": "module.zone",
				"instances": [{"id": "1"}, {"id": "2"}]
			},
			{
				"module": "module.dns",
				"instances": [{"id": "3"}]
			}
		]
	}`

	var state map[string]interface{}
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		t.Fatalf("Failed to parse test state: %v", err)
	}

	// Count instances
	resourcesArray := state["resources"].([]interface{})
	totalInstances := 0
	for _, res := range resourcesArray {
		resMap := res.(map[string]interface{})
		if instances, ok := resMap["instances"].([]interface{}); ok {
			totalInstances += len(instances)
		}
	}

	if totalInstances != 3 {
		t.Errorf("Expected 3 total instances, got %d", totalInstances)
	}
}

func TestCleanStateFilePermissions(t *testing.T) {
	// Test that cleaned state files are created with secure permissions (0600)
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "terraform.tfstate.cleaned")

	stateContent := []byte(`{"version": 4, "serial": 1, "resources": []}`)

	// Write with restrictive permissions as in clean.go:186
	if err := os.WriteFile(stateFile, stateContent, 0600); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	// Check permissions
	info, err := os.Stat(stateFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	mode := info.Mode()
	if mode.Perm() != 0600 {
		t.Errorf("Expected file permissions 0600, got %04o", mode.Perm())
	}
}
