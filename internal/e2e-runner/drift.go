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
	Name                     string           `yaml:"name"`
	Description              string           `yaml:"description"`
	ResourceTypes            []string         `yaml:"resource_types,omitempty"`
	ResourceNamePatterns     []string         `yaml:"resource_name_patterns,omitempty"`
	Attributes               []string         `yaml:"attributes,omitempty"`
	Patterns                 []string         `yaml:"patterns,omitempty"`
	AllowResourceCreation    bool             `yaml:"allow_resource_creation,omitempty"`
	AllowResourceDestruction bool             `yaml:"allow_resource_destruction,omitempty"`
	AllowResourceReplacement bool             `yaml:"allow_resource_replacement,omitempty"`
	Enabled                  bool             `yaml:"enabled"`
	compiledPatterns         []*regexp.Regexp // Pre-compiled patterns for performance
	compiledNamePatterns     []*regexp.Regexp // Pre-compiled resource name patterns
	source                   string           // Which config file this came from
}

// ExemptionSettings holds global drift exemption settings
type ExemptionSettings struct {
	ApplyExemptions        bool `yaml:"apply_exemptions"`
	VerboseExemptions      bool `yaml:"verbose_exemptions"`
	WarnUnusedExemptions   bool `yaml:"warn_unused_exemptions"`
	LoadResourceExemptions bool `yaml:"load_resource_exemptions"`
}

// DriftExemptionsConfig represents the drift exemptions configuration file
type DriftExemptionsConfig struct {
	Version    int                `yaml:"version"`
	Exemptions []DriftExemption   `yaml:"exemptions"`
	Settings   ExemptionSettings  `yaml:"settings"`
	Source     string             // "global", "legacy", or "resource:{name}"
}

// loadDriftExemptions loads exemptions from multiple sources and merges them
func loadDriftExemptions(resourceFilter []string) (*DriftExemptionsConfig, error) {
	repoRoot := getRepoRoot()

	// 1. Load global exemptions (e2e/global-drift-exemptions.yaml)
	globalConfig, err := loadGlobalExemptions(repoRoot)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load global exemptions: %w", err)
	}

	// 2. Load resource-specific exemptions (e2e/drift-exemptions/{resource}.yaml)
	resourceConfigs := make(map[string]*DriftExemptionsConfig)
	if globalConfig != nil && globalConfig.Settings.LoadResourceExemptions {
		for _, resource := range resourceFilter {
			resourceConfig, err := loadResourceExemptions(repoRoot, resource)
			if err != nil && !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to load exemptions for %s: %w", resource, err)
			}
			if resourceConfig != nil {
				resourceConfigs[resource] = resourceConfig
			}
		}
	}

	// 3. Merge exemptions (later overrides earlier)
	merged := mergeExemptions(globalConfig, resourceConfigs)

	// 4. Pre-compile regex patterns
	if err := compilePatterns(merged); err != nil {
		return nil, fmt.Errorf("failed to compile patterns: %w", err)
	}

	return merged, nil
}

// loadGlobalExemptions loads global exemptions config
func loadGlobalExemptions(repoRoot string) (*DriftExemptionsConfig, error) {
	configPath := filepath.Join(repoRoot, "e2e", "global-drift-exemptions.yaml")

	// If file doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &DriftExemptionsConfig{
			Version:    1,
			Exemptions: []DriftExemption{},
			Settings: ExemptionSettings{
				ApplyExemptions:        false,
				VerboseExemptions:      false,
				WarnUnusedExemptions:   false,
				LoadResourceExemptions: true,
			},
			Source: "default",
		}, nil
	}

	config, err := loadConfigFromFile(configPath, "global")
	if err != nil {
		return nil, err
	}

	// Set defaults for any unset settings
	if config.Settings.LoadResourceExemptions == false && config.Version == 0 {
		// If LoadResourceExemptions wasn't explicitly set and version is 0, default to true
		config.Settings.LoadResourceExemptions = true
	}

	return config, nil
}

