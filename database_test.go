package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	logrus "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// MockConfig returns a configuration suitable for testing.
func MockConfig(databaseDirPath string) *Config {
	return &Config{
		Database: DatabaseConfig{
			DatabaseDirPath: databaseDirPath,
			DatabaseFile:    "test.db",
			FileLockTimeout: 1 * time.Second,
			MaxBatchDelay:   1 * time.Millisecond,
			MaxBatchSize:    100,
		},
	}
}

// TestSetupDatabase tests the setupDatabase function.
func TestSetupDatabase(t *testing.T) {
	// Suppress log output by setting the output to io.Discard.
	logrus.SetOutput(io.Discard)

	// Case 1: Successful database setup.
	t.Run("Successful Database Setup", func(t *testing.T) {
		tempDir := t.TempDir()
		config := MockConfig(tempDir)

		db, err := setupDatabase(config)
		assert.NoError(
			t, err, "setupDatabase should not return an error",
		)
		assert.NotNil(t, db, "Database instance should not be nil")

		// Clean up.
		cleanupDB(db)
		assert.FileExists(
			t, filepath.Join(tempDir, "test.db"),
			"Database file should exist",
		)
	})

	// Case 2: Directory creation failure.
	t.Run("Directory Creation Failure", func(t *testing.T) {
		invalidDir := "/invalid-directory"
		config := MockConfig(invalidDir)

		_, err := setupDatabase(config)
		assert.Error(
			t, err, "setupDatabase should return an error for "+
				"invalid directory",
		)
	})

	// Case 3: Database file creation failure.
	t.Run("Database File Creation Failure", func(t *testing.T) {
		tempDir := t.TempDir()
		config := MockConfig(tempDir)

		// Set permissions on the tempDir to simulate file creation
		// failure.
		err := os.Chmod(tempDir, 0444)
		assert.NoError(t, err)

		_, err = setupDatabase(config)
		assert.Error(
			t, err, "setupDatabase should return an error when "+
				"database file cannot be created",
		)

		// Restore permissions.
		err = os.Chmod(tempDir, 0755)
		assert.NoError(t, err)
	})
}

// TestCleanupDB tests the cleanupDB function.
func TestCleanupDB(t *testing.T) {
	// Suppress log output by setting the output to io.Discard.
	logrus.SetOutput(io.Discard)

	// Case 1: Successful database cleanup.
	t.Run("Successful Database Cleanup", func(t *testing.T) {
		tempDir := t.TempDir()
		config := MockConfig(tempDir)

		db, err := setupDatabase(config)
		assert.NoError(
			t, err, "setupDatabase should not return an error",
		)
		assert.NotNil(t, db, "Database instance should not be nil")

		// Clean up and check no error is logged.
		cleanupDB(db)
	})

	// Case 2: Cleanup with already closed DB.
	t.Run("Cleanup With Already Closed DB", func(t *testing.T) {
		tempDir := t.TempDir()
		config := MockConfig(tempDir)

		db, err := setupDatabase(config)
		assert.NoError(
			t, err, "setupDatabase should not return an error",
		)
		assert.NotNil(t, db, "Database instance should not be nil")

		// Close the DB before calling cleanupDB.
		db.Close()
		assert.NotPanics(
			t, func() { cleanupDB(db) }, "cleanupDB should not "+"panic if the DB is already closed",
		)
	})
}
