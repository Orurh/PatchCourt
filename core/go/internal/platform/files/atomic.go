package files

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// EnsureDir creates a directory tree if path is not empty.
func EnsureDir(path string) error {
	if path == "" {
		return nil
	}

	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", path, err)
	}

	return nil
}

// WriteFileAtomic writes data to path through a temp file in the same directory,
// then renames it over the target.
//
// This prevents readers from observing partially written artifacts such as
// project-model.json, metadata.json, cache files, or generated reports.
func WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}

	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", path, err)
	}

	tmpPath := tmp.Name()
	renamed := false
	defer func() {
		if !renamed {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file %s: %w", tmpPath, err)
	}

	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp file %s: %w", tmpPath, err)
	}

	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp file %s: %w", tmpPath, err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file %s: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp file %s to %s: %w", tmpPath, path, err)
	}

	renamed = true
	syncDirBestEffort(dir)

	return nil
}

// WriteJSONAtomic writes indented JSON atomically.
func WriteJSONAtomic(path string, value any) error {
	var buf bytes.Buffer

	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("encode json for %s: %w", path, err)
	}

	return WriteFileAtomic(path, buf.Bytes(), 0o644)
}

func syncDirBestEffort(dir string) {
	handle, err := os.Open(dir)
	if err != nil {
		return
	}
	defer handle.Close()

	_ = handle.Sync()
}
