// Package storage abstracts where persisted blobs (the data snapshot,
// auth accounts) live: a local file by default, or Firestore when the server
// runs on a host with no persistent disk (e.g. Render's free tier).
// The snapshot content and schema are identical in both backends — only the
// storage medium changes.
package storage

import (
	"errors"
	"os"
	"path/filepath"
)

// Blob is one persisted document. Load returns (nil, nil) when the blob does
// not exist yet; Save must be atomic so a crash never leaves partial data.
type Blob interface {
	Load() ([]byte, error)
	Save([]byte) error
}

// ---- file backend (default; unchanged local behavior) ----

type fileBlob struct{ path string }

// NewFileBlob persists a blob at the given file path (temp file + rename).
func NewFileBlob(path string) Blob { return fileBlob{path: path} }

func (f fileBlob) Load() ([]byte, error) {
	b, err := os.ReadFile(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return b, err
}

func (f fileBlob) Save(b []byte) error {
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return err
	}
	tmp := f.path + ".tmp"
	// 0600: blobs may contain tokens (auth.json).
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, f.path)
}
