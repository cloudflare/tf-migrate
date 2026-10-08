// version_suffix.go provides per-target-version isolation for the v4→v5
// e2e-runner track (init/migrate/run).
//
// Without this, every invocation of `e2e-runner run` shares the same local
// directories (e2e/tf/v4/, e2e/migrated-v4_to_v5/) and the same R2 remote
// state key (v4/terraform.tfstate) — the same key the real e2e-tests.yml CI
// job uses on every push to main. Running a manual/matrix test against a
// specific --target-provider-version without isolation risks clobbering that
// shared state.
//
// This mirrors the isolation pattern already used by the v5-upgrade track
// (see v5_upgrade_version.go's GenerateStateKey), adapted for the v4→v5
// track's fixed directory layout.
package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// sanitizeVersionSuffix normalizes a version string for safe use in file
// paths and R2 state keys, e.g. "v5.19.0" -> "5.19.0". Any character outside
// [A-Za-z0-9.-] is replaced with "-" so arbitrary/untrusted input can't
// escape the intended directory or produce an invalid state key.
func sanitizeVersionSuffix(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return ""
	}

	var b strings.Builder
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}

// versionedV4Dir returns the local v4 working directory for the given
// (unsanitized) version suffix. An empty suffix preserves the original,
// unversioned path (e2e/tf/v4/) for backward compatibility with existing
// tooling and CI that don't pass a suffix.
func versionedV4Dir(e2eRoot, rawSuffix string) string {
	suffix := sanitizeVersionSuffix(rawSuffix)
	if suffix == "" {
		return filepath.Join(e2eRoot, "tf", "v4")
	}
	return filepath.Join(e2eRoot, "tf", "v4-"+suffix)
}

// versionedV5Dir returns the local migrated-v5 working directory for the
// given (unsanitized) version suffix. An empty suffix preserves the
// original, unversioned path (e2e/migrated-v4_to_v5/).
func versionedV5Dir(e2eRoot, rawSuffix string) string {
	suffix := sanitizeVersionSuffix(rawSuffix)
	if suffix == "" {
		return filepath.Join(e2eRoot, "migrated-v4_to_v5")
	}
	return filepath.Join(e2eRoot, "migrated-v4_to_v5-"+suffix)
}

// versionedStateKey returns the R2 backend state key for the given
// (unsanitized) version suffix. An empty suffix preserves the original key
// (v4/terraform.tfstate) that the shared CI job uses. A non-empty suffix
// namespaces the key under v4/versions/<suffix>/ so it can never collide
// with that shared key.
func versionedStateKey(rawSuffix string) string {
	suffix := sanitizeVersionSuffix(rawSuffix)
	if suffix == "" {
		return "v4/terraform.tfstate"
	}
	return fmt.Sprintf("v4/versions/%s/terraform.tfstate", suffix)
}

// versionedV5StateKey returns the R2 backend state key for the v5-side
// (post-migration) state, keyed by the v5 "target" suffix — which is
// independent from the v4 base suffix used by versionedV4Dir/versionedStateKey
// (see RunConfig.V5TargetSuffix). This is what makes each target version's
// v5 state an independently persistent, re-runnable lineage sharing one
// common v4 resource base, instead of all versions sharing one v5 state that
// only ever represents whichever version was tested most recently.
//
// Uses a distinct "v5-direct/" prefix (vs. v4's "v4/versions/...") so it can
// never collide with the unrelated, already-existing v5-upgrade track's own
// state keys (see v5_upgrade_version.go's GenerateStateKey, which uses
// "v5/<from>-<to>-terraform.tfstate").
func versionedV5StateKey(rawSuffix string) string {
	suffix := sanitizeVersionSuffix(rawSuffix)
	if suffix == "" {
		return "v5-direct/terraform.tfstate"
	}
	return fmt.Sprintf("v5-direct/versions/%s/terraform.tfstate", suffix)
}

// parseInstalledProviderVersion reads the cloudflare/cloudflare provider
// version that `terraform init` actually resolved into .terraform.lock.hcl
// in dir. This is the ground truth for what got installed — as opposed to
// what required_providers *asked* for — so it can catch cases where a
// version constraint didn't parse the way we expected and Terraform silently
// resolved a different release than the one under test.
//
// Mirrors cmd/tf-migrate/version_check.go's parseVersionFromLockFile (kept
// as a small, self-contained copy here since that file lives in package
// main and isn't importable).
func parseInstalledProviderVersion(dir string) (string, error) {
	lockFilePath := filepath.Join(dir, ".terraform.lock.hcl")

	content, err := os.ReadFile(lockFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", lockFilePath, err)
	}

	parsed, diags := hclwrite.ParseConfig(content, ".terraform.lock.hcl", hcl.InitialPos)
	if diags.HasErrors() {
		return "", fmt.Errorf("failed to parse %s: %s", lockFilePath, diags.Error())
	}

	for _, block := range parsed.Body().Blocks() {
		if block.Type() != "provider" {
			continue
		}

		labels := block.Labels()
		if len(labels) == 0 {
			continue
		}

		providerLabel := labels[0]
		if providerLabel == "registry.terraform.io/cloudflare/cloudflare" || providerLabel == "cloudflare/cloudflare" {
			versionAttr := block.Body().GetAttribute("version")
			if versionAttr == nil {
				continue
			}
			versionStr := string(versionAttr.Expr().BuildTokens(nil).Bytes())
			versionStr = strings.Trim(strings.TrimSpace(versionStr), `"`)
			return versionStr, nil
		}
	}

	return "", fmt.Errorf("no cloudflare/cloudflare provider entry found in %s", lockFilePath)
}

