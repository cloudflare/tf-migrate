package pages_project

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

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
      usage_model         = "bundled"
      fail_open           = false
    }
    production = {
      fail_open           = false
      usage_model         = "bundled"
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
      fail_open           = true
    }
    production = {
      fail_open           = true
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
      usage_model         = "bundled"
      fail_open           = false
    }
    production = {
      fail_open           = false
      usage_model         = "bundled"
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
      usage_model         = "bundled"
      fail_open           = false
    }
    production = {
      fail_open           = false
      usage_model         = "bundled"
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
      compatibility_date  = "2024-01-01"
      usage_model         = "bundled"
      fail_open           = false
      placement = {
        mode = "smart"
      }
    }
    production = {
      compatibility_date  = "2024-01-01"
      fail_open           = false
      usage_model         = "bundled"
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
      usage_model         = "bundled"
      fail_open           = false
      placement = {
        mode = "smart"
      }
    }
  }
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Minimal state",
				Input: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main"
  }
}`,
				Expected: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "canonical_deployment": null,
    "framework": "",
    "framework_version": "",
    "latest_deployment": null,
    "source": null,
    "uses_functions": null
  },
  "schema_version": 0
}`,
			},
			{
				Name: "State with build_config array to object",
				Input: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": [{
      "build_command": "npm run build",
      "destination_dir": "public"
    }]
  }
}`,
				Expected: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {
      "build_command": "npm run build",
      "destination_dir": "public"
    },
    "canonical_deployment": null,
    "framework": "",
    "framework_version": "",
    "latest_deployment": null,
    "source": null,
    "uses_functions": null
  },
  "schema_version": 0
}`,
			},
			{
				Name: "State with source config and field rename",
				Input: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "source": [{
      "type": "github",
      "config": [{
        "owner": "cloudflare",
        "repo_name": "test-repo",
        "production_deployment_enabled": true
      }]
    }]
  }
}`,
				Expected: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "source": {
      "type": "github",
      "config": {
        "owner": "cloudflare",
        "repo_name": "test-repo",
      "production_deployments_enabled": true
    }
  },

"canonical_deployment": null,
  "framework": "",
    "framework_version": "",
    "latest_deployment": null,
    "uses_functions": null
  },
  "schema_version": 0
}`,
			},
			{
				Name: "State with environment_variables and secrets merge",
				Input: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "deployment_configs": [{
      "preview": [{
        "environment_variables": {
          "NODE_ENV": "preview",
          "API_URL": "https://preview-api.example.com"
        },
        "secrets": {
          "API_KEY": "secret-key-123",
          "DATABASE_PASSWORD": "secret-pwd"
        }
      }]
    }]
  }
}`,
				Expected: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "deployment_configs": {
      "preview": {
        "always_use_latest_compatibility_date": false,
        "compatibility_date": null,
        "compatibility_flags": null,
        "usage_model": "bundled",
        "fail_open": false,
        "env_vars": {
          "NODE_ENV": {
            "type": "plain_text",
            "value": "preview"
          },
          "API_URL": {
            "type": "plain_text",
            "value": "https://preview-api.example.com"
          },
          "API_KEY": {
            "type": "secret_text",
            "value": "secret-key-123"
          },
          "DATABASE_PASSWORD": {
            "type": "secret_text",
            "value": "secret-pwd"
          }
        },
        "ai_bindings": null,
        "analytics_engine_datasets": null,
        "browsers": null,
        "build_image_major_version": 3,
        "hyperdrive_bindings": null,
        "kv_namespaces": null,
        "limits": null,
        "mtls_certificates": null,
        "placement": null,
        "queue_producers": null,
        "d1_databases": null,
        "durable_object_namespaces": null,
        "r2_buckets": null,
        "services": null,
        "vectorize_bindings": null,
        "wrangler_config_hash": null
      }
    },
"canonical_deployment": null,
  "framework": "",
    "framework_version": "",
    "latest_deployment": null,
    "source": null,
    "uses_functions": null
  },
  "schema_version": 0
}`,
			},
			{
				Name: "State with TypeMap wrapping (kv_namespaces, d1_databases)",
				Input: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "deployment_configs": [{
      "preview": [{
        "kv_namespaces": {
          "MY_KV": "kv-namespace-id-123"
        },
        "d1_databases": {
          "MY_DB": "database-id-456"
        }
      }]
    }]
  }
}`,
				Expected: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "deployment_configs": {
      "preview": {
        "always_use_latest_compatibility_date": false,
        "compatibility_date": null,
        "compatibility_flags": null,
        "usage_model": "bundled",
        "fail_open": false,
        "kv_namespaces": {
          "MY_KV": {
            "namespace_id": "kv-namespace-id-123"
          }
        },
        "d1_databases": {
          "MY_DB": {
            "id": "database-id-456"
          }
        },
        "ai_bindings": null,
        "analytics_engine_datasets": null,
        "browsers": null,
        "build_image_major_version": 3,
        "hyperdrive_bindings": null,
        "limits": null,
        "mtls_certificates": null,
        "placement": null,
        "queue_producers": null,
        "durable_object_namespaces": null,
        "r2_buckets": null,
        "env_vars": null,
        "services": null,
        "vectorize_bindings": null,
        "wrangler_config_hash": null
      }
    },
"canonical_deployment": null,
  "framework": "",
    "framework_version": "",
    "latest_deployment": null,
    "source": null,
    "uses_functions": null
  },
  "schema_version": 0
}`,
			},
			{
				Name: "State with service_binding to services conversion",
				Input: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "deployment_configs": [{
      "preview": [{
        "service_binding": [
          {
            "name": "MY_SERVICE",
            "service": "my-worker",
            "environment": "production"
          },
          {
            "name": "ANOTHER_SERVICE",
            "service": "another-worker",
            "environment": "preview"
          }
        ]
      }]
    }]
  }
}`,
				Expected: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "deployment_configs": {
      "preview": {
        "always_use_latest_compatibility_date": false,
        "compatibility_date": null,
        "compatibility_flags": null,
        "usage_model": "bundled",
        "fail_open": false,
        "services": {
          "MY_SERVICE": {
            "service": "my-worker",
            "environment": "production"
          },
          "ANOTHER_SERVICE": {
            "service": "another-worker",
            "environment": "preview"
          }
        },
        "ai_bindings": null,
        "analytics_engine_datasets": null,
        "browsers": null,
        "build_image_major_version": 3,
        "hyperdrive_bindings": null,
        "kv_namespaces": null,
        "limits": null,
        "mtls_certificates": null,
        "placement": null,
        "queue_producers": null,
        "d1_databases": null,
        "durable_object_namespaces": null,
        "r2_buckets": null,
        "env_vars": null,
        "vectorize_bindings": null,
        "wrangler_config_hash": null
      }
    },
"canonical_deployment": null,
  "framework": "",
    "framework_version": "",
    "latest_deployment": null,
    "source": null,
    "uses_functions": null
  },
  "schema_version": 0
}`,
			},
			{
				Name: "Full state with all transformations",
				Input: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": [{
      "build_command": "npm run build",
      "destination_dir": "public"
    }],
    "source": [{
      "type": "github",
      "config": [{
        "owner": "cloudflare",
        "repo_name": "test-repo",
        "production_deployment_enabled": true
      }]
    }],
    "deployment_configs": [{
      "preview": [{
        "environment_variables": {
          "NODE_ENV": "preview"
        },
        "secrets": {
          "API_KEY": "secret"
        },
        "kv_namespaces": {
          "MY_KV": "kv-id"
        },
        "placement": [{
          "mode": "smart"
        }]
      }],
      "production": [{
        "environment_variables": {
          "NODE_ENV": "production"
        },
        "r2_buckets": {
          "MY_BUCKET": "bucket-name"
        }
      }]
    }]
  }
}`,
				Expected: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {
      "build_command": "npm run build",
      "destination_dir": "public"
    },
    "source": {
      "type": "github",
      "config": {
        "owner": "cloudflare",
        "repo_name": "test-repo",
        "production_deployments_enabled": true
      }
    },
    "deployment_configs": {
      "preview": {
        "always_use_latest_compatibility_date": false,
        "compatibility_date": null,
        "compatibility_flags": null,
        "usage_model": "bundled",
        "fail_open": false,
        "env_vars": {
          "NODE_ENV": {
            "type": "plain_text",
            "value": "preview"
          },
          "API_KEY": {
            "type": "secret_text",
            "value": "secret"
          }
        },
        "kv_namespaces": {
          "MY_KV": {
            "namespace_id": "kv-id"
          }
        },
        "placement": {
          "mode": "smart"
        },
        "ai_bindings": null,
        "analytics_engine_datasets": null,
        "browsers": null,
        "build_image_major_version": 3,
        "hyperdrive_bindings": null,
        "limits": null,
        "mtls_certificates": null,
        "queue_producers": null,
        "d1_databases": null,
        "durable_object_namespaces": null,
        "r2_buckets": null,
        "services": null,
        "vectorize_bindings": null,
        "wrangler_config_hash": null
      },
      "production": {
        "always_use_latest_compatibility_date": false,
        "compatibility_date": null,
        "compatibility_flags": null,
        "usage_model": "bundled",
        "fail_open": false,
        "env_vars": {
          "NODE_ENV": {
            "type": "plain_text",
            "value": "production"
          }
        },
        "r2_buckets": {
          "MY_BUCKET": {
            "name": "bucket-name"
          }
        },
        "ai_bindings": null,
        "analytics_engine_datasets": null,
        "browsers": null,
        "build_image_major_version": 3,
        "hyperdrive_bindings": null,
        "kv_namespaces": null,
        "limits": null,
        "mtls_certificates": null,
        "placement": null,
        "queue_producers": null,
        "d1_databases": null,
        "durable_object_namespaces": null,
        "services": null,
        "vectorize_bindings": null,
        "wrangler_config_hash": null
      }
    },
    "canonical_deployment": null,
    "framework": "",
    "framework_version": "",
    "latest_deployment": null,
    "uses_functions": null
  },
  "schema_version": 0
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})

	t.Run("DefaultValueHandling", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "Missing usage_model gets default (bundled)",
				Input: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "deployment_configs": [{
      "production": [{
        "compatibility_date": "2024-01-01"
      }]
    }]
  }
}`,
				Expected: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "deployment_configs": {
      "production": {
        "always_use_latest_compatibility_date": false,
        "compatibility_date": "2024-01-01",
        "compatibility_flags": null,
        "usage_model": "bundled",
        "fail_open": false,
        "ai_bindings": null,
        "analytics_engine_datasets": null,
        "browsers": null,
        "build_image_major_version": 3,
        "hyperdrive_bindings": null,
        "kv_namespaces": null,
        "limits": null,
        "mtls_certificates": null,
        "placement": null,
        "queue_producers": null,
        "d1_databases": null,
        "durable_object_namespaces": null,
        "r2_buckets": null,
        "env_vars": null,
        "services": null,
        "vectorize_bindings": null,
        "wrangler_config_hash": null
      }
    },
