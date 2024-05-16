package main

import (
	"os"
	"path/filepath"
	"testing"

	logrus "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestSetupLogging tests the setupLogging function.
func TestSetupLogging(t *testing.T) {
	// Define a temporary directory for test logs.
	tempDir := t.TempDir()

	// Helper function to create a config with the specified log level.
	createConfig := func(logDirPath, logFile, logLevel string) *Config {
		return &Config{
			Log: LogConfig{
				LogDirPath: logDirPath,
				LogFile:    logFile,
				LogLevel:   logLevel,
			},
		}
	}

	// Case 1: Successful logging setup.
	t.Run("Successful logging setup", func(t *testing.T) {
		config := createConfig(tempDir, "test-log.log", "info")
		err := setupLogging(config)
		assert.NoError(t, err)

		// Verify that the log file was created.
		logFilePath := filepath.Join(tempDir, "test-log.log")
		assert.FileExists(t, logFilePath)

		// Open the log file.
		file, err := os.OpenFile(
			logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644,
		)
		assert.NoError(t, err)
		defer file.Close()

		// Set output of logging to the file only.
		logrus.SetOutput(file)

		// Write a log entry and verify its content.
		logrus.Info("Test log entry")
		content, err := os.ReadFile(logFilePath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "Test log entry")
	})

	// Case 2: Log directory does not exist and needs to be created.
	t.Run("Log directory creation", func(t *testing.T) {
		newDir := filepath.Join(tempDir, "new-log-dir")
		config := createConfig(newDir, "test-log.log", "info")
		err := setupLogging(config)
		assert.NoError(t, err)

		// Verify that the log directory and file were created.
		logFilePath := filepath.Join(newDir, "test-log.log")
		assert.DirExists(t, newDir)
		assert.FileExists(t, logFilePath)
	})

	// Case 3: Error creating log directory.
	t.Run("Error creating log directory", func(t *testing.T) {
		// Use an invalid directory path to force an error.
		invalidDir := "/invalid-directory"
		config := createConfig(invalidDir, "test-log.log", "info")
		err := setupLogging(config)
		assert.Error(t, err)
	})

	// Case 4: Error opening log file.
	t.Run("Error opening log file", func(t *testing.T) {
		// Use an invalid file name to force an error.
		config := createConfig(tempDir, "", "info")
		err := setupLogging(config)
		assert.Error(t, err)
	})

	// Case 5: Invalid log level.
	t.Run("Invalid log level", func(t *testing.T) {

		config := createConfig(tempDir, "test-log.log", "invalid")
		err := setupLogging(config)
		assert.NoError(t, err)

		// Verify that the log file was created.
		logFilePath := filepath.Join(tempDir, "test-log.log")
		assert.FileExists(t, logFilePath)

		// Open the log file.
		file, err := os.OpenFile(
			logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644,
		)
		assert.NoError(t, err)
		defer file.Close()

		// Set output of logging to the file only.
		logrus.SetOutput(file)

		// Write a log entry and verify its content.
		logrus.Info("Test log entry with invalid log level")
		content, err := os.ReadFile(logFilePath)

		assert.NoError(t, err)
		assert.Contains(
			t, string(content),
			"Test log entry with invalid log level",
		)
	})
}

// TestConvertLogLevel tests the convertLogLevel function.
func TestConvertLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected logrus.Level
	}{
		{"fatal", logrus.FatalLevel},
		{"error", logrus.ErrorLevel},
		{"warn", logrus.WarnLevel},
		{"warning", logrus.WarnLevel},
		{"info", logrus.InfoLevel},
		{"debug", logrus.DebugLevel},
		{"unknown", logrus.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertLogLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
