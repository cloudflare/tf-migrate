package zero_trust_split_tunnel

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func TestV4ToV5Transformation(t *testing.T) {
	t.Run("ConfigTransformation", func(t *testing.T) {
		t.Run("DefaultDeviceProfileSplitTunnels", testDefaultDeviceProfileSplitTunnelsConfig)
		t.Run("CustomDeviceProfileSplitTunnels", testCustomDeviceProfileSplitTunnelsConfig)
		t.Run("DefaultAndCustomDeviceProfileSplitTunnels", testDefaultAndCustomDeviceProfileSplitTunnelsConfig)
		t.Run("DeviceProfileV4SplitTunnels", testDeviceProfileV4SplitTunnelsConfig)
		t.Run("DeviceProfileNotFoundSplitTunnels", testDeviceProfileNotFoundSplitTunnelsConfig)
	})

}

func testDefaultDeviceProfileSplitTunnelsConfig(t *testing.T) {
	t.Run("Single exclude tunnel merged into default profile", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_default_profile" "default" {
  account_id = "abc123"
}

resource "cloudflare_split_tunnel" "exclude_tunnel" {
  account_id = "abc123"
  mode       = "exclude"
  tunnels {
    address     = "192.168.1.0/24"
    description = "Local network"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify the default profile still exists
		if !containsString(result, `resource "cloudflare_zero_trust_device_default_profile" "default"`) {
			t.Error("Expected cloudflare_zero_trust_device_default_profile resource to still exist")
		}

		// Verify exclude attribute is added
		if !containsString(result, "exclude = [") {
			t.Error("Expected exclude attribute to be added to default profile")
		}

		// Verify include attribute is NOT present (only exclude mode)
		if containsString(result, "include = [") {
			t.Error("Did not expect include attribute when only exclude tunnel is present")
		}

		// Verify tunnel data is present
		if !containsString(result, `"192.168.1.0/24"`) {
			t.Error("Expected tunnel address in exclude array")
		}

		if !containsString(result, `"Local network"`) {
			t.Error("Expected tunnel description in exclude array")
		}
	})

	t.Run("Single include tunnel merged into default profile", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_default_profile" "default" {
  account_id = "abc123"
}

resource "cloudflare_split_tunnel" "include_tunnel" {
  account_id = "abc123"
  mode       = "include"
  tunnels {
    address     = "10.0.0.0/8"
    description = "Corporate network"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify include attribute is added
		if !containsString(result, "include = [") {
			t.Error("Expected include attribute to be added to default profile")
		}

		// Verify exclude attribute is NOT present (only include mode)
		if containsString(result, "exclude = [") {
			t.Error("Did not expect exclude attribute when only include tunnel is present")
		}

		// Verify tunnel data is present
		if !containsString(result, `"10.0.0.0/8"`) {
			t.Error("Expected tunnel address in include array")
		}

		if !containsString(result, `"Corporate network"`) {
			t.Error("Expected tunnel description in include array")
		}
	})

	t.Run("Multiple exclude tunnels merged into default profile", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_default_profile" "default" {
  account_id = "abc123"
}

resource "cloudflare_split_tunnel" "exclude_tunnel1" {
  account_id = "abc123"
  mode       = "exclude"
  tunnels {
    address     = "192.168.0.0/16"
    description = "Private network"
  }
}

resource "cloudflare_split_tunnel" "exclude_tunnel2" {
  account_id = "abc123"
  mode       = "exclude"
  tunnels {
    address = "10.0.0.0/8"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify exclude attribute is added
		if !containsString(result, "exclude = [") {
			t.Error("Expected exclude attribute to be added to default profile")
		}

		// Verify include attribute is NOT present (only exclude mode)
		if containsString(result, "include = [") {
			t.Error("Did not expect include attribute when only exclude tunnels are present")
		}

		// Verify both tunnels are present
		if !containsString(result, `"192.168.0.0/16"`) {
			t.Error("Expected first tunnel address in exclude array")
		}

		if !containsString(result, `"10.0.0.0/8"`) {
			t.Error("Expected second tunnel address in exclude array")
		}
	})

	t.Run("Multiple include tunnels merged into default profile", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_default_profile" "default" {
  account_id = "abc123"
}

resource "cloudflare_split_tunnel" "include_tunnel1" {
  account_id = "abc123"
  mode       = "include"
  tunnels {
    address     = "10.0.0.0/8"
    description = "Corp network"
  }
}

resource "cloudflare_split_tunnel" "include_tunnel2" {
  account_id = "abc123"
  mode       = "include"
  tunnels {
    address = "172.16.0.0/12"
    host    = "internal.example.com"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify include attribute is added
		if !containsString(result, "include = [") {
			t.Error("Expected include attribute to be added to default profile")
		}

		// Verify exclude attribute is NOT present (only include mode)
		if containsString(result, "exclude = [") {
			t.Error("Did not expect exclude attribute when only include tunnels are present")
		}

		// Verify both tunnels are present
		if !containsString(result, `"10.0.0.0/8"`) {
			t.Error("Expected first tunnel address in include array")
		}

		if !containsString(result, `"172.16.0.0/12"`) {
			t.Error("Expected second tunnel address in include array")
		}

		if !containsString(result, `"internal.example.com"`) {
			t.Error("Expected host attribute in second tunnel")
		}
	})

	t.Run("Both exclude and include tunnels merged into default profile", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_default_profile" "default" {
  account_id = "abc123"
}

resource "cloudflare_split_tunnel" "exclude_tunnel" {
  account_id = "abc123"
  mode       = "exclude"
  tunnels {
    address = "192.168.0.0/16"
  }
}

resource "cloudflare_split_tunnel" "include_tunnel" {
  account_id = "abc123"
  mode       = "include"
  tunnels {
    address = "10.0.0.0/8"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify both exclude and include attributes are added
		if !containsString(result, "exclude = [") {
			t.Error("Expected exclude attribute to be added to default profile")
		}

		if !containsString(result, "include = [") {
			t.Error("Expected include attribute to be added to default profile")
		}

		// Verify both tunnel addresses are present
		if !containsString(result, `"192.168.0.0/16"`) {
			t.Error("Expected exclude tunnel address")
		}

		if !containsString(result, `"10.0.0.0/8"`) {
			t.Error("Expected include tunnel address")
		}
	})

	t.Run("Single split_tunnel resource with multiple tunnels blocks", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_default_profile" "default" {
  account_id = "abc123"
}

resource "cloudflare_split_tunnel" "multi_tunnels" {
  account_id = "abc123"
  mode       = "exclude"
  tunnels {
    address     = "10.0.0.0/8"
    description = "Network 1"
  }
  tunnels {
    address     = "192.168.0.0/16"
    description = "Network 2"
  }
  tunnels {
    address = "172.16.0.0/12"
    host    = "internal.example.com"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify exclude attribute is added
		if !containsString(result, "exclude = [") {
			t.Error("Expected exclude attribute to be added to default profile")
		}

		// Verify all three tunnel addresses are present
		if !containsString(result, `"10.0.0.0/8"`) {
			t.Error("Expected first tunnel address")
		}

		if !containsString(result, `"192.168.0.0/16"`) {
			t.Error("Expected second tunnel address")
		}

		if !containsString(result, `"172.16.0.0/12"`) {
			t.Error("Expected third tunnel address")
		}

		// Verify descriptions are present
		if !containsString(result, `"Network 1"`) {
			t.Error("Expected first tunnel description")
		}

		if !containsString(result, `"Network 2"`) {
			t.Error("Expected second tunnel description")
		}

		// Verify host attribute is present
		if !containsString(result, `"internal.example.com"`) {
			t.Error("Expected third tunnel host attribute")
		}
	})

	t.Run("Split tunnel with empty tunnels blocks are skipped", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_default_profile" "default" {
  account_id = "abc123"
}

resource "cloudflare_split_tunnel" "with_empty" {
  account_id = "abc123"
  mode       = "exclude"
  tunnels {
    # Empty block - should be skipped
  }
  tunnels {
    address = "10.0.0.0/8"
    description = "Valid tunnel"
  }
  tunnels {
    # Another empty block
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify exclude attribute is added
		if !containsString(result, "exclude = [") {
			t.Error("Expected exclude attribute to be added to default profile")
		}

		// Verify only the valid tunnel is present
		if !containsString(result, `"10.0.0.0/8"`) {
			t.Error("Expected valid tunnel address")
		}

		if !containsString(result, `"Valid tunnel"`) {
			t.Error("Expected valid tunnel description")
		}
	})

	t.Run("Split tunnel without mode specified defaults to exclude", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_default_profile" "default" {
  account_id = "abc123"
}

resource "cloudflare_split_tunnel" "no_mode" {
  account_id = "abc123"
  # No mode specified - should default to "exclude"
  tunnels {
    address     = "192.168.100.0/24"
    description = "Default mode test"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify exclude attribute is added (default mode)
		if !containsString(result, "exclude = [") {
			t.Error("Expected exclude attribute to be added to default profile (default mode is exclude)")
		}

		// Verify include attribute is NOT present
		if containsString(result, "include = [") {
			t.Error("Did not expect include attribute when mode defaults to exclude")
		}

		// Verify tunnel data is present
		if !containsString(result, `"192.168.100.0/24"`) {
			t.Error("Expected tunnel address in exclude array")
		}

		if !containsString(result, `"Default mode test"`) {
			t.Error("Expected tunnel description in exclude array")
		}
	})

	t.Run("Migration is idempotent - running twice produces same result", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_default_profile" "default" {
  account_id = "abc123"
}

resource "cloudflare_zero_trust_device_custom_profile" "employees" {
  account_id = "abc123"
  name       = "Employees"
  match      = "identity.email endsWith \"@example.com\""
  precedence = 100
}

resource "cloudflare_split_tunnel" "default_tunnel" {
  account_id = "abc123"
  mode       = "exclude"
  tunnels {
    address = "10.0.0.0/8"
  }
}

resource "cloudflare_split_tunnel" "employee_tunnel" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.employees.id
  mode       = "include"
  tunnels {
    address = "192.168.0.0/16"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		// Run migration first time
		ProcessCrossResourceConfigMigration(file)
		result1 := string(file.Bytes())

		// Parse the result and run migration again
		file2, diags2 := hclwrite.ParseConfig([]byte(result1), "test.tf", hcl.InitialPos)
		if diags2.HasErrors() {
			t.Fatalf("Failed to parse first migration result: %v", diags2)
		}

		ProcessCrossResourceConfigMigration(file2)
		result2 := string(file2.Bytes())

		// Results should be identical (idempotent)
		if result1 != result2 {
			t.Errorf("Migration is not idempotent. First run differs from second run.\nFirst:\n%s\n\nSecond:\n%s", result1, result2)
		}

		// Verify no split_tunnel resources in final result
		if containsString(result2, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify both profiles have their tunnels
		if !containsString(result2, "exclude = [") {
			t.Error("Expected exclude attribute in default profile")
		}

		if !containsString(result2, "include = [") {
			t.Error("Expected include attribute in custom profile")
		}
	})

	t.Run("Tunnel with only optional fields (no address) is skipped", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_default_profile" "default" {
  account_id = "abc123"
}

resource "cloudflare_split_tunnel" "missing_address" {
  account_id = "abc123"
  mode       = "exclude"
  tunnels {
    # No address field - should be skipped
    description = "Invalid - no address"
    host        = "example.com"
  }
  tunnels {
    address     = "10.0.0.0/8"
    description = "Valid tunnel"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify exclude attribute is added
		if !containsString(result, "exclude = [") {
			t.Error("Expected exclude attribute to be added to default profile")
		}

		// Verify only the valid tunnel (with address) is present
		if !containsString(result, `"10.0.0.0/8"`) {
			t.Error("Expected valid tunnel address")
		}

		if !containsString(result, `"Valid tunnel"`) {
			t.Error("Expected valid tunnel description")
		}

		// The invalid tunnel without address should NOT be present
		// (We can't easily verify absence of "Invalid - no address" since comments might exist)
	})

	t.Run("Mixed V4 and V5 resource types in same file", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_default_profile" "default" {
  account_id = "abc123"
}

resource "cloudflare_zero_trust_device_profiles" "v4_custom" {
  account_id = "abc123"
  name       = "V4 Custom"
  match      = "identity.email endsWith \"@v4.com\""
  precedence = 100
}

resource "cloudflare_zero_trust_device_custom_profile" "v5_custom" {
  account_id = "abc123"
  name       = "V5 Custom"
  match      = "identity.email endsWith \"@v5.com\""
  precedence = 200
}

resource "cloudflare_split_tunnel" "default_tunnel" {
  account_id = "abc123"
  mode       = "exclude"
  tunnels {
    address = "10.0.0.0/8"
    description = "Default profile tunnel"
  }
}

resource "cloudflare_split_tunnel" "v4_tunnel" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_profiles.v4_custom.id
  mode       = "include"
  tunnels {
    address = "192.168.0.0/16"
    description = "V4 custom profile tunnel"
  }
}

