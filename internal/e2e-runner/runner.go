// Package e2e provides end-to-end testing infrastructure for Terraform provider migrations.
//
// This package implements a complete testing workflow that validates migrations between
// different versions of the Cloudflare Terraform provider (e.g., v4 to v5). The testing
// process includes:
//
//   - Initialization: Setting up test infrastructure and Terraform configurations
//   - V4 Application: Applying v4 provider configs to create real infrastructure
//   - Migration: Running the tf-migrate tool to convert v4 configs to v5
//   - V5 Validation: Applying v5 configs and verifying no unexpected changes
//   - Drift Detection: Comparing states and detecting infrastructure drift
//   - Cleanup: Managing remote state and test artifacts
//
// The package supports both full migrations and targeted resource testing, with
// configurable drift exemptions for known acceptable differences between provider versions.
package e2e

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// File permission constants for consistent permission management
const (
	permDir        = 0755 // rwxr-xr-x - directories
	permFile       = 0644 // rw-r--r-- - regular files
	permSecretFile = 0600 // rw------- - sensitive files (state, secrets)
)

// RunConfig holds configuration for e2e test run
type RunConfig struct {
	SkipV4Test            bool
	ApplyExemptions       bool
	NoRefreshSnapshot     bool
	Parallelism           int
	Resources             string
	Exclude               string // comma-separated resource names to exclude
	ProviderPath          string
	TargetProviderVersion string // explicit provider version to set in required_providers
	// VersionSuffix isolates this run's local directories (e2e/tf/v4-<suffix>,
	// e2e/migrated-v4_to_v5-<suffix>) and R2 state key
	// (v4/versions/<suffix>/terraform.tfstate) from the shared, unversioned
	// ones used by default (unsuffixed) runs. If empty and
	// TargetProviderVersion is set, it defaults to TargetProviderVersion —
	// pass an explicit value only to override that default. See
	// version_suffix.go.
	VersionSuffix string
	// Clean, when true, destroys this run's v4 and v5 test infrastructure on
	// exit — success or failure — via a deferred cleanup. Mirrors the
	// v5-upgrade track's --clean flag. Primarily for version-isolated
	// supportability-matrix runs against a shared test account/zone with
	// limited resources (ruleset phase slots, spectrum IPv4 quota, unique
	// hostnames, etc.), where leftover resources from one leg break the next.
	Clean bool
	// KeepCreated, when true, skips the automatic cleanup of resources this
	// run newly created with no prior real-world identity (no moved {} or
	// import {} block protecting them — see cleanupCreateOnlyResources).
	// That cleanup runs by default on every successful run specifically to
	// prevent the kind of unbounded duplicate accumulation documented in
	// e2e/SUPPORTABILITY_MATRIX.md §10 (the Access Policy and Gateway
	// Certificate quota incidents). Pass this only to deliberately inspect
	// freshly-created resources after a local debug run.
	KeepCreated bool
}

// testContext holds shared state for e2e test execution
type testContext struct {
	cfg          *RunConfig
	env          *E2EEnv
	repoRoot     string
	e2eRoot      string
	v4Dir        string
	v5Dir        string
	tmpDir       string
	targetArgs   []string
	resourceList []string
	tfConfigFile string

	// Drift tracking
	hasChanges               bool
	hasPostApplyChanges      bool
	hasNoRefreshChanges      bool
	v5InitialDrift           []string
	v5PostApplyDrift         []string
	v5NoRefreshDrift         []string
	v5InitialMaterial        int
	v5PostApplyMaterial      int
	v5NoRefreshMaterial      int
	v5InitialReal            int
	v5PostApplyReal          int
	v5NoRefreshReal          int
	v5InitialExempted        int
	v5PostApplyExempted      int
	v5NoRefreshExempted      int
	v5InitialExemptedLines   []string
	v5PostApplyExemptedLines []string
	v5NoRefreshExemptedLines []string

	// Output tracking
	v5PlanOutput     string
	v5PostPlanOutput string
}

