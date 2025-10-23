package internal

import (
	"testing"
	"github.com/cloudflare/tf-migrate/internal/transform"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/tidwall/gjson"
)

type mockMigrator struct {
	version string
}

func (m *mockMigrator) CanHandle(resourceType string) bool {
	return true
}

func (m *mockMigrator) TransformConfig(ctx *transform.Context, block *hclwrite.Block) (*transform.TransformResult, error) {
	return nil, nil
}

func (m *mockMigrator) TransformState(ctx *transform.Context, json gjson.Result, resourcePath string) (string, error) {
	return "", nil
}

func (m *mockMigrator) GetResourceType() string {
	return "test_resource"
}

func (m *mockMigrator) Preprocess(content string) string {
	return content
}

func TestVersionBasedMigratorSelection(t *testing.T) {
	// Clear any existing registrations for clean test
	migrators = make(map[string]*MigratorRegistration)
	
	// Register migrators for different version pairs
	RegisterVersioned("test_resource", "v4", "v5", func() transform.ResourceTransformer {
		return &mockMigrator{version: "v4-v5"}
	})
	
	RegisterVersioned("test_resource", "v5", "v6", func() transform.ResourceTransformer {
		return &mockMigrator{version: "v5-v6"}
	})
	
	// Test v4 to v5 migration
	migrator := GetMigrator("test_resource", "v4", "v5")
	if migrator == nil {
		t.Fatal("Expected migrator for v4->v5, got nil")
	}
	if m, ok := migrator.(*mockMigrator); ok {
		if m.version != "v4-v5" {
			t.Errorf("Expected v4-v5 migrator, got %s", m.version)
		}
	}
	
	// Test v5 to v6 migration
	migrator = GetMigrator("test_resource", "v5", "v6")
	if migrator == nil {
		t.Fatal("Expected migrator for v5->v6, got nil")
	}
	if m, ok := migrator.(*mockMigrator); ok {
		if m.version != "v5-v6" {
			t.Errorf("Expected v5-v6 migrator, got %s", m.version)
		}
	}
	
	// Test non-existent migration path
	migrator = GetMigrator("test_resource", "v3", "v4")
	if migrator != nil {
		t.Error("Expected nil for non-existent v3->v4 migration")
	}
	
	// Test GetAllMigrators with version filtering
	all := GetAllMigrators("v4", "v5")
	if len(all) != 1 {
		t.Errorf("Expected 1 migrator for v4->v5, got %d", len(all))
	}
	
	all = GetAllMigrators("v5", "v6")
	if len(all) != 1 {
		t.Errorf("Expected 1 migrator for v5->v6, got %d", len(all))
	}
	
	all = GetAllMigrators("v3", "v4")
	if len(all) != 0 {
		t.Errorf("Expected 0 migrators for v3->v4, got %d", len(all))
	}
}

func TestRegisterConvenienceFunction(t *testing.T) {
	// Clear any existing registrations for clean test
	migrators = make(map[string]*MigratorRegistration)
	
	// Test that Register defaults to v4->v5
	Register("convenience_resource", func() transform.ResourceTransformer {
		return &mockMigrator{version: "default"}
	})
	
	// Should be accessible as v4->v5
	migrator := GetMigrator("convenience_resource", "v4", "v5")
	if migrator == nil {
		t.Fatal("Expected migrator for v4->v5 via Register(), got nil")
	}
	
	// Should not be accessible for other version pairs
	migrator = GetMigrator("convenience_resource", "v5", "v6")
	if migrator != nil {
		t.Error("Expected nil for v5->v6 when registered via Register()")
	}
}