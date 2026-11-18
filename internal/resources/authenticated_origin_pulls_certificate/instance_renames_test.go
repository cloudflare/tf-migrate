package authenticated_origin_pulls_certificate

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

func transformCert(t *testing.T, m *V4ToV5Migrator, filePath, name, typeExpr string) hcl.Diagnostics {
	t.Helper()
	input := `resource "cloudflare_authenticated_origin_pulls_certificate" "` + name + `" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  certificate = "cert"
  private_key = "key"
  type        = ` + typeExpr + `
}`
	file, diags := hclwrite.ParseConfig([]byte(input), filepath.Base(filePath), hcl.InitialPos)
	require.False(t, diags.HasErrors(), "parse error: %v", diags)

	ctx := &transform.Context{
		Content:     []byte(input),
		Filename:    filepath.Base(filePath),
		FilePath:    filePath,
		Diagnostics: make(hcl.Diagnostics, 0),
		Metadata:    make(map[string]interface{}),
	}
	_, err := m.TransformConfig(ctx, file.Body().Blocks()[0])
	require.NoError(t, err)
	return ctx.Diagnostics
}

func newMigrator() *V4ToV5Migrator {
	return &V4ToV5Migrator{perHostname: make(map[perHostnameKey]string)}
}

func TestGetInstanceRenames_PerHostnameOnlyScopedByDir(t *testing.T) {
	m := newMigrator()
	_ = transformCert(t, m, "/cfg/b_certs.tf", "host", `"per-hostname"`)
	_ = transformCert(t, m, "/cfg/b_certs.tf", "zone", `"per-zone"`)
	_ = transformCert(t, m, "/cfg/modules/x/main.tf", "host", `"per-hostname"`)
	_ = transformCert(t, m, "/cfg/modules/y/main.tf", "host", `"per-zone"`)

	assert.Equal(t, []transform.InstanceRename{
		{OldType: perZoneType, NewType: perHostnameType, Name: "host", Dir: "/cfg", File: "/cfg/b_certs.tf"},
		{OldType: perZoneType, NewType: perHostnameType, Name: "host", Dir: "/cfg/modules/x", File: "/cfg/modules/x/main.tf"},
	}, m.GetInstanceRenames())
}

func TestNonLiteralTypeIsWarned(t *testing.T) {
	for _, expr := range []string{`var.cert_type`, `"per-${var.kind}"`, `local.per_hostname ? "per-hostname" : "per-zone"`} {
		t.Run(expr, func(t *testing.T) {
			m := newMigrator()
			diags := transformCert(t, m, "/cfg/main.tf", "dyn", expr)

			require.Len(t, diags, 1, "%v", diags)
			assert.Equal(t, hcl.DiagWarning, diags[0].Severity)
			assert.Contains(t, diags[0].Summary, "cloudflare_authenticated_origin_pulls_certificate.dyn")
			assert.Contains(t, diags[0].Detail, "type = "+expr)
			assert.Contains(t, diags[0].Detail, "to   = cloudflare_authenticated_origin_pulls_hostname_certificate.dyn")
			assert.Empty(t, m.GetInstanceRenames(), "non-literal type must not be treated as per-hostname")
		})
	}
}

func TestLiteralTypeIsNotWarned(t *testing.T) {
	for _, expr := range []string{`"per-hostname"`, `"per-zone"`} {
		diags := transformCert(t, newMigrator(), "/cfg/main.tf", "lit", expr)
		for _, d := range diags {
			assert.False(t, strings.Contains(d.Summary, "could not resolve type"), "%s: %s", expr, d.Summary)
		}
	}
}
