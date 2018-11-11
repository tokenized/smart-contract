package storage

import "os"

// Options for writing data. Not all Storage implementations will support
// all options.
//
// For example, writing a file wouldn't support TTL.
type Options struct {
	TTL     int64
	Mode    os.FileMode
	DirMode os.FileMode
}

// NewOptions returns an Options struct with sane defaults set.
//
// TTL with zero value means never expire.
func NewOptions() Options {
	return Options{
		TTL:     0,
		Mode:    0644,
		DirMode: 0755,
	}
}
