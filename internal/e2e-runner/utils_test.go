package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetRepoRoot(t *testing.T) {
	// Create a temporary directory structure with go.mod
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "a", "b", "c")
	err := os.MkdirAll(nestedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	// Create go.mod at tmpDir
	goModPath := filepath.Join(tmpDir, "go.mod")
	err = os.WriteFile(goModPath, []byte("module test\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Change to nested directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldWd)

	err = os.Chdir(nestedDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Call getRepoRoot
	root := getRepoRoot()

	// Resolve symlinks for comparison (macOS /var vs /private/var)
	rootResolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		rootResolved = root
	}
	tmpDirResolved, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		tmpDirResolved = tmpDir
	}

	// Verify it found tmpDir
	if rootResolved != tmpDirResolved {
		t.Errorf("getRepoRoot() = %v, want %v", rootResolved, tmpDirResolved)
	}
}

func TestGetRepoRoot_NoGoMod(t *testing.T) {
	// Create a temporary directory without go.mod
	tmpDir := t.TempDir()

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Call getRepoRoot - should return "." when no go.mod found
	root := getRepoRoot()

	if root != "." {
		t.Errorf("getRepoRoot() = %v, want . when no go.mod found", root)
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcFile := filepath.Join(tmpDir, "source.txt")
	content := []byte("test content")
	err := os.WriteFile(srcFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy file
	dstFile := filepath.Join(tmpDir, "dest.txt")
	err = copyFile(srcFile, dstFile)
	if err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	// Verify destination file exists and has same content
	dstContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(dstContent) != string(content) {
		t.Errorf("Content mismatch: got %q, want %q", dstContent, content)
	}
}

func TestCopyFile_NonexistentSource(t *testing.T) {
	tmpDir := t.TempDir()

	srcFile := filepath.Join(tmpDir, "nonexistent.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")

	err := copyFile(srcFile, dstFile)
	if err == nil {
		t.Error("Expected error for nonexistent source file")
	}
}

func TestCopyFile_InvalidDestination(t *testing.T) {
	tmpDir := t.TempDir()

	srcFile := filepath.Join(tmpDir, "source.txt")
	err := os.WriteFile(srcFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Try to copy to a directory that doesn't exist
	dstFile := filepath.Join(tmpDir, "nonexistent", "dest.txt")

	err = copyFile(srcFile, dstFile)
	if err == nil {
		t.Error("Expected error for invalid destination path")
	}
}

func TestCopyDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source directory structure
	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("content2"), 0644)

	// Copy directory
	dstDir := filepath.Join(tmpDir, "dst")
	err := copyDir(srcDir, dstDir)
	if err != nil {
		t.Fatalf("copyDir() error = %v", err)
	}

	// Verify files copied
	tests := []struct {
		path    string
		content string
	}{
		{filepath.Join(dstDir, "file1.txt"), "content1"},
		{filepath.Join(dstDir, "subdir", "file2.txt"), "content2"},
	}

	for _, tt := range tests {
		content, err := os.ReadFile(tt.path)
		if err != nil {
			t.Errorf("Failed to read %s: %v", tt.path, err)
			continue
		}
		if string(content) != tt.content {
			t.Errorf("Content mismatch for %s: got %q, want %q", tt.path, content, tt.content)
		}
	}
}

func TestCopyDir_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "empty_src")
	dstDir := filepath.Join(tmpDir, "empty_dst")

	os.MkdirAll(srcDir, 0755)

	err := copyDir(srcDir, dstDir)
	if err != nil {
		t.Fatalf("copyDir() error = %v", err)
	}

	// Verify destination directory exists
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		t.Error("Destination directory was not created")
	}
}

func TestCopyDir_NonexistentSource(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "nonexistent")
	dstDir := filepath.Join(tmpDir, "dst")

	err := copyDir(srcDir, dstDir)
	if err == nil {
		t.Error("Expected error for nonexistent source directory")
	}
}

func TestCopyFile_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a 1MB file
	srcFile := filepath.Join(tmpDir, "large_source.txt")
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	err := os.WriteFile(srcFile, largeContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy file
	dstFile := filepath.Join(tmpDir, "large_dest.txt")
	err = copyFile(srcFile, dstFile)
	if err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	// Verify size matches
	srcInfo, _ := os.Stat(srcFile)
	dstInfo, _ := os.Stat(dstFile)

	if srcInfo.Size() != dstInfo.Size() {
		t.Errorf("File size mismatch: got %d, want %d", dstInfo.Size(), srcInfo.Size())
	}
}

func TestCopyDir_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create deeply nested structure
	srcDir := filepath.Join(tmpDir, "src")
	nestedPath := filepath.Join(srcDir, "a", "b", "c", "d")
	os.MkdirAll(nestedPath, 0755)
	os.WriteFile(filepath.Join(nestedPath, "deep.txt"), []byte("deep content"), 0644)

	// Copy directory
	dstDir := filepath.Join(tmpDir, "dst")
	err := copyDir(srcDir, dstDir)
	if err != nil {
		t.Fatalf("copyDir() error = %v", err)
	}

	// Verify nested file was copied
	dstNestedFile := filepath.Join(dstDir, "a", "b", "c", "d", "deep.txt")
	content, err := os.ReadFile(dstNestedFile)
	if err != nil {
		t.Errorf("Failed to read nested file: %v", err)
	}
	if string(content) != "deep content" {
		t.Errorf("Content mismatch: got %q, want %q", content, "deep content")
	}
}