resource "cloudflare_split_tunnel" "v5_tunnel" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.v5_custom.id
  mode       = "include"
  tunnels {
    address = "172.16.0.0/12"
    description = "V5 custom profile tunnel"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify all profile types still exist
		if !containsString(result, `resource "cloudflare_zero_trust_device_default_profile" "default"`) {
			t.Error("Expected V5 default profile to still exist")
		}
		if !containsString(result, `resource "cloudflare_zero_trust_device_profiles" "v4_custom"`) {
			t.Error("Expected V4 custom profile to still exist")
		}
		if !containsString(result, `resource "cloudflare_zero_trust_device_custom_profile" "v5_custom"`) {
			t.Error("Expected V5 custom profile to still exist")
		}

		// Find all profile blocks to verify correct tunnel merging
		defaultBlock := findResourceBlock(file.Body(), "cloudflare_zero_trust_device_default_profile", "default")
		v4CustomBlock := findResourceBlock(file.Body(), "cloudflare_zero_trust_device_profiles", "v4_custom")
		v5CustomBlock := findResourceBlock(file.Body(), "cloudflare_zero_trust_device_custom_profile", "v5_custom")

		if defaultBlock == nil {
			t.Fatal("Could not find default profile block")
		}
		if v4CustomBlock == nil {
			t.Fatal("Could not find V4 custom profile block")
		}
		if v5CustomBlock == nil {
			t.Fatal("Could not find V5 custom profile block")
		}

		// Default profile should have exclude
		if defaultBlock.Body().GetAttribute("exclude") == nil {
			t.Error("Expected default profile to have exclude attribute")
		}

		// V4 custom profile should have include
		if v4CustomBlock.Body().GetAttribute("include") == nil {
			t.Error("Expected V4 custom profile to have include attribute")
		}

		// V5 custom profile should have include
		if v5CustomBlock.Body().GetAttribute("include") == nil {
			t.Error("Expected V5 custom profile to have include attribute")
		}

		// Verify all tunnel addresses are present
		if !containsString(result, `"10.0.0.0/8"`) {
			t.Error("Expected default profile tunnel address")
		}
		if !containsString(result, `"192.168.0.0/16"`) {
			t.Error("Expected V4 custom profile tunnel address")
		}
		if !containsString(result, `"172.16.0.0/12"`) {
			t.Error("Expected V5 custom profile tunnel address")
		}
	})
}

