// Package storage provides implementations for file storage adapters.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

type MultiStorageAdapter struct {
	backend  string
	localDir string
	s3Bucket string
	s3Prefix string
}

func NewMultiStorageAdapter(backend, localDir, s3Bucket, s3Prefix string) (*MultiStorageAdapter, error) {
	a := &MultiStorageAdapter{
		backend:  backend,
		localDir: localDir,
		s3Bucket: s3Bucket,
		s3Prefix: s3Prefix,
	}

	if backend == "s3" && s3Bucket != "" {
		if err := a.testS3(); err != nil {
			return nil, fmt.Errorf("S3 connection failed: %w", err)
		}
	}

	return a, nil
}

func (a *MultiStorageAdapter) Save(id string, data []byte) error {
	if a.backend == "s3" {
		return a.saveToS3(id, data)
	}
	path := filepath.Join(a.localDir, id)
	return os.WriteFile(path, data, 0644)
}

func (a *MultiStorageAdapter) Load(id string) ([]byte, error) {
	if a.backend == "s3" {
		return a.loadFromS3(id)
	}
	path := filepath.Join(a.localDir, id)
	return os.ReadFile(path)
}

func (a *MultiStorageAdapter) Delete(id string) error {
	if a.backend == "s3" {
		return a.deleteFromS3(id)
	}
	path := filepath.Join(a.localDir, id)
	return os.Remove(path)
}

func (a *MultiStorageAdapter) testS3() error {
	return nil
}

func (a *MultiStorageAdapter) saveToS3(id string, data []byte) error {
	return fmt.Errorf("S3 not configured - set S3_BUCKET environment variable")
}

func (a *MultiStorageAdapter) loadFromS3(id string) ([]byte, error) {
	return nil, fmt.Errorf("S3 not configured")
}

func (a *MultiStorageAdapter) deleteFromS3(id string) error {
	return fmt.Errorf("S3 not configured")
}
