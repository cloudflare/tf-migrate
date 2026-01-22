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

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "basic state with required fields only",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "1x77AFD96B08Cce6f9SvX4Yt4E2p6UBYV",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "example-widget",
    "domains": ["example.com"],
    "mode": "managed",
    "secret": "0x4AABBaaFFGEG1dFF3ACFdTF6FDYyGDKAfEGLqw"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "1x77AFD96B08Cce6f9SvX4Yt4E2p6UBYV",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "example-widget",
    "domains": ["example.com"],
    "mode": "managed",
    "secret": "0x4AABBaaFFGEG1dFF3ACFdTF6FDYyGDKAfEGLqw",
    "sitekey": "1x77AFD96B08Cce6f9SvX4Yt4E2p6UBYV"
  }
}`,
			},
			{
				Name: "with all optional fields",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "1x77AFD96B08Cce6f9SvX4Yt4E2p6UBYV",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "example-widget",
    "domains": ["example.com", "test.com"],
    "mode": "invisible",
    "region": "world",
    "bot_fight_mode": true,
    "offlabel": false,
    "secret": "0x4AABBaaFFGEG1dFF3ACFdTF6FDYyGDKAfEGLqw"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "1x77AFD96B08Cce6f9SvX4Yt4E2p6UBYV",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "example-widget",
    "domains": ["example.com", "test.com"],
    "mode": "invisible",
    "region": "world",
    "bot_fight_mode": true,
    "offlabel": false,
    "secret": "0x4AABBaaFFGEG1dFF3ACFdTF6FDYyGDKAfEGLqw",
    "sitekey": "1x77AFD96B08Cce6f9SvX4Yt4E2p6UBYV"
  }
}`,
			},
			{
				Name: "with non-interactive mode",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "2y88BEE07C19Ddf7g0TwY5Zu5F3r7VCZW",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "non-interactive-widget",
    "domains": ["noninteractive.com"],
    "mode": "non-interactive",
    "secret": "0x5BBCCbbGGFHF2eGG4BDGeUG7GEZzHELBgFHMrx"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "2y88BEE07C19Ddf7g0TwY5Zu5F3r7VCZW",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "non-interactive-widget",
    "domains": ["noninteractive.com"],
    "mode": "non-interactive",
    "secret": "0x5BBCCbbGGFHF2eGG4BDGeUG7GEZzHELBgFHMrx"
,
    "sitekey": "2y88BEE07C19Ddf7g0TwY5Zu5F3r7VCZW"
  }
}`,
			},
			{
				Name: "with empty domains array",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "3z99CFF18D20Eeg8h1UxZ6Av6G4s8WDAY",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "empty-domains-widget",
    "domains": [],
    "mode": "managed",
    "secret": "0x6CCDDccHHGIG3fHH5CEHfVH8HFAaIFMChGINsy"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "3z99CFF18D20Eeg8h1UxZ6Av6G4s8WDAY",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "empty-domains-widget",
    "domains": [],
    "mode": "managed",
    "secret": "0x6CCDDccHHGIG3fHH5CEHfVH8HFAaIFMChGINsy"
,
    "sitekey": "3z99CFF18D20Eeg8h1UxZ6Av6G4s8WDAY"
  }
}`,
			},
			{
				Name: "with null optional fields",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "4a00DGG29E31Ffh9i2VyA7Bw7H5t9XEBZ",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "null-fields-widget",
    "domains": ["example.com"],
    "mode": "managed",
    "region": null,
    "bot_fight_mode": null,
    "offlabel": null,
    "secret": "0x7DDEEddIIHJH4gII6DFIgWI9IGBbJGNDiHJOtz"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "4a00DGG29E31Ffh9i2VyA7Bw7H5t9XEBZ",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "null-fields-widget",
    "domains": ["example.com"],
    "mode": "managed",
    "region": null,
    "bot_fight_mode": null,
    "offlabel": null,
    "secret": "0x7DDEEddIIHJH4gII6DFIgWI9IGBbJGNDiHJOtz"
,
    "sitekey": "4a00DGG29E31Ffh9i2VyA7Bw7H5t9XEBZ"
  }
}`,
			},
			{
				Name: "state without schema_version field",
				Input: `{
  "attributes": {
    "id": "5b11EHH30F42Ggi0j3WzB8Cx8I6u0YFCA",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "no-schema-version",
    "domains": ["example.com"],
    "mode": "invisible",
    "secret": "0x8EEFFeeJJIKI5hJJ7EGJhXJ0JHCcKHOEjIKPua"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "5b11EHH30F42Ggi0j3WzB8Cx8I6u0YFCA",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "no-schema-version",
    "domains": ["example.com"],
    "mode": "invisible",
    "secret": "0x8EEFFeeJJIKI5hJJ7EGJhXJ0JHCcKHOEjIKPua"
,
    "sitekey": "5b11EHH30F42Ggi0j3WzB8Cx8I6u0YFCA"
  }
}`,
			},
			{
				Name: "invalid instance without attributes",
				Input: `{
  "schema_version": 1
}`,
				Expected: `{
  "schema_version": 0
}`,
			},
			{
				Name: "empty instance",
				Input: `{}`,
				Expected: `{
  "schema_version": 0
}`,
			},
			{
				Name: "with multiple domains",
				Input: `{
  "schema_version": 1,
  "attributes": {
    "id": "6c22FII41G53Hhj1k4XaC9Dy9J7v1ZGDB",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "multi-domain-widget",
    "domains": ["one.com", "two.com", "three.com", "four.com"],
    "mode": "managed",
    "secret": "0x9FFGGffKKJLJ6iKK8FHKiYK1KIDdLIPFkJLQvb"
  }
}`,
				Expected: `{
  "schema_version": 0,
  "attributes": {
    "id": "6c22FII41G53Hhj1k4XaC9Dy9J7v1ZGDB",
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "name": "multi-domain-widget",
    "domains": ["four.com", "one.com", "three.com", "two.com"],
    "mode": "managed",
    "secret": "0x9FFGGffKKJLJ6iKK8FHKiYK1KIDdLIPFkJLQvb"
,
    "sitekey": "6c22FII41G53Hhj1k4XaC9Dy9J7v1ZGDB"
  }
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
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

