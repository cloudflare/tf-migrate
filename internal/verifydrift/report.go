package verifydrift

import (
	"fmt"
	"strings"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[1;33m"
	colorCyan   = "\033[0;36m"
	colorBold   = "\033[1m"
)

// PrintReport writes a human-readable drift verification report to stdout.
//
// Output format:
//
//	Cloudflare Terraform Migration - Drift Verification
//	====================================================
//	Plan file: ./plan.txt
//	Resources detected: zone_setting, dns_record
//
//	✓ Exempted Changes  (N rules matched, M changes)
//	─────────────────────────────────────────────────
//	  Rule:    zone_setting_migration_drift
//	  Reason:  "Allow zone_setting resources to be created after migration"
//	  Changes:
//	    + cloudflare_zone_setting.minimal_brotli will be created
//	    - cloudflare_zone_setting.minimal will be destroyed
//
//	✗ Unexpected Drift  (K changes require attention)
//	─────────────────────────────────────────────────
//	  module.zone_setting.cloudflare_zone_setting.with_functions:
//	    ~ value = "aggressive" -> null
//
//	====================================================
//	Result: MIGRATION NEEDS ATTENTION
func PrintReport(result VerifyResult, planFile string) {
	divider := strings.Repeat("=", 52)
	sectionLine := strings.Repeat("─", 52)

	// ── Header ──────────────────────────────────────────
	fmt.Println()
	printBold("Cloudflare Terraform Migration - Drift Verification")
	fmt.Println(divider)
	fmt.Printf("Plan file:          %s\n", planFile)
	if len(result.DetectedResources) > 0 {
		fmt.Printf("Resources detected: %s\n", strings.Join(result.DetectedResources, ", "))
	} else {
		fmt.Printf("Resources detected: (none — plan may have no changes)\n")
	}
	fmt.Println()

	// ── Exempted section ─────────────────────────────────
	totalExempted := 0
	for _, g := range result.ExemptedGroups {
		totalExempted += len(g.Lines)
	}

	if len(result.ExemptedGroups) > 0 {
		printGreen("✓ Exempted Changes  (%d rule(s) matched, %d change(s))",
			len(result.ExemptedGroups), totalExempted)
		fmt.Println(sectionLine)

		for _, group := range result.ExemptedGroups {
			fmt.Printf("  Rule:    %s%s%s\n", colorCyan, group.RuleName, colorReset)
			if group.Description != "" {
				fmt.Printf("  Reason:  %s\n", group.Description)
			}
			fmt.Printf("  Changes:\n")
			for _, line := range group.Lines {
				fmt.Printf("    %s\n", colorizeDiffLine(line))
			}
			fmt.Println()
		}
	} else {
		printGreen("✓ No exempted changes")
		fmt.Println(sectionLine)
		fmt.Println()
	}

	// ── Unexpected drift section ──────────────────────────
	if result.HasUnexpected {
		printRed("✗ Unexpected Drift  (%d change(s) require attention)", len(result.UnexpectedDrift))
		fmt.Println(sectionLine)
		printGroupedDrift(result.UnexpectedDrift)
		fmt.Println()
	} else {
		printGreen("✓ No unexpected drift")
		fmt.Println(sectionLine)
		fmt.Println()
	}

	// ── Summary ───────────────────────────────────────────
	fmt.Println(divider)
	if result.HasUnexpected {
		printRed("Result: MIGRATION NEEDS ATTENTION")
		if len(result.ExemptedGroups) > 0 {
			fmt.Printf("  %d exemption rule(s) applied (%d expected change(s))\n",
				len(result.ExemptedGroups), totalExempted)
		}
		fmt.Printf("  %d unexpected change(s) require review\n", len(result.UnexpectedDrift))
	} else {
		printGreen("Result: ✓ MIGRATION LOOKS GOOD")
		if len(result.ExemptedGroups) > 0 {
			fmt.Printf("  %d exemption rule(s) applied (%d expected change(s))\n",
				len(result.ExemptedGroups), totalExempted)
		}
		fmt.Println("  No unexpected drift detected")
	}
	fmt.Println()
}

// printGroupedDrift prints drift lines with resource-name grouping.
// Lines prefixed with "  resource.name: change" are grouped under the resource name.
func printGroupedDrift(lines []string) {
	currentResource := ""
	for _, line := range lines {
		// Lines look like "  module.foo.cloudflare_bar.baz: ~ attr = old -> new"
		// or plain "  ~ attr = old -> new"
		colonIdx := strings.Index(line, ": ")
		if colonIdx > 0 {
			resource := strings.TrimSpace(line[:colonIdx])
			change := line[colonIdx+2:]
			if resource != currentResource {
				currentResource = resource
				printYellow("  %s:", resource)
			}
			fmt.Printf("    %s\n", colorizeDiffLine(change))
		} else {
			fmt.Printf("  %s\n", colorizeDiffLine(strings.TrimSpace(line)))
		}
	}
}

// colorizeDiffLine applies ANSI colour to a single plan diff line based on its
// leading operator: green for additions (+), red for deletions (-), yellow for
// modifications (~).
func colorizeDiffLine(line string) string {
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "+"):
		return colorGreen + line + colorReset
	case strings.HasPrefix(trimmed, "-"):
		return colorRed + line + colorReset
	case strings.HasPrefix(trimmed, "~"):
		return colorYellow + line + colorReset
	default:
		return line
	}
}

func printBold(format string, args ...interface{}) {
	fmt.Printf(colorBold+format+colorReset+"\n", args...)
}

func printGreen(format string, args ...interface{}) {
	fmt.Printf(colorGreen+format+colorReset+"\n", args...)
}

func printRed(format string, args ...interface{}) {
	fmt.Printf(colorRed+format+colorReset+"\n", args...)
}

func printYellow(format string, args ...interface{}) {
	fmt.Printf(colorYellow+format+colorReset+"\n", args...)
}
