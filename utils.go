package main

import "os"

// checkFilesExist checks if all the specified files exist.
//
// Parameters:
// - files: A variadic list of file paths to check.
//
// Returns:
//   - An error if any of the specified files do not exist, or nil if all files
//     exist.
func checkFilesExist(files ...string) error {
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
