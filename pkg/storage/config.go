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
	Region     string
	AccessKey  string
	Secret     string
	Bucket     string
	Root       string
	MaxRetries int
}

// NewConfig returns a new Config with AWS style options.
func NewConfig(region, accessKey, secret, bucket, root string) Config {
	return Config{
		Region:     region,
		AccessKey:  accessKey,
		Secret:     secret,
		Bucket:     bucket,
		Root:       root,
		MaxRetries: DefaultMaxRetries,
	}
}

func (c Config) String() string {
	secret := ""
	if len(c.Secret) > 0 {
		secret = "****"
	}

	root := ""
	if len(c.Root) > 0 {
		root = fmt.Sprintf("Root:%s", c.Root)
	}

	return fmt.Sprintf("{Region:%v AccessKey:%v Secret:%v Bucket:%v %sMaxRetries:%v}",
		c.Region,
		c.AccessKey,
		secret,
		c.Bucket,
		root,
		c.MaxRetries)
}
