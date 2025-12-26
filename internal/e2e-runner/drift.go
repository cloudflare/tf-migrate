// drift.go implements sophisticated drift detection for Terraform provider migrations.
//
// This file provides tools to analyze Terraform plan output and detect infrastructure
// drift between provider versions. It supports:
//   - Parsing plan output to extract resource changes
//   - Identifying computed-only changes that can be safely exempted
//   - Loading drift exemption rules from YAML configuration
//   - Generating detailed drift reports with actionable information
//
// Drift detection is critical for validating that provider migrations don't
// introduce unexpected infrastructure changes.
package e2e

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Package-level compiled regex patterns for drift detection (performance optimization)
var (
	resourceHeaderPattern      = regexp.MustCompile(`^\s*#\s+(\S+)\s+will be`)
	updateLinePattern          = regexp.MustCompile(`^\s+~\s+\w+\s+=`)
	arrowPattern               = regexp.MustCompile(`->`)
	additionPattern            = regexp.MustCompile(`^\s+\+\s+\w+\s+=`)
	deletionPattern            = regexp.MustCompile(`^\s+\-\s+\w+\s+=`)
	resourceDeclarationPattern = regexp.MustCompile(`^\s+[+~-]\s+(resource|data)\s+"`)
	knownAfterApplyPattern     = regexp.MustCompile(`\(known after apply\)`)
	statusActivePattern        = regexp.MustCompile(`status.*->.*"active"`)
	planPattern                = regexp.MustCompile(`^Plan:.*`)
)

// DriftExemption represents a single drift exemption rule
type DriftExemption struct {
	Name             string           `yaml:"name"`
	Description      string           `yaml:"description"`
	ResourceTypes    []string         `yaml:"resource_types,omitempty"`
	Attributes       []string         `yaml:"attributes,omitempty"`
	Patterns         []string         `yaml:"patterns"`
	Enabled          bool             `yaml:"enabled"`
	compiledPatterns []*regexp.Regexp // Pre-compiled patterns for performance
}

// DriftExemptionsConfig represents the drift exemptions configuration file
type DriftExemptionsConfig struct {
	Exemptions []DriftExemption `yaml:"exemptions"`
	Settings   struct {
		ApplyExemptions    bool `yaml:"apply_exemptions"`
		VerboseExemptions  bool `yaml:"verbose_exemptions"`
	} `yaml:"settings"`
}

// loadDriftExemptions loads the drift exemptions from the YAML config file
func loadDriftExemptions() (*DriftExemptionsConfig, error) {
	repoRoot := getRepoRoot()
	configPath := filepath.Join(repoRoot, "e2e", "drift-exemptions.yaml")

	// If file doesn't exist, return empty config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &DriftExemptionsConfig{
			Exemptions: []DriftExemption{},
			Settings: struct {
				ApplyExemptions   bool `yaml:"apply_exemptions"`
				VerboseExemptions bool `yaml:"verbose_exemptions"`
			}{
				ApplyExemptions:   false,
				VerboseExemptions: false,
			},
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config DriftExemptionsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Pre-compile regex patterns for performance
	for i := range config.Exemptions {
		exemption := &config.Exemptions[i]
		exemption.compiledPatterns = make([]*regexp.Regexp, 0, len(exemption.Patterns))

		for _, pattern := range exemption.Patterns {
			compiled, err := regexp.Compile(pattern)
			if err != nil {
				// Fail fast on invalid regex patterns - don't silently skip
				return nil, fmt.Errorf("invalid regex pattern '%s' in exemption '%s': %w", pattern, exemption.Name, err)
			}
			exemption.compiledPatterns = append(exemption.compiledPatterns, compiled)
		}
	}

	return &config, nil
}

// DriftCheckResult holds the result of drift detection
type DriftCheckResult struct {
	OnlyComputedChanges   bool
	TriggeredExemptions   map[string]int // exemption name -> count of matches
	ExemptionsEnabled     bool
	RealDriftLines        []string       // actual drift detected (non-exempted changes)
}

// hasOnlyComputedChanges checks if a terraform plan only has "known after apply" changes
// Returns true if only computed refreshes or acceptable changes, false if real drift
func hasOnlyComputedChanges(planOutput string) bool {
	result := checkDrift(planOutput)
	return result.OnlyComputedChanges
}

