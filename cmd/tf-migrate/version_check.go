package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// minimumProviderVersion is the minimum Cloudflare provider version required
// for v4 to v5 migrations. This version introduced critical state migration
// capabilities needed for tf-migrate to work correctly.
const minimumProviderVersion = "4.52.5"

// checkMinimumProviderVersion verifies that the installed Cloudflare provider
// version meets the minimum requirement (4.52.5) for v4 to v5 migrations.
// It checks the lock file first, then falls back to required_providers.
func checkMinimumProviderVersion(cfg config) error {
	// Skip check if explicitly disabled (for testing/CI)
	if cfg.skipVersionCheck {
		return nil
	}

	// Only apply this check for v4 to v5 migrations
	if cfg.sourceVersion != "v4" || cfg.targetVersion != "v5" {
		return nil
	}

	// First, try to get the version from the lock file (most reliable)
	version, err := parseVersionFromLockFile(cfg.configDir)
	if err == nil {
		if version == "" {
			return fmt.Errorf("no Cloudflare provider found in .terraform.lock.hcl")
		}
		if err := verifyVersionMeetsMinimum(version); err != nil {
			return err
		}
		return nil
	}

	// Lock file doesn't exist or couldn't be parsed, try required_providers
	versionConstraint, constraintSource, err := parseVersionFromRequiredProviders(cfg)
	if err == nil && versionConstraint != "" {
		// Check if the constraint could possibly allow a version >= 4.52.5
		if !constraintCouldAllowMinVersion(versionConstraint) {
			return fmt.Errorf("Cloudflare provider version constraint %q in %s may allow versions below the required %s\n\nPlease update your required_providers to require at least %s, run 'terraform init', then re-run tf-migrate",
				versionConstraint, constraintSource, minimumProviderVersion, minimumProviderVersion)
		}
		// Constraint might be OK, but we can't verify without a lock file
		return fmt.Errorf("Could not verify the installed Cloudflare provider version.\nNo .terraform.lock.hcl found in %s.\n\nThe v4 to v5 migration requires Cloudflare provider v%s or higher.\nPlease run 'terraform init' in your configuration directory first, then re-run tf-migrate.",
			cfg.configDir, minimumProviderVersion)
	}

	// Neither lock file nor required_providers found
	return fmt.Errorf("Could not verify the installed Cloudflare provider version.\nNo .terraform.lock.hcl or required_providers block found in %s.\n\nThe v4 to v5 migration requires Cloudflare provider v%s or higher.\nPlease ensure you have a terraform block with required_providers, run 'terraform init', then re-run tf-migrate.",
		cfg.configDir, minimumProviderVersion)
}

// parseVersionFromLockFile reads .terraform.lock.hcl and extracts the
// installed Cloudflare provider version.
// Returns empty string if the cloudflare provider is not in the lock file.
func parseVersionFromLockFile(configDir string) (string, error) {
	lockFilePath := filepath.Join(configDir, ".terraform.lock.hcl")

	content, err := os.ReadFile(lockFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("lock file not found")
		}
		return "", fmt.Errorf("failed to read lock file: %w", err)
	}

	parsed, diags := hclwrite.ParseConfig(content, ".terraform.lock.hcl", hcl.InitialPos)
	if diags.HasErrors() {
		return "", fmt.Errorf("failed to parse lock file: %s", diags.Error())
	}

	for _, block := range parsed.Body().Blocks() {
		if block.Type() != "provider" {
			continue
		}

		labels := block.Labels()
		if len(labels) == 0 {
			continue
		}

		// Check for registry.terraform.io/cloudflare/cloudflare
		// or just cloudflare/cloudflare (older formats)
		providerLabel := labels[0]
		if providerLabel == "registry.terraform.io/cloudflare/cloudflare" ||
			providerLabel == "cloudflare/cloudflare" {

			versionAttr := block.Body().GetAttribute("version")
			if versionAttr == nil {
				continue
			}

			versionStr := string(versionAttr.Expr().BuildTokens(nil).Bytes())
			// Remove quotes from the version string
			versionStr = strings.Trim(versionStr, ` "`)
			return versionStr, nil
		}
	}

	return "", nil
}