func testCustomDeviceProfileSplitTunnelsConfig(t *testing.T) {
	t.Run("Single exclude tunnel merged into custom profile", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_custom_profile" "contractors" {
  account_id = "abc123"
  name       = "Contractors"
  match      = "identity.email == \"contractor@example.com\""
  precedence = 100
}

resource "cloudflare_split_tunnel" "exclude_tunnel" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.contractors.id
  mode       = "exclude"
  tunnels {
    address     = "10.0.0.0/8"
    description = "Internal network"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify the custom profile still exists
		if !containsString(result, `resource "cloudflare_zero_trust_device_custom_profile" "contractors"`) {
			t.Error("Expected cloudflare_zero_trust_device_custom_profile resource to still exist")
		}

		// Verify exclude attribute is added
		if !containsString(result, "exclude = [") {
			t.Error("Expected exclude attribute to be added to custom profile")
		}

		// Verify include attribute is NOT present (only exclude mode)
		if containsString(result, "include = [") {
			t.Error("Did not expect include attribute when only exclude tunnel is present")
		}

		// Verify tunnel data is present
		if !containsString(result, `"10.0.0.0/8"`) {
			t.Error("Expected tunnel address in exclude array")
		}

		if !containsString(result, `"Internal network"`) {
			t.Error("Expected tunnel description in exclude array")
		}
	})

	t.Run("Single include tunnel merged into custom profile", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_custom_profile" "employees" {
  account_id = "abc123"
  name       = "Employees"
  match      = "identity.groups == \"employees\""
  precedence = 200
}