// RunE2ETests executes the complete e2e test suite
func RunE2ETests(cfg *RunConfig) error {
	if cfg.Parallelism < 0 {
		return fmt.Errorf("parallelism must be >= 0, got %d", cfg.Parallelism)
	}

	// Default the version suffix from --target-provider-version when not
	// explicitly given, so a matrix run like
	// `run --target-provider-version 5.19.0 --resources ...` is isolated by
	// default without needing a second flag. Pass --version-suffix explicitly
	// to override. See version_suffix.go.
	if cfg.VersionSuffix == "" && cfg.TargetProviderVersion != "" {
		cfg.VersionSuffix = cfg.TargetProviderVersion
	}

	// Get paths
	repoRoot := getRepoRoot()
	e2eRoot := filepath.Join(repoRoot, "e2e")
	v4Dir := versionedV4Dir(e2eRoot, cfg.VersionSuffix)
	v5Dir := versionedV5Dir(e2eRoot, cfg.VersionSuffix)
	tmpDir := filepath.Join(e2eRoot, "tmp")
	if sanitizeVersionSuffix(cfg.VersionSuffix) != "" {
		tmpDir = filepath.Join(e2eRoot, "tmp-"+sanitizeVersionSuffix(cfg.VersionSuffix))
	}

	// Create tmp directory
	if err := os.MkdirAll(tmpDir, permDir); err != nil {
		return fmt.Errorf("failed to create tmp directory %s: %w", tmpDir, err)
	}

	// Apply --exclude filter: remove excluded resources from cfg.Resources
	if cfg.Exclude != "" {
		excludeSet := make(map[string]bool)
		for _, r := range strings.Split(cfg.Exclude, ",") {
			excludeSet[strings.TrimSpace(r)] = true
		}

		// Resolve the full resource list to filter against
		var base []string
		if cfg.Resources != "" {
			for _, r := range strings.Split(cfg.Resources, ",") {
				base = append(base, strings.TrimSpace(r))
			}
		} else {
			all, err := discoverAllResources()
			if err != nil {
				return fmt.Errorf("failed to discover resources for exclusion: %w", err)
			}
			base = all
		}

		var kept []string
		for _, r := range base {
			if !excludeSet[r] {
				kept = append(kept, r)
			}
		}

		excluded := []string{}
		for r := range excludeSet {
			excluded = append(excluded, r)
		}
		printYellow("Excluding resources: %s", strings.Join(excluded, ", "))

		cfg.Resources = strings.Join(kept, ",")
		printCyan("Remaining resources after exclusion: %s", cfg.Resources)
		fmt.Println()
	}

	// Build target arguments if resources specified
	var targetArgs []string
	var resourceList []string
	if cfg.Resources != "" {
		printCyan("Targeting specific resources: %s", cfg.Resources)
		resourceList = strings.Split(cfg.Resources, ",")
		for _, resource := range resourceList {
			resource = strings.TrimSpace(resource)
			targetArgs = append(targetArgs, "-target=module."+resource)
		}
		printCyan("Target arguments: %s", strings.Join(targetArgs, " "))
		fmt.Println()
	}

	printHeader("E2E Migration Test")

	// Step 0: Initialize test resources
	printYellow("Step 0: Initializing test resources")

	// Load required environment variables
	env, err := LoadEnv(EnvForRunner)
	if err != nil {
		return err
	}

	// tfConfigFile is set later (only if --provider is given), but the
	// cleanup closure below reads it by reference at defer-execution time
	// (function exit), not at defer-registration time (now) — so it's safe
	// to declare and register the defer before tfConfigFile has its final
	// value.
	var tfConfigFile string
	if cfg.Clean {
		defer func() {
			runCleanupOnExit(cfg, env, v4Dir, v5Dir, tfConfigFile)
		}()
		printYellow("Clean mode (--clean): test infrastructure will be destroyed on exit, success or failure")
		fmt.Println()
	}

	printYellow("Running tests with:")
	printYellow("  User:       %s", env.Email)
	printYellow("  Account ID: %s", env.AccountID)
	printYellow("  Zone ID:    %s", env.ZoneID)
	printYellow("  Domain:     %s", env.Domain)
	if cfg.ProviderPath != "" {
		printYellow("  Provider:   Local (%s)", cfg.ProviderPath)
	} else if cfg.TargetProviderVersion != "" {
		printYellow("  Provider:   Registry (exact version %s)", cfg.TargetProviderVersion)
	} else {
		printYellow("  Provider:   Registry (latest)")
	}
	if cfg.VersionSuffix != "" {
		printYellow("  Isolation:  v4Dir=%s v5Dir=%s stateKey=%s", v4Dir, v5Dir, versionedStateKey(cfg.VersionSuffix))
	}
	fmt.Println()

	printYellow("Running init script...")
	if err := RunInit(cfg.Resources, cfg.VersionSuffix); err != nil {
		printError("Init script failed")
		return err
	}
	printSuccess("Test resources initialized")
	fmt.Println()

	// Set up local provider if specified (tfConfigFile declared earlier,
	// above, so the cleanup defer can capture it by reference)
	if cfg.ProviderPath != "" {
		printHeader("Setting up local provider")
		printYellow("Using provider from: %s", cfg.ProviderPath)

		// Determine if ProviderPath is a file or directory
		var providerBinary string
		var providerDir string

		info, err := os.Stat(cfg.ProviderPath)
		if err != nil {
			// Path doesn't exist - assume it's a binary path and extract directory
			if os.IsNotExist(err) {
				providerBinary = cfg.ProviderPath
				providerDir = filepath.Dir(cfg.ProviderPath)

				// Verify the directory exists
				if dirInfo, dirErr := os.Stat(providerDir); dirErr != nil || !dirInfo.IsDir() {
					return fmt.Errorf("provider directory does not exist: %s", providerDir)
				}
			} else {
				return fmt.Errorf("failed to check provider path: %w", err)
			}
		} else if info.IsDir() {
			// ProviderPath is a directory, expect binary inside
			providerDir = cfg.ProviderPath
			providerBinary = filepath.Join(providerDir, "terraform-provider-cloudflare")
		} else {
			// ProviderPath is the binary file itself, extract directory
			providerBinary = cfg.ProviderPath
			providerDir = filepath.Dir(cfg.ProviderPath)
		}

		// Always rebuild the provider to ensure latest code is used
		printYellow("Building provider...")
		printYellow("  Building in: %s", providerDir)
		printYellow("  Output: %s", providerBinary)

		// Build the provider
		buildCmd := exec.Command("go", "build", "-o", providerBinary, ".")
		buildCmd.Dir = providerDir
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr

		if err := buildCmd.Run(); err != nil {
			printError("Failed to build provider: %v", err)
			return fmt.Errorf("failed to build provider: %w", err)
		}

		printSuccess("Provider built successfully: %s", providerBinary)

		// Create dev overrides config - use absolute directory path
		tfConfigFile = filepath.Join(repoRoot, ".terraformrc-tf-migrate")

		// Convert to absolute path to avoid issues when running from subdirectories
		absProviderDir, err := filepath.Abs(providerDir)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for provider directory: %w", err)
		}

		configContent := fmt.Sprintf(`provider_installation {
  dev_overrides {
    "cloudflare/cloudflare" = "%s"
  }

  # For all other providers, install them directly as normal.
  direct {}
}
`, absProviderDir)

		if err := os.WriteFile(tfConfigFile, []byte(configContent), permFile); err != nil {
			return fmt.Errorf("failed to create provider config at %s: %w", tfConfigFile, err)
		}

		printSuccess("Created dev overrides config: %s", tfConfigFile)
		printSuccess("Local provider will be used for v5 testing")
		fmt.Println()
		printYellow("Note: v4 tests will use the registry provider (v4.x)")
		printYellow("      v5 tests will use the local provider with dev overrides")
		fmt.Println()
	}

	// Create test context for shared state
	ctx := &testContext{
		cfg:          cfg,
		env:          env,
		repoRoot:     repoRoot,
		e2eRoot:      e2eRoot,
		v4Dir:        v4Dir,
		v5Dir:        v5Dir,
		tmpDir:       tmpDir,
		targetArgs:   targetArgs,
		resourceList: resourceList,
		tfConfigFile: tfConfigFile,
	}

	// Step 1: Test v4 configurations
	if !cfg.SkipV4Test {
		if err := runV4Tests(ctx); err != nil {
			return err
		}
	} else {
		printCyan("Step 1: Skipped v4 testing (--skip-v4-test)")
		fmt.Println()
	}

	// Step 2: Run migration
	fmt.Println()
	printCyan("Step 2: Running migration")
	printYellow("Running ./scripts/migrate...")

	// Run the full migration directly (--skip-phase-check bypasses phased migration detection).
	// The e2e runner handles state cleanup itself below via terraform state rm,
	// which is simpler and reliable. The phased migration (_phase1_cleanup.tf)
	// is for real Atlantis users who cannot run terraform state rm.
	if err := RunMigrate(cfg.Resources, true, cfg.TargetProviderVersion, cfg.VersionSuffix); err != nil {
		printError("Migration failed")
		return err
	}
	printSuccess("Migration successful")

	// Remove state entries for resource types the v5 provider has no schema for.
	// cloudflare_zone_settings_override does not exist in v5 — attempting a plan
	// with these entries in state produces schema errors.
	//
	// We manipulate the local state JSON file directly rather than running
	// `terraform state rm`, which requires `terraform init` to have been run
	// first (modules must be installed). Direct JSON manipulation works on the
	// local state file before init, avoiding the "Module not installed" error.
	// (Real Atlantis users use the _phase1_cleanup.tf phased approach instead.)
	obsoleteTypes := map[string]bool{
		"cloudflare_zone_settings_override":  true,
		"cloudflare_access_policy":           true, // Application-scoped policies with application_id cannot be migrated; removed{} blocks handle state cleanup
		"cloudflare_split_tunnel":            true, // Dissolved into device profile exclude/include attributes in v5
		"cloudflare_zero_trust_split_tunnel": true, // Newer v4 name for split_tunnel — also dissolved in v5
		"cloudflare_workers_secret":          true, // Folded into workers_script bindings in v5
		"cloudflare_worker_secret":           true, // Deprecated singular form — also folded into workers_script bindings
	}
	// Some instances of an "obsolete" type are actually just being RENAMED
	// (e.g. non-application-scoped cloudflare_access_policy -> cloudflare_zero_trust_access_policy
	// via a moved {} block), not truly removed. Blanket-deleting by type name would strip
	// their state entry before the moved {} block ever runs, making Terraform see the new
	// address as brand new and plan to recreate it. Scan the migrated config for moved {}
	// blocks and protect any state entry that has one from this cleanup.
	protectedAddrs, err := findMovedFromAddresses(v5Dir)
	if err != nil {
		printYellow("Warning: failed to scan for moved {} blocks: %v", err)
		protectedAddrs = map[string]bool{}
	}
	stateFilePath := filepath.Join(v5Dir, "terraform.tfstate")
	if removed, skipped, err := removeObsoleteStateEntries(stateFilePath, obsoleteTypes, protectedAddrs); err != nil {
		printYellow("Warning: failed to clean obsolete state entries: %v", err)
	} else {
		for _, addr := range removed {
			printYellow("Removing obsolete state entry (no v5 schema): %s", addr)
		}
		for _, addr := range skipped {
			printYellow("Keeping state entry (protected by moved {} block): %s", addr)
		}
	}

	// Step 3: Test v5 configurations
	fmt.Println()
	printCyan("Step 3: Testing v5 configurations")
	printYellow("Running terraform init in migrated-v4_to_v5/...")

	// Clean .terraform and .terraform.lock.hcl to ensure dev_overrides are used
	v5TFDir := filepath.Join(v5Dir, ".terraform")
	if _, err := os.Stat(v5TFDir); err == nil {
		printYellow("Cleaning v5 .terraform directory for fresh init...")
		if err := os.RemoveAll(v5TFDir); err != nil {
			return fmt.Errorf("failed to remove v5 .terraform directory %s: %w", v5TFDir, err)
		}
	}

	// Remove lock file so dev_overrides work correctly
	v5LockFile := filepath.Join(v5Dir, ".terraform.lock.hcl")
	if _, err := os.Stat(v5LockFile); err == nil {
		printYellow("Removing .terraform.lock.hcl to allow dev_overrides...")
		if err := os.Remove(v5LockFile); err != nil {
			return fmt.Errorf("failed to remove lock file %s: %w", v5LockFile, err)
		}
	}

	v5TF := NewTerraformRunner(v5Dir)
	if tfConfigFile != "" {
		v5TF.TFConfigFile = tfConfigFile
	}
	v5TF.EnvVars["TF_VAR_account_id"] = os.Getenv("CLOUDFLARE_ACCOUNT_ID")

	// Initialize v5
	v5InitArgs := []string{"init", "-no-color", "-input=false"}
	if err := v5TF.RunToFile(filepath.Join(tmpDir, "v5-init.log"), v5InitArgs...); err != nil {
		printError("Terraform init failed for v5")
		fmt.Println()
		printRed("Error output:")
		content, _ := os.ReadFile(filepath.Join(tmpDir, "v5-init.log"))
		fmt.Println(string(content))
		return err
	}
	printSuccess("Terraform init successful")

	// When no local --provider was given, this run is expected to install
	// the provider straight from the public Terraform Registry, pinned to
	// exactly cfg.TargetProviderVersion (tf-migrate's own
	// --target-provider-version rewrite already makes required_providers use
	// an exact, unqualified version string). Verify that's actually what got
	// installed rather than assuming it — don't just check out and build the
	// provider from source, and don't silently trust a version constraint
	// that might have resolved to something else.
	if cfg.ProviderPath == "" && cfg.TargetProviderVersion != "" {
		if err := verifyRegistryProviderVersion(v5Dir, cfg.TargetProviderVersion); err != nil {
			printError("%v", err)
			return err
		}
		printSuccess("Confirmed registry install: cloudflare/cloudflare v%s (no local build, no dev_overrides)", strings.TrimPrefix(cfg.TargetProviderVersion, "v"))
	}

	// Optional diagnostic snapshot before refresh
	if cfg.NoRefreshSnapshot {
		printYellow("Running terraform plan in v5/ (without refresh)...")
		v5NoRefreshPlanArgs := append([]string{"plan", "-no-color", "-refresh=false", "-input=false"}, targetArgs...)
		v5NoRefreshPlanArgs = addParallelismArg(v5NoRefreshPlanArgs, cfg.Parallelism)
		ctx.v5PlanOutput, err = v5TF.Run(v5NoRefreshPlanArgs...)
		if err != nil {
			printError("Terraform plan failed for v5 (without refresh)")
			fmt.Println()
			printRed("Error output:")
			fmt.Println(ctx.v5PlanOutput)
			return err
		}

		v5NoRefreshPlanLog := filepath.Join(tmpDir, "v5-plan-no-refresh.log")
		if err := os.WriteFile(v5NoRefreshPlanLog, []byte(ctx.v5PlanOutput), permFile); err != nil {
			printYellow("Warning: Failed to save v5 no-refresh plan log to %s: %v", v5NoRefreshPlanLog, err)
		}

		noRefreshDriftResult := checkAndDisplayDrift(ctx.v5PlanOutput, cfg, "initial-no-refresh", ctx.resourceList)
		ctx.hasNoRefreshChanges = noRefreshDriftResult.hasDrift
		ctx.v5NoRefreshDrift = noRefreshDriftResult.driftLines
		ctx.v5NoRefreshMaterial = noRefreshDriftResult.materialCount
		ctx.v5NoRefreshReal = noRefreshDriftResult.realCount
		ctx.v5NoRefreshExempted = noRefreshDriftResult.exemptedCount
		ctx.v5NoRefreshExemptedLines = noRefreshDriftResult.exemptedLines

		fmt.Println()
	}
	printYellow("Running terraform plan in v5/...")
	v5PlanArgs := append([]string{"plan", "-no-color", "-out=" + filepath.Join(tmpDir, "v5.tfplan"), "-input=false"}, targetArgs...)
	v5PlanArgs = addParallelismArg(v5PlanArgs, cfg.Parallelism)
	ctx.v5PlanOutput, err = v5TF.Run(v5PlanArgs...)
	if err != nil {
		printError("Terraform plan failed for v5")
		fmt.Println()
		printRed("Error output:")
		fmt.Println(ctx.v5PlanOutput)
		return err
	}

	// Save v5 plan output for debugging
	v5PlanLog := filepath.Join(tmpDir, "v5-plan.log")
	if err := os.WriteFile(v5PlanLog, []byte(ctx.v5PlanOutput), permFile); err != nil {
		printYellow("Warning: Failed to save v5 plan log to %s: %v", v5PlanLog, err)
	}

	// Check if plan shows changes
	driftResult := checkAndDisplayDrift(ctx.v5PlanOutput, cfg, "initial", ctx.resourceList)
	ctx.hasChanges = driftResult.hasDrift
	ctx.v5InitialDrift = driftResult.driftLines
	ctx.v5InitialMaterial = driftResult.materialCount
	ctx.v5InitialReal = driftResult.realCount
	ctx.v5InitialExempted = driftResult.exemptedCount
	ctx.v5InitialExemptedLines = driftResult.exemptedLines

	// Apply v5
	printYellow("Running terraform apply in v5/...")
	v5ApplyArgs := []string{"apply", "-no-color", "-auto-approve", "-input=false"}
	v5ApplyArgs = addParallelismArg(v5ApplyArgs, cfg.Parallelism)
	v5ApplyArgs = append(v5ApplyArgs, filepath.Join(tmpDir, "v5.tfplan"))
	v5ApplyOutput, err := v5TF.Run(v5ApplyArgs...)
	if err != nil {
		printError("Terraform apply failed for v5")
		fmt.Println()
		printRed("Error output:")
		fmt.Println(v5ApplyOutput)
		return err
	}

	// Save v5 apply output for debugging
	v5ApplyLog := filepath.Join(tmpDir, "v5-apply.log")
	if err := os.WriteFile(v5ApplyLog, []byte(v5ApplyOutput), permFile); err != nil {
		printYellow("Warning: Failed to save v5 apply log to %s: %v", v5ApplyLog, err)
	}
	printSuccess("Terraform apply successful")

	// Capture v5 state snapshot for debugging.
	// This is non-fatal: terraform show -json can fail if any state entry has a
	// schema version mismatch (e.g. v0 in state vs v500 in provider). The provider's
	// UpgradeState runs during plan/apply, not during show, so unvisited resources
	// may still have the old schema_version in state at this point.
	printYellow("Capturing v5 state...")
	v5StateOutput, err := v5TF.Run("show", "-no-color", "-json")
	if err != nil {
		// Extract the resource name from the error message for a cleaner warning.
		// Error format: "schema version 0 for <resource> in state does not match version 500"
		errMsg := err.Error()
		resource := ""
		if i := strings.Index(errMsg, "schema version 0 for "); i >= 0 {
			rest := errMsg[i+len("schema version 0 for "):]
			if j := strings.Index(rest, " in state"); j >= 0 {
				resource = rest[:j]
			}
		}
		if resource != "" {
			printYellow("Warning: Skipping v5 state snapshot — %s has not been visited by this apply yet (schema upgrade pending)", resource)
		} else {
			printYellow("Warning: Skipping v5 state snapshot — schema version mismatch for an unvisited resource")
		}
	} else {
		v5StateLog := filepath.Join(tmpDir, "v5-state.json")
		if err := os.WriteFile(v5StateLog, []byte(v5StateOutput), permFile); err != nil {
			printYellow("Warning: Failed to save v5 state to %s: %v", v5StateLog, err)
		} else {
			printSuccess("Saved v5 state to tmp/v5-state.json")
		}
	}

	// Step 4: Verify stable state
	fmt.Println()
	printCyan("Step 4: Verifying stable state (v5 plan after apply)")
	printYellow("Running terraform plan again to check for ongoing drift...")

	v5PostPlanArgs := append([]string{"plan", "-no-color", "-out=" + filepath.Join(tmpDir, "v5-post-apply.tfplan"), "-input=false"}, targetArgs...)
	v5PostPlanArgs = addParallelismArg(v5PostPlanArgs, cfg.Parallelism)
	ctx.v5PostPlanOutput, err = v5TF.Run(v5PostPlanArgs...)
	if err != nil {
		printError("Terraform plan failed for v5 (post-apply)")
		fmt.Println()
		printRed("Error output:")
		fmt.Println(ctx.v5PostPlanOutput)
		return err
	}

	// Save post-apply plan output for debugging
	v5PostPlanLog := filepath.Join(tmpDir, "v5-post-apply-plan.log")
	if err := os.WriteFile(v5PostPlanLog, []byte(ctx.v5PostPlanOutput), permFile); err != nil {
		printYellow("Warning: Failed to save v5 post-apply plan log to %s: %v", v5PostPlanLog, err)
	}

	// Check for ongoing drift
	postDriftResult := checkAndDisplayDrift(ctx.v5PostPlanOutput, cfg, "post-apply", ctx.resourceList)
	ctx.hasPostApplyChanges = postDriftResult.hasDrift
	ctx.v5PostApplyDrift = postDriftResult.driftLines
	ctx.v5PostApplyMaterial = postDriftResult.materialCount
	ctx.v5PostApplyReal = postDriftResult.realCount
	ctx.v5PostApplyExempted = postDriftResult.exemptedCount
	ctx.v5PostApplyExemptedLines = postDriftResult.exemptedLines

	// Display drift report if there were real changes OR exempted changes
	fmt.Println()
	hasExemptedChanges := len(ctx.v5InitialExemptedLines) > 0 || len(ctx.v5PostApplyExemptedLines) > 0
	if ctx.hasChanges || ctx.hasPostApplyChanges || hasExemptedChanges {
		printHeader("Drift Report")

		if ctx.hasChanges && len(ctx.v5InitialDrift) > 0 {
			printYellow("Real drift detected in v5 plan (before apply):")
			displayGroupedDrift(ctx.v5InitialDrift)
			fmt.Println()

			// Show affected resources
			affectedResources := extractAffectedResources(ctx.v5PlanOutput)
			if len(affectedResources) > 0 {
				printYellow("Affected Resources:")
				for _, resource := range affectedResources {
					printYellow("  - %s", resource)
				}
				fmt.Println()
			}
		}

		if len(ctx.v5InitialExemptedLines) > 0 {
			printSuccess("Exempted changes in v5 plan (before apply):")
			printYellow("The following changes were detected but exempted by drift exemption rules:")
			displayGroupedDrift(ctx.v5InitialExemptedLines)
			fmt.Println()
		}

		if ctx.hasPostApplyChanges && len(ctx.v5PostApplyDrift) > 0 {
			printYellow("Ongoing drift detected in v5 plan (after apply):")
			displayGroupedDrift(ctx.v5PostApplyDrift)
			fmt.Println()

			// Show affected resources
			affectedResources := extractAffectedResources(ctx.v5PostPlanOutput)
			if len(affectedResources) > 0 {
				printYellow("Affected Resources:")
				for _, resource := range affectedResources {
					printYellow("  - %s", resource)
				}
				fmt.Println()
			}
		}

		if len(ctx.v5PostApplyExemptedLines) > 0 {
			printSuccess("Exempted changes in v5 plan (after apply):")
			printYellow("The following changes were detected but exempted by drift exemption rules:")
			displayGroupedDrift(ctx.v5PostApplyExemptedLines)
			fmt.Println()
		}
	}

	// Summary at the END
	fmt.Println()
	if ctx.hasPostApplyChanges || ctx.hasChanges {
		printHeader("✗ E2E Test Failed!")
	} else {
		printHeader("✓ E2E Test Complete!")
	}

	printYellow("Summary:")
	printYellow("")
	printYellow("  Step 1: v4 terraform apply")
	printYellow("    Status: %s", colorGreen+"✓ SUCCESS"+colorReset)
	printYellow("")

	printYellow("  Step 2: Migration (v4 → v5)")
	printYellow("    Status: %s", colorGreen+"✓ SUCCESS"+colorReset)
	printYellow("")

	printYellow("  Step 3: v5 plan (before apply)")
	if cfg.NoRefreshSnapshot && (ctx.v5NoRefreshMaterial > 0 || ctx.hasNoRefreshChanges || ctx.v5NoRefreshExempted > 0) {
		printYellow("    No-refresh snapshot (diagnostic only): %d material changes (%d matched exemptions, %d unmatched)", ctx.v5NoRefreshMaterial, ctx.v5NoRefreshExempted, ctx.v5NoRefreshReal)
	}
	v5PlanSummary := extractPlanSummary(ctx.v5PlanOutput)
	if v5PlanSummary == "" {
		printYellow("    Status: %s", colorGreen+"✓ No changes needed"+colorReset)
	} else {
		if ctx.hasChanges {
			uniqueDrifts := countUniqueDrifts(ctx.v5InitialDrift)
			printYellow("    Status: %s", colorRed+"✗ FAILED - Migration produced drift"+colorReset)
			if ctx.v5InitialExempted > 0 {
				printYellow("    Result: %d real changes detected (%d exempted)", uniqueDrifts, ctx.v5InitialExempted)
			} else {
				printYellow("    Result: %d real changes detected", uniqueDrifts)
			}
			printYellow("    Terraform: %s", v5PlanSummary)
		} else if ctx.v5InitialMaterial > 0 {
			printYellow("    Status: %s", colorGreen+"✓ SUCCESS - Drift matched exemptions"+colorReset)
			printYellow("    Result: %d material changes (%d matched exemptions, %d unmatched)", ctx.v5InitialMaterial, ctx.v5InitialExempted, ctx.v5InitialReal)
			printYellow("    Terraform: %s", v5PlanSummary)
		} else {
			printYellow("    Status: %s", colorGreen+"✓ No material changes"+colorReset)
			printYellow("    Result: 0 material changes")
			printYellow("    Terraform: %s", v5PlanSummary)
		}
	}
	printYellow("")

	printYellow("  Step 4: v5 terraform apply")
	if ctx.hasChanges {
		printYellow("    Status: %s", colorRed+"✗ FAILED - Applied drift changes"+colorReset)
	} else {
		printYellow("    Status: %s", colorGreen+"✓ SUCCESS"+colorReset)
	}
	printYellow("")

	printYellow("  Step 5: v5 plan (after apply)")
	if ctx.hasPostApplyChanges {
		uniqueDrifts := countUniqueDrifts(ctx.v5PostApplyDrift)
		printYellow("    Status: %s", colorRed+"✗ FAILED - Resources keep changing"+colorReset)
		if ctx.v5PostApplyExempted > 0 {
			printYellow("    Result: %d ongoing drift patterns (%d exempted)", uniqueDrifts, ctx.v5PostApplyExempted)
		} else {
			printYellow("    Result: %d ongoing drift patterns", uniqueDrifts)
		}
		postPlanSummary := extractPlanSummary(ctx.v5PostPlanOutput)
		if postPlanSummary != "" {
			printYellow("    Terraform: %s", postPlanSummary)
		}
	} else {
		if ctx.v5PostApplyMaterial > 0 {
			printYellow("    Status: %s", colorGreen+"✓ SUCCESS - Stable state achieved"+colorReset)
			printYellow("    Result: %d material changes matched exemptions (%d unmatched)", ctx.v5PostApplyMaterial, ctx.v5PostApplyReal)
		} else {
			printYellow("    Status: %s", colorGreen+"✓ SUCCESS - Stable state achieved"+colorReset)
			printYellow("    Result: No changes detected")
		}
	}

	fmt.Println()
	printYellow("Logs saved to:")
	printCyan("  - %s", tmpDir)
	fmt.Println()

	// Exit with error if there's ongoing drift or if there were real changes in first v5 plan
	if ctx.hasPostApplyChanges {
		printRed("Test failed: Resources are unstable and keep changing")
		printYellow("This prevents safe migration to v5 - likely a provider bug")
		return fmt.Errorf("ongoing drift detected - resources keep changing")
	}

	if ctx.hasChanges {
		printRed("Test failed: Migration produced drift")
		printYellow("The migrated v5 config doesn't match your infrastructure")
		printYellow("Review the changes above and check for migration tool bugs")
		return fmt.Errorf("migration produced drift")
	}

	// Step 5: Clean up create-only resources (see cleanupCreateOnlyResources).
	// Only reached on this fully-successful path — a failed run's resources
	// are left in place for debugging, and --keep-created opts out entirely.
	if !cfg.KeepCreated {
		fmt.Println()
		printCyan("Step 5: Cleaning up create-only test resources")
		destroyed, cleanupErr := cleanupCreateOnlyResources(v5TF, v5Dir, ctx.v5PlanOutput, tmpDir, cfg)
		if cleanupErr != nil {
			printYellow("Warning: cleanup did not fully succeed: %v", cleanupErr)
			printYellow("This does not affect the test result above, which already passed.")
			printYellow("See %s for details. If this keeps recurring for the same resource,", filepath.Join(tmpDir, "v5-cleanup-destroy.log"))
			printYellow("it may need to be excluded from cleanup or investigated manually.")
		} else if len(destroyed) == 0 {
			printSuccess("No create-only resources needed cleanup")
		} else {
			printSuccess("Destroyed %d create-only resource(s):", len(destroyed))
			for _, addr := range destroyed {
				printYellow("  - %s", addr)
			}
		}
	}

	return nil
}

