package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{"equal versions", "4.52.5", "4.52.5", 0},
		{"greater major", "5.0.0", "4.52.5", 1},
		{"less major", "3.9.9", "4.52.5", -1},
		{"greater minor", "4.53.0", "4.52.5", 1},
		{"less minor", "4.51.9", "4.52.5", -1},
		{"greater patch", "4.52.6", "4.52.5", 1},
		{"less patch", "4.52.4", "4.52.5", -1},
		{"different lengths v1 shorter", "4.52", "4.52.5", -1},
		{"different lengths v2 shorter", "4.52.5", "4.52", 1},
		{"exact minimum", "4.52.5", minimumProviderVersion, 0},
		{"just below minimum", "4.52.4", minimumProviderVersion, -1},
		{"just above minimum", "4.52.6", minimumProviderVersion, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestConstraintCouldAllowMinVersion(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		expected   bool
	}{
		// Exact versions
		{"exact version above min", "4.53.0", true},
		{"exact version at min", "4.52.5", true},
		{"exact version below min", "4.52.4", false},
		{"exact version old major", "4.49.0", false},

		// >= constraints
		{"greater than or equal above min", ">= 4.53.0", true},
		{"greater than or equal at min", ">= 4.52.5", true},
		{"greater than or equal below min", ">= 4.52.0", false},
		{"greater than or equal old major", ">= 4.0.0", false},

		// ~> constraints (pessimistic)
		{"pessimistic recent", "~> 4.52", true},
		{"pessimistic exact", "~> 4.52.5", true},
		{"pessimistic old", "~> 4.49", true},     // Could be updated to 4.52.5+
		{"pessimistic very old", "~> 4.0", true}, // Could be updated

		// < constraints - ambiguous, allow
		{"less than", "< 5.0.0", true},

		// <= constraints - ambiguous, allow
		{"less than or equal", "<= 5.0.0", true},

		// > constraints
		{"greater than below min", "> 4.52.0", true}, // > 4.52.0 allows 4.52.5+
		{"greater than at min", "> 4.52.5", true},

		// = constraints
		{"equals at min", "= 4.52.5", true},
		{"equals below min", "= 4.52.4", false},

		// Unparseable - should default to true
		{"unparseable constraint", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := constraintCouldAllowMinVersion(tt.constraint)
			if result != tt.expected {
				t.Errorf("constraintCouldAllowMinVersion(%q) = %v, want %v", tt.constraint, result, tt.expected)
			}
		})
	}
}

