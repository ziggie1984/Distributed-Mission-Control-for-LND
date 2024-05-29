package main

import (
	"fmt"
	"os"
	"time"
)

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

// mostRecentUnixTimestamp returns the most recent UNIX timestamp from the
// provided list.
//
// Parameters:
//   - timestamps: A variadic list of UNIX timestamps to compare.
//
// Returns:
//   - The most recent UNIX timestamp from the provided list.
func mostRecentUnixTimestamp(timestamps ...int64) int64 {
	var mostRecent int64
	for _, ts := range timestamps {
		if ts > mostRecent {
			mostRecent = ts
		}
	}
	return mostRecent
}

// formatDuration formats a time.Duration into a string representing
// hours, minutes, and seconds.
//
// Parameters:
//   - duration: The time.Duration to be formatted.
//
// Returns:
//   - A string representing the duration in the format "hours, minutes, seconds".
func formatDuration(duration time.Duration) string {
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	return fmt.Sprintf("%d hours, %d minutes, %d seconds", hours, minutes,
		seconds)
}
