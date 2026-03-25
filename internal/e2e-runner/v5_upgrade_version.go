// state_upgrader_version.go provides version management utilities for v5 upgrade tests.
//
// This file handles:
//   - Resolving version strings (e.g., "latest" → appropriate constraint)
//   - Generating Terraform version constraints
//   - Generating R2 state keys based on version combinations
package e2e

import (
	"fmt"
	"strings"
)

// DefaultFromVersion is the default starting provider version for v5 upgrade tests
const DefaultFromVersion = "5.5.0"

// DefaultToVersion is the default target provider version
const DefaultToVersion = "latest"

// ResolveTargetVersion resolves a version string to a Terraform provider constraint.
// For "latest", it returns a loose constraint that gets the latest published version.
// For specific versions like "5.20.0", it returns a constraint matching that minor version.
func ResolveTargetVersion(version string) (constraint string, isLatest bool) {
	if version == "latest" || version == "" {
		// Use loose constraint to get latest published v5 version
		return "~> 5.0", true
	}

	// For specific versions, use a constraint that matches the minor version
	// e.g., "5.20.0" → "~> 5.20"
	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		return fmt.Sprintf("~> %s.%s", parts[0], parts[1]), false
	}

	// Fallback: use the version as-is with ~>
	return fmt.Sprintf("~> %s", version), false
}

// ResolveFromVersion resolves the source version to a Terraform provider constraint.
// This is more strict than target version - we want the exact minor version.
func ResolveFromVersion(version string) string {
	if version == "" {
		version = DefaultFromVersion
	}

	// For source version, use exact minor version constraint
	// e.g., "5.18.0" → "~> 5.18.0"
	return fmt.Sprintf("~> %s", version)
}

// GenerateStateKey generates the R2 state key for a version combination.
// Format: v5/{fromVersion}-{toVersion}-terraform.tfstate
func GenerateStateKey(fromVersion, toVersion string) string {
	if fromVersion == "" {
		fromVersion = DefaultFromVersion
	}
	if toVersion == "" {
		toVersion = DefaultToVersion
	}
	return fmt.Sprintf("v5/%s-%s-terraform.tfstate", fromVersion, toVersion)
}

// NormalizeVersion normalizes a version string for consistent key generation.
// Removes 'v' prefix if present and ensures consistent format.
func NormalizeVersion(version string) string {
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimSpace(version)
	if version == "" {
		return "latest"
	}
	return version
}

// GenerateProviderTF generates the provider.tf content with the specified version constraint.
func GenerateProviderTF(versionConstraint string) string {
	return fmt.Sprintf(`terraform {
  required_version = ">= 1.0"

  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "%s"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
  }

  # Remote state backend (R2)
  # Configuration provided via backend.configured.hcl
  backend "s3" {}
}

# Uses CLOUDFLARE_API_KEY and CLOUDFLARE_EMAIL environment variables
provider "cloudflare" {}

# Common variables from environment
variable "cloudflare_account_id" {
  description = "Cloudflare account ID for resources"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for DNS records"
  type        = string
}

variable "cloudflare_domain" {
  description = "Cloudflare domain for testing"
  type        = string
}

variable "from_version" {
  description = "Provider version under test, used to namespace resource names"
  type        = string
}

variable "crowdstrike_client_id" {
  description = "CrowdStrike client ID for posture integration tests"
  type        = string
}

variable "crowdstrike_client_secret" {
  description = "CrowdStrike client secret for posture integration tests"
  type        = string
  sensitive   = true
}

variable "crowdstrike_api_url" {
  description = "CrowdStrike API URL for posture integration tests"
  type        = string
}

variable "crowdstrike_customer_id" {
  description = "CrowdStrike customer ID for posture integration tests"
  type        = string
}
`, versionConstraint)
}

// GenerateBackendConfiguredHCL generates the backend.configured.hcl content
// with the actual account ID and state key.
func GenerateBackendConfiguredHCL(accountID, fromVersion, toVersion string) string {
	stateKey := GenerateStateKey(fromVersion, toVersion)
	return fmt.Sprintf(`# Auto-generated backend configuration for v5 upgrade tests
# From: %s → To: %s

endpoint = "https://%s.r2.cloudflarestorage.com"

bucket = "tf-migrate-e2e-state"
key    = "%s"
region = "auto"

# Skip AWS-specific validations
skip_credentials_validation = true
skip_region_validation      = true
skip_requesting_account_id  = true
skip_metadata_api_check     = true
skip_s3_checksum            = true

# Use path-style access for R2
force_path_style = true
`, fromVersion, toVersion, accountID, stateKey)
}
