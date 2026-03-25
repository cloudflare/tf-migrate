// state_upgrader.go provides the main orchestration for v5→v5 v5 upgrade tests.
//
// This file implements the complete test workflow that validates provider v5 upgrades
// work correctly when upgrading between v5 versions. Unlike v4→v5 migrations, these tests:
//   - Do NOT use tf-migrate (configs are already v5 syntax)
//   - Test the provider's built-in UpgradeState mechanism
//   - Create resources with an older v5 version, then upgrade the provider
//
// Test workflow:
//  1. Initialize: Sync testdata, generate configs with source version
//  2. Create: terraform init/apply with source version provider
//  3. Upgrade: Update provider version, terraform init -upgrade
//  4. Verify: terraform plan to check state upgrade success
//  5. Stabilize: Apply if needed, verify no ongoing drift
package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// suTestContext holds shared state for v5 upgrade test execution
type suTestContext struct {
	cfg          *V5UpgradeConfig
	env          *E2EEnv
	repoRoot     string
	e2eV5Root    string
	tfDir        string
	tmpDir       string
	testdataRoot string
	resourceList []string
	tfConfigFile string // for dev_overrides when using local provider

	// Drift tracking
	hasChanges             bool
	hasPostApplyChanges    bool
	initialDrift           []string
	postApplyDrift         []string
	initialMaterial        int
	postApplyMaterial      int
	initialReal            int
	postApplyReal          int
	initialExempted        int
	postApplyExempted      int
	initialExemptedLines   []string
	postApplyExemptedLines []string

	// Output tracking
	planOutput     string
	postPlanOutput string
}

// RunV5UpgradeTests executes the complete v5 upgrade test workflow
func RunV5UpgradeTests(cfg *V5UpgradeConfig) error {
	if cfg.Parallelism < 0 {
		return fmt.Errorf("parallelism must be >= 0, got %d", cfg.Parallelism)
	}

	// Normalize versions
	if cfg.FromVersion == "" {
		cfg.FromVersion = DefaultFromVersion
	}
	if cfg.ToVersion == "" {
		cfg.ToVersion = DefaultToVersion
	}

	// Get paths
	repoRoot := getRepoRoot()
	e2eV5Root := filepath.Join(repoRoot, "e2e-v5")
	tfDir := filepath.Join(e2eV5Root, "tf")
	tmpDir := filepath.Join(e2eV5Root, "tmp")
	testdataRoot := filepath.Join(e2eV5Root, "testdata")

	// Create tmp directory
	if err := os.MkdirAll(tmpDir, permDir); err != nil {
		return fmt.Errorf("failed to create tmp directory %s: %w", tmpDir, err)
	}

	printHeader("V5 Upgrade Tests")
	printCyan("Testing provider v5 upgrades: %s → %s", cfg.FromVersion, cfg.ToVersion)
	fmt.Println()

	printYellow("Configuration:")
	printYellow("  From version:     %s", cfg.FromVersion)
	if cfg.ProviderPath != "" {
		printYellow("  To version:       Local provider (%s)", cfg.ProviderPath)
	} else {
		printYellow("  To version:       %s", cfg.ToVersion)
	}
	printYellow("  State key:        %s", GenerateStateKey(cfg.FromVersion, cfg.ToVersion))
	if cfg.Resources != "" {
		printYellow("  Resources:        %s", cfg.Resources)
	}
	if cfg.ApplyExemptions {
		printYellow("  Exemptions:       Enabled")
	}
	fmt.Println()

	// Load environment
	env, err := LoadEnv(EnvForRunner)
	if err != nil {
		return err
	}

	// Parse resource list
	var resourceList []string
	if cfg.Resources != "" {
		for _, r := range strings.Split(cfg.Resources, ",") {
			resourceList = append(resourceList, strings.TrimSpace(r))
		}
	}

	// Build target args for terraform commands
	var targetArgs []string
	for _, resource := range resourceList {
		targetArgs = append(targetArgs, "-target=module."+resource)
	}

	// Set up local provider if specified (for target version)
	var tfConfigFile string
	if cfg.ProviderPath != "" {
		var err error
		tfConfigFile, err = setupLocalProvider(cfg.ProviderPath, repoRoot)
		if err != nil {
			return err
		}
	}

	// Create test context
	ctx := &suTestContext{
		cfg:          cfg,
		env:          env,
		repoRoot:     repoRoot,
		e2eV5Root:    e2eV5Root,
		tfDir:        tfDir,
		tmpDir:       tmpDir,
		testdataRoot: testdataRoot,
		resourceList: resourceList,
		tfConfigFile: tfConfigFile,
	}

	// Step 1: Initialize test resources
	if !cfg.SkipCreate {
		printCyan("Step 1: Initializing test resources")
		if err := RunV5UpgradeInit(cfg); err != nil {
			printError("Initialization failed")
			return err
		}
		printSuccess("Initialization complete")
		fmt.Println()
	} else {
		printCyan("Step 1: Skipped initialization (--skip-create)")
		fmt.Println()
	}

	// cleanOnExit runs cleanup if --clean is set, regardless of success or failure.
	// This prevents orphaned resources when the test fails partway through.
	cleanOnExit := func(testErr error) error {
		if !cfg.Clean {
			return testErr
		}
		fmt.Println()
		if testErr != nil {
			printCyan("Cleaning up resources after failure (--clean)...")
		} else {
			printCyan("Step 7: Cleaning up resources (--clean)")
		}
		if cleanErr := RunV5UpgradeClean(cfg, nil); cleanErr != nil {
			printError("Cleanup failed: %v", cleanErr)
			if testErr == nil {
				return cleanErr
			}
		}
		return testErr
	}

	// Step 2: Create resources with source version
	if !cfg.SkipCreate {
		if err := runCreateWithSourceVersion(ctx); err != nil {
			return cleanOnExit(err)
		}
	} else {
		printCyan("Step 2: Skipped resource creation (--skip-create)")
		fmt.Println()
	}

	// Step 3: Upgrade provider version
	if err := runUpgradeProvider(ctx); err != nil {
		return cleanOnExit(err)
	}

	// Steps 4-6: Verify state upgrade (plan → apply → plan)
	if err := runVerifyStateUpgrade(ctx); err != nil {
		return cleanOnExit(err)
	}

	// Report results (not a numbered step - just summary)
	testErr := reportResults(ctx)

	return cleanOnExit(testErr)
}