// runCleanupOnExit destroys whatever test infrastructure this run created,
// in both the v5 (migrated) and v4 directories, regardless of whether the
// run succeeded or failed. Invoked via defer when cfg.Clean is set (see
// RunE2ETests). Cleanup failures are logged as warnings only — they never
// override or mask the original run's result, since the caller's return
// value was already determined before this defer runs.
//
// This exists for version-isolated supportability-matrix runs against a
// shared test account/zone with resources the Cloudflare API treats as
// account/zone-wide singletons or quota-limited (one ruleset per phase per
// zone, a fixed spectrum IPv4 quota, unique hostnames) — leftovers from one
// leg silently break the next leg's create step in ways no amount of
// resource-name namespacing can fix.
func runCleanupOnExit(cfg *RunConfig, env *E2EEnv, v4Dir, v5Dir, tfConfigFile string) {
	printHeader("Cleanup (--clean): destroying test infrastructure")

	// Destroy the v5 (migrated) side first — this is what's actually live if
	// migration + v5 apply got far enough to create anything.
	if _, err := os.Stat(v5Dir); err == nil {
		v5TF := NewTerraformRunner(v5Dir)
		if tfConfigFile != "" {
			v5TF.TFConfigFile = tfConfigFile
		}
		v5TF.EnvVars["TF_VAR_account_id"] = env.AccountID

		if _, err := os.Stat(filepath.Join(v5Dir, ".terraform")); err != nil {
			if _, initErr := v5TF.Run("init", "-no-color", "-input=false"); initErr != nil {
				printYellow("Warning: cleanup init failed for v5 dir %s: %v", v5Dir, initErr)
			}
		}
		if out, destroyErr := v5TF.Run("destroy", "-auto-approve", "-no-color", "-input=false"); destroyErr != nil {
			printYellow("Warning: cleanup destroy failed for v5 dir %s: %v", v5Dir, destroyErr)
			printYellow(out)
		} else {
			printSuccess("Destroyed v5 test infrastructure in %s", v5Dir)
		}
	} else {
		printYellow("Skipping v5 cleanup — %s does not exist", v5Dir)
	}

	// Destroy the v4 side too, in case anything was created and recorded in
	// v4 remote state but never carried through migration (e.g. a resource
	// failed partway through the v4 apply before migration ran).
	if _, err := os.Stat(v4Dir); err == nil {
		r2AccessKey := os.Getenv("CLOUDFLARE_R2_ACCESS_KEY_ID")
		r2SecretKey := os.Getenv("CLOUDFLARE_R2_SECRET_ACCESS_KEY")
		if r2AccessKey == "" || r2SecretKey == "" {
			printYellow("Warning: skipping v4 cleanup — R2 credentials not set")
		} else {
			v4TF := NewTerraformRunner(v4Dir)
			v4TF.EnvVars["AWS_ACCESS_KEY_ID"] = r2AccessKey
			v4TF.EnvVars["AWS_SECRET_ACCESS_KEY"] = r2SecretKey
			v4TF.EnvVars["TF_VAR_account_id"] = env.AccountID

			backendConfig := filepath.Join(v4Dir, "backend.hcl")
			backendConfigTmp := filepath.Join(v4Dir, "backend.configured.hcl")
			backendContent, readErr := os.ReadFile(backendConfig)
			if readErr != nil {
				printYellow("Warning: skipping v4 cleanup — failed to read %s: %v", backendConfig, readErr)
			} else {
				configuredContent := strings.ReplaceAll(string(backendContent), "ACCOUNT_ID", env.AccountID)
				if cfg.VersionSuffix != "" {
					isolatedKey := versionedStateKey(cfg.VersionSuffix)
					configuredContent = strings.ReplaceAll(configuredContent, `key    = "v4/terraform.tfstate"`, `key    = "`+isolatedKey+`"`)
				}

				if writeErr := os.WriteFile(backendConfigTmp, []byte(configuredContent), permFile); writeErr != nil {
					printYellow("Warning: skipping v4 cleanup — failed to write backend config: %v", writeErr)
				} else {
					defer func() {
						if err := os.Remove(backendConfigTmp); err != nil && !os.IsNotExist(err) {
							printYellow("Warning: failed to remove temp backend config %s: %v", backendConfigTmp, err)
						}
					}()

					if _, initErr := v4TF.Run("init", "-no-color", "-reconfigure", "-input=false", "-backend-config="+backendConfigTmp); initErr != nil {
						printYellow("Warning: cleanup init failed for v4 dir %s: %v", v4Dir, initErr)
					} else if out, destroyErr := v4TF.Run("destroy", "-auto-approve", "-no-color", "-input=false"); destroyErr != nil {
						printYellow("Warning: cleanup destroy failed for v4 dir %s: %v", v4Dir, destroyErr)
						printYellow(out)
					} else {
						printSuccess("Destroyed v4 test infrastructure in %s", v4Dir)
					}
				}
			}
		}
	} else {
		printYellow("Skipping v4 cleanup — %s does not exist", v4Dir)
	}

	printHeader("Cleanup complete")
}