// checkDrift performs drift detection and returns detailed results
func checkDrift(planOutput string) DriftCheckResult {
	// Load exemptions from config
	config, err := loadDriftExemptions()
	if err != nil {
		// If we can't load config, fall back to default behavior
		return DriftCheckResult{
			OnlyComputedChanges: hasOnlyComputedChangesDefault(planOutput),
			TriggeredExemptions: make(map[string]int),
			ExemptionsEnabled:   false,
			RealDriftLines:      []string{},
		}
	}

	// If exemptions are disabled, use default behavior
	if !config.Settings.ApplyExemptions {
		return DriftCheckResult{
			OnlyComputedChanges: hasOnlyComputedChangesDefault(planOutput),
			TriggeredExemptions: make(map[string]int),
			ExemptionsEnabled:   false,
			RealDriftLines:      []string{},
		}
	}

	onlyComputed, triggeredExemptions, realDriftLines := hasOnlyComputedChangesWithExemptions(planOutput, config)
	return DriftCheckResult{
		OnlyComputedChanges: onlyComputed,
		TriggeredExemptions: triggeredExemptions,
		ExemptionsEnabled:   true,
		RealDriftLines:      realDriftLines,
	}
}

// hasOnlyComputedChangesDefault is the original implementation without exemptions
func hasOnlyComputedChangesDefault(planOutput string) bool {
	scanner := bufio.NewScanner(strings.NewReader(planOutput))

	// Patterns to detect
	hasRealChanges := false
	hasAdditions := false
	hasDeletions := false

	for scanner.Scan() {
		line := scanner.Text()

		// Check for update lines
		if updateLinePattern.MatchString(line) {
			// Skip if it's a "known after apply" change
			if knownAfterApplyPattern.MatchString(line) {
				continue
			}

			// Skip if it's a status -> "active" change
			if statusActivePattern.MatchString(line) {
				continue
			}

			// If it has an arrow (->), it's a real change
			if arrowPattern.MatchString(line) {
				hasRealChanges = true
			}
		}

		// Check for additions
		if additionPattern.MatchString(line) {
			hasAdditions = true
		}

		// Check for deletions
		if deletionPattern.MatchString(line) {
			hasDeletions = true
		}
	}

	// Return true (only computed) if no real changes, additions, or deletions
	return !hasRealChanges && !hasAdditions && !hasDeletions
}

// hasOnlyComputedChangesWithExemptions checks drift using exemption rules from config
// Returns whether only computed changes exist, a map of triggered exemptions, and the real drift lines
func hasOnlyComputedChangesWithExemptions(planOutput string, config *DriftExemptionsConfig) (bool, map[string]int, []string) {
	scanner := bufio.NewScanner(strings.NewReader(planOutput))

	// Patterns to detect
	hasRealChanges := false
	hasAdditions := false
	hasDeletions := false
	currentResourceType := ""
	currentResourceName := ""
	triggeredExemptions := make(map[string]int)
	realDriftLines := []string{}

	// Helper function to check if a line is exempted
	checkExemption := func(line string) (bool, string) {
		for _, exemption := range config.Exemptions {
			if !exemption.Enabled {
				continue
			}

			// Check if resource type matches (if specified)
			resourceTypeMatches := len(exemption.ResourceTypes) == 0
			if !resourceTypeMatches {
				for _, resourceType := range exemption.ResourceTypes {
					if resourceType == currentResourceType {
						resourceTypeMatches = true
						break
					}
				}
			}

			if !resourceTypeMatches {
				continue
			}

			// Check if line matches any pattern (using pre-compiled patterns)
			for _, compiledPattern := range exemption.compiledPatterns {
				if compiledPattern.MatchString(line) {
					if config.Settings.VerboseExemptions {
						printYellow("  [Exempted:%s] %s", exemption.Name, line)
					}
					return true, exemption.Name
				}
			}
		}
		return false, ""
	}

	for scanner.Scan() {
		line := scanner.Text()

		// Track current resource from lines like: # module.foo.cloudflare_zone.example will be updated
		if matches := resourceHeaderPattern.FindStringSubmatch(line); len(matches) > 1 {
			currentResourceName = matches[1]
			// Extract resource type (e.g., cloudflare_zone from module.foo.cloudflare_zone.example)
			parts := strings.Split(matches[1], ".")
			if len(parts) >= 2 {
				// Safe: len(parts) >= 2 means parts[len(parts)-2] is at index 0 or higher
				currentResourceType = parts[len(parts)-2]
			} else {
				// Fallback: if resource name doesn't have expected format, clear type
				currentResourceType = ""
			}
		}

		// Skip resource/data block declarations (like "+ resource" or "~ data")
		if resourceDeclarationPattern.MatchString(line) {
			continue
		}

		// Check for update lines
		if updateLinePattern.MatchString(line) {
			// Check if this line matches any exemption pattern
			isExempted, matchedExemptionName := checkExemption(line)

			if isExempted {
				// Track this exemption
				triggeredExemptions[matchedExemptionName]++
				continue
			}

			// If it has an arrow (->), it's a real change
			if arrowPattern.MatchString(line) {
				hasRealChanges = true
				// Store the drift line with resource context
				if currentResourceName != "" {
					realDriftLines = append(realDriftLines, "  "+currentResourceName+": "+strings.TrimSpace(line))
				} else {
					realDriftLines = append(realDriftLines, "  "+strings.TrimSpace(line))
				}
			}
		}

		// Check for additions
		if additionPattern.MatchString(line) {
			// Check if this addition is exempted
			isExempted, matchedExemptionName := checkExemption(line)

			if isExempted {
				// Track this exemption
				triggeredExemptions[matchedExemptionName]++
				continue
			}

			hasAdditions = true
			// Store addition as drift
			if currentResourceName != "" {
				realDriftLines = append(realDriftLines, "  "+currentResourceName+": "+strings.TrimSpace(line))
			} else {
				realDriftLines = append(realDriftLines, "  "+strings.TrimSpace(line))
			}
		}

		// Check for deletions
		if deletionPattern.MatchString(line) {
			// Check if this deletion is exempted
			isExempted, matchedExemptionName := checkExemption(line)

			if isExempted {
				// Track this exemption
				triggeredExemptions[matchedExemptionName]++
				continue
			}

			hasDeletions = true
			// Store deletion as drift
			if currentResourceName != "" {
				realDriftLines = append(realDriftLines, "  "+currentResourceName+": "+strings.TrimSpace(line))
			} else {
				realDriftLines = append(realDriftLines, "  "+strings.TrimSpace(line))
			}
		}
	}

	// Return true (only computed) if no real changes, additions, or deletions
	onlyComputed := !hasRealChanges && !hasAdditions && !hasDeletions
	return onlyComputed, triggeredExemptions, realDriftLines
}