// runCreateWithSourceVersion creates resources using the source provider version
func runCreateWithSourceVersion(ctx *suTestContext) error {
	printCyan("Step 2: Creating resources with provider %s", ctx.cfg.FromVersion)

	tf := NewTerraformRunner(ctx.tfDir)
	tf.EnvVars["AWS_ACCESS_KEY_ID"] = ctx.env.R2AccessKeyID
	tf.EnvVars["AWS_SECRET_ACCESS_KEY"] = ctx.env.R2SecretAccessKey

	// Initialize terraform with source version
	printYellow("Running terraform init...")
	initArgs := []string{"init", "-no-color", "-input=false", "-backend-config=backend.configured.hcl", "-reconfigure", "-upgrade"}
	if err := tf.RunToFile(filepath.Join(ctx.tmpDir, "source-init.log"), initArgs...); err != nil {
		printError("Terraform init failed")
		content, _ := os.ReadFile(filepath.Join(ctx.tmpDir, "source-init.log"))
		fmt.Println(string(content))
		return err
	}
	printSuccess("Terraform init successful")

	// Run plan
	printYellow("Running terraform plan...")
	planArgs := append([]string{"plan", "-no-color", "-out=" + filepath.Join(ctx.tmpDir, "source.tfplan"), "-input=false"}, ctx.targetArgs()...)
	planArgs = addParallelismArg(planArgs, ctx.cfg.Parallelism)
	if err := tf.RunToFile(filepath.Join(ctx.tmpDir, "source-plan.log"), planArgs...); err != nil {
		content, _ := os.ReadFile(filepath.Join(ctx.tmpDir, "source-plan.log"))
		planLog := string(content)

		// When running a targeted batch with an older source provider, stale state
		// from previously-upgraded modules can trigger:
		// "Resource instance managed by newer provider version".
		// In that case, prune non-target module state and retry the source plan once.
		if strings.Contains(planLog, "Resource instance managed by newer provider version") && len(ctx.resourceList) > 0 {
			printYellow("Detected newer-provider state outside target modules; pruning non-target module state and retrying plan once...")
			removed, pruneErr := pruneNonTargetModuleState(tf, ctx.resourceList)
			if pruneErr != nil {
				printError("Failed to prune non-target module state")
				fmt.Println(planLog)
				return pruneErr
			}
			if removed > 0 {
				printSuccess("Pruned %d non-target state entries", removed)
			} else {
				printYellow("No non-target module state entries found to prune")
			}

			if retryErr := tf.RunToFile(filepath.Join(ctx.tmpDir, "source-plan.log"), planArgs...); retryErr != nil {
				printError("Terraform plan failed (after prune retry)")
				retryContent, _ := os.ReadFile(filepath.Join(ctx.tmpDir, "source-plan.log"))
				fmt.Println(string(retryContent))
				return retryErr
			}
			printSuccess("Terraform plan successful after state prune")
		} else {
			printError("Terraform plan failed")
			fmt.Println(planLog)
			return err
		}
	}
	printSuccess("Terraform plan successful")

	// Run apply
	printYellow("Running terraform apply...")
	applyArgs := []string{"apply", "-no-color", "-input=false", filepath.Join(ctx.tmpDir, "source.tfplan")}
	if err := tf.RunToFile(filepath.Join(ctx.tmpDir, "source-apply.log"), applyArgs...); err != nil {
		printError("Terraform apply failed")
		content, _ := os.ReadFile(filepath.Join(ctx.tmpDir, "source-apply.log"))
		fmt.Println(string(content))
		return err
	}
	printSuccess("Terraform apply successful")
	printSuccess("Resources created with provider %s", ctx.cfg.FromVersion)
	fmt.Println()

	return nil
}

