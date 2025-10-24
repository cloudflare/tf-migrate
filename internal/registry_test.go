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
	migrators = make(map[string]*Migrator)

	// Register migrators for different version pairs
	Register("test_resource", 4, 5, func() transform.ResourceTransformer {
		return &mockMigrator{version: "4-5"}
	})

	Register("test_resource", 5, 6, func() transform.ResourceTransformer {
		return &mockMigrator{version: "5-6"}
	})

	// Test 4 to 5 migration
	migrator := GetMigrator("test_resource", 4, 5)
	if migrator == nil {
		t.Fatal("Expected migrator for 4->5, got nil")
	}
	if m, ok := migrator.(*mockMigrator); ok {
		if m.version != "4-5" {
			t.Errorf("Expected 4-5 migrator, got %s", m.version)
		}
	}

	// Test 5 to 6 migration
	migrator = GetMigrator("test_resource", 5, 6)
	if migrator == nil {
		t.Fatal("Expected migrator for 5->6, got nil")
	}
	if m, ok := migrator.(*mockMigrator); ok {
		if m.version != "5-6" {
			t.Errorf("Expected 5-6 migrator, got %s", m.version)
		}
	}

	// Test non-existent migration path
	migrator = GetMigrator("test_resource", 3, 4)
	if migrator != nil {
		t.Error("Expected nil for non-existent 3->4 migration")
	}

	// Test GetAllMigrators with version filtering
	all := GetAllMigrators(4, 5)
	if len(all) != 1 {
		t.Errorf("Expected 1 migrator for 4->5, got %d", len(all))
	}

	all = GetAllMigrators(5, 6)
	if len(all) != 1 {
		t.Errorf("Expected 1 migrator for 5->6, got %d", len(all))
	}

	all = GetAllMigrators(3, 4)
	if len(all) != 0 {
		t.Errorf("Expected 0 migrators for 3->4, got %d", len(all))
	}
}
