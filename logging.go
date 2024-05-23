package main

import (
	"io"
	"os"
	"path/filepath"

	logrus "github.com/sirupsen/logrus"
)

// setupLogging configures logging settings.
func setupLogging(config *Config) error {
	// Ensure the database directory exists.
	if _, err := os.Stat(config.Log.LogDirPath); os.IsNotExist(err) {
		err := os.Mkdir(config.Log.LogDirPath, LogDirPermissions)
		if err != nil {
			return err
		}
	}

	// Construct the full path to the database file.
	logFilePath := filepath.Join(config.Log.LogDirPath, config.Log.LogFile)

	// Create and open the log file with strict permissions.
	logFile, err := os.OpenFile(
		logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		LogFilePermissions,
	)
	if err != nil {
		return err
	}

	// Create a multi-writer to write to both standard output and the log
	// file.
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logrus.SetOutput(multiWriter)

	// Set log format to include date and time.
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Set the log level based on the config.
	logLevel := convertLogLevel(config.Log.LogLevel)
	logrus.SetLevel(logLevel)

	return nil
}

// convertLogLevel converts a log level string from the config to a logrus log
// level.
func convertLogLevel(level string) logrus.Level {
	switch level {
	case "fatal":
		return logrus.FatalLevel
	case "error":
		return logrus.ErrorLevel
	case "warn", "warning":
		return logrus.WarnLevel
	case "info":
		return logrus.InfoLevel
	case "debug":
		return logrus.DebugLevel
	default:
		// Default to info level if the provided level is unrecognized.
		return logrus.InfoLevel
	}
}
