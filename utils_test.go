package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCheckFilesExist tests the checkFilesExist function.
func TestCheckFilesExist(t *testing.T) {
	// Define a temporary directory for test files.
	tempDir := t.TempDir()

	// Case 1: All files exist.
	t.Run("All files exist", func(t *testing.T) {
		// Create test files.
		file1 := filepath.Join(tempDir, "file1.txt")
		file2 := filepath.Join(tempDir, "file2.txt")
		assert.NoError(
			t, os.WriteFile(file1, []byte("test content"), 0644),
		)
		assert.NoError(
			t, os.WriteFile(file2, []byte("test content"), 0644),
		)

		// Check that all files exist.
		err := checkFilesExist(file1, file2)
		assert.NoError(t, err, "All specified files should exist")
	})

	// Case 2: One file does not exist.
	t.Run("One file does not exist", func(t *testing.T) {
		// Create one test file.
		file1 := filepath.Join(tempDir, "file1.txt")
		assert.NoError(
			t, os.WriteFile(file1, []byte("test content"), 0644),
		)

		// Check that one file does not exist.
		err := checkFilesExist(
			file1, filepath.Join(tempDir, "nonexistent.txt"),
		)
		assert.Error(
			t, err, "An error should be returned if any specified "+
				"file does not exist",
		)
	})

	// Case 3: No files exist.
	t.Run("No files exist", func(t *testing.T) {
		// Check that no files exist.
		err := checkFilesExist(
			filepath.Join(tempDir, "nonexistent1.txt"),
			filepath.Join(tempDir, "nonexistent2.txt"),
		)
		assert.Error(
			t, err, "An error should be returned if no "+
				"specified files exist",
		)
	})

	// Case 4: Empty file list.
	t.Run("Empty file list", func(t *testing.T) {
		// Check with an empty list of files.
		err := checkFilesExist()
		assert.NoError(
			t, err, "No error should be returned for an empty "+
				"file list",
		)
	})
}

// TestMostRecentUnixTimestamp tests the mostRecentUnixTimestamp function.
func TestMostRecentUnixTimestamp(t *testing.T) {
	// Case 1: Two timestamps provided.
	t.Run("Two timestamps provided", func(t *testing.T) {
		result := mostRecentUnixTimestamp(
			1622132035, 1622132045,
		)
		expected := int64(1622132045)
		if result != expected {
			t.Errorf("mostRecentUnixTimestamp returned %d, "+
				"expected %d", result, expected)
		}
	})

	// Case 2: Three timestamps provided.
	t.Run("Three timestamps provided", func(t *testing.T) {
		result := mostRecentUnixTimestamp(
			1622132035, 1622132045, 1622132055,
		)
		expected := int64(1622132055)
		if result != expected {
			t.Errorf("mostRecentUnixTimestamp returned %d, "+
				"expected %d", result, expected)
		}
	})

	// Case 3: Four timestamps provided with one negative value.
	t.Run("Four timestamps with negative value", func(t *testing.T) {
		result := mostRecentUnixTimestamp(
			-1622132035, 1622132045, 1622132055, 1622132065,
		)
		expected := int64(1622132065)
		if result != expected {
			t.Errorf("mostRecentUnixTimestamp returned %d, "+
				"expected %d", result, expected)
		}
	})

	// Case 4: Five timestamps provided with one zero value.
	t.Run("Five timestamps with zero value", func(t *testing.T) {
		result := mostRecentUnixTimestamp(
			0, 1622132035, 1622132045, 1622132055, 1622132065,
		)
		expected := int64(1622132065)
		if result != expected {
			t.Errorf("mostRecentUnixTimestamp returned %d, "+
				"expected %d", result, expected)
		}
	})

	// Case 5: No timestamps provided.
	t.Run("No timestamps provided", func(t *testing.T) {
		result := mostRecentUnixTimestamp()
		// By default, return 0 if no timestamps provided.
		expected := int64(0)
		if result != expected {
			t.Errorf("mostRecentUnixTimestamp returned %d, "+
				"expected %d", result, expected)
		}
	})
}

// TestFormatDuration tests the formatDuration function.
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration // Duration to be formatted
		expected string        // Expected formatted string
	}{
		{duration: time.Second * 45, expected: "0 hours, 0 minutes, 45 seconds"},                                 // Less than a minute
		{duration: time.Minute * 60, expected: "1 hours, 0 minutes, 0 seconds"},                                  // Exactly one hour
		{duration: time.Minute*120 + time.Second*30, expected: "2 hours, 0 minutes, 30 seconds"},                 // More than one hour
		{duration: time.Hour*25 + time.Minute*30 + time.Second*15, expected: "25 hours, 30 minutes, 15 seconds"}, // Spanning multiple days
		{duration: 0, expected: "0 hours, 0 minutes, 0 seconds"},                                                 // Zero duration
		{duration: -time.Minute * 30, expected: "0 hours, -30 minutes, 0 seconds"},                               // Negative duration
	}

	for _, test := range tests {
		result := formatDuration(test.duration)
		if result != test.expected {
			t.Errorf("formatDuration(%v) returned %s, expected %s", test.duration, result, test.expected)
		}
	}
}