// pruneNonTargetModuleState removes module.* state entries that do not belong
// to the current resource target list.
func pruneNonTargetModuleState(tf *TerraformRunner, targetModules []string) (int, error) {
	output, err := tf.Run("state", "list")
	if err != nil {
		return 0, err
	}

	targetSet := make(map[string]struct{}, len(targetModules))
	for _, m := range targetModules {
		targetSet[m] = struct{}{}
	}

	removed := 0
	for _, line := range strings.Split(output, "\n") {
		addr := strings.TrimSpace(line)
		if addr == "" || !strings.HasPrefix(addr, "module.") {
			continue
		}

		remainder := strings.TrimPrefix(addr, "module.")
		dot := strings.Index(remainder, ".")
		if dot <= 0 {
			continue
		}
		moduleName := remainder[:dot]

		if _, ok := targetSet[moduleName]; ok {
			continue
		}

		if _, rmErr := tf.Run("state", "rm", addr); rmErr != nil {
			return removed, rmErr
		}
		removed++
	}

	return removed, nil
}

// runUpgradeProvider upgrades the provider to the target version
func runUpgradeProvider(ctx *suTestContext) error {
	printCyan("Step 3: Upgrading provider to %s", ctx.cfg.ToVersion)

	tf := NewTerraformRunner(ctx.tfDir)
	tf.EnvVars["AWS_ACCESS_KEY_ID"] = ctx.env.R2AccessKeyID
	tf.EnvVars["AWS_SECRET_ACCESS_KEY"] = ctx.env.R2SecretAccessKey

	if ctx.cfg.ProviderPath != "" {
		// Using local provider - set up dev_overrides
		printYellow("Using local provider from: %s", ctx.cfg.ProviderPath)
		tf.TFConfigFile = ctx.tfConfigFile

		// Clean .terraform directory to ensure dev_overrides are used
		tfDirPath := filepath.Join(ctx.tfDir, ".terraform")
		if _, err := os.Stat(tfDirPath); err == nil {
			printYellow("Cleaning .terraform directory for fresh init with local provider...")
			if err := os.RemoveAll(tfDirPath); err != nil {
				return fmt.Errorf("failed to remove .terraform directory: %w", err)
			}
		}

		// Remove lock file
		lockFile := filepath.Join(ctx.tfDir, ".terraform.lock.hcl")
		if _, err := os.Stat(lockFile); err == nil {
			if err := os.Remove(lockFile); err != nil {
				return fmt.Errorf("failed to remove lock file: %w", err)
			}
		}

		// Re-init with local provider
		printYellow("Running terraform init with local provider...")
		initArgs := []string{"init", "-no-color", "-input=false", "-backend-config=backend.configured.hcl"}
		if err := tf.RunToFile(filepath.Join(ctx.tmpDir, "upgrade-init.log"), initArgs...); err != nil {
			printError("Terraform init failed")
			content, _ := os.ReadFile(filepath.Join(ctx.tmpDir, "upgrade-init.log"))
			fmt.Println(string(content))
			return err
		}
	} else {
		// Using registry provider - update provider.tf and run init -upgrade
		printYellow("Updating provider.tf to version %s...", ctx.cfg.ToVersion)
		targetConstraint, _ := ResolveTargetVersion(ctx.cfg.ToVersion)
		providerContent := GenerateProviderTF(targetConstraint)
		providerFile := filepath.Join(ctx.tfDir, "provider.tf")
		if err := os.WriteFile(providerFile, []byte(providerContent), permFile); err != nil {
			return fmt.Errorf("failed to update provider.tf: %w", err)
		}

		// Run terraform init -upgrade
		printYellow("Running terraform init -upgrade...")
		initArgs := []string{"init", "-no-color", "-input=false", "-backend-config=backend.configured.hcl", "-upgrade"}
		if err := tf.RunToFile(filepath.Join(ctx.tmpDir, "upgrade-init.log"), initArgs...); err != nil {
			printError("Terraform init -upgrade failed")
			content, _ := os.ReadFile(filepath.Join(ctx.tmpDir, "upgrade-init.log"))
			fmt.Println(string(content))
			return err
		}
	}

	printSuccess("Provider upgraded successfully")
	fmt.Println()

	// Resync module configs to the default (target-version) files.
	// If any module had a version-specific config for the source version
	// (e.g. page_rule_5_1_0.tf), it must be replaced with the default config
	// before the post-upgrade plan runs, since the old syntax is no longer
	// valid against the upgraded provider.
	if err := resyncForTargetVersion(ctx.testdataRoot, ctx.tfDir, ctx.resourceList); err != nil {
		return fmt.Errorf("failed to resync configs for target version: %w", err)
	}
	fmt.Println()

	return nil
}

