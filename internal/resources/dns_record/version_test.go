package dns_record

import (
	"testing"
	"github.com/cloudflare/tf-migrate/internal"
)

func TestDNSRecordVersionBasedSelection(t *testing.T) {
	// Test that dns_record is registered for v4->v5
	migrator := internal.GetMigrator("cloudflare_dns_record", "v4", "v5")
	if migrator == nil {
		t.Fatal("Expected dns_record migrator for v4->v5, got nil")
	}
	
	// Also check legacy cloudflare_record type
	migrator = internal.GetMigrator("cloudflare_record", "v4", "v5")
	if migrator == nil {
		t.Fatal("Expected cloudflare_record migrator for v4->v5, got nil")
	}
	
	// Test that it's NOT available for other version pairs
	migrator = internal.GetMigrator("cloudflare_dns_record", "v5", "v6")
	if migrator != nil {
		t.Error("Expected nil for v5->v6 migration (not implemented yet)")
	}
	
	migrator = internal.GetMigrator("cloudflare_dns_record", "v3", "v4")
	if migrator != nil {
		t.Error("Expected nil for v3->v4 migration (not implemented)")
	}
}

func TestGetMigratorFunction(t *testing.T) {
	// Test the GetMigrator function in this package
	migrator := GetMigrator("v4", "v5")
	if migrator == nil {
		t.Fatal("Expected migrator from GetMigrator(v4, v5)")
	}
	
	// Should return nil for unsupported versions
	migrator = GetMigrator("v5", "v6")
	if migrator != nil {
		t.Error("Expected nil from GetMigrator(v5, v6)")
	}
	
	migrator = GetMigrator("v3", "v4")
	if migrator != nil {
		t.Error("Expected nil from GetMigrator(v3, v4)")
	}
}