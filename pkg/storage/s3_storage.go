package storage

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/tokenized/smart-contract/pkg/logger"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
)

// S3Storage implements the Storage interface for interacting with AWS S3.
type S3Storage struct {
	Config  Config
	Session *session.Session
}

// NewS3Storage creates a new S3Storage with a new aws.Session.
func NewS3Storage(config Config) S3Storage {
	return S3Storage{
		Config:  config,
		Session: newAWSSession(config),
	}
}

// NewS3StorageWithSession returns a new S3Storage with a given AWS Session.
func NewS3StorageWithSession(config Config,
	session *session.Session) S3Storage {

	return S3Storage{
		Config:  config,
		Session: session,
	}
}

// Write writes the data to the key in the S3 Bucket, with Options applied.
func (s S3Storage) Write(ctx context.Context,
	key string,
	body []byte,
	options *Options) error {

	svc := s3.New(s.Session)

	poi := s3.PutObjectInput{
		Bucket: aws.String(s.Config.Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	}

	var err error
	for i := 0; i <= s.Config.MaxRetries; i++ {
		if i != 0 {
			time.Sleep(time.Duration(s.Config.RetryDelay) * time.Millisecond)
		}

		if options != nil {
			if options.TTL > 0 {
				expiry := time.Now().Add(time.Duration(options.TTL) * time.Second)
				poi.Expires = &expiry
			}
		}

		_, err = svc.PutObject(&poi)
		if err == nil {
			break
		}

		logger.Error(ctx, "S3CallFailed to write to %v : %v", key, err)
	}

	if err != nil {
		logger.Error(ctx, "S3CallAborted write to %v : %v", key, err)
		return errors.Wrap(err, fmt.Sprintf("Failed to write to %v", key))
	}
	return nil
}

// Read will read the data from the S3 Bucket.
func (s S3Storage) Read(ctx context.Context, key string) ([]byte, error) {
	svc := s3.New(s.Session)

	var err error
	var document *s3.GetObjectOutput
	var b []byte
	for i := 0; i <= s.Config.MaxRetries; i++ {
		if i != 0 {
			time.Sleep(time.Duration(s.Config.RetryDelay) * time.Millisecond)
		}

		document, err = svc.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(s.Config.Bucket),
			Key:    aws.String(key),
		})

		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() == s3.ErrCodeNoSuchKey {
					// specifically handle the "not found" case
					return nil, ErrNotFound
				}
			}

			logger.Error(ctx, "S3CallFailed to read from %v : %v", key, err)
			continue
		}

		b, err = ioutil.ReadAll(document.Body)
		if err != nil {
			logger.Error(ctx, "S3CallFailed to read from %v : %v", key, err)
			continue
		}

		break
	}

	if err != nil {
		logger.Error(ctx, "S3CallAborted read from %v : %v", key, err)
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to read from %v", key))
	}
	return b, nil
}

// Remove removes the object stored at key, in the S3 Bucket.
func (s S3Storage) Remove(ctx context.Context, key string) error {
	svc := s3.New(s.Session)

	do := &s3.DeleteObjectInput{
		Bucket: aws.String(s.Config.Bucket),
		Key:    aws.String(key),
	}

	var err error
	for i := 0; i <= s.Config.MaxRetries; i++ {
		if i != 0 {
			time.Sleep(time.Duration(s.Config.RetryDelay) * time.Millisecond)
		}

		_, err = svc.DeleteObject(do)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() == s3.ErrCodeNoSuchKey {
					// specifically handle the "not found" case
					return ErrNotFound
				}
			}

			logger.Error(ctx, "S3CallFailed to delete object at %v : %v", key, err)
			continue
		}
	}

	if err != nil {
		logger.Error(ctx, "S3CallAborted delete object at %v : %v", key, err)
		return errors.Wrap(err, fmt.Sprintf("Failed to delete object at %v", key))
	}
	return nil
}

