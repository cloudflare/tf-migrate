package turnstile_widget

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "basic turnstile_widget with required fields only",
				Input: `resource "cloudflare_turnstile_widget" "example" {
  account_id = var.account_id
  name       = "example-widget"
  domains    = toset(["example.com"])
  mode       = "managed"
}`,
				Expected: `resource "cloudflare_turnstile_widget" "example" {
  account_id = var.account_id
  name       = "example-widget"
  domains    = ["example.com"]
  mode       = "managed"
}`,
			},
			{
				Name: "with multiple domains in toset",
				Input: `resource "cloudflare_turnstile_widget" "example" {
  account_id = var.account_id
  name       = "example-widget"
  domains    = toset(["example.com", "test.com", "demo.com"])
  mode       = "invisible"
}`,
				Expected: `resource "cloudflare_turnstile_widget" "example" {
  account_id = var.account_id
  name       = "example-widget"
  domains    = ["demo.com", "example.com", "test.com"]
  mode       = "invisible"
}`,
			},
			{
				Name: "domains without toset wrapper (already a list)",
				Input: `resource "cloudflare_turnstile_widget" "example" {
  account_id = var.account_id
  name       = "example-widget"
  domains    = ["example.com", "test.com"]
  mode       = "non-interactive"
}`,
				Expected: `resource "cloudflare_turnstile_widget" "example" {
  account_id = var.account_id
  name       = "example-widget"
  domains    = ["example.com", "test.com"]
  mode       = "non-interactive"
}`,
			},
			{
				Name: "with all optional fields",
				Input: `resource "cloudflare_turnstile_widget" "example" {
  account_id      = var.account_id
  name            = "example-widget"
  domains         = toset(["example.com"])
  mode            = "managed"
  region          = "world"
  bot_fight_mode  = true
  offlabel        = false
}`,
				Expected: `resource "cloudflare_turnstile_widget" "example" {
  account_id      = var.account_id
  name            = "example-widget"
  domains         = ["example.com"]
  mode            = "managed"
  region          = "world"
  bot_fight_mode  = true
  offlabel        = false
}`,
			},
			{
				Name: "all widget modes - managed, invisible, non-interactive",
				Input: `resource "cloudflare_turnstile_widget" "managed" {
  account_id = var.account_id
  name       = "managed-widget"
  domains    = toset(["managed.com"])
  mode       = "managed"
}

resource "cloudflare_turnstile_widget" "invisible" {
  account_id = var.account_id
  name       = "invisible-widget"
  domains    = toset(["invisible.com"])
  mode       = "invisible"
}

resource "cloudflare_turnstile_widget" "non_interactive" {
  account_id = var.account_id
  name       = "non-interactive-widget"
  domains    = toset(["noninteractive.com"])
  mode       = "non-interactive"
}`,
				Expected: `resource "cloudflare_turnstile_widget" "managed" {
  account_id = var.account_id
  name       = "managed-widget"
  domains    = ["managed.com"]
  mode       = "managed"
}

resource "cloudflare_turnstile_widget" "invisible" {
  account_id = var.account_id
  name       = "invisible-widget"
  domains    = ["invisible.com"]
  mode       = "invisible"
}

resource "cloudflare_turnstile_widget" "non_interactive" {
  account_id = var.account_id
  name       = "non-interactive-widget"
  domains    = ["noninteractive.com"]
  mode       = "non-interactive"
}`,
			},
			{
				Name: "with variable references preserved",
				Input: `resource "cloudflare_turnstile_widget" "example" {
  account_id = var.account_id
  name       = var.widget_name
  domains    = toset(var.domains)
  mode       = var.mode
}`,
				Expected: `resource "cloudflare_turnstile_widget" "example" {
  account_id = var.account_id
  name       = var.widget_name
  domains    = toset(var.domains)
  mode       = var.mode
}`,
			},
			{
				Name: "with for_each and literal array in domains",
				Input: `resource "cloudflare_turnstile_widget" "example" {
  for_each   = toset(var.widget_configs)
  account_id = var.account_id
  name       = each.key
  domains    = toset([each.value.domain])
  mode       = each.value.mode
}`,
				Expected: `resource "cloudflare_turnstile_widget" "example" {
  for_each   = toset(var.widget_configs)
  account_id = var.account_id
  name       = each.key
  domains    = [each.value.domain]
  mode       = each.value.mode
}`,
			},
			{
				Name: "with count",
				Input: `resource "cloudflare_turnstile_widget" "example" {
  count      = length(var.domains)
  account_id = var.account_id
  name       = "widget-${count.index}"
  domains    = toset([var.domains[count.index]])
  mode       = "managed"
}`,
				Expected: `resource "cloudflare_turnstile_widget" "example" {
  count      = length(var.domains)
  account_id = var.account_id
  name       = "widget-${count.index}"
  domains    = [var.domains[count.index]]
  mode       = "managed"
}`,
			},
			{
				Name: "with lifecycle block",
				Input: `resource "cloudflare_turnstile_widget" "example" {
  account_id = var.account_id
  name       = "example-widget"
  domains    = toset(["example.com"])
  mode       = "managed"

  lifecycle {
    prevent_destroy = true
  }
}`,
				Expected: `resource "cloudflare_turnstile_widget" "example" {
  account_id = var.account_id
  name       = "example-widget"
  domains    = ["example.com"]
  mode       = "managed"

  lifecycle {
    prevent_destroy = true
  }
}`,
			},
			{
				Name: "multiple widgets in one file",
				Input: `resource "cloudflare_turnstile_widget" "prod" {
  account_id = var.account_id
  name       = "prod-widget"
  domains    = toset(["prod.example.com"])
  mode       = "managed"
}

resource "cloudflare_turnstile_widget" "staging" {
  account_id = var.account_id
  name       = "staging-widget"
  domains    = toset(["staging.example.com", "test.example.com"])
  mode       = "invisible"
}`,
				Expected: `resource "cloudflare_turnstile_widget" "prod" {
  account_id = var.account_id
  name       = "prod-widget"
  domains    = ["prod.example.com"]
  mode       = "managed"
}

resource "cloudflare_turnstile_widget" "staging" {
  account_id = var.account_id
  name       = "staging-widget"
  domains    = ["staging.example.com", "test.example.com"]
  mode       = "invisible"
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}

func TestMigratorInterface(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("GetResourceType", func(t *testing.T) {
		if got := migrator.GetResourceType(); got != "cloudflare_turnstile_widget" {
			t.Errorf("GetResourceType() = %v, want cloudflare_turnstile_widget", got)
		}
	})

	t.Run("CanHandle", func(t *testing.T) {
		tests := []struct {
			name         string
			resourceType string
			want         bool
		}{
			{
				name:         "handles correct resource type",
				resourceType: "cloudflare_turnstile_widget",
				want:         true,
			},
			{
				name:         "rejects other resource types",
				resourceType: "cloudflare_zone",
				want:         false,
			},
			{
				name:         "rejects empty string",
				resourceType: "",
				want:         false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := migrator.CanHandle(tt.resourceType); got != tt.want {
					t.Errorf("CanHandle(%v) = %v, want %v", tt.resourceType, got, tt.want)
				}
			})
		}
	})

	t.Run("Preprocess", func(t *testing.T) {
		input := `resource "cloudflare_turnstile_widget" "test" {
  account_id = var.account_id
  name       = "test-widget"
  domains    = toset(["example.com"])
  mode       = "managed"
}`
		// Preprocess should return content unchanged
		if got := migrator.Preprocess(input); got != input {
			t.Errorf("Preprocess() modified content, want unchanged")
		}
	})
}
