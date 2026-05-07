// Package storage provides implementations for file storage adapters.
package storage

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type LocalStorageAdapter struct {
	BaseDir string
}

var safeIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_\.-]+$`)

func isValidID(id string) bool {
	if id == "" || len(id) > 255 {
		return false
	}
	return safeIDRegex.MatchString(id)
}

func NewLocalStorageAdapter(baseDir string) (*LocalStorageAdapter, error) {
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		err := os.MkdirAll(baseDir, 0755)
		if err != nil {
			return nil, err
		}
	}
	return &LocalStorageAdapter{BaseDir: baseDir}, nil
}

func (a *LocalStorageAdapter) Save(id string, data []byte) error {
	if !isValidID(id) {
		return &PathTraversalError{ID: id}
	}
	path := filepath.Join(a.BaseDir, id)
	if !strings.HasPrefix(filepath.Dir(path), a.BaseDir) {
		return &PathTraversalError{ID: id}
	}
	return os.WriteFile(path, data, 0600)
}

func (a *LocalStorageAdapter) Load(id string) ([]byte, error) {
	if !isValidID(id) {
		return nil, &PathTraversalError{ID: id}
	}
	path := filepath.Join(a.BaseDir, id)
	if !strings.HasPrefix(filepath.Dir(path), a.BaseDir) {
		return nil, &PathTraversalError{ID: id}
	}
	return os.ReadFile(path)
}

func (a *LocalStorageAdapter) Delete(id string) error {
	if !isValidID(id) {
		return &PathTraversalError{ID: id}
	}
	path := filepath.Join(a.BaseDir, id)
	if !strings.HasPrefix(filepath.Dir(path), a.BaseDir) {
		return &PathTraversalError{ID: id}
	}
	return os.Remove(path)
}

type PathTraversalError struct {
	ID string
}

func (e *PathTraversalError) Error() string {
	return "invalid or potentially malicious file ID"
}
