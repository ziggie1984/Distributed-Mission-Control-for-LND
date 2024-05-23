package main

import (
	"os"
	"path/filepath"

	logrus "github.com/sirupsen/logrus"
	bbolt "go.etcd.io/bbolt"
)

// setupDatabase initializes and returns a bbolt DB instance based on the
// provided configuration. It ensures the database directory exists,
// creates the database file if necessary, and configures the database with the
// specified settings.
func setupDatabase(config *Config) (*bbolt.DB, error) {
	// Ensure the database directory exists.
	_, err := os.Stat(config.Database.DatabaseDirPath)
	if os.IsNotExist(err) {
		err := os.Mkdir(
			config.Database.DatabaseDirPath,
			DatabaseDirPermissions,
		)
		if err != nil {
			return nil, err
		}
	}

	// Construct the full path to the database file.
	dbFilePath := filepath.Join(
		config.Database.DatabaseDirPath, config.Database.DatabaseFile,
	)

	// Open the database with a timeout.
	options := &bbolt.Options{Timeout: config.Database.FileLockTimeout}
	db, err := bbolt.Open(
		dbFilePath, DatabaseFilePermissions, options,
	)
	if err != nil {
		return nil, err
	}

	// Create the main bucket for mission control data if it doesn't exist.
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(
			[]byte(DatabaseBucketName),
		)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		db.Close()
		return nil, err
	}

	// Configure MaxBatchDelay and MaxBatchSize.
	db.MaxBatchDelay = config.Database.MaxBatchDelay
	db.MaxBatchSize = config.Database.MaxBatchSize

	return db, nil
}

// cleanupDB closes the database connection and logs any errors encountered
// during the process. It exits the program with a status code of 1 if the
// database fails to close.
func cleanupDB(db *bbolt.DB) {
	if err := db.Close(); err != nil {
		logrus.Errorf("Failed to close database: %v"+"\n", err)
		os.Exit(1)
	}
	logrus.Info("Database connection closed")
}
