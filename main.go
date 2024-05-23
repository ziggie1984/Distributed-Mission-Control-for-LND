package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	logrus "github.com/sirupsen/logrus"
)

// AppPath returns the application path based on the OS.
func AppPath(goos, homeDir string) string {
	// Determine the application path based on the operating system.
	switch goos {
	case "windows":
		// On Windows, use the LOCALAPPDATA environment variable.
		return filepath.Join(os.Getenv("LOCALAPPDATA"), DefaultAppName)

	case "darwin":
		// On macOS, place the application data in the
		// "Application Support" directory.
		return filepath.Join(
			homeDir, "Library", "Application Support", DefaultAppName,
		)

	case "plan9":
		// On Plan 9, use the home directory with the application name
		// in lowercase.
		return filepath.Join(homeDir, strings.ToLower(DefaultAppName))

	// POSIX (Linux, etc.)
	default:
		// On other POSIX systems (e.g., Linux), use a hidden directory
		// in the home directory.
		appName := fmt.Sprintf(".%s", strings.ToLower(DefaultAppName))
		return filepath.Join(homeDir, appName)
	}
}

// EnsureAppPathExists ensures the application directory exists.
func EnsureAppPathExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, AppDirPermissions)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	// Get the user home directory depending on the OS.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logrus.Fatalf("Failed to get user home directory: %v", err)
	}
	// Get the app path directory.
	appPath := AppPath(runtime.GOOS, homeDir)

	// Create App Path if it doesn't exist.
	err = EnsureAppPathExists(appPath)
	if err != nil {
		logrus.Fatalf("Failed to create app directory: %v",
			err)
	}

	// Initiate Config.
	config, err := initConfig(appPath, DefaultConfigFilename)
	if err != nil {
		logrus.Fatalf("Failed to initialize configuration: %v", err)
	}

	// Setup logging.
	err = setupLogging(config)
	if err != nil {
		logrus.Fatalf("Failed to set up logging : %v", err)
	}
	logrus.Info("Logging setup complete")

	// Setup the database.
	db, err := setupDatabase(config)
	if err != nil {
		logrus.Fatalf("Failed to set up database: %v", err)
	}
	defer cleanupDB(db)
	logrus.Info("Database setup complete")

	// Create Third Party TLS Path if it doesn't exit.
	if err := CreateThirdPartyTLSDirIfNotExist(config); err != nil {
		logrus.Fatalf("Failed to create third party TLS dir: %v ", err)
	}

	// Load TLS Configurations.
	tlsCreds, err := loadTLSCredentials(config)
	if err != nil {
		logrus.Fatalf("Failed to load TLS credentials: %v", err)
	}
	logrus.Info("TLS configurations loaded")

	// Initialize and start the pprof server.
	pprofServer := initializePProfServer(config, tlsCreds)
	go func() {
		if err := startPProfServer(config, pprofServer); err != nil {
			logrus.Fatalf("Failed to start pprof server: %v", err)
		}
	}()

	// Initialize and start the gRPC server.
	server, lis, err := initializeGRPCServer(config, tlsCreds, db)
	if err != nil {
		logrus.Fatalf("Failed to initialize gRPC server: %v", err)
	}
	go func() {
		if err := startGRPCServer(config, server, lis); err != nil {
			logrus.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	// Persistent context for the gRPC REST gateway.
	restCtx, restCancel := context.WithCancel(context.Background())
	defer restCancel()

	// Initialize and start the HTTP server for the gRPC REST gateway.
	httpServer, err := initializeHTTPServer(restCtx, tlsCreds, config)
	if err != nil {
		logrus.Fatalf("Failed to initialize HTTP server: %v", err)
	}
	go func() {
		if err := startHTTPServer(config, httpServer); err != nil {
			logrus.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Create a channel to listen for interrupt or termination signals from
	// the OS.
	sigChan := make(chan os.Signal, 1)

	// Notify sigChan on os.Interrupt or syscall.SIGTERM.
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Handle graceful shutdown for the gRPC, HTTP, and pprof servers.
	gracefulShutdown(sigChan, server, httpServer, pprofServer)
}
