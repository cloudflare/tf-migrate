package api_shield

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Migration(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Single header characteristic",
				Input: `
resource "cloudflare_api_shield" "example" {
  zone_id = "023e105f4ecef8ad9ca31a8372d0c353"

  auth_id_characteristics {
    type = "header"
    name = "authorization"
  }
}`,
				Expected: `
resource "cloudflare_api_shield" "example" {
  zone_id = "023e105f4ecef8ad9ca31a8372d0c353"

  auth_id_characteristics = [
    {
      type = "header"
      name = "authorization"
    }
  ]
}`,
			},
			{
				Name: "Single cookie characteristic",
				Input: `
resource "cloudflare_api_shield" "example" {
  zone_id = "023e105f4ecef8ad9ca31a8372d0c353"

  auth_id_characteristics {
    type = "cookie"
    name = "session_id"
  }
}`,
				Expected: `
resource "cloudflare_api_shield" "example" {
  zone_id = "023e105f4ecef8ad9ca31a8372d0c353"

  auth_id_characteristics = [
    {
      type = "cookie"
      name = "session_id"
    }
  ]
}`,
			},
			{
				Name: "Multiple characteristics",
				Input: `
resource "cloudflare_api_shield" "example" {
  zone_id = "023e105f4ecef8ad9ca31a8372d0c353"

  auth_id_characteristics {
    type = "header"
    name = "authorization"
  }

  auth_id_characteristics {
    type = "cookie"
    name = "session_id"
  }

  auth_id_characteristics {
    type = "header"
    name = "x-api-key"
  }
}`,
				Expected: `
resource "cloudflare_api_shield" "example" {
  zone_id = "023e105f4ecef8ad9ca31a8372d0c353"

  auth_id_characteristics = [
    {
      type = "header"
      name = "authorization"
    },
    {
      type = "cookie"
      name = "session_id"
    },
    {
      type = "header"
      name = "x-api-key"
    }
  ]
}`,
			},
			{
				Name: "Multiple resources",
				Input: `
resource "cloudflare_api_shield" "production" {
  zone_id = "023e105f4ecef8ad9ca31a8372d0c353"

  auth_id_characteristics {
    type = "header"
    name = "authorization"
  }
}

resource "cloudflare_api_shield" "staging" {
  zone_id = "123e456f7890a1b2c3d4e5f6a7b8c9d0"

  auth_id_characteristics {
    type = "cookie"
    name = "staging_token"
  }
}`,
				Expected: `
resource "cloudflare_api_shield" "production" {
  zone_id = "023e105f4ecef8ad9ca31a8372d0c353"

  auth_id_characteristics = [
    {
      type = "header"
      name = "authorization"
    }
  ]
}

resource "cloudflare_api_shield" "staging" {
  zone_id = "123e456f7890a1b2c3d4e5f6a7b8c9d0"

  auth_id_characteristics = [
    {
      type = "cookie"
      name = "staging_token"
    }
  ]
}`,
			},
			{
				Name: "Missing auth_id_characteristics - sets empty array",
				Input: `
resource "cloudflare_api_shield" "example" {
  zone_id = "023e105f4ecef8ad9ca31a8372d0c353"
}`,
				Expected: `
resource "cloudflare_api_shield" "example" {
  zone_id                 = "023e105f4ecef8ad9ca31a8372d0c353"
  auth_id_characteristics = []
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