// extractPlanChanges extracts and formats the changes section from terraform plan output
func extractPlanChanges(planOutput string) string {
	scanner := bufio.NewScanner(strings.NewReader(planOutput))
	var changes []string
	inChanges := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "Terraform will perform the following actions:") {
			inChanges = true
			continue
		}

		if strings.HasPrefix(line, "Plan:") {
			inChanges = false
			break
		}

		if inChanges && strings.HasPrefix(line, "  # ") {
			// Start of a resource block
			changes = append(changes, formatResourceChange(line))

			// Read the rest of the resource block
			for scanner.Scan() {
				line = scanner.Text()
				if !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "      ") && !strings.HasPrefix(line, "        ") {
					break
				}
				changes = append(changes, formatAttributeChange(line))
			}

			changes = append(changes, "")
		}
	}

	return strings.Join(changes, "\n")
}

// formatResourceChange formats a resource header line with color
func formatResourceChange(line string) string {
	return colorYellow + line + colorReset
}

// formatAttributeChange formats an attribute change line with color
func formatAttributeChange(line string) string {
	if strings.Contains(line, "~ ") {
		return colorYellow + line + colorReset
	} else if strings.Contains(line, "+ ") {
		return colorGreen + line + colorReset
	} else if strings.Contains(line, "- ") {
		return colorRed + line + colorReset
	} else if strings.Contains(line, "-/+ ") {
		return "\033[0;35m" + line + colorReset // Magenta
	}
	return line
}

// extractPlanSummary extracts the "Plan: X to add, Y to change, Z to destroy" line
func extractPlanSummary(planOutput string) string {
	scanner := bufio.NewScanner(strings.NewReader(planOutput))

	for scanner.Scan() {
		line := scanner.Text()
		if planPattern.MatchString(line) {
			return line
		}
	}

	return ""
}

// extractAffectedResources extracts unique module names from plan output
// Returns a sorted list of affected module names (e.g., "healthcheck", "zero_trust_gateway_policy")
func extractAffectedResources(planOutput string) []string {
	scanner := bufio.NewScanner(strings.NewReader(planOutput))
	resourcesMap := make(map[string]bool)

	for scanner.Scan() {
		line := scanner.Text()

		// Look for resource headers like: # module.healthcheck.cloudflare_healthcheck.counted[0] will be updated
		if matches := resourceHeaderPattern.FindStringSubmatch(line); len(matches) > 1 {
			resourceName := matches[1]

			// Extract module name from patterns like:
			// - module.healthcheck.cloudflare_healthcheck.counted[0]
			// - module.zero_trust_gateway_policy.cloudflare_zero_trust_gateway_policy.complex
			if strings.HasPrefix(resourceName, "module.") {
				parts := strings.Split(resourceName, ".")
				if len(parts) >= 2 {
					moduleName := parts[1]
					resourcesMap[moduleName] = true
				}
			}
		}
	}

	// Convert map to sorted slice
	resources := make([]string, 0, len(resourcesMap))
	for resource := range resourcesMap {
		resources = append(resources, resource)
	}

	// Simple alphabetical sort
	for i := 0; i < len(resources); i++ {
		for j := i + 1; j < len(resources); j++ {
			if resources[i] > resources[j] {
				resources[i], resources[j] = resources[j], resources[i]
			}
		}
	}

	return resources
}
