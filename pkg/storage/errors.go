package storage

import "errors"

var (
	// ErrNotFound should be returned if the file was not found.
	ErrNotFound = errors.New("Not found")
)
