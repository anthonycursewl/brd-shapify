package storage

import (
	"os"
	"path/filepath"
)

type LocalStorageAdapter struct {
	BaseDir string
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
	path := filepath.Join(a.BaseDir, id)
	return os.WriteFile(path, data, 0644)
}

func (a *LocalStorageAdapter) Load(id string) ([]byte, error) {
	path := filepath.Join(a.BaseDir, id)
	return os.ReadFile(path)
}

func (a *LocalStorageAdapter) Delete(id string) error {
	path := filepath.Join(a.BaseDir, id)
	return os.Remove(path)
}
