package hcl

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func TestExtractMetaArguments(t *testing.T) {
	tests := []struct {
		name          string
		hcl           string
		expectCount   bool
		expectForEach bool
		expectLifecycle bool
		expectDependsOn bool
		expectProvider bool
		expectTimeouts bool
	}{
		{
			name: "Block with all meta-arguments",
			hcl: `
resource "cloudflare_zone" "example" {
  zone_id = "abc123"
  count = 5
  for_each = var.zones
  depends_on = [cloudflare_account.main]
  provider = cloudflare.alt

  lifecycle {
    ignore_changes = [modified_on]
  }

  timeouts {
    create = "30m"
  }
}`,
			expectCount: true,
			expectForEach: true,
			expectLifecycle: true,
			expectDependsOn: true,
			expectProvider: true,
			expectTimeouts: true,
		},
		{
			name: "Block with no meta-arguments",
			hcl: `
resource "cloudflare_zone" "example" {
  zone_id = "abc123"
}`,
			expectCount: false,
			expectForEach: false,
			expectLifecycle: false,
			expectDependsOn: false,
			expectProvider: false,
			expectTimeouts: false,
		},
		{
			name: "Block with only count",
			hcl: `
resource "cloudflare_zone" "example" {
  zone_id = "abc123"
  count = 3
}`,
			expectCount: true,
			expectForEach: false,
			expectLifecycle: false,
			expectDependsOn: false,
			expectProvider: false,
			expectTimeouts: false,
		},
		{
			name: "Block with only for_each",
			hcl: `
resource "cloudflare_zone" "example" {
  zone_id = "abc123"
  for_each = var.zones
}`,
			expectCount: false,
			expectForEach: true,
			expectLifecycle: false,
			expectDependsOn: false,
			expectProvider: false,
			expectTimeouts: false,
		},
		{
			name: "Block with only lifecycle",
			hcl: `
resource "cloudflare_zone" "example" {
  zone_id = "abc123"

  lifecycle {
    prevent_destroy = true
  }
}`,
			expectCount: false,
			expectForEach: false,
			expectLifecycle: true,
			expectDependsOn: false,
			expectProvider: false,
			expectTimeouts: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, _ := hclwrite.ParseConfig([]byte(tt.hcl), "test.tf", hcl.Pos{})
			blocks := file.Body().Blocks()
			if len(blocks) == 0 {
				t.Fatal("No blocks found in HCL")
			}

			meta := ExtractMetaArguments(blocks[0])

			if (meta.Count != nil) != tt.expectCount {
				t.Errorf("Count: got %v, want %v", meta.Count != nil, tt.expectCount)
			}
			if (meta.ForEach != nil) != tt.expectForEach {
				t.Errorf("ForEach: got %v, want %v", meta.ForEach != nil, tt.expectForEach)
			}
			if (meta.Lifecycle != nil) != tt.expectLifecycle {
				t.Errorf("Lifecycle: got %v, want %v", meta.Lifecycle != nil, tt.expectLifecycle)
			}
			if (meta.DependsOn != nil) != tt.expectDependsOn {
				t.Errorf("DependsOn: got %v, want %v", meta.DependsOn != nil, tt.expectDependsOn)
			}
			if (meta.Provider != nil) != tt.expectProvider {
				t.Errorf("Provider: got %v, want %v", meta.Provider != nil, tt.expectProvider)
			}
			if (meta.Timeouts != nil) != tt.expectTimeouts {
				t.Errorf("Timeouts: got %v, want %v", meta.Timeouts != nil, tt.expectTimeouts)
			}
		})
	}
}

