package fsutil

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

var _ fs.File = (*errorReader)(nil)
var _ fs.FS = (*errorFS)(nil)

// errorReader always returns an error on Read
type errorReader struct {
	fs.File
}

func (e *errorReader) Read([]byte) (int, error) {
	return 0, errors.New("failed read")
}

// errorFS is a wrapper around that returns an errorReader for a specific file
type errorFS struct {
	fs.FS
	errorFile string
}

func (e *errorFS) Open(name string) (fs.File, error) {
	if name == e.errorFile {
		return &errorReader{}, nil
	}
	return e.FS.Open(name)
}

func TestReadFileAll(t *testing.T) {
	t.Run("successfully reads existing file", func(t *testing.T) {
		fs := fstest.MapFS{
			"testfile.txt": &fstest.MapFile{Data: []byte("Hello, World!")},
		}

		content, err := ReadFileAll(fs, "testfile.txt")

		assert.NoError(t, err)
		assert.Equal(t, []byte("Hello, World!"), content)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		fs := fstest.MapFS{}
		_, err := ReadFileAll(fs, "nonexistent.txt")

		assert.Error(t, err)
	})

	t.Run("returns error for directory", func(t *testing.T) {
		fs := fstest.MapFS{
			"testdir": &fstest.MapFile{Mode: fs.ModeDir},
		}
		_, err := ReadFileAll(fs, "testdir")

		assert.Error(t, err)
	})

	t.Run("handles empty file", func(t *testing.T) {
		fs := fstest.MapFS{
			"empty.txt": &fstest.MapFile{Data: []byte{}},
		}

		content, err := ReadFileAll(fs, "empty.txt")

		assert.NoError(t, err)
		assert.Empty(t, content)
	})

	t.Run("handles large file", func(t *testing.T) {
		largeContent := make([]byte, 1024*1024) // 1MB
		for i := range largeContent {
			largeContent[i] = byte(i % 256)
		}

		fs := fstest.MapFS{
			"large.bin": &fstest.MapFile{Data: largeContent},
		}

		content, err := ReadFileAll(fs, "large.bin")

		assert.NoError(t, err)
		assert.Equal(t, largeContent, content)
	})

	t.Run("handles read errors", func(t *testing.T) {
		errorFS := &errorFS{
			FS: fstest.MapFS{
				"error.txt": &fstest.MapFile{Data: []byte("Some data")},
			},
			errorFile: "error.txt",
		}

		_, err := ReadFileAll(errorFS, "error.txt")

		assert.Error(t, err)
	})
}