// driftCheckResult holds the result of checking and displaying drift
type driftCheckResult struct {
	hasDrift      bool
	driftLines    []string
	exemptedCount int
	exemptedLines []string
	computedLines []string
	materialCount int
	realCount     int
}

// checkAndDisplayDrift checks for drift in plan output and displays results
// Returns true if real drift was detected (excluding exemptions)
func checkAndDisplayDrift(planOutput string, cfg *RunConfig, stage string, resourceFilter []string) driftCheckResult {
	result := driftCheckResult{}
	isNoRefreshDiagnostic := stage == "initial-no-refresh"

	if strings.Contains(planOutput, "No changes") {
		if isNoRefreshDiagnostic {
			printSuccess("No-refresh snapshot: no changes")
		} else if stage == "initial" {
			printSuccess("Terraform plan shows no changes (expected)")
		} else {
			printSuccess("No ongoing drift detected - migration achieved stable state!")
		}
		return result
	}

	// Check if only computed changes when --apply-exemptions is set
	if cfg.ApplyExemptions {
		driftResult := checkDrift(planOutput, resourceFilter)

		// Calculate total exempted count
		totalExempted := 0
		for _, count := range driftResult.TriggeredExemptions {
			totalExempted += count
		}
		result.exemptedCount = totalExempted
		result.exemptedLines = driftResult.ExemptedDriftLines
		result.computedLines = driftResult.ComputedRefreshLines
		result.realCount = len(driftResult.RealDriftLines)
		result.materialCount = len(driftResult.RealDriftLines) + len(driftResult.ExemptedDriftLines)

		if len(driftResult.ComputedRefreshLines) > 0 || totalExempted > 0 || len(driftResult.RealDriftLines) > 0 {
			printYellow("Drift breakdown: %d material, %d computed refresh, %d exempted", result.materialCount, len(driftResult.ComputedRefreshLines), totalExempted)
		}

		if driftResult.OnlyComputedChanges {
			if isNoRefreshDiagnostic {
				printSuccess("No-refresh snapshot shows only computed value refreshes")
			} else if stage == "initial" {
				printSuccess("Terraform plan shows only computed value refreshes (ignored with --apply-exemptions)")
			} else {
				printSuccess("Only computed value refreshes detected (ignored with --apply-exemptions) - migration achieved stable state!")
			}

			// Log triggered exemptions
			if driftResult.ExemptionsEnabled && len(driftResult.TriggeredExemptions) > 0 {
				fmt.Println()
				printYellow("Drift exemptions applied:")
				for exemptionName, count := range driftResult.TriggeredExemptions {
					printYellow("  - %s: %d change(s) exempted", exemptionName, count)
				}
			}
			return result
		}

		// Real drift detected
		result.hasDrift = true
		result.driftLines = driftResult.RealDriftLines

		if isNoRefreshDiagnostic {
			printYellow("No-refresh snapshot detected drift (diagnostic only; refresh plan is authoritative)")
		} else if stage == "initial" {
			printRed("⚠ Migration produced drift - v5 config wants to make changes")
		} else {
			printRed("✗ Ongoing drift detected - resources keep changing")
		}

		planSummary := extractPlanSummary(planOutput)
		if planSummary != "" {
			fmt.Println("  " + planSummary)
		}
		if !isNoRefreshDiagnostic {
			fmt.Println()

			// Show detailed changes
			if stage == "initial" {
				printYellow("Detailed changes:")
			} else {
				printYellow("Detailed ongoing drift:")
			}
			fmt.Println()
			fmt.Println(extractPlanChanges(planOutput))
			fmt.Println()

			// Explain what this means
			printRed("What this means:")
			if stage == "initial" {
				printRed("  The migrated v5 config doesn't match your existing infrastructure.")
				printRed("  This indicates the migration may not be correct.")
			} else {
				printRed("  Your resources are unstable - they change with every apply.")
				printRed("  This is a serious issue that prevents using v5 in production.")
			}
			fmt.Println()

			// Show affected resources
			affectedResources := extractAffectedResources(planOutput)
			if len(affectedResources) > 0 {
				printYellow("Affected Resources:")
				for _, resource := range affectedResources {
					printYellow("  - %s", resource)
				}
			}

			printYellow("")
			printYellow("Next steps:")
			if stage == "initial" {
				printYellow("  1. Review the changes above")
				printYellow("  2. Check if the migration tool has a bug")
				printYellow("  3. Consider using --apply-exemptions if changes are expected")
			} else {
				printYellow("  1. This is likely a provider or migration tool bug")
				printYellow("  2. Review the changes above to understand what's changing")
				printYellow("  3. Report this issue with the logs from tmp/")
			}
		}

		return result
	}

	// No exemptions - all drift is real
	result.hasDrift = true

	// Extract drift lines for the Drift Report
	// When exemptions are disabled, we still need to populate drift lines for the report
	planChangesText := extractPlanChanges(planOutput)
	if planChangesText != "" {
		// Split into lines and filter out empty lines
		for _, line := range strings.Split(planChangesText, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				result.driftLines = append(result.driftLines, line)
			}
		}
	}
	result.realCount = len(result.driftLines)
	result.materialCount = len(result.driftLines)

	if isNoRefreshDiagnostic {
		printYellow("No-refresh snapshot detected drift (diagnostic only; refresh plan is authoritative)")
	} else if stage == "initial" {
		printRed("⚠ Migration produced drift - v5 config wants to make changes")
	} else {
		printRed("✗ Ongoing drift detected - resources keep changing")
	}

	planSummary := extractPlanSummary(planOutput)
	if planSummary != "" {
		fmt.Println("  " + planSummary)
	}
	if !isNoRefreshDiagnostic {
		fmt.Println()

		// Show detailed changes
		if stage == "initial" {
			printYellow("Detailed changes:")
		} else {
			printYellow("Detailed ongoing drift:")
		}
		fmt.Println()
		fmt.Println(extractPlanChanges(planOutput))
		fmt.Println()

		// Explain what this means
		printRed("What this means:")
		if stage == "initial" {
			printRed("  The migrated v5 config doesn't match your existing infrastructure.")
			printRed("  This indicates the migration may not be correct.")
		} else {
			printRed("  Your resources are unstable - they change with every apply.")
			printRed("  This is a serious issue that prevents using v5 in production.")
		}
		fmt.Println()

		// Show affected resources
		affectedResources := extractAffectedResources(planOutput)
		if len(affectedResources) > 0 {
			printYellow("Affected Resources:")
			for _, resource := range affectedResources {
				printYellow("  - %s", resource)
			}
		}

		printYellow("")
		printYellow("Next steps:")
		if stage == "initial" {
			printYellow("  1. Review the changes above")
			printYellow("  2. Check if the migration tool has a bug")
			printYellow("  3. Consider using --apply-exemptions if changes are expected")
		} else {
			printYellow("  1. This is likely a provider or migration tool bug")
			printYellow("  2. Review the changes above to understand what's changing")
			printYellow("  3. Report this issue with the logs from tmp/")
		}
	}

	return result
}

