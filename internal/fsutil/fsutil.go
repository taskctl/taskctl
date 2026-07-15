// Package fsutil provides small filesystem and working-directory helpers.
package fsutil

import (
	"errors"
	"os"
)

// FileExists checks if the file exists
func FileExists(file string) bool {
	_, err := os.Stat(file)
	if err == nil {
		return true
	}

	return !errors.Is(err, os.ErrNotExist)
}

// MustGetwd returns the current working directory.
// Panics if os.Getwd() returns error
func MustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return wd
}

// MustGetUserHomeDir returns the current user's home directory.
// Panics if os.UserHomeDir() returns error
func MustGetUserHomeDir() string {
	hd, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	return hd
}
