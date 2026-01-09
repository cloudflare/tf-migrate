package zero_trust_split_tunnel

import (
	"testing"

	"github.com/cloudflare/tf-migrate/internal/testhelpers"
)

func TestV4ToV5Transformation(t *testing.T) {
	migrator := NewV4ToV5Migrator()

	t.Run("ConfigTransformation", func(t *testing.T) {
		tests := []testhelpers.ConfigTestCase{
			{
				Name: "no policy id include mode",
				Input: `
resource "cloudflare_split_tunnel" "vpn_only" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "include"

  tunnels {
    address     = "10.0.0.0/8"
    description = "Private network"
    host        = ""
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "vpn_only" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  include = [
    {
      address     = "10.0.0.0/8"
      description = "Private network"
      host        = ""
    }
  ]
}`,
			},
			{
				Name: "no policy id exclude mode",
				Input: `
resource "cloudflare_split_tunnel" "corporate" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "exclude"

  tunnels {
    address     = "192.168.1.0/24"
    description = "Local network"
    host        = ""
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "corporate" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  exclude = [
    {
      address     = "192.168.1.0/24"
      description = "Local network"
      host        = ""
    }
  ]
}`,
			},
			{
				Name: "no policy id include mode multiple tunnels",
				Input: `
resource "cloudflare_split_tunnel" "multi_include" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "include"

  tunnels {
    address     = "10.0.0.0/8"
    description = "Private network"
    host        = ""
  }

  tunnels {
    address     = "172.16.0.0/12"
    description = "Another private range"
    host        = ""
  }

  tunnels {
    address     = ""
    description = "VPN endpoint"
    host        = "vpn.company.com"
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "multi_include" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  include = [
    {
      address     = "10.0.0.0/8"
      description = "Private network"
      host        = ""
    },
    {
      address     = "172.16.0.0/12"
      description = "Another private range"
      host        = ""
    },
    {
      address     = ""
      description = "VPN endpoint"
      host        = "vpn.company.com"
    }
  ]
}`,
			},
			{
				Name: "no policy id exclude mode multiple tunnels",
				Input: `
resource "cloudflare_split_tunnel" "multi_exclude" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "exclude"

  tunnels {
    address     = "192.168.1.0/24"
    description = "Local network"
    host        = ""
  }

  tunnels {
    address     = ""
    description = "Internal site"
    host        = "intranet.company.com"
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "multi_exclude" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  exclude = [
    {
      address     = "192.168.1.0/24"
      description = "Local network"
      host        = ""
    },
    {
      address     = ""
      description = "Internal site"
      host        = "intranet.company.com"
    }
  ]
}`,
			},
			{
				Name: "no policy id include mode alternate resource name",
				Input: `
resource "cloudflare_zero_trust_split_tunnel" "alt_name" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "include"

  tunnels {
    address     = "10.0.0.0/8"
    description = "Private network"
    host        = ""
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "alt_name" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  include = [
    {
      address     = "10.0.0.0/8"
      description = "Private network"
      host        = ""
    }
  ]
}`,
			},
			{
				Name: "policy id null exclude mode",
				Input: `
resource "cloudflare_split_tunnel" "null_policy" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = null
  mode       = "exclude"

  tunnels {
    address     = "192.168.1.0/24"
    description = "Local network"
    host        = ""
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "null_policy" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  exclude = [
    {
      address     = "192.168.1.0/24"
      description = "Local network"
      host        = ""
    }
  ]
}`,
			},
			{
				Name: "policy id empty string include mode",
				Input: `
resource "cloudflare_split_tunnel" "empty_policy" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = ""
  mode       = "include"

  tunnels {
    address     = "10.0.0.0/8"
    description = "Private network"
    host        = ""
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "empty_policy" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  include = [
    {
      address     = "10.0.0.0/8"
      description = "Private network"
      host        = ""
    }
  ]
}`,
			},
			{
				Name: "policy id default exclude mode",
				Input: `
resource "cloudflare_split_tunnel" "default_policy" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = "default"
  mode       = "exclude"

  tunnels {
    address     = "192.168.1.0/24"
    description = "Local network"
    host        = ""
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "default_policy" {
  account_id = "f037e56e89293a057740de681ac9abbe"

  exclude = [
    {
      address     = "192.168.1.0/24"
      description = "Local network"
      host        = ""
    }
  ]
}`,
			},
			{
				Name: "no tunnels include mode",
				Input: `
resource "cloudflare_split_tunnel" "empty_tunnels" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "include"
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "empty_tunnels" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  include    = []
}`,
			},
			{
				Name: "no tunnels exclude mode",
				Input: `
resource "cloudflare_split_tunnel" "empty_tunnels_exclude" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  mode       = "exclude"
}`,
				Expected: `
resource "cloudflare_zero_trust_device_default_profile" "empty_tunnels_exclude" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  exclude    = []
}`,
			},
			{
				Name: "policy id with mode include",
				Input: `
resource "cloudflare_device_settings_policy" "engineering" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Engineering Team"
  match       = "identity.email == \"*@engineering.company.com\""
  precedence  = 10
  description = "Policy for engineering team"
}

resource "cloudflare_split_tunnel" "with_policy" {
  account_id = "f037e56e89293a057740de681ac9abbe"
  policy_id  = cloudflare_device_settings_policy.engineering.id
  mode       = "include"

  tunnels {
    address     = "10.0.0.0/8"
    description = "Private network"
    host        = ""
  }
}`,
				Expected: `
resource "cloudflare_zero_trust_device_custom_profile" "engineering" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  name        = "Engineering Team"
  match       = "identity.email == \"*@engineering.company.com\""
  precedence  = 10
  description = "Policy for engineering team"

  include = [
    {
      address     = "10.0.0.0/8"
      description = "Private network"
      host        = ""
    }
  ]
}`,
			},
		}

		testhelpers.RunConfigTransformTests(t, tests, migrator)
	})

	t.Run("StateTransformation", func(t *testing.T) {
		tests := []testhelpers.StateTestCase{
			{
				Name: "no policy id include mode",
				Input: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_split_tunnel",
					"name": "vpn_only",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"mode": "include",
							"tunnels": [{
								"address": "10.0.0.0/8",
								"description": "Private network",
								"host": ""
							}]
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_zero_trust_device_default_profile",
					"name": "vpn_only",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"include": [{
								"address": "10.0.0.0/8",
								"description": "Private network",
								"host": ""
							}]
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			{
				Name: "no policy id exclude mode",
				Input: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_split_tunnel",
					"name": "corporate",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"mode": "exclude",
							"tunnels": [{
								"address": "192.168.1.0/24",
								"description": "Local network",
								"host": ""
							}]
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_zero_trust_device_default_profile",
					"name": "corporate",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"exclude": [{
								"address": "192.168.1.0/24",
								"description": "Local network",
								"host": ""
							}]
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			{
				Name: "no policy id include mode multiple tunnels",
				Input: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_split_tunnel",
					"name": "multi_include",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"mode": "include",
							"tunnels": [
								{
									"address": "10.0.0.0/8",
									"description": "Private network",
									"host": ""
								},
								{
									"address": "172.16.0.0/12",
									"description": "Another private range",
									"host": ""
								},
								{
									"address": "",
									"description": "VPN endpoint",
									"host": "vpn.company.com"
								}
							]
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_zero_trust_device_default_profile",
					"name": "multi_include",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"include": [
								{
									"address": "10.0.0.0/8",
									"description": "Private network",
									"host": ""
								},
								{
									"address": "172.16.0.0/12",
									"description": "Another private range",
									"host": ""
								},
								{
									"address": "",
									"description": "VPN endpoint",
									"host": "vpn.company.com"
								}
							]
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			{
				Name: "no policy id exclude mode multiple tunnels",
				Input: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_split_tunnel",
					"name": "multi_exclude",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"mode": "exclude",
							"tunnels": [
								{
									"address": "192.168.1.0/24",
									"description": "Local network",
									"host": ""
								},
								{
									"address": "",
									"description": "Internal site",
									"host": "intranet.company.com"
								}
							]
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_zero_trust_device_default_profile",
					"name": "multi_exclude",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"exclude": [
								{
									"address": "192.168.1.0/24",
									"description": "Local network",
									"host": ""
								},
								{
									"address": "",
									"description": "Internal site",
									"host": "intranet.company.com"
								}
							]
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			{
				Name: "no policy id include mode alternate resource name",
				Input: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_zero_trust_split_tunnel",
					"name": "alt_name",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"mode": "include",
							"tunnels": [{
								"address": "10.0.0.0/8",
								"description": "Private network",
								"host": ""
							}]
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_zero_trust_device_default_profile",
					"name": "alt_name",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"include": [{
								"address": "10.0.0.0/8",
								"description": "Private network",
								"host": ""
							}]
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			{
				Name: "policy id null exclude mode",
				Input: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_split_tunnel",
					"name": "null_policy",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"policy_id": null,
							"mode": "exclude",
							"tunnels": [{
								"address": "192.168.1.0/24",
								"description": "Local network",
								"host": ""
							}]
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_zero_trust_device_default_profile",
					"name": "null_policy",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"exclude": [{
								"address": "192.168.1.0/24",
								"description": "Local network",
								"host": ""
							}]
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			{
				Name: "policy id empty string include mode",
				Input: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_split_tunnel",
					"name": "empty_policy",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"policy_id": "",
							"mode": "include",
							"tunnels": [{
								"address": "10.0.0.0/8",
								"description": "Private network",
								"host": ""
							}]
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_zero_trust_device_default_profile",
					"name": "empty_policy",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"include": [{
								"address": "10.0.0.0/8",
								"description": "Private network",
								"host": ""
							}]
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			{
				Name: "policy id default exclude mode",
				Input: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_split_tunnel",
					"name": "default_policy",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"policy_id": "default",
							"mode": "exclude",
							"tunnels": [{
								"address": "192.168.1.0/24",
								"description": "Local network",
								"host": ""
							}]
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_zero_trust_device_default_profile",
					"name": "default_policy",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"exclude": [{
								"address": "192.168.1.0/24",
								"description": "Local network",
								"host": ""
							}]
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			{
				Name: "no tunnels include mode",
				Input: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_split_tunnel",
					"name": "empty_tunnels",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"mode": "include",
							"tunnels": []
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_zero_trust_device_default_profile",
					"name": "empty_tunnels",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"include": []
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
			{
				Name: "no tunnels exclude mode",
				Input: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_split_tunnel",
					"name": "empty_tunnels_exclude",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"mode": "exclude",
							"tunnels": []
						},
						"schema_version": 0
					}]
				}]
			}`,
				Expected: `{
				"version": 4,
				"terraform_version": "1.5.0",
				"resources": [{
					"type": "cloudflare_zero_trust_device_default_profile",
					"name": "empty_tunnels_exclude",
					"instances": [{
						"attributes": {
							"id": "f037e56e89293a057740de681ac9abbe",
							"account_id": "f037e56e89293a057740de681ac9abbe",
							"exclude": []
						},
						"schema_version": 0
					}]
				}]
			}`,
			},
		}

		testhelpers.RunStateTransformTests(t, tests, migrator)
	})
}
