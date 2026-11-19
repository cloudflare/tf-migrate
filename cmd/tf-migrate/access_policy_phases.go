package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal/accesspolicyphase"
)

// accessPolicyDir is a directory with application-scoped Access policies.
type accessPolicyDir struct {
	dir      string
	policies []accesspolicyphase.Policy
	skipped  []string
	state    accesspolicyphase.State
}

// findLocalModuleCalls maps each local module directory to its call. Only a
// module called once, without count or for_each, from a directory that is not
// itself a module, is supported: its policies' imports can then be written
// in the caller as module.<name>.<address>. Other module directories are
// returned in unsupported with the reason.
func findLocalModuleCalls(cfg config) (calls map[string]accesspolicyphase.ModuleCall, unsupported map[string]string, err error) {
	files, err := findTerraformFilesWithRecursion(cfg.configDir, cfg.recursive, cfg.exclude)
	if err != nil {
		return nil, nil, err
	}
	type call struct {
		accesspolicyphase.ModuleCall
		repeated bool
	}
	var found = map[string][]call{}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		f, diags := hclwrite.ParseConfig(content, filepath.Base(file), hcl.InitialPos)
		if diags.HasErrors() {
			continue
		}
		for _, b := range f.Body().Blocks() {
			if b.Type() != "module" || len(b.Labels()) != 1 || b.Body().GetAttribute("source") == nil {
				continue
			}
			src := strings.Trim(strings.TrimSpace(string(b.Body().GetAttribute("source").Expr().BuildTokens(nil).Bytes())), `"`)
			if !strings.HasPrefix(src, "./") && !strings.HasPrefix(src, "../") {
				continue
			}
			dir := filepath.Clean(filepath.Join(filepath.Dir(file), src))
			found[dir] = append(found[dir], call{
				ModuleCall: accesspolicyphase.ModuleCall{CallerDir: filepath.Clean(filepath.Dir(file)), Name: b.Labels()[0]},
				repeated:   b.Body().GetAttribute("count") != nil || b.Body().GetAttribute("for_each") != nil,
			})
		}
	}
	calls = map[string]accesspolicyphase.ModuleCall{}
	unsupported = map[string]string{}
	for dir, cs := range found {
		switch {
		case len(cs) > 1:
			unsupported[dir] = "is in a module called more than once"
		case cs[0].repeated:
			unsupported[dir] = "is in a module called with count or for_each"
		case len(found[cs[0].CallerDir]) > 0:
			unsupported[dir] = "is in a nested module (imports must be in the root module)"
		default:
			calls[dir] = cs[0].ModuleCall
		}
	}
	return calls, unsupported, nil
}

func scanAccessPolicyDirs(cfg config) ([]accessPolicyDir, error) {
	files, err := findTerraformFilesWithRecursion(cfg.configDir, cfg.recursive, cfg.exclude)
	if err != nil {
		return nil, err
	}
	calls, unsupportedModules, err := findLocalModuleCalls(cfg)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var dirs []string
	for _, f := range files {
		if d := filepath.Dir(f); !seen[d] {
			seen[d] = true
			dirs = append(dirs, d)
		}
	}
	sort.Strings(dirs)
	var out []accessPolicyDir
	for _, d := range dirs {
		policies, skipped, err := accesspolicyphase.Scan(d)
		if err != nil {
			return nil, err
		}
		state := accesspolicyphase.StateOf(d)
		reason, ok := unsupportedModules[filepath.Clean(d)]
		declaredRoot := cfg.rootModule && filepath.Clean(d) == filepath.Clean(cfg.configDir)
		if _, called := calls[filepath.Clean(d)]; !ok && !called && !declaredRoot && len(policies) > 0 && !looksLikeRoot(d) {
			// A module run on its own (e.g. a shared-module repo): step 4's
			// import blocks would land in the module, and Terraform only
			// allows them in the root module.
			reason, ok = "is in a directory with no provider block, backend, lock file or state, so it looks like a module called from outside --config-dir (run tf-migrate from the root module, with --recursive; if this is the root module, pass --root-module)", true
		}
		if ok {
			for _, p := range policies {
				skipped = append(skipped, fmt.Sprintf("%s.%s: %s", p.Type, p.Name, reason))
			}
			policies = nil
		}
		if len(policies) == 0 && len(skipped) == 0 && state == accesspolicyphase.NoRecord {
			continue
		}
		out = append(out, accessPolicyDir{dir: d, policies: policies, skipped: skipped, state: state})
	}
	return out, nil
}