func (s S3Storage) Search(ctx context.Context,
	query map[string]string) ([][]byte, error) {

	path := query["path"]

	var err error
	var keys []string
	for i := 0; i <= s.Config.MaxRetries; i++ {
		keys, err = s.findKeys(ctx, path)
		if err == nil {
			break
		}

		logger.Error(ctx, "S3CallFailed to search %v : %v", path, err)
		time.Sleep(time.Duration(s.Config.RetryDelay) * time.Millisecond)
	}

	if err != nil {
		logger.Error(ctx, "S3CallAborted search %v : %v", path, err)
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to search %v", path))
	}

	svc := s3manager.NewDownloader(s.Session)

	buf := newObjectStore()

	objects := make([]s3manager.BatchDownloadObject, len(keys), len(keys))

	bucket := &s.Config.Bucket

	for i, k := range keys {
		o := s3manager.BatchDownloadObject{
			Object: &s3.GetObjectInput{
				Bucket: bucket,
				Key:    aws.String(k),
			},
			Writer: buf,
		}

		objects[i] = o
	}

	iter := &s3manager.DownloadObjectsIterator{Objects: objects}

	for i := 0; i <= s.Config.MaxRetries; i++ {
		if i != 0 {
			time.Sleep(time.Duration(s.Config.RetryDelay) * time.Millisecond)
		}

		err = svc.DownloadWithIterator(ctx, iter)
		if err == nil {
			break
		}

		logger.Error(ctx, "S3CallFailed to download with iterator %v : %v", path, err)
	}

	if err != nil {
		logger.Error(ctx, "S3CallAborted download with iterator %v : %v", path, err)
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to download with iterator %v", path))
	}

	return buf.objects(), nil
}

func (s S3Storage) Clear(ctx context.Context, query map[string]string) error {
	path := query["path"]

	var err error
	var keys []string
	for i := 0; i <= s.Config.MaxRetries; i++ {
		if i != 0 {
			time.Sleep(time.Duration(s.Config.RetryDelay) * time.Millisecond)
		}

		keys, err = s.findKeys(ctx, path)
		if err == nil {
			break
		}

		logger.Error(ctx, "S3CallFailed to search %v : %v", path, err)
	}

	if err != nil {
		logger.Error(ctx, "S3CallAborted search %v : %v", path, err)
		return errors.Wrap(err, fmt.Sprintf("Failed to search %v", path))
	}

	svc := s3manager.NewBatchDelete(s.Session)

	objects := make([]s3manager.BatchDeleteObject, len(keys), len(keys))

	bucket := &s.Config.Bucket

	for i, k := range keys {
		o := s3manager.BatchDeleteObject{
			Object: &s3.DeleteObjectInput{
				Bucket: bucket,
				Key:    aws.String(k),
			},
		}

		objects[i] = o
	}

	iter := &s3manager.DeleteObjectsIterator{Objects: objects}

	for i := 0; i <= s.Config.MaxRetries; i++ {
		if i != 0 {
			time.Sleep(time.Duration(s.Config.RetryDelay) * time.Millisecond)
		}

		err = svc.Delete(ctx, iter)
		if err == nil {
			break
		}

		logger.Error(ctx, "S3CallFailed to delete %v : %v", path, err)
	}

	if err != nil {
		logger.Error(ctx, "S3CallAborted delete %v : %v", path, err)
		return errors.Wrap(err, fmt.Sprintf("Failed to delete %v", path))
	}

	return nil
}

func (s S3Storage) List(ctx context.Context, path string) ([]string, error) {
	var err error
	var keys []string
	for i := 0; i <= s.Config.MaxRetries; i++ {
		if i != 0 {
			time.Sleep(time.Duration(s.Config.RetryDelay) * time.Millisecond)
		}

		keys, err = s.findKeys(ctx, path)
		if err == nil {
			return keys, nil
		}

		logger.Error(ctx, "S3CallFailed to search %v : %v", path, err)
	}

	logger.Error(ctx, "S3CallAborted search %v : %v", path, err)
	return nil, errors.Wrap(err, fmt.Sprintf("Failed to search %v", path))
}

func (s S3Storage) findKeys(ctx context.Context, path string) ([]string, error) {

	svc := s3.New(s.Session)

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.Config.Bucket),
		Prefix: &path,
	}

	out, err := svc.ListObjectsV2(input)
	if err != nil {
		return nil, err
	}

	keys := make([]string, len(out.Contents), len(out.Contents))
	for i, o := range out.Contents {
		keys[i] = *o.Key
	}

	return keys, nil
}

// newAwsSession creates a new AWS Session from the credentials in the Config.
func newAWSSession(config Config) *session.Session {
	awsConfig := aws.NewConfig()
	return session.New(awsConfig)
}

// objectStore implements the WriteAt interface.
type objectStore struct {
	data map[int][]byte
}

func newObjectStore() *objectStore {
	return &objectStore{
		data: map[int][]byte{},
	}
}

func (o objectStore) WriteAt(p []byte, pos int64) (int, error) {
	idx := len(o.data)
	o.data[idx] = p

	return len(p), nil
}

func (o objectStore) objects() [][]byte {
	var objs = make([][]byte, len(o.data), len(o.data))

	for i, o := range o.data {
		objs[i] = o
	}

	return objs
}
