package backup_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudflare/tf-migrate/internal/infrastructure/backup"
)

func TestBackupAndRestore(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()
	
	// Create test files
	testFile1 := filepath.Join(tmpDir, "test1.tf")
	testFile2 := filepath.Join(tmpDir, "test2.tf")
	originalContent1 := []byte("resource \"test\" \"one\" {}")
	originalContent2 := []byte("resource \"test\" \"two\" {}")
	
	if err := os.WriteFile(testFile1, originalContent1, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFile2, originalContent2, 0644); err != nil {
		t.Fatal(err)
	}
	
	// Create backup manager
	manager := backup.New()
	
	// Test backup creation
	t.Run("Create backups", func(t *testing.T) {
		if err := manager.Backup(testFile1); err != nil {
			t.Fatalf("Failed to backup file1: %v", err)
		}
		if err := manager.Backup(testFile2); err != nil {
			t.Fatalf("Failed to backup file2: %v", err)
		}
		
		// Verify backup files exist
		if _, err := os.Stat(testFile1 + ".backup"); err != nil {
			t.Error("Backup file1 not created")
		}
		if _, err := os.Stat(testFile2 + ".backup"); err != nil {
			t.Error("Backup file2 not created")
		}
	})
	
	// Test file modification and rollback
	t.Run("Modify and rollback", func(t *testing.T) {
		// Modify the original files
		modifiedContent1 := []byte("MODIFIED CONTENT 1")
		modifiedContent2 := []byte("MODIFIED CONTENT 2")
		
		if err := os.WriteFile(testFile1, modifiedContent1, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(testFile2, modifiedContent2, 0644); err != nil {
			t.Fatal(err)
		}
		
		// Verify files were modified
		content, _ := os.ReadFile(testFile1)
		if string(content) != string(modifiedContent1) {
			t.Error("File1 was not modified")
		}
		
		// Rollback
		if err := manager.Rollback(); err != nil {
			t.Fatalf("Rollback failed: %v", err)
		}
		
		// Verify files were restored
		content1, err := os.ReadFile(testFile1)
		if err != nil {
			t.Fatal(err)
		}
		if string(content1) != string(originalContent1) {
			t.Errorf("File1 not restored correctly. Got: %s, Want: %s", content1, originalContent1)
		}
		
		content2, err := os.ReadFile(testFile2)
		if err != nil {
			t.Fatal(err)
		}
		if string(content2) != string(originalContent2) {
			t.Errorf("File2 not restored correctly. Got: %s, Want: %s", content2, originalContent2)
		}
	})
	
	// Test cleanup
	t.Run("Cleanup backups", func(t *testing.T) {
		// Create new backups
		manager2 := backup.New()
		_ = manager2.Backup(testFile1)
		
		backupPath, exists := manager2.GetBackupPath(testFile1)
		if !exists {
			t.Fatal("Backup path not tracked")
		}
		
		// Verify backup exists
		if _, err := os.Stat(backupPath); err != nil {
			t.Fatal("Backup file doesn't exist")
		}
		
		// Cleanup
		if err := manager2.Cleanup(); err != nil {
			t.Fatalf("Cleanup failed: %v", err)
		}
		
		// Verify backup was removed
		if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
			t.Error("Backup file was not removed during cleanup")
		}
	})
}

func TestBackupDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create test files
	tf1 := filepath.Join(tmpDir, "main.tf")
	tf2 := filepath.Join(tmpDir, "variables.tf")
	txt := filepath.Join(tmpDir, "readme.txt")
	
	_ = os.WriteFile(tf1, []byte("content1"), 0644)
	_ = os.WriteFile(tf2, []byte("content2"), 0644)
	_ = os.WriteFile(txt, []byte("text"), 0644)
	
	manager := backup.New()
	
	// Backup only .tf files
	if err := manager.BackupDirectory(tmpDir, ".tf"); err != nil {
		t.Fatalf("BackupDirectory failed: %v", err)
	}
	
	// Check that .tf files were backed up
	if _, err := os.Stat(tf1 + ".backup"); err != nil {
		t.Error("main.tf was not backed up")
	}
	if _, err := os.Stat(tf2 + ".backup"); err != nil {
		t.Error("variables.tf was not backed up")
	}
	
	// Check that .txt file was NOT backed up
	if _, err := os.Stat(txt + ".backup"); !os.IsNotExist(err) {
		t.Error("readme.txt should not have been backed up")
	}
}

func TestBackupNonExistentFile(t *testing.T) {
	manager := backup.New()
	
	// Backing up non-existent file should not error (might be creating new file)
	nonExistent := "/tmp/does-not-exist-12345.tf"
	if err := manager.Backup(nonExistent); err != nil {
		t.Errorf("Backup of non-existent file should not error: %v", err)
	}
}

func TestDuplicateBackup(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.tf")
	_ = os.WriteFile(testFile, []byte("content"), 0644)
	
	manager := backup.New()
	
	// First backup
	if err := manager.Backup(testFile); err != nil {
		t.Fatal(err)
	}
	
	// Second backup of same file should be idempotent
	if err := manager.Backup(testFile); err != nil {
		t.Errorf("Duplicate backup should not error: %v", err)
	}
	
	// Should still only have one backup file
	backupPath, _ := manager.GetBackupPath(testFile)
	if backupPath != testFile+".backup" {
		t.Error("Backup path changed on duplicate backup")
	}
}