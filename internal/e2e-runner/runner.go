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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// File permission constants for consistent permission management
const (
	permDir        = 0755 // rwxr-xr-x - directories
	permFile       = 0644 // rw-r--r-- - regular files
	permSecretFile = 0600 // rw------- - sensitive files (state, secrets)
)

// RunConfig holds configuration for e2e test run
type RunConfig struct {
	SkipV4Test       bool
	ApplyExemptions  bool
	Resources        string
	ProviderPath     string
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
	v5InitialDrift           []string
	v5PostApplyDrift         []string
	v5InitialExempted        int
	v5PostApplyExempted      int
	v5InitialExemptedLines   []string
	v5PostApplyExemptedLines []string

	// Output tracking
	v5PlanOutput     string
	v5PostPlanOutput string
}

// RunE2ETests executes the complete e2e test suite
func RunE2ETests(cfg *RunConfig) error {
	// Get paths
	repoRoot := getRepoRoot()
	e2eRoot := filepath.Join(repoRoot, "e2e")
	v4Dir := filepath.Join(e2eRoot, "tf", "v4")
	v5Dir := filepath.Join(e2eRoot, "migrated-v4_to_v5")
	tmpDir := filepath.Join(e2eRoot, "tmp")

	// Create tmp directory
	if err := os.MkdirAll(tmpDir, permDir); err != nil {
		return fmt.Errorf("failed to create tmp directory %s: %w", tmpDir, err)
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

	printYellow("Running tests with:")
	printYellow("  User:       %s", env.Email)
	printYellow("  Account ID: %s", env.AccountID)
	printYellow("  Zone ID:    %s", env.ZoneID)
	printYellow("  Domain:     %s", env.Domain)
	if cfg.ProviderPath != "" {
		printYellow("  Provider:   Local (%s)", cfg.ProviderPath)
	} else {
		printYellow("  Provider:   Registry (latest)")
	}
	fmt.Println()

	printYellow("Running init script...")
	if err := RunInit(cfg.Resources); err != nil {
		printError("Init script failed")
		return err
	}
	printSuccess("Test resources initialized")
	fmt.Println()

	// Set up local provider if specified
	var tfConfigFile string
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

	if err := RunMigrate(cfg.Resources); err != nil {
		printError("Migration failed")
		return err
	}
	printSuccess("Migration successful")

	// Step 3: Test v5 configurations
	fmt.Println()
	printCyan("Step 3: Testing v5 configurations")
	printYellow("Running terraform init in migrated-v4_to_v5/...")

	// Clean .terraform if it exists
	v5TFDir := filepath.Join(v5Dir, ".terraform")
	if _, err := os.Stat(v5TFDir); err == nil {
		printYellow("Cleaning v5 .terraform directory for fresh init...")
		if err := os.RemoveAll(v5TFDir); err != nil {
			return fmt.Errorf("failed to remove v5 .terraform directory %s: %w", v5TFDir, err)
		}
	}

	v5TF := NewTerraformRunner(v5Dir)
	if tfConfigFile != "" {
		v5TF.TFConfigFile = tfConfigFile
	}

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

	// Plan v5
	printYellow("Running terraform plan in v5/...")
	v5PlanArgs := append([]string{"plan", "-no-color", "-out=" + filepath.Join(tmpDir, "v5.tfplan"), "-input=false"}, targetArgs...)
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
	driftResult := checkAndDisplayDrift(ctx.v5PlanOutput, cfg, "initial")
	ctx.hasChanges = driftResult.hasDrift
	ctx.v5InitialDrift = driftResult.driftLines
	ctx.v5InitialExempted = driftResult.exemptedCount
	ctx.v5InitialExemptedLines = driftResult.exemptedLines

	// Apply v5
	printYellow("Running terraform apply in v5/...")
	v5ApplyArgs := []string{"apply", "-no-color", "-auto-approve", "-input=false", filepath.Join(tmpDir, "v5.tfplan")}
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

	// Capture v5 state
	printYellow("Capturing v5 state...")
	v5StateOutput, err := v5TF.Run("show", "-no-color", "-json")
	if err != nil {
		return fmt.Errorf("failed to capture v5 state: %w", err)
	}
	// Save v5 state snapshot for comparison
	v5StateLog := filepath.Join(tmpDir, "v5-state.json")
	if err := os.WriteFile(v5StateLog, []byte(v5StateOutput), permFile); err != nil {
		printYellow("Warning: Failed to save v5 state to %s: %v", v5StateLog, err)
	} else {
		printSuccess("Saved v5 state to tmp/v5-state.json")
	}

	// Step 4: Verify stable state
	fmt.Println()
	printCyan("Step 4: Verifying stable state (v5 plan after apply)")
	printYellow("Running terraform plan again to check for ongoing drift...")

	v5PostPlanArgs := append([]string{"plan", "-no-color", "-out=" + filepath.Join(tmpDir, "v5-post-apply.tfplan"), "-input=false"}, targetArgs...)
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
	postDriftResult := checkAndDisplayDrift(ctx.v5PostPlanOutput, cfg, "post-apply")
	ctx.hasPostApplyChanges = postDriftResult.hasDrift
	ctx.v5PostApplyDrift = postDriftResult.driftLines
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
		} else {
			printYellow("    Status: %s", colorYellow+"⚠ Changes detected but all exempted"+colorReset)
			printYellow("    Result: 0 real changes (%d exempted)", ctx.v5InitialExempted)
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
		printYellow("    Status: %s", colorGreen+"✓ SUCCESS - Stable state achieved"+colorReset)
		printYellow("    Result: No changes detected")
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

	return nil
}

// driftCheckResult holds the result of checking and displaying drift
type driftCheckResult struct {
	hasDrift        bool
	driftLines      []string
	exemptedCount   int
	exemptedLines   []string
}

// checkAndDisplayDrift checks for drift in plan output and displays results
// Returns true if real drift was detected (excluding exemptions)
func checkAndDisplayDrift(planOutput string, cfg *RunConfig, stage string) driftCheckResult {
	result := driftCheckResult{}

	if strings.Contains(planOutput, "No changes") {
		if stage == "initial" {
			printSuccess("Terraform plan shows no changes (expected)")
		} else {
			printSuccess("No ongoing drift detected - migration achieved stable state!")
		}
		return result
	}

	// Check if only computed changes when --apply-exemptions is set
	if cfg.ApplyExemptions {
		driftResult := checkDrift(planOutput)

		// Calculate total exempted count
		totalExempted := 0
		for _, count := range driftResult.TriggeredExemptions {
			totalExempted += count
		}
		result.exemptedCount = totalExempted
		result.exemptedLines = driftResult.ExemptedDriftLines

		if driftResult.OnlyComputedChanges {
			if stage == "initial" {
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

		if stage == "initial" {
			printRed("⚠ Migration produced drift - v5 config wants to make changes")
		} else {
			printRed("✗ Ongoing drift detected - resources keep changing")
		}

		planSummary := extractPlanSummary(planOutput)
		if planSummary != "" {
			fmt.Println("  " + planSummary)
		}
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

	if stage == "initial" {
		printRed("⚠ Migration produced drift - v5 config wants to make changes")
	} else {
		printRed("✗ Ongoing drift detected - resources keep changing")
	}

	planSummary := extractPlanSummary(planOutput)
	if planSummary != "" {
		fmt.Println("  " + planSummary)
	}
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

	return result
}

// runV4Tests executes Step 1: Test v4 configurations
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

	// Create backend config
	backendConfig := filepath.Join(ctx.v4Dir, "backend.hcl")
	backendConfigTmp := filepath.Join(ctx.v4Dir, "backend.configured.hcl")

	backendContent, err := os.ReadFile(backendConfig)
	if err != nil {
		return fmt.Errorf("failed to read backend config: %w", err)
	}

	configuredContent := strings.ReplaceAll(string(backendContent), "ACCOUNT_ID", ctx.env.AccountID)
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


	// Run terraform plan
	printYellow("Running terraform plan in v4/...")
	planArgs := append([]string{"plan", "-no-color", "-out=" + filepath.Join(ctx.tmpDir, "v4.tfplan"), "-input=false"}, ctx.targetArgs...)
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
	applyArgs := []string{"apply", "-no-color", "-auto-approve", "-input=false", filepath.Join(ctx.tmpDir, "v4.tfplan")}
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

	patternCounts := make(map[string]int)          // pattern -> count
	patternResources := make(map[string][]string)  // pattern -> list of resources

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
