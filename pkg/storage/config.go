package storage

import "fmt"

const (
	// DefaultMaxRetries is the number of retries for a write operation
	DefaultMaxRetries = 4

	// DefaultRetryDelay is the number of milliseconds to wait before attempting a retry after a
	//   failure.
	DefaultRetryDelay = 5000
)

// Config holds all configuration for the Storage.
//
// Config is geared towards "bucket" style storage, where you have a
// specific root (the Bucket).
type Config struct {
	Bucket     string
	Root       string
	MaxRetries int
	RetryDelay int
}

// NewConfig returns a new Config with AWS style options.
func NewConfig(bucket, root string) Config {
	return Config{
		Bucket:     bucket,
		Root:       root,
		MaxRetries: DefaultMaxRetries,
		RetryDelay: DefaultRetryDelay,
	}
}

func (c Config) String() string {
	root := ""
	if len(c.Root) > 0 {
		root = fmt.Sprintf("Root:%s", c.Root)
	}

	return fmt.Sprintf("{Bucket:%v %s MaxRetries:%v RetryDelay:%v}",
		c.Bucket,
		root,
		c.MaxRetries,
		c.RetryDelay)
}