resource "cloudflare_split_tunnel" "include_tunnel" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.employees.id
  mode       = "include"
  tunnels {
    address     = "172.16.0.0/12"
    description = "Corporate resources"
    host        = "corporate.example.com"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify the custom profile still exists
		if !containsString(result, `resource "cloudflare_zero_trust_device_custom_profile" "employees"`) {
			t.Error("Expected cloudflare_zero_trust_device_custom_profile resource to still exist")
		}

		// Verify include attribute is added
		if !containsString(result, "include = [") {
			t.Error("Expected include attribute to be added to custom profile")
		}

		// Verify exclude attribute is NOT present (only include mode)
		if containsString(result, "exclude = [") {
			t.Error("Did not expect exclude attribute when only include tunnel is present")
		}

		// Verify tunnel data is present
		if !containsString(result, `"172.16.0.0/12"`) {
			t.Error("Expected tunnel address in include array")
		}

		if !containsString(result, `"Corporate resources"`) {
			t.Error("Expected tunnel description in include array")
		}

		if !containsString(result, `"corporate.example.com"`) {
			t.Error("Expected host attribute in include array")
		}
	})

	t.Run("Both exclude and include tunnels merged into custom profile", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_custom_profile" "mixed" {
  account_id = "abc123"
  name       = "Mixed Policy"
  match      = "identity.email endsWith \"@example.com\""
  precedence = 150
}

resource "cloudflare_split_tunnel" "exclude_tunnel" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.mixed.id
  mode       = "exclude"
  tunnels {
    address = "192.168.0.0/16"
  }
}

resource "cloudflare_split_tunnel" "include_tunnel" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.mixed.id
  mode       = "include"
  tunnels {
    address = "10.0.0.0/8"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify the custom profile still exists
		if !containsString(result, `resource "cloudflare_zero_trust_device_custom_profile" "mixed"`) {
			t.Error("Expected cloudflare_zero_trust_device_custom_profile resource to still exist")
		}

		// Verify both exclude and include attributes are added
		if !containsString(result, "exclude = [") {
			t.Error("Expected exclude attribute to be added to custom profile")
		}

		if !containsString(result, "include = [") {
			t.Error("Expected include attribute to be added to custom profile")
		}

		// Verify both tunnel addresses are present
		if !containsString(result, `"192.168.0.0/16"`) {
			t.Error("Expected exclude tunnel address")
		}

		if !containsString(result, `"10.0.0.0/8"`) {
			t.Error("Expected include tunnel address")
		}
	})

	t.Run("Multiple custom profiles with single tunnel mapped to correct profile", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_custom_profile" "contractors" {
  account_id = "abc123"
  name       = "Contractors"
  match      = "identity.email == \"contractor@example.com\""
  precedence = 100
}

resource "cloudflare_zero_trust_device_custom_profile" "employees" {
  account_id = "abc123"
  name       = "Employees"
  match      = "identity.groups == \"employees\""
  precedence = 200
}

resource "cloudflare_split_tunnel" "exclude_tunnel" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.employees.id
  mode       = "exclude"
  tunnels {
    address     = "10.0.0.0/8"
    description = "Employee exclusion"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify all custom profiles still exist
		if !containsString(result, `resource "cloudflare_zero_trust_device_custom_profile" "contractors"`) {
			t.Error("Expected contractors profile to still exist")
		}
		if !containsString(result, `resource "cloudflare_zero_trust_device_custom_profile" "employees"`) {
			t.Error("Expected employees profile to still exist")
		}

		// Verify tunnel is only in employees profile by checking HCL structure
		employeesBlock := findResourceBlock(file.Body(), "cloudflare_zero_trust_device_custom_profile", "employees")
		contractorsBlock := findResourceBlock(file.Body(), "cloudflare_zero_trust_device_custom_profile", "contractors")

		if employeesBlock == nil {
			t.Fatal("Could not find employees profile block")
		}
		if contractorsBlock == nil {
			t.Fatal("Could not find contractors profile block")
		}

		// Verify only employees profile has exclude attribute
		if employeesBlock.Body().GetAttribute("exclude") == nil {
			t.Error("Expected employees profile to have exclude attribute")
		}
		if contractorsBlock.Body().GetAttribute("exclude") != nil {
			t.Error("Did not expect contractors profile to have exclude attribute")
		}

		// Verify tunnel data is present
		if !containsString(result, `"10.0.0.0/8"`) {
			t.Error("Expected tunnel address to be present")
		}
		if !containsString(result, `"Employee exclusion"`) {
			t.Error("Expected tunnel description to be present")
		}
	})

	t.Run("Two profiles with two tunnels each mapping to different profile", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_custom_profile" "contractors" {
  account_id = "abc123"
  name       = "Contractors"
  match      = "identity.email == \"contractor@example.com\""
  precedence = 100
}

resource "cloudflare_zero_trust_device_custom_profile" "employees" {
  account_id = "abc123"
  name       = "Employees"
  match      = "identity.groups == \"employees\""
  precedence = 200
}

resource "cloudflare_split_tunnel" "exclude_tunnel" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.employees.id
  mode       = "exclude"
  tunnels {
    address     = "10.0.0.0/8"
    description = "Employee exclusion"
  }
}

