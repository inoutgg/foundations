// Package fsutil provides utilities to work with filesystem.
package fsutil

import (
	"fmt"
	"io"
	"io/fs"
)

// ReadFileAll reads a file by given [path] from [fs]. Read file is buffered.
func ReadFileAll(fs fs.FS, path string) ([]byte, error) {
	file, err := fs.Open(path)
	if err != nil {
		return []byte{}, fmt.Errorf("foundations/fsutil: failed opening file %q: %w", path, err)
	}

	content, err := io.ReadAll(file)
	if err != nil {
		return []byte{}, fmt.Errorf("foundations/fsutil: failed to read file %q: %w", path, err)
	}

	return content, nil
}
