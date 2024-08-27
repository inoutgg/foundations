package migration

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"runtime"
	"strconv"
	"strings"

	"go.inout.gg/foundations/must"
)

var registry = make(map[int]migration)

const Sep = "---- create above / drop below ----"

func Put(up, down MigrateFunc) {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic(errors.New("migrations: failed to retrieve caller information"))
	}

	version, _ := must.Must3(extractVersionFromFilename(filename))
	registry[version] = &goMigration{version, up, down}
}

// example: 0001_create_user.sql, 0002_create_river_queue.go
func extractVersionFromFilename(filename string) (int, string, error) {
	basename := path.Base(filename)
	extension := path.Ext(basename)
	if !(extension == "go" || extension == "sql") {
		return 0, "", errors.New("migrations: unknown migration file extension")
	}

	verStr, name, ok := strings.Cut(basename, "_")
	if !ok {
		return 0, "", fmt.Errorf("migrations: malformed migration filename, expected format: <numeric-version-part>_<string-part>.[go|sql], was: %s", basename)
	}

	ver, err := strconv.Atoi(verStr)
	if err != nil {
		return 0, "", fmt.Errorf("migrations: unable parse version number from the migration %s filename: %w", basename, err)
	}

	return ver, name, nil
}

func read(fs fs.FS, path string) (string, error) {
	file, err := fs.Open(path)
	if err != nil {
		return "", fmt.Errorf("migrations: failed opening file %q: %w", path, err)
	}

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("migrations: failed to read file %q: %w", path, err)
	}

	return string(content), nil
}

func parse(fs fs.FS, root, path string) (migration, error) {
	filename := path[len(root)+1:]
	version, _, err := extractVersionFromFilename(filename)
	if err != nil {
		return nil, err
	}

	content, err := read(fs, path)
	if err != nil {
		return nil, err
	}

	migration := sqlMigration{version: version}
	up, down, ok := strings.Cut(content, Sep)

	up = strings.TrimSpace(up)
	if up == "" {
		return nil, errors.New("migrations: empty migration script")
	}
	migration.up = up

	if ok {
		migration.down = strings.TrimSpace(down)
	}

	return &migration, nil
}

func SQLMigrationsFromFS(fsys fs.FS, root string) (migrations []migration, err error) {
	err = fs.WalkDir(fsys, root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("migrations: error occurred while parsing migrations directory: %w", err)
		}

		// Matching the root directory, skip it.
		if path == root {
			return nil
		}

		migration, err := parse(fsys, root, path)
		if err != nil {
			return err
		}

		migrations = append(migrations, migration)

		return nil
	})
	if err != nil {
		return []migration{}, nil
	}

	return migrations, nil
}
