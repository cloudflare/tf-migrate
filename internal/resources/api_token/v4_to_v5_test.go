package api_token

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Migration(t *testing.T) {
	migrator := NewV4ToV5Migrator()
	tests := []testhelpers.ConfigTestCase{
		{
			Name: "basic api token with single policy",
			Input: `
resource "cloudflare_api_token" "example" {
  name = "terraform-example-token"

  policy {
    effect = "allow"
    resources = {
      "com.cloudflare.api.account.*" = "*"
    }
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
      }, {
      id = "82e64a83756745bbbb1c9c2701bf816b"
    }]
  }
}`,
			Expected: `
resource "cloudflare_api_token" "example" {
  name = "terraform-example-token"

  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
      }, {
      id = "82e64a83756745bbbb1c9c2701bf816b"
    }]
  }]
}`,
		},
		{
			Name: "api token with multiple policies",
			Input: `
resource "cloudflare_api_token" "multi_policy" {
		 name = "multi-policy-token"

		 policy {
		   effect = "allow"
		   permission_groups = [
		     "c8fed203ed3043cba015a93ad1616f1f"
		   ]
		   resources = {
		     "com.cloudflare.api.account.*" = "*"
		   }
		 }

		 policy {
		   effect = "allow"
		   permission_groups = [
		     "82e64a83756745bbbb1c9c2701bf816b"
		   ]
		   resources = {
		     "com.cloudflare.api.account.zone.*" = "*"
		   }
		 }
		}`,
			Expected: `
resource "cloudflare_api_token" "multi_policy" {
  name = "multi-policy-token"


  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
    }, {
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.zone.*" = "*"
    })
    permission_groups = [{
      id = "82e64a83756745bbbb1c9c2701bf816b"
    }]
  }]
}`,
		},
		{
			Name: "api token with condition block",
			Input: `
resource "cloudflare_api_token" "with_condition" {
  name = "conditional-token"

  policy {
    effect = "allow"
    resources = {
      "com.cloudflare.api.account.*" = "*"
    }
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
  }

  condition {
    request_ip {
      in = [
        "192.168.1.0/24",
        "10.0.0.0/8"
      ]
    }
  }
}`,
			Expected: `
resource "cloudflare_api_token" "with_condition" {
  name = "conditional-token"


  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
  }]
  condition = {
    request_ip = {
      in = [
        "192.168.1.0/24",
        "10.0.0.0/8"
      ]
    }
  }
}`,
		},
		{
			Name: "api token with condition including not_in",
			Input: `
		resource "cloudflare_api_token" "with_not_in_condition" {
		 name = "restricted-token"

		 policy {
		   effect = "allow"
		   permission_groups = [
		     "c8fed203ed3043cba015a93ad1616f1f"
		   ]
		   resources = {
		     "com.cloudflare.api.account.*" = "*"
		   }
		 }

		 condition {
		   request_ip {
		     in = [
		       "192.168.1.0/24"
		     ]
		     not_in = [
		       "192.168.1.100/32"
		     ]
		   }
		 }
		}`,
			Expected: `
resource "cloudflare_api_token" "with_not_in_condition" {
  name = "restricted-token"


  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
  }]
  condition = {
    request_ip = {
      in = [
        "192.168.1.0/24"
      ]
      not_in = [
        "192.168.1.100/32"
      ]
    }
  }
}`,
		},
		{
			Name: "api token with expires_on and not_before",
			Input: `
		resource "cloudflare_api_token" "time_limited" {
		 name       = "time-limited-token"
		 expires_on = "2025-01-01T00:00:00Z"
		 not_before = "2024-01-01T00:00:00Z"

		 policy {
		   effect = "allow"
		   permission_groups = [
		     "c8fed203ed3043cba015a93ad1616f1f"
		   ]
		   resources = {
		     "com.cloudflare.api.account.*" = "*"
		   }
		 }
		}`,
			Expected: `
resource "cloudflare_api_token" "time_limited" {
  name       = "time-limited-token"
  expires_on = "2025-01-01T00:00:00Z"
  not_before = "2024-01-01T00:00:00Z"

  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
  }]
}`,
		},
		{
			Name: "api token with status field",
			Input: `
		resource "cloudflare_api_token" "with_status" {
		 name   = "status-token"
		 status = "active"

		 policy {
		   effect = "allow"
		   permission_groups = [
		     "c8fed203ed3043cba015a93ad1616f1f"
		   ]
		   resources = {
		     "com.cloudflare.api.account.*" = "*"
		   }
		 }
		}`,
			Expected: `
resource "cloudflare_api_token" "with_status" {
  name   = "status-token"
  status = "active"

  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
  }]
}`,
		},
		{
			Name: "api token with complex resources mapping",
			Input: `
		resource "cloudflare_api_token" "complex_resources" {
		 name = "complex-resources-token"

		 policy {
		   effect = "allow"
		   permission_groups = [
		     "c8fed203ed3043cba015a93ad1616f1f"
		   ]
		   resources = {
		     "com.cloudflare.api.account.*"      = "*"
		     "com.cloudflare.api.account.zone.*" = "*"
		     "com.cloudflare.api.user.*"         = "*"
		   }
		 }
		}`,
			Expected: `
resource "cloudflare_api_token" "complex_resources" {
  name = "complex-resources-token"

  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.*"      = "*"
      "com.cloudflare.api.account.zone.*" = "*"
      "com.cloudflare.api.user.*"         = "*"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
  }]
}`,
		},
		{
			Name: "api token with deny effect",
			Input: `
		resource "cloudflare_api_token" "deny_policy" {
		 name = "deny-policy-token"

		 policy {
		   effect = "allow"
		   permission_groups = [
		     "c8fed203ed3043cba015a93ad1616f1f"
		   ]
		   resources = {
		     "com.cloudflare.api.account.*" = "*"
		   }
		 }

		 policy {
		   effect = "deny"
		   permission_groups = [
		     "82e64a83756745bbbb1c9c2701bf816b"
		   ]
		   resources = {
		     "com.cloudflare.api.account.billing.*" = "*"
		   }
		 }
		}`,
			Expected: `
resource "cloudflare_api_token" "deny_policy" {
  name = "deny-policy-token"


  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
    }]
    }, {
    effect = "deny"
    resources = jsonencode({
      "com.cloudflare.api.account.billing.*" = "*"
    })
    permission_groups = [{
      id = "82e64a83756745bbbb1c9c2701bf816b"
    }]
  }]
}`,
		},
		{
			Name: "api token with empty permission groups",
			Input: `
		resource "cloudflare_api_token" "empty_perms" {
		 name = "empty-perms-token"

		 policy {
		   effect            = "allow"
		   permission_groups = []
		   resources = {
		     "com.cloudflare.api.account.*" = "*"
		   }
		 }
		}`,
			Expected: `
resource "cloudflare_api_token" "empty_perms" {
  name = "empty-perms-token"

  policies = [{
    effect            = "allow"
    permission_groups = []
    resources = jsonencode({
      "com.cloudflare.api.account.*" = "*"
    })
  }]
}`,
		},
		{
			Name: "api token with all optional fields",
			Input: `
		resource "cloudflare_api_token" "full_example" {
		 name       = "full-example-token"
		 status     = "active"
		 expires_on = "2025-12-31T23:59:59Z"
		 not_before = "2024-01-01T00:00:00Z"

		 policy {
		   effect = "allow"
		   permission_groups = [
		     "c8fed203ed3043cba015a93ad1616f1f",
		     "82e64a83756745bbbb1c9c2701bf816b"
		   ]
		   resources = {
		     "com.cloudflare.api.account.*"         = "*"
		     "com.cloudflare.api.account.zone.*"    = "*"
		     "com.cloudflare.api.account.billing.*" = "read"
		   }
		 }

		 policy {
		   effect = "deny"
		   permission_groups = [
		     "f7f0eda5697f475c90846e879bab8666"
		   ]
		   resources = {
		     "com.cloudflare.api.account.billing.*" = "edit"
		   }
		 }

		 condition {
		   request_ip {
		     in = [
		       "192.168.0.0/16",
		       "10.0.0.0/8",
		       "172.16.0.0/12"
		     ]
		     not_in = [
		       "192.168.1.1/32",
		       "10.0.0.1/32"
		     ]
		   }
		 }
		}`,
			Expected: `
resource "cloudflare_api_token" "full_example" {
  name       = "full-example-token"
  status     = "active"
  expires_on = "2025-12-31T23:59:59Z"
  not_before = "2024-01-01T00:00:00Z"



  policies = [{
    effect = "allow"
    resources = jsonencode({
      "com.cloudflare.api.account.*"         = "*"
      "com.cloudflare.api.account.zone.*"    = "*"
      "com.cloudflare.api.account.billing.*" = "read"
    })
    permission_groups = [{
      id = "c8fed203ed3043cba015a93ad1616f1f"
      }, {
      id = "82e64a83756745bbbb1c9c2701bf816b"
    }]
    }, {
    effect = "deny"
    resources = jsonencode({
      "com.cloudflare.api.account.billing.*" = "edit"
    })
    permission_groups = [{
      id = "f7f0eda5697f475c90846e879bab8666"
    }]
  }]
  condition = {
    request_ip = {
      in = [
        "192.168.0.0/16",
        "10.0.0.0/8",
        "172.16.0.0/12"
      ]
      not_in = [
        "192.168.1.1/32",
        "10.0.0.1/32"
      ]
    }
  }
}`,
		},
		{
			Name: "api token with data reference and timestamps",
			Input: `
resource "cloudflare_api_token" "api_token_create" {
  name = "api_token_create"

  policy {
    permission_groups = [
      data.cloudflare_api_token_permission_groups.all.user["API Tokens Write"],
    ]
    resources = {
      "com.cloudflare.api.user.${var.user_id}" = "*"
    }
  }

  not_before = "2018-07-01T05:20:00Z"
  expires_on = "2020-01-01T00:00:00Z"

  condition {
    request_ip {
      in     = ["192.0.2.1/32"]
      not_in = ["198.51.100.1/32"]
    }
  }
}`,
			Expected: `
resource "cloudflare_api_token" "api_token_create" {
  name = "api_token_create"


  not_before = "2018-07-01T05:20:00Z"
  expires_on = "2020-01-01T00:00:00Z"

  policies = [{
    resources = jsonencode({
      "com.cloudflare.api.user.${var.user_id}" = "*"
    })
    effect = "allow"
    permission_groups = [{
      id = "API Tokens Write"
    }]
  }]
  condition = {
    request_ip = {
      in     = ["192.0.2.1/32"]
      not_in = ["198.51.100.1/32"]
    }
  }
}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			testhelpers.RunConfigTransformTests(t, tests, migrator)
		})
	}
}
