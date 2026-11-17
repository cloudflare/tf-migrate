package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

const (
	aopCertV4   = "cloudflare_authenticated_origin_pulls_certificate"
	aopHostCert = "cloudflare_authenticated_origin_pulls_hostname_certificate"
)

func TestApplyInstanceRenames(t *testing.T) {
	renames := []transform.InstanceRename{
		{OldType: aopCertV4, NewType: aopHostCert, Name: "host", Dir: "/cfg"},
		{OldType: aopCertV4, NewType: aopHostCert, Name: "other_module", Dir: "/cfg/modules/x"},
		{OldType: aopCertV4, NewType: aopHostCert, Name: "anywhere"},
	}

	input := `output "a" { value = cloudflare_authenticated_origin_pulls_certificate.host.id }
output "b" { value = cloudflare_authenticated_origin_pulls_certificate.host["k"].id }
output "c" { value = cloudflare_authenticated_origin_pulls_certificate.host2.id }
output "d" { value = cloudflare_authenticated_origin_pulls_certificate.zone.id }
output "e" { value = cloudflare_authenticated_origin_pulls_certificate.other_module.id }
output "f" { value = cloudflare_authenticated_origin_pulls_certificate.anywhere.id }
output "g" { value = data.cloudflare_authenticated_origin_pulls_certificate.host.id }
locals { deps = [cloudflare_authenticated_origin_pulls_certificate.host] }

moved {
  from = cloudflare_authenticated_origin_pulls_certificate.host
  to   = cloudflare_authenticated_origin_pulls_hostname_certificate.host
}
`
	got := applyInstanceRenames(input, renames, "/cfg")

	for _, want := range []string{
		`value = cloudflare_authenticated_origin_pulls_hostname_certificate.host.id`,
		`value = cloudflare_authenticated_origin_pulls_hostname_certificate.host["k"].id`,
		`deps = [cloudflare_authenticated_origin_pulls_hostname_certificate.host]`,
		`value = cloudflare_authenticated_origin_pulls_hostname_certificate.anywhere.id`,
		// Not renamed: a different name, a per-zone certificate, a certificate
		// in another module, a data source, and the moved block's from address.
		`value = cloudflare_authenticated_origin_pulls_certificate.host2.id`,
		`value = cloudflare_authenticated_origin_pulls_certificate.zone.id`,
		`value = cloudflare_authenticated_origin_pulls_certificate.other_module.id`,
		`value = data.cloudflare_authenticated_origin_pulls_certificate.host.id`,
		`from = cloudflare_authenticated_origin_pulls_certificate.host`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestInputDirForOutput(t *testing.T) {
	cfg := config{configDir: "/in", outputDir: "/out", recursive: true}
	if got := inputDirForOutput(cfg, "/out/modules/x/main.tf"); got != "/in/modules/x" {
		t.Errorf("recursive: got %q", got)
	}
	cfg.recursive = false
	if got := inputDirForOutput(cfg, "/out/main.tf"); got != "/in" {
		t.Errorf("non-recursive: got %q", got)
	}
}

// A per-hostname certificate declared in a file processed after the files that
// reference it must still have every reference renamed, and a same-named
// per-zone certificate in another module must not be touched.
func TestPerHostnameCertificateReferencesAcrossFilesAndModules(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cert := func(certType string) string {
		return `resource "cloudflare_authenticated_origin_pulls_certificate" "host" {
  zone_id     = "0da42c8d2132a9ddaf714f9e7c920711"
  certificate = "cert"
  private_key = "key"
  type        = "` + certType + `"
}
`
	}

	// a_aop.tf sorts before b_certs.tf, so it is transformed first.
	write("a_aop.tf", `resource "cloudflare_authenticated_origin_pulls" "host" {
  zone_id                                = "0da42c8d2132a9ddaf714f9e7c920711"
  hostname                               = "aop.example.com"
  authenticated_origin_pulls_certificate = cloudflare_authenticated_origin_pulls_certificate.host.id
  enabled                                = true
}

output "cert_id" {
  value = cloudflare_authenticated_origin_pulls_certificate.host.id
}
`)
	write("b_certs.tf", cert("per-hostname"))
	write("modules/zonecert/main.tf", cert("per-zone")+`
output "id" {
  value = cloudflare_authenticated_origin_pulls_certificate.host.id
}
`)

	cfg := config{
		configDir:     dir,
		outputDir:     dir,
		recursive:     true,
		sourceVersion: "v4",
		targetVersion: "v5",
	}
	if err := runFullMigration(newTestLogger(), cfg); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	read := func(rel string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	aop := read("a_aop.tf")
	if strings.Contains(aop, aopCertV4+".host") {
		t.Errorf("a_aop.tf still references the v4 type:\n%s", aop)
	}
	if n := strings.Count(aop, aopHostCert+".host.id"); n != 2 {
		t.Errorf("want cert_id and output renamed (2 references), got %d:\n%s", n, aop)
	}

	certs := read("b_certs.tf")
	if !strings.Contains(certs, `resource "`+aopHostCert+`" "host"`) ||
		!strings.Contains(certs, "from = "+aopCertV4+".host") {
		t.Errorf("b_certs.tf: want renamed resource and moved block:\n%s", certs)
	}

	mod := read("modules/zonecert/main.tf")
	if strings.Contains(mod, aopHostCert) {
		t.Errorf("per-zone certificate in another module was renamed:\n%s", mod)
	}
}
