package sqlcomment

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"go.inout.gg/foundations/dbsql/dbtx"
)

var (
	_ dbtx.DB = (*DB)(nil)
	_ pgx.Tx  = (*Tx)(nil)
)

type commenter struct {
	taggers []Tagger
}

// DB is a DB wrapper that adds an SQL comment.
type DB struct {
	dbtx.DB
	commenter
}

// NewWithDefaults decorates the given DB and adds an SQL comment to every query.
func NewWithDefaults(db dbtx.DB, options ...Option) *DB {
	var c commenter
	for _, opt := range options {
		opt(&c)
	}
	WithTagger(NewOTELTagger(), contextTagger{})(&c)

	return &DB{DB: db, commenter: c}
}

func (c commenter) withComment(ctx context.Context, query string) string {
	if skipFromContext(ctx) {
		return query
	}

	tags := make(Tags)
	for _, h := range c.taggers {
		tags.Merge(h.Tag(ctx))
	}

	return fmt.Sprintf("%s /*%s*/", query, tags.Marshal())
}

func (c commenter) commentBatch(ctx context.Context, b *pgx.Batch) *pgx.Batch {
	if skipFromContext(ctx) {
		return b
	}

	commented := &pgx.Batch{QueuedQueries: make([]*pgx.QueuedQuery, 0, len(b.QueuedQueries))}
	for _, qq := range b.QueuedQueries {
		copy := *qq
		copy.SQL = c.withComment(ctx, copy.SQL)
		commented.QueuedQueries = append(commented.QueuedQueries, &copy)
	}

	return commented
}

func (d *DB) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return d.DB.Exec(ctx, d.withComment(ctx, query), args...)
}

func (d *DB) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return d.DB.Query(ctx, d.withComment(ctx, query), args...)
}

func (d *DB) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return d.DB.QueryRow(ctx, d.withComment(ctx, query), args...)
}

func (d *DB) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return d.DB.SendBatch(ctx, d.commentBatch(ctx, b))
}

func (d *DB) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := d.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return &Tx{Tx: tx, commenter: d.commenter}, nil
}

func (d *DB) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	tx, err := d.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &Tx{Tx: tx, commenter: d.commenter}, nil
}

// Tx is a pgx transaction wrapper that adds an SQL comment.
type Tx struct {
	pgx.Tx
	commenter
}

func (d *Tx) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := d.Tx.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return &Tx{Tx: tx, commenter: d.commenter}, nil
}

func (d *Tx) Prepare(
	ctx context.Context,
	name string,
	query string,
) (*pgconn.StatementDescription, error) {
	return d.Tx.Prepare(ctx, name, d.withComment(ctx, query))
}

func (d *Tx) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return d.Tx.Exec(ctx, d.withComment(ctx, query), args...)
}

func (d *Tx) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return d.Tx.Query(ctx, d.withComment(ctx, query), args...)
}

func (d *Tx) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return d.Tx.QueryRow(ctx, d.withComment(ctx, query), args...)
}

func (d *Tx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return d.Tx.SendBatch(ctx, d.commentBatch(ctx, b))
}
