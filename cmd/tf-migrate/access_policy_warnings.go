package main

import (
	"fmt"
	"sort"

	"github.com/cloudflare/tf-migrate/internal/resources/zero_trust_access_policy"
	"github.com/hashicorp/hcl/v2"
)

func printAccessPolicyManualActions(affected []string) {
	sort.Strings(affected)
	fmt.Printf("\nACTION REQUIRED: Access policy migration is incomplete (%d policies)\n\n", len(affected))
	for _, address := range affected {
		fmt.Printf("  - %s\n", address)
	}
	fmt.Println("\nBefore terraform apply, add these policies inline to their parent application's")
	fmt.Println("policies attribute, preserving all conditions and precedence.")
	fmt.Println("tf-migrate generates state-removal blocks but does NOT attach these policies.")
	fmt.Println("Applying before completing this step can detach and delete existing app-scoped policies.")
	fmt.Println("The complete original policies are preserved as comments in the removed blocks.")
	fmt.Printf("See: %s\n", zero_trust_access_policy.ManualMigrationGuide)
}

// Print last, even on dry runs or when provider-version discovery fails, so the
// generic next steps cannot bury the outstanding manual policy migration.
func printAccessPolicyMigrationReminder(diags hcl.Diagnostics, cfg config) {
	if cfg.quiet {
		return
	}
	var affected []string
	for _, d := range diags {
		if d.Severity == hcl.DiagWarning && d.Summary == zero_trust_access_policy.ManualMigrationSummary {
			affected = append(affected, extractResourceName(d.Detail))
		}
	}
	if len(affected) > 0 {
		printAccessPolicyManualActions(affected)
	}
}
