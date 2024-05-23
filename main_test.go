package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAppPath verifies that the application path is correctly determined
// for different operating systems. It sets up test cases for Windows,
// Darwin (macOS), Plan 9, and Linux, and checks the expected application
// paths against the actual paths returned by the AppPath function.
func TestAppPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}

	tests := []struct {
		name        string
		goos        string
		expected    string
		envVarName  string
		envVarValue string
	}{
		{
			name:     "windows",
			goos:     "windows",
			expected: filepath.Join(os.Getenv("LOCALAPPDATA"), DefaultAppName),
		},
		{
			name:     "darwin",
			goos:     "darwin",
			expected: filepath.Join(homeDir, "Library", "Application Support", DefaultAppName),
		},
		{
			name:     "plan9",
			goos:     "plan9",
			expected: filepath.Join(homeDir, strings.ToLower(DefaultAppName)),
		},
		{
			name:     "linux",
			goos:     "linux",
			expected: filepath.Join(homeDir, "."+strings.ToLower(DefaultAppName)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVarName != "" {
				os.Setenv(tt.envVarName, tt.envVarValue)
				defer os.Unsetenv(tt.envVarName)
			}
			assert.Equal(t, tt.expected, AppPath(tt.goos, homeDir))
		})
	}
}

// TestEnsureAppPathExists checks the EnsureAppPathExists function to
// ensure it correctly handles the existence of the application path.
func TestEnsureAppPathExists(t *testing.T) {
	tmpDir := os.TempDir()
	appPath := filepath.Join(tmpDir, DefaultAppName)

	// Case 1: Directory does not exist and is created successfully.
	t.Run("DirectoryDoesNotExist", func(t *testing.T) {
		defer os.RemoveAll(appPath)

		err := EnsureAppPathExists(appPath)
		assert.NoError(t, err)

		_, err = os.Stat(appPath)
		assert.False(t, os.IsNotExist(err))
	})

	// Case 2: Directory already exists.
	t.Run("DirectoryAlreadyExists", func(t *testing.T) {
		defer os.RemoveAll(appPath)

		err := os.MkdirAll(appPath, AppDirPermissions)
		assert.NoError(t, err)

		err = EnsureAppPathExists(appPath)
		assert.NoError(t, err)
	})
}
