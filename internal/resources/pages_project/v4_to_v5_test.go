package pages_project

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	// Note: State transformation is handled by the provider's state upgrader (UpgradeState).
	// tf-migrate only handles HCL config transformation.
	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Minimal resource",
				Input: `resource "cloudflare_pages_project" "example" {
  account_id        = "test123"
  name              = "test-project"
  production_branch = "main"
}`,
				Expected: `resource "cloudflare_pages_project" "example" {
  account_id        = "test123"
  name              = "test-project"
  production_branch = "main"
  deployment_configs = {
    preview = {
      fail_open = false
    }
    production = {
      fail_open = false
    }
  }
}`,
			},
			{
				Name: "FailOpen true resource",
				Input: `resource "cloudflare_pages_project" "failOpenTrue" {
  account_id        = "test123"
  name              = "test-project"
  production_branch = "main"
  deployment_configs = {
    preview = {
      fail_open = true
    }
    production = {
      fail_open = true
    }
  }
}`,
				Expected: `resource "cloudflare_pages_project" "failOpenTrue" {
  account_id        = "test123"
  name              = "test-project"
  production_branch = "main"
  deployment_configs = {
    preview = {
      fail_open = true
    }
    production = {
      fail_open = true
    }
  }
}`,
			},
			{
				Name: "Resource with build_config block",
				Input: `resource "cloudflare_pages_project" "example" {
  account_id        = "test123"
  name              = "test-project"
  production_branch = "main"

  build_config {
    build_command   = "npm run build"
    destination_dir = "public"
  }
}`,
				Expected: `resource "cloudflare_pages_project" "example" {
  account_id        = "test123"
  name              = "test-project"
  production_branch = "main"

  build_config = {
    build_command   = "npm run build"
    destination_dir = "public"
  }
  deployment_configs = {
    preview = {
      fail_open = false
    }
    production = {
      fail_open = false
    }
  }
}`,
			},
			{
				Name: "Resource with source and config blocks",
				Input: `resource "cloudflare_pages_project" "example" {
  account_id        = "test123"
  name              = "test-project"
  production_branch = "main"

  source {
    type = "github"

    config {
      owner                         = "cloudflare"
      repo_name                     = "test-repo"
      production_branch             = "main"
      production_deployment_enabled = true
    }
  }
}`,
				Expected: `resource "cloudflare_pages_project" "example" {
  account_id        = "test123"
  name              = "test-project"
  production_branch = "main"

  source = {
    type = "github"
    config = {
      owner                          = "cloudflare"
      repo_name                      = "test-repo"
      production_branch              = "main"
      production_deployments_enabled = true
    }
  }
  deployment_configs = {
    preview = {
      fail_open = false
    }
    production = {
      fail_open = false
    }
  }
}`,
			},
			{
				Name: "Resource with deployment_configs and nested blocks",
				Input: `resource "cloudflare_pages_project" "example" {
  account_id        = "test123"
  name              = "test-project"
  production_branch = "main"

  deployment_configs {
    preview {
      compatibility_date = "2024-01-01"

      placement {
        mode = "smart"
      }
    }

    production {
      compatibility_date = "2024-01-01"

      placement {
        mode = "smart"
      }
    }
  }
}`,
				Expected: `resource "cloudflare_pages_project" "example" {
  account_id        = "test123"
  name              = "test-project"
  production_branch = "main"

  deployment_configs = {
    preview = {
      compatibility_date = "2024-01-01"
      fail_open          = false
      placement = {
        mode = "smart"
      }
    }
    production = {
      compatibility_date = "2024-01-01"
      fail_open          = false
      placement = {
        mode = "smart"
      }
    }
  }
}`,
			},
			{
				Name: "Full resource with all nested blocks",
				Input: `resource "cloudflare_pages_project" "example" {
  account_id        = "test123"
  name              = "test-project"
  production_branch = "main"

  build_config {
    build_command   = "npm run build"
    destination_dir = "public"
  }

  source {
    type = "github"

    config {
      owner                         = "cloudflare"
      repo_name                     = "test-repo"
      production_deployment_enabled = true
    }
  }

  deployment_configs {
    preview {
      placement {
        mode = "smart"
      }
    }
  }
}`,
				Expected: `resource "cloudflare_pages_project" "example" {
  account_id        = "test123"
  name              = "test-project"
  production_branch = "main"

  build_config = {
    build_command   = "npm run build"
    destination_dir = "public"
  }
  source = {
    type = "github"
    config = {
      owner                          = "cloudflare"
      repo_name                      = "test-repo"
      production_deployments_enabled = true
    }
  }
  deployment_configs = {
    preview = {
      fail_open = false
      placement = {
        mode = "smart"
      }
    }
  }
}`,
			},
			{
				Name: "Resource with kv_namespaces binding",
				Input: `resource "cloudflare_pages_project" "example" {
  account_id        = "test123"
  name              = "test-project"
  production_branch = "main"

  deployment_configs {
    preview {
      kv_namespaces = {
        MY_KV = "kv-namespace-id-123"
      }
    }
    production {
      kv_namespaces = {
        PROD_KV = "kv-namespace-id-456"
      }
    }
  }
}`,
				Expected: `resource "cloudflare_pages_project" "example" {
  account_id        = "test123"
  name              = "test-project"
  production_branch = "main"

  deployment_configs = {
    preview = {
      fail_open = false
      kv_namespaces = {
        MY_KV = {
          namespace_id = "kv-namespace-id-123"
        }
      }
    }
    production = {
      fail_open = false
      kv_namespaces = {
        PROD_KV = {
          namespace_id = "kv-namespace-id-456"
        }
      }
    }
  }
}`,
			},
			{
				Name: "Resource with service_binding blocks",
				Input: `resource "cloudflare_pages_project" "example" {
  account_id        = "test123"
  name              = "test-project"
  production_branch = "main"

  deployment_configs {
    production {
      service_binding {
        name    = "MY_SERVICE"
        service = "my-worker"
      }
      service_binding {
        name        = "OTHER_SERVICE"
        service     = "other-worker"
        environment = "production"
      }
    }
  }
}`,
				Expected: `resource "cloudflare_pages_project" "example" {
  account_id        = "test123"
  name              = "test-project"
  production_branch = "main"

  deployment_configs = {
    production = {
      fail_open = false
      services = {
        MY_SERVICE = {
          service = "my-worker"
        }
        OTHER_SERVICE = {
          service     = "other-worker"
          environment = "production"
        }
      }
    }
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})
}