func TestParseVersionFromLockFile(t *testing.T) {
	tempDir := t.TempDir()

	// Test with valid lock file
	lockContent := `# This file is maintained automatically by "terraform init".
provider "registry.terraform.io/cloudflare/cloudflare" {
  version     = "4.52.5"
  constraints = "~> 4.0"
  hashes = [
    "h1:+rfzF+16ZcWZWnTyW/p1HHTzYbPKX8Zt2nIFtR/+f+E=",
  ]
}
`
	lockFile := filepath.Join(tempDir, ".terraform.lock.hcl")
	err := os.WriteFile(lockFile, []byte(lockContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write lock file: %v", err)
	}

	version, err := parseVersionFromLockFile(tempDir)
	if err != nil {
		t.Fatalf("parseVersionFromLockFile failed: %v", err)
	}
	if version != "4.52.5" {
		t.Errorf("Expected version 4.52.5, got %s", version)
	}

	// Test with lock file without cloudflare provider
	lockContentNoCF := `# This file is maintained automatically by "terraform init".
provider "registry.terraform.io/hashicorp/aws" {
  version     = "5.0.0"
  constraints = "~> 5.0"
}
`
	tempDir2 := t.TempDir()
	lockFile2 := filepath.Join(tempDir2, ".terraform.lock.hcl")
	err = os.WriteFile(lockFile2, []byte(lockContentNoCF), 0644)
	if err != nil {
		t.Fatalf("Failed to write lock file: %v", err)
	}

	version, err = parseVersionFromLockFile(tempDir2)
	if err != nil {
		t.Fatalf("parseVersionFromLockFile failed: %v", err)
	}
	if version != "" {
		t.Errorf("Expected empty version, got %s", version)
	}

	// Test with no lock file
	tempDir3 := t.TempDir()
	_, err = parseVersionFromLockFile(tempDir3)
	if err == nil {
		t.Error("Expected error for missing lock file, got nil")
	}
}

func TestParseVersionFromRequiredProviders(t *testing.T) {
	tempDir := t.TempDir()

	// Test with required_providers in main.tf
	tfContent := `terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.52"
    }
  }
}
`
	tfFile := filepath.Join(tempDir, "main.tf")
	err := os.WriteFile(tfFile, []byte(tfContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write tf file: %v", err)
	}

	cfg := config{configDir: tempDir, recursive: false}
	constraint, source, err := parseVersionFromRequiredProviders(cfg)
	if err != nil {
		t.Fatalf("parseVersionFromRequiredProviders failed: %v", err)
	}
	if constraint != "~> 4.52" {
		t.Errorf("Expected constraint '~> 4.52', got %s", constraint)
	}
	if source != tfFile {
		t.Errorf("Expected source %s, got %s", tfFile, source)
	}

	// Test with no required_providers
	tempDir2 := t.TempDir()
	tfContentNoReq := `resource "aws_instance" "example" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}
`
	tfFile2 := filepath.Join(tempDir2, "main.tf")
	err = os.WriteFile(tfFile2, []byte(tfContentNoReq), 0644)
	if err != nil {
		t.Fatalf("Failed to write tf file: %v", err)
	}

	cfg2 := config{configDir: tempDir2, recursive: false}
	_, _, err = parseVersionFromRequiredProviders(cfg2)
	if err == nil {
		t.Error("Expected error for missing required_providers, got nil")
	}
}

func TestCheckMinimumProviderVersion(t *testing.T) {
	// Test v4-v5 migration with skip flag
	cfg := config{
		sourceVersion:    "v4",
		targetVersion:    "v5",
		skipVersionCheck: true,
		configDir:        "/tmp",
	}
	err := checkMinimumProviderVersion(cfg)
	if err != nil {
		t.Errorf("Expected no error with skipVersionCheck, got %v", err)
	}

	// Test non-v4-v5 migration (should skip check)
	cfg2 := config{
		sourceVersion:    "v5",
		targetVersion:    "v5",
		skipVersionCheck: false,
		configDir:        "/tmp",
	}
	err = checkMinimumProviderVersion(cfg2)
	if err != nil {
		t.Errorf("Expected no error for v5-v5 migration, got %v", err)
	}

	// Test v4-v5 migration with no lock file and no required_providers
	tempDir := t.TempDir()
	cfg3 := config{
		sourceVersion:    "v4",
		targetVersion:    "v5",
		skipVersionCheck: false,
		configDir:        tempDir,
		recursive:        false,
	}
	err = checkMinimumProviderVersion(cfg3)
	if err == nil {
		t.Error("Expected error for missing lock file and required_providers, got nil")
	}

	// Test v4-v5 migration with lock file having sufficient version
	tempDir4 := t.TempDir()
	lockContent := `provider "registry.terraform.io/cloudflare/cloudflare" {
  version = "4.52.5"
}
`
	lockFile := filepath.Join(tempDir4, ".terraform.lock.hcl")
	os.WriteFile(lockFile, []byte(lockContent), 0644)
	cfg4 := config{
		sourceVersion:    "v4",
		targetVersion:    "v5",
		skipVersionCheck: false,
		configDir:        tempDir4,
	}
	err = checkMinimumProviderVersion(cfg4)
	if err != nil {
		t.Errorf("Expected no error with sufficient version, got %v", err)
	}

	// Test v4-v5 migration with lock file having insufficient version
	tempDir5 := t.TempDir()
	lockContentOld := `provider "registry.terraform.io/cloudflare/cloudflare" {
  version = "4.49.0"
}
`
	lockFile5 := filepath.Join(tempDir5, ".terraform.lock.hcl")
	os.WriteFile(lockFile5, []byte(lockContentOld), 0644)
	cfg5 := config{
		sourceVersion:    "v4",
		targetVersion:    "v5",
		skipVersionCheck: false,
		configDir:        tempDir5,
	}
	err = checkMinimumProviderVersion(cfg5)
	if err == nil {
		t.Error("Expected error for old version, got nil")
	}
}

func TestVerifyVersionMeetsMinimum(t *testing.T) {
	// Test with sufficient version
	err := verifyVersionMeetsMinimum("4.52.5")
	if err != nil {
		t.Errorf("Expected no error for version 4.52.5, got %v", err)
	}

	err = verifyVersionMeetsMinimum("5.0.0")
	if err != nil {
		t.Errorf("Expected no error for version 5.0.0, got %v", err)
	}

	// Test with insufficient version
	err = verifyVersionMeetsMinimum("4.52.4")
	if err == nil {
		t.Error("Expected error for version 4.52.4, got nil")
	}

	err = verifyVersionMeetsMinimum("4.49.0")
	if err == nil {
		t.Error("Expected error for version 4.49.0, got nil")
	}
}
