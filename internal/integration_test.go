package internal

import (
	"testing"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
)

// mockV5ToV6Migrator simulates a future v5->v6 migrator
type mockV5ToV6Migrator struct{}

func (m *mockV5ToV6Migrator) CanHandle(resourceType string) bool {
	return resourceType == "cloudflare_dns_record"
}

func (m *mockV5ToV6Migrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	// Simulate some v5->v6 transformation
	body := block.Body()
	// Example: rename 'ttl' to 'time_to_live' in v6
	if attr := body.GetAttribute("ttl"); attr != nil {
		body.SetAttributeRaw("time_to_live", attr.Expr().BuildTokens(nil))
		body.RemoveAttribute("ttl")
	}
	return &transform.TransformResult{
		Blocks:         []*hclwrite.Block{block},
		RemoveOriginal: false,
	}, nil
}

func (m *mockV5ToV6Migrator) TransformState(ctx *transform.Context, json gjson.Result, resourcePath string) (string, error) {
	return "", nil
}

func (m *mockV5ToV6Migrator) GetResourceType() string {
	return "cloudflare_dns_record"
}

func (m *mockV5ToV6Migrator) Preprocess(content string) string {
	return content
}

func TestMultiVersionSupport(t *testing.T) {
	// Save current migrators and restore after test
	originalMigrators := migrators
	defer func() { migrators = originalMigrators }()
	
	// Clear for test
	migrators = make(map[string]*MigratorRegistration)
	
	// Register a v5->v6 migrator
	RegisterVersioned("cloudflare_dns_record", "v5", "v6", func() transform.ResourceTransformer {
		return &mockV5ToV6Migrator{}
	})
	
	// v5->v6 should return the mock migrator
	migrator := GetMigrator("cloudflare_dns_record", "v5", "v6")
	if migrator == nil {
		t.Fatal("Expected v5->v6 migrator, got nil")
	}
	if _, ok := migrator.(*mockV5ToV6Migrator); !ok {
		t.Error("Expected mockV5ToV6Migrator type")
	}
	
	// v4->v5 should return nil (not registered in this test)
	migrator = GetMigrator("cloudflare_dns_record", "v4", "v5")
	if migrator != nil {
		t.Error("Expected nil for v4->v5 (not registered in test)")
	}
	
	// GetAllMigrators should only return v5->v6 migrators
	all := GetAllMigrators("v5", "v6")
	if len(all) != 1 {
		t.Errorf("Expected 1 migrator for v5->v6, got %d", len(all))
	}
	
	all = GetAllMigrators("v4", "v5")
	if len(all) != 0 {
		t.Errorf("Expected 0 migrators for v4->v5 in test, got %d", len(all))
	}
}