resource "cloudflare_split_tunnel" "include_tunnel" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.contractors.id
  mode       = "include"
  tunnels {
    address     = "192.168.0.0/16"
    description = "Contractor inclusion"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify both profiles still exist
		if !containsString(result, `resource "cloudflare_zero_trust_device_custom_profile" "contractors"`) {
			t.Error("Expected contractors profile to still exist")
		}
		if !containsString(result, `resource "cloudflare_zero_trust_device_custom_profile" "employees"`) {
			t.Error("Expected employees profile to still exist")
		}

		// Verify tunnels are in correct profiles using HCL structure
		employeesBlock := findResourceBlock(file.Body(), "cloudflare_zero_trust_device_custom_profile", "employees")
		contractorsBlock := findResourceBlock(file.Body(), "cloudflare_zero_trust_device_custom_profile", "contractors")

		if employeesBlock == nil {
			t.Fatal("Could not find employees profile block")
		}
		if contractorsBlock == nil {
			t.Fatal("Could not find contractors profile block")
		}

		// Employees should have exclude, not include
		if employeesBlock.Body().GetAttribute("exclude") == nil {
			t.Error("Expected employees profile to have exclude attribute")
		}
		if employeesBlock.Body().GetAttribute("include") != nil {
			t.Error("Did not expect employees profile to have include attribute")
		}

		// Contractors should have include, not exclude
		if contractorsBlock.Body().GetAttribute("include") == nil {
			t.Error("Expected contractors profile to have include attribute")
		}
		if contractorsBlock.Body().GetAttribute("exclude") != nil {
			t.Error("Did not expect contractors profile to have exclude attribute")
		}

		// Verify tunnel data is present
		if !containsString(result, `"10.0.0.0/8"`) {
			t.Error("Expected employee tunnel address to be present")
		}
		if !containsString(result, `"192.168.0.0/16"`) {
			t.Error("Expected contractor tunnel address to be present")
		}
	})

	t.Run("Two profiles with both tunnels mapping to same profile", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_custom_profile" "contractors" {
  account_id = "abc123"
  name       = "Contractors"
  match      = "identity.email == \"contractor@example.com\""
  precedence = 100
}

resource "cloudflare_zero_trust_device_custom_profile" "employees" {
  account_id = "abc123"
  name       = "Employees"
  match      = "identity.groups == \"employees\""
  precedence = 200
}

resource "cloudflare_split_tunnel" "exclude_tunnel1" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.employees.id
  mode       = "exclude"
  tunnels {
    address     = "10.0.0.0/8"
    description = "Employee exclusion 1"
  }
}

resource "cloudflare_split_tunnel" "exclude_tunnel2" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.employees.id
  mode       = "exclude"
  tunnels {
    address     = "172.16.0.0/12"
    description = "Employee exclusion 2"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify tunnels are in correct profiles using HCL structure
		employeesBlock := findResourceBlock(file.Body(), "cloudflare_zero_trust_device_custom_profile", "employees")
		contractorsBlock := findResourceBlock(file.Body(), "cloudflare_zero_trust_device_custom_profile", "contractors")

		if employeesBlock == nil {
			t.Fatal("Could not find employees profile block")
		}
		if contractorsBlock == nil {
			t.Fatal("Could not find contractors profile block")
		}

		// Employees should have exclude with both tunnels
		if employeesBlock.Body().GetAttribute("exclude") == nil {
			t.Error("Expected employees profile to have exclude attribute")
		}

		// Contractors should have no tunnels
		if contractorsBlock.Body().GetAttribute("exclude") != nil {
			t.Error("Did not expect contractors profile to have exclude attribute")
		}
		if contractorsBlock.Body().GetAttribute("include") != nil {
			t.Error("Did not expect contractors profile to have include attribute")
		}

		// Verify both tunnel addresses are present in employees profile
		if !containsString(result, `"10.0.0.0/8"`) {
			t.Error("Expected first employee tunnel address to be present")
		}
		if !containsString(result, `"172.16.0.0/12"`) {
			t.Error("Expected second employee tunnel address to be present")
		}
	})

	t.Run("Two profiles with three tunnels in mixed mapping", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_custom_profile" "contractors" {
  account_id = "abc123"
  name       = "Contractors"
  match      = "identity.email == \"contractor@example.com\""
  precedence = 100
}

resource "cloudflare_zero_trust_device_custom_profile" "employees" {
  account_id = "abc123"
  name       = "Employees"
  match      = "identity.groups == \"employees\""
  precedence = 200
}

resource "cloudflare_split_tunnel" "exclude_tunnel1" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.employees.id
  mode       = "exclude"
  tunnels {
    address     = "10.0.0.0/8"
    description = "Employee exclusion 1"
  }
}

resource "cloudflare_split_tunnel" "exclude_tunnel2" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.employees.id
  mode       = "exclude"
  tunnels {
    address     = "172.16.0.0/12"
    description = "Employee exclusion 2"
  }
}

resource "cloudflare_split_tunnel" "include_tunnel" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.contractors.id
  mode       = "include"
  tunnels {
    address     = "192.168.0.0/16"
    description = "Contractor inclusion"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify tunnels are in correct profiles using HCL structure
		employeesBlock := findResourceBlock(file.Body(), "cloudflare_zero_trust_device_custom_profile", "employees")
		contractorsBlock := findResourceBlock(file.Body(), "cloudflare_zero_trust_device_custom_profile", "contractors")

		if employeesBlock == nil {
			t.Fatal("Could not find employees profile block")
		}
		if contractorsBlock == nil {
			t.Fatal("Could not find contractors profile block")
		}

		// Employees should have exclude (with 2 tunnels)
		if employeesBlock.Body().GetAttribute("exclude") == nil {
			t.Error("Expected employees profile to have exclude attribute")
		}
		if employeesBlock.Body().GetAttribute("include") != nil {
			t.Error("Did not expect employees profile to have include attribute")
		}

		// Contractors should have include (with 1 tunnel)
		if contractorsBlock.Body().GetAttribute("include") == nil {
			t.Error("Expected contractors profile to have include attribute")
		}
		if contractorsBlock.Body().GetAttribute("exclude") != nil {
			t.Error("Did not expect contractors profile to have exclude attribute")
		}

		// Verify all tunnel addresses are present
		if !containsString(result, `"10.0.0.0/8"`) {
			t.Error("Expected first employee tunnel address to be present")
		}
		if !containsString(result, `"172.16.0.0/12"`) {
			t.Error("Expected second employee tunnel address to be present")
		}
		if !containsString(result, `"192.168.0.0/16"`) {
			t.Error("Expected contractor tunnel address to be present")
		}
	})

	t.Run("Split tunnel with bracket notation policy_id reference", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_custom_profile" "contractors" {
  account_id = "abc123"
  name       = "Contractors"
  match      = "identity.email == \"contractor@example.com\""
  precedence = 100
}