// runAccessPolicyRecordStep runs step 1 of the Access policy migration (see
// internal/accesspolicyphase) where it is due. It returns done=true when the
// run should stop here, and the directories whose record step 2 must freeze.
func runAccessPolicyRecordStep(log hclog.Logger, cfg config) (done bool, toFreeze []string, err error) {
	dirs, err := scanAccessPolicyDirs(cfg)
	if err != nil {
		return false, nil, err
	}
	var record []accessPolicyDir
	var recorded []string
	var refs []string
	for _, d := range dirs {
		for _, s := range d.skipped {
			fmt.Printf("⚠ %s: %s. It stays on the manual migration path.\n", relDir(cfg, d.dir), s)
		}
		for _, r := range accesspolicyphase.References(d.dir, d.policies) {
			refs = append(refs, filepath.Join(relDir(cfg, d.dir), r))
		}
		switch {
		case len(d.policies) == 0:
			continue
		case d.state == accesspolicyphase.NoRecord:
			record = append(record, d)
		case d.state == accesspolicyphase.Recorded:
			// A policy added or removed since the record was written: rewrite
			// it and apply again before anything is removed from state.
			current, _ := os.ReadFile(filepath.Join(d.dir, accesspolicyphase.RecordFile))
			if !bytes.Equal(current, accesspolicyphase.RenderRecord(d.policies)) {
				record = append(record, d)
			} else {
				recorded = append(recorded, d.dir)
			}
		}
	}

	if len(record) > 0 {
		fmt.Println("Access policy migration — step 1 of 4: record policy IDs")
		fmt.Println("=========================================================")
		fmt.Println()
		for _, d := range record {
			path := filepath.Join(d.dir, accesspolicyphase.RecordFile)
			if cfg.dryRun {
				fmt.Printf("(dry run) would write %s (%d policies)\n", path, len(d.policies))
				continue
			}
			if err := os.WriteFile(path, accesspolicyphase.RenderRecord(d.policies), 0644); err != nil {
				return false, nil, fmt.Errorf("write %s: %w", path, err)
			}
			fmt.Printf("✓ wrote %s (%d policies)\n", path, len(d.policies))
			if c, ok := accesspolicyphase.ModuleOf(d.dir); ok {
				if err := os.WriteFile(accesspolicyphase.CallerFile(c), accesspolicyphase.RenderCaller(c, d.policies, false), 0644); err != nil {
					return false, nil, err
				}
				fmt.Printf("✓ wrote %s (re-exports module.%s's record)\n", accesspolicyphase.CallerFile(c), c.Name)
			}
		}
		printPolicyReferences(refs)
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  1. Commit, plan and apply with the current (v4) provider.")
		fmt.Println("     Expect only terraform_data." + accesspolicyphase.Name + " to be added; the plan")
		fmt.Println("     output shows each recorded policy ID.")
		fmt.Println("  2. Re-run tf-migrate for step 2.")
		log.Debug("Access policy record step written", "dirs", len(record))
		return true, nil, nil
	}

	if len(recorded) == 0 {
		return false, nil, nil
	}
	if cfg.skipPhaseCheck {
		return false, recorded, nil
	}
	if len(refs) > 0 {
		// Step 2 removes the policies from config; these would fail the plan.
		printPolicyReferences(refs)
		fmt.Println("Remove these references, then re-run tf-migrate for step 2.")
		return true, nil, nil
	}
	fmt.Println()
	fmt.Println("The Access policy record (step 1) is written in:")
	for _, d := range recorded {
		fmt.Printf("  • %s\n", relDir(cfg, d))
	}
	fmt.Print("Did you apply it with the v4 provider? [y/N]: ")
	if !readYes() {
		fmt.Println()
		fmt.Println("Apply step 1 first, then re-run tf-migrate.")
		return true, nil, nil
	}
	return false, recorded, nil
}