// healCorruptedLeakedCredentialCheckRuleState detects and repairs a known v4
// provider bug: cloudflare_leaked_credential_check_rule's Read() lists all
// detection patterns and loops looking for one matching state's id, but if
// none match (e.g. because the real pattern was deleted outside Terraform)
// it does not error — it silently writes a zero-value result back to state,
// leaving id/username/password all null. Every subsequent Update() then
// fails with "required missing detection ID", and the resource can never
// recover on its own (confirmed directly in the v4 provider source,
// internal/framework/service/leaked_credential_check_rule/resource.go).
//
// The identical anti-pattern also exists in
// internal/framework/service/content_scanning_expression/resource.go (same
// "doesn't offer a single get operation" comment, same no-match-found
// handling), but that resource has no tf-migrate migrator or testdata, so
// it's not in scope here — only leaked_credential_check_rule is actually
// exercised by this harness today.
//
// This runs after v4 init, before v4 plan, looking for that corruption
// signature (id == null) in the module's state and removing the entry so
// the next plan/apply cleanly recreates it instead of repeatedly failing.
// Safe to run every time: if the resource is healthy or not present in
// state at all, this is a no-op.
func healCorruptedLeakedCredentialCheckRuleState(v4TF *TerraformRunner) error {
	const addr = "module.leaked_credential_check_rule.cloudflare_leaked_credential_check_rule.basic"

	showOutput, err := v4TF.Run("state", "show", addr)
	if err != nil {
		// Not in state (not targeted this run, or never created yet) — nothing to heal.
		return nil
	}

	if !isStateShowIDNull(showOutput) {
		return nil
	}

	printYellow("Detected corrupted state for %s (id is null — known v4 Read() bug after out-of-band deletion). Removing so it recreates cleanly...", addr)
	if _, err := v4TF.Run("state", "rm", addr); err != nil {
		return fmt.Errorf("failed to remove corrupted state for %s: %w", addr, err)
	}
	printSuccess("Removed corrupted state entry for %s", addr)
	return nil
}

