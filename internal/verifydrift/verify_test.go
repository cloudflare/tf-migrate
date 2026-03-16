package verifydrift

import (
	"strings"
	"testing"
)

// Minimal plan snippet that looks like "No changes" — nothing to drift-check.
const planNoChanges = `
Terraform will perform the following actions:

No changes. Infrastructure is up-to-date.
`

// A plan containing an exempted pattern: "(known after apply)" is matched by
// the "computed_value_refreshes" global rule.
// Resource is under a module so DetectResourcesFromPlan can extract it.
const planComputedOnly = `
Terraform used the selected providers to generate the following execution plan.

  # module.dns_record.cloudflare_dns_record.example will be updated in-place
  ~ resource "cloudflare_dns_record" "example" {
      ~ ttl = (known after apply)
    }

Plan: 0 to add, 1 to change, 0 to destroy.
`

// A plan containing real (non-exempted) drift.
const planRealDrift = `
Terraform used the selected providers to generate the following execution plan.

  # module.dns_record.cloudflare_dns_record.example will be updated in-place
  ~ resource "cloudflare_dns_record" "example" {
      ~ value = "old-value" -> "new-value"
    }

Plan: 0 to add, 1 to change, 0 to destroy.
`

func TestVerify_NoChanges(t *testing.T) {
	result, err := Verify(planNoChanges)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if result.HasUnexpected {
		t.Errorf("expected HasUnexpected=false for no-changes plan, got true; UnexpectedDrift=%v", result.UnexpectedDrift)
	}
	if len(result.UnexpectedDrift) != 0 {
		t.Errorf("expected empty UnexpectedDrift, got %v", result.UnexpectedDrift)
	}
}

func TestVerify_ComputedOnlyIsExempted(t *testing.T) {
	result, err := Verify(planComputedOnly)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if result.HasUnexpected {
		t.Errorf("expected HasUnexpected=false for computed-only plan, got true; UnexpectedDrift=%v", result.UnexpectedDrift)
	}
}

func TestVerify_RealDriftIsUnexpected(t *testing.T) {
	result, err := Verify(planRealDrift)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if !result.HasUnexpected {
		t.Errorf("expected HasUnexpected=true for real-drift plan, got false")
	}
	if len(result.UnexpectedDrift) == 0 {
		t.Errorf("expected non-empty UnexpectedDrift, got empty slice")
	}
}

func TestVerify_DetectsResources(t *testing.T) {
	result, err := Verify(planComputedOnly)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	// DetectResourcesFromPlan extracts module names from "module.<name>.cloudflare_*" lines.
	found := false
	for _, r := range result.DetectedResources {
		if r == "dns_record" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected dns_record in DetectedResources, got %v", result.DetectedResources)
	}
}

// --- parseExemptionTag unit tests ---

func TestParseExemptionTag_WithTag(t *testing.T) {
	line := "  ~ ttl = (known after apply) [exempted: computed_value_refreshes]"
	name, clean := parseExemptionTag(line)
	if name != "computed_value_refreshes" {
		t.Errorf("expected name=%q, got %q", "computed_value_refreshes", name)
	}
	if strings.Contains(clean, "[exempted:") {
		t.Errorf("clean line should not contain tag, got %q", clean)
	}
}

func TestParseExemptionTag_NoTag(t *testing.T) {
	line := "  ~ value = old -> new"
	name, clean := parseExemptionTag(line)
	if name != "" {
		t.Errorf("expected empty name for untagged line, got %q", name)
	}
	if clean != line {
		t.Errorf("expected clean line to equal original, got %q", clean)
	}
}

func TestParseExemptionTag_MalformedTagNotClosed(t *testing.T) {
	line := "  ~ value [exempted: some_rule"
	name, clean := parseExemptionTag(line)
	if name != "" {
		t.Errorf("expected empty name for unclosed tag, got %q", name)
	}
	if clean != line {
		t.Errorf("expected clean line to equal original, got %q", clean)
	}
}

// --- groupExemptedLines unit tests ---

func TestGroupExemptedLines_GroupsByRule(t *testing.T) {
	lines := []string{
		"  ~ foo [exempted: rule_a]",
		"  ~ bar [exempted: rule_b]",
		"  ~ baz [exempted: rule_a]",
	}
	descByName := map[string]string{
		"rule_a": "desc a",
		"rule_b": "desc b",
	}
	groups := groupExemptedLines(lines, descByName)

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d: %+v", len(groups), groups)
	}

	// rule_a should come first (insertion order).
	if groups[0].RuleName != "rule_a" {
		t.Errorf("expected first group rule_a, got %q", groups[0].RuleName)
	}
	if groups[0].Description != "desc a" {
		t.Errorf("expected description %q, got %q", "desc a", groups[0].Description)
	}
	if len(groups[0].Lines) != 2 {
		t.Errorf("expected 2 lines in rule_a group, got %d", len(groups[0].Lines))
	}

	if groups[1].RuleName != "rule_b" {
		t.Errorf("expected second group rule_b, got %q", groups[1].RuleName)
	}
	if len(groups[1].Lines) != 1 {
		t.Errorf("expected 1 line in rule_b group, got %d", len(groups[1].Lines))
	}
}

func TestGroupExemptedLines_NoTagGoesToUnknown(t *testing.T) {
	lines := []string{
		"  ~ something without a tag",
	}
	groups := groupExemptedLines(lines, map[string]string{})
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].RuleName != "unknown" {
		t.Errorf("expected rule name 'unknown', got %q", groups[0].RuleName)
	}
}

func TestGroupExemptedLines_Empty(t *testing.T) {
	groups := groupExemptedLines(nil, map[string]string{})
	if len(groups) != 0 {
		t.Errorf("expected 0 groups for nil input, got %d", len(groups))
	}
}
