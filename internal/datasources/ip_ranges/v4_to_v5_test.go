package ip_ranges

import (
	"strings"
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/stretchr/testify/assert"
)

// The v4 datasource takes no input arguments, so the block body must survive
// the migration untouched. Everything this migrator does happens through
// cross-file reference rewriting, not block transformation.
func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.ConfigTestCase{
		{
			Name:     "empty body is left unchanged",
			Input:    `data "cloudflare_ip_ranges" "cloudflare" {}`,
			Expected: `data "cloudflare_ip_ranges" "cloudflare" {}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, testCases, migrator)
}

// Only the two attributes with an exact v5 equivalent are renamed. The
// china_* attributes are deliberately absent here: they are a 2 → 1 merge into
// jdcloud_cidrs and are reported as warnings instead.
func TestGetAttributeRenames(t *testing.T) {
	migrator := &V4ToV5Migrator{}
	renames := migrator.GetAttributeRenames()

	assert.Len(t, renames, 2)

	byOld := make(map[string]transform.AttributeRename, len(renames))
	for _, r := range renames {
		assert.Equal(t, "data.cloudflare_ip_ranges", r.ResourceType)
		byOld[r.OldAttribute] = r
	}

	assert.Equal(t, "ipv4_cidrs", byOld["ipv4_cidr_blocks"].NewAttribute)
	assert.Equal(t, "ipv6_cidrs", byOld["ipv6_cidr_blocks"].NewAttribute)

	// These have no drop-in replacement and must not be auto-renamed.
	assert.NotContains(t, byOld, "cidr_blocks")
	assert.NotContains(t, byOld, "china_ipv4_cidr_blocks")
	assert.NotContains(t, byOld, "china_ipv6_cidr_blocks")
}

func TestGetInvalidAttributeReferences(t *testing.T) {
	migrator := &V4ToV5Migrator{}
	refs := migrator.GetInvalidAttributeReferences()

	assert.Len(t, refs, 3)

	byAttr := make(map[string]transform.InvalidAttributeReference, len(refs))
	for _, r := range refs {
		assert.Equal(t, "data.cloudflare_ip_ranges", r.ResourceType)
		assert.NotEmpty(t, r.Suggestion)
		byAttr[r.Attribute] = r
	}

	// cidr_blocks was the combined v4 list; v5 splits it by family.
	assert.Contains(t, byAttr, "cidr_blocks")
	assert.Contains(t, byAttr["cidr_blocks"].Suggestion, "concat(")
	assert.Contains(t, byAttr["cidr_blocks"].Suggestion, "ipv4_cidrs")
	assert.Contains(t, byAttr["cidr_blocks"].Suggestion, "ipv6_cidrs")

	// Both china_* lists merged into jdcloud_cidrs, which requires opting in
	// via networks = "jdcloud". The guidance must say so, or the user gets a
	// null list with no explanation.
	for _, attr := range []string{"china_ipv4_cidr_blocks", "china_ipv6_cidr_blocks"} {
		assert.Contains(t, byAttr, attr)
		s := byAttr[attr].Suggestion
		assert.Contains(t, s, "jdcloud_cidrs")
		assert.Contains(t, s, `networks = "jdcloud"`)
		// The suggestion names the attribute it is about.
		assert.Contains(t, s, attr)
	}
}

// The rename and warning patterns are built as
// <ResourceType>\.([a-zA-Z0-9_-]+)\.<Attribute>, so a declared attribute that
// is a suffix of another attribute could match inside it. Guard against that:
// cidr_blocks is a suffix of ipv4_cidr_blocks and china_ipv4_cidr_blocks, and
// is only safe because a literal "." must precede it.
func TestDeclaredAttributesAreNotAmbiguous(t *testing.T) {
	migrator := &V4ToV5Migrator{}

	v4Attributes := []string{
		"china_ipv4_cidr_blocks",
		"china_ipv6_cidr_blocks",
		"cidr_blocks",
		"ipv4_cidr_blocks",
		"ipv6_cidr_blocks",
	}

	var declared []string
	for _, r := range migrator.GetAttributeRenames() {
		declared = append(declared, r.OldAttribute)
	}
	for _, r := range migrator.GetInvalidAttributeReferences() {
		declared = append(declared, r.Attribute)
	}

	// Every v4 computed attribute is accounted for exactly once, so no
	// reference is silently left behind.
	assert.ElementsMatch(t, v4Attributes, declared)

	// No declared attribute may be a prefix of another, which would let the
	// unbounded trailing match corrupt the longer name.
	for _, a := range declared {
		for _, b := range declared {
			if a == b {
				continue
			}
			assert.False(t, strings.HasPrefix(b, a),
				"%q is a prefix of %q: the rename pattern has no trailing boundary and would corrupt it", a, b)
		}
	}
}

func TestCanHandle(t *testing.T) {
	migrator := &V4ToV5Migrator{}

	assert.True(t, migrator.CanHandle("data.cloudflare_ip_ranges"))
	assert.False(t, migrator.CanHandle("cloudflare_ip_ranges"))
	assert.False(t, migrator.CanHandle("data.cloudflare_ip_range"))
	assert.False(t, migrator.CanHandle("data.cloudflare_zones"))
}

func TestGetResourceType(t *testing.T) {
	migrator := &V4ToV5Migrator{}
	assert.Equal(t, "cloudflare_ip_ranges", migrator.GetResourceType())
}

func TestGetResourceRename(t *testing.T) {
	migrator := &V4ToV5Migrator{}
	old, new := migrator.GetResourceRename()

	// The datasource keeps its name in v5.
	assert.Equal(t, []string{"data.cloudflare_ip_ranges"}, old)
	assert.Equal(t, "data.cloudflare_ip_ranges", new)
}

func TestImplementsInterfaces(t *testing.T) {
	migrator := &V4ToV5Migrator{}

	assert.Implements(t, (*transform.AttributeRenamer)(nil), migrator)
	assert.Implements(t, (*transform.InvalidAttributeReferenceDetector)(nil), migrator)
	assert.Implements(t, (*transform.ResourceTransformer)(nil), migrator)
}