// freezeAccessPolicyRecords writes the step 2 record into dirs.
func freezeAccessPolicyRecords(cfg config, dirs []string) error {
	if cfg.dryRun || len(dirs) == 0 {
		return nil
	}
	for _, d := range dirs {
		path := filepath.Join(d, accesspolicyphase.RecordFile)
		if err := os.WriteFile(path, accesspolicyphase.RenderFrozen(), 0644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		fmt.Printf("✓ %s — record frozen\n", path)
	}
	fmt.Println()
	fmt.Println("Access policy migration — this is step 2 of 4 (remove from state).")
	fmt.Println("Step 3 of 4, after this apply and before re-running tf-migrate:")
	fmt.Println("  Convert the recorded policies to reusable ones (same IDs, still attached):")
	for _, d := range dirs {
		root, output := d, accesspolicyphase.Name
		if c, ok := accesspolicyphase.ModuleOf(d); ok {
			root, output = c.CallerDir, accesspolicyphase.CallerOutput(c)
		}
		ids := filepath.Join(root, output+".json")
		fmt.Printf("    terraform -chdir=%s output -json %s > %s\n", root, output, ids)
		fmt.Printf("    tf-migrate make-access-policies-reusable --ids %s\n", ids)
	}
	return nil
}

// confirmAccessPoliciesReusable asks before step 3 when a frozen record exists.
func confirmAccessPoliciesReusable(cfg config) (bool, error) {
	dirs, err := scanAccessPolicyDirs(cfg)
	if err != nil {
		return false, err
	}
	var frozen []string
	for _, d := range dirs {
		if d.state == accesspolicyphase.Frozen {
			frozen = append(frozen, relDir(cfg, d.dir))
		}
	}
	if len(frozen) == 0 || cfg.skipPhaseCheck {
		return true, nil
	}
	fmt.Println("Access policy migration — step 4 of 4: migrate to v5")
	fmt.Printf("Did you run make-access-policies-reusable (step 3) for %s? [y/N]: ", strings.Join(frozen, ", "))
	return readYes(), nil
}

func readYes() bool {
	var answer string
	if _, err := fmt.Scanln(&answer); err != nil {
		return false
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}

func relDir(cfg config, dir string) string {
	if r, err := filepath.Rel(cfg.configDir, dir); err == nil {
		return r
	}
	return dir
}

// moduleImports returns, for each frozen local-module directory, the
// policies whose imports its caller needs. Call it after the phase-1 blocks
// are restored and before the v5 migration rewrites them.
func moduleImports(cfg config) (map[string][]accesspolicyphase.Policy, error) {
	dirs, err := scanAccessPolicyDirs(cfg)
	if err != nil {
		return nil, err
	}
	out := map[string][]accesspolicyphase.Policy{}
	for _, d := range dirs {
		if _, ok := accesspolicyphase.ModuleOf(d.dir); ok && d.state == accesspolicyphase.Frozen && len(d.policies) > 0 {
			out[d.dir] = d.policies
		}
	}
	return out, nil
}

// writeModuleImports writes the import blocks for local-module policies into
// their callers.
func writeModuleImports(cfg config, imports map[string][]accesspolicyphase.Policy) error {
	if cfg.dryRun {
		return nil
	}
	for dir, policies := range imports {
		c, _ := accesspolicyphase.ModuleOf(dir)
		if err := os.WriteFile(accesspolicyphase.CallerFile(c), accesspolicyphase.RenderCaller(c, policies, true), 0644); err != nil {
			return err
		}
		fmt.Printf("✓ %s — imports for module.%s's policies\n", accesspolicyphase.CallerFile(c), c.Name)
	}
	return nil
}

// setAccessPolicyModules tells the migrators which directories are local
// modules.
func setAccessPolicyModules(cfg config) error {
	calls, _, err := findLocalModuleCalls(cfg)
	if err != nil {
		return err
	}
	accesspolicyphase.SetModules(calls)
	return nil
}

// looksLikeRoot reports whether dir has something only a root module usually
// has: a provider block, a backend or cloud block, a lock file or a local
// state file.
func looksLikeRoot(dir string) bool {
	for _, f := range []string{".terraform.lock.hcl", "terraform.tfstate"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			return true
		}
	}
	paths, _ := filepath.Glob(filepath.Join(dir, "*.tf"))
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		f, diags := hclwrite.ParseConfig(content, filepath.Base(path), hcl.InitialPos)
		if diags.HasErrors() {
			continue
		}
		for _, b := range f.Body().Blocks() {
			if b.Type() == "provider" {
				return true
			}
			if b.Type() == "terraform" {
				for _, inner := range b.Body().Blocks() {
					if inner.Type() == "backend" || inner.Type() == "cloud" {
						return true
					}
				}
			}
		}
	}
	return false
}

func printPolicyReferences(refs []string) {
	if len(refs) == 0 {
		return
	}
	fmt.Println()
	fmt.Println("⚠ These lines refer to app-scoped policies that step 2 removes from the")
	fmt.Println("  configuration, so step 2's plan would fail (\"Reference to undeclared")
	fmt.Println("  resource\"; for an import block, its target no longer exists in config).")
	fmt.Println("  Remove them before step 2 (step 2 waits until they are gone):")
	for _, r := range refs {
		fmt.Printf("  • %s\n", r)
	}
}