// loadResourceExemptions loads resource-specific exemptions
func loadResourceExemptions(repoRoot string, resource string) (*DriftExemptionsConfig, error) {
	// Load from e2e/drift-exemptions/{resource}.yaml
	configPath := filepath.Join(repoRoot, "e2e", "drift-exemptions", resource+".yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil
	}

	config, err := loadConfigFromFile(configPath, fmt.Sprintf("resource:%s", resource))
	if err != nil {
		return nil, err
	}

	// Validate resource_type if specified
	if config.Version >= 1 {
		expectedType := "cloudflare_" + resource
		// Check if any exemption has a different resource type restriction
		for _, e := range config.Exemptions {
			if len(e.ResourceTypes) > 0 {
				found := false
				for _, rt := range e.ResourceTypes {
					if rt == expectedType {
						found = true
						break
					}
				}
				if !found {
					printYellow("⚠ Warning: Resource exemption in %s specifies resource_types that don't match expected type %s", configPath, expectedType)
				}
			}
		}
	}

	return config, nil
}

// loadConfigFromFile loads a drift exemptions config from a file
func loadConfigFromFile(path string, source string) (*DriftExemptionsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config DriftExemptionsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	config.Source = source

	// Mark all exemptions with their source
	for i := range config.Exemptions {
		config.Exemptions[i].source = source
	}

	return &config, nil
}

// mergeExemptions combines exemptions from multiple sources
func mergeExemptions(
	global *DriftExemptionsConfig,
	resourceSpecific map[string]*DriftExemptionsConfig,
) *DriftExemptionsConfig {
	merged := &DriftExemptionsConfig{
		Version:    1,
		Exemptions: []DriftExemption{},
		Settings:   global.Settings,
		Source:     "merged",
	}

	exemptionMap := make(map[string]DriftExemption) // name -> exemption

	// 1. Add global exemptions
	for _, e := range global.Exemptions {
		exemptionMap[e.Name] = e
	}

	// 2. Merge resource-specific exemptions (override global)
	for _, config := range resourceSpecific {
		for _, e := range config.Exemptions {
			exemptionMap[e.Name] = e // Override or add
		}
	}

	// Convert map back to slice
	for _, e := range exemptionMap {
		merged.Exemptions = append(merged.Exemptions, e)
	}

	return merged
}

// compilePatterns pre-compiles all regex patterns for performance
func compilePatterns(config *DriftExemptionsConfig) error {
	for i := range config.Exemptions {
		exemption := &config.Exemptions[i]

		// Compile drift patterns
		exemption.compiledPatterns = make([]*regexp.Regexp, 0, len(exemption.Patterns))
		for _, pattern := range exemption.Patterns {
			compiled, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("invalid regex pattern '%s' in exemption '%s' (from %s): %w",
					pattern, exemption.Name, exemption.source, err)
			}
			exemption.compiledPatterns = append(exemption.compiledPatterns, compiled)
		}

		// Compile resource name patterns
		exemption.compiledNamePatterns = make([]*regexp.Regexp, 0, len(exemption.ResourceNamePatterns))
		for _, pattern := range exemption.ResourceNamePatterns {
			compiled, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("invalid resource name pattern '%s' in exemption '%s' (from %s): %w",
					pattern, exemption.Name, exemption.source, err)
			}
			exemption.compiledNamePatterns = append(exemption.compiledNamePatterns, compiled)
		}
	}
	return nil
}

// DriftCheckResult holds the result of drift detection
type DriftCheckResult struct {
	OnlyComputedChanges   bool
	TriggeredExemptions   map[string]int // exemption name -> count of matches
	ExemptionsEnabled     bool
	RealDriftLines        []string       // actual drift detected (non-exempted changes)
	ExemptedDriftLines    []string       // exempted changes (for display purposes)
}

// hasOnlyComputedChanges checks if a terraform plan only has "known after apply" changes
// Returns true if only computed refreshes or acceptable changes, false if real drift
func hasOnlyComputedChanges(planOutput string, resourceFilter []string) bool {
	result := checkDrift(planOutput, resourceFilter)
	return result.OnlyComputedChanges
}

