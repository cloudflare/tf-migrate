package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"

	"github.com/cloudflare/tf-migrate/internal/registry"
)

func init() {
	// Ensure all resource migrations are registered for tests
	registry.RegisterAllMigrations()
}

func TestRunPreMigrationScan_ClassifiesResources(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a .tf file with a mix of resource types
	content := `
resource "cloudflare_access_application" "admin_api" {
  account_id = "abc123"
  name       = "Admin API"
  type       = "self_hosted"
}

resource "cloudflare_ruleset" "rate_limit" {
  zone_id = "abc123"
  name    = "Rate Limit"
  kind    = "zone"
  phase   = "http_ratelimit"
}

resource "cloudflare_dns_record" "root_a" {
  zone_id = "abc123"
  name    = "example.com"
  type    = "A"
  value   = "192.0.2.1"
}

resource "aws_s3_bucket" "logs" {
  bucket = "my-logs"
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	log := hclog.NewNullLogger()
	cfg := config{
		configDir:     tmpDir,
		sourceVersion: "v4",
		targetVersion: "v5",
	}

	report, err := runPreMigrationScan(log, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Should NOT include the aws_s3_bucket (not a cloudflare resource)
	for _, r := range report.Resources {
		if r.ResourceType == "aws_s3_bucket" {
			t.Error("Non-cloudflare resource should not be included in scan")
		}
	}

	// Should find cloudflare resources
	if len(report.Resources) == 0 {
		t.Fatal("Expected to find cloudflare resources")
	}

	// Check that access_application is classified as renamed
	found := false
	for _, r := range report.Resources {
		if r.ResourceType == "cloudflare_access_application" {
			found = true
			if r.Class != classRenamed {
				t.Errorf("Expected access_application to be classRenamed, got %d", r.Class)
			}
			if r.NewType != "cloudflare_zero_trust_access_application" {
				t.Errorf("Expected new type cloudflare_zero_trust_access_application, got %s", r.NewType)
			}
		}
	}
	if !found {
		t.Error("Expected to find cloudflare_access_application in scan results")
	}
}

func TestRunPreMigrationScan_DetectsMovedBlocks(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a main.tf with a resource
	mainContent := `
resource "cloudflare_access_application" "admin_api" {
  account_id = "abc123"
  name       = "Admin API"
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(mainContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Write a moved.tf with hand-written moved blocks
	movedContent := `
moved {
  from = cloudflare_access_application.admin_api
  to   = cloudflare_zero_trust_access_application.admin_api
}

moved {
  from = cloudflare_record.root_a
  to   = cloudflare_dns_record.root_a
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "moved.tf"), []byte(movedContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	log := hclog.NewNullLogger()
	cfg := config{
		configDir:     tmpDir,
		sourceVersion: "v4",
		targetVersion: "v5",
	}

	report, err := runPreMigrationScan(log, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.MovedBlocks) != 2 {
		t.Fatalf("Expected 2 moved blocks, got %d", len(report.MovedBlocks))
	}

	// Should have a warning about the duplicate moved block for access_application
	if len(report.Warnings) == 0 {
		t.Error("Expected warnings about pre-existing moved blocks")
	}

	foundDuplicateWarning := false
	for _, w := range report.Warnings {
		if contains(w, "tf-migrate will generate this moved block automatically") {
			foundDuplicateWarning = true
		}
	}
	if !foundDuplicateWarning {
		t.Errorf("Expected warning about duplicate moved block, got warnings: %v", report.Warnings)
	}
}

func TestRunPreMigrationScan_ClassifiesAppScopedAccessPolicyAsManual(t *testing.T) {
	tmpDir := t.TempDir()

	content := `
resource "cloudflare_access_policy" "app_scoped" {
  account_id     = "abc123"
  application_id = "app-123"
  name           = "App Scoped"
  decision       = "allow"

  include {
    everyone = true
  }
}
`

	err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	log := hclog.NewNullLogger()
	cfg := config{
		configDir:     tmpDir,
		sourceVersion: "v4",
		targetVersion: "v5",
	}

	report, err := runPreMigrationScan(log, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(report.Resources))
	}

	got := report.Resources[0]
	if got.Class != classManualIntervention {
		t.Fatalf("Expected classManualIntervention, got %d", got.Class)
	}

	if !contains(got.Detail, "application_id") {
		t.Fatalf("Expected manual detail to mention application_id, got: %s", got.Detail)
	}

	if !contains(got.Detail, "do not use moved block") {
		t.Fatalf("Expected manual detail to mention moved block guidance, got: %s", got.Detail)
	}
}

func TestDetectMovedBlockConflicts_DuplicateMovedBlock(t *testing.T) {
	renames := map[string]string{
		"cloudflare_access_application": "cloudflare_zero_trust_access_application",
	}

	report := &preflightReport{
		MovedBlocks: []existingMovedBlock{
			{
				File:     "moved.tf",
				FromType: "cloudflare_access_application",
				FromName: "admin_api",
				ToType:   "cloudflare_zero_trust_access_application",
				ToName:   "admin_api",
			},
		},
	}

	warnings := detectMovedBlockConflicts(report, renames)
	if len(warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !contains(warnings[0], "tf-migrate will generate this moved block automatically") {
		t.Errorf("Unexpected warning: %s", warnings[0])
	}
}

func TestDetectMovedBlockConflicts_ConflictingTarget(t *testing.T) {
	renames := map[string]string{
		"cloudflare_access_application": "cloudflare_zero_trust_access_application",
	}

	report := &preflightReport{
		MovedBlocks: []existingMovedBlock{
			{
				File:     "moved.tf",
				FromType: "cloudflare_access_application",
				FromName: "admin_api",
				ToType:   "cloudflare_wrong_type",
				ToName:   "admin_api",
			},
		},
	}

	warnings := detectMovedBlockConflicts(report, renames)
	if len(warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !contains(warnings[0], "Conflicting moved block") {
		t.Errorf("Expected conflicting moved block warning, got: %s", warnings[0])
	}
}

func TestDetectMovedBlockConflicts_NoConflict(t *testing.T) {
	renames := map[string]string{
		"cloudflare_access_application": "cloudflare_zero_trust_access_application",
	}

	// A moved block for a non-renamed resource (no conflict)
	report := &preflightReport{
		MovedBlocks: []existingMovedBlock{
			{
				File:     "moved.tf",
				FromType: "cloudflare_custom_resource",
				FromName: "example",
				ToType:   "cloudflare_custom_resource_v2",
				ToName:   "example",
			},
		},
		Resources: []scannedResource{
			{
				ResourceType: "cloudflare_custom_resource",
				ResourceName: "example",
			},
		},
	}

	warnings := detectMovedBlockConflicts(report, renames)
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings, got %d: %v", len(warnings), warnings)
	}
}

func TestDetectMovedBlockConflicts_ManualResourceMovedBlock(t *testing.T) {
	renames := map[string]string{
		"cloudflare_access_policy": "cloudflare_zero_trust_access_policy",
	}

	report := &preflightReport{
		Resources: []scannedResource{
			{
				ResourceType: "cloudflare_access_policy",
				ResourceName: "app_scoped",
				Class:        classManualIntervention,
			},
		},
		MovedBlocks: []existingMovedBlock{
			{
				File:     "moved.tf",
				FromType: "cloudflare_access_policy",
				FromName: "app_scoped",
				ToType:   "cloudflare_zero_trust_access_policy",
				ToName:   "app_scoped",
			},
		},
	}

	warnings := detectMovedBlockConflicts(report, renames)
	if len(warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !contains(warnings[0], "requires manual migration") {
		t.Fatalf("Expected manual migration warning, got: %s", warnings[0])
	}
	if !contains(warnings[0], "Do not use a moved block") {
		t.Fatalf("Expected moved-block guidance warning, got: %s", warnings[0])
	}
}

func TestClassifyResource_UnsupportedResource(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a .tf file with a made-up cloudflare resource type
	content := `
resource "cloudflare_nonexistent_resource" "test" {
  name = "test"
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	log := hclog.NewNullLogger()
	cfg := config{
		configDir:     tmpDir,
		sourceVersion: "v4",
		targetVersion: "v5",
	}

	report, err := runPreMigrationScan(log, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(report.Resources))
	}
	if report.Resources[0].Class != classUnsupported {
		t.Errorf("Expected classUnsupported, got %d", report.Resources[0].Class)
	}
}

func TestRunPreMigrationScan_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	log := hclog.NewNullLogger()
	cfg := config{
		configDir:     tmpDir,
		sourceVersion: "v4",
		targetVersion: "v5",
	}

	report, err := runPreMigrationScan(log, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.Resources) != 0 {
		t.Errorf("Expected 0 resources, got %d", len(report.Resources))
	}
	if len(report.MovedBlocks) != 0 {
		t.Errorf("Expected 0 moved blocks, got %d", len(report.MovedBlocks))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestRunPreMigrationScan_AutoFixMovedBlocks(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a resource file with a v4 resource
	resourceContent := `resource "cloudflare_access_identity_provider" "myid_saml" {
  account_id = "test-account"
  name       = "MyIdentity"
  type       = "saml"
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "resources.tf"), []byte(resourceContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create a moved block with v4 type names that should be auto-fixed
	movedContent := `moved {
  from = cloudflare_access_identity_provider.old_saml
  to   = cloudflare_access_identity_provider.myid_saml
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "moved.tf"), []byte(movedContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	log := hclog.NewNullLogger()
	cfg := config{
		configDir:     tmpDir,
		sourceVersion: "v4",
		targetVersion: "v5",
		verbose:       true,
	}

	report, err := runPreMigrationScan(log, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Should find the resource
	if len(report.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(report.Resources))
	}

	// Should find the moved block
	if len(report.MovedBlocks) != 1 {
		t.Fatalf("Expected 1 moved block, got %d", len(report.MovedBlocks))
	}

	// Verify the moved block was updated in the report
	mb := report.MovedBlocks[0]
	if mb.FromType != "cloudflare_zero_trust_access_identity_provider" {
		t.Errorf("Expected FromType to be updated to 'cloudflare_zero_trust_access_identity_provider', got %q", mb.FromType)
	}
	if mb.ToType != "cloudflare_zero_trust_access_identity_provider" {
		t.Errorf("Expected ToType to be updated to 'cloudflare_zero_trust_access_identity_provider', got %q", mb.ToType)
	}

	// Verify the file was actually updated on disk
	updatedContent, err := os.ReadFile(filepath.Join(tmpDir, "moved.tf"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(updatedContent), "cloudflare_zero_trust_access_identity_provider.old_saml") {
		t.Errorf("Updated file should contain v5 type in 'from' field")
	}
	if !strings.Contains(string(updatedContent), "cloudflare_zero_trust_access_identity_provider.myid_saml") {
		t.Errorf("Updated file should contain v5 type in 'to' field")
	}

	// Should NOT have the "Conflicting moved block" warning since it was auto-fixed
	for _, w := range report.Warnings {
		if strings.Contains(w, "Conflicting moved block") {
			t.Errorf("Should not have 'Conflicting moved block' warning after auto-fix, but got: %s", w)
		}
	}
}