// runVerifyStateUpgrade verifies the state upgrade was successful
// This follows the same flow as v4→v5 tests: Plan → Apply → Plan (verify stable)
func runVerifyStateUpgrade(ctx *suTestContext) error {
	printCyan("Step 4: Verifying state upgrade")

	tf := NewTerraformRunner(ctx.tfDir)
	tf.EnvVars["AWS_ACCESS_KEY_ID"] = ctx.env.R2AccessKeyID
	tf.EnvVars["AWS_SECRET_ACCESS_KEY"] = ctx.env.R2SecretAccessKey
	if ctx.tfConfigFile != "" {
		tf.TFConfigFile = ctx.tfConfigFile
	}

	// Run terraform plan
	printYellow("Running terraform plan...")
	planArgs := append([]string{"plan", "-no-color", "-out=" + filepath.Join(ctx.tmpDir, "upgrade.tfplan"), "-input=false"}, ctx.targetArgs()...)
	planArgs = addParallelismArg(planArgs, ctx.cfg.Parallelism)

	var err error
	ctx.planOutput, err = tf.Run(planArgs...)
	if err != nil {
		printError("Terraform plan failed - STATE UPGRADE MAY HAVE FAILED")
		fmt.Println()
		printRed("Error output:")
		fmt.Println(ctx.planOutput)

		// Save plan output for debugging
		planLog := filepath.Join(ctx.tmpDir, "upgrade-plan.log")
		os.WriteFile(planLog, []byte(ctx.planOutput), permFile)

		return fmt.Errorf("state upgrade verification failed: terraform plan error")
	}

	// Save plan output
	planLog := filepath.Join(ctx.tmpDir, "upgrade-plan.log")
	if err := os.WriteFile(planLog, []byte(ctx.planOutput), permFile); err != nil {
		printYellow("Warning: Failed to save plan log: %v", err)
	}

	// Check for drift
	driftResult := checkAndDisplayDriftForSU(ctx.planOutput, ctx.cfg, "upgrade", ctx.resourceList, ctx.e2eV5Root)
	ctx.hasChanges = driftResult.hasDrift
	ctx.initialDrift = driftResult.driftLines
	ctx.initialMaterial = driftResult.materialCount
	ctx.initialReal = driftResult.realCount
	ctx.initialExempted = driftResult.exemptedCount
	ctx.initialExemptedLines = driftResult.exemptedLines

	// Always apply (even if no changes) to match v4→v5 test flow
	// This ensures we verify state stability consistently
	printYellow("Running terraform apply...")
	applyArgs := []string{"apply", "-no-color", "-auto-approve", "-input=false", filepath.Join(ctx.tmpDir, "upgrade.tfplan")}
	applyOutput, err := tf.Run(applyArgs...)
	if err != nil {
		printError("Terraform apply failed")
		fmt.Println()
		printRed("Error output:")
		fmt.Println(applyOutput)
		return err
	}

	// Save apply output for debugging
	applyLog := filepath.Join(ctx.tmpDir, "upgrade-apply.log")
	if err := os.WriteFile(applyLog, []byte(applyOutput), permFile); err != nil {
		printYellow("Warning: Failed to save apply log: %v", err)
	}
	printSuccess("Terraform apply successful")

	// Capture state snapshot
	printYellow("Capturing state...")
	stateOutput, err := tf.Run("show", "-no-color", "-json")
	if err != nil {
		printYellow("Warning: Failed to capture state: %v", err)
	} else {
		stateLog := filepath.Join(ctx.tmpDir, "upgrade-state.json")
		if err := os.WriteFile(stateLog, []byte(stateOutput), permFile); err != nil {
			printYellow("Warning: Failed to save state to %s: %v", stateLog, err)
		} else {
			printSuccess("Saved state to tmp/upgrade-state.json")
		}
	}

	// Step 5: Verify stable state (plan after apply)
	fmt.Println()
	printCyan("Step 5: Verifying stable state (plan after apply)")
	printYellow("Running terraform plan again to check for ongoing drift...")

	postPlanArgs := append([]string{"plan", "-no-color", "-out=" + filepath.Join(ctx.tmpDir, "upgrade-post.tfplan"), "-input=false"}, ctx.targetArgs()...)
	postPlanArgs = addParallelismArg(postPlanArgs, ctx.cfg.Parallelism)

	ctx.postPlanOutput, err = tf.Run(postPlanArgs...)
	if err != nil {
		printError("Post-apply terraform plan failed")
		fmt.Println()
		printRed("Error output:")
		fmt.Println(ctx.postPlanOutput)
		return err
	}

	// Save post-apply plan output
	postPlanLog := filepath.Join(ctx.tmpDir, "upgrade-post-plan.log")
	if err := os.WriteFile(postPlanLog, []byte(ctx.postPlanOutput), permFile); err != nil {
		printYellow("Warning: Failed to save post-apply plan log: %v", err)
	}

	// Check for ongoing drift
	postDriftResult := checkAndDisplayDriftForSU(ctx.postPlanOutput, ctx.cfg, "post-apply", ctx.resourceList, ctx.e2eV5Root)
	ctx.hasPostApplyChanges = postDriftResult.hasDrift
	ctx.postApplyDrift = postDriftResult.driftLines
	ctx.postApplyMaterial = postDriftResult.materialCount
	ctx.postApplyReal = postDriftResult.realCount
	ctx.postApplyExempted = postDriftResult.exemptedCount
	ctx.postApplyExemptedLines = postDriftResult.exemptedLines

	fmt.Println()
	return nil
}