"canonical_deployment": null,
  "framework": "",
    "framework_version": "",
    "latest_deployment": null,
    "source": null,
    "uses_functions": null
  },
  "schema_version": 0
}`,
			},
			{
				Name: "Missing fail_open gets v4 default (false)",
				Input: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "deployment_configs": [{
      "preview": [{
        "compatibility_date": "2024-01-01",
        "usage_model": "standard"
      }]
    }]
  }
}`,
				Expected: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "deployment_configs": {
      "preview": {
        "always_use_latest_compatibility_date": false,
        "compatibility_date": "2024-01-01",
        "compatibility_flags": null,
        "usage_model": "standard",
        "fail_open": false,
        "ai_bindings": null,
        "analytics_engine_datasets": null,
        "browsers": null,
        "build_image_major_version": 3,
        "hyperdrive_bindings": null,
        "kv_namespaces": null,
        "limits": null,
        "mtls_certificates": null,
        "placement": null,
        "queue_producers": null,
        "d1_databases": null,
        "durable_object_namespaces": null,
        "r2_buckets": null,
        "env_vars": null,
        "services": null,
        "vectorize_bindings": null,
        "wrangler_config_hash": null
      }
    },
"canonical_deployment": null,
  "framework": "",
    "framework_version": "",
    "latest_deployment": null,
    "source": null,
    "uses_functions": null
  },
  "schema_version": 0
}`,
			},
			{
				Name: "Existing usage_model is preserved (not overridden)",
				Input: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "deployment_configs": [{
      "production": [{
        "compatibility_date": "2024-01-01",
        "usage_model": "standard"
      }]
    }]
  }
}`,
				Expected: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "deployment_configs": {
      "production": {
        "always_use_latest_compatibility_date": false,
        "compatibility_date": "2024-01-01",
        "compatibility_flags": null,
        "usage_model": "standard",
        "fail_open": false,
        "ai_bindings": null,
        "analytics_engine_datasets": null,
        "browsers": null,
        "build_image_major_version": 3,
        "hyperdrive_bindings": null,
        "kv_namespaces": null,
        "limits": null,
        "mtls_certificates": null,
        "placement": null,
        "queue_producers": null,
        "d1_databases": null,
        "durable_object_namespaces": null,
        "r2_buckets": null,
        "env_vars": null,
        "services": null,
        "vectorize_bindings": null,
        "wrangler_config_hash": null
      }
    },
