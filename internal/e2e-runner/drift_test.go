package e2e

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestHasOnlyComputedChangesDefault(t *testing.T) {
	tests := []struct {
		name       string
		planOutput string
		want       bool
	}{
		{
			name: "only known after apply changes",
			planOutput: `
Terraform will perform the following actions:

  # cloudflare_zone.example will be updated in-place
  ~ resource "cloudflare_zone" "example" {
      ~ status = (known after apply)
    }

Plan: 0 to add, 1 to change, 0 to destroy.
`,
			want: true,
		},
		{
			name: "real value change with updates only",
			planOutput: `
Terraform will perform the following actions:

  # cloudflare_zone_dnssec.example will be updated in-place
  ~ resource "cloudflare_zone_dnssec" "example" {
      ~ algorithm = "13" -> "14"
    }

Plan: 0 to add, 1 to change, 0 to destroy.
`,
			want: false, // Real change detected (arrow pattern)
		},
		{
			name: "status to active change (allowed)",
			planOutput: `
Terraform will perform the following actions:

  # cloudflare_zone_dnssec.example will be updated in-place
  ~ resource "cloudflare_zone_dnssec" "example" {
      ~ status = "pending" -> "active"
    }

Plan: 0 to add, 1 to change, 0 to destroy.
`,
			want: true,
		},
		{
			name: "addition detected",
			planOutput: `
Terraform will perform the following actions:

  # cloudflare_zone.new will be created
  + resource "cloudflare_zone" "new" {
      + zone = "example.com"
    }

Plan: 1 to add, 0 to change, 0 to destroy.
`,
			want: false,
		},
		{
			name: "deletion detected",
			planOutput: `
Terraform will perform the following actions:

  # cloudflare_zone.old will be destroyed
  - resource "cloudflare_zone" "old" {
      - zone = "example.com"
    }

Plan: 0 to add, 0 to change, 1 to destroy.
`,
			want: false,
		},
		{
			name: "only updates (0 add, X change, 0 to destroy)",
			planOutput: `
Terraform will perform the following actions:

  # cloudflare_zone.example will be updated in-place
  ~ resource "cloudflare_zone" "example" {
      ~ some_field = "old" -> "new"
    }

Plan: 0 to add, 5 to change, 0 to destroy.
`,
			want: false, // Real change detected
		},
		{
			name: "no changes",
			planOutput: `
No changes. Your infrastructure matches the configuration.
`,
			want: true,
		},
		{
			name: "precedence change - real drift",
			planOutput: `
Terraform will perform the following actions:

  # module.zero_trust_gateway_policy.cloudflare_zero_trust_gateway_policy.with_settings will be updated in-place
  ~ resource "cloudflare_zero_trust_gateway_policy" "with_settings" {
      ~ created_at     = "2025-12-22T00:47:31Z" -> (known after apply)
      + deleted_at     = (known after apply)
      + expiration     = (known after apply)
        id             = "67cdcc4b-a96a-4dd4-b310-81e8148469c5"
        name           = "cftftest Block Policy with Settings"
      ~ precedence     = 200895 -> 200
    }

Plan: 0 to add, 1 to change, 0 to destroy.
`,
			want: false, // precedence change is real drift
		},
		{
			name: "nested block attribute deletion - real drift",
			planOutput: `
Terraform will perform the following actions:

  # module.zero_trust_gateway_policy.cloudflare_zero_trust_gateway_policy.complex will be updated in-place
  ~ resource "cloudflare_zero_trust_gateway_policy" "complex" {
      ~ created_at     = "2025-12-22T00:47:36Z" -> (known after apply)
        id             = "937e6ef7-6db0-4f04-8e0b-71cf6e4698f6"
        name           = "cftftest Complex Policy"
      ~ precedence     = 400412 -> 400
      ~ rule_settings  = {
          ~ biso_admin_controls = {
              - dp = true -> null
            }
        }
    }

Plan: 0 to add, 1 to change, 0 to destroy.
`,
			want: false, // dp deletion is real drift
		},
		{
			name: "array element deletion - real drift",
			planOutput: `
Terraform will perform the following actions:

  # module.zero_trust_access_identity_provider.cloudflare_zero_trust_access_identity_provider.saml will be updated in-place
  ~ resource "cloudflare_zero_trust_access_identity_provider" "saml" {
      ~ config      = {
          - attributes = [
              - "email",
            ]
        }
    }

Plan: 0 to add, 1 to change, 0 to destroy.
`,
			want: false, // attributes deletion is real drift
		},
		{
			name: "multiple real changes with computed changes mixed",
			planOutput: `
Terraform will perform the following actions:

  # cloudflare_resource.example will be updated in-place
  ~ resource "cloudflare_resource" "example" {
      ~ created_at     = "2025-12-22T00:47:31Z" -> (known after apply)
      ~ precedence     = 200895 -> 200
      + deleted_at     = (known after apply)
      ~ config         = {
          - old_field = "value" -> null
        }
    }

Plan: 0 to add, 1 to change, 0 to destroy.
`,
			want: false, // Mixed real and computed changes should be detected
		},
		{
			name: "deletion within nested block structure",
			planOutput: `
Terraform will perform the following actions:

  # cloudflare_resource.example will be updated in-place
  ~ resource "cloudflare_resource" "example" {
      ~ settings = {
          ~ nested = {
              - field1 = "value1" -> null
              - field2 = "value2" -> null
            }
        }
    }

Plan: 0 to add, 5 to change, 0 to destroy.
`,
			want: false, // Deletions in nested blocks
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasOnlyComputedChangesDefault(tt.planOutput)
			if got != tt.want {
				t.Errorf("hasOnlyComputedChangesDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadDriftExemptions(t *testing.T) {
	// Create temp directory for test configs
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		configYAML string
		setupFunc  func(string) string // Returns path to go.mod to set as repo root
		wantErr    bool
		validate   func(t *testing.T, config *DriftExemptionsConfig)
	}{
		{
			name: "valid config",
			configYAML: `
exemptions:
  - name: "test-exemption"
    description: "Test exemption"
    patterns:
      - "status.*active"
    enabled: true
settings:
  apply_exemptions: true
  verbose_exemptions: false
`,
			setupFunc: func(dir string) string {
				// Create e2e/drift-exemptions.yaml
				e2eDir := filepath.Join(dir, "e2e")
				os.MkdirAll(e2eDir, 0755)
				configPath := filepath.Join(e2eDir, "drift-exemptions.yaml")
				os.WriteFile(configPath, []byte(`
exemptions:
  - name: "test-exemption"
    description: "Test exemption"
    patterns:
      - "status.*active"
    enabled: true
settings:
  apply_exemptions: true
  verbose_exemptions: false
`), 0644)
				// Create go.mod to mark as repo root
				os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0644)
				return dir
			},
			wantErr: false,
			validate: func(t *testing.T, config *DriftExemptionsConfig) {
				if len(config.Exemptions) != 1 {
					t.Errorf("Expected 1 exemption, got %d", len(config.Exemptions))
				}
				if config.Exemptions[0].Name != "test-exemption" {
					t.Errorf("Expected exemption name 'test-exemption', got %v", config.Exemptions[0].Name)
				}
				if !config.Settings.ApplyExemptions {
					t.Error("Expected apply_exemptions to be true")
				}
			},
		},
		{
			name: "no config file returns empty config",
			setupFunc: func(dir string) string {
				// Create go.mod but no drift-exemptions.yaml
				os.MkdirAll(filepath.Join(dir, "e2e"), 0755)
				os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0644)
				return dir
			},
			wantErr: false,
			validate: func(t *testing.T, config *DriftExemptionsConfig) {
				if len(config.Exemptions) != 0 {
					t.Errorf("Expected 0 exemptions, got %d", len(config.Exemptions))
				}
				if config.Settings.ApplyExemptions {
					t.Error("Expected apply_exemptions to be false by default")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := filepath.Join(tmpDir, tt.name)
			os.MkdirAll(testDir, 0755)

			// Setup test environment
			if tt.setupFunc != nil {
				repoRoot := tt.setupFunc(testDir)
				// Change to test directory
				oldWd, _ := os.Getwd()
				os.Chdir(repoRoot)
				defer os.Chdir(oldWd)
			}

			config, err := loadDriftExemptions()

			if (err != nil) != tt.wantErr {
				t.Errorf("loadDriftExemptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}

func TestHasOnlyComputedChangesWithExemptions(t *testing.T) {
	config := &DriftExemptionsConfig{
		Exemptions: []DriftExemption{
			{
				Name:        "computed-fields",
				Description: "Known computed field changes",
				Patterns:    []string{`id\s+=.*\(known after apply\)`},
				Enabled:     true,
			},
			{
				Name:          "zone-specific",
				Description:   "Zone-specific exemptions",
				ResourceTypes: []string{"cloudflare_zone"},
				Patterns:      []string{`name_servers`},
				Enabled:       true,
			},
			{
				Name:        "disabled-exemption",
				Description: "This should not match",
				Patterns:    []string{`should_not_match`},
				Enabled:     false,
			},
		},
		Settings: struct {
			ApplyExemptions   bool `yaml:"apply_exemptions"`
			VerboseExemptions bool `yaml:"verbose_exemptions"`
		}{
			ApplyExemptions:   true,
			VerboseExemptions: false,
		},
	}

	// Pre-compile patterns for test (mimicking what loadDriftExemptions does)
	for i := range config.Exemptions {
		exemption := &config.Exemptions[i]
		exemption.compiledPatterns = make([]*regexp.Regexp, 0, len(exemption.Patterns))
		for _, pattern := range exemption.Patterns {
			if compiled, err := regexp.Compile(pattern); err == nil {
				exemption.compiledPatterns = append(exemption.compiledPatterns, compiled)
			}
		}
	}

	tests := []struct {
		name                    string
		planOutput              string
		wantOnlyComputed        bool
		wantTriggeredExemptions map[string]int
	}{
		{
			name: "exempted computed field",
			planOutput: `
# cloudflare_zone.example will be updated in-place
  ~ resource "cloudflare_zone" "example" {
      ~ id = (known after apply)
    }
`,
			wantOnlyComputed:        true,
			wantTriggeredExemptions: map[string]int{"computed-fields": 1},
		},
		{
			name: "exempted resource-specific change",
			planOutput: `
# cloudflare_zone.example will be updated in-place
  ~ resource "cloudflare_zone" "example" {
      ~ name_servers = ["ns1", "ns2"]
    }
`,
			wantOnlyComputed:        true,
			wantTriggeredExemptions: map[string]int{"zone-specific": 1},
		},
		{
			name: "non-exempted change",
			planOutput: `
# cloudflare_zone.example will be updated in-place
  ~ resource "cloudflare_zone" "example" {
      ~ real_change = "old" -> "new"
    }
`,
			wantOnlyComputed:        false,
			wantTriggeredExemptions: map[string]int{},
		},
		{
			name: "disabled exemption not triggered",
			planOutput: `
# cloudflare_zone.example will be updated in-place
  ~ resource "cloudflare_zone" "example" {
      ~ should_not_match = "test" -> "new"
    }

Plan: 0 to add, 1 to change, 0 to destroy.
`,
			wantOnlyComputed:        false, // Real change detected (disabled exemption doesn't match)
			wantTriggeredExemptions: map[string]int{},
		},
		{
			name: "multiple exemptions triggered",
			planOutput: `
# cloudflare_zone.example will be updated in-place
  ~ resource "cloudflare_zone" "example" {
      ~ id = (known after apply)
      ~ name_servers = ["ns1"]
    }
`,
			wantOnlyComputed:        true,
			wantTriggeredExemptions: map[string]int{"computed-fields": 1, "zone-specific": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotComputed, gotExemptions, gotDriftLines, _ := hasOnlyComputedChangesWithExemptions(tt.planOutput, config)

			if gotComputed != tt.wantOnlyComputed {
				t.Errorf("hasOnlyComputedChangesWithExemptions() computed = %v, want %v", gotComputed, tt.wantOnlyComputed)
			}

			if len(gotExemptions) != len(tt.wantTriggeredExemptions) {
				t.Errorf("Triggered exemptions count = %d, want %d", len(gotExemptions), len(tt.wantTriggeredExemptions))
			}

			for name, count := range tt.wantTriggeredExemptions {
				if gotExemptions[name] != count {
					t.Errorf("Exemption %s triggered %d times, want %d", name, gotExemptions[name], count)
				}
			}

			// Verify drift lines are captured when there are real changes
			if !gotComputed && len(gotDriftLines) == 0 {
				t.Error("Expected drift lines to be captured for non-computed changes")
			}
		})
	}
}

func TestCheckDrift(t *testing.T) {
	// Create temp config for test
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "e2e"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644)

	configPath := filepath.Join(tmpDir, "e2e", "drift-exemptions.yaml")
	configYAML := `
exemptions:
  - name: "test-exemption"
    description: "Test"
    patterns:
      - "test_field"
    enabled: true
settings:
  apply_exemptions: true
  verbose_exemptions: false
`
	os.WriteFile(configPath, []byte(configYAML), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	planOutput := `
  ~ resource "cloudflare_zone" "example" {
      ~ test_field = "old" -> "new"
    }
`

	result := checkDrift(planOutput)

	if !result.OnlyComputedChanges {
		t.Error("Expected OnlyComputedChanges to be true with exemption")
	}

	if !result.ExemptionsEnabled {
		t.Error("Expected ExemptionsEnabled to be true")
	}

	if result.TriggeredExemptions["test-exemption"] != 1 {
		t.Errorf("Expected 1 triggered exemption, got %d", result.TriggeredExemptions["test-exemption"])
	}
}

func TestExtractPlanSummary(t *testing.T) {
	tests := []struct {
		name       string
		planOutput string
		want       string
	}{
		{
			name: "standard plan summary",
			planOutput: `
Some output...
Plan: 1 to add, 2 to change, 0 to destroy.
More output...
`,
			want: "Plan: 1 to add, 2 to change, 0 to destroy.",
		},
		{
			name: "no changes",
			planOutput: `
No changes detected
`,
			want: "",
		},
		{
			name: "plan with resources",
			planOutput: `
Terraform will perform the following actions:
  # resource changes
Plan: 0 to add, 0 to change, 1 to destroy.
`,
			want: "Plan: 0 to add, 0 to change, 1 to destroy.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPlanSummary(tt.planOutput)
			if got != tt.want {
				t.Errorf("extractPlanSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractPlanChanges(t *testing.T) {
	tests := []struct {
		name       string
		planOutput string
		wantContains []string
		wantNotContains []string
	}{
		{
			name: "extracts resource changes",
			planOutput: `
Terraform will perform the following actions:

  # cloudflare_zone.example will be updated in-place
  ~ resource "cloudflare_zone" "example" {
      ~ status = "pending" -> "active"
      ~ id     = "old" -> "new"
    }

  # cloudflare_record.test will be created
  + resource "cloudflare_record" "test" {
      + name = "test.example.com"
    }

Plan: 1 to add, 1 to change, 0 to destroy.
`,
			wantContains: []string{
				"cloudflare_zone.example",
				"cloudflare_record.test",
				"status",
				"name",
			},
			wantNotContains: []string{
				"Plan:",
			},
		},
		{
			name: "no changes section",
			planOutput: `
No changes. Infrastructure is up-to-date.
`,
			wantContains: []string{},
			wantNotContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPlanChanges(tt.planOutput)

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("extractPlanChanges() should contain %q, got: %s", want, got)
				}
			}

			for _, notWant := range tt.wantNotContains {
				if strings.Contains(got, notWant) {
					t.Errorf("extractPlanChanges() should not contain %q, got: %s", notWant, got)
				}
			}
		})
	}
}

func TestFormatResourceChange(t *testing.T) {
	line := "  # cloudflare_zone.example will be updated"
	result := formatResourceChange(line)

	// Should contain the line with color codes
	if !strings.Contains(result, line) {
		t.Errorf("formatResourceChange() should contain original line, got: %s", result)
	}

	// Should contain yellow color
	if !strings.Contains(result, colorYellow) {
		t.Errorf("formatResourceChange() should contain yellow color code")
	}

	// Should contain reset color
	if !strings.Contains(result, colorReset) {
		t.Errorf("formatResourceChange() should contain color reset code")
	}
}

func TestFormatAttributeChange(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		color string
	}{
		{
			name:  "update with tilde",
			line:  "    ~ status = \"pending\" -> \"active\"",
			color: colorYellow,
		},
		{
			name:  "addition with plus",
			line:  "    + name = \"test\"",
			color: colorGreen,
		},
		{
			name:  "deletion with minus",
			line:  "    - old_field = \"value\"",
			color: colorRed,
		},
		{
			name:  "replace with -/+",
			line:  "    -/+ id = \"old\" -> \"new\"",
			color: colorGreen, // Actually matches "+ " first due to order of checks
		},
		{
			name:  "no change marker",
			line:  "      id = \"12345\"",
			color: "", // No color expected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAttributeChange(tt.line)

			// Should contain the original line
			if !strings.Contains(result, strings.TrimSpace(tt.line)) && tt.line != "      id = \"12345\"" {
				t.Errorf("formatAttributeChange() should contain line content")
			}

			// Check for expected color
			if tt.color != "" {
				if !strings.Contains(result, tt.color) {
					t.Errorf("formatAttributeChange() should contain color %q, got: %s", tt.color, result)
				}
			}
		})
	}
}

func TestCountUniqueDrifts(t *testing.T) {
	tests := []struct {
		name       string
		driftLines []string
		want       int
	}{
		{
			name: "no drift lines",
			driftLines: []string{},
			want: 0,
		},
		{
			name: "all unique drift lines",
			driftLines: []string{
				"  module.foo.cloudflare_zone.test: ~ status = \"pending\" -> \"active\"",
				"  module.bar.cloudflare_record.test: + name = \"test.com\"",
				"  module.baz.cloudflare_list.test: - description = \"old\"",
			},
			want: 3,
		},
		{
			name: "duplicate drift lines",
			driftLines: []string{
				"  + allow_child_bypass = (known after apply)",
				"  + allow_child_bypass = (known after apply)",
				"  + allow_child_bypass = (known after apply)",
				"  + schedule = (known after apply)",
				"  + schedule = (known after apply)",
			},
			want: 2,
		},
		{
			name: "mixed unique and duplicate",
			driftLines: []string{
				"  - check_regions = [",
				"  - check_regions = [",
				"  - check_regions = [",
				"  ~ status = \"pending\" -> \"active\"",
				"  + name = \"test\"",
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countUniqueDrifts(tt.driftLines)
			if got != tt.want {
				t.Errorf("countUniqueDrifts() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestHasOnlyComputedChanges(t *testing.T) {
	// This is a simple wrapper function, so we just test it delegates correctly
	planOutput := `
Terraform will perform the following actions:

  # cloudflare_zone.example will be updated in-place
  ~ resource "cloudflare_zone" "example" {
      ~ status = (known after apply)
    }

Plan: 0 to add, 1 to change, 0 to destroy.
`

	result := hasOnlyComputedChanges(planOutput)

	// Should return true for computed-only changes
	if !result {
		t.Errorf("hasOnlyComputedChanges() = false, want true for computed-only changes")
	}

	realChangePlan := `
Terraform will perform the following actions:

  # cloudflare_zone.example will be updated in-place
  ~ resource "cloudflare_zone" "example" {
      ~ status = "pending" -> "active"
    }

Plan: 0 to add, 1 to change, 0 to destroy.
`

	result = hasOnlyComputedChanges(realChangePlan)

	// Should return false for real changes
	if result {
		t.Errorf("hasOnlyComputedChanges() = true, want false for real changes")
	}
}

func TestResourceDeclarationSkipping_NestedResource(t *testing.T) {
	// Test that resource declarations are not captured as drift
	planOutput := `
Terraform will perform the following actions:

  # module.foo.cloudflare_zone.example will be updated in-place
  ~ resource "cloudflare_zone" "example" {
      ~ name = "old" -> "new"
    }

  # cloudflare_argo.test will be created
  + resource "cloudflare_argo" "test" {
      + zone_id = "abc123"
      + name = "test"
    }

Plan: 1 to add, 1 to change, 0 to destroy.
`

	result := checkDrift(planOutput)

	// Check that drift lines don't contain resource declarations
	for _, line := range result.RealDriftLines {
		if strings.Contains(line, "+ resource \"") || strings.Contains(line, "~ resource \"") {
			t.Errorf("Drift lines should not contain resource declarations, but found: %s", line)
		}
	}

	// Should detect the real name change
	if result.OnlyComputedChanges {
		t.Errorf("Expected real changes to be detected")
	}

	// Should have drift line for the name change
	hasNameChange := false
	for _, line := range result.RealDriftLines {
		if strings.Contains(line, "name") && strings.Contains(line, "->") {
			hasNameChange = true
			break
		}
	}
	if !hasNameChange {
		t.Errorf("Expected drift lines to include name change")
	}
}

func TestNonAttributePatternsDontMatch(t *testing.T) {
	// Test that patterns like "+ create" or "+ deleted_at" without = sign don't match
	tests := []struct {
		name       string
		planOutput string
		wantDrift  bool
	}{
		{
			name: "spurious + create should not be detected",
			planOutput: `
Terraform will perform the following actions:

  # resource.test will be updated
  ~ resource "cloudflare_test" "example" {
      + create
      ~ attr = "old" -> "new"
    }

Plan: 0 to add, 1 to change, 0 to destroy.
`,
			wantDrift: true, // Should detect the attr change, not "+ create"
		},
		{
			name: "line with + word but no equals should not match addition pattern",
			planOutput: `
Terraform will perform the following actions:

  # resource.test will be updated
  ~ resource "cloudflare_test" "example" {
      + something
      + another_thing
      ~ real_attr = "old" -> "new"
    }

Plan: 0 to add, 1 to change, 0 to destroy.
`,
			wantDrift: true, // Should only detect the real_attr change
		},
		{
			name: "proper addition with equals should match",
			planOutput: `
Terraform will perform the following actions:

  # resource.test will be updated
  ~ resource "cloudflare_test" "example" {
      + new_attr = "value"
    }

Plan: 0 to add, 1 to change, 0 to destroy.
`,
			wantDrift: true, // Should detect the addition
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasOnlyComputedChangesDefault(tt.planOutput)
			hasDrift := !result

			if hasDrift != tt.wantDrift {
				t.Errorf("hasOnlyComputedChangesDefault() detected drift = %v, want %v", hasDrift, tt.wantDrift)
			}
		})
	}
}

func TestExtractAffectedResources(t *testing.T) {
	tests := []struct {
		name       string
		planOutput string
		want       []string
	}{
		{
			name: "single module with multiple resources",
			planOutput: `
Terraform will perform the following actions:

  # module.healthcheck.cloudflare_healthcheck.counted[0] will be updated in-place
  ~ resource "cloudflare_healthcheck" "counted" {
      ~ check_regions = []
    }

  # module.healthcheck.cloudflare_healthcheck.counted[1] will be updated in-place
  ~ resource "cloudflare_healthcheck" "counted" {
      ~ check_regions = []
    }

Plan: 0 to add, 2 to change, 0 to destroy.
`,
			want: []string{"healthcheck"},
		},
		{
			name: "multiple different modules",
			planOutput: `
Terraform will perform the following actions:

  # module.healthcheck.cloudflare_healthcheck.example will be updated in-place
  ~ resource "cloudflare_healthcheck" "example" {
      ~ check_regions = []
    }

  # module.zero_trust_gateway_policy.cloudflare_zero_trust_gateway_policy.complex will be updated in-place
  ~ resource "cloudflare_zero_trust_gateway_policy" "complex" {
      ~ precedence = 400412 -> 400
    }

  # module.snippet.cloudflare_snippet.test will be updated in-place
  ~ resource "cloudflare_snippet" "test" {
      ~ name = "old" -> "new"
    }

Plan: 0 to add, 3 to change, 0 to destroy.
`,
			want: []string{"healthcheck", "snippet", "zero_trust_gateway_policy"},
		},
		{
			name: "resources without module prefix should be ignored",
			planOutput: `
Terraform will perform the following actions:

  # cloudflare_zone.example will be updated in-place
  ~ resource "cloudflare_zone" "example" {
      ~ status = "pending" -> "active"
    }

  # module.list.cloudflare_list.example will be updated in-place
  ~ resource "cloudflare_list" "example" {
      ~ name = "test"
    }

Plan: 0 to add, 2 to change, 0 to destroy.
`,
			want: []string{"list"},
		},
		{
			name: "no module resources",
			planOutput: `
Terraform will perform the following actions:

  # cloudflare_zone.example will be updated in-place
  ~ resource "cloudflare_zone" "example" {
      ~ status = "pending" -> "active"
    }

Plan: 0 to add, 1 to change, 0 to destroy.
`,
			want: []string{},
		},
		{
			name: "empty plan",
			planOutput: `
No changes. Your infrastructure matches the configuration.
`,
			want: []string{},
		},
		{
			name: "alphabetically sorted output",
			planOutput: `
Terraform will perform the following actions:

  # module.zone_dnssec.cloudflare_zone_dnssec.example will be updated in-place
  ~ resource "cloudflare_zone_dnssec" "example" {
      ~ algorithm = "13" -> "14"
    }

  # module.argo.cloudflare_argo.test will be updated in-place
  ~ resource "cloudflare_argo" "test" {
      ~ smart_routing = "on"
    }

  # module.list.cloudflare_list.example will be updated in-place
  ~ resource "cloudflare_list" "example" {
      ~ name = "test"
    }

Plan: 0 to add, 3 to change, 0 to destroy.
`,
			want: []string{"argo", "list", "zone_dnssec"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAffectedResources(tt.planOutput)

			if len(got) != len(tt.want) {
				t.Errorf("extractAffectedResources() returned %d resources, want %d\nGot: %v\nWant: %v",
					len(got), len(tt.want), got, tt.want)
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractAffectedResources()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