// isStateShowIDNull reports whether `terraform state show <addr>` output has
// a top-level `id = null` line — the exact corruption signature left by the
// leaked_credential_check_rule v4 provider bug (a healthy resource always
// shows `id = "<real-id>"`).
func isStateShowIDNull(stateShowOutput string) bool {
	scanner := bufio.NewScanner(strings.NewReader(stateShowOutput))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "id") && strings.Contains(line, "= null") {
			return true
		}
	}
	return false
}

// findMovedFromAddresses scans every .tf file under v5Dir for `moved { from = ... }`
// blocks and returns the set of full state addresses (e.g.
// "module.zero_trust_access_policy.cloudflare_access_policy.example") that those
// blocks reference as their source. tf-migrate writes moved {} blocks as local,
// module-relative references (no "module.X." prefix) inside each resource's own
// module directory, so the module prefix is reconstructed here from each file's
// path relative to v5Dir (a file directly in v5Dir has no module prefix; a file in
// v5Dir/<name>/... belongs to "module.<name>").
//
// This exists so obsolete-type state cleanup (removeObsoleteStateEntries) can avoid
// deleting entries that are actually being renamed via a moved {} block rather than
// truly removed — deleting them first would make the moved {} block a no-op and
// cause Terraform to plan a spurious recreate.
func findMovedFromAddresses(v5Dir string) (map[string]bool, error) {
	protected := map[string]bool{}

	err := filepath.Walk(v5Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".tf") {
			return nil
		}

		rel, err := filepath.Rel(v5Dir, path)
		if err != nil {
			return err
		}
		modulePrefix := ""
		if dir := filepath.Dir(rel); dir != "." {
			// Only the first path segment is used: modules are one level deep
			// (v5Dir/<module>/<file>.tf), matching how this harness lays out output.
			first := strings.Split(dir, string(filepath.Separator))[0]
			modulePrefix = "module." + first
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		f, diags := hclwrite.ParseConfig(data, path, hcl.InitialPos)
		if diags.HasErrors() || f == nil {
			// Not fatal for this best-effort scan — just skip unparsable files.
			return nil
		}
		for _, block := range f.Body().Blocks() {
			if block.Type() != "moved" {
				continue
			}
			fromAttr := block.Body().GetAttribute("from")
			if fromAttr == nil {
				continue
			}
			fromText := strings.TrimSpace(string(hclwrite.Format(fromAttr.Expr().BuildTokens(nil).Bytes())))
			if fromText == "" {
				continue
			}
			addr := fromText
			if modulePrefix != "" {
				addr = modulePrefix + "." + fromText
			}
			protected[addr] = true
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan for moved blocks: %w", err)
	}
	return protected, nil
}

// findImportToAddresses scans every .tf file under v5Dir for `import { to = ... }`
// blocks and returns the set of resource addresses they target.
//
// This exists so the create-only-resource cleanup step (see cleanupCreateOnlyResources)
// can never destroy a resource that an import {} block says already exists for real —
// Terraform's plan JSON reports an import-driven resource with the same "create" action
// as a genuinely brand-new one, so action alone can't distinguish "safe to destroy test
// fixture" from "do not touch, this is a real pre-existing resource" (e.g.
// cloudflare_argo_tiered_caching, a DLP predefined profile, zero_trust_organization).
//
// tf-migrate normally hoists import {} blocks to the root main.tf, where "to" is already
// fully module-qualified (module.<name>.<type>.<label>). This scan tolerates either form:
// if "to" already starts with "module.", it's used as-is; otherwise the module prefix is
// reconstructed from the file's path, the same way findMovedFromAddresses does for moved
// blocks, so an unhoisted import block is still protected correctly.
func findImportToAddresses(v5Dir string) (map[string]bool, error) {
	protected := map[string]bool{}

	err := filepath.Walk(v5Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".tf") {
			return nil
		}

		rel, err := filepath.Rel(v5Dir, path)
		if err != nil {
			return err
		}
		modulePrefix := ""
		if dir := filepath.Dir(rel); dir != "." {
			first := strings.Split(dir, string(filepath.Separator))[0]
			modulePrefix = "module." + first
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		f, diags := hclwrite.ParseConfig(data, path, hcl.InitialPos)
		if diags.HasErrors() || f == nil {
			// Not fatal for this best-effort scan — just skip unparsable files.
			return nil
		}
		for _, block := range f.Body().Blocks() {
			if block.Type() != "import" {
				continue
			}
			toAttr := block.Body().GetAttribute("to")
			if toAttr == nil {
				continue
			}
			toText := strings.TrimSpace(string(hclwrite.Format(toAttr.Expr().BuildTokens(nil).Bytes())))
			if toText == "" {
				continue
			}
			addr := toText
			if modulePrefix != "" && !strings.HasPrefix(toText, "module.") {
				addr = modulePrefix + "." + toText
			}
			protected[addr] = true
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan for import blocks: %w", err)
	}
	return protected, nil
}

// pureCreateLinePattern matches Terraform's human-readable plan header for a
// genuinely new resource, e.g.:
//
//	  # module.queue.cloudflare_queue.minimal will be created
//
// Deliberately anchored to end right after "will be created" (allowing only
// trailing whitespace) so it does NOT match any other action Terraform
// renders differently:
//   - "will be updated in-place"  (adopted via a moved {} block / already real)
//   - "will be imported"          (an import {} block — a real pre-existing resource)
//   - "will be destroyed"
//   - "must be replaced"          (a real resource already existed and is being swapped)
var pureCreateLinePattern = regexp.MustCompile(`^\s*#\s+(\S+)\s+will be created\s*$`)

// extractPureCreateAddresses scans Terraform's human-readable plan output (the
// same text already captured for drift detection — see checkAndDisplayDrift)
// and returns the addresses of resources that are genuinely new, with no prior
// real-world identity.
//
// This parses plan TEXT rather than `terraform show -json <planfile>`
// deliberately: the JSON plan format always marshals the full prior state
// (not just the -target-scoped resources), so a single unrelated resource
// elsewhere in state with a stale/incompatible provider schema makes the
// entire `show -json` call fail outright — even though the actual -target
// plan/apply for the resources this run cares about succeeded cleanly. The
// text plan output this package already relies on throughout has no such
// problem, since it reflects only what the -target-scoped plan evaluated.
func extractPureCreateAddresses(planOutput string) []string {
	scanner := bufio.NewScanner(strings.NewReader(planOutput))
	var addrs []string
	for scanner.Scan() {
		if matches := pureCreateLinePattern.FindStringSubmatch(scanner.Text()); len(matches) > 1 {
			addrs = append(addrs, matches[1])
		}
	}
	return addrs
}

// cleanupCreateOnlyResources identifies every resource this run's v5 apply created
// with no prior real-world identity (a pure "create" action from planFilePath, and
// not protected by a moved {} or import {} block) and destroys exactly those,
// leaving the real v4 base and every adopted v5 resource completely untouched.
//
// This exists because v5-side Terraform state is never persisted between runs by
// design (see e2e/SUPPORTABILITY_MATRIX.md §11) — resources with a moved {} block
// are always safely re-adopted by their real ID regardless, but resources with no
// v4 counterpart at all have no such anchor. Left alone, every run (local or CI,
// on a weekly schedule) would create another copy, silently accumulating
// duplicates until some account-level quota is exhausted — exactly the mechanism
// behind the Access Policy and Gateway Certificate incidents documented in
// e2e/SUPPORTABILITY_MATRIX.md §10.
//
// Returns the addresses it attempted to destroy. A destroy failure for one or
// more individual resources is reported as an error for the caller to log as a
// warning — some create-only resources genuinely cannot be destroyed via
// Terraform (e.g. cloudflare_argo_tiered_caching, an account/zone-wide singleton
// keyed by zone_id rather than an independently generated ID, which carries no
// accumulation risk in the first place) — this must never retroactively fail an
// otherwise-successful migration test, so the caller decides how to surface it.
func cleanupCreateOnlyResources(v5TF *TerraformRunner, v5Dir string, planOutput string, tmpDir string, cfg *RunConfig) ([]string, error) {
	candidates := extractPureCreateAddresses(planOutput)
	if len(candidates) == 0 {
		return nil, nil
	}

	movedProtected, err := findMovedFromAddresses(v5Dir)
	if err != nil {
		return nil, fmt.Errorf("failed to scan moved blocks before cleanup: %w", err)
	}
	importProtected, err := findImportToAddresses(v5Dir)
	if err != nil {
		return nil, fmt.Errorf("failed to scan import blocks before cleanup: %w", err)
	}

	var targets []string
	for _, addr := range candidates {
		if movedProtected[addr] || importProtected[addr] {
			continue
		}
		targets = append(targets, addr)
	}
	if len(targets) == 0 {
		return nil, nil
	}

	destroyArgs := []string{"destroy", "-auto-approve", "-no-color", "-input=false"}
	for _, addr := range targets {
		destroyArgs = append(destroyArgs, "-target="+addr)
	}
	destroyArgs = addParallelismArg(destroyArgs, cfg.Parallelism)

	destroyOutput, destroyErr := v5TF.Run(destroyArgs...)
	destroyLog := filepath.Join(tmpDir, "v5-cleanup-destroy.log")
	if writeErr := os.WriteFile(destroyLog, []byte(destroyOutput), permFile); writeErr != nil {
		printYellow("Warning: failed to save cleanup destroy log to %s: %v", destroyLog, writeErr)
	}
	if destroyErr != nil {
		return targets, fmt.Errorf("terraform destroy failed for one or more create-only resources: %w", destroyErr)
	}

	return targets, nil
}

// removeObsoleteStateEntries removes resource entries of the given types directly
// from the local terraform.tfstate JSON file. This avoids running `terraform state rm`
// which requires modules to be installed (terraform init). Entries whose address is
// present in protectedAddrs are kept even if their type is obsolete — these are
// instances being renamed via a moved {} block rather than truly removed (see
// findMovedFromAddresses). Returns the list of removed addresses and the list of
// addresses that matched an obsolete type but were kept due to protection.
func removeObsoleteStateEntries(stateFilePath string, obsoleteTypes map[string]bool, protectedAddrs map[string]bool) ([]string, []string, error) {
	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	var state map[string]interface{}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	resources, ok := state["resources"].([]interface{})
	if !ok {
		return nil, nil, nil
	}

	var kept []interface{}
	var removed []string
	var skipped []string

	for _, r := range resources {
		res, ok := r.(map[string]interface{})
		if !ok {
			kept = append(kept, r)
			continue
		}
		rType, _ := res["type"].(string)
		if obsoleteTypes[rType] {
			rModule, _ := res["module"].(string)
			rName, _ := res["name"].(string)
			addr := rType + "." + rName
			if rModule != "" {
				addr = rModule + "." + addr
			}
			if protectedAddrs[addr] {
				skipped = append(skipped, addr)
				kept = append(kept, r)
				continue
			}
			removed = append(removed, addr)
		} else {
			kept = append(kept, r)
		}
	}

	if len(removed) == 0 {
		return nil, skipped, nil
	}

	state["resources"] = kept
	updated, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize state: %w", err)
	}
	if err := os.WriteFile(stateFilePath, updated, permFile); err != nil {
		return nil, nil, fmt.Errorf("failed to write state file: %w", err)
	}
	return removed, skipped, nil
}

func runV4Tests(ctx *testContext) error {
	printYellow("Step 1: Testing v4 configurations")

	// Initialize terraform
	printYellow("Running terraform init in v4/...")
	v4TF := NewTerraformRunner(ctx.v4Dir)

	// Configure R2 backend
	r2AccessKey := os.Getenv("CLOUDFLARE_R2_ACCESS_KEY_ID")
	r2SecretKey := os.Getenv("CLOUDFLARE_R2_SECRET_ACCESS_KEY")

	if r2AccessKey == "" || r2SecretKey == "" {
		printError("R2 credentials not set")
		return fmt.Errorf("please set: CLOUDFLARE_R2_ACCESS_KEY_ID and CLOUDFLARE_R2_SECRET_ACCESS_KEY")
	}

	// Set R2 credentials
	v4TF.EnvVars["AWS_ACCESS_KEY_ID"] = r2AccessKey
	v4TF.EnvVars["AWS_SECRET_ACCESS_KEY"] = r2SecretKey
	v4TF.EnvVars["TF_VAR_account_id"] = ctx.env.AccountID

	// Create backend config
	backendConfig := filepath.Join(ctx.v4Dir, "backend.hcl")
	backendConfigTmp := filepath.Join(ctx.v4Dir, "backend.configured.hcl")

	backendContent, err := os.ReadFile(backendConfig)
	if err != nil {
		return fmt.Errorf("failed to read backend config: %w", err)
	}

	configuredContent := strings.ReplaceAll(string(backendContent), "ACCOUNT_ID", ctx.env.AccountID)

	// Isolate the R2 state key when this run has a version suffix, so it
	// can't collide with the shared v4/terraform.tfstate key used by
	// default (unsuffixed) runs. See version_suffix.go.
	if ctx.cfg.VersionSuffix != "" {
		isolatedKey := versionedStateKey(ctx.cfg.VersionSuffix)
		configuredContent = strings.ReplaceAll(configuredContent, `key    = "v4/terraform.tfstate"`, `key    = "`+isolatedKey+`"`)
		printYellow("Using isolated R2 state key: %s", isolatedKey)
	}

	if err := os.WriteFile(backendConfigTmp, []byte(configuredContent), permFile); err != nil {
		return fmt.Errorf("failed to write backend config to %s: %w", backendConfigTmp, err)
	}
	defer func() {
		if err := os.Remove(backendConfigTmp); err != nil && !os.IsNotExist(err) {
			printYellow("Warning: Failed to remove temp backend config %s: %v", backendConfigTmp, err)
		}
	}()

	// Check if local state exists
	localState := filepath.Join(ctx.v4Dir, "terraform.tfstate")
	if _, err := os.Stat(localState); err == nil {
		printYellow("Found local state file, backing up and using remote state...")
		os.Remove(localState)
	}

	// Run terraform init
	initArgs := []string{"init", "-no-color", "-reconfigure", "-backend-config=" + backendConfigTmp}
	if err := v4TF.RunToFile(filepath.Join(ctx.tmpDir, "v4-init.log"), initArgs...); err != nil {
		printError("Terraform init failed for v4")
		fmt.Println()
		printRed("Error output:")
		content, _ := os.ReadFile(filepath.Join(ctx.tmpDir, "v4-init.log"))
		fmt.Println(string(content))
		return err
	}
	printSuccess("Terraform init successful (remote state loaded from R2)")

	if err := healCorruptedLeakedCredentialCheckRuleState(v4TF); err != nil {
		printYellow("Warning: health check for leaked_credential_check_rule state failed: %v", err)
	}

	// Run terraform plan
	printYellow("Running terraform plan in v4/...")
	planArgs := append([]string{"plan", "-no-color", "-out=" + filepath.Join(ctx.tmpDir, "v4.tfplan"), "-input=false"}, ctx.targetArgs...)
	planArgs = addParallelismArg(planArgs, ctx.cfg.Parallelism)
	planOutput, err := v4TF.Run(planArgs...)
	if err != nil {
		printError("Terraform plan failed for v4")
		fmt.Println()
		printRed("Error output:")
		fmt.Println(planOutput)
		return err
	}

	// Save plan output for debugging
	v4PlanLog := filepath.Join(ctx.tmpDir, "v4-plan.log")
	if err := os.WriteFile(v4PlanLog, []byte(planOutput), permFile); err != nil {
		printYellow("Warning: Failed to save v4 plan log to %s: %v", v4PlanLog, err)
	}

	// Check for changes
	if strings.Contains(planOutput, "No changes") {
		printSuccess("Terraform plan shows no changes")
	} else {
		printSuccess("Terraform plan successful")
		planSummary := extractPlanSummary(planOutput)
		if planSummary != "" {
			fmt.Println("  " + planSummary)
		}
		fmt.Println()

		// Show detailed changes
		printYellow("Detailed changes:")
		fmt.Println()
		fmt.Println(extractPlanChanges(planOutput))
	}
	fmt.Println()

	// Run terraform apply
	printYellow("Running terraform apply in v4/...")
	applyArgs := []string{"apply", "-no-color", "-auto-approve", "-input=false"}
	applyArgs = addParallelismArg(applyArgs, ctx.cfg.Parallelism)
	applyArgs = append(applyArgs, filepath.Join(ctx.tmpDir, "v4.tfplan"))
	applyOutput, err := v4TF.Run(applyArgs...)
	if err != nil {
		printError("Terraform apply failed for v4")
		fmt.Println()
		printRed("Error output:")
		fmt.Println(applyOutput)
		return err
	}

	// Save apply output for debugging
	v4ApplyLog := filepath.Join(ctx.tmpDir, "v4-apply.log")
	if err := os.WriteFile(v4ApplyLog, []byte(applyOutput), permFile); err != nil {
		printYellow("Warning: Failed to save v4 apply log to %s: %v", v4ApplyLog, err)
	}
	printSuccess("Terraform apply successful")

	// Show apply summary
	if strings.Contains(applyOutput, "Apply complete!") {
		lines := strings.Split(applyOutput, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Apply complete!") {
				fmt.Println("  " + line)
				break
			}
		}
	}

	// Pull state from remote
	printYellow("Syncing state from remote...")
	if err := v4TF.StatePull("terraform.tfstate"); err != nil {
		printYellow("⚠ Could not pull state (may be empty)")
	} else {
		printSuccess("Local state file synced from R2")
	}

	// Capture v4 state
	printYellow("Capturing v4 state...")
	stateOutput, err := v4TF.Run("show", "-no-color", "-json")
	if err != nil {
		return fmt.Errorf("failed to capture v4 state: %w", err)
	}
	// Save v4 state snapshot for comparison
	v4StateLog := filepath.Join(ctx.tmpDir, "v4-state.json")
	if err := os.WriteFile(v4StateLog, []byte(stateOutput), permFile); err != nil {
		printYellow("Warning: Failed to save v4 state to %s: %v", v4StateLog, err)
	} else {
		printSuccess("Saved v4 state to tmp/v4-state.json")
	}
	fmt.Println()

	return nil
}

