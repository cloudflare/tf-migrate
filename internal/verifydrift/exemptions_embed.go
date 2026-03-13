// Package verifydrift implements the verify-drift command which analyses a
// terraform plan output file against Cloudflare's known migration drift
// exemptions and reports which changes are expected vs unexpected.
//
// Drift exemptions are fetched from GitHub (always latest) with a 1-hour TTL
// cache in ~/.cache/tf-migrate/exemptions/. If GitHub is unreachable the
// binary falls back to the exemption files embedded at build time.
//
// To refresh the embedded fallback after updating e2e/drift-exemptions/:
//
//	make sync-exemptions
package verifydrift

import "embed"

// embeddedExemptions contains the YAML exemption files bundled at build time.
// These are used as a fallback when GitHub is unreachable.
//
// Source of truth: e2e/global-drift-exemptions.yaml and e2e/drift-exemptions/
// Keep in sync with: make sync-exemptions
//
//go:embed exemptions/global-drift-exemptions.yaml exemptions/drift-exemptions/*.yaml
var embeddedExemptions embed.FS
