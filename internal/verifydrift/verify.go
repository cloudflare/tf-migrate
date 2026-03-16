package verifydrift

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	e2e "github.com/cloudflare/tf-migrate/internal/e2e-runner"
)

// VerifyResult holds the outcome of analysing a terraform plan against the
// known migration drift exemptions.
type VerifyResult struct {
	// DetectedResources is the list of Cloudflare resource types found in the plan
	// (e.g. ["dns_record", "zone_setting"]). Derived automatically from the plan output.
	DetectedResources []string

	// ExemptedGroups groups exempted plan changes by the exemption rule that matched
	// them. Each group carries the rule name, description, and the matching plan lines.
	ExemptedGroups []ExemptedGroup

	// UnexpectedDrift contains plan lines that were not matched by any exemption rule.
	// These require the customer's attention.
	UnexpectedDrift []string

	// ComputedLines contains "(known after apply)" lines that are purely informational
	// and always ignored.
	ComputedLines []string

	// HasUnexpected is true when UnexpectedDrift is non-empty.
	HasUnexpected bool
}

// ExemptedGroup is a set of plan lines matched by a single exemption rule.
type ExemptedGroup struct {
	// RuleName is the exemption's unique name (from the YAML `name:` field).
	RuleName string

	// Description is the human-readable explanation from the YAML `description:` field.
	Description string

	// Lines are the raw plan diff lines that this rule matched (with resource context prefix).
	Lines []string
}

// Verify analyses planText against the embedded drift exemptions and returns
// a structured result. Resources are auto-detected from the plan output.
func Verify(planText string) (VerifyResult, error) {
	cfg, err := loadEmbeddedExemptions(planText)
	if err != nil {
		return VerifyResult{}, fmt.Errorf("loading drift exemptions: %w", err)
	}

	resources := e2e.DetectResourcesFromPlan(planText)
	result := e2e.CheckDriftWithConfig(planText, cfg)

	// Build a name→description lookup from the loaded config.
	descByName := make(map[string]string, len(cfg.Exemptions))
	for _, ex := range cfg.Exemptions {
		descByName[ex.Name] = ex.Description
	}

	// Re-group exempted lines by rule name.
	groups := groupExemptedLines(result.ExemptedDriftLines, descByName)

	return VerifyResult{
		DetectedResources: resources,
		ExemptedGroups:    groups,
		UnexpectedDrift:   result.RealDriftLines,
		ComputedLines:     result.ComputedRefreshLines,
		HasUnexpected:     len(result.RealDriftLines) > 0,
	}, nil
}

// loadEmbeddedExemptions builds a *DriftExemptionsConfig by reading from the
// embedded FS. It always loads all per-resource YAML files regardless of the
// plan contents — the full set is small and it ensures no exemption is missed.
func loadEmbeddedExemptions(planText string) (*e2e.DriftExemptionsConfig, error) {
	// Write the embedded FS to a temp directory so LoadDriftExemptionsFromDir
	// (which expects an os.DirFS-style path) can consume it. This avoids
	// duplicating the YAML parsing logic.
	//
	// We use a virtual approach instead: parse directly from the embed.FS.
	return loadFromFS(embeddedExemptions)
}

// loadFromFS parses a DriftExemptionsConfig from an fs.FS whose layout mirrors
// the e2e/ directory:
//
//	exemptions/global-drift-exemptions.yaml
//	exemptions/drift-exemptions/<resource>.yaml
func loadFromFS(fsys fs.FS) (*e2e.DriftExemptionsConfig, error) {
	globalData, err := fs.ReadFile(fsys, "exemptions/global-drift-exemptions.yaml")
	if err != nil {
		return nil, fmt.Errorf("reading global exemptions: %w", err)
	}

	globalCfg, err := e2e.ParseDriftExemptionsConfig(globalData, "global")
	if err != nil {
		return nil, fmt.Errorf("parsing global exemptions: %w", err)
	}

	resourceCfgs := make(map[string]*e2e.DriftExemptionsConfig)

	if globalCfg.Settings.LoadResourceExemptions {
		entries, err := fs.ReadDir(fsys, "exemptions/drift-exemptions")
		if err != nil {
			return nil, fmt.Errorf("listing resource exemptions: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}
			resource := strings.TrimSuffix(entry.Name(), ".yaml")
			path := path.Join("exemptions", "drift-exemptions", entry.Name())

			data, err := fs.ReadFile(fsys, path)
			if err != nil {
				return nil, fmt.Errorf("reading exemptions for %s: %w", resource, err)
			}
			cfg, err := e2e.ParseDriftExemptionsConfig(data, fmt.Sprintf("resource:%s", resource))
			if err != nil {
				return nil, fmt.Errorf("parsing exemptions for %s: %w", resource, err)
			}
			resourceCfgs[resource] = cfg
		}
	}

	merged, err := e2e.MergeAndCompileExemptions(globalCfg, resourceCfgs)
	if err != nil {
		return nil, fmt.Errorf("merging exemptions: %w", err)
	}
	return merged, nil
}

// groupExemptedLines parses the [exempted: <name>] tag appended to each
// exempted drift line and groups them into ExemptedGroup slices.
// Lines that don't carry a tag are placed in an "unknown" group.
func groupExemptedLines(lines []string, descByName map[string]string) []ExemptedGroup {
	// Preserve insertion order of rule names for deterministic output.
	order := []string{}
	byName := make(map[string]*ExemptedGroup)

	for _, line := range lines {
		name, cleanLine := parseExemptionTag(line)
		if name == "" {
			name = "unknown"
		}
		if _, exists := byName[name]; !exists {
			desc := descByName[name]
			byName[name] = &ExemptedGroup{
				RuleName:    name,
				Description: desc,
			}
			order = append(order, name)
		}
		byName[name].Lines = append(byName[name].Lines, cleanLine)
	}

	result := make([]ExemptedGroup, 0, len(order))
	for _, name := range order {
		result = append(result, *byName[name])
	}
	return result
}

// parseExemptionTag extracts the rule name from the trailing " [exempted: <name>]"
// suffix added by hasOnlyComputedChangesWithExemptions, and returns both the name
// and the line with the tag stripped.
func parseExemptionTag(line string) (name string, cleanLine string) {
	const prefix = "[exempted: "
	idx := strings.LastIndex(line, prefix)
	if idx == -1 {
		return "", line
	}
	tag := line[idx:]
	if !strings.HasSuffix(tag, "]") {
		return "", line
	}
	name = tag[len(prefix) : len(tag)-1]
	cleanLine = strings.TrimRight(line[:idx], " ")
	return name, cleanLine
}