// verifyRegistryProviderVersion confirms that the provider actually
// installed into dir by `terraform init` is an exact match for
// wantVersion, and that it was installed as a normal registry release
// rather than a local dev_overrides build (which bypasses version
// resolution/checksums entirely and wouldn't have a meaningful lock file
// entry to check). Call this only when no --provider path was given.
func verifyRegistryProviderVersion(dir, wantVersion string) error {
	if wantVersion == "" {
		return nil
	}

	got, err := parseInstalledProviderVersion(dir)
	if err != nil {
		return fmt.Errorf("could not verify installed provider version (expected registry install of exactly %s): %w", wantVersion, err)
	}

	wantNormalized := strings.TrimPrefix(strings.TrimSpace(wantVersion), "v")
	if got != wantNormalized {
		return fmt.Errorf("provider version mismatch: requested exactly %q via --target-provider-version, but terraform init resolved %q from the registry — refusing to continue with an unverified provider version", wantNormalized, got)
	}

	return nil
}

// defaultTestResourcePrefix is the fixed prefix testdata uses for real
// Cloudflare-side resource names (DNS names, list names, etc.), enforced
// repo-wide by `make lint-testdata` (scripts/lint_testdata_names.go). It is
// also what integration/v4_to_v5/testdata/*/expected/*.tf fixtures contain
// literally, and `make test-integration` compares migrated output against
// those fixtures with a plain string diff — so this exact value must remain
// the default whenever no version suffix is in play.
const defaultTestResourcePrefix = "cftftest"

// namespacedTestResourcePrefix computes the real-resource-name prefix to use
// for a given (unsanitized) version suffix. Empty suffix (or a suffix with
// no digits to derive a tag from) returns defaultTestResourcePrefix
// unchanged — so unversioned runs, and integration tests that string-compare
// against "cftftest"-prefixed fixtures, are completely unaffected. A
// non-empty suffix returns a namespaced prefix, e.g. "cftftest5190" for
// "5.19.0", so this leg's real Cloudflare-side resource names can't collide
// with another leg's, or with a shared/unversioned run's.
func namespacedTestResourcePrefix(rawSuffix string) string {
	tag := shortVersionTag(rawSuffix)
	if tag == "" {
		return defaultTestResourcePrefix
	}
	return defaultTestResourcePrefix + tag
}

// shortVersionTag derives a compact, digits-only tag from a version string,
// e.g. "5.19.0" -> "5190". Kept short (vs. sanitizeVersionSuffix's longer,
// dash-separated form used for file paths/state keys) since some Cloudflare
// name fields have tight length limits.
func shortVersionTag(rawSuffix string) string {
	suffix := sanitizeVersionSuffix(rawSuffix)
	var b strings.Builder
	for _, r := range suffix {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return ""
	}
	return b.String()
}

// namespaceTestResourceNames rewrites every occurrence of
// defaultTestResourcePrefix ("cftftest") to namespacedTestResourcePrefix(rawSuffix)
// across every .tf file under dir. When rawSuffix is empty (or has no usable
// digits), namespacedTestResourcePrefix returns defaultTestResourcePrefix
// unchanged, so this is a guaranteed no-op for unversioned runs — files are
// not even opened for writing, which matters because
// integration/v4_to_v5/testdata/*/expected/*.tf fixtures are compared with a
// plain string diff and must never be touched by this.
//
// This is the piece versionedV4Dir/versionedV5Dir/versionedStateKey don't
// cover: those isolate local Terraform *state*, but the Cloudflare API
// enforces name uniqueness independently of Terraform state. Without this, a
// version-isolated leg still starts from an empty state and tries to
// *create* resources under names that may already exist in the account from
// an earlier (unversioned or different-version) run — the "already exists"
// class of error this fixes.
//
// Must run on a copy of testdata (never the canonical
// integration/v4_to_v5/testdata/ source) — call it against a
// versionedV4Dir(..., suffix) directory after RunInit has synced files into
// it, before any terraform apply.
func namespaceTestResourceNames(dir, rawSuffix string) (int, error) {
	namespacedPrefix := namespacedTestResourcePrefix(rawSuffix)
	if namespacedPrefix == defaultTestResourcePrefix {
		return 0, nil
	}

	filesChanged := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".tf") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		if !strings.Contains(string(content), defaultTestResourcePrefix) {
			return nil
		}

		updated := strings.ReplaceAll(string(content), defaultTestResourcePrefix, namespacedPrefix)
		if err := os.WriteFile(path, []byte(updated), permFile); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
		filesChanged++
		return nil
	})
	if err != nil {
		return filesChanged, err
	}
	return filesChanged, nil
}