// parseVersionFromRequiredProviders scans all .tf files for the required_providers
// block and extracts the cloudflare provider version constraint.
// Returns the version constraint string and the file path where it was found.
func parseVersionFromRequiredProviders(cfg config) (versionConstraint, sourceFile string, err error) {
	files, err := findTerraformFilesWithRecursion(cfg.configDir, cfg.recursive)
	if err != nil {
		return "", "", err
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		parsed, diags := hclwrite.ParseConfig(content, filepath.Base(file), hcl.InitialPos)
		if diags.HasErrors() {
			continue
		}

		for _, block := range parsed.Body().Blocks() {
			if block.Type() != "terraform" {
				continue
			}

			for _, inner := range block.Body().Blocks() {
				if inner.Type() != "required_providers" {
					continue
				}

				cfAttr := inner.Body().GetAttribute("cloudflare")
				if cfAttr == nil {
					continue
				}

				// Extract the version from the cloudflare attribute
				attrStr := string(cfAttr.BuildTokens(nil).Bytes())
				if !strings.Contains(attrStr, "cloudflare/cloudflare") {
					continue
				}

				// Look for version = "..." pattern
				re := regexp.MustCompile(`version\s*=\s*"([^"]+)"`)
				matches := re.FindStringSubmatch(attrStr)
				if len(matches) > 1 {
					return matches[1], file, nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("no required_providers block with cloudflare provider found")
}

// verifyVersionMeetsMinimum compares a version string against the minimum required.
// Versions must be in MAJOR.MINOR.PATCH format.
func verifyVersionMeetsMinimum(version string) error {
	if compareVersions(version, minimumProviderVersion) < 0 {
		return fmt.Errorf("Cloudflare provider version %s is below the minimum required %s.\n\nPlease upgrade your provider to at least v%s before migrating:\n  1. Update your required_providers version constraint to \">= %s\"\n  2. Run: terraform init -upgrade\n  3. Run: terraform apply (to ensure state is compatible)\n  4. Re-run tf-migrate",
			version, minimumProviderVersion, minimumProviderVersion, minimumProviderVersion)
	}
	return nil
}

// compareVersions compares two version strings in MAJOR.MINOR.PATCH format.
// Returns:
//
//	-1 if v1 < v2
//	 0 if v1 == v2
//	 1 if v1 > v2
func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Ensure we have at least 3 parts for both
	for len(parts1) < 3 {
		parts1 = append(parts1, "0")
	}
	for len(parts2) < 3 {
		parts2 = append(parts2, "0")
	}

	for i := 0; i < 3; i++ {
		n1, _ := strconv.Atoi(parts1[i])
		n2, _ := strconv.Atoi(parts2[i])

		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}

	return 0
}

// constraintCouldAllowMinVersion checks if a version constraint could possibly
// allow a version >= 4.52.5. This is a simplified check for common constraint formats.
// Returns true if the constraint MIGHT allow a sufficient version, false if
// it definitely doesn't.
func constraintCouldAllowMinVersion(constraint string) bool {
	constraint = strings.TrimSpace(constraint)

	// Extract version numbers from various constraint patterns
	// Patterns: ~> 4.0, >= 4.0, = 4.0, 4.0, < 5.0, etc.
	// Note: In raw string literals (`), backslashes are literal, so `\s` is correct for whitespace
	re := regexp.MustCompile(`^(?:>=?|<=?|~>|=)?\s*(\d+)(?:\.(\d+))?(?:\.(\d+))?`)
	matches := re.FindStringSubmatch(constraint)

	if len(matches) == 0 {
		// Unparseable constraint, assume it might be OK (will need lock file check)
		return true
	}

	major, _ := strconv.Atoi(matches[1])
	minor := 0
	patch := 0

	if len(matches) > 2 && matches[2] != "" {
		minor, _ = strconv.Atoi(matches[2])
	}
	if len(matches) > 3 && matches[3] != "" {
		patch, _ = strconv.Atoi(matches[3])
	}

	// Build the version string from the extracted components
	extractedVersion := fmt.Sprintf("%d.%d.%d", major, minor, patch)

	// If constraint starts with ~>, it's pessimistic and only allows patch updates
	// for the specified minor version, or minor updates within the same major.
	// Examples: ~> 4.52 allows 4.52.x and 4.53.0+ but not 5.0.0
	//           ~> 4.0 allows 4.0.x through 4.99.99 but not 5.0.0
	if strings.HasPrefix(constraint, "~>") {
		// For ~>, the version must be >= the specified version
		// If ~> 4.52, we need at least 4.52.0, but if the constraint is ~> 4.49,
		// we still need 4.52.5, so we check if the minimum is possible
		return compareVersions(extractedVersion, minimumProviderVersion) <= 0 ||
			(major == 4 && minor >= 52)
	}

	// For >= constraints, check if the lower bound is sufficient
	if strings.HasPrefix(constraint, ">=") {
		return compareVersions(extractedVersion, minimumProviderVersion) >= 0
	}

	// For > constraints: allows versions strictly greater than extractedVersion.
	// If extractedVersion >= minimum, then extractedVersion+1 >= minimum too.
	// If extractedVersion < minimum, we need to check if minimum > extractedVersion.
	// For practical purposes, "> X" where X is any reasonable version will allow
	// versions >= 4.52.5 (e.g., > 4.52.4 allows 4.52.5+, > 4.0 allows 4.52.5+, etc.)
	// Only fail if extractedVersion is absurdly high (>= 999.0.0)
	if strings.HasPrefix(constraint, ">") && !strings.HasPrefix(constraint, ">=") {
		return compareVersions(extractedVersion, "999.0.0") < 0
	}

	// For = constraints: exact version must be >= minimum
	if strings.HasPrefix(constraint, "=") {
		return compareVersions(extractedVersion, minimumProviderVersion) >= 0
	}

	// For exact version without operator (e.g., "4.52.5")
	if regexp.MustCompile(`^\d`).MatchString(constraint) {
		return compareVersions(extractedVersion, minimumProviderVersion) >= 0
	}

	// For other constraints (<=, <, !=), we can't easily determine
	// without full constraint parsing, so assume it might be OK
	return true
}