func TestCopyMetaArgumentsToBlock(t *testing.T) {
	tests := []struct {
		name string
		sourceHCL string
		expectInDest []string
	}{
		{
			name: "Copy count",
			sourceHCL: `
resource "cloudflare_zone" "example" {
  zone_id = "abc123"
  count = 5
}`,
			expectInDest: []string{"count"},
		},
		{
			name: "Copy lifecycle",
			sourceHCL: `
resource "cloudflare_zone" "example" {
  zone_id = "abc123"

  lifecycle {
    ignore_changes = [modified_on]
  }
}`,
			expectInDest: []string{"lifecycle"},
		},
		{
			name: "Copy multiple meta-arguments",
			sourceHCL: `
resource "cloudflare_zone" "example" {
  zone_id = "abc123"
  count = 3
  depends_on = [cloudflare_account.main]

  lifecycle {
    prevent_destroy = true
  }
}`,
			expectInDest: []string{"count", "depends_on", "lifecycle"},
		},
		{
			name: "Copy nil meta (no-op)",
			sourceHCL: `
resource "cloudflare_zone" "example" {
  zone_id = "abc123"
}`,
			expectInDest: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, _ := hclwrite.ParseConfig([]byte(tt.sourceHCL), "test.tf", hcl.Pos{})
			sourceBlock := file.Body().Blocks()[0]

			meta := ExtractMetaArguments(sourceBlock)

			destBlock := hclwrite.NewBlock("resource", []string{"cloudflare_zone", "dest"})
			CopyMetaArgumentsToBlock(destBlock, meta)

			destBody := destBlock.Body()

			// Check expected attributes exist
			for _, attrName := range tt.expectInDest {
				if attrName == "lifecycle" || attrName == "timeouts" {
					if FindBlockByType(destBody, attrName) == nil {
						t.Errorf("Expected block %q not found", attrName)
					}
				} else {
					if destBody.GetAttribute(attrName) == nil {
						t.Errorf("Expected attribute %q not found", attrName)
					}
				}
			}
		})
	}
}

func TestCopyMetaArgumentsToImport(t *testing.T) {
	tests := []struct {
		name string
		sourceHCL string
		expectInImport []string
		notExpectInImport []string
	}{
		{
			name: "Copy for_each (allowed)",
			sourceHCL: `
resource "cloudflare_zone" "example" {
  zone_id = "abc123"
  for_each = var.zones
}`,
			expectInImport: []string{"for_each"},
			notExpectInImport: []string{},
		},
		{
			name: "Don't copy count (not allowed)",
			sourceHCL: `
resource "cloudflare_zone" "example" {
  zone_id = "abc123"
  count = 5
}`,
			expectInImport: []string{},
			notExpectInImport: []string{"count"},
		},
		{
			name: "Copy provider (allowed)",
			sourceHCL: `
resource "cloudflare_zone" "example" {
  zone_id = "abc123"
  provider = cloudflare.alt
}`,
			expectInImport: []string{"provider"},
			notExpectInImport: []string{},
		},
		{
			name: "Don't copy lifecycle (not allowed)",
			sourceHCL: `
resource "cloudflare_zone" "example" {
  zone_id = "abc123"

  lifecycle {
    prevent_destroy = true
  }
}`,
			expectInImport: []string{},
			notExpectInImport: []string{"lifecycle"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, _ := hclwrite.ParseConfig([]byte(tt.sourceHCL), "test.tf", hcl.Pos{})
			sourceBlock := file.Body().Blocks()[0]

			meta := ExtractMetaArguments(sourceBlock)

			importBlock := hclwrite.NewBlock("import", nil)
			CopyMetaArgumentsToImport(importBlock, meta)

			importBody := importBlock.Body()

			// Check expected attributes exist
			for _, attrName := range tt.expectInImport {
				if importBody.GetAttribute(attrName) == nil {
					t.Errorf("Expected attribute %q not found in import block", attrName)
				}
			}

			// Check unwanted attributes don't exist
			for _, attrName := range tt.notExpectInImport {
				if attrName == "lifecycle" || attrName == "timeouts" {
					if FindBlockByType(importBody, attrName) != nil {
						t.Errorf("Unexpected block %q found in import block", attrName)
					}
				} else {
					if importBody.GetAttribute(attrName) != nil {
						t.Errorf("Unexpected attribute %q found in import block", attrName)
					}
				}
			}
		})
	}
}
