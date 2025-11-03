package argo

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// Test configuration transformations
	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "smart_routing only",
				Input: `resource "cloudflare_argo" "example" {
  zone_id       = var.zone_id
  smart_routing = "on"
}`,
				Expected: `resource "cloudflare_argo_smart_routing" "example" {
  zone_id = var.zone_id
  value   = "on"
}
moved {
  from = cloudflare_argo.example
  to   = cloudflare_argo_smart_routing.example
}`,
			},
			{
				Name: "smart_routing off",
				Input: `resource "cloudflare_argo" "example" {
  zone_id       = var.api_openai_com_zone_id
  smart_routing = "off"
}`,
				Expected: `resource "cloudflare_argo_smart_routing" "example" {
  zone_id = var.api_openai_com_zone_id
  value   = "off"
}
moved {
  from = cloudflare_argo.example
  to   = cloudflare_argo_smart_routing.example
}`,
			},
			{
				Name: "smart_routing with zone reference",
				Input: `resource "cloudflare_argo" "example" {
  zone_id       = cloudflare_zone.operator_chatgpt_com.id
  smart_routing = "on"
}`,
				Expected: `resource "cloudflare_argo_smart_routing" "example" {
  zone_id = cloudflare_zone.operator_chatgpt_com.id
  value   = "on"
}
moved {
  from = cloudflare_argo.example
  to   = cloudflare_argo_smart_routing.example
}`,
			},
			{
				Name: "both smart_routing and tiered_caching",
				Input: `resource "cloudflare_argo" "example" {
  zone_id         = var.zone_id
  smart_routing   = "on"
  tiered_caching  = "on"
}`,
				Expected: `resource "cloudflare_argo_smart_routing" "example" {
  zone_id = var.zone_id
  value   = "on"
}
moved {
  from = cloudflare_argo.example
  to   = cloudflare_argo_smart_routing.example
}
resource "cloudflare_argo_tiered_caching" "example_tiered" {
  zone_id = var.zone_id
  value   = "on"
}
moved {
  from = cloudflare_argo.example
  to   = cloudflare_argo_tiered_caching.example_tiered
}`,
			},
			{
				Name: "tiered_caching only",
				Input: `resource "cloudflare_argo" "example" {
  zone_id         = var.zone_id
  tiered_caching  = "on"
}`,
				Expected: `resource "cloudflare_argo_tiered_caching" "example" {
  zone_id = var.zone_id
  value   = "on"
}
moved {
  from = cloudflare_argo.example
  to   = cloudflare_argo_tiered_caching.example
}`,
			},
			{
				Name: "no attributes defaults to smart_routing off",
				Input: `resource "cloudflare_argo" "example" {
  zone_id = var.zone_id
}`,
				Expected: `resource "cloudflare_argo_smart_routing" "example" {
  zone_id = var.zone_id
  value   = "off"
}
moved {
  from = cloudflare_argo.example
  to   = cloudflare_argo_smart_routing.example
}`,
			},
			{
				Name: "with lifecycle block",
				Input: `resource "cloudflare_argo" "example" {
  zone_id       = var.zone_id
  smart_routing = "on"

  lifecycle {
    prevent_destroy = true
  }
}`,
				Expected: `resource "cloudflare_argo_smart_routing" "example" {
  zone_id = var.zone_id
  value   = "on"
  lifecycle {
    prevent_destroy = true
  }
}
moved {
  from = cloudflare_argo.example
  to   = cloudflare_argo_smart_routing.example
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