resource "cloudflare_split_tunnel" "bracket_ref" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile["contractors"].id
  mode       = "include"
  tunnels {
    address     = "10.50.0.0/16"
    description = "Bracket notation test"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify the custom profile still exists
		if !containsString(result, `resource "cloudflare_zero_trust_device_custom_profile" "contractors"`) {
			t.Error("Expected cloudflare_zero_trust_device_custom_profile resource to still exist")
		}

		// Verify include attribute is added
		if !containsString(result, "include = [") {
			t.Error("Expected include attribute to be added to custom profile")
		}

		// Verify tunnel data is present
		if !containsString(result, `"10.50.0.0/16"`) {
			t.Error("Expected tunnel address in include array")
		}

		if !containsString(result, `"Bracket notation test"`) {
			t.Error("Expected tunnel description in include array")
		}
	})

	t.Run("Split tunnel with deprecated cloudflare_device_settings_policy reference", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_custom_profile" "legacy" {
  account_id = "abc123"
  name       = "Legacy Policy"
  match      = "identity.email == \"legacy@example.com\""
  precedence = 150
}

resource "cloudflare_split_tunnel" "legacy_ref" {
  account_id = "abc123"
  policy_id  = cloudflare_device_settings_policy.legacy.id
  mode       = "exclude"
  tunnels {
    address     = "10.250.0.0/16"
    description = "Legacy reference test"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify the custom profile still exists
		if !containsString(result, `resource "cloudflare_zero_trust_device_custom_profile" "legacy"`) {
			t.Error("Expected cloudflare_zero_trust_device_custom_profile resource to still exist")
		}

		// Verify exclude attribute is added (tunnel correctly merged despite deprecated reference)
		if !containsString(result, "exclude = [") {
			t.Error("Expected exclude attribute to be added to custom profile")
		}

		// Verify tunnel data is present
		if !containsString(result, `"10.250.0.0/16"`) {
			t.Error("Expected tunnel address in exclude array")
		}

		if !containsString(result, `"Legacy reference test"`) {
			t.Error("Expected tunnel description in exclude array")
		}
	})
}

func testDefaultAndCustomDeviceProfileSplitTunnelsConfig(t *testing.T) {
	t.Run("Default profile and two custom profiles with multiple tunnels mapped correctly", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_default_profile" "default" {
  account_id = "abc123"
}

resource "cloudflare_zero_trust_device_custom_profile" "contractors" {
  account_id = "abc123"
  name       = "Contractors"
  match      = "identity.email == \"contractor@example.com\""
  precedence = 100
}

resource "cloudflare_zero_trust_device_custom_profile" "employees" {
  account_id = "abc123"
  name       = "Employees"
  match      = "identity.groups == \"employees\""
  precedence = 200
}

resource "cloudflare_split_tunnel" "default_exclude" {
  account_id = "abc123"
  mode       = "exclude"
  tunnels {
    address     = "10.0.0.0/8"
    description = "Default exclusion"
  }
}

resource "cloudflare_split_tunnel" "employee_exclude" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.employees.id
  mode       = "exclude"
  tunnels {
    address     = "172.16.0.0/12"
    description = "Employee exclusion"
  }
}

resource "cloudflare_split_tunnel" "employee_include" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.employees.id
  mode       = "include"
  tunnels {
    address     = "192.168.1.0/24"
    description = "Employee inclusion"
  }
}

resource "cloudflare_split_tunnel" "contractor_include" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.contractors.id
  mode       = "include"
  tunnels {
    address     = "192.168.100.0/24"
    description = "Contractor inclusion"
  }
}

resource "cloudflare_split_tunnel" "default_include" {
  account_id = "abc123"
  mode       = "include"
  tunnels {
    address     = "203.0.113.0/24"
    description = "Default inclusion"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify all profiles still exist
		if !containsString(result, `resource "cloudflare_zero_trust_device_default_profile" "default"`) {
			t.Error("Expected default profile to still exist")
		}
		if !containsString(result, `resource "cloudflare_zero_trust_device_custom_profile" "contractors"`) {
			t.Error("Expected contractors profile to still exist")
		}
		if !containsString(result, `resource "cloudflare_zero_trust_device_custom_profile" "employees"`) {
			t.Error("Expected employees profile to still exist")
		}

		// Find all profile blocks using HCL structure
		defaultBlock := findResourceBlock(file.Body(), "cloudflare_zero_trust_device_default_profile", "default")
		employeesBlock := findResourceBlock(file.Body(), "cloudflare_zero_trust_device_custom_profile", "employees")
		contractorsBlock := findResourceBlock(file.Body(), "cloudflare_zero_trust_device_custom_profile", "contractors")

		if defaultBlock == nil {
			t.Fatal("Could not find default profile block")
		}
		if employeesBlock == nil {
			t.Fatal("Could not find employees profile block")
		}
		if contractorsBlock == nil {
			t.Fatal("Could not find contractors profile block")
		}

		// Default profile should have both exclude and include
		if defaultBlock.Body().GetAttribute("exclude") == nil {
			t.Error("Expected default profile to have exclude attribute")
		}
		if defaultBlock.Body().GetAttribute("include") == nil {
			t.Error("Expected default profile to have include attribute")
		}

		// Employees profile should have both exclude and include
		if employeesBlock.Body().GetAttribute("exclude") == nil {
			t.Error("Expected employees profile to have exclude attribute")
		}
		if employeesBlock.Body().GetAttribute("include") == nil {
			t.Error("Expected employees profile to have include attribute")
		}

		// Contractors profile should have only include
		if contractorsBlock.Body().GetAttribute("include") == nil {
			t.Error("Expected contractors profile to have include attribute")
		}
		if contractorsBlock.Body().GetAttribute("exclude") != nil {
			t.Error("Did not expect contractors profile to have exclude attribute")
		}

		// Verify all tunnel addresses are present in the result
		expectedAddresses := []string{
			"10.0.0.0/8",       // default exclude
			"203.0.113.0/24",   // default include
			"172.16.0.0/12",    // employee exclude
			"192.168.1.0/24",   // employee include
			"192.168.100.0/24", // contractor include
		}
		for _, addr := range expectedAddresses {
			if !containsString(result, fmt.Sprintf(`"%s"`, addr)) {
				t.Errorf("Expected tunnel address %s to be present", addr)
			}
		}
	})
}