// countUniqueDrifts returns the number of unique drift patterns
func countUniqueDrifts(driftLines []string) int {
	driftCounts := make(map[string]int)
	for _, line := range driftLines {
		driftCounts[line]++
	}
	return len(driftCounts)
}

func addParallelismArg(args []string, parallelism int) []string {
	if parallelism > 0 {
		return append(args, fmt.Sprintf("-parallelism=%d", parallelism))
	}
	return args
}

// getDriftColorFunc returns the appropriate color function based on drift type
func getDriftColorFunc(pattern string) func(string, ...interface{}) {
	trimmed := strings.TrimSpace(pattern)
	if strings.HasPrefix(trimmed, "-") {
		return printRed // Deletion
	} else if strings.HasPrefix(trimmed, "+") {
		return printGreen // Addition
	} else if strings.HasPrefix(trimmed, "~") {
		return printYellow // Modification
	}
	return printYellow // Default to yellow
}

// displayGroupedDrift groups duplicate drift lines and displays them with counts
func displayGroupedDrift(driftLines []string) {
	// Separate drift lines by extracting resource name and drift pattern
	type driftInfo struct {
		resource string
		pattern  string
	}

	patternCounts := make(map[string]int)         // pattern -> count
	patternResources := make(map[string][]string) // pattern -> list of resources

	for _, line := range driftLines {
		// Try to extract resource name and pattern
		// Format is: "  resource.name: pattern" or just "  pattern"
		parts := strings.SplitN(strings.TrimSpace(line), ": ", 2)

		var resource, pattern string
		if len(parts) == 2 {
			resource = parts[0]
			pattern = parts[1]
		} else {
			resource = ""
			pattern = parts[0]
		}

		patternCounts[pattern]++
		// Store unique resources for this pattern
		if !contains(patternResources[pattern], resource) && resource != "" {
			patternResources[pattern] = append(patternResources[pattern], resource)
		}
	}

	// Sort by count (descending) for better readability
	type driftEntry struct {
		pattern   string
		count     int
		resources []string
	}
	var entries []driftEntry
	for pattern, count := range patternCounts {
		entries = append(entries, driftEntry{pattern, count, patternResources[pattern]})
	}

	// Sort by count descending
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].count > entries[i].count {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Display drift patterns with resource context
	for _, entry := range entries {
		// Get the appropriate color function based on drift type
		colorFunc := getDriftColorFunc(entry.pattern)

		if entry.count > 1 {
			// Show pattern with count
			colorFunc("  %s (×%d)", entry.pattern, entry.count)
			// Show first few resources as examples
			if len(entry.resources) > 0 {
				numToShow := 3
				if len(entry.resources) < numToShow {
					numToShow = len(entry.resources)
				}
				printBlue("    Resources: %s", strings.Join(entry.resources[:numToShow], ", "))
				if len(entry.resources) > numToShow {
					printBlue("    ... and %d more", len(entry.resources)-numToShow)
				}
			}
		} else {
			// Single occurrence - show with resource name if available
			if len(entry.resources) > 0 {
				printYellow("  %s:", entry.resources[0])
				colorFunc("    %s", entry.pattern)
			} else {
				colorFunc("  %s", entry.pattern)
			}
		}
	}

	if len(entries) == 0 && len(driftLines) > 0 {
		printYellow("  Total drift lines: %d", len(driftLines))
	}
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// discoverAllResources finds all resource directories in testdata
func discoverAllResources() ([]string, error) {
	repoRoot := getRepoRoot()
	testdataRoot := filepath.Join(repoRoot, "integration", "v4_to_v5", "testdata")

	entries, err := os.ReadDir(testdataRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read testdata directory: %w", err)
	}

	var resources []string
	for _, entry := range entries {
		if entry.IsDir() {
			resources = append(resources, entry.Name())
		}
	}

	return resources, nil
}
