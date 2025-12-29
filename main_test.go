package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldMigrate(t *testing.T) {
	// Test case 1: marker exists, should return false
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	markerPath := filepath.Join(tempDir, ".migrated")

	// Create marker file
	if err := os.WriteFile(markerPath, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	if shouldMigrate(dbPath) {
		t.Error("Expected false when marker exists")
	}

	// Test case 2: marker doesn't exist, should return true
	os.Remove(markerPath)
	if !shouldMigrate(dbPath) {
		t.Error("Expected true when marker doesn't exist")
	}
}