// reportResults reports the final test results
// This follows the same reporting style as v4→v5 tests with drift report and step summary
func reportResults(ctx *suTestContext) error {
	// Display drift report if there were real changes OR exempted changes
	hasExemptedChanges := len(ctx.initialExemptedLines) > 0 || len(ctx.postApplyExemptedLines) > 0
	if ctx.hasChanges || ctx.hasPostApplyChanges || hasExemptedChanges {
		printHeader("Drift Report")

		if ctx.hasChanges && len(ctx.initialDrift) > 0 {
			printYellow("Real drift detected after upgrade (before apply):")
			displayGroupedDrift(ctx.initialDrift)
			fmt.Println()
		}

		if len(ctx.initialExemptedLines) > 0 {
			printSuccess("Exempted changes after upgrade (before apply):")
			printYellow("The following changes were detected but exempted by drift exemption rules:")
			displayGroupedDrift(ctx.initialExemptedLines)
			fmt.Println()
		}

		if ctx.hasPostApplyChanges && len(ctx.postApplyDrift) > 0 {
			printYellow("Ongoing drift detected (after apply):")
			displayGroupedDrift(ctx.postApplyDrift)
			fmt.Println()
		}

		if len(ctx.postApplyExemptedLines) > 0 {
			printSuccess("Exempted changes (after apply):")
			printYellow("The following changes were detected but exempted by drift exemption rules:")
			displayGroupedDrift(ctx.postApplyExemptedLines)
			fmt.Println()
		}
	}

	// Summary at the END (matching v4→v5 style)
	fmt.Println()
	if ctx.hasPostApplyChanges || ctx.hasChanges {
		printHeader("✗ V5 Upgrade Test Failed!")
	} else {
		printHeader("✓ V5 Upgrade Test Complete!")
	}

	printYellow("Summary:")
	printYellow("")
	printYellow("  Provider upgrade: %s → %s", ctx.cfg.FromVersion, ctx.cfg.ToVersion)
	printYellow("  State key:        %s", GenerateStateKey(ctx.cfg.FromVersion, ctx.cfg.ToVersion))
	printYellow("")

	printYellow("  Step 1: Initialization")
	printYellow("    Status: %s", colorGreen+"✓ SUCCESS"+colorReset)
	printYellow("")

	printYellow("  Step 2: Create resources with %s", ctx.cfg.FromVersion)
	if ctx.cfg.SkipCreate {
		printYellow("    Status: %s", colorYellow+"⊘ SKIPPED"+colorReset)
	} else {
		printYellow("    Status: %s", colorGreen+"✓ SUCCESS"+colorReset)
	}
	printYellow("")

	printYellow("  Step 3: Upgrade provider to %s", ctx.cfg.ToVersion)
	printYellow("    Status: %s", colorGreen+"✓ SUCCESS"+colorReset)
	printYellow("")

	printYellow("  Step 4: Plan after upgrade")
	if !ctx.hasChanges {
		printYellow("    Status: %s", colorGreen+"✓ No changes needed"+colorReset)
	} else {
		uniqueDrifts := countUniqueDrifts(ctx.initialDrift)
		printYellow("    Status: %s", colorRed+"✗ DRIFT DETECTED"+colorReset)
		if ctx.initialExempted > 0 {
			printYellow("    Result: %d real changes detected (%d exempted)", uniqueDrifts, ctx.initialExempted)
		} else {
			printYellow("    Result: %d real changes detected", uniqueDrifts)
		}
	}
	printYellow("")

	printYellow("  Step 5: Apply")
	if ctx.hasChanges {
		printYellow("    Status: %s", colorYellow+"⚠ Applied drift changes"+colorReset)
	} else {
		printYellow("    Status: %s", colorGreen+"✓ SUCCESS"+colorReset)
	}
	printYellow("")

	printYellow("  Step 6: Plan after apply (stability check)")
	if ctx.hasPostApplyChanges {
		uniqueDrifts := countUniqueDrifts(ctx.postApplyDrift)
		printYellow("    Status: %s", colorRed+"✗ FAILED - Resources keep changing"+colorReset)
		if ctx.postApplyExempted > 0 {
			printYellow("    Result: %d ongoing drift patterns (%d exempted)", uniqueDrifts, ctx.postApplyExempted)
		} else {
			printYellow("    Result: %d ongoing drift patterns", uniqueDrifts)
		}
	} else {
		if ctx.postApplyMaterial > 0 {
			printYellow("    Status: %s", colorGreen+"✓ SUCCESS - Stable state achieved"+colorReset)
			printYellow("    Result: %d material changes matched exemptions (%d unmatched)", ctx.postApplyMaterial, ctx.postApplyReal)
		} else {
			printYellow("    Status: %s", colorGreen+"✓ SUCCESS - Stable state achieved"+colorReset)
			printYellow("    Result: No changes detected")
		}
	}

	fmt.Println()
	printYellow("Logs saved to:")
	printCyan("  - %s", ctx.tmpDir)
	fmt.Println()

	// Exit with error if there's ongoing drift or if there were real changes in first plan
	if ctx.hasPostApplyChanges {
		printRed("Test failed: Resources are unstable and keep changing")
		printYellow("This prevents safe upgrade to %s - likely a provider bug", ctx.cfg.ToVersion)
		return fmt.Errorf("ongoing drift detected - resources keep changing")
	}

	if ctx.hasChanges {
		printRed("Test failed: Upgrade produced drift")
		printYellow("The upgraded state doesn't match your infrastructure")
		printYellow("Review the changes above and check for StateUpgrader bugs")
		return fmt.Errorf("upgrade produced drift")
	}

	return nil
}

