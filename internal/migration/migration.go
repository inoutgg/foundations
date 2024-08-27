package migration

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
)

type Direction int

const (
	Up Direction = iota
	Down
)

type Config struct {
	Module string
}

type Migrator struct {
	FS     fs.FS
	Config *Config
}

func (m *Migrator) Up(ctx context.Context, tx *sql.Tx) error {
	return nil
}

func (m *Migrator) Down(ctx context.Context, tx *sql.Tx) error {
	return nil
}

type Migration struct {
	Name    string
	Version int
	Module  string
}

