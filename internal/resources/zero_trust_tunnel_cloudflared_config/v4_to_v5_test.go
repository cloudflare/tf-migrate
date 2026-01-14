package zero_trust_tunnel_cloudflared_config

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "Minimal config - deprecated resource name",
				Input: `resource "cloudflare_tunnel_config" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4d76-8c21-c00e98a7fbde"
  config {
    ingress_rule {
      service = "http_status:404"
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_config" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4d76-8c21-c00e98a7fbde"
  config = {
    ingress = [
      {
        service = "http_status:404"
      }
    ]
    origin_request = {
      ca_pool                  = ""
      connect_timeout          = 30
      disable_chunked_encoding = false
      keep_alive_timeout       = 90
      no_tls_verify            = false
      origin_server_name       = ""
      proxy_type               = ""
      tcp_keep_alive           = 30
      tls_timeout              = 10
    }
  }
}`,
			},
			{
				Name: "Minimal config - preferred resource name",
				Input: `resource "cloudflare_zero_trust_tunnel_cloudflared_config" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4d76-8c21-c00e98a7fbde"
  config {
    ingress_rule {
      service = "http_status:404"
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_config" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4d76-8c21-c00e98a7fbde"
  config = {
    ingress = [
      {
        service = "http_status:404"
      }
    ]
    origin_request = {
      ca_pool                  = ""
      connect_timeout          = 30
      disable_chunked_encoding = false
      keep_alive_timeout       = 90
      no_tls_verify            = false
      origin_server_name       = ""
      proxy_type               = ""
      tcp_keep_alive           = 30
      tls_timeout              = 10
    }
  }
}`,
			},
			{
				Name: "Remove warp_routing block",
				Input: `resource "cloudflare_tunnel_config" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4d76-8c21-c00e98a7fbde"
  config {
    warp_routing {
      enabled = true
    }
    ingress_rule {
      service = "http_status:404"
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_config" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4d76-8c21-c00e98a7fbde"
  config = {
    ingress = [
      {
        service = "http_status:404"
      }
    ]
    origin_request = {
      ca_pool                  = ""
      connect_timeout          = 30
      disable_chunked_encoding = false
      keep_alive_timeout       = 90
      no_tls_verify            = false
      origin_server_name       = ""
      proxy_type               = ""
      tcp_keep_alive           = 30
      tls_timeout              = 10
    }
  }
}`,
			},
			{
				Name: "Remove ip_rules from config-level origin_request",
				Input: `resource "cloudflare_tunnel_config" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4d76-8c21-c00e98a7fbde"
  config {
    origin_request {
      connect_timeout = "30s"
      ip_rules {
        prefix = "192.0.2.0/24"
        ports  = [80, 443]
        allow  = true
      }
    }
    ingress_rule {
      service = "http_status:404"
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_config" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4d76-8c21-c00e98a7fbde"
  config = {
    ingress = [
      {
        service = "http_status:404"
      }
    ]
    origin_request = {
      ca_pool                  = ""
      connect_timeout          = 30
      disable_chunked_encoding = false
      keep_alive_timeout       = 90
      no_tls_verify            = false
      origin_server_name       = ""
      proxy_type               = ""
      tcp_keep_alive           = 30
      tls_timeout              = 10
    }
  }
}`,
			},
			{
				Name: "Remove ip_rules from ingress-level origin_request",
				Input: `resource "cloudflare_tunnel_config" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4d76-8c21-c00e98a7fbde"
  config {
    ingress_rule {
      hostname = "app.example.com"
      service  = "http://localhost:8080"
      origin_request {
        connect_timeout = "10s"
        ip_rules {
          prefix = "198.51.100.0/24"
          allow  = false
        }
      }
    }
    ingress_rule {
      service = "http_status:404"
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_config" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4d76-8c21-c00e98a7fbde"
  config = {
    ingress = [
      {
        hostname = "app.example.com"
        service  = "http://localhost:8080"
        origin_request = {
          ca_pool                  = ""
          connect_timeout          = 10
          disable_chunked_encoding = false
          keep_alive_timeout       = 90
          no_tls_verify            = false
          origin_server_name       = ""
          proxy_type               = ""
          tcp_keep_alive           = 30
          tls_timeout              = 10
        }
      },
      {
        service = "http_status:404"
      }
    ]
    origin_request = {
      ca_pool                  = ""
      connect_timeout          = 30
      disable_chunked_encoding = false
      keep_alive_timeout       = 90
      no_tls_verify            = false
      origin_server_name       = ""
      proxy_type               = ""
      tcp_keep_alive           = 30
      tls_timeout              = 10
    }
  }
}`,
			},
			{
				Name: "Remove all deprecated blocks - comprehensive",
				Input: `resource "cloudflare_tunnel_config" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4d76-8c21-c00e98a7fbde"
  config {
    warp_routing {
      enabled = true
    }
    origin_request {
      connect_timeout = "30s"
      ip_rules {
        prefix = "192.0.2.0/24"
        ports  = [80, 443]
        allow  = true
      }
    }
    ingress_rule {
      hostname = "app.example.com"
      service  = "http://localhost:8080"
      origin_request {
        tls_timeout = "10s"
        ip_rules {
          prefix = "198.51.100.0/24"
          allow  = false
        }
      }
    }
    ingress_rule {
      service = "http_status:404"
    }
  }
}`,
				Expected: `resource "cloudflare_zero_trust_tunnel_cloudflared_config" "example" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  tunnel_id  = "f70ff02e-f290-4d76-8c21-c00e98a7fbde"
  config = {
    ingress = [
      {
        hostname = "app.example.com"
        service  = "http://localhost:8080"
        origin_request = {
          ca_pool                  = ""
          connect_timeout          = 30
          disable_chunked_encoding = false
          keep_alive_timeout       = 90
          no_tls_verify            = false
          origin_server_name       = ""
          proxy_type               = ""
          tcp_keep_alive           = 30
          tls_timeout              = 10
        }
      },
      {
        service = "http_status:404"
      }
    ]
    origin_request = {
      ca_pool                  = ""
      connect_timeout          = 30
      disable_chunked_encoding = false
      keep_alive_timeout       = 90
      no_tls_verify            = false
      origin_server_name       = ""
      proxy_type               = ""
      tcp_keep_alive           = 30
      tls_timeout              = 10
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
				Name: "Minimal state - deprecated resource name",
				Input: `{
  "type": "cloudflare_tunnel_config",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tunnel_id": "f70ff02e-f290-4d76-8c21-c00e98a7fbde",
    "config": [
      {
        "ingress_rule": [
          {
            "service": "http_status:404"
          }
        ]
      }
    ]
  },
  "schema_version": 0
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_tunnel_cloudflared_config",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tunnel_id": "f70ff02e-f290-4d76-8c21-c00e98a7fbde",
    "config": {
      "ingress": [
        {
          "service": "http_status:404"
        }
      ]
    }
  },
  "schema_version": 0
}`,
			},
			{
				Name: "Minimal state - preferred resource name",
				Input: `{
  "type": "cloudflare_zero_trust_tunnel_cloudflared_config",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tunnel_id": "f70ff02e-f290-4d76-8c21-c00e98a7fbde",
    "config": [
      {
        "ingress_rule": [
          {
            "service": "http_status:404"
          }
        ]
      }
    ]
  },
  "schema_version": 0
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_tunnel_cloudflared_config",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tunnel_id": "f70ff02e-f290-4d76-8c21-c00e98a7fbde",
    "config": {
      "ingress": [
        {
          "service": "http_status:404"
        }
      ]
    }
  },
  "schema_version": 0
}`,
			},
			{
				Name: "Complete transformation with all fields",
				Input: `{
  "type": "cloudflare_tunnel_config",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tunnel_id": "f70ff02e-f290-4d76-8c21-c00e98a7fbde",
    "config": [
      {
        "warp_routing": [
          {
            "enabled": true
          }
        ],
        "origin_request": [
          {
            "connect_timeout": "30s",
            "tls_timeout": "10s",
            "tcp_keep_alive": "1m30s",
            "keep_alive_timeout": "1m30s",
            "keep_alive_connections": 100,
            "bastion_mode": true,
            "proxy_address": "127.0.0.1",
            "proxy_port": 8080,
            "ip_rules": [
              {
                "prefix": "192.0.2.0/24",
                "ports": [80, 443],
                "allow": true
              }
            ],
            "access": [
              {
                "required": true,
                "team_name": "my-team",
                "aud_tag": ["abc123"]
              }
            ]
          }
        ],
        "ingress_rule": [
          {
            "hostname": "app.example.com",
            "service": "http://localhost:8080",
            "origin_request": [
              {
                "connect_timeout": "15s",
                "ip_rules": [
                  {
                    "prefix": "198.51.100.0/24",
                    "allow": false
                  }
                ],
                "access": [
                  {
                    "required": false
                  }
                ]
              }
            ]
          },
          {
            "service": "http_status:404"
          }
        ]
      }
    ]
  },
  "schema_version": 0
}`,
				Expected: `{
  "type": "cloudflare_zero_trust_tunnel_cloudflared_config",
  "name": "example",
  "attributes": {
    "account_id": "f037e56e89293a057740de681ac9abbe",
    "tunnel_id": "f70ff02e-f290-4d76-8c21-c00e98a7fbde",
    "config": {
      "origin_request": {
        "connect_timeout": 30,
        "tls_timeout": 10,
        "tcp_keep_alive": 90,
        "keep_alive_timeout": 90,
        "keep_alive_connections": 100,
        "access": {
          "required": true,
          "team_name": "my-team",
          "aud_tag": ["abc123"]
        }
      },
      "ingress": [
        {
          "hostname": "app.example.com",
          "service": "http://localhost:8080",
          "origin_request": {
            "connect_timeout": 15,
            "access": {
              "required": false
            }
          }
        },
        {
          "service": "http_status:404"
        }
      ]
    }
  },
  "schema_version": 0
}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