// RunV5UpgradeClean removes specified modules from remote Terraform state
// This directly manipulates state JSON (pull, modify, push) to handle
// resources that may have been deleted outside of Terraform.
func RunV5UpgradeClean(cfg *V5UpgradeConfig, modules []string) error {
	// Normalize versions
	if cfg.FromVersion == "" {
		cfg.FromVersion = DefaultFromVersion
	}
	if cfg.ToVersion == "" {
		cfg.ToVersion = DefaultToVersion
	}

	// Load environment
	env, err := LoadEnv(EnvForClean)
	if err != nil {
		return err
	}

	// Get paths
	repoRoot := getRepoRoot()
	e2eV5Root := filepath.Join(repoRoot, "e2e-v5")
	tfDir := filepath.Join(e2eV5Root, "tf")

	printHeader("V5 Upgrade Cleanup")
	printCyan("Cleaning up: %s → %s", cfg.FromVersion, cfg.ToVersion)
	printCyan("State key: %s", GenerateStateKey(cfg.FromVersion, cfg.ToVersion))
	fmt.Println()

	// Check if tf directory exists
	if _, err := os.Stat(tfDir); os.IsNotExist(err) {
		printYellow("No tf directory found at %s - nothing to clean", tfDir)
		return nil
	}

	tf := NewTerraformRunner(tfDir)
	tf.EnvVars["AWS_ACCESS_KEY_ID"] = env.R2AccessKeyID
	tf.EnvVars["AWS_SECRET_ACCESS_KEY"] = env.R2SecretAccessKey

	// Check if terraform is initialized
	if _, err := os.Stat(filepath.Join(tfDir, ".terraform")); os.IsNotExist(err) {
		printYellow("Terraform not initialized - running init...")
		initArgs := []string{"init", "-no-color", "-input=false", "-backend-config=backend.configured.hcl"}
		tmpDir := filepath.Join(e2eV5Root, "tmp")
		os.MkdirAll(tmpDir, permDir)
		if err := tf.RunToFile(filepath.Join(tmpDir, "clean-init.log"), initArgs...); err != nil {
			printError("Terraform init failed")
			return err
		}
	}

	// If specific modules requested, clean them from state
	if len(modules) > 0 {
		printYellow("Modules to clean:")
		for _, module := range modules {
			printYellow("  - %s", module)
		}
		fmt.Println()

		// Pull latest state from remote
		printYellow("Pulling latest state from R2...")
		stateJSON, err := tf.Run("state", "pull")
		if err != nil {
			printError("Failed to pull state from R2")
			fmt.Println(stateJSON)
			return err
		}
		printSuccess("State pulled successfully")

		// Parse state
		var state map[string]interface{}
		if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
			return fmt.Errorf("failed to parse state: %w", err)
		}

		// Count resources before cleanup
		resourcesArray, ok := state["resources"].([]interface{})
		if !ok {
			return fmt.Errorf("invalid state format: resources array not found")
		}
		totalBefore := len(resourcesArray)
		printCyan("Total resources before cleanup: %d", totalBefore)
		fmt.Println()

		// Clean the state - remove specified modules
		printYellow("Cleaning state...")
		var filteredResources []interface{}
		removed := 0

		for _, res := range resourcesArray {
			resMap, ok := res.(map[string]interface{})
			if !ok {
				filteredResources = append(filteredResources, res)
				continue
			}

			resModule, ok := resMap["module"].(string)
			if !ok {
				filteredResources = append(filteredResources, res)
				continue
			}

			// Check if this resource belongs to any of the modules to clean
			shouldRemove := false
			for _, module := range modules {
				if resModule == "module."+module {
					shouldRemove = true
					removed++
					break
				}
			}

			if !shouldRemove {
				filteredResources = append(filteredResources, res)
			}
		}

		state["resources"] = filteredResources

		// Increment serial number
		if serial, ok := state["serial"].(float64); ok {
			state["serial"] = serial + 1
		}

		// Count resources after cleanup
		totalAfter := len(filteredResources)

		printSuccess("State cleaned")
		printCyan("  Before: %d resources", totalBefore)
		printCyan("  After:  %d resources", totalAfter)
		printSuccess("  Removed: %d resources", removed)
		fmt.Println()

		// Save cleaned state to temp file
		cleanedStateFile := filepath.Join(tfDir, "terraform.tfstate.cleaned")
		cleanedStateJSON, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal cleaned state: %w", err)
		}

		if err := os.WriteFile(cleanedStateFile, cleanedStateJSON, permSecretFile); err != nil {
			return fmt.Errorf("failed to write cleaned state to %s: %w", cleanedStateFile, err)
		}

		// Push cleaned state back to remote
		printYellow("Pushing cleaned state to R2...")
		output, err := tf.Run("state", "push", cleanedStateFile)
		if err != nil {
			printError("Failed to push state to R2")
			fmt.Println(output)
			printYellow("You can manually push it with: terraform state push %s", cleanedStateFile)
			return err
		}
		printSuccess("State pushed successfully")

		// Remove temp file
		os.Remove(cleanedStateFile)

		fmt.Println()
		printHeader("Cleanup Complete")
		printSuccess("Removed %d resources from %d modules", removed, len(modules))
		fmt.Println()

		return nil
	}

	// No specific modules - destroy all resources.
	//
	// Always ensure provider.tf is at the TARGET version before destroying.
	// After a successful upgrade the state is at schema_version=500; using the
	// source version provider would fail with "Resource instance managed by
	// newer provider version". This also handles the standalone --clean case
	// where runUpgradeProvider never ran and provider.tf is still at the source
	// version constraint.
	printYellow("Destroying all resources...")
	tmpDir := filepath.Join(e2eV5Root, "tmp")
	os.MkdirAll(tmpDir, permDir)

	// Always refresh provider.tf to target constraint for destroy.
	// This keeps cleanup deterministic and ensures required provider metadata
	// matches the active configuration (including auxiliary providers like tls).
	printYellow("Updating provider.tf to target version %s for destroy...", cfg.ToVersion)
	targetConstraint, _ := ResolveTargetVersion(cfg.ToVersion)
	providerContent := GenerateProviderTF(targetConstraint)
	providerFile := filepath.Join(tfDir, "provider.tf")
	if err := os.WriteFile(providerFile, []byte(providerContent), permFile); err != nil {
		return fmt.Errorf("failed to update provider.tf for destroy: %w", err)
	}

	if cfg.ProviderPath != "" {
		// Local provider: configure dev_overrides so destroy uses the local binary.
		// Keep lockfile and run init so auxiliary providers (e.g. hashicorp/tls)
		// are selected consistently before destroy.
		printYellow("Configuring local provider for destroy: %s", cfg.ProviderPath)
		tfConfigFile, err := setupLocalProvider(cfg.ProviderPath, repoRoot)
		if err != nil {
			return fmt.Errorf("failed to set up local provider for destroy: %w", err)
		}
		tf.TFConfigFile = tfConfigFile

		initArgs := []string{"init", "-no-color", "-input=false", "-backend-config=backend.configured.hcl", "-upgrade"}
		if err := tf.RunToFile(filepath.Join(tmpDir, "clean-init.log"), initArgs...); err != nil {
			printError("Terraform init for destroy failed")
			printYellow("See log: %s", filepath.Join(tmpDir, "clean-init.log"))
			return err
		}
		printSuccess("Provider and lockfile initialized for destroy")
	} else {
		// Registry provider: run init -upgrade so Terraform installs target binaries.
		initArgs := []string{"init", "-no-color", "-input=false", "-backend-config=backend.configured.hcl", "-upgrade"}
		if err := tf.RunToFile(filepath.Join(tmpDir, "clean-init.log"), initArgs...); err != nil {
			printError("Terraform init -upgrade for destroy failed")
			printYellow("See log: %s", filepath.Join(tmpDir, "clean-init.log"))
			return err
		}
		printSuccess("Provider updated to target version for destroy")
	}

	// Run terraform destroy
	destroyArgs := []string{"destroy", "-no-color", "-input=false", "-auto-approve"}
	printYellow("Running terraform destroy...")
	destroyLog := filepath.Join(tmpDir, "clean-destroy.log")
	if err := tf.RunToFile(destroyLog, destroyArgs...); err != nil {
		printError("Terraform destroy failed")
		printYellow("See log: %s", destroyLog)
		return err
	}
	printSuccess("Resources destroyed")
	fmt.Println()

	printHeader("Cleanup Complete")
	printSuccess("State key %s has been cleaned", GenerateStateKey(cfg.FromVersion, cfg.ToVersion))
	fmt.Println()

	return nil
}

