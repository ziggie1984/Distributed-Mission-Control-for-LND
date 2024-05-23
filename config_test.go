package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ory/viper"
	"github.com/stretchr/testify/assert"
)

// TestWriteConfigSection tests the writeConfigSection function to ensure it
// correctly writes configuration sections and key-value pairs to the writer.
func TestWriteConfigSection(t *testing.T) {
	// Case 1: Basic writing of configuration.
	t.Run("Basic Writing", func(t *testing.T) {
		var b bytes.Buffer

		// Define a simple config struct for testing purposes.
		type testConfig struct {
			Name    string `mapstructure:"name" description:"The name of the user"`
			Enabled bool   `mapstructure:"enabled" description:"Whether feature is enabled"`
		}
		config := testConfig{
			Name:    "test-user",
			Enabled: true,
		}

		// Write the config section to the buffer.
		err := writeConfigSection(
			&b, reflect.ValueOf(config), reflect.TypeOf(config), "",
		)
		assert.NoError(
			t, err,
			"Expected no error while writing config section",
		)

		// Convert buffer contents to a string for comparison.
		result := b.String()
		expected := strings.Join([]string{
			"; The name of the user",
			"name = test-user",
			"",
			"; Whether feature is enabled",
			"enabled = true",
			"",
		}, "\n") + "\n"

		// Compare the actual output with the expected output.
		assert.Equal(
			t, expected, result, "Config section output mismatch",
		)
	})

	// Case 2: Handling of nil writer.
	t.Run("Nil Writer", func(t *testing.T) {
		type testConfig struct {
			Name string `mapstructure:"name" description:"The name of the user"`
		}
		config := testConfig{Name: "test-user"}

		// Attempt to write the config section to a nil writer.
		err := writeConfigSection(
			nil, reflect.ValueOf(config), reflect.TypeOf(config),
			"",
		)
		assert.Error(t, err, "Expected error when writer is nil")
	})

	// Case 3: Writing an empty struct.
	t.Run("Empty Struct", func(t *testing.T) {
		var b bytes.Buffer
		type emptyConfig struct{}

		emptyInstance := emptyConfig{}
		err := writeConfigSection(&b, reflect.ValueOf(emptyInstance), reflect.TypeOf(emptyInstance), "")
		assert.NoError(
			t, err, "Expected no error while writing empty struct",
		)
		assert.Equal(
			t, "", b.String(),
			"Expected empty output for empty struct",
		)
	})

	// Case 4: Writing nested structs.
	t.Run("Nested Structs", func(t *testing.T) {
		var b bytes.Buffer
		type InnerConfig struct {
			Enabled bool `mapstructure:"enabled" description:"Enable feature"`
		}
		type OuterConfig struct {
			Inner InnerConfig `mapstructure:"inner" description:"Inner config"`
		}
		config := OuterConfig{Inner: InnerConfig{Enabled: true}}

		// Write the nested struct to the buffer.
		err := writeConfigSection(
			&b, reflect.ValueOf(config), reflect.TypeOf(config), "",
		)
		assert.NoError(
			t, err, "Expected no error while writing nested struct",
		)

		// Define the expected output for nested structs.
		expected := strings.Join([]string{
			"; Inner config",
			"[inner]",
			"; Enable feature",
			"enabled = true",
			"",
			"",
		}, "\n")
		assert.Equal(
			t, expected, b.String(),
			"Nested struct output mismatch",
		)
	})

	// Case 5: Handling writer failures.
	t.Run("Failing Writer", func(t *testing.T) {
		type testConfig struct {
			Name string `mapstructure:"name" description:"The name of the user"`
		}
		config := testConfig{Name: "test-user"}

		// Attempt to write the config section to a failing writer.
		err := writeConfigSection(
			&failingWriter{}, reflect.ValueOf(config),
			reflect.TypeOf(config), "",
		)
		assert.Error(t, err, "Expected error when using failing writer")
	})
}

// failingWriter is a mock writer that always fails to write.
type failingWriter struct{}

// Write implements the io.Writer interface for failingWriter
// and always returns an error.
func (f *failingWriter) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("intentional write failure")
}