"canonical_deployment": null,
  "framework": "",
    "framework_version": "",
    "latest_deployment": null,
    "source": null,
    "uses_functions": null
  },
  "schema_version": 0
}`,
			},
			{
				Name: "Existing fail_open is preserved",
				Input: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "deployment_configs": [{
      "production": [{
        "compatibility_date": "2024-01-01",
        "fail_open": false
      }]
    }]
  }
}`,
				Expected: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "deployment_configs": {
      "production": {
        "always_use_latest_compatibility_date": false,
        "compatibility_date": "2024-01-01",
        "compatibility_flags": null,
        "usage_model": "bundled",
        "fail_open": false,
        "ai_bindings": null,
        "analytics_engine_datasets": null,
        "browsers": null,
        "build_image_major_version": 3,
        "hyperdrive_bindings": null,
        "kv_namespaces": null,
        "limits": null,
        "mtls_certificates": null,
        "placement": null,
        "queue_producers": null,
        "d1_databases": null,
        "durable_object_namespaces": null,
        "r2_buckets": null,
        "env_vars": null,
        "services": null,
        "vectorize_bindings": null,
        "wrangler_config_hash": null
      }
    },
"canonical_deployment": null,
  "framework": "",
    "framework_version": "",
    "latest_deployment": null,
    "source": null,
    "uses_functions": null
  },
  "schema_version": 0
}`,
			},
			{
				Name: "Both preview and production missing defaults",
				Input: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "deployment_configs": [{
      "preview": [{
        "compatibility_date": "2024-01-01"
      }],
      "production": [{
        "compatibility_date": "2024-01-01"
      }]
    }]
  }
}`,
				Expected: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "deployment_configs": {
      "preview": {
        "always_use_latest_compatibility_date": false,
        "compatibility_date": "2024-01-01",
        "compatibility_flags": null,
        "usage_model": "bundled",
        "fail_open": false,
        "ai_bindings": null,
        "analytics_engine_datasets": null,
        "browsers": null,
        "build_image_major_version": 3,
        "hyperdrive_bindings": null,
        "kv_namespaces": null,
        "limits": null,
        "mtls_certificates": null,
        "placement": null,
        "queue_producers": null,
        "d1_databases": null,
        "durable_object_namespaces": null,
        "r2_buckets": null,
        "env_vars": null,
        "services": null,
        "vectorize_bindings": null,
        "wrangler_config_hash": null
      },
      "production": {
        "always_use_latest_compatibility_date": false,
        "compatibility_date": "2024-01-01",
        "compatibility_flags": null,
        "usage_model": "bundled",
        "fail_open": false,
        "ai_bindings": null,
        "analytics_engine_datasets": null,
        "browsers": null,
        "build_image_major_version": 3,
        "hyperdrive_bindings": null,
        "kv_namespaces": null,
        "limits": null,
        "mtls_certificates": null,
        "placement": null,
        "queue_producers": null,
        "d1_databases": null,
        "durable_object_namespaces": null,
        "r2_buckets": null,
        "env_vars": null,
        "services": null,
        "vectorize_bindings": null,
        "wrangler_config_hash": null
      }
    },
