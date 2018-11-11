package storage

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// FilesystemStorage implements the Storage interface for interacting with
// the local filesystem.
type FilesystemStorage struct {
	Config Config
}

// NewFilesystemStorage implements the Storage interface for simple S3 like
// file system interactions.
func NewFilesystemStorage(config Config) FilesystemStorage {
	return FilesystemStorage{
		Config: config,
	}
}

// Write will write the data to the key in the S3 Bucket.
func (f FilesystemStorage) Write(ctx context.Context,
	key string,
	body []byte,
	options *Options) error {

	// make sure that the Options argument is valid
	if options == nil {
		opts := NewOptions()
		options = &opts
	}

	filename := f.buildPath(key)

	// make sure directory exists.
	dir := path.Dir(filename)

	if err := f.ensureExists(dir, nil); err != nil {
		return err
	}

	var mode os.FileMode = 0644

	if options != nil && options.Mode != 0 {
		mode = options.Mode
	}

	return ioutil.WriteFile(filename, body, mode)
}

// Read reads the data from a file on the local filesystem.
func (f FilesystemStorage) Read(ctx context.Context,
	key string) ([]byte, error) {

	filename := f.buildPath(key)

	// check for existence of file
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, ErrNotFound
	}

	return ioutil.ReadFile(filename)
}

// Remove removes the object stored at key, in the S3 Bucket.
func (f FilesystemStorage) Remove(ctx context.Context, key string) error {
	filename := f.buildPath(key)

	return os.Remove(filename)
}

// All returns all objects in the store, from a given path.
//
// The path can be empty.
func (f FilesystemStorage) Search(ctx context.Context,
	query map[string]string) ([][]byte, error) {

	path := query["path"]

	dir := f.buildPath(path)

	if err := f.ensureExists(dir, nil); err != nil {
		return nil, err
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	objects := [][]byte{}

	for _, info := range files {
		path := strings.Join([]string{path, info.Name()}, "/")
		b, err := f.Read(ctx, path)
		if err != nil {
			return nil, err
		}

		objects = append(objects, b)

	}

	return objects, nil
}

func (f FilesystemStorage) buildPath(key string) string {
	parts := []string{
		f.Config.Root,
		f.Config.Bucket,
	}

	if len(key) > 0 {
		parts = append(parts, key)
	}

	s := strings.Join(parts, "/")

	return filepath.FromSlash(s)
}

func (f FilesystemStorage) ensureExists(dir string, options *Options) error {
	if options == nil {
		opts := NewOptions()
		options = &opts
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, options.DirMode); err != nil {
			return err
		}
	}

	return nil
}
