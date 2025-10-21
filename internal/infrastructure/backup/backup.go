package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	
	"github.com/cloudflare/tf-migrate/internal/core"
)

// Manager handles backup and restore operations for migration
type Manager struct {
	backups map[string]string // original path -> backup path
	created []string          // list of backup files created (for cleanup)
}

// New creates a new backup manager
func New() *Manager {
	return &Manager{
		backups: make(map[string]string),
		created: make([]string, 0),
	}
}

// Backup creates a backup of a file and tracks it for potential rollback
func (m *Manager) Backup(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File doesn't exist, nothing to backup (might be creating new file)
		return nil
	} else if err != nil {
		return core.NewError(core.BackupError).
			WithOperation("checking file status").
			WithFile(path).
			WithCause(err).
			Build()
	}

	// Check if already backed up
	if _, exists := m.backups[path]; exists {
		return nil // Already backed up
	}

	// Create backup path
	backupPath := path + ".backup"
	
	// Ensure unique backup name if it already exists
	counter := 1
	for {
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			break
		}
		backupPath = fmt.Sprintf("%s.backup.%d", path, counter)
		counter++
	}

	// Copy file to backup
	if err := m.copyFile(path, backupPath); err != nil {
		return core.NewError(core.BackupError).
			WithOperation("creating backup").
			WithFile(path).
			WithContext("backup_path", backupPath).
			WithCause(err).
			Build()
	}

	// Track the backup
	m.backups[path] = backupPath
	m.created = append(m.created, backupPath)

	return nil
}

// BackupDirectory creates backups for all matching files in a directory
func (m *Manager) BackupDirectory(dir string, pattern string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return core.NewError(core.FileError).
			WithOperation("reading directory").
			WithFile(dir).
			WithCause(err).
			Build()
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if file matches pattern
		if pattern != "" && !strings.HasSuffix(entry.Name(), pattern) {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())
		if err := m.Backup(fullPath); err != nil {
			// Error already wrapped by Backup method
			return err
		}
	}

	return nil
}

// Rollback restores all backed up files to their original state
func (m *Manager) Rollback() error {
	errorList := core.NewErrorList(20)

	// Restore all backups in reverse order
	for originalPath, backupPath := range m.backups {
		if err := m.restoreFile(backupPath, originalPath); err != nil {
			errorList.Add(core.NewError(core.BackupError).
				WithOperation("restoring file from backup").
				WithFile(originalPath).
				WithContext("backup_path", backupPath).
				WithCause(err).
				Build())
			// Continue trying to restore other files
		}
	}

	// Clean up any new files that were created (not backed up)
	// This would need to be tracked separately if needed

	if errorList.HasErrors() {
		return errorList
	}

	return nil
}

// Cleanup removes all backup files (call after successful migration)
func (m *Manager) Cleanup() error {
	errorList := core.NewErrorList(20)

	for _, backupPath := range m.created {
		if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
			errorList.Add(core.NewError(core.FileError).
				WithOperation("removing backup file").
				WithFile(backupPath).
				WithCause(err).
				Recoverable(). // Cleanup errors are recoverable
				Build())
		}
	}

	// Clear the tracking maps
	m.backups = make(map[string]string)
	m.created = []string{}

	if errorList.HasErrors() {
		return errorList
	}

	return nil
}

// GetBackupPath returns the backup path for a given original file
func (m *Manager) GetBackupPath(originalPath string) (string, bool) {
	backup, exists := m.backups[originalPath]
	return backup, exists
}

// HasBackups returns true if there are any backups
func (m *Manager) HasBackups() bool {
	return len(m.backups) > 0
}

// copyFile copies a file from src to dst
func (m *Manager) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Get file info to preserve permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// restoreFile restores a backup file to its original location
func (m *Manager) restoreFile(backupPath, originalPath string) error {
	// Remove the current file if it exists
	if err := os.Remove(originalPath); err != nil && !os.IsNotExist(err) {
		return core.NewError(core.FileError).
			WithOperation("removing current file for restore").
			WithFile(originalPath).
			WithCause(err).
			Build()
	}

	// Copy backup back to original location
	if err := m.copyFile(backupPath, originalPath); err != nil {
		return core.NewError(core.BackupError).
			WithOperation("restoring backup").
			WithFile(originalPath).
			WithContext("backup_path", backupPath).
			WithCause(err).
			Build()
	}
	return nil
}