"canonical_deployment": null,
  "framework": "",
    "framework_version": "",
    "latest_deployment": null,
    "source": null,
    "uses_functions": null
  },
  "schema_version": 0
}`,
			},
			{
				Name: "State with empty compatibility_flags arrays",
				Input: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "deployment_configs": [{
      "preview": [{
        "compatibility_date": "2024-01-01",
        "compatibility_flags": null,
        "usage_model": "bundled",
        "fail_open": false
      }],
      "production": [{
        "compatibility_date": "2024-01-01",
        "compatibility_flags": null,
        "usage_model": "bundled",
        "fail_open": false
      }]
    }]
  }
}`,
				Expected: `{
  "type": "cloudflare_pages_project",
  "name": "example",
  "attributes": {
    "account_id": "test123",
    "name": "test-project",
    "production_branch": "main",
    "build_config": {},
    "deployment_configs": {
      "preview": {
        "always_use_latest_compatibility_date": false,
        "compatibility_date": "2024-01-01",
        "compatibility_flags": null,
        "usage_model": "bundled",
        "fail_open": false,
        "ai_bindings": null,
        "analytics_engine_datasets": null,
        "browsers": null,
        "build_image_major_version": 3,
        "hyperdrive_bindings": null,
        "kv_namespaces": null,
        "limits": null,
        "mtls_certificates": null,
        "placement": null,
        "queue_producers": null,
        "d1_databases": null,
        "durable_object_namespaces": null,
        "r2_buckets": null,
        "env_vars": null,
        "services": null,
        "vectorize_bindings": null,
        "wrangler_config_hash": null
      },
      "production": {
        "always_use_latest_compatibility_date": false,
        "compatibility_date": "2024-01-01",
        "compatibility_flags": null,
        "usage_model": "bundled",
        "fail_open": false,
        "ai_bindings": null,
        "analytics_engine_datasets": null,
        "browsers": null,
        "build_image_major_version": 3,
        "hyperdrive_bindings": null,
        "kv_namespaces": null,
        "limits": null,
        "mtls_certificates": null,
        "placement": null,
        "queue_producers": null,
        "d1_databases": null,
        "durable_object_namespaces": null,
        "r2_buckets": null,
        "env_vars": null,
        "services": null,
        "vectorize_bindings": null,
        "wrangler_config_hash": null
      }
    },
"canonical_deployment": null,
  "framework": "",
    "framework_version": "",
    "latest_deployment": null,
    "source": null,
    "uses_functions": null
  },
  "schema_version": 0
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