// TestWrapText tests the wrapText function to ensure it wraps text at the
// specified line width.
func TestWrapText(t *testing.T) {
	// Case 1: Basic text wrapping.
	t.Run("Basic Wrapping", func(t *testing.T) {
		text := "This is a long string that needs to be wrapped"
		lineWidth := 10
		expected := "This is a\n; long\n; string\n; that needs\n; to be\n; wrapped"
		result := wrapText(text, lineWidth)
		assert.Equal(
			t, expected, result, "Basic wrapping output mismatch",
		)
	})

	// Case 2: Text with exact width match.
	t.Run("Exact Width Match", func(t *testing.T) {
		text := "12345 67890"
		lineWidth := 11
		expected := "12345 67890"
		result := wrapText(text, lineWidth)
		assert.Equal(
			t, expected, result,
			"Exact width match output mismatch",
		)
	})

	// Case 3: Single word that exceeds the line width.
	t.Run("Single Word", func(t *testing.T) {
		text := "1234567890"
		lineWidth := 5
		expected := "1234567890"
		result := wrapText(text, lineWidth)
		assert.Equal(t, expected, result, "Single word output mismatch")
	})

	// Case 4: Zero line width, should not wrap.
	t.Run("Zero Line Width", func(t *testing.T) {
		text := "This should not wrap"
		lineWidth := 0
		expected := "This should not wrap"
		result := wrapText(text, lineWidth)
		assert.Equal(
			t, expected, result, "Zero line width output mismatch",
		)
	})

	// Case 5: Empty string, should return empty.
	t.Run("Empty String", func(t *testing.T) {
		text := ""
		lineWidth := 10
		expected := ""
		result := wrapText(text, lineWidth)
		assert.Equal(
			t, expected, result, "Empty string output mismatch",
		)
	})

	// Case 6: Whitespace only string, should return the original whitespace.
	t.Run("Whitespace Only", func(t *testing.T) {
		text := "    "
		lineWidth := 10
		expected := "    "
		result := wrapText(text, lineWidth)
		assert.Equal(
			t, expected, result, "Whitespace only output mismatch",
		)
	})
}

// TestInitConfig tests the initConfig function to ensure it initializes
// configuration correctly.
func TestInitConfig(t *testing.T) {
	// Define a temporary directory for test config files.
	tempDir := t.TempDir()

	// Function to reset Viper between test cases.
	resetViper := func() {
		viper.Reset()
	}

	// Case 1: Create new config file when it does not exist.
	t.Run("Create new config file", func(t *testing.T) {
		defer resetViper()

		configFileName := "testconfig.conf"
		configFilePath := filepath.Join(tempDir, configFileName)
		config, err := initConfig(tempDir, configFileName)

		assert.NoError(t, err, "initConfig should not return an error")
		assert.FileExists(
			t, configFilePath, "Config file should be created",
		)

		defaultConfig, err := DefaultConfig()
		assert.NoError(t, err)
		assert.Equal(
			t, defaultConfig, *config,
			"Config should match default values",
		)
	})

	// Case 2: Read existing config file and load its values.
	t.Run("Read existing config file", func(t *testing.T) {
		defer resetViper()

		configFileName := "testconfig.conf"
		configFilePath := filepath.Join(tempDir, configFileName)

		// Create a sample config file.
		err := os.WriteFile(configFilePath, []byte(`
[section]
key = "value"
`), 0644)
		assert.NoError(
			t, err, "Should be able to write sample config file",
		)

		_, err = initConfig(tempDir, configFileName)
		assert.NoError(t, err, "initConfig should not return an error")

		// Check if Viper reads the correct value.
		viper.SetConfigFile(configFilePath)
		viper.SetConfigType("ini")
		err = viper.ReadInConfig()
		assert.NoError(t, err)
		assert.Equal(
			t, "value", viper.GetString("section.key"),
			"Config value should be loaded from file",
		)
	})

	// Case 3: Handle errors during config file creation.
	t.Run("Error creating config file", func(t *testing.T) {
		defer resetViper()

		// Set path to an invalid directory to force an error.
		invalidDir := "/invalid-directory"
		configFileName := "config.ini"
		_, err := initConfig(invalidDir, configFileName)

		assert.Error(
			t, err, "initConfig should return an error when "+
				"failing to create config file",
		)
	})
}