func testDeviceProfileV4SplitTunnelsConfig(t *testing.T) {
	t.Run("V4 cloudflare_zero_trust_device_profiles custom profile with split tunnel", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_profiles" "employees" {
  account_id = "abc123"
  name       = "Employees"
  match      = "identity.groups == \"employees\""
  precedence = 200
}

resource "cloudflare_split_tunnel" "employee_tunnel" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_profiles.employees.id
  mode       = "include"
  tunnels {
    address     = "10.100.0.0/16"
    description = "Employee resources"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify the v4 profile still exists (not renamed by this migration)
		if !containsString(result, `resource "cloudflare_zero_trust_device_profiles" "employees"`) {
			t.Error("Expected cloudflare_zero_trust_device_profiles resource to still exist")
		}

		// Verify include attribute is added
		if !containsString(result, "include = [") {
			t.Error("Expected include attribute to be added to profile")
		}

		// Verify tunnel data is present
		if !containsString(result, `"10.100.0.0/16"`) {
			t.Error("Expected tunnel address in include array")
		}

		if !containsString(result, `"Employee resources"`) {
			t.Error("Expected tunnel description in include array")
		}
	})

	t.Run("V4 cloudflare_zero_trust_device_profiles default profile with split tunnel without policy_id", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_profiles" "default" {
  account_id = "abc123"
  name       = "Default Profile"
  description = "Default device settings"
}

resource "cloudflare_split_tunnel" "default_tunnel" {
  account_id = "abc123"
  mode       = "exclude"
  tunnels {
    address     = "192.168.0.0/16"
    description = "Private network"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify the v4 profile still exists (becomes default profile due to lack of match/precedence)
		if !containsString(result, `resource "cloudflare_zero_trust_device_profiles" "default"`) {
			t.Error("Expected cloudflare_zero_trust_device_profiles resource to still exist")
		}

		// Verify exclude attribute is added (since mode is exclude)
		if !containsString(result, "exclude = [") {
			t.Error("Expected exclude attribute to be added to default profile")
		}

		// Verify include attribute is NOT present
		if containsString(result, "include = [") {
			t.Error("Did not expect include attribute when only exclude tunnel is present")
		}

		// Verify tunnel data is present
		if !containsString(result, `"192.168.0.0/16"`) {
			t.Error("Expected tunnel address in exclude array")
		}

		if !containsString(result, `"Private network"`) {
			t.Error("Expected tunnel description in exclude array")
		}
	})

	t.Run("V4 cloudflare_zero_trust_device_profiles with both default and custom profile and split tunnel without policy_id", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_profiles" "default" {
  account_id  = "abc123"
  name        = "Default Profile"
  description = "Default device settings"
}

resource "cloudflare_zero_trust_device_profiles" "contractors" {
  account_id = "abc123"
  name       = "Contractors"
  match      = "identity.email endsWith \"@contractor.com\""
  precedence = 100
}

