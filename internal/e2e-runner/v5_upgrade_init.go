// state_upgrader_init.go handles initialization of v5→v5 v5 upgrade test infrastructure.
//
// This file provides functionality to set up v5 upgrade test environments, including:
//   - Syncing v5 test fixtures from e2e-v5/testdata/ to e2e-v5/tf/
//   - Generating provider.tf with appropriate version constraints
//   - Generating backend configuration with version-specific state keys
//   - Generating main.tf with module references
package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// V5UpgradeConfig holds configuration for v5 upgrade tests
type V5UpgradeConfig struct {
	FromVersion     string // Source provider version (e.g., "5.18.0")
	ToVersion       string // Target provider version (e.g., "latest" or "5.20.0")
	Resources       string // Comma-separated resource names to include
	Exclude         string // Comma-separated resource names to exclude
	ApplyExemptions bool
	Parallelism     int
	SkipCreate      bool   // Skip resource creation, use existing state
	ProviderPath    string // Optional: path to local provider source
	Clean           bool   // Destroy resources and clean up state after test
}

// RunV5UpgradeInit syncs v5 testdata and generates terraform configs
func RunV5UpgradeInit(cfg *V5UpgradeConfig) error {
	// Normalize versions
	if cfg.FromVersion == "" {
		cfg.FromVersion = DefaultFromVersion
	}
	if cfg.ToVersion == "" {
		cfg.ToVersion = DefaultToVersion
	}

	// Load required environment variables
	env, err := LoadEnv(EnvForInit)
	if err != nil {
		return err
	}

	// Parse resource filter if provided
	var targetResources []string
	if cfg.Resources != "" {
		for _, r := range strings.Split(cfg.Resources, ",") {
			targetResources = append(targetResources, strings.TrimSpace(r))
		}
	}

	// Parse exclude filter if provided
	excludeSet := make(map[string]bool)
	if cfg.Exclude != "" {
		for _, r := range strings.Split(cfg.Exclude, ",") {
			excludeSet[strings.TrimSpace(r)] = true
		}
	}

	// Get paths
	repoRoot := getRepoRoot()
	e2eV5Root := filepath.Join(repoRoot, "e2e-v5")
	tfDir := filepath.Join(e2eV5Root, "tf")
	testdataRoot := filepath.Join(e2eV5Root, "testdata")

	// Check if testdata directory exists
	if _, err := os.Stat(testdataRoot); os.IsNotExist(err) {
		return fmt.Errorf("testdata directory not found at %s\nRun 'make generate-su-testdata' to create test fixtures", testdataRoot)
	}

	printHeader("V5 Upgrade Test Initialization")
	printCyan("Provider versions: %s → %s", cfg.FromVersion, cfg.ToVersion)
	fmt.Println()

	if len(targetResources) > 0 {
		printYellow("Filtering to specific resources: %s", strings.Join(targetResources, ", "))
	}
	if len(excludeSet) > 0 {
		var excluded []string
		for r := range excludeSet {
			excluded = append(excluded, r)
		}
		printYellow("Excluding resources: %s", strings.Join(excluded, ", "))
	}

	// Create tf directory if it doesn't exist
	if err := os.MkdirAll(tfDir, permDir); err != nil {
		return fmt.Errorf("failed to create tf directory %s: %w", tfDir, err)
	}

	// Sync resource files from testdata
	printYellow("Syncing resource files from testdata...")
	fileCount := 0
	var moduleNames []string

	// Find all resource directories in testdata
	entries, err := os.ReadDir(testdataRoot)
	if err != nil {
		return fmt.Errorf("failed to read testdata directory: %w", err)
	}

	// Filter to target resources if specified, and apply exclusions
	var resourceDirs []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		resourceName := entry.Name()

		// Skip excluded resources
		if excludeSet[resourceName] {
			continue
		}

		// If specific resources requested, only include those
		if len(targetResources) > 0 {
			found := false
			for _, target := range targetResources {
				if resourceName == target {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		resourceDirs = append(resourceDirs, entry)
	}

	if len(resourceDirs) == 0 {
		if len(targetResources) > 0 {
			return fmt.Errorf("no matching resources found in testdata for: %s", strings.Join(targetResources, ", "))
		}
		return fmt.Errorf("no resources found in testdata directory: %s", testdataRoot)
	}

	// versionSuffix is the normalized from-version used to find version-specific overrides.
	// e.g. "5.1.0" → "_5_1_0"
	versionSuffix := "_" + strings.ReplaceAll(cfg.FromVersion, ".", "_")

	// Process each resource directory
	for _, entry := range resourceDirs {
		resourceName := entry.Name()
		moduleNames = append(moduleNames, resourceName)

		srcDir := filepath.Join(testdataRoot, resourceName)
		dstDir := filepath.Join(tfDir, resourceName)

		// Create/clean module directory
		if err := os.RemoveAll(dstDir); err != nil {
			return fmt.Errorf("failed to clean module directory %s: %w", dstDir, err)
		}
		if err := os.MkdirAll(dstDir, permDir); err != nil {
			return fmt.Errorf("failed to create module directory %s: %w", dstDir, err)
		}

		copiedFiles, err := syncModuleFiles(srcDir, dstDir, versionSuffix)
		if err != nil {
			return err
		}
		fileCount += copiedFiles

		// Generate versions.tf for the module with required_providers
		// This is critical - without explicit source, Terraform defaults to hashicorp/cloudflare
		versionsContent := `terraform {
  required_version = ">= 1.0"

  required_providers {
    cloudflare = {
      source = "cloudflare/cloudflare"
    }
  }
}
`
		versionsFile := filepath.Join(dstDir, "versions.tf")
		if err := os.WriteFile(versionsFile, []byte(versionsContent), permFile); err != nil {
			return fmt.Errorf("failed to write versions.tf for %s: %w", resourceName, err)
		}
		fileCount++

		printSuccess("  %s/ (%d files)", resourceName, copiedFiles+1)
	}

	printSuccess("Synced %d files for %d modules", fileCount, len(moduleNames))
	fmt.Println()

	// Generate provider.tf with source version constraint
	printYellow("Generating provider.tf with version %s...", cfg.FromVersion)
	versionConstraint := ResolveFromVersion(cfg.FromVersion)
	providerContent := GenerateProviderTF(versionConstraint)
	providerFile := filepath.Join(tfDir, "provider.tf")
	if err := os.WriteFile(providerFile, []byte(providerContent), permFile); err != nil {
		return fmt.Errorf("failed to write provider.tf: %w", err)
	}
	printSuccess("Generated provider.tf")

	// Generate backend.configured.hcl
	printYellow("Generating backend configuration...")
	backendContent := GenerateBackendConfiguredHCL(env.AccountID, cfg.FromVersion, cfg.ToVersion)
	backendFile := filepath.Join(tfDir, "backend.configured.hcl")
	if err := os.WriteFile(backendFile, []byte(backendContent), permFile); err != nil {
		return fmt.Errorf("failed to write backend.configured.hcl: %w", err)
	}
	stateKey := GenerateStateKey(cfg.FromVersion, cfg.ToVersion)
	printSuccess("Generated backend.configured.hcl (key: %s)", stateKey)

	// Generate main.tf with module references
	printYellow("Generating main.tf with module references...")
	sort.Strings(moduleNames)
	if err := validateModuleSpecificEnv(tfDir, moduleNames, env); err != nil {
		return err
	}
	mainContent, err := generateMainTF(tfDir, moduleNames)
	if err != nil {
		return fmt.Errorf("failed to generate main.tf content: %w", err)
	}
	mainFile := filepath.Join(tfDir, "main.tf")
	if err := os.WriteFile(mainFile, []byte(mainContent), permFile); err != nil {
		return fmt.Errorf("failed to write main.tf: %w", err)
	}
	printSuccess("Generated main.tf with %d modules", len(moduleNames))

	// Generate terraform.tfvars
	printYellow("Generating terraform.tfvars...")
	tfvarsContent := generateTFVars(env, cfg.FromVersion)
	tfvarsFile := filepath.Join(tfDir, "terraform.tfvars")
	if err := os.WriteFile(tfvarsFile, []byte(tfvarsContent), permFile); err != nil {
		return fmt.Errorf("failed to write terraform.tfvars: %w", err)
	}
	printSuccess("Generated terraform.tfvars")

	fmt.Println()
	printHeader("Initialization Complete")
	fmt.Println()

	printYellow("Test configuration:")
	printYellow("  From version: %s", cfg.FromVersion)
	printYellow("  To version:   %s", cfg.ToVersion)
	printYellow("  State key:    %s", stateKey)
	printYellow("  Modules:      %d", len(moduleNames))
	fmt.Println()

	printYellow("Next steps:")
	printYellow("  1. Run: e2e-runner v5-upgrade run --from-version %s --to-version %s", cfg.FromVersion, cfg.ToVersion)
	printYellow("  2. Or with local provider: e2e-runner v5-upgrade run --from-version %s --provider ../provider", cfg.FromVersion)
	fmt.Println()

	return nil
}

// generateMainTF generates the main.tf content with module references.
// It injects only variables that each module actually declares.
func generateMainTF(tfDir string, moduleNames []string) (string, error) {
	var sb strings.Builder
	sb.WriteString("# Auto-generated main.tf for v5 upgrade tests\n")
	sb.WriteString("# DO NOT EDIT - regenerated by 'e2e-runner v5-upgrade init'\n\n")

	for _, name := range moduleNames {
		moduleDir := filepath.Join(tfDir, name)
		moduleVars, err := discoverModuleVariables(moduleDir)
		if err != nil {
			return "", fmt.Errorf("discover module variables for %s: %w", name, err)
		}

		sb.WriteString(fmt.Sprintf("module %q {\n  source = %q\n", name, "./"+name))

		if moduleVars["cloudflare_account_id"] {
			sb.WriteString("\n  cloudflare_account_id = var.cloudflare_account_id")
		}
		if moduleVars["cloudflare_zone_id"] {
			sb.WriteString("\n  cloudflare_zone_id    = var.cloudflare_zone_id")
		}
		if moduleVars["cloudflare_domain"] {
			sb.WriteString("\n  cloudflare_domain     = var.cloudflare_domain")
		}
		if moduleVars["from_version"] {
			sb.WriteString("\n  from_version          = var.from_version")
		}
		if moduleVars["crowdstrike_client_id"] {
			sb.WriteString("\n  crowdstrike_client_id     = var.crowdstrike_client_id")
		}
		if moduleVars["crowdstrike_client_secret"] {
			sb.WriteString("\n  crowdstrike_client_secret = var.crowdstrike_client_secret")
		}
		if moduleVars["crowdstrike_api_url"] {
			sb.WriteString("\n  crowdstrike_api_url       = var.crowdstrike_api_url")
		}
		if moduleVars["crowdstrike_customer_id"] {
			sb.WriteString("\n  crowdstrike_customer_id   = var.crowdstrike_customer_id")
		}

		sb.WriteString("\n}\n\n")
	}

	return sb.String(), nil
}

// generateTFVars generates the terraform.tfvars content from environment
func generateTFVars(env *E2EEnv, fromVersion string) string {
	var sb strings.Builder
	sb.WriteString("# Auto-generated terraform.tfvars for v5 upgrade tests\n")
	sb.WriteString("# DO NOT EDIT - regenerated by 'e2e-runner v5-upgrade init'\n\n")

	sb.WriteString(fmt.Sprintf("cloudflare_account_id = %q\n", env.AccountID))
	sb.WriteString(fmt.Sprintf("cloudflare_zone_id    = %q\n", env.ZoneID))
	sb.WriteString(fmt.Sprintf("cloudflare_domain     = %q\n", env.Domain))
	sb.WriteString(fmt.Sprintf("from_version          = %q\n", fromVersion))
	sb.WriteString(fmt.Sprintf("crowdstrike_client_id     = %q\n", env.CrowdstrikeClientID))
	sb.WriteString(fmt.Sprintf("crowdstrike_client_secret = %q\n", env.CrowdstrikeClientSecret))
	sb.WriteString(fmt.Sprintf("crowdstrike_api_url       = %q\n", env.CrowdstrikeAPIURL))
	sb.WriteString(fmt.Sprintf("crowdstrike_customer_id   = %q\n", env.CrowdstrikeCustomerID))

	return sb.String()
}

func validateModuleSpecificEnv(tfDir string, moduleNames []string, env *E2EEnv) error {
	requiresCrowdstrike := false
	for _, name := range moduleNames {
		moduleDir := filepath.Join(tfDir, name)
		moduleVars, err := discoverModuleVariables(moduleDir)
		if err != nil {
			return fmt.Errorf("failed to discover variables for module %s: %w", name, err)
		}
		if moduleVars["crowdstrike_client_id"] || moduleVars["crowdstrike_client_secret"] || moduleVars["crowdstrike_api_url"] || moduleVars["crowdstrike_customer_id"] {
			requiresCrowdstrike = true
			break
		}
	}

	if !requiresCrowdstrike {
		return nil
	}

	if env.CrowdstrikeClientID == "" {
		return fmt.Errorf("CLOUDFLARE_CROWDSTRIKE_CLIENT_ID environment variable is required for selected resources")
	}
	if env.CrowdstrikeClientSecret == "" {
		return fmt.Errorf("CLOUDFLARE_CROWDSTRIKE_CLIENT_SECRET environment variable is required for selected resources")
	}
	if env.CrowdstrikeAPIURL == "" {
		return fmt.Errorf("CLOUDFLARE_CROWDSTRIKE_API_URL environment variable is required for selected resources")
	}
	if env.CrowdstrikeCustomerID == "" {
		return fmt.Errorf("CLOUDFLARE_CROWDSTRIKE_CUSTOMER_ID environment variable is required for selected resources")
	}

	return nil
}

// syncModuleFiles copies the default .tf files from srcDir to dstDir, substituting
// any file that has a version-specific override (e.g. page_rule_5_1_0.tf → page_rule.tf).
// Version-specific files are never written to the destination directly.
// Pass versionSuffix="" to always use the default files (i.e. for the target version).
func syncModuleFiles(srcDir, dstDir, versionSuffix string) (int, error) {
	allFiles, err := filepath.Glob(filepath.Join(srcDir, "*.tf"))
	if err != nil {
		return 0, fmt.Errorf("failed to glob tf files in %s: %w", srcDir, err)
	}

	// Build set of version-specific filenames so they are never copied directly.
	versionedFileNames := make(map[string]bool)
	if versionSuffix != "" {
		for _, f := range allFiles {
			base := filepath.Base(f)
			name := strings.TrimSuffix(base, ".tf")
			if strings.Contains(name, versionSuffix) {
				versionedFileNames[base] = true
			}
		}
	} else {
		// No specific version: skip all versioned files (anything matching _N_N_N pattern).
		for _, f := range allFiles {
			base := filepath.Base(f)
			name := strings.TrimSuffix(base, ".tf")
			// A versioned file has the form <base>_<d>_<d>_<d> where d are digits.
			// We detect this by checking whether the name contains a segment that
			// looks like a version suffix (_\d+_\d+_\d+).
			if isVersionedFileName(name) {
				versionedFileNames[base] = true
			}
		}
	}

	copiedFiles := 0
	for _, srcFile := range allFiles {
		base := filepath.Base(srcFile)

		// Skip version-specific files.
		if versionedFileNames[base] {
			continue
		}

		// When a versionSuffix is given, check for an override.
		if versionSuffix != "" {
			name := strings.TrimSuffix(base, ".tf")
			overridePath := filepath.Join(srcDir, name+versionSuffix+".tf")
			if _, err := os.Stat(overridePath); err == nil {
				srcFile = overridePath
			}
		}

		content, err := os.ReadFile(srcFile)
		if err != nil {
			return 0, fmt.Errorf("failed to read %s: %w", srcFile, err)
		}
		if err := os.WriteFile(filepath.Join(dstDir, base), content, permFile); err != nil {
			return 0, fmt.Errorf("failed to write %s: %w", filepath.Join(dstDir, base), err)
		}
		copiedFiles++
	}

	return copiedFiles, nil
}

// isVersionedFileName returns true if the filename stem ends with a version suffix
// of the form _<major>_<minor>_<patch> (e.g. "page_rule_5_1_0").
func isVersionedFileName(name string) bool {
	parts := strings.Split(name, "_")
	if len(parts) < 3 {
		return false
	}
	// Check if the last three segments are all numeric.
	for _, p := range parts[len(parts)-3:] {
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
		}
		if len(p) == 0 {
			return false
		}
	}
	return true
}

// resyncForTargetVersion re-copies testdata files into the tf module directories
// using the default (non-versioned) configs. This is called after the provider is
// upgraded so that configs with version-specific syntax are replaced with the
// current-provider-compatible equivalents before the post-upgrade plan runs.
//
// resourceDirs may be empty, in which case all subdirectories of testdataRoot are used.
func resyncForTargetVersion(testdataRoot, tfDir string, resourceDirs []string) error {
	// If no explicit resource list, derive it from the testdata directory.
	if len(resourceDirs) == 0 {
		entries, err := os.ReadDir(testdataRoot)
		if err != nil {
			return fmt.Errorf("failed to read testdata directory: %w", err)
		}
		for _, e := range entries {
			if e.IsDir() {
				resourceDirs = append(resourceDirs, e.Name())
			}
		}
	}

	printYellow("Resyncing configs to target-version syntax...")
	resynced := 0
	for _, resourceName := range resourceDirs {
		srcDir := filepath.Join(testdataRoot, resourceName)
		dstDir := filepath.Join(tfDir, resourceName)

		// Only resync modules where a versioned override actually exists — no-op otherwise.
		allFiles, err := filepath.Glob(filepath.Join(srcDir, "*.tf"))
		if err != nil {
			return fmt.Errorf("failed to glob tf files in %s: %w", srcDir, err)
		}
		hasVersionedFile := false
		for _, f := range allFiles {
			if isVersionedFileName(strings.TrimSuffix(filepath.Base(f), ".tf")) {
				hasVersionedFile = true
				break
			}
		}
		if !hasVersionedFile {
			continue
		}

		if _, err := syncModuleFiles(srcDir, dstDir, ""); err != nil {
			return err
		}
		resynced++
		printSuccess("  %s/ (resynced to default configs)", resourceName)
	}
	if resynced == 0 {
		printYellow("  No version-specific configs to resync")
	}
	return nil
}