// setupLocalProvider builds the local provider and creates dev_overrides config
func setupLocalProvider(providerPath, repoRoot string) (string, error) {
	printHeader("Setting up local provider")
	printYellow("Using provider from: %s", providerPath)

	// Determine if providerPath is a file or directory
	var providerBinary string
	var providerDir string

	info, err := os.Stat(providerPath)
	if err != nil {
		if os.IsNotExist(err) {
			providerBinary = providerPath
			providerDir = filepath.Dir(providerPath)
			if dirInfo, dirErr := os.Stat(providerDir); dirErr != nil || !dirInfo.IsDir() {
				return "", fmt.Errorf("provider directory does not exist: %s", providerDir)
			}
		} else {
			return "", fmt.Errorf("failed to check provider path: %w", err)
		}
	} else if info.IsDir() {
		providerDir = providerPath
		providerBinary = filepath.Join(providerDir, "terraform-provider-cloudflare")
	} else {
		providerBinary = providerPath
		providerDir = filepath.Dir(providerPath)
	}

	// Build the provider
	printYellow("Building provider...")
	buildCmd := exec.Command("go", "build", "-o", providerBinary, ".")
	buildCmd.Dir = providerDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		printError("Failed to build provider: %v", err)
		return "", fmt.Errorf("failed to build provider: %w", err)
	}
	printSuccess("Provider built successfully: %s", providerBinary)

	// Create dev overrides config
	tfConfigFile := filepath.Join(repoRoot, ".terraformrc-su-test")
	absProviderDir, err := filepath.Abs(providerDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for provider directory: %w", err)
	}

	configContent := fmt.Sprintf(`provider_installation {
  dev_overrides {
    "cloudflare/cloudflare" = "%s"
  }
  direct {}
}
`, absProviderDir)

	if err := os.WriteFile(tfConfigFile, []byte(configContent), permFile); err != nil {
		return "", fmt.Errorf("failed to create provider config at %s: %w", tfConfigFile, err)
	}

	printSuccess("Created dev overrides config: %s", tfConfigFile)
	fmt.Println()

	return tfConfigFile, nil
}