resource "cloudflare_split_tunnel" "default_tunnel" {
  account_id = "abc123"
  mode       = "exclude"
  tunnels {
    address     = "172.16.0.0/12"
    description = "Default exclusion"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify both V4 profiles still exist
		if !containsString(result, `resource "cloudflare_zero_trust_device_profiles" "default"`) {
			t.Error("Expected default cloudflare_zero_trust_device_profiles resource to still exist")
		}
		if !containsString(result, `resource "cloudflare_zero_trust_device_profiles" "contractors"`) {
			t.Error("Expected contractors cloudflare_zero_trust_device_profiles resource to still exist")
		}

		// Find both profile blocks using HCL structure
		defaultBlock := findResourceBlock(file.Body(), "cloudflare_zero_trust_device_profiles", "default")
		contractorsBlock := findResourceBlock(file.Body(), "cloudflare_zero_trust_device_profiles", "contractors")

		if defaultBlock == nil {
			t.Fatal("Could not find default profile block")
		}
		if contractorsBlock == nil {
			t.Fatal("Could not find contractors profile block")
		}

		// Default profile should have exclude attribute (tunnel merged here)
		if defaultBlock.Body().GetAttribute("exclude") == nil {
			t.Error("Expected default profile to have exclude attribute with merged tunnel")
		}

		// Contractors profile should NOT have exclude attribute (tunnel should not merge here)
		if contractorsBlock.Body().GetAttribute("exclude") != nil {
			t.Error("Did not expect contractors profile to have exclude attribute")
		}

		// Verify tunnel data is present in the result (should be in default profile)
		if !containsString(result, `"172.16.0.0/12"`) {
			t.Error("Expected tunnel address to be present in default profile")
		}

		if !containsString(result, `"Default exclusion"`) {
			t.Error("Expected tunnel description to be present in default profile")
		}
	})

	t.Run("V4 cloudflare_zero_trust_device_profiles with explicit default=true attribute", func(t *testing.T) {
		input := `resource "cloudflare_zero_trust_device_profiles" "explicit_default" {
  account_id  = "abc123"
  name        = "Explicit Default"
  default     = true
  description = "Explicitly marked as default"
}

resource "cloudflare_split_tunnel" "explicit_default_tunnel" {
  account_id = "abc123"
  mode       = "include"
  tunnels {
    address     = "10.200.0.0/16"
    description = "Explicit default tunnel"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed
		if containsString(result, `resource "cloudflare_split_tunnel"`) {
			t.Error("Expected cloudflare_split_tunnel resources to be removed")
		}

		// Verify the v4 profile still exists
		if !containsString(result, `resource "cloudflare_zero_trust_device_profiles" "explicit_default"`) {
			t.Error("Expected cloudflare_zero_trust_device_profiles resource to still exist")
		}

		// Verify include attribute is added (tunnel merged to this profile due to explicit default=true)
		if !containsString(result, "include = [") {
			t.Error("Expected include attribute to be added to profile")
		}

		// Verify tunnel data is present
		if !containsString(result, `"10.200.0.0/16"`) {
			t.Error("Expected tunnel address in include array")
		}

		if !containsString(result, `"Explicit default tunnel"`) {
			t.Error("Expected tunnel description in include array")
		}
	})
}

func testDeviceProfileNotFoundSplitTunnelsConfig(t *testing.T) {
	t.Run("Split tunnel with missing default profile adds warning", func(t *testing.T) {
		input := `resource "cloudflare_split_tunnel" "exclude_tunnel" {
  account_id = "abc123"
  mode       = "exclude"
  tunnels {
    address     = "10.0.0.0/8"
    description = "Corporate network"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed as actual resource blocks (not just in comments)
		parsedFile, parseDiags := hclwrite.ParseConfig([]byte(result), "", hcl.Pos{Line: 1, Column: 1})
		if parseDiags.HasErrors() {
			t.Fatalf("Failed to parse result: %v", parseDiags)
		}
		for _, block := range parsedFile.Body().Blocks() {
			if block.Type() == "resource" && len(block.Labels()) >= 1 && block.Labels()[0] == "cloudflare_split_tunnel" {
				t.Error("Expected cloudflare_split_tunnel resource to be removed")
			}
		}

		// Verify a MIGRATION_WARNING comment is emitted for the orphaned default tunnel
		if !containsString(result, `/** MIGRATION_WARNING:`) {
			t.Error("Expected MIGRATION_WARNING comment for orphaned default tunnel")
		}
		// Verify the warning references the tunnel resource name
		if !containsString(result, `"exclude_tunnel"`) {
			t.Error("Expected warning to reference the orphaned tunnel resource name")
		}
		// Verify the warning mentions the default profile resource type
		if !containsString(result, "cloudflare_zero_trust_device_default_profile") {
			t.Error("Expected warning to mention cloudflare_zero_trust_device_default_profile")
		}
		// Verify the original tunnel config is preserved in the comment body
		if !containsString(result, "10.0.0.0/8") {
			t.Error("Expected original tunnel config to be preserved in the warning comment")
		}
	})

	t.Run("Split tunnel with missing custom profile adds warning", func(t *testing.T) {
		input := `resource "cloudflare_split_tunnel" "exclude_tunnel" {
  account_id = "abc123"
  policy_id  = cloudflare_zero_trust_device_custom_profile.missing.id
  mode       = "exclude"
  tunnels {
    address     = "10.0.0.0/8"
    description = "Corporate network"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed as actual resource blocks (not just in comments)
		// We need to parse the file and check that there are no actual resource blocks
		parsedFile, diags := hclwrite.ParseConfig([]byte(result), "", hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			t.Fatalf("Failed to parse result: %v", diags)
		}
		for _, block := range parsedFile.Body().Blocks() {
			if block.Type() == "resource" && len(block.Labels()) >= 1 && block.Labels()[0] == "cloudflare_split_tunnel" {
				t.Error("Expected cloudflare_split_tunnel resource to be removed")
			}
		}

		// Verify warning comment is present
		if !containsString(result, `/** MIGRATION_WARNING:`) {
			t.Error("Expected multi-line comment block with resource content")
		}
		if !containsString(result, "missing") && !containsString(result, "not found") {
			t.Error("Expected warning about missing custom profile")
		}
	})

	t.Run("Split tunnel with variable policy_id adds warning", func(t *testing.T) {
		input := `variable "policy_id" {
  type = string
}

resource "cloudflare_split_tunnel" "exclude_tunnel" {
  account_id = "abc123"
  policy_id  = var.policy_id
  mode       = "exclude"
  tunnels {
    address     = "10.0.0.0/8"
    description = "Corporate network"
  }
}`
		file, diags := hclwrite.ParseConfig([]byte(input), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("Failed to parse input: %v", diags)
		}

		ProcessCrossResourceConfigMigration(file)

		result := string(file.Bytes())

		// Verify split_tunnel resources are removed as actual resource blocks (not just in comments)
		// We need to parse the file and check that there are no actual resource blocks
		parsedFile, diags := hclwrite.ParseConfig([]byte(result), "", hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			t.Fatalf("Failed to parse result: %v", diags)
		}
		for _, block := range parsedFile.Body().Blocks() {
			if block.Type() == "resource" && len(block.Labels()) >= 1 && block.Labels()[0] == "cloudflare_split_tunnel" {
				t.Error("Expected cloudflare_split_tunnel resource to be removed")
			}
		}

		// Verify warning comment is present
		if !containsString(result, `/** MIGRATION_WARNING:`) {
			t.Error("Expected multi-line comment block with resource content")
		}
		if !containsString(result, "unparseable") {
			t.Error("Expected warning about unparseable policy_id reference")
		}
	})
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// findResourceBlock finds a resource block by type and name in the HCL body
func findResourceBlock(body *hclwrite.Body, resourceType, resourceName string) *hclwrite.Block {
	for _, block := range body.Blocks() {
		if block.Type() == "resource" {
			labels := block.Labels()
			if len(labels) == 2 && labels[0] == resourceType && labels[1] == resourceName {
				return block
			}
		}
	}
	return nil
}
