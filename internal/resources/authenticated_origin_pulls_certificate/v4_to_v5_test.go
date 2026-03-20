package authenticated_origin_pulls_certificate

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestConfigTransformation_PerZone(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "per-zone certificate with all fields",
			Input: `
resource "cloudflare_authenticated_origin_pulls_certificate" "test" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----\nMIIE...test\n-----END CERTIFICATE-----"
  private_key = "-----BEGIN PRIVATE KEY-----\nMIIE...test\n-----END PRIVATE KEY-----"
  type        = "per-zone"
}`,
			Expected: `
resource "cloudflare_authenticated_origin_pulls_certificate" "test" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----\nMIIE...test\n-----END CERTIFICATE-----"
  private_key = "-----BEGIN PRIVATE KEY-----\nMIIE...test\n-----END PRIVATE KEY-----"
}`,
		},
		{
			Name: "per-zone certificate with variable references",
			Input: `
resource "cloudflare_authenticated_origin_pulls_certificate" "test" {
  zone_id     = var.cloudflare_zone_id
  certificate = file("cert.pem")
  private_key = file("key.pem")
  type        = "per-zone"
}`,
			Expected: `
resource "cloudflare_authenticated_origin_pulls_certificate" "test" {
  zone_id     = var.cloudflare_zone_id
  certificate = file("cert.pem")
  private_key = file("key.pem")
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestConfigTransformation_PerHostname(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "per-hostname certificate with all fields",
			Input: `
resource "cloudflare_authenticated_origin_pulls_certificate" "test" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----\nMIIE...test\n-----END CERTIFICATE-----"
  private_key = "-----BEGIN PRIVATE KEY-----\nMIIE...test\n-----END PRIVATE KEY-----"
  type        = "per-hostname"
}`,
			Expected: `
resource "cloudflare_authenticated_origin_pulls_hostname_certificate" "test" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----\nMIIE...test\n-----END CERTIFICATE-----"
  private_key = "-----BEGIN PRIVATE KEY-----\nMIIE...test\n-----END PRIVATE KEY-----"
}

moved {
  from = cloudflare_authenticated_origin_pulls_certificate.test
  to   = cloudflare_authenticated_origin_pulls_hostname_certificate.test
}`,
		},
		{
			Name: "per-hostname certificate with variable references",
			Input: `
resource "cloudflare_authenticated_origin_pulls_certificate" "test" {
  zone_id     = var.cloudflare_zone_id
  certificate = file("hostname_cert.pem")
  private_key = file("hostname_key.pem")
  type        = "per-hostname"
}`,
			Expected: `
resource "cloudflare_authenticated_origin_pulls_hostname_certificate" "test" {
  zone_id     = var.cloudflare_zone_id
  certificate = file("hostname_cert.pem")
  private_key = file("hostname_key.pem")
}

moved {
  from = cloudflare_authenticated_origin_pulls_certificate.test
  to   = cloudflare_authenticated_origin_pulls_hostname_certificate.test
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}

func TestConfigTransformation_Mixed(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	tests := []testhelpers.ConfigTestCase{
		{
			Name: "both per-zone and per-hostname in same file",
			Input: `
resource "cloudflare_authenticated_origin_pulls_certificate" "per_zone" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----\nzone cert\n-----END CERTIFICATE-----"
  private_key = "-----BEGIN PRIVATE KEY-----\nzone key\n-----END PRIVATE KEY-----"
  type        = "per-zone"
}

resource "cloudflare_authenticated_origin_pulls_certificate" "per_hostname" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----\nhost cert\n-----END CERTIFICATE-----"
  private_key = "-----BEGIN PRIVATE KEY-----\nhost key\n-----END PRIVATE KEY-----"
  type        = "per-hostname"
}`,
			Expected: `
resource "cloudflare_authenticated_origin_pulls_certificate" "per_zone" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----\nzone cert\n-----END CERTIFICATE-----"
  private_key = "-----BEGIN PRIVATE KEY-----\nzone key\n-----END PRIVATE KEY-----"
}

resource "cloudflare_authenticated_origin_pulls_hostname_certificate" "per_hostname" {
  zone_id     = "abc123"
  certificate = "-----BEGIN CERTIFICATE-----\nhost cert\n-----END CERTIFICATE-----"
  private_key = "-----BEGIN PRIVATE KEY-----\nhost key\n-----END PRIVATE KEY-----"
}

moved {
  from = cloudflare_authenticated_origin_pulls_certificate.per_hostname
  to   = cloudflare_authenticated_origin_pulls_hostname_certificate.per_hostname
}`,
		},
	}

	testhelpers.RunConfigTransformTests(t, tests, migrator)
}