// checkDrift performs drift detection and returns detailed results
func checkDrift(planOutput string, resourceFilter []string) DriftCheckResult {
	// Load exemptions from config
	config, err := loadDriftExemptions(resourceFilter)
	if err != nil {
		// If we can't load config, fall back to default behavior
		return DriftCheckResult{
			OnlyComputedChanges: hasOnlyComputedChangesDefault(planOutput),
			TriggeredExemptions: make(map[string]int),
			ExemptionsEnabled:   false,
			RealDriftLines:      []string{},
			ExemptedDriftLines:  []string{},
		}
	}

	// If exemptions are disabled, use default behavior
	if !config.Settings.ApplyExemptions {
		return DriftCheckResult{
			OnlyComputedChanges: hasOnlyComputedChangesDefault(planOutput),
			TriggeredExemptions: make(map[string]int),
			ExemptionsEnabled:   false,
			RealDriftLines:      []string{},
			ExemptedDriftLines:  []string{},
		}
	}

	onlyComputed, triggeredExemptions, realDriftLines, exemptedDriftLines := hasOnlyComputedChangesWithExemptions(planOutput, config)
	return DriftCheckResult{
		OnlyComputedChanges: onlyComputed,
		TriggeredExemptions: triggeredExemptions,
		ExemptionsEnabled:   true,
		RealDriftLines:      realDriftLines,
		ExemptedDriftLines:  exemptedDriftLines,
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
// Returns whether only computed changes exist, a map of triggered exemptions, real drift lines, and exempted drift lines
func hasOnlyComputedChangesWithExemptions(planOutput string, config *DriftExemptionsConfig) (bool, map[string]int, []string, []string) {
	scanner := bufio.NewScanner(strings.NewReader(planOutput))

	// Patterns to detect
	hasRealChanges := false
	hasAdditions := false
	hasDeletions := false
	currentResourceType := ""
	currentResourceName := ""
	currentChangeType := ""
	skipCurrentResource := false
	triggeredExemptions := make(map[string]int)
	exemptionUsageCounts := make(map[string]int) // Track which exemptions were used
	realDriftLines := []string{}
	exemptedDriftLines := []string{}

	// Helper function to check if exemption applies to current resource scope
	matchesResourceScope := func(exemption *DriftExemption) bool {
		// Check resource type filter
		if len(exemption.ResourceTypes) > 0 {
			matches := false
			for _, rt := range exemption.ResourceTypes {
				if rt == currentResourceType {
					matches = true
					break
				}
			}
			if !matches {
				return false
			}
		}

		// Check resource name pattern filter
		if len(exemption.ResourceNamePatterns) > 0 {
			matches := false
			for _, pattern := range exemption.compiledNamePatterns {
				if pattern.MatchString(currentResourceName) {
					matches = true
					break
				}
			}
			if !matches {
				return false
			}
		}

		return true
	}

	// Helper function to check if a line/resource is exempted
	checkExemption := func(line string, changeType string) (bool, string) {
		for i := range config.Exemptions {
			exemption := &config.Exemptions[i]
			if !exemption.Enabled {
				continue
			}

			// Check resource scope
			if !matchesResourceScope(exemption) {
				continue
			}

			// Check simplified patterns first
			if exemption.AllowResourceCreation && changeType == "creation" {
				exemptionUsageCounts[exemption.Name]++
				if config.Settings.VerboseExemptions {
					printYellow("  [Exempted:%s (from %s)] Resource creation allowed", exemption.Name, exemption.source)
				}
				return true, exemption.Name
			}

			if exemption.AllowResourceDestruction && changeType == "destruction" {
				exemptionUsageCounts[exemption.Name]++
				if config.Settings.VerboseExemptions {
					printYellow("  [Exempted:%s (from %s)] Resource destruction allowed", exemption.Name, exemption.source)
				}
				return true, exemption.Name
			}

			if exemption.AllowResourceReplacement && changeType == "replacement" {
				exemptionUsageCounts[exemption.Name]++
				if config.Settings.VerboseExemptions {
					printYellow("  [Exempted:%s (from %s)] Resource replacement allowed", exemption.Name, exemption.source)
				}
				return true, exemption.Name
			}

			// Check line patterns (using pre-compiled patterns)
			for _, compiledPattern := range exemption.compiledPatterns {
				if compiledPattern.MatchString(line) {
					exemptionUsageCounts[exemption.Name]++
					if config.Settings.VerboseExemptions {
						printYellow("  [Exempted:%s (from %s)] %s", exemption.Name, exemption.source, line)
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
			skipCurrentResource = false // Reset skip flag for new resource

			// Extract resource type (e.g., cloudflare_zone from module.foo.cloudflare_zone.example)
			parts := strings.Split(matches[1], ".")
			if len(parts) >= 2 {
				// Safe: len(parts) >= 2 means parts[len(parts)-2] is at index 0 or higher
				currentResourceType = parts[len(parts)-2]
			} else {
				// Fallback: if resource name doesn't have expected format, clear type
				currentResourceType = ""
			}

			// Detect change type from header line
			if strings.Contains(line, "will be created") {
				currentChangeType = "creation"
			} else if strings.Contains(line, "will be destroyed") {
				currentChangeType = "destruction"
			} else if strings.Contains(line, "must be replaced") {
				currentChangeType = "replacement"
			} else if strings.Contains(line, "will be updated") {
				currentChangeType = "update"
			} else {
				currentChangeType = "unknown"
			}

			// Check if entire resource change is exempted
			isExempted, matchedName := checkExemption(line, currentChangeType)
			if isExempted {
				triggeredExemptions[matchedName]++
				skipCurrentResource = true
				// Store exempted resource for display
				exemptedDriftLines = append(exemptedDriftLines, "  "+currentResourceName+": "+line+" [exempted: "+matchedName+"]")
				continue
			}
		}

		// Skip lines if entire resource is exempted
		if skipCurrentResource {
			continue
		}

		// Skip resource/data block declarations (like "+ resource" or "~ data")
		if resourceDeclarationPattern.MatchString(line) {
			continue
		}

		// Check for update lines
		if updateLinePattern.MatchString(line) {
			// Check if this line matches any exemption pattern
			isExempted, matchedExemptionName := checkExemption(line, "update")

			if isExempted {
				// Track this exemption
				triggeredExemptions[matchedExemptionName]++
				// Store exempted line for display
				if arrowPattern.MatchString(line) {
					if currentResourceName != "" {
						exemptedDriftLines = append(exemptedDriftLines, "  "+currentResourceName+": "+strings.TrimSpace(line)+" [exempted: "+matchedExemptionName+"]")
					} else {
						exemptedDriftLines = append(exemptedDriftLines, "  "+strings.TrimSpace(line)+" [exempted: "+matchedExemptionName+"]")
					}
				}
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
			isExempted, matchedExemptionName := checkExemption(line, "addition")

			if isExempted {
				// Track this exemption
				triggeredExemptions[matchedExemptionName]++
				// Store exempted line for display
				if currentResourceName != "" {
					exemptedDriftLines = append(exemptedDriftLines, "  "+currentResourceName+": "+strings.TrimSpace(line)+" [exempted: "+matchedExemptionName+"]")
				} else {
					exemptedDriftLines = append(exemptedDriftLines, "  "+strings.TrimSpace(line)+" [exempted: "+matchedExemptionName+"]")
				}
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
			isExempted, matchedExemptionName := checkExemption(line, "deletion")

			if isExempted {
				// Track this exemption
				triggeredExemptions[matchedExemptionName]++
				// Store exempted line for display
				if currentResourceName != "" {
					exemptedDriftLines = append(exemptedDriftLines, "  "+currentResourceName+": "+strings.TrimSpace(line)+" [exempted: "+matchedExemptionName+"]")
				} else {
					exemptedDriftLines = append(exemptedDriftLines, "  "+strings.TrimSpace(line)+" [exempted: "+matchedExemptionName+"]")
				}
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

	// Warn about unused exemptions if configured
	if config.Settings.WarnUnusedExemptions {
		for i := range config.Exemptions {
			exemption := &config.Exemptions[i]
			if exemption.Enabled && exemptionUsageCounts[exemption.Name] == 0 {
				printYellow("⚠ Exemption '%s' (from %s) was not used - consider removing it", exemption.Name, exemption.source)
			}
		}
	}

	// Return true (only computed) if no real changes, additions, or deletions
	onlyComputed := !hasRealChanges && !hasAdditions && !hasDeletions
	return onlyComputed, triggeredExemptions, realDriftLines, exemptedDriftLines
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
