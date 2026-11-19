package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/go-hclog"

	"github.com/cloudflare/tf-migrate/internal/accesspolicyphase"
)

// accessPolicyDir is a directory with application-scoped Access policies.
type accessPolicyDir struct {
	dir      string
	policies []accesspolicyphase.Policy
	skipped  []string
	state    accesspolicyphase.State
}

func scanAccessPolicyDirs(cfg config) ([]accessPolicyDir, error) {
	files, err := findTerraformFilesWithRecursion(cfg.configDir, cfg.recursive, cfg.exclude)
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
	for _, d := range dirs {
		for _, s := range d.skipped {
			fmt.Printf("⚠ %s: %s. It stays on the manual migration path.\n", relDir(cfg, d.dir), s)
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
		fmt.Println("Access policy migration — step 1 of 3: record policy IDs")
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
		}
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
	fmt.Println("Access policies — after this apply, before re-running tf-migrate:")
	fmt.Println("  Convert the recorded policies to reusable ones (same IDs, still attached):")
	for _, d := range dirs {
		fmt.Printf("    terraform -chdir=%s output -json %s > %s.json\n", d, accesspolicyphase.Name, filepath.Join(d, "access-policy-ids"))
		fmt.Printf("    tf-migrate make-access-policies-reusable --ids %s.json\n", filepath.Join(d, "access-policy-ids"))
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
	fmt.Printf("Did you run make-access-policies-reusable for %s? [y/N]: ", strings.Join(frozen, ", "))
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
