package storage

import "fmt"

const (
	// DefaultMaxRetries is the number of retries for a write operation
	DefaultMaxRetries = 4
)

// Config holds all configuration for the Storage.
//
// Config is geared towards "bucket" style storage, where you have a
// specific root (the Bucket).
type Config struct {
	Bucket     string
	Root       string
	MaxRetries int
}

// NewConfig returns a new Config with AWS style options.
func NewConfig(bucket, root string) Config {
	return Config{
		Bucket:     bucket,
		Root:       root,
		MaxRetries: DefaultMaxRetries,
	}
}

func (c Config) String() string {
	root := ""
	if len(c.Root) > 0 {
		root = fmt.Sprintf("Root:%s", c.Root)
	}

	return fmt.Sprintf("{Bucket:%v %s MaxRetries:%v}",
		c.Bucket,
		root,
		c.MaxRetries)
}