// targetArgs returns terraform target arguments for the test context
func (ctx *suTestContext) targetArgs() []string {
	var args []string
	for _, resource := range ctx.resourceList {
		args = append(args, "-target=module."+resource)
	}
	return args
}

// checkAndDisplayDriftForSU checks drift for v5 upgrade tests using e2e-v5/ exemptions.
//
// Unlike checkAndDisplayDrift (which hardcodes e2e/), this function loads exemptions
// from e2eV5Root (e2e-v5/) so that v5-specific exemption rules are applied.
func checkAndDisplayDriftForSU(planOutput string, cfg *V5UpgradeConfig, phase string, resourceList []string, e2eV5Root string) driftCheckResult {
	result := driftCheckResult{}

	if strings.Contains(planOutput, "No changes") {
		printSuccess("No drift detected - state upgrade successful!")
		return result
	}

	// Load exemptions from e2e-v5/ directory (not e2e/)
	exemptionConfig, err := LoadDriftExemptionsFromDir(e2eV5Root, resourceList)
	if err != nil {
		printYellow("Warning: failed to load v5 drift exemptions from %s: %v", e2eV5Root, err)
		// Fall back to standard drift check without exemptions
		runCfg := &RunConfig{ApplyExemptions: false}
		return checkAndDisplayDrift(planOutput, runCfg, phase, resourceList)
	}

	// Override ApplyExemptions from config
	if !cfg.ApplyExemptions {
		exemptionConfig.Settings.ApplyExemptions = false
	}

	driftResult := CheckDriftWithConfig(planOutput, exemptionConfig)

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
		printYellow("Drift breakdown: %d material, %d computed refresh, %d exempted",
			result.materialCount, len(driftResult.ComputedRefreshLines), totalExempted)
	}

	if driftResult.OnlyComputedChanges {
		if phase == "upgrade" {
			printSuccess("Only computed value refreshes detected after upgrade (exempted) - state upgrade successful!")
		} else {
			printSuccess("Only computed value refreshes detected (exempted) - state is stable!")
		}

		// Show plan summary even for exempted changes
		planSummary := extractPlanSummary(planOutput)
		if planSummary != "" {
			fmt.Println()
			printYellow("Plan summary: %s", planSummary)
		}

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

	if phase == "upgrade" {
		printRed("Drift detected after state upgrade:")
	} else {
		printRed("Ongoing drift detected after apply:")
	}

	// Display plan summary and changes
	planSummary := extractPlanSummary(planOutput)
	if planSummary != "" {
		fmt.Println()
		printYellow("Plan summary: %s", planSummary)
	}
	fmt.Println()
	fmt.Println(extractPlanChanges(planOutput))

	// Also list drift lines for quick reference
	if len(driftResult.RealDriftLines) > 0 {
		fmt.Println()
		printYellow("Affected resources:")
		for _, line := range driftResult.RealDriftLines {
			printRed("  %s", line)
		}
	}

	return result
}
