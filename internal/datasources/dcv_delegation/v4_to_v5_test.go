package dcv_delegation

import (
	"strings"
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/stretchr/testify/assert"
)

// zone_id is the only input and is unchanged in v5, so the block body must
// survive the migration untouched.
func TestConfigTransformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	testCases := []testhelpers.ConfigTestCase{
		{
			Name: "zone_id is left unchanged",
			Input: `data "cloudflare_dcv_delegation" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
}`,
			Expected: `data "cloudflare_dcv_delegation" "example" {
  zone_id = "0da42c8d2132a9ddaf714f9e7c920711"
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, testCases, migrator)
}

// v4 id held the delegation UUID; v5 calls it uuid and has no id.
func TestGetAttributeRenames(t *testing.T) {
	migrator := &V4ToV5Migrator{}
	renames := migrator.GetAttributeRenames()

	assert.Len(t, renames, 1)
	assert.Equal(t, "data.cloudflare_dcv_delegation", renames[0].ResourceType)
	assert.Equal(t, "id", renames[0].OldAttribute)
	assert.Equal(t, "uuid", renames[0].NewAttribute)
}

// hostname was removed in v5. The warning must give the expression that
// reproduces the v4 value, or the user has no way to rebuild their CNAME.
func TestGetInvalidAttributeReferences(t *testing.T) {
	migrator := &V4ToV5Migrator{}
	refs := migrator.GetInvalidAttributeReferences()

	assert.Len(t, refs, 1)
	assert.Equal(t, "data.cloudflare_dcv_delegation", refs[0].ResourceType)
	assert.Equal(t, "hostname", refs[0].Attribute)
	assert.Contains(t, refs[0].Suggestion, ".uuid}.dcv.cloudflare.com")
}

// The rename and warning patterns are built as
// <ResourceType>\.([a-zA-Z0-9_-]+)\.<Attribute> with no trailing boundary.
// Every v4 computed attribute must be declared once, and none may be a prefix
// of another v4 attribute that is still valid in v5 (zone_id).
func TestDeclaredAttributesAreNotAmbiguous(t *testing.T) {
	migrator := &V4ToV5Migrator{}

	var declared []string
	for _, r := range migrator.GetAttributeRenames() {
		declared = append(declared, r.OldAttribute)
	}
	for _, r := range migrator.GetInvalidAttributeReferences() {
		declared = append(declared, r.Attribute)
	}

	assert.ElementsMatch(t, []string{"id", "hostname"}, declared)

	for _, a := range declared {
		for _, b := range append([]string{"zone_id"}, declared...) {
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

	assert.True(t, migrator.CanHandle("data.cloudflare_dcv_delegation"))
	assert.False(t, migrator.CanHandle("cloudflare_dcv_delegation"))
	assert.False(t, migrator.CanHandle("data.cloudflare_zone"))
}

func TestGetResourceType(t *testing.T) {
	migrator := &V4ToV5Migrator{}
	assert.Equal(t, "cloudflare_dcv_delegation", migrator.GetResourceType())
}

func TestGetResourceRename(t *testing.T) {
	migrator := &V4ToV5Migrator{}
	old, new := migrator.GetResourceRename()

	assert.Equal(t, []string{"data.cloudflare_dcv_delegation"}, old)
	assert.Equal(t, "data.cloudflare_dcv_delegation", new)
}

func TestImplementsInterfaces(t *testing.T) {
	migrator := &V4ToV5Migrator{}

	assert.Implements(t, (*transform.AttributeRenamer)(nil), migrator)
	assert.Implements(t, (*transform.InvalidAttributeReferenceDetector)(nil), migrator)
	assert.Implements(t, (*transform.ResourceTransformer)(nil), migrator)
